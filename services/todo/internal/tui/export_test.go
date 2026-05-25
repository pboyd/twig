package tui

import (
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// ExportBuildVisible exposes buildVisible for tests.
func ExportBuildVisible(tree []*cli.TreeNode, expanded map[int64]bool, showCompleted bool, pendingComplete *int64) []*visibleRow {
	return buildVisible(tree, expanded, showCompleted, pendingComplete)
}

// ExportNewModel creates a Model with a fake tree for unit tests.
func ExportNewModel(client taskv1connect.TaskServiceClient, tree []*cli.TreeNode) Model {
	m := newModel(client, "")
	m.tree = tree
	m.visible = buildVisible(tree, m.expanded, m.showCompleted, m.pendingComplete)
	return m
}

// ExportVisibleRow exposes the visibleRow type for inspection in tests.
type ExportVisibleRow = visibleRow
