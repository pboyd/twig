package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	planv1 "github.com/pboyd/twig/services/twig/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/services/twig/gen/plan/v1/planv1connect"
	taskv1 "github.com/pboyd/twig/services/twig/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/services/twig/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/services/twig/internal/cli"
	"github.com/pboyd/twig/services/twig/internal/cli/timeparse"
)

// ── message types ──────────────────────────────────────────────────────────

// planEntriesMsg carries the result of a ListPlanEntries call.
type planEntriesMsg struct {
	entries     []*planv1.PlanEntry
	highlightID int32 // entry to highlight after loading; 0 = clamp cursor
	err         error
}

// planMutatedMsg is returned after a plan mutation: carries the entry to highlight on reload.
type planMutatedMsg struct {
	highlightID int32
	err         error
}

// planTasksMsg carries the task tree for the task picker.
type planTasksMsg struct {
	tree []*cli.TreeNode
	err  error
}

// planTickMsg is fired by the planning-scoped 1-second tick.
type planTickMsg struct{}

// ── command factories ──────────────────────────────────────────────────────

func listPlanCmd(client planv1connect.PlanServiceClient, day string) tea.Cmd {
	return listPlanHighlightCmd(client, day, 0)
}

func listPlanHighlightCmd(client planv1connect.PlanServiceClient, day string, highlightID int32) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListPlanEntries(context.Background(), connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
		if err != nil {
			return planEntriesMsg{err: err}
		}
		return planEntriesMsg{entries: resp.Msg.Entries, highlightID: highlightID}
	}
}

func addPlanTaskCmd(client planv1connect.PlanServiceClient, day string, taskID int64, start, dur int) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.AddPlanTask(context.Background(), connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day:            day,
			TaskId:         taskID,
			StartMinute:    int32(start),
			DurationMinute: int32(dur),
		}))
		if err != nil {
			return planMutatedMsg{err: err}
		}
		return planMutatedMsg{highlightID: resp.Msg.Entry.GetId()}
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

func movePlanCmd(client planv1connect.PlanServiceClient, day string, id int32, start, dur int) tea.Cmd {
	return func() tea.Msg {
		_, err := client.MovePlanEntry(context.Background(), connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day:            day,
			Id:             id,
			StartMinute:    int32(start),
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
	m.plan.mode = planEventForm
}

// initTaskTimeForm opens the start/duration form for adding a task entry.
func (m *Model) initTaskTimeForm(taskID int64) {
	start := newPlanInput("Start time (e.g. 09:00)")
	start.Focus()
	dur := newPlanInput("Duration (e.g. 30m, optional)")

	m.plan.form = planFormState{
		fields: []textinput.Model{start, dur},
		focus:  0,
		taskID: taskID,
	}
	m.plan.mode = planTaskTime
}

// initEditForm opens the unified edit form for the selected entry.
// Name is prefilled; Start and Duration are blank (blank = keep existing).
func (m *Model) initEditForm() {
	if len(m.plan.entries) == 0 {
		return
	}
	entry := m.plan.entries[m.plan.cursor]
	name := newPlanInput("Name")
	name.SetValue(entry.Name)
	name.Focus()
	start := newPlanInput("New start time (e.g. 09:00, blank=keep)")
	dur := newPlanInput("New duration (e.g. 30m, blank=keep)")

	m.plan.form = planFormState{
		fields:  []textinput.Model{name, start, dur},
		focus:   0,
		entryID: entry.Id,
	}
	m.plan.mode = planEdit
}

// cyclePlanFormFocus moves focus to the next/prev field in the current form.
func (m *Model) cyclePlanFormFocus(delta int) {
	if len(m.plan.form.fields) == 0 {
		return
	}
	m.plan.form.fields[m.plan.form.focus].Blur()
	n := len(m.plan.form.fields)
	m.plan.form.focus = ((m.plan.form.focus+delta)%n + n) % n
	m.plan.form.fields[m.plan.form.focus].Focus()
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
	m.plan.mode = planList
	return addPlanTaskCmd(m.planClient, m.plan.day, m.plan.form.taskID, start, dur)
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
	m.plan.mode = planList
	return addPlanEventCmd(m.planClient, m.plan.day, nameStr, start, dur)
}

// submitEditForm validates and submits the unified Edit form.
// Issues RenamePlanEntry if the name changed, MovePlanEntry if a start time was given.
// If neither changed, closes the form without an RPC (no-op).
func (m *Model) submitEditForm() tea.Cmd {
	nameStr := strings.TrimSpace(m.plan.form.fields[0].Value())
	startStr := strings.TrimSpace(m.plan.form.fields[1].Value())
	durStr := strings.TrimSpace(m.plan.form.fields[2].Value())

	if nameStr == "" {
		m.plan.err = fmt.Errorf("name cannot be empty")
		return nil
	}

	// Look up the original entry to detect whether the name changed.
	var originalName string
	for _, e := range m.plan.entries {
		if e.Id == m.plan.form.entryID {
			originalName = e.Name
			break
		}
	}

	var cmds []tea.Cmd

	if nameStr != originalName {
		cmds = append(cmds, renamePlanCmd(m.planClient, m.plan.day, m.plan.form.entryID, nameStr))
	}

	if startStr != "" {
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
		cmds = append(cmds, movePlanCmd(m.planClient, m.plan.day, m.plan.form.entryID, start, dur))
	}

	m.plan.err = nil
	m.plan.mode = planList
	if len(cmds) == 0 {
		return nil
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}

// ── plan reducer ──────────────────────────────────────────────────────────

// handlePlanEntriesMsg processes a ListPlanEntries response: replaces the
// entries, sets loaded, clamps cursor, and optionally highlights an entry by id.
func (m Model) handlePlanEntriesMsg(msg planEntriesMsg, _ int32) Model {
	if msg.err != nil {
		m.plan.err = msg.err
		return m
	}
	m.plan.err = nil
	m.plan.entries = msg.entries
	m.plan.loaded = true

	if msg.highlightID != 0 {
		for i, e := range m.plan.entries {
			if e.Id == msg.highlightID {
				m.plan.cursor = i
				return m
			}
		}
	}
	m.plan.cursor = clampCursor(m.plan.cursor, len(m.plan.entries))
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
