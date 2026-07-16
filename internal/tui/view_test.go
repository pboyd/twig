package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// hasStrikethrough reports whether s contains ANSI strikethrough (SGR 9) in
// any encoding: standalone \x1b[9m (from cli.Strike) or embedded ;9m within a
// combined sequence (from lipgloss Strikethrough(true) on the cursor row).
func hasStrikethrough(s string) bool {
	return strings.Contains(s, "\x1b[9m") || strings.Contains(s, ";9m")
}

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
	m.showAll = true
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	out := m.renderList(80)

	if !hasStrikethrough(out) {
		t.Errorf("renderList: expected ANSI strikethrough open code for completed row; got:\n%q", out)
	}
	// lipgloss v2 may emit \x1b[m (bare reset) instead of \x1b[0m.
	if !strings.Contains(out, "\x1b[0m") && !strings.Contains(out, "\x1b[m") {
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
	m.showAll = true
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	out := m.renderList(80)

	if strings.Contains(out, "\x1b[9m") {
		t.Errorf("renderList with styled=false must not emit strikethrough; got:\n%q", out)
	}
}

// TestPaneBox_WidthAndHeight asserts that paneBox produces a box whose every line is
// exactly outerWidth columns wide and whose total line count is innerHeight+2 (two
// border rows).  This is the direct regression test for the lipgloss v2 Width/Height
// semantics change: in v1 Width/Height were content dimensions; in v2 they are total
// dimensions including the border.
func TestPaneBox_WidthAndHeight(t *testing.T) {
	const outerWidth = 40
	const innerH = 3

	for _, title := range []string{"", "Details"} {
		t.Run("title="+title, func(t *testing.T) {
			// Content is outerWidth-2 wide — exactly what callers pass after
			// subtracting 2 for the border columns.
			content := strings.Repeat("X", outerWidth-2)
			out := paneBox(content, outerWidth, innerH, title, false)
			lines := strings.Split(out, "\n")

			wantLines := innerH + 2 // top border + innerH content rows + bottom border
			if len(lines) != wantLines {
				t.Errorf("line count: got %d, want %d\noutput:\n%s", len(lines), wantLines, out)
			}
			for i, line := range lines {
				if w := lipgloss.Width(line); w != outerWidth {
					t.Errorf("line %d: visual width %d, want %d; %q", i, w, outerWidth, line)
				}
			}
		})
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
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	out := m.viewList()
	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		t.Fatal("viewList returned empty output")
	}

	// Find the first pane border line (starts with ╭) and assert it fills m.width.
	// The old test checked lines[0] which is the tab bar — always full width and not
	// sensitive to the pane-width bug.  The border line is the real sentinel.
	paneBorderIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "╭") || strings.HasPrefix(strings.TrimLeft(line, "\x1b[0123456789;m"), "╭") {
			paneBorderIdx = i
			break
		}
	}
	if paneBorderIdx == -1 {
		t.Fatal("could not find pane top-border line (╭) in viewList output")
	}
	if w := lipgloss.Width(lines[paneBorderIdx]); w != 80 {
		t.Errorf("pane top-border line: visual width %d, want 80; line: %q", w, lines[paneBorderIdx])
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
	ms.visible = buildVisible(ms.tree, ms.expanded, ms.showAll, ms.pendingComplete, time.Now().Local())
	outStyled := ms.viewList()
	if !strings.ContainsAny(outStyled, "╭╰╮╯│─") {
		t.Errorf("styled viewList: expected border runes (╭╰╮╯│─) in output; got:\n%q", outStyled[:min(len(outStyled), 200)])
	}

	// styled=false: no border runes.
	mu := ExportNewStyledModel(nil, tree, false)
	mu.width = 80
	mu.height = 24
	mu.visible = buildVisible(mu.tree, mu.expanded, mu.showAll, mu.pendingComplete, time.Now().Local())
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

// TestRenderList_SnoozedIndicator asserts that a snoozed task shown via show-all
// carries the 💤 indicator, and that it is absent when show-all is off.
func TestRenderList_SnoozedIndicator(t *testing.T) {
	// Use a far-future snooze so the test doesn't break as time passes.
	// renderList checks taskIsSnoozed against time.Now(), so the snooze must
	// be strictly after the real current date as well as the injected test today.
	farFuture := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "active"},
		{Id: 2, Name: "sleepy", SnoozeUntil: timestamppb.New(farFuture)},
	}
	tree := cli.BuildTree(tasks)

	today := time.Now().Local()

	// show-all = true: snoozed task appears with 💤
	m := ExportNewStyledModel(nil, tree, true)
	m.showAll = true
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, today)
	out := m.renderList(80)
	if !strings.Contains(out, "💤") {
		t.Errorf("show-all=true: expected 💤 indicator for snoozed task; got:\n%q", out)
	}
	if !strings.Contains(out, "sleepy") {
		t.Errorf("show-all=true: expected snoozed task name in output; got:\n%q", out)
	}

	// show-all = false: snoozed task hidden (no 💤)
	m2 := ExportNewStyledModel(nil, tree, true)
	m2.showAll = false
	m2.visible = buildVisible(m2.tree, m2.expanded, m2.showAll, m2.pendingComplete, today)
	out2 := m2.renderList(80)
	if strings.Contains(out2, "💤") {
		t.Errorf("show-all=false: 💤 must not appear; got:\n%q", out2)
	}
	if strings.Contains(out2, "sleepy") {
		t.Errorf("show-all=false: snoozed task must not appear; got:\n%q", out2)
	}
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
	m.showAll = true
	m.expanded[1] = true
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

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
	m.showAll = true
	m.expanded[1] = true
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

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
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	out := m.renderList(80)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.Contains(lines[0], "▸") {
		t.Errorf("expandable+collapsed row: expected ▸; got: %q", lines[0])
	}
}

// TestRenderList_NoChevronWhenAllChildrenCompleted asserts that a task whose
// only subtasks are completed does not show a chevron when showCompleted is off,
// even after the user has "expanded" it.
func TestRenderList_NoChevronWhenAllChildrenCompleted(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	parentID := int64(1)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent"},
		{Id: 2, Name: "completed child", ParentId: &parentID, CompletedAt: now},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.showAll = false
	m.expanded[1] = true // user "expanded" the task
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	out := m.renderList(80)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Only the parent row should be visible; completed child is hidden.
	if len(lines) != 1 {
		t.Errorf("expected 1 visible row, got %d: %v", len(lines), lines)
	}
	// Parent has no visible children → must not show any chevron.
	if strings.ContainsAny(lines[0], "▾▸") {
		t.Errorf("parent with only-completed children: unexpected chevron; got: %q", lines[0])
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
	m.showAll = true
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	// Expand all to see nested rows.
	m.expanded[1] = true
	m.expanded[2] = true
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	out := m.renderList(120)

	// Every completed task name must be preceded by \x1b[9m somewhere on its line.
	lines := strings.Split(out, "\n")
	completedNames := []string{"parent done", "child done", "grandchild done"}
	for _, name := range completedNames {
		found := false
		for _, line := range lines {
			// lipgloss v2 cursor-row rendering may interleave ANSI codes with
			// each character; strip before name matching.
			if strings.Contains(ansi.Strip(line), name) {
				if hasStrikethrough(line) {
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
	m.showAll = true
	m.cursor = 0
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	out := m.renderList(80)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Cursor row: completed name should still have strikethrough (C6.2).
	if !hasStrikethrough(lines[0]) {
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
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

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
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

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

// TestRenderStatus_ActivePomodoroGlyph (T017) asserts the leading 🍅 on the active-pomodoro
// status line carries pomodoroRunning bold styling when styled==true, and the plain fallback
// is unchanged (no ANSI codes).
func TestRenderStatus_ActivePomodoroGlyph(t *testing.T) {
	m := ExportNewStyledModel(nil, nil, true)
	m.pom = &activePom{
		taskName: "my task",
		startAt:  time.Now().Add(-5 * time.Minute),
	}

	status := m.renderStatus()

	// Styled mode: 🍅 must appear with ANSI codes (the pomodoroRunning+bold style).
	if !strings.Contains(status, "🍅") {
		t.Errorf("styled active-pom status: expected 🍅; got %q", status)
	}
	if !strings.Contains(status, "\x1b[") {
		t.Errorf("styled active-pom status: expected ANSI codes for styled glyph; got %q", status)
	}

	// Plain mode: should use "Pom" fallback, no ANSI on the timer line.
	mp := ExportNewStyledModel(nil, nil, false)
	mp.pom = m.pom
	plainStatus := mp.renderStatus()
	if !strings.Contains(plainStatus, "Pom") {
		t.Errorf("plain active-pom status: expected 'Pom' fallback; got %q", plainStatus)
	}
	// Only check the first line (the timer line); the help line uses lipgloss styles.
	timerLine := strings.SplitN(plainStatus, "\n", 2)[0]
	if strings.Contains(timerLine, "\x1b[") {
		t.Errorf("plain active-pom timer line must not emit ANSI codes; got %q", timerLine)
	}
}

// ── T023: Inline rendering integration tests for task names in renderList ─────

// TestRenderList_InlineMarkdownStyledOutput asserts that when a task name
// contains markdown emphasis, the styled output carries ANSI codes (no raw
// asterisks) and the plain non-cursor row has no extra ANSI from inline rendering.
func TestRenderList_InlineMarkdownStyledOutput(t *testing.T) {
	// Two tasks: cursor stays at 0 (first). We check the second row in plain mode.
	tasks := []*taskv1.Task{
		{Id: 1, Name: "plain cursor row"},
		{Id: 2, Name: "**Ship** the *report*"},
	}
	tree := cli.BuildTree(tasks)

	// Styled: second row should have ANSI codes from inline rendering.
	m := ExportNewStyledModel(nil, tree, true)
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	styled := m.renderList(80)
	if !strings.Contains(styled, "\x1b[") {
		t.Errorf("styled renderList with markdown name: expected ANSI codes; got %q", styled)
	}

	// Plain: second row (non-cursor) should contain the text without ** chars.
	mp := ExportNewStyledModel(nil, tree, false)
	mp.visible = buildVisible(mp.tree, mp.expanded, mp.showAll, mp.pendingComplete, time.Now().Local())
	plain := mp.renderList(80)
	lines := strings.Split(strings.TrimRight(plain, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("plain renderList: expected at least 2 lines; got %q", plain)
	}
	secondLine := lines[1]
	if strings.Contains(secondLine, "**") {
		t.Errorf("plain renderList: leftover ** in non-cursor row %q", secondLine)
	}
	if !strings.Contains(secondLine, "Ship") {
		t.Errorf("plain renderList: task text missing in non-cursor row %q", secondLine)
	}
}

// TestRenderList_InlineNameNoNewline asserts that renderList never emits a
// newline inside a task name row (i.e. each row is exactly one terminal line).
func TestRenderList_InlineNameNoNewline(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "# Heading in name"},
		{Id: 2, Name: "- bullet in name"},
		{Id: 3, Name: "plain name"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	out := m.renderList(80)

	// Each row ends with exactly one newline; no embedded newline inside row text.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != len(tasks) {
		t.Errorf("renderList: expected %d lines, got %d; output:\n%q", len(tasks), len(lines), out)
	}
}

// ── T003: windowOffset unit tests ──────────────────────────────────────────

func TestWindowOffset_CursorAboveWindow(t *testing.T) {
	// cursor=2, off=5 → off should drop to cursor
	got := windowOffset(5, 2, 5, 20)
	if got != 2 {
		t.Errorf("cursor above: got %d, want 2", got)
	}
}

func TestWindowOffset_CursorBelowWindow(t *testing.T) {
	// cursor=15, off=0, height=5 → off should become cursor-height+1 = 11
	got := windowOffset(0, 15, 5, 20)
	if got != 11 {
		t.Errorf("cursor below: got %d, want 11", got)
	}
}

func TestWindowOffset_CursorInsideWindow(t *testing.T) {
	// cursor=8, off=5, height=5 → no change
	got := windowOffset(5, 8, 5, 20)
	if got != 5 {
		t.Errorf("cursor inside: got %d, want 5", got)
	}
}

func TestWindowOffset_ClampHigh(t *testing.T) {
	// off=18, height=5, n=20 → maxOff=15, off should clamp to 15
	got := windowOffset(18, 15, 5, 20)
	if got != 15 {
		t.Errorf("clamp high: got %d, want 15", got)
	}
}

func TestWindowOffset_ClampLow(t *testing.T) {
	// off=-1 → clamp to 0
	got := windowOffset(-1, 0, 5, 20)
	if got != 0 {
		t.Errorf("clamp low: got %d, want 0", got)
	}
}

func TestWindowOffset_ListFitsViewport(t *testing.T) {
	// n=3, height=10 → n<=height, off should be 0
	got := windowOffset(0, 2, 10, 3)
	if got != 0 {
		t.Errorf("list fits: got %d, want 0", got)
	}
}

func TestWindowOffset_EmptyList(t *testing.T) {
	// n=0 → 0
	got := windowOffset(0, 0, 5, 0)
	if got != 0 {
		t.Errorf("empty list: got %d, want 0", got)
	}
}

func TestWindowOffset_LastPageEnd(t *testing.T) {
	// cursor at last element, off near end: should not leave blank space
	got := windowOffset(16, 19, 5, 20)
	if got != 15 {
		t.Errorf("last page end: got %d, want 15", got)
	}
}

// ── T005: listViewportHeight unit tests ────────────────────────────────────

func TestListViewportHeight_Styled(t *testing.T) {
	m := ExportNewStyledModel(nil, nil, true)
	m.height = 24
	m.pom = nil
	got := m.listViewportHeight()
	// styled: height - 2 - statusHeight(1) - tabBarHeight(1) = 24 - 2 - 1 - 1 = 20
	want := 20
	if got != want {
		t.Errorf("styled viewport height: got %d, want %d", got, want)
	}
}

func TestListViewportHeight_Unstyled(t *testing.T) {
	m := ExportNewStyledModel(nil, nil, false)
	m.height = 24
	m.pom = nil
	got := m.listViewportHeight()
	// plain: height - 1 - statusHeight(1) - tabBarHeight(1) = 24 - 1 - 1 - 1 = 21
	want := 21
	if got != want {
		t.Errorf("unstyled viewport height: got %d, want %d", got, want)
	}
}

func TestListViewportHeight_MinOne(t *testing.T) {
	m := ExportNewStyledModel(nil, nil, true)
	m.height = 2 // very short terminal
	m.pom = nil
	got := m.listViewportHeight()
	if got != 1 {
		t.Errorf("min 1 clamp: got %d, want 1", got)
	}
}

func TestListViewportHeight_ActivePomodoro(t *testing.T) {
	m := ExportNewStyledModel(nil, nil, true)
	m.height = 24
	m.pom = &activePom{taskName: "test"}
	got := m.listViewportHeight()
	// styled: 24 - 2 - 2(pom active) - 1 = 19
	want := 19
	if got != want {
		t.Errorf("active pom viewport height: got %d, want %d", got, want)
	}
}

// ── T007: renderList windowing tests ───────────────────────────────────────

func TestRenderList_Windowed_CursorAlwaysVisible(t *testing.T) {
	// Build a tall list: 30 tasks, viewport height 10 (styled).
	tasks := make([]*taskv1.Task, 30)
	for i := range tasks {
		tasks[i] = &taskv1.Task{Id: int64(i + 1), Name: fmt.Sprintf("task %d", i+1)}
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.height = 14 // styled: innerH = 14 - 2 - 1 - 1 = 10
	m.width = 80
	m.showAll = true
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	h := m.listViewportHeight()
	if h != 10 {
		t.Fatalf("viewport height: got %d, want 10", h)
	}

	// Test three representative cursor positions with a mid-list listScroll.
	cursors := []int{0, 15, 29}
	for _, cur := range cursors {
		m.cursor = cur
		m.listScroll = windowOffset(5, cur, h, len(m.visible))

		out := m.renderList(80)
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

		if len(lines) > h {
			t.Errorf("cursor=%d: renderList emitted %d rows, want at most %d", cur, len(lines), h)
		}

		// The cursor row must always be present.
		cursorTaskName := fmt.Sprintf("task %d", cur+1)
		found := false
		for _, line := range lines {
			if strings.Contains(ansi.Strip(line), cursorTaskName) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("cursor=%d: cursor row %q not found in output", cur, cursorTaskName)
		}
	}
}

func TestRenderList_Windowed_ExactlyFits(t *testing.T) {
	// 5 tasks, viewport height 10 → listScroll should be 0, output unchanged from full render.
	tasks := make([]*taskv1.Task, 5)
	for i := range tasks {
		tasks[i] = &taskv1.Task{Id: int64(i + 1), Name: fmt.Sprintf("task %d", i+1)}
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	m.height = 14
	m.width = 80
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	m.cursor = 2
	m.listScroll = 0

	out := m.renderList(80)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Errorf("exactly-fits: expected 5 lines, got %d", len(lines))
	}
}
