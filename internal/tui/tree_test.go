package tui

import (
	"testing"

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
	rows := buildVisible(tree, nil, false, nil)
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
	rows := buildVisible(tree, map[int64]bool{}, false, nil)
	if len(rows) != 1 {
		t.Fatalf("expected 1 root row (collapsed), got %d", len(rows))
	}
	if rows[0].node.Task.Id != 1 {
		t.Errorf("expected root id=1")
	}
}

func TestBuildVisible_ExpandedShowsChildren(t *testing.T) {
	tree := makeTree()
	rows := buildVisible(tree, map[int64]bool{1: true}, false, nil)
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
	rows := buildVisible(tree, map[int64]bool{}, false, nil)
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
	rows := buildVisible(tree, map[int64]bool{}, true, nil)
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
	rows := buildVisible(tree, map[int64]bool{}, false, &pendingID)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (pending stays visible), got %d", len(rows))
	}
}

func TestBuildVisible_ExpandableCollapsed(t *testing.T) {
	tree := makeTree()
	rows := buildVisible(tree, map[int64]bool{}, false, nil)
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
	rows := buildVisible(tree, map[int64]bool{1: true}, false, nil)
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
	rows := buildVisible(tree, map[int64]bool{}, false, nil)
	// leaf node (no children) → expandable=false
	if rows[0].expandable {
		t.Errorf("expected expandable=false for leaf node")
	}
}

func TestBuildVisible_DepthValues(t *testing.T) {
	tree := makeTree()
	rows := buildVisible(tree, map[int64]bool{1: true}, false, nil)
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
	rows := buildVisible(tree, map[int64]bool{1: false}, false, nil)
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
	rows := buildVisible(tree, expanded, false, nil)
	if len(rows) != 3 {
		t.Fatalf("after expand: expected 3 rows, got %d", len(rows))
	}

	// Collapse root (H)
	expanded[1] = false
	rows = buildVisible(tree, expanded, false, nil)
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
	rows := buildVisible(tree, map[int64]bool{}, false, nil)
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
	rows := buildVisible(tree, nil, false, nil)
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
	rows := buildVisible(tree, map[int64]bool{}, false, nil)
	if len(rows) != 1 {
		t.Fatalf("expected 1 visible row (root only, collapsed), got %d", len(rows))
	}
	if !rows[0].expandable {
		t.Errorf("expandable should be true when there is at least one visible child")
	}
}
