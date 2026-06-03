package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
)

// moveCandidate is one entry in the move-dialog candidate list.
type moveCandidate struct {
	taskID int64
	label  string
	depth  int
}

// moveState holds all state for the "move task" dialog.
type moveState struct {
	taskID     int64
	task       *taskv1.Task
	candidates []moveCandidate
	cursor     int
	errMsg     string
}

// buildCandidates produces the ordered candidate list for moving taskID.
// The sentinel "(no parent)" is always index 0. Incomplete tasks are listed
// in tree-walk order (DFS), excluding the moving task and all its descendants.
func buildCandidates(tree []*cli.TreeNode, movingID int64) []moveCandidate {
	candidates := []moveCandidate{{taskID: 0, label: "(no parent)", depth: 0}}

	var walk func(nodes []*cli.TreeNode, depth int)
	walk = func(nodes []*cli.TreeNode, depth int) {
		for _, node := range nodes {
			if node.Task.Id == movingID {
				// Skip the moving task and all its descendants.
				continue
			}
			if node.Task.GetCompletedAt() != nil {
				// Completed tasks are not valid parents, but their children may be.
				walk(node.Children, depth+1)
				continue
			}
			candidates = append(candidates, moveCandidate{
				taskID: node.Task.Id,
				label:  node.Task.Name,
				depth:  depth,
			})
			walk(node.Children, depth+1)
		}
	}
	walk(tree, 0)
	return candidates
}

// findTaskInTree returns the task with the given id, or nil if not found.
func findTaskInTree(tree []*cli.TreeNode, id int64) *taskv1.Task {
	for _, node := range tree {
		if node.Task.Id == id {
			return node.Task
		}
		if found := findTaskInTree(node.Children, id); found != nil {
			return found
		}
	}
	return nil
}

// newMoveState constructs a moveState for moving taskID. Returns nil if the
// task is not found in the model tree.
func newMoveState(m *Model, taskID int64) *moveState {
	task := findTaskInTree(m.tree, taskID)
	if task == nil {
		return nil
	}
	candidates := buildCandidates(m.tree, taskID)

	// Pre-select the current parent (or index 0 for top-level tasks).
	cursor := 0
	if task.ParentId != nil {
		currentParent := task.GetParentId()
		for i, c := range candidates {
			if c.taskID == currentParent {
				cursor = i
				break
			}
		}
	}

	return &moveState{
		taskID:     taskID,
		task:       task,
		candidates: candidates,
		cursor:     cursor,
	}
}

// Update handles key events for the move dialog.
// Returns (cmd, keepOpen): keepOpen=false means the caller should close the dialog.
func (s *moveState) Update(msg tea.Msg, m *Model) (tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, true
	}

	switch {
	case key.Matches(keyMsg, m.keys.Up):
		if s.cursor > 0 {
			s.cursor--
		}

	case key.Matches(keyMsg, m.keys.Down):
		if s.cursor < len(s.candidates)-1 {
			s.cursor++
		}

	case keyMsg.Type == tea.KeyEnter:
		chosen := s.candidates[s.cursor]
		var newParentID *int64
		if chosen.taskID != 0 {
			id := chosen.taskID
			newParentID = &id
		}
		// keepOpen=true while the RPC is in flight; result handled by moveTaskResultMsg.
		return moveTaskCmd(m.client, s.task, newParentID), true

	case key.Matches(keyMsg, m.keys.Cancel):
		return nil, false
	}

	return nil, true
}

// View renders the move dialog panel.
func (s *moveState) View(width, _ int) string {
	var sb strings.Builder
	sb.WriteString("Move task: select new parent\n\n")

	for i, c := range s.candidates {
		indent := strings.Repeat("  ", c.depth)
		line := indent + c.label
		if lipgloss.Width(line) > width {
			line = line[:width]
		}
		if i == s.cursor {
			line = highlightStyle.Render(padRightAnsi(line, width))
		}
		sb.WriteString(line + "\n")
	}

	if s.errMsg != "" {
		sb.WriteString("\n" + errorStyle.Render(s.errMsg))
	}

	sb.WriteString("\n↑/↓: navigate  Enter: confirm  Esc: cancel")
	return sb.String()
}
