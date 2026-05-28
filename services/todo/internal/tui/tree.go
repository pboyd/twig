package tui

import (
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// visibleRow is one rendered row in the task list pane.
type visibleRow struct {
	node       *cli.TreeNode
	depth      int
	treePrefix string
	expandable bool
	expanded   bool
}

// buildVisible flattens the tree into visible rows applying expansion state and
// completed-task filtering. Matches the spec's view-model.md "Visible-row
// flattening" algorithm.
func buildVisible(tree []*cli.TreeNode, expanded map[int64]bool, showCompleted bool, pendingComplete *int64) []*visibleRow {
	var rows []*visibleRow
	for i, root := range tree {
		last := i == len(tree)-1
		emitNode(root, 0, "", last, expanded, showCompleted, pendingComplete, &rows)
	}
	return rows
}

func emitNode(
	node *cli.TreeNode,
	depth int,
	parentPrefix string,
	isLast bool,
	expanded map[int64]bool,
	showCompleted bool,
	pendingComplete *int64,
	rows *[]*visibleRow,
) {
	id := node.Task.Id
	completed := node.Task.GetCompletedAt() != nil

	// Skip completed tasks unless filter is on or this task is pendingComplete.
	isPending := pendingComplete != nil && *pendingComplete == id
	if completed && !showCompleted && !isPending {
		return
	}

	hasChildren := len(node.Children) > 0
	isExpanded := expanded[id]

	// [+] only when collapsing would reveal children — i.e. at least one child
	// would be visible given the current filter settings.
	hasExpandableChildren := false
	if hasChildren && !isExpanded {
		for _, child := range node.Children {
			childCompleted := child.Task.GetCompletedAt() != nil
			childIsPending := pendingComplete != nil && *pendingComplete == child.Task.Id
			if !childCompleted || showCompleted || childIsPending {
				hasExpandableChildren = true
				break
			}
		}
	}

	// Compute this row's connector and the prefix for children.
	var connector string
	var childPrefix string
	if depth == 0 {
		connector = ""
		childPrefix = ""
	} else {
		if isLast {
			connector = "└── "
			childPrefix = parentPrefix + "    "
		} else {
			connector = "├── "
			childPrefix = parentPrefix + "│   "
		}
	}

	treePrefix := parentPrefix + connector

	// expandable: has children that can be toggled (collapsed-with-visible-children,
	// or currently expanded). expanded: is currently open.
	nodeExpandable := hasExpandableChildren || (hasChildren && isExpanded)
	nodeExpanded := hasChildren && isExpanded

	*rows = append(*rows, &visibleRow{
		node:       node,
		depth:      depth,
		treePrefix: treePrefix,
		expandable: nodeExpandable,
		expanded:   nodeExpanded,
	})

	if hasChildren && isExpanded {
		for i, child := range node.Children {
			lastChild := i == len(node.Children)-1
			emitNode(child, depth+1, childPrefix, lastChild, expanded, showCompleted, pendingComplete, rows)
		}
	}
}
