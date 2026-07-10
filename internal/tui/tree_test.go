package tui

import (
	"testing"
	"time"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ptr64(i int64) *int64 { return &i }

// makeTree builds a simple tree: root(1) → child(2), child(3)
func makeTree() []*cli.TreeNode {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child-a", ParentId: ptr64(1)},
		{Id: 3, Name: "child-b", ParentId: ptr64(1)},
	}
	return cli.BuildTree(tasks)
}

func TestBuildVisible_RootOnly(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "root"}}
	tree := cli.BuildTree(tasks)
	rows := buildVisible(tree, nil, false, nil, time.Now().Local())
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].node.Task.Id != 1 {
		t.Errorf("expected id=1, got %d", rows[0].node.Task.Id)
	}
}

func TestBuildVisible_CollapsedSubtreeSkipped(t *testing.T) {
	tree := makeTree()
	// expanded is empty — children should not appear
	rows := buildVisible(tree, map[int64]bool{}, false, nil, time.Now().Local())
	if len(rows) != 1 {
		t.Fatalf("expected 1 root row (collapsed), got %d", len(rows))
	}
	if rows[0].node.Task.Id != 1 {
		t.Errorf("expected root id=1")
	}
}

func TestBuildVisible_ExpandedShowsChildren(t *testing.T) {
	tree := makeTree()
	rows := buildVisible(tree, map[int64]bool{1: true}, false, nil, time.Now().Local())
	// root + 2 children
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	ids := []int64{rows[0].node.Task.Id, rows[1].node.Task.Id, rows[2].node.Task.Id}
	if ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("unexpected row ids: %v", ids)
	}
}

func TestBuildVisible_CompletedFilteredByDefault(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "incomplete"},
		{Id: 2, Name: "done", CompletedAt: now},
	}
	tree := cli.BuildTree(tasks)
	rows := buildVisible(tree, map[int64]bool{}, false, nil, time.Now().Local())
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (completed hidden), got %d", len(rows))
	}
	if rows[0].node.Task.Id != 1 {
		t.Errorf("expected incomplete task, got id=%d", rows[0].node.Task.Id)
	}
}

func TestBuildVisible_ShowCompletedIncludesDone(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "incomplete"},
		{Id: 2, Name: "done", CompletedAt: now},
	}
	tree := cli.BuildTree(tasks)
	rows := buildVisible(tree, map[int64]bool{}, true, nil, time.Now().Local())
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}

func TestBuildVisible_PendingCompleteStaysVisible(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "incomplete"},
		{Id: 2, Name: "done", CompletedAt: now},
	}
	tree := cli.BuildTree(tasks)
	pendingID := int64(2)
	rows := buildVisible(tree, map[int64]bool{}, false, &pendingID, time.Now().Local())
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (pending stays visible), got %d", len(rows))
	}
}

func TestBuildVisible_ExpandableCollapsed(t *testing.T) {
	tree := makeTree()
	rows := buildVisible(tree, map[int64]bool{}, false, nil, time.Now().Local())
	// root is collapsed and has visible children → expandable=true, expanded=false
	if !rows[0].expandable {
		t.Errorf("expected expandable=true for collapsed node with children")
	}
	if rows[0].expanded {
		t.Errorf("expected expanded=false for collapsed node")
	}
}

func TestBuildVisible_ExpandedNode(t *testing.T) {
	tree := makeTree()
	rows := buildVisible(tree, map[int64]bool{1: true}, false, nil, time.Now().Local())
	// root is expanded → expandable=true, expanded=true
	if !rows[0].expandable {
		t.Errorf("expected expandable=true for node with visible children")
	}
	if !rows[0].expanded {
		t.Errorf("expected expanded=true for expanded node")
	}
}

func TestBuildVisible_LeafNode(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "leaf"}}
	tree := cli.BuildTree(tasks)
	rows := buildVisible(tree, map[int64]bool{}, false, nil, time.Now().Local())
	// leaf node (no children) → expandable=false
	if rows[0].expandable {
		t.Errorf("expected expandable=false for leaf node")
	}
}

func TestBuildVisible_DepthValues(t *testing.T) {
	tree := makeTree()
	rows := buildVisible(tree, map[int64]bool{1: true}, false, nil, time.Now().Local())
	if rows[0].depth != 0 {
		t.Errorf("root depth: want 0, got %d", rows[0].depth)
	}
	if rows[1].depth != 1 {
		t.Errorf("child depth: want 1, got %d", rows[1].depth)
	}
}

// TestBuildVisible_CollapseParentKeepsParentVisible verifies that collapsing
// a parent does not remove the parent row itself — only its subtree.
func TestBuildVisible_CollapseParentKeepsParentVisible(t *testing.T) {
	tree := makeTree()
	// Start expanded, then simulate collapse of root
	rows := buildVisible(tree, map[int64]bool{1: false}, false, nil, time.Now().Local())
	// Only root should appear (children hidden)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row after collapse, got %d", len(rows))
	}
	if rows[0].node.Task.Id != 1 {
		t.Errorf("expected root row, got id=%d", rows[0].node.Task.Id)
	}
}

// TestExpandCollapse tests that H/L correctly set expanded state.
func TestExpandCollapse_SetExpanded(t *testing.T) {
	tree := makeTree()
	expanded := map[int64]bool{}

	// Expand root (L)
	expanded[1] = true
	rows := buildVisible(tree, expanded, false, nil, time.Now().Local())
	if len(rows) != 3 {
		t.Fatalf("after expand: expected 3 rows, got %d", len(rows))
	}

	// Collapse root (H)
	expanded[1] = false
	rows = buildVisible(tree, expanded, false, nil, time.Now().Local())
	if len(rows) != 1 {
		t.Fatalf("after collapse: expected 1 row, got %d", len(rows))
	}
	// Parent (root) still visible
	if rows[0].node.Task.Id != 1 {
		t.Errorf("parent should remain visible after collapse, got id=%d", rows[0].node.Task.Id)
	}
}

func TestMarker_AllChildrenCompletedNotExpandable(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "done", CompletedAt: now, ParentId: ptr64(1)},
	}
	tree := cli.BuildTree(tasks)

	// showCompleted=false: child is hidden, so root should not be expandable.
	rows := buildVisible(tree, map[int64]bool{}, false, nil, time.Now().Local())
	if len(rows) != 1 {
		t.Fatalf("expected 1 visible row (completed child filtered), got %d", len(rows))
	}
	if rows[0].expandable {
		t.Errorf("expandable should be false when all children are filtered out")
	}
}

// TestBuildVisible_UnseededTaskDefaultsCollapsed verifies that tasks with no
// stored expansion entry (nil expanded map) render collapsed — new/unknown tasks
// default correctly without extra logic (FR-004).
func TestBuildVisible_UnseededTaskDefaultsCollapsed(t *testing.T) {
	tree := makeTree()
	// nil expanded map: root has children but no entry → renders collapsed.
	rows := buildVisible(tree, nil, false, nil, time.Now().Local())
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (collapsed root), got %d", len(rows))
	}
	if rows[0].node.Task.Id != 1 {
		t.Errorf("expected root id=1, got %d", rows[0].node.Task.Id)
	}
}

func TestMarker_SomeChildrenCompletedIsExpandable(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "done", CompletedAt: now, ParentId: ptr64(1)},
		{Id: 3, Name: "incomplete", ParentId: ptr64(1)},
	}
	tree := cli.BuildTree(tasks)

	// showCompleted=false: one incomplete child still visible → root should be expandable.
	rows := buildVisible(tree, map[int64]bool{}, false, nil, time.Now().Local())
	if len(rows) != 1 {
		t.Fatalf("expected 1 visible row (root only, collapsed), got %d", len(rows))
	}
	if !rows[0].expandable {
		t.Errorf("expandable should be true when there is at least one visible child")
	}
}

// --- visibleSiblings tests ---

func TestVisibleSiblings_Roots(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "A", Position: 0},
		{Id: 2, Name: "B", Position: 1},
		{Id: 3, Name: "C", Position: 2},
	}
	tree := cli.BuildTree(tasks)
	expanded := map[int64]bool{}

	prev, next := visibleSiblings(tree, expanded, false, 2)
	if prev != 1 {
		t.Errorf("prev of B = %d, want 1 (A)", prev)
	}
	if next != 3 {
		t.Errorf("next of B = %d, want 3 (C)", next)
	}

	// First element: no prev.
	prev, next = visibleSiblings(tree, expanded, false, 1)
	if prev != 0 {
		t.Errorf("prev of A = %d, want 0 (none)", prev)
	}
	if next != 2 {
		t.Errorf("next of A = %d, want 2 (B)", next)
	}

	// Last element: no next.
	prev, next = visibleSiblings(tree, expanded, false, 3)
	if prev != 2 {
		t.Errorf("prev of C = %d, want 2 (B)", prev)
	}
	if next != 0 {
		t.Errorf("next of C = %d, want 0 (none)", next)
	}
}

func TestVisibleSiblings_Children(t *testing.T) {
	parentID := int64(1)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root", Position: 0},
		{Id: 2, Name: "child-first", Position: 0, ParentId: &parentID},
		{Id: 3, Name: "child-second", Position: 1, ParentId: &parentID},
	}
	tree := cli.BuildTree(tasks)
	expanded := map[int64]bool{1: true}

	// Siblings among children only.
	prev, next := visibleSiblings(tree, expanded, false, 2)
	if prev != 0 {
		t.Errorf("prev of child-first = %d, want 0", prev)
	}
	if next != 3 {
		t.Errorf("next of child-first = %d, want 3", next)
	}
}

func TestVisibleSiblings_HiddenCompleted(t *testing.T) {
	// When showCompleted=false, completed siblings are hidden and skipped.
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "A", Position: 0},
		{Id: 2, Name: "B (completed)", Position: 1, CompletedAt: now},
		{Id: 3, Name: "C", Position: 2},
	}
	tree := cli.BuildTree(tasks)
	expanded := map[int64]bool{}

	// B is hidden; A's next visible sibling should be C.
	prev, next := visibleSiblings(tree, expanded, false, 1)
	if next != 3 {
		t.Errorf("next of A (completed B hidden) = %d, want 3 (C)", next)
	}
	_ = prev

	// C's prev visible sibling should be A (B is hidden).
	prev, _ = visibleSiblings(tree, expanded, false, 3)
	if prev != 1 {
		t.Errorf("prev of C (completed B hidden) = %d, want 1 (A)", prev)
	}
}

// --- snooze tests ---

func TestTaskIsSnoozed_NilSnoozeUntil(t *testing.T) {
	today := time.Date(2026, 6, 6, 12, 0, 0, 0, time.Local)
	task := &taskv1.Task{Id: 1, Name: "task"}
	if taskIsSnoozed(task, today) {
		t.Error("expected not snoozed when snooze_until is nil")
	}
}

func TestTaskIsSnoozed_FutureDay(t *testing.T) {
	today := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	tomorrow := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	task := &taskv1.Task{Id: 1, Name: "task", SnoozeUntil: timestamppb.New(tomorrow)}
	if !taskIsSnoozed(task, today) {
		t.Error("expected snoozed when snooze_until is tomorrow")
	}
}

func TestTaskIsSnoozed_TodayIsNotSnoozed(t *testing.T) {
	today := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	todayUTC := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC)
	task := &taskv1.Task{Id: 1, Name: "task", SnoozeUntil: timestamppb.New(todayUTC)}
	if taskIsSnoozed(task, today) {
		t.Error("expected not snoozed when snooze_until equals today")
	}
}

func TestTaskIsSnoozed_PastDayIsNotSnoozed(t *testing.T) {
	today := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	yesterday := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	task := &taskv1.Task{Id: 1, Name: "task", SnoozeUntil: timestamppb.New(yesterday)}
	if taskIsSnoozed(task, today) {
		t.Error("expected not snoozed when snooze_until is in the past")
	}
}

func TestBuildVisible_FutureSnoozedHiddenByDefault(t *testing.T) {
	today := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	tomorrow := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "active"},
		{Id: 2, Name: "snoozed", SnoozeUntil: timestamppb.New(tomorrow)},
	}
	tree := cli.BuildTree(tasks)
	rows := buildVisible(tree, map[int64]bool{}, false, nil, today)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (snoozed hidden), got %d", len(rows))
	}
	if rows[0].node.Task.Id != 1 {
		t.Errorf("expected active task, got id=%d", rows[0].node.Task.Id)
	}
}

func TestBuildVisible_FutureSnoozedShownWhenShowAll(t *testing.T) {
	today := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	tomorrow := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "active"},
		{Id: 2, Name: "snoozed", SnoozeUntil: timestamppb.New(tomorrow)},
	}
	tree := cli.BuildTree(tasks)
	rows := buildVisible(tree, map[int64]bool{}, true, nil, today)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (show-all), got %d", len(rows))
	}
}

func TestBuildVisible_TodaySnoozedIsVisible(t *testing.T) {
	today := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	todayUTC := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "wakes-today", SnoozeUntil: timestamppb.New(todayUTC)},
	}
	tree := cli.BuildTree(tasks)
	rows := buildVisible(tree, map[int64]bool{}, false, nil, today)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (snoozed today is visible), got %d", len(rows))
	}
}

func TestBuildVisible_SnoozedParentHidesSubtree(t *testing.T) {
	today := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	tomorrow := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	// Parent snoozed; child not snoozed. Child should be hidden along with parent.
	tasks := []*taskv1.Task{
		{Id: 1, Name: "snoozed-parent", SnoozeUntil: timestamppb.New(tomorrow)},
		{Id: 2, Name: "child", ParentId: ptr64(1)},
	}
	tree := cli.BuildTree(tasks)
	rows := buildVisible(tree, map[int64]bool{1: true}, false, nil, today)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows (parent + child hidden), got %d", len(rows))
	}
}

func TestBuildVisibleFiltered_OnlyTrueAncestorsIncluded(t *testing.T) {
	// Two sibling subtrees: A(1) → A-child(2); B(3) → B-child(4).
	// A match deep in subtree B must not pull in subtree A's rows.
	tasks := []*taskv1.Task{
		{Id: 1, Name: "subtree A"},
		{Id: 2, Name: "A-child", ParentId: ptr64(1)},
		{Id: 3, Name: "subtree B"},
		{Id: 4, Name: "B-child match", ParentId: ptr64(3)},
	}
	tree := cli.BuildTree(tasks)
	filteredIDs := map[int64]bool{4: true}

	rows := buildVisibleFiltered(tree, map[int64]bool{}, false, nil, time.Now().Local(), filteredIDs)

	gotIDs := make(map[int64]bool, len(rows))
	for _, r := range rows {
		gotIDs[r.node.Task.Id] = true
	}

	// Only the match and its true ancestor (3) should appear.
	if len(gotIDs) != 2 || !gotIDs[3] || !gotIDs[4] {
		t.Fatalf("rows = %v, want {3, 4} (match + true ancestor only, not subtree A)", gotIDs)
	}
	if gotIDs[1] || gotIDs[2] {
		t.Errorf("rows unexpectedly include unrelated subtree A: %v", gotIDs)
	}
}

func TestBuildVisibleFiltered_IgnoresCollapsedState(t *testing.T) {
	// Parent(1) → Child(2) → Grandchild-match(3). Parent is collapsed.
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent"},
		{Id: 2, Name: "child", ParentId: ptr64(1)},
		{Id: 3, Name: "grandchild match", ParentId: ptr64(2)},
	}
	tree := cli.BuildTree(tasks)
	filteredIDs := map[int64]bool{3: true}

	// expanded is empty — everything is collapsed.
	rows := buildVisibleFiltered(tree, map[int64]bool{}, false, nil, time.Now().Local(), filteredIDs)

	gotIDs := make(map[int64]bool, len(rows))
	for _, r := range rows {
		gotIDs[r.node.Task.Id] = true
	}

	// The match and both its ancestors must appear despite being collapsed.
	if len(gotIDs) != 3 || !gotIDs[1] || !gotIDs[2] || !gotIDs[3] {
		t.Fatalf("rows = %v, want {1, 2, 3} (match visible despite collapsed ancestors)", gotIDs)
	}
}
