package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli"
	"github.com/pboyd/todo/services/todo/internal/config"
)

// ExportBuildVisible exposes buildVisible for tests.
func ExportBuildVisible(tree []*cli.TreeNode, expanded map[int64]bool, showCompleted bool, pendingComplete *int64) []*visibleRow {
	return buildVisible(tree, expanded, showCompleted, pendingComplete)
}

// ExportNewModel creates a Model with a fake tree for unit tests.
func ExportNewModel(client taskv1connect.TaskServiceClient, tree []*cli.TreeNode) Model {
	m := newModel(client, "", config.PomodoroConfig{})
	m.tree = tree
	m.visible = buildVisible(tree, m.expanded, m.showCompleted, m.pendingComplete)
	return m
}

// ExportVisibleRow exposes the visibleRow type for inspection in tests.
type ExportVisibleRow = visibleRow

// ExportUpdateTaskCmd exposes updateTaskCmd for unit tests.
func ExportUpdateTaskCmd(client taskv1connect.TaskServiceClient, msg editSavedMsg) tea.Cmd {
	return updateTaskCmd(client, msg)
}

// ExportNewStyledModel creates a Model with the styled flag set for rendering tests.
func ExportNewStyledModel(client taskv1connect.TaskServiceClient, tree []*cli.TreeNode, styled bool) Model {
	m := ExportNewModel(client, tree)
	m.styled = styled
	return m
}

// ExportRenderList exposes renderList for unit tests.
func ExportRenderList(m Model, width int) string {
	return m.renderList(width)
}

// ExportRenderDetails exposes renderDetails for unit tests.
func ExportRenderDetails(task *taskv1.Task, width int) string {
	return renderDetails(task, width)
}
