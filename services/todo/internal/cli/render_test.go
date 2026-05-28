package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Strike helper tests (T-A) ---

func TestStrike_TTYOn(t *testing.T) {
	got := Strike("hello", true)
	if got != "\x1b[9m"+"hello"+"\x1b[0m" {
		t.Errorf("Strike with isTTY=true: got %q", got)
	}
}

func TestStrike_TTYOff(t *testing.T) {
	got := Strike("hello", false)
	if got != "hello" {
		t.Errorf("Strike with isTTY=false: got %q, want %q", got, "hello")
	}
}

// --- parseDue unit tests (T007 / US1) ---

func TestParseDueRFC3339(t *testing.T) {
	ts, err := ParseDue("2026-06-01T17:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 6, 1, 17, 0, 0, 0, time.UTC)
	if !ts.AsTime().Equal(want) {
		t.Errorf("got %v, want %v", ts.AsTime(), want)
	}
}

func TestParseDueBareDate(t *testing.T) {
	ts, err := ParseDue("2026-06-01")
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
		_, err := ParseDue(v)
		if err == nil {
			t.Errorf("expected error for %q, got nil", v)
		}
	}
}

func TestFormatDueNil(t *testing.T) {
	if got := FormatDue(nil); got != "" {
		t.Errorf("expected empty string for nil, got %q", got)
	}
}

func TestFormatDueUTC(t *testing.T) {
	ts := timestamppb.New(time.Date(2026, 6, 1, 17, 0, 0, 0, time.UTC))
	got := FormatDue(ts)
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
	roots := BuildTree(tasks)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	if roots[0].Task.Id != 1 || roots[1].Task.Id != 2 {
		t.Errorf("roots not in ascending order")
	}
}

func TestBuildTreeChildren(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child", ParentId: ptr(1)},
		{Id: 3, Name: "grandchild", ParentId: ptr(2)},
	}
	roots := BuildTree(tasks)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(roots[0].Children))
	}
	if len(roots[0].Children[0].Children) != 1 {
		t.Fatalf("expected 1 grandchild")
	}
}

func TestRenderTreeGlyphs(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "first", ParentId: ptr(1)},
		{Id: 3, Name: "second", ParentId: ptr(1)},
	}
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, false)
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
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, false)
	out := buf.String()
	if strings.Contains(out, "(due ") {
		t.Errorf("unexpected due date in output: %s", out)
	}
}

func TestRenderTreeWithDue(t *testing.T) {
	ts := timestamppb.New(time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{{Id: 1, Name: "task", Due: ts}}
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, false)
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
	roots := BuildTree(tasks)
	if roots[0].Task.Id != 1 || roots[1].Task.Id != 2 || roots[2].Task.Id != 3 {
		t.Errorf("roots not sorted ascending by id")
	}
}

func TestCompletedAt(t *testing.T) {
	now := time.Date(2026, 5, 20, 18, 42, 11, 0, time.UTC)
	ts := timestamppb.New(now)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "incomplete"},
		{Id: 2, Name: "complete", CompletedAt: ts},
	}
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, false)
	out := buf.String()

	if strings.Contains(out, "[ ] ") || strings.Contains(out, "[x] ") {
		t.Errorf("checkbox prefixes must not appear in output: %s", out)
	}
	if !strings.Contains(out, "(completed 2026-05-20T18:42:11Z)") {
		t.Errorf("expected completed timestamp in output: %s", out)
	}
	if !strings.Contains(out, "[1] incomplete") {
		t.Errorf("expected [1] incomplete in output: %s", out)
	}
	if !strings.Contains(out, "[2] complete") {
		t.Errorf("expected [2] complete in output: %s", out)
	}
}

func TestPruneIncomplete(t *testing.T) {
	t.Run("complete leaf is dropped", func(t *testing.T) {
		now := timestamppb.New(time.Now())
		tasks := []*taskv1.Task{{Id: 1, Name: "done", CompletedAt: now}}
		roots := BuildTree(tasks)
		pruned := pruneIncomplete(roots)
		if len(pruned) != 0 {
			t.Errorf("expected 0 roots after pruning complete leaf, got %d", len(pruned))
		}
	})

	t.Run("incomplete leaf is kept", func(t *testing.T) {
		tasks := []*taskv1.Task{{Id: 1, Name: "todo"}}
		roots := BuildTree(tasks)
		pruned := pruneIncomplete(roots)
		if len(pruned) != 1 {
			t.Errorf("expected 1 root after pruning, got %d", len(pruned))
		}
	})

	t.Run("incomplete parent with mixed children keeps itself and incomplete children", func(t *testing.T) {
		now := timestamppb.New(time.Now())
		tasks := []*taskv1.Task{
			{Id: 1, Name: "parent"},
			{Id: 2, Name: "done child", ParentId: ptr(1), CompletedAt: now},
			{Id: 3, Name: "todo child", ParentId: ptr(1)},
		}
		roots := BuildTree(tasks)
		pruned := pruneIncomplete(roots)
		if len(pruned) != 1 {
			t.Fatalf("expected 1 root, got %d", len(pruned))
		}
		if len(pruned[0].Children) != 1 {
			t.Errorf("expected 1 child (incomplete only), got %d", len(pruned[0].Children))
		}
		if pruned[0].Children[0].Task.Id != 3 {
			t.Errorf("expected incomplete child id=3, got id=%d", pruned[0].Children[0].Task.Id)
		}
	})

	t.Run("fully complete subtree disappears", func(t *testing.T) {
		now := timestamppb.New(time.Now())
		tasks := []*taskv1.Task{
			{Id: 1, Name: "parent", CompletedAt: now},
			{Id: 2, Name: "child", ParentId: ptr(1), CompletedAt: now},
		}
		roots := BuildTree(tasks)
		pruned := pruneIncomplete(roots)
		if len(pruned) != 0 {
			t.Errorf("expected 0 roots for fully complete subtree, got %d", len(pruned))
		}
	})

	t.Run("id ordering preserved among kept siblings", func(t *testing.T) {
		now := timestamppb.New(time.Now())
		tasks := []*taskv1.Task{
			{Id: 1, Name: "parent"},
			{Id: 2, Name: "done", ParentId: ptr(1), CompletedAt: now},
			{Id: 3, Name: "todo-a", ParentId: ptr(1)},
			{Id: 5, Name: "todo-b", ParentId: ptr(1)},
		}
		roots := BuildTree(tasks)
		pruned := pruneIncomplete(roots)
		if len(pruned[0].Children) != 2 {
			t.Fatalf("expected 2 kept children, got %d", len(pruned[0].Children))
		}
		if pruned[0].Children[0].Task.Id != 3 || pruned[0].Children[1].Task.Id != 5 {
			t.Errorf("children out of order: %d, %d", pruned[0].Children[0].Task.Id, pruned[0].Children[1].Task.Id)
		}
	})
}

func TestFilterCompleted(t *testing.T) {
	t.Run("complete leaf is kept", func(t *testing.T) {
		now := timestamppb.New(time.Now())
		tasks := []*taskv1.Task{{Id: 1, Name: "done", CompletedAt: now}}
		roots := BuildTree(tasks)
		filtered := filterCompleted(roots)
		if len(filtered) != 1 {
			t.Errorf("expected 1 root, got %d", len(filtered))
		}
	})

	t.Run("incomplete leaf is dropped", func(t *testing.T) {
		tasks := []*taskv1.Task{{Id: 1, Name: "todo"}}
		roots := BuildTree(tasks)
		filtered := filterCompleted(roots)
		if len(filtered) != 0 {
			t.Errorf("expected 0 roots, got %d", len(filtered))
		}
	})

	t.Run("complete children of incomplete parent are promoted to root", func(t *testing.T) {
		now := timestamppb.New(time.Now())
		tasks := []*taskv1.Task{
			{Id: 1, Name: "parent"},
			{Id: 2, Name: "done child", ParentId: ptr(1), CompletedAt: now},
			{Id: 3, Name: "todo child", ParentId: ptr(1)},
		}
		roots := BuildTree(tasks)
		filtered := filterCompleted(roots)
		if len(filtered) != 1 {
			t.Fatalf("expected 1 promoted root (done child), got %d", len(filtered))
		}
		if filtered[0].Task.Id != 2 {
			t.Errorf("expected promoted task id=2, got id=%d", filtered[0].Task.Id)
		}
	})

	t.Run("id ordering preserved", func(t *testing.T) {
		now := timestamppb.New(time.Now())
		tasks := []*taskv1.Task{
			{Id: 1, Name: "todo"},
			{Id: 2, Name: "done-a", CompletedAt: now},
			{Id: 3, Name: "done-b", CompletedAt: now},
		}
		roots := BuildTree(tasks)
		filtered := filterCompleted(roots)
		if len(filtered) != 2 {
			t.Fatalf("expected 2 roots, got %d", len(filtered))
		}
		if filtered[0].Task.Id != 2 || filtered[1].Task.Id != 3 {
			t.Errorf("order wrong: %d, %d", filtered[0].Task.Id, filtered[1].Task.Id)
		}
	})
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
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, false)
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

// --- T009: styled rendering tests (US3) ---

func TestRenderStyledCompletedTask(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	ts := timestamppb.New(now)
	tasks := []*taskv1.Task{{Id: 7, Name: "done task", CompletedAt: ts}}
	roots := BuildTree(tasks)

	t.Run("styled=true wraps post-id content", func(t *testing.T) {
		var buf bytes.Buffer
		renderRoots(&buf, roots, true)
		out := buf.String()
		if !strings.Contains(out, "\x1b[2;9m") {
			t.Errorf("expected dim+strikethrough open code in styled output: %q", out)
		}
		if !strings.Contains(out, "\x1b[0m") {
			t.Errorf("expected reset code in styled output: %q", out)
		}
		if strings.Contains(out, "[ ] ") || strings.Contains(out, "[x] ") {
			t.Errorf("checkbox markers must not appear: %q", out)
		}
		// Leading [id] prefix must NOT be inside the escape sequence
		idx := strings.Index(out, "[7]")
		escIdx := strings.Index(out, "\x1b[2;9m")
		if escIdx < idx {
			t.Errorf("[id] prefix appears after ANSI open code — it must not be styled: %q", out)
		}
	})

	t.Run("styled=false produces no ANSI codes", func(t *testing.T) {
		var buf bytes.Buffer
		renderRoots(&buf, roots, false)
		out := buf.String()
		if strings.ContainsAny(out, "\x1b") {
			t.Errorf("no escape codes expected in unstyled output: %q", out)
		}
		if strings.Contains(out, "[ ] ") || strings.Contains(out, "[x] ") {
			t.Errorf("checkbox markers must not appear: %q", out)
		}
	})
}

func TestRenderStyledIncompleteTask(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 3, Name: "pending"}}
	roots := BuildTree(tasks)

	for _, styled := range []bool{true, false} {
		var buf bytes.Buffer
		renderRoots(&buf, roots, styled)
		out := buf.String()
		if strings.ContainsAny(out, "\x1b") {
			t.Errorf("styled=%v: incomplete task must not have ANSI codes: %q", styled, out)
		}
	}
}

// --- T011: estimate rendering tests (US4) ---

func TestRenderEstimateNonZero(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "task", Estimate: 3}}
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, false)
	out := buf.String()
	if !strings.Contains(out, " (3)") {
		t.Errorf("expected \" (3)\" in output: %s", out)
	}
}

func TestRenderEstimateZero(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "task", Estimate: 0}}
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, false)
	out := buf.String()
	// Should not contain any parenthesized number
	if strings.Contains(out, " (0)") {
		t.Errorf("zero estimate must not appear in output: %s", out)
	}
}

func TestRenderEstimateOrderBeforeDue(t *testing.T) {
	due := timestamppb.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{{Id: 1, Name: "task", Estimate: 2, Due: due}}
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, false)
	out := buf.String()
	estIdx := strings.Index(out, " (2)")
	dueIdx := strings.Index(out, " (due ")
	if estIdx < 0 || dueIdx < 0 {
		t.Fatalf("missing estimate or due in output: %s", out)
	}
	if estIdx > dueIdx {
		t.Errorf("estimate must appear before due date; got: %s", out)
	}
}

func TestRenderEstimateStyledCompleted(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{{Id: 5, Name: "done", Estimate: 2, CompletedAt: now}}
	roots := BuildTree(tasks)
	var buf bytes.Buffer
	renderRoots(&buf, roots, true)
	out := buf.String()
	// (2) must be inside the styled region (between \x1b[2;9m and \x1b[0m)
	open := strings.Index(out, "\x1b[2;9m")
	reset := strings.Index(out, "\x1b[0m")
	est := strings.Index(out, "(2)")
	if open < 0 || reset < 0 || est < 0 {
		t.Fatalf("missing styled codes or estimate in output: %q", out)
	}
	if !(open < est && est < reset) {
		t.Errorf("estimate (2) must be inside styled region; got: %q", out)
	}
}
