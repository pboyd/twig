package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli"
	"github.com/pboyd/todo/services/todo/internal/pomodoro"
)

// ── message types ──────────────────────────────────────────────────────────

// listTasksResultMsg carries the result of a bare ListTasks (init / Ctrl-R).
type listTasksResultMsg struct {
	tree []*cli.TreeNode
	err  error
}

// refreshedMsg is returned by mutation commands: RPC + re-fetch bundled together.
type refreshedMsg struct {
	tree        []*cli.TreeNode
	highlightID int64 // task to focus after refresh; 0 = use clamped cursor
	err         error
}

// moveTaskResultMsg carries the result of a move-parent UpdateTask call.
type moveTaskResultMsg struct {
	taskID int64
	tree   []*cli.TreeNode
	err    error
}

// ── command factories ───────────────────────────────────────────────────────

func listTasksCmd(client taskv1connect.TaskServiceClient) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return listTasksResultMsg{err: err}
		}
		return listTasksResultMsg{tree: cli.BuildTree(resp.Msg.Tasks)}
	}
}

func updateTaskCmd(client taskv1connect.TaskServiceClient, msg editSavedMsg) tea.Cmd {
	return func() tea.Msg {
		id := *msg.taskID
		req := &taskv1.UpdateTaskRequest{
			Id:          id,
			Name:        msg.name,
			Description: msg.description,
		}
		if msg.parentID != nil {
			req.ParentId = msg.parentID
		}
		if msg.dueStr != "" {
			ts, err := cli.ParseDue(msg.dueStr)
			if err != nil {
				return refreshedMsg{err: err}
			}
			req.Due = ts
		}

		_, err := client.UpdateTask(context.Background(), connect.NewRequest(req))
		if err != nil {
			return refreshedMsg{err: err}
		}
		return fetchAfterMutation(client, id)
	}
}

func createTaskCmd(client taskv1connect.TaskServiceClient, msg editSavedMsg) tea.Cmd {
	return func() tea.Msg {
		req := &taskv1.CreateTaskRequest{
			Name:        msg.name,
			Description: msg.description,
		}
		if msg.parentID != nil {
			req.ParentId = msg.parentID
		}
		if msg.dueStr != "" {
			ts, err := cli.ParseDue(msg.dueStr)
			if err != nil {
				return refreshedMsg{err: err}
			}
			req.Due = ts
		}

		resp, err := client.CreateTask(context.Background(), connect.NewRequest(req))
		if err != nil {
			return refreshedMsg{err: err}
		}
		newID := resp.Msg.Task.Id
		return fetchAfterMutation(client, newID)
	}
}

func deleteTaskCmd(client taskv1connect.TaskServiceClient, id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.DeleteTask(context.Background(), connect.NewRequest(&taskv1.DeleteTaskRequest{Id: id}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		// highlightID 0 means: caller will compute cursor from position.
		return refreshedMsg{tree: cli.BuildTree(resp.Msg.Tasks), highlightID: 0}
	}
}

func completeTaskCmd(client taskv1connect.TaskServiceClient, id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.CompleteTask(context.Background(), connect.NewRequest(&taskv1.CompleteTaskRequest{Id: id}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		return fetchAfterMutation(client, id)
	}
}

func uncompleteTaskCmd(client taskv1connect.TaskServiceClient, id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.UncompleteTask(context.Background(), connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: id}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		return fetchAfterMutation(client, id)
	}
}

func setEstimateCmd(client taskv1connect.TaskServiceClient, id int64, estimate int32) tea.Cmd {
	return func() tea.Msg {
		_, err := client.SetEstimate(context.Background(), connect.NewRequest(&taskv1.SetEstimateRequest{
			TaskId:   id,
			Estimate: estimate,
		}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		return fetchAfterMutation(client, id)
	}
}

func moveTaskCmd(client taskv1connect.TaskServiceClient, task *taskv1.Task, newParentID *int64) tea.Cmd {
	return func() tea.Msg {
		req := &taskv1.UpdateTaskRequest{
			Id:          task.Id,
			Name:        task.Name,
			Description: task.GetDescription(),
		}
		if task.Due != nil {
			req.Due = task.Due
		}
		if newParentID != nil {
			req.ParentId = newParentID
		}
		_, err := client.UpdateTask(context.Background(), connect.NewRequest(req))
		if err != nil {
			return moveTaskResultMsg{taskID: task.Id, err: err}
		}
		resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return moveTaskResultMsg{taskID: task.Id, err: err}
		}
		return moveTaskResultMsg{taskID: task.Id, tree: cli.BuildTree(resp.Msg.Tasks)}
	}
}

func fetchAfterMutation(client taskv1connect.TaskServiceClient, highlightID int64) tea.Msg {
	resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		return refreshedMsg{err: err}
	}
	return refreshedMsg{tree: cli.BuildTree(resp.Msg.Tasks), highlightID: highlightID}
}

// ── Init ───────────────────────────────────────────────────────────────────

func pomodoroRemaining(startAt time.Time, now time.Time) time.Duration {
	return pomodoro.Remaining(startAt, now)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(listTasksCmd(m.client), getActivePomCmd(m.client, m.tree))
}

// ── Update ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case listTasksResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Preserve cursor by task id.
		var curID int64
		if len(m.visible) > 0 && m.cursor < len(m.visible) {
			curID = m.visible[m.cursor].node.Task.Id
		}
		m.tree = msg.tree
		m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
		if curID != 0 {
			m.cursor = findCursor(m.visible, curID)
		} else {
			m.cursor = clampCursor(m.cursor, len(m.visible))
		}
		return m, nil

	case refreshedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.mode = modeList
			return m, nil
		}
		prevCursor := m.cursor
		m.tree = msg.tree
		m.err = nil
		m.mode = modeList
		m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
		if msg.highlightID != 0 {
			m.cursor = findCursor(m.visible, msg.highlightID)
			// If it's a subtask, ensure parent is expanded.
			if msg.highlightID != 0 {
				m.ensureVisible(msg.highlightID)
				m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
				m.cursor = findCursor(m.visible, msg.highlightID)
			}
		} else {
			m.cursor = clampCursor(prevCursor, len(m.visible))
		}
		return m, nil

	case pomTickMsg:
		if m.pom == nil || m.pom.completed {
			return m, nil
		}
		remaining := pomodoroRemaining(m.pom.startAt, time.Now())
		if remaining == 0 && !m.pom.completed {
			m.pom.completed = true
			taskName := m.pom.taskName
			return m, completePomCmd(m.client, taskName)
		}
		return m, pomTickCmd()

	case pomStartedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.pom = &activePom{
			taskID:   msg.taskID,
			taskName: msg.taskName,
			startAt:  msg.startAt,
		}
		m.err = nil
		return m, tea.Batch(pomTickCmd(), runPomHook(m.pomConfig.OnStart, "on_start"))

	case pomCancelledMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		hookCmd := runPomHook(m.pomConfig.OnCancel, "on_cancel")
		m.pom = nil
		m.err = nil
		return m, hookCmd

	case pomCompletedMsg:
		if m.pom != nil {
			banner := "🍅 Pomodoro complete! · " + msg.taskName
			if !m.styled {
				banner = "Pomodoro complete! " + msg.taskName
			}
			m.pom.banner = banner
		}
		if msg.err != nil {
			m.err = msg.err
		}
		hookCmd := runPomHook(m.pomConfig.OnComplete, "on_complete")
		return m, tea.Batch(hookCmd, pomBannerExpireCmd())

	case pomBannerExpireMsg:
		m.pom = nil
		return m, nil

	case pomHookErrMsg:
		m.err = msg.err
		return m, nil

	case pomActiveMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if msg.pom != nil {
			m.pom = msg.pom
			return m, pomTickCmd()
		}
		return m, nil

	case moveTaskResultMsg:
		return m.handleMoveTaskResult(msg)

	case editSavedMsg:
		return m.handleEditSaved(msg)

	case editCancelledMsg:
		m.mode = modeList
		m.cursor = msg.originalCursor
		m.err = nil
		return m, nil

	case planEntriesMsg:
		m = m.handlePlanEntriesMsg(msg, msg.highlightID)
		return m, nil

	case planTasksMsg:
		if msg.err != nil {
			m.plan.err = msg.err
			m.plan.mode = planList
			return m, nil
		}
		// Expand all nodes so sub-tasks are visible in the picker.
		expanded := allTaskIDs(msg.tree)
		m.plan.picker = pickerState{
			tree:     msg.tree,
			visible:  buildVisible(msg.tree, expanded, false, nil),
			cursor:   0,
			expanded: expanded,
		}
		// mode was already set to planPickTask by initAddTaskForm
		return m, nil

	case planMutatedMsg:
		if msg.err != nil {
			m.plan.err = msg.err
			return m, nil
		}
		return m, listPlanHighlightCmd(m.planClient, m.plan.day, msg.highlightID)

	case planTickMsg:
		if m.activeTab == tabPlanning && planIsToday(m.plan.day) {
			return m, planTickCmd()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.activeTab == tabPlanning {
		return m.handlePlanningKey(msg)
	}
	switch m.mode {
	case modeList:
		return m.handleListKey(msg)
	case modeEdit, modeNewSubtask, modeNewRoot:
		return m.handleEditKey(msg)
	case modeHelp:
		return m.handleHelpKey(msg)
	case modeMove:
		return m.handleMoveKey(msg)
	}
	return m, nil
}

// handlePlanningKey handles all key events while the Planning tab is active.
func (m Model) handlePlanningKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Dismiss help overlay — any key closes it, mirroring handleHelpKey.
	if m.mode == modeHelp {
		m.mode = modeList
		return m, nil
	}

	// Handle quit confirmation overlay (mirrors handleListKey).
	if m.confirmingQuit {
		switch msg.String() {
		case "y":
			return m, tea.Quit
		case "n", "esc":
			m.confirmingQuit = false
		}
		return m, nil
	}

	// Quit is always available.
	if key.Matches(msg, m.keys.Quit) {
		if m.pom != nil && !m.pom.completed {
			m.confirmingQuit = true
			return m, nil
		}
		return m, tea.Quit
	}

	// While a modal is open, route to the modal handler; tab-switch is ignored.
	if m.plan.mode != planList {
		return m.handlePlanModalKey(msg)
	}

	// Tab / Shift+Tab switches to the Tasks tab (not blocked — planList mode).
	if key.Matches(msg, m.keys.NextTab) || key.Matches(msg, m.keys.PrevTab) {
		m.activeTab = tabTasks
		m.keys.PlanningMode = false
		m.plan.err = nil
		return m, nil
	}

	// Navigation.
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.plan.cursor > 0 {
			m.plan.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.plan.cursor < len(m.plan.entries)-1 {
			m.plan.cursor++
		}

	// Day navigation.
	case key.Matches(msg, m.keys.PlanPrevDay):
		t, err := time.Parse("2006-01-02", m.plan.day)
		if err == nil {
			m.plan.day = t.AddDate(0, 0, -1).Format("2006-01-02")
			m.plan.loaded = false
			return m, listPlanCmd(m.planClient, m.plan.day)
		}
	case key.Matches(msg, m.keys.PlanNextDay):
		t, err := time.Parse("2006-01-02", m.plan.day)
		if err == nil {
			m.plan.day = t.AddDate(0, 0, 1).Format("2006-01-02")
			m.plan.loaded = false
			return m, listPlanCmd(m.planClient, m.plan.day)
		}
	case key.Matches(msg, m.keys.PlanToday):
		today := time.Now().Format("2006-01-02")
		if m.plan.day != today {
			m.plan.day = today
			m.plan.loaded = false
			return m, tea.Batch(listPlanCmd(m.planClient, m.plan.day), planTickCmd())
		}
	case key.Matches(msg, m.keys.Refresh):
		m.plan.loaded = false
		return m, listPlanCmd(m.planClient, m.plan.day)

	// Entry actions.
	case key.Matches(msg, m.keys.PlanAddTask):
		m.initAddTaskForm()
		return m, listTasksForPickerCmd(m.client)
	case key.Matches(msg, m.keys.PlanAddEvent):
		m.initAddEventForm()
	case key.Matches(msg, m.keys.PlanEdit):
		if len(m.plan.entries) > 0 {
			m.initEditForm()
		}
	case key.Matches(msg, m.keys.PlanRemove):
		if len(m.plan.entries) > 0 {
			entry := m.plan.entries[m.plan.cursor]
			return m, removePlanCmd(m.planClient, m.plan.day, entry.Id)
		}

	// Pomodoro cancel — mirrors the Tasks tab handler.
	case key.Matches(msg, m.keys.PomCancel):
		if m.pom != nil && !m.pom.completed {
			m.err = nil
			return m, cancelPomCmd(m.client)
		}

	// Help.
	case key.Matches(msg, m.keys.Help):
		m.mode = modeHelp
	}

	return m, nil
}

// handlePlanModalKey handles keys when a planning modal is open.
func (m Model) handlePlanModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.plan.mode {
	case planPickTask:
		return m.handlePickerKey(msg)
	case planTaskTime, planEventForm, planEdit:
		return m.handlePlanFormKey(msg)
	}
	return m, nil
}

// handlePickerKey handles key events in the task picker.
func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.plan.mode = planList
		m.plan.err = nil
	case key.Matches(msg, m.keys.Up):
		if m.plan.picker.cursor > 0 {
			m.plan.picker.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.plan.picker.cursor < len(m.plan.picker.visible)-1 {
			m.plan.picker.cursor++
		}
	case msg.Type == tea.KeyEnter:
		if len(m.plan.picker.visible) > 0 {
			row := m.plan.picker.visible[m.plan.picker.cursor]
			taskID := row.node.Task.Id
			m.initTaskTimeForm(taskID)
		} else {
			m.plan.mode = planList
		}
	}
	return m, nil
}

// handlePlanFormKey handles key events in a planning text-input form.
func (m Model) handlePlanFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.plan.mode = planList
		m.plan.err = nil
		return m, nil
	case key.Matches(msg, m.keys.Save) || msg.Type == tea.KeyEnter:
		cmd := m.submitPlanForm()
		return m, cmd
	case key.Matches(msg, m.keys.Tab):
		m.cyclePlanFormFocus(1)
		return m, nil
	case key.Matches(msg, m.keys.ShiftTab):
		m.cyclePlanFormFocus(-1)
		return m, nil
	}

	// Forward to focused text input.
	if len(m.plan.form.fields) > 0 {
		var cmd tea.Cmd
		m.plan.form.fields[m.plan.form.focus], cmd = m.plan.form.fields[m.plan.form.focus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle quit confirmation overlay.
	if m.confirmingQuit {
		switch msg.String() {
		case "y":
			return m, tea.Quit
		case "n", "esc":
			m.confirmingQuit = false
		}
		return m, nil
	}

	// Clear completion banner on any keypress (banner clears, key still acts).
	if m.pom != nil && m.pom.completed && m.pom.banner != "" {
		m.pom = nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.pom != nil && !m.pom.completed {
			m.confirmingQuit = true
			return m, nil
		}
		return m, tea.Quit

	case key.Matches(msg, m.keys.NextTab) || key.Matches(msg, m.keys.PrevTab):
		m.activeTab = tabPlanning
		m.keys.PlanningMode = true
		m.err = nil
		var cmds []tea.Cmd
		cmds = append(cmds, listPlanCmd(m.planClient, m.plan.day))
		if planIsToday(m.plan.day) {
			cmds = append(cmds, planTickCmd())
		}
		return m, tea.Batch(cmds...)

	case key.Matches(msg, m.keys.Up):
		m.pendingComplete = nil
		m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
		if m.cursor > 0 {
			m.cursor--
		}
		m.cursor = clampCursor(m.cursor, len(m.visible))

	case key.Matches(msg, m.keys.Down):
		var targetID int64
		if m.cursor+1 < len(m.visible) {
			targetID = m.visible[m.cursor+1].node.Task.Id
		}
		m.pendingComplete = nil
		m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
		if targetID != 0 {
			m.cursor = findCursor(m.visible, targetID)
		} else {
			m.cursor = clampCursor(m.cursor, len(m.visible))
		}

	case key.Matches(msg, m.keys.Collapse):
		if len(m.visible) > 0 {
			row := m.visible[m.cursor]
			id := row.node.Task.Id
			if len(row.node.Children) > 0 && m.expanded[id] {
				// Task is expanded — collapse it in place.
				m.expanded[id] = false
			} else if parent := findParentNode(m.tree, id); parent != nil {
				// Task is a leaf or already collapsed — collapse the parent and
				// move the cursor up to it.
				m.expanded[parent.Task.Id] = false
				m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
				m.cursor = findCursor(m.visible, parent.Task.Id)
				return m, nil
			}
			m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
		}

	case key.Matches(msg, m.keys.Expand):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			m.expanded[id] = true
			m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
			// Move to first child if it is visible immediately after the cursor.
			// Works whether the subtree was just expanded or was already open.
			if next := m.cursor + 1; next < len(m.visible) {
				if pid := m.visible[next].node.Task.ParentId; pid != nil && *pid == id {
					m.cursor = next
				}
			}
		}

	case key.Matches(msg, m.keys.Edit):
		if len(m.visible) > 0 {
			task := m.visible[m.cursor].node.Task
			m.originalCursor = m.cursor
			m.edit = NewEditForm(task, m.cursor)
			m.mode = modeEdit
			m.err = nil
		}

	case key.Matches(msg, m.keys.NewSub):
		if len(m.visible) > 0 {
			parentID := m.visible[m.cursor].node.Task.Id
			m.originalCursor = m.cursor
			m.edit = NewSubtaskForm(parentID, m.cursor)
			m.mode = modeNewSubtask
			m.err = nil
		}

	case key.Matches(msg, m.keys.NewRoot):
		m.originalCursor = m.cursor
		m.edit = NewRootForm(m.cursor)
		m.mode = modeNewRoot
		m.err = nil

	case key.Matches(msg, m.keys.Move):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			if ms := newMoveState(&m, id); ms != nil {
				m.move = ms
				m.mode = modeMove
			}
		}

	case key.Matches(msg, m.keys.PomStart):
		if len(m.visible) > 0 {
			task := m.visible[m.cursor].node.Task
			m.err = nil
			return m, startPomCmd(m.client, task.Id, task.Name)
		}

	case key.Matches(msg, m.keys.PomCancel):
		if m.pom != nil && !m.pom.completed {
			m.err = nil
			return m, cancelPomCmd(m.client)
		}

	case key.Matches(msg, m.keys.Delete):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			m.err = nil
			return m, deleteTaskCmd(m.client, id)
		}

	case key.Matches(msg, m.keys.Complete):
		if len(m.visible) > 0 {
			node := m.visible[m.cursor].node
			id := node.Task.Id
			m.err = nil
			if node.Task.GetCompletedAt() == nil {
				m.pendingComplete = &id
				return m, completeTaskCmd(m.client, id)
			}
			m.pendingComplete = nil
			return m, uncompleteTaskCmd(m.client, id)
		}

	case key.Matches(msg, m.keys.Refresh):
		m.pendingComplete = nil
		m.err = nil
		return m, listTasksCmd(m.client)

	case key.Matches(msg, m.keys.Filter):
		m.showCompleted = !m.showCompleted
		var curID int64
		if len(m.visible) > 0 {
			curID = m.visible[m.cursor].node.Task.Id
		}
		m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
		m.cursor = findCursor(m.visible, curID)

	case key.Matches(msg, m.keys.Help):
		m.mode = modeHelp

	default:
		// Digit keys 0-9 set the pomodoro estimate.
		if len(msg.Runes) == 1 {
			r := msg.Runes[0]
			if r >= '0' && r <= '9' && len(m.visible) > 0 {
				id := m.visible[m.cursor].node.Task.Id
				est := int32(r - '0')
				m.err = nil
				return m, setEstimateCmd(m.client, id, est)
			}
		}
	}

	return m, nil
}

func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newEdit, cmd := m.edit.Update(msg, m.keys)
	m.edit = newEdit
	return m, cmd
}

func (m Model) handleHelpKey(_ tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeList
	return m, nil
}

func (m Model) handleEditSaved(msg editSavedMsg) (tea.Model, tea.Cmd) {
	// Validate name.
	if strings.TrimSpace(msg.name) == "" {
		m.err = fmt.Errorf("task name cannot be empty")
		return m, nil
	}

	// Validate estimate if provided.
	if msg.estimateStr != "" {
		_, err := strconv.ParseInt(msg.estimateStr, 10, 32)
		if err != nil {
			m.err = fmt.Errorf("estimate must be an integer")
			return m, nil
		}
	}

	// Validate due if provided (ParseDue is called inside the cmd; we do a
	// pre-check here so validation errors stay in edit mode).
	if msg.dueStr != "" {
		if _, err := cli.ParseDue(msg.dueStr); err != nil {
			m.err = err
			return m, nil
		}
	}

	m.err = nil

	if msg.taskID != nil {
		return m, updateTaskCmd(m.client, msg)
	}

	// New task: after create, expand the parent so the new subtask is visible.
	if msg.parentID != nil {
		m.expanded[*msg.parentID] = true
	}
	return m, createTaskCmd(m.client, msg)
}

func (m Model) handleMoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.move == nil {
		m.mode = modeList
		return m, nil
	}
	cmd, keepOpen := m.move.Update(msg, &m)
	if !keepOpen {
		m.mode = modeList
		m.move = nil
	}
	return m, cmd
}

func (m Model) handleMoveTaskResult(msg moveTaskResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		if m.move != nil {
			m.move.errMsg = msg.err.Error()
		}
		return m, nil
	}
	m.mode = modeList
	m.move = nil
	m.err = nil
	m.tree = msg.tree
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
	if msg.taskID != 0 {
		m.ensureVisible(msg.taskID)
		m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
		m.cursor = findCursor(m.visible, msg.taskID)
	} else {
		m.cursor = clampCursor(m.cursor, len(m.visible))
	}
	return m, nil
}

// ensureVisible walks the tree to find the task with the given id and expands
// all its ancestors so it appears in the visible list.
func (m *Model) ensureVisible(id int64) {
	var walk func(nodes []*cli.TreeNode) bool
	walk = func(nodes []*cli.TreeNode) bool {
		for _, n := range nodes {
			if n.Task.Id == id {
				return true
			}
			if walk(n.Children) {
				m.expanded[n.Task.Id] = true
				return true
			}
		}
		return false
	}
	walk(m.tree)
}

// ── helpers ────────────────────────────────────────────────────────────────

// findParentNode returns the tree node whose Children slice contains childID,
// or nil if childID is a root task.
func findParentNode(tree []*cli.TreeNode, childID int64) *cli.TreeNode {
	for _, node := range tree {
		for _, child := range node.Children {
			if child.Task.Id == childID {
				return node
			}
		}
		if found := findParentNode(node.Children, childID); found != nil {
			return found
		}
	}
	return nil
}

func findCursor(visible []*visibleRow, taskID int64) int {
	for i, r := range visible {
		if r.node.Task.Id == taskID {
			return i
		}
	}
	return 0
}

func clampCursor(cur, length int) int {
	if length == 0 {
		return 0
	}
	if cur >= length {
		return length - 1
	}
	if cur < 0 {
		return 0
	}
	return cur
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
