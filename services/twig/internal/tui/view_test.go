package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	taskv1 "github.com/pboyd/twig/services/twig/gen/task/v1"
	"github.com/pboyd/twig/services/twig/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestRenderList_StrikethroughOnCompletedRow (T-F) asserts that renderList emits
// ANSI strikethrough around the name of a completed row and not on an incomplete row.
func TestRenderList_StrikethroughOnCompletedRow(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{
		{Id: 1, Name: "done task", CompletedAt: now},
		{Id: 2, Name: "todo task"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	// Show completed tasks so the completed row is in m.visible.
	m.showCompleted = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(80)

	if !strings.Contains(out, "\x1b[9m") {
		t.Errorf("renderList: expected ANSI strikethrough open code for completed row; got:\n%q", out)
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Errorf("renderList: expected ANSI reset code for completed row; got:\n%q", out)
	}

	// Verify the incomplete row does NOT contain strikethrough.
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "todo task") && strings.Contains(line, "\x1b[9m") {
			t.Errorf("incomplete row must not contain strikethrough; got:\n%q", line)
		}
	}
}

// TestRenderList_NoStrikethroughWhenUnstyled asserts that renderList produces no
// ANSI strikethrough codes when styled=false (non-TTY mode).
func TestRenderList_NoStrikethroughWhenUnstyled(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{
		{Id: 1, Name: "done task", CompletedAt: now},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, false)
	m.showCompleted = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(80)

	if strings.Contains(out, "\x1b[9m") {
		t.Errorf("renderList with styled=false must not emit strikethrough; got:\n%q", out)
	}
}

// TestViewList_PaneWidths (T004) asserts that each content line of viewList output
// has visual width == m.width (C4.1/C4.2): both panes together fill the terminal width.
func TestViewList_PaneWidths(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "task A"}, {Id: 2, Name: "task B"}}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.width = 80
	m.height = 24
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.viewList()
	lines := strings.Split(out, "\n")
	// The status line is the last non-empty line; pane lines precede it.
	// Check the first line (top border) has exactly m.width visual columns.
	if len(lines) == 0 {
		t.Fatal("viewList returned empty output")
	}
	if w := lipgloss.Width(lines[0]); w != 80 {
		t.Errorf("top border line: visual width %d, want 80; line: %q", w, lines[0])
	}
	// No line should exceed m.width.
	for i, line := range lines {
		if w := lipgloss.Width(line); w > 80 {
			t.Errorf("line %d overflow: visual width %d > 80; line: %q", i, w, line)
		}
	}
}

// TestViewList_BorderRunes (T005) asserts border box-drawing runes appear iff styled==true (C1/C3).
func TestViewList_BorderRunes(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "task A"}}
	tree := cli.BuildTree(tasks)

	// styled=true: top-left rounded border rune must appear.
	ms := ExportNewStyledModel(nil, tree, true)
	ms.width = 80
	ms.height = 24
	ms.visible = buildVisible(ms.tree, ms.expanded, ms.showCompleted, ms.pendingComplete)
	outStyled := ms.viewList()
	if !strings.ContainsAny(outStyled, "╭╰╮╯│─") {
		t.Errorf("styled viewList: expected border runes (╭╰╮╯│─) in output; got:\n%q", outStyled[:min(len(outStyled), 200)])
	}

	// styled=false: no border runes.
	mu := ExportNewStyledModel(nil, tree, false)
	mu.width = 80
	mu.height = 24
	mu.visible = buildVisible(mu.tree, mu.expanded, mu.showCompleted, mu.pendingComplete)
	outUnstyled := mu.viewList()
	for _, r := range []string{"╭", "╰", "╮", "╯", "│", "─"} {
		if strings.Contains(outUnstyled, r) {
			t.Errorf("unstyled viewList: border rune %q must not appear; got:\n%q", r, outUnstyled[:min(len(outUnstyled), 200)])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestRenderList_ChevronIffExpandable (T012) asserts styled-path: chevron iff expandable, checkbox on every row.
func TestRenderList_ChevronIffExpandable(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	parentID := int64(1)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent"},
		{Id: 2, Name: "child", ParentId: &parentID, CompletedAt: now},
		{Id: 3, Name: "leaf"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.showCompleted = true
	m.expanded[1] = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(80)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Row 0: parent is expandable (has child) and expanded → chevron ▾
	if !strings.Contains(lines[0], "▾") {
		t.Errorf("row 0 (expandable+expanded): expected ▾; got: %q", lines[0])
	}
	// Row 1: child is a leaf (no children) → no chevron
	if strings.ContainsAny(lines[1], "▾▸") {
		t.Errorf("row 1 (leaf child): unexpected chevron; got: %q", lines[1])
	}
	// Row 2: leaf root → no chevron
	if strings.ContainsAny(lines[2], "▾▸") {
		t.Errorf("row 2 (leaf): unexpected chevron; got: %q", lines[2])
	}
	// All rows must have a checkbox (☐ or ☑)
	for i, line := range lines {
		if !strings.ContainsAny(line, "☐☑") {
			t.Errorf("row %d: expected checkbox (☐/☑); got: %q", i, line)
		}
	}
	// Completed child (row 1) → ☑; incomplete rows → ☐
	if !strings.Contains(lines[1], "☑") {
		t.Errorf("row 1 (completed): expected ☑; got: %q", lines[1])
	}
	if !strings.Contains(lines[0], "☐") {
		t.Errorf("row 0 (incomplete): expected ☐; got: %q", lines[0])
	}
}

// TestRenderList_NoGlyphsWhenUnstyled (T013) asserts no decorative glyph runes in unstyled path.
func TestRenderList_NoGlyphsWhenUnstyled(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	parentID := int64(1)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent"},
		{Id: 2, Name: "child", ParentId: &parentID, CompletedAt: now},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, false)
	m.showCompleted = true
	m.expanded[1] = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(80)
	for _, r := range []string{"▾", "▸", "☐", "☑"} {
		if strings.Contains(out, r) {
			t.Errorf("unstyled renderList: glyph %q must not appear; got:\n%q", r, out)
		}
	}
}

// TestRenderList_CollapsedExpandableChevron asserts chevron ▸ for expandable+collapsed.
func TestRenderList_CollapsedExpandableChevron(t *testing.T) {
	parentID := int64(1)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent"},
		{Id: 2, Name: "child", ParentId: &parentID},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	// expandable but not expanded
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(80)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.Contains(lines[0], "▸") {
		t.Errorf("expandable+collapsed row: expected ▸; got: %q", lines[0])
	}
}

// TestRenderList_StrikethroughAtMultipleDepths (T013 / FR-003) asserts that
// strikethrough is present on every completed row regardless of nesting depth.
func TestRenderList_StrikethroughAtMultipleDepths(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	p1ID := int64(1)
	p2ID := int64(2)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent done", CompletedAt: now},
		{Id: 2, Name: "child done", ParentId: &p1ID, CompletedAt: now},
		{Id: 3, Name: "grandchild done", ParentId: &p2ID, CompletedAt: now},
		{Id: 4, Name: "todo root"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	// Show completed so all rows are visible.
	m.showCompleted = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
	// Expand all to see nested rows.
	m.expanded[1] = true
	m.expanded[2] = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(120)

	// Every completed task name must be preceded by \x1b[9m somewhere on its line.
	lines := strings.Split(out, "\n")
	completedNames := []string{"parent done", "child done", "grandchild done"}
	for _, name := range completedNames {
		found := false
		for _, line := range lines {
			if strings.Contains(line, name) {
				if strings.Contains(line, "\x1b[9m") {
					found = true
				}
				break
			}
		}
		if !found {
			t.Errorf("renderList: expected strikethrough on completed row %q", name)
		}
	}

	// Incomplete row must NOT have strikethrough.
	for _, line := range lines {
		if strings.Contains(line, "todo root") && strings.Contains(line, "\x1b[9m") {
			t.Errorf("incomplete row 'todo root' must not have strikethrough; got:\n%q", line)
		}
	}
}

// TestRenderList_CursorHighlight (T018) asserts cursor row uses background tint and bold (C6).
func TestRenderList_CursorHighlight(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{
		{Id: 1, Name: "completed task", CompletedAt: now},
		{Id: 2, Name: "other task"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.showCompleted = true
	m.cursor = 0
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(80)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Cursor row: completed name should still have strikethrough (C6.2).
	if !strings.Contains(lines[0], "\x1b[9m") {
		t.Errorf("cursor row: completed name must still have strikethrough; got: %q", lines[0])
	}
	// Non-cursor rows must not have strikethrough on incomplete tasks.
	if strings.Contains(lines[1], "\x1b[9m") {
		t.Errorf("non-cursor incomplete row must not have strikethrough; got: %q", lines[1])
	}
}

// ── US4: no redundant pane titles ──────────────────────────────────────────

// TestViewList_NoRedundantTasksTitle checks that viewList (styled) does not
// render a "Tasks" left-pane title (T015).
func TestViewList_NoRedundantTasksTitle(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "task A"}}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.width = 80
	m.height = 24
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.viewList()

	// The left pane title "Tasks" must not appear in the border.
	// The right pane "Details" is expected.
	if strings.Contains(out, "─ Tasks ─") {
		t.Errorf("viewList: left pane should not have 'Tasks' title in border; found it in:\n%q", out[:min(len(out), 300)])
	}
	if !strings.Contains(out, "Details") {
		t.Errorf("viewList: right pane should still have 'Details' title; got:\n%q", out[:min(len(out), 300)])
	}
}

// TestRenderStatus_FooterWidth (T019) asserts the status line is available (non-empty) and
// error messages use the error color style (C7).
func TestRenderStatus_FooterAndError(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "task"}}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	// No error: status should return the help view (non-empty).
	status := m.renderStatus()
	if status == "" {
		t.Errorf("renderStatus: expected non-empty help text; got empty string")
	}

	// With error: status should contain error text.
	m.err = fmt.Errorf("something broke")
	errStatus := m.renderStatus()
	if !strings.Contains(errStatus, "something broke") {
		t.Errorf("renderStatus with error: expected error text; got: %q", errStatus)
	}
}
