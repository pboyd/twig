package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli"
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

func fetchAfterMutation(client taskv1connect.TaskServiceClient, highlightID int64) tea.Msg {
	resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		return refreshedMsg{err: err}
	}
	return refreshedMsg{tree: cli.BuildTree(resp.Msg.Tasks), highlightID: highlightID}
}

// ── Init ───────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return listTasksCmd(m.client)
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

	case pomodoroRequestMsg:
		m.mode = modePomodoro
		if msg.resume {
			return m, execPomodoroResume(m.client, m.pomConfig)
		}
		return m, execPomodoroStart(m.client, m.pomConfig, msg.taskID)

	case pomodoroDoneMsg:
		m.mode = modeList
		if msg.err != nil {
			m.err = msg.err
		}
		return m, listTasksCmd(m.client)

	case editSavedMsg:
		return m.handleEditSaved(msg)

	case editCancelledMsg:
		m.mode = modeList
		m.cursor = msg.originalCursor
		m.err = nil
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeList:
		return m.handleListKey(msg)
	case modeEdit, modeNewSubtask, modeNewRoot:
		return m.handleEditKey(msg)
	case modeHelp:
		return m.handleHelpKey(msg)
	}
	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		m.pendingComplete = nil
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		m.pendingComplete = nil
		if m.cursor < len(m.visible)-1 {
			m.cursor++
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

	case key.Matches(msg, m.keys.PomStart):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			m.err = nil
			return m, startPomodoroCmd(id)
		}

	case key.Matches(msg, m.keys.PomResume):
		m.err = nil
		return m, resumePomodoroCmd()

	case key.Matches(msg, m.keys.Delete):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			m.err = nil
			return m, deleteTaskCmd(m.client, id)
		}

	case key.Matches(msg, m.keys.Complete):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			m.err = nil
			return m, completeTaskCmd(m.client, id)
		}

	case key.Matches(msg, m.keys.Refresh):
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

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Help), key.Matches(msg, m.keys.Cancel):
		m.mode = modeList
	}
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
