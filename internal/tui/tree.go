package tui

import (
	"github.com/pboyd/twig/internal/cli"
)

// visibleSiblings returns the visible siblings of the task at taskID, given
// the current tree, expansion state, and completed-task filter. It returns the
// id of the previous visible sibling (0 if none) and the next visible sibling
// (0 if none). A sibling is visible when it would appear in the visible row
// list under current filter settings.
func visibleSiblings(tree []*cli.TreeNode, expanded map[int64]bool, showCompleted bool, taskID int64) (prev, next int64) {
	rows := buildVisible(tree, expanded, showCompleted, nil)

	targetIdx := -1
	for i, r := range rows {
		if r.node.Task.Id == taskID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return 0, 0
	}

	target := rows[targetIdx]

	sameParent := func(r *visibleRow) bool {
		if r.depth != target.depth {
			return false
		}
		ap, bp := r.node.Task.ParentId, target.node.Task.ParentId
		if ap == nil && bp == nil {
			return true
		}
		if ap == nil || bp == nil {
			return false
		}
		return *ap == *bp
	}

	for i := targetIdx - 1; i >= 0; i-- {
		r := rows[i]
		if r.depth < target.depth {
			break
		}
		if sameParent(r) {
			prev = r.node.Task.Id
			break
		}
	}

	for i := targetIdx + 1; i < len(rows); i++ {
		r := rows[i]
		if r.depth < target.depth {
			break
		}
		if sameParent(r) {
			next = r.node.Task.Id
			break
		}
	}

	return prev, next
}

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

	// hasVisibleChildren: at least one child would appear given current filter
	// settings. Computed unconditionally (regardless of expanded state) so that
	// the expandable/expanded flags are always driven by visible children, not
	// all children (which may include hidden completed ones).
	hasVisibleChildren := false
	if hasChildren {
		for _, child := range node.Children {
			childCompleted := child.Task.GetCompletedAt() != nil
			childIsPending := pendingComplete != nil && *pendingComplete == child.Task.Id
			if !childCompleted || showCompleted || childIsPending {
				hasVisibleChildren = true
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
	nodeExpandable := hasVisibleChildren
	nodeExpanded := hasVisibleChildren && isExpanded

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
