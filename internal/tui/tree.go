package tui

import (
	"time"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
)

// taskIsSnoozed reports whether the task should be hidden on the given local
// day. A task is snoozed when snooze_until is set and its UTC calendar date is
// strictly after today (the client's local calendar date).
func taskIsSnoozed(task *taskv1.Task, today time.Time) bool {
	if task.SnoozeUntil == nil {
		return false
	}
	snoozeUTC := task.SnoozeUntil.AsTime().UTC()
	snoozeDay := time.Date(snoozeUTC.Year(), snoozeUTC.Month(), snoozeUTC.Day(), 0, 0, 0, 0, time.UTC)
	todayUTC := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	return snoozeDay.After(todayUTC)
}

// visibleSiblings returns the visible siblings of the task at taskID, given
// the current tree, expansion state, and completed-task filter. It returns the
// id of the previous visible sibling (0 if none) and the next visible sibling
// (0 if none). A sibling is visible when it would appear in the visible row
// list under current filter settings.
func visibleSiblings(tree []*cli.TreeNode, expanded map[int64]bool, showAll bool, taskID int64) (prev, next int64) {
	rows := buildVisible(tree, expanded, showAll, nil, time.Now().Local())

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
// completed-task / snooze filtering. Matches the spec's view-model.md
// "Visible-row flattening" algorithm. today is the client's local calendar day
// used to evaluate the snooze predicate.
func buildVisible(tree []*cli.TreeNode, expanded map[int64]bool, showAll bool, pendingComplete *int64, today time.Time) []*visibleRow {
	var rows []*visibleRow
	for i, root := range tree {
		last := i == len(tree)-1
		emitNode(root, 0, "", last, expanded, showAll, pendingComplete, today, &rows)
	}
	return rows
}

// buildVisibleFiltered builds the visible row list for filter mode. Per the
// display contract, collapsed/expansion state is ignored while a filter is
// active, so it flattens the tree fully expanded (and with showAll=true, to
// include completed/snoozed tasks), then keeps only matched rows plus the
// true ancestors of each matched row (walked via each task's actual
// parent_id), preserving tree order.
func buildVisibleFiltered(tree []*cli.TreeNode, expanded map[int64]bool, showAll bool, pendingComplete *int64, today time.Time, filteredIDs map[int64]bool) []*visibleRow {
	// Flatten ignoring expansion/collapse state so every descendant is
	// present regardless of the caller's expanded map.
	allExpanded := make(map[int64]bool)
	markAllExpanded(tree, allExpanded)
	all := buildVisible(tree, allExpanded, true, pendingComplete, today)

	byID := make(map[int64]*visibleRow, len(all))
	for _, row := range all {
		byID[row.node.Task.Id] = row
	}

	include := make(map[int64]bool, len(all))
	for _, row := range all {
		if !filteredIDs[row.node.Task.Id] {
			continue
		}
		// Include the matched row itself, then walk its true ancestor chain
		// via parent_id (not depth), stopping once we hit an already-included
		// ancestor or run out of parents.
		id := row.node.Task.Id
		for {
			if include[id] {
				break
			}
			include[id] = true
			r, ok := byID[id]
			if !ok || r.node.Task.ParentId == nil {
				break
			}
			id = *r.node.Task.ParentId
		}
	}

	var rows []*visibleRow
	for _, row := range all {
		if include[row.node.Task.Id] {
			rows = append(rows, row)
		}
	}
	return rows
}

// markAllExpanded sets expanded[id]=true for every node in the tree so a
// flatten pass visits every descendant regardless of collapse state.
func markAllExpanded(tree []*cli.TreeNode, expanded map[int64]bool) {
	for _, node := range tree {
		expanded[node.Task.Id] = true
		markAllExpanded(node.Children, expanded)
	}
}

func emitNode(
	node *cli.TreeNode,
	depth int,
	parentPrefix string,
	isLast bool,
	expanded map[int64]bool,
	showAll bool,
	pendingComplete *int64,
	today time.Time,
	rows *[]*visibleRow,
) {
	id := node.Task.Id
	completed := node.Task.GetCompletedAt() != nil
	snoozed := taskIsSnoozed(node.Task, today)

	// Skip completed or future-snoozed tasks unless "show all" is on, or this
	// task is pendingComplete.
	isPending := pendingComplete != nil && *pendingComplete == id
	if (completed || snoozed) && !showAll && !isPending {
		return
	}

	hasChildren := len(node.Children) > 0
	isExpanded := expanded[id]

	// hasVisibleChildren: at least one child would appear given current filter
	// settings. Computed unconditionally (regardless of expanded state) so that
	// the expandable/expanded flags are always driven by visible children, not
	// all children (which may include hidden completed/snoozed ones).
	hasVisibleChildren := false
	if hasChildren {
		for _, child := range node.Children {
			childCompleted := child.Task.GetCompletedAt() != nil
			childSnoozed := taskIsSnoozed(child.Task, today)
			childIsPending := pendingComplete != nil && *pendingComplete == child.Task.Id
			if (!childCompleted && !childSnoozed) || showAll || childIsPending {
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
			emitNode(child, depth+1, childPrefix, lastChild, expanded, showAll, pendingComplete, today, rows)
		}
	}
}
