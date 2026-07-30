package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"connectrpc.com/connect"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/cli/timeparse"
)

// ── message types ──────────────────────────────────────────────────────────

// planEntriesMsg carries the result of a ListPlanEntries call.
// bg marks a clock-initiated background load; day records which day the
// call was dispatched for.
type planEntriesMsg struct {
	entries     []*planv1.PlanEntry
	highlightID int32 // entry to highlight after loading; 0 = clamp cursor
	err         error
	bg          bool
	day         string
}

// planMutatedMsg is returned after a plan mutation: carries the entry to highlight on reload.
type planMutatedMsg struct {
	highlightID    int32
	err            error
	notice         string // success notice to show in the status bar (Tasks-tab sends)
	tabAgnosticErr bool   // when true, route err to m.err instead of m.plan.err
}

// planTasksMsg carries the task tree for the task picker.
type planTasksMsg struct {
	tree []*cli.TreeNode
	err  error
}

// planTickMsg is fired by the planning-scoped 1-second tick.
type planTickMsg struct{}

// ── command factories ──────────────────────────────────────────────────────

func listPlanCmd(client planv1connect.PlanServiceClient, day string, bg bool) tea.Cmd {
	return listPlanHighlightCmd(client, day, 0, bg)
}

func listPlanHighlightCmd(client planv1connect.PlanServiceClient, day string, highlightID int32, bg bool) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListPlanEntries(context.Background(), connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
		if err != nil {
			return planEntriesMsg{err: err, bg: bg, day: day}
		}
		return planEntriesMsg{entries: resp.Msg.Entries, highlightID: highlightID, bg: bg, day: day}
	}
}

func addPlanTaskCmd(client planv1connect.PlanServiceClient, day string, taskID int64, start, dur int, timed bool, notice ...string) tea.Cmd {
	successNotice := ""
	tabAgnostic := false
	if len(notice) > 0 {
		successNotice = notice[0]
		tabAgnostic = true
	}
	return func() tea.Msg {
		var sm *int32
		if timed {
			v := int32(start)
			sm = &v
		}
		resp, err := client.AddPlanTask(context.Background(), connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day:            day,
			TaskId:         taskID,
			StartMinute:    sm,
			DurationMinute: int32(dur),
		}))
		if err != nil {
			return planMutatedMsg{err: err, tabAgnosticErr: tabAgnostic}
		}
		return planMutatedMsg{highlightID: resp.Msg.Entry.GetId(), notice: successNotice}
	}
}

func addPlanEventCmd(client planv1connect.PlanServiceClient, day, name string, start, dur int) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.AddPlanEvent(context.Background(), connect.NewRequest(&planv1.AddPlanEventRequest{
			Day:            day,
			Name:           name,
			StartMinute:    int32(start),
			DurationMinute: int32(dur),
		}))
		if err != nil {
			return planMutatedMsg{err: err}
		}
		return planMutatedMsg{highlightID: resp.Msg.Entry.GetId()}
	}
}

func renamePlanCmd(client planv1connect.PlanServiceClient, day string, id int32, name string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.RenamePlanEntry(context.Background(), connect.NewRequest(&planv1.RenamePlanEntryRequest{
			Day:  day,
			Id:   id,
			Name: name,
		}))
		if err != nil {
			return planMutatedMsg{err: err}
		}
		return planMutatedMsg{highlightID: id}
	}
}

func movePlanCmd(client planv1connect.PlanServiceClient, day string, id int32, start, dur int, timed bool) tea.Cmd {
	return func() tea.Msg {
		var sm *int32
		if timed {
			v := int32(start)
			sm = &v
		}
		_, err := client.MovePlanEntry(context.Background(), connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day:            day,
			Id:             id,
			StartMinute:    sm,
			DurationMinute: int32(dur),
		}))
		if err != nil {
			return planMutatedMsg{err: err}
		}
		return planMutatedMsg{highlightID: id}
	}
}

func removePlanCmd(client planv1connect.PlanServiceClient, day string, id int32) tea.Cmd {
	return func() tea.Msg {
		_, err := client.RemovePlanEntry(context.Background(), connect.NewRequest(&planv1.RemovePlanEntryRequest{
			Day: day,
			Id:  id,
		}))
		if err != nil {
			return planMutatedMsg{err: err}
		}
		return planMutatedMsg{highlightID: 0}
	}
}

// reorderPlanEntryCmd moves an untimed entry before/after an anchor untimed entry,
// then reloads the day via the planMutatedMsg reload path.
func reorderPlanEntryCmd(client planv1connect.PlanServiceClient, day string, id int32, anchorID int32, insertBefore bool) tea.Cmd {
	return func() tea.Msg {
		req := &planv1.ReorderPlanEntryRequest{Day: day, Id: id}
		if insertBefore {
			req.Anchor = &planv1.ReorderPlanEntryRequest_BeforeId{BeforeId: anchorID}
		} else {
			req.Anchor = &planv1.ReorderPlanEntryRequest_AfterId{AfterId: anchorID}
		}
		_, err := client.ReorderPlanEntry(context.Background(), connect.NewRequest(req))
		if err != nil {
			return planMutatedMsg{err: err}
		}
		return planMutatedMsg{highlightID: id}
	}
}

func completePlanTaskCmd(taskClient taskv1connect.TaskServiceClient, day string, taskID int64, complete bool, entryID int32, notice string) tea.Cmd {
	return func() tea.Msg {
		if complete {
			_, err := taskClient.CompleteTask(context.Background(), connect.NewRequest(&taskv1.CompleteTaskRequest{Id: taskID}))
			if err != nil {
				return planMutatedMsg{err: err}
			}
		} else {
			_, err := taskClient.UncompleteTask(context.Background(), connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: taskID}))
			if err != nil {
				return planMutatedMsg{err: err}
			}
		}
		return planMutatedMsg{notice: notice, highlightID: entryID}
	}
}

// listTasksForPickerCmd fetches incomplete tasks for the task picker.
func listTasksForPickerCmd(client taskv1connect.TaskServiceClient) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return planTasksMsg{err: err}
		}
		tree := cli.BuildTree(resp.Msg.Tasks)
		// Filter to incomplete tasks only.
		tree = filterIncomplete(tree)
		return planTasksMsg{tree: tree}
	}
}

// filterIncomplete removes completed tasks from the tree (keeping subtrees of incomplete roots).
func filterIncomplete(nodes []*cli.TreeNode) []*cli.TreeNode {
	var result []*cli.TreeNode
	for _, n := range nodes {
		if n.Task.GetCompletedAt() != nil {
			continue
		}
		n.Children = filterIncomplete(n.Children)
		result = append(result, n)
	}
	return result
}

// planTickCmd fires a planTickMsg after one second. It should only be re-armed
// while the Planning tab is active on today.
func planTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return planTickMsg{}
	})
}

// ── form helpers ──────────────────────────────────────────────────────────

func newPlanInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	return ti
}

// planFormFieldValues returns the current value of each text field in the form.
func planFormFieldValues(f planFormState) []string {
	vals := make([]string, len(f.fields))
	for i := range f.fields {
		vals[i] = f.fields[i].Value()
	}
	return vals
}

// planFormDirty returns true when any field has been changed from its original
// snapshot (using trimmed comparison).
func planFormDirty(f planFormState) bool {
	if len(f.origFields) != len(f.fields) {
		return false
	}
	for i := range f.fields {
		if strings.TrimSpace(f.fields[i].Value()) != strings.TrimSpace(f.origFields[i]) {
			return true
		}
	}
	return false
}

// initAddTaskForm opens the task picker (step 1 of add-task flow).
// The actual picker state is set via planTasksMsg.
func (m *Model) initAddTaskForm() {
	m.plan.mode = planPickTask
	m.plan.form = planFormState{}
}

// initAddEventForm opens the add-event form (name, start, duration).
func (m *Model) initAddEventForm() {
	name := newPlanInput("Event name")
	name.Focus()
	start := newPlanInput("Start time (e.g. 09:00)")
	dur := newPlanInput("Duration (e.g. 30m, optional)")

	m.plan.form = planFormState{
		fields: []textinput.Model{name, start, dur},
		focus:  0,
	}
	m.plan.form.origFields = planFormFieldValues(m.plan.form)
	m.plan.mode = planEventForm
}

// initTaskTimeForm opens the start/duration form for adding a task entry.
func (m *Model) initTaskTimeForm(taskID int64) {
	start := newPlanInput("Start time (optional, e.g. 09:00)")
	start.Focus()
	dur := newPlanInput("Duration (e.g. 30m, optional)")

	m.plan.form = planFormState{
		fields: []textinput.Model{start, dur},
		focus:  0,
		taskID: taskID,
	}
	m.plan.form.origFields = planFormFieldValues(m.plan.form)
	m.plan.mode = planTaskTime
}

// initEditForm opens the unified edit form for the selected entry.
// Name, Start (HH:MM), and Duration (compact) are pre-filled from the entry.
func (m *Model) initEditForm() {
	if len(m.plan.entries) == 0 {
		return
	}
	entry := m.plan.entries[m.plan.cursor]
	name := newPlanInput("Name")
	name.SetValue(entry.Name)
	name.Focus()

	start := newPlanInput("e.g. 09:00")
	if entry.StartMinute != nil {
		min := int(entry.GetStartMinute())
		start.SetValue(fmt.Sprintf("%02d:%02d", min/60, min%60))
	}

	dur := newPlanInput("e.g. 30m")
	if entry.DurationMinute > 0 {
		dur.SetValue(timeparse.FormatDuration(int(entry.DurationMinute)))
	}

	m.plan.form = planFormState{
		fields:  []textinput.Model{name, start, dur},
		focus:   0,
		entryID: entry.Id,
	}
	m.plan.form.origFields = planFormFieldValues(m.plan.form)
	m.plan.mode = planEdit
}

// cyclePlanFormFocus moves focus forward/back across all fields and the two
// virtual button slots (Save = len(fields), Cancel = len(fields)+1).
func (m *Model) cyclePlanFormFocus(delta int) {
	if len(m.plan.form.fields) == 0 {
		return
	}
	n := len(m.plan.form.fields) + 2 // +2 for Save / Cancel
	m.plan.form.focus = ((m.plan.form.focus+delta)%n + n) % n

	for i := range m.plan.form.fields {
		m.plan.form.fields[i].Blur()
	}
	if m.plan.form.focus < len(m.plan.form.fields) {
		m.plan.form.fields[m.plan.form.focus].Focus()
	}
}

// submitPlanForm validates and submits the current planning form.
// Returns the command to run, or sets plan.err if invalid.
func (m *Model) submitPlanForm() tea.Cmd {
	switch m.plan.mode {
	case planTaskTime:
		return m.submitTaskTimeForm()
	case planEventForm:
		return m.submitEventForm()
	case planEdit:
		return m.submitEditForm()
	}
	return nil
}

func (m *Model) submitTaskTimeForm() tea.Cmd {
	startStr := strings.TrimSpace(m.plan.form.fields[0].Value())
	durStr := strings.TrimSpace(m.plan.form.fields[1].Value())

	var start int
	var timed bool
	if startStr != "" {
		var err error
		start, err = timeparse.ParseStart(startStr)
		if err != nil {
			m.plan.err = fmt.Errorf("invalid start time: %w", err)
			return nil
		}
		timed = true
	}
	var dur int
	if durStr != "" {
		var err error
		dur, err = timeparse.ParseDurationOrEnd(durStr, start)
		if err != nil {
			m.plan.err = fmt.Errorf("invalid duration: %w", err)
			return nil
		}
	}
	m.plan.err = nil
	return addPlanTaskCmd(m.planClient, m.plan.day, m.plan.form.taskID, start, dur, timed)
}

func (m *Model) submitEventForm() tea.Cmd {
	nameStr := strings.TrimSpace(m.plan.form.fields[0].Value())
	startStr := strings.TrimSpace(m.plan.form.fields[1].Value())
	durStr := strings.TrimSpace(m.plan.form.fields[2].Value())

	if nameStr == "" {
		m.plan.err = fmt.Errorf("event name cannot be empty")
		return nil
	}
	start, err := timeparse.ParseStart(startStr)
	if err != nil {
		m.plan.err = fmt.Errorf("invalid start time: %w", err)
		return nil
	}
	var dur int
	if durStr != "" {
		dur, err = timeparse.ParseDurationOrEnd(durStr, start)
		if err != nil {
			m.plan.err = fmt.Errorf("invalid duration: %w", err)
			return nil
		}
	}
	m.plan.err = nil
	return addPlanEventCmd(m.planClient, m.plan.day, nameStr, start, dur)
}

// submitEditForm validates and submits the unified Edit form using change
// detection against the pre-filled original values.
func (m *Model) submitEditForm() tea.Cmd {
	nameStr := strings.TrimSpace(m.plan.form.fields[0].Value())
	startStr := strings.TrimSpace(m.plan.form.fields[1].Value())
	durStr := strings.TrimSpace(m.plan.form.fields[2].Value())

	if nameStr == "" {
		m.plan.err = fmt.Errorf("name cannot be empty")
		return nil
	}

	// Find the original entry for change detection.
	var orig *planv1.PlanEntry
	for _, e := range m.plan.entries {
		if e.Id == m.plan.form.entryID {
			orig = e
			break
		}
	}

	var cmds []tea.Cmd

	if orig != nil && nameStr != orig.Name {
		cmds = append(cmds, renamePlanCmd(m.planClient, m.plan.day, m.plan.form.entryID, nameStr))
	}

	if startStr == "" {
		// Staying/becoming untimed (no start time). Parse an optional duration.
		var dur int
		if durStr != "" {
			var err error
			dur, err = timeparse.ParseDuration(durStr)
			if err != nil {
				m.plan.err = fmt.Errorf("invalid duration: %w", err)
				return nil
			}
		}
		wasTimed := orig != nil && orig.StartMinute != nil
		origDur := 0
		if orig != nil {
			origDur = int(orig.DurationMinute)
		}
		durChanged := durStr != "" && dur != origDur
		if wasTimed || durChanged {
			// dur==0 tells the server to keep the existing duration; a positive
			// value updates it. Either way StartMinute stays nil (untimed).
			sendDur := 0
			if durChanged {
				sendDur = dur
			}
			cmds = append(cmds, movePlanCmd(m.planClient, m.plan.day, m.plan.form.entryID, 0, sendDur, false))
		}
	} else {
		start, err := timeparse.ParseStart(startStr)
		if err != nil {
			m.plan.err = fmt.Errorf("invalid start time: %w", err)
			return nil
		}
		var dur int
		if durStr != "" {
			dur, err = timeparse.ParseDurationOrEnd(durStr, start)
			if err != nil {
				m.plan.err = fmt.Errorf("invalid duration: %w", err)
				return nil
			}
		}
		// Only move if start or duration changed from original.
		origStart := -1 // sentinel: was untimed
		origDur := 0
		if orig != nil && orig.StartMinute != nil {
			origStart = int(orig.GetStartMinute())
			origDur = int(orig.DurationMinute)
		}
		if origStart != start || origDur != dur {
			cmds = append(cmds, movePlanCmd(m.planClient, m.plan.day, m.plan.form.entryID, start, dur, true))
		}
	}

	m.plan.err = nil
	if len(cmds) == 0 {
		m.plan.mode = planList
		return nil
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}

// displayedPlanEntries filters the raw entry list for display: drops completed
// untimed entries unless their id matches pendingComplete (just-completed highlight).
// Timed entries and events are always kept; incomplete untimed entries are always kept.
func displayedPlanEntries(entries []*planv1.PlanEntry, pendingComplete *int32) []*planv1.PlanEntry {
	result := make([]*planv1.PlanEntry, 0, len(entries))
	for _, e := range entries {
		if e.StartMinute == nil && e.Completed {
			if pendingComplete == nil || e.Id != *pendingComplete {
				continue
			}
		}
		result = append(result, e)
	}
	return result
}

// ── plan reducer ──────────────────────────────────────────────────────────

// handlePlanEntriesMsg processes a ListPlanEntries response: replaces the
// entries, sets loaded, clamps cursor, and optionally highlights an entry by id.
func (m Model) handlePlanEntriesMsg(msg planEntriesMsg, _ int32) Model {
	// Background loads only apply to the tab and day they were dispatched for.
	if msg.bg && m.activeTab != tabPlanning {
		return m
	}
	// Drop a response for a day the user has already navigated away from.
	// Applies to user-initiated loads too: two fast day-nav presses leave two
	// fetches in flight, and without this the last to land wins regardless of
	// which day is on screen.
	if msg.day != "" && msg.day != m.plan.day {
		return m
	}
	// Background failures are silent: keep the existing data on screen.
	if msg.bg && msg.err != nil {
		return m
	}
	if msg.err != nil {
		m.plan.err = msg.err
		return m
	}
	// Only user-initiated loads clear the visible error.
	if !msg.bg {
		m.plan.err = nil
	}
	// Preserve cursor by entry id.
	var curID int32
	if len(m.plan.entries) > 0 && m.plan.cursor < len(m.plan.entries) {
		curID = m.plan.entries[m.plan.cursor].Id
	}
	m.plan.entries = displayedPlanEntries(msg.entries, m.plan.pendingComplete)
	m.plan.loaded = true

	if msg.highlightID != 0 {
		for i, e := range m.plan.entries {
			if e.Id == msg.highlightID {
				m.plan.cursor = i
				m.plan.lastLoad = m.nowOrDefault()
				return m
			}
		}
	}
	if curID != 0 {
		for i, e := range m.plan.entries {
			if e.Id == curID {
				m.plan.cursor = i
				m.plan.lastLoad = m.nowOrDefault()
				return m
			}
		}
	}
	m.plan.cursor = clampCursor(m.plan.cursor, len(m.plan.entries))
	m.plan.lastLoad = m.nowOrDefault()
	return m
}

// planIsToday returns true if the plan's in-view day is today.
func planIsToday(day string) bool {
	return day == time.Now().Format("2006-01-02")
}

// allTaskIDs returns a map of every task ID present in the tree (parents +
// all descendants). Used to build a fully-expanded picker visible list.
func allTaskIDs(tree []*cli.TreeNode) map[int64]bool {
	ids := make(map[int64]bool)
	var walk func([]*cli.TreeNode)
	walk = func(nodes []*cli.TreeNode) {
		for _, n := range nodes {
			ids[n.Task.Id] = true
			walk(n.Children)
		}
	}
	walk(tree)
	return ids
}
