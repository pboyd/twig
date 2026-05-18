package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- parseDue unit tests (T007 / US1) ---

func TestParseDueRFC3339(t *testing.T) {
	ts, err := parseDue("2026-06-01T17:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 6, 1, 17, 0, 0, 0, time.UTC)
	if !ts.AsTime().Equal(want) {
		t.Errorf("got %v, want %v", ts.AsTime(), want)
	}
}

func TestParseDueBareDate(t *testing.T) {
	ts, err := parseDue("2026-06-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if !ts.AsTime().Equal(want) {
		t.Errorf("got %v, want %v", ts.AsTime(), want)
	}
}

func TestParseDueMalformed(t *testing.T) {
	cases := []string{"not-a-date", "2026/06/01", "June 1, 2026", ""}
	for _, v := range cases {
		_, err := parseDue(v)
		if err == nil {
			t.Errorf("expected error for %q, got nil", v)
		}
	}
}

func TestFormatDueNil(t *testing.T) {
	if got := formatDue(nil); got != "" {
		t.Errorf("expected empty string for nil, got %q", got)
	}
}

func TestFormatDueUTC(t *testing.T) {
	ts := timestamppb.New(time.Date(2026, 6, 1, 17, 0, 0, 0, time.UTC))
	got := formatDue(ts)
	if got != "2026-06-01T17:00:00Z" {
		t.Errorf("got %q", got)
	}
}

// --- Tree rendering unit tests (T011 / US2) ---

func ptr(i int64) *int64 { return &i }

func TestBuildTreeRoots(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
	}
	roots := buildTree(tasks)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	if roots[0].task.Id != 1 || roots[1].task.Id != 2 {
		t.Errorf("roots not in ascending order")
	}
}

func TestBuildTreeChildren(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child", ParentId: ptr(1)},
		{Id: 3, Name: "grandchild", ParentId: ptr(2)},
	}
	roots := buildTree(tasks)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(roots[0].children))
	}
	if len(roots[0].children[0].children) != 1 {
		t.Fatalf("expected 1 grandchild")
	}
}

func TestRenderTreeGlyphs(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "first", ParentId: ptr(1)},
		{Id: 3, Name: "second", ParentId: ptr(1)},
	}
	roots := buildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots)
	out := buf.String()

	if !strings.Contains(out, "├──") {
		t.Errorf("expected ├── connector for non-last child: %s", out)
	}
	if !strings.Contains(out, "└──") {
		t.Errorf("expected └── connector for last child: %s", out)
	}
}

func TestRenderTreeNoDue(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "task"}}
	roots := buildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots)
	out := buf.String()
	if strings.Contains(out, "(due ") {
		t.Errorf("unexpected due date in output: %s", out)
	}
}

func TestRenderTreeWithDue(t *testing.T) {
	ts := timestamppb.New(time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{{Id: 1, Name: "task", Due: ts}}
	roots := buildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots)
	out := buf.String()
	if !strings.Contains(out, "2026-05-25T00:00:00Z") {
		t.Errorf("expected RFC3339 date in output: %s", out)
	}
}

func TestSiblingOrder(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 3, Name: "c"},
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
	}
	roots := buildTree(tasks)
	if roots[0].task.Id != 1 || roots[1].task.Id != 2 || roots[2].task.Id != 3 {
		t.Errorf("roots not sorted ascending by id")
	}
}

func TestNestedDepth(t *testing.T) {
	// Build a tree where root has two children, first child has a grandchild.
	// The │ glyph must appear on the grandchild line because root still has a
	// second child after the first.
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "childA", ParentId: ptr(1)},
		{Id: 3, Name: "grandchild", ParentId: ptr(2)},
		{Id: 4, Name: "childB", ParentId: ptr(1)},
	}
	roots := buildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots)
	out := buf.String()
	// All tasks should appear
	for _, name := range []string{"root", "childA", "grandchild", "childB"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected %q in output: %s", name, out)
		}
	}
	// │ should appear for the grandchild line (root still has childB pending)
	if !strings.Contains(out, "│") {
		t.Errorf("expected │ glyph for depth: %s", out)
	}
}
