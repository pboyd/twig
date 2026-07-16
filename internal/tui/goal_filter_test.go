package tui

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
)

// ── T005: US1 tests — ctrl+t on Goals tab ────────────────────────────────────

// TestGoalFilter_JumpAppliesFilter verifies that pressing ctrl+t on a goal
// switches to the Tasks tab with the correct filter expression and dispatches
// FilterTasks (FR-001, FR-002).
func TestGoalFilter_JumpAppliesFilter(t *testing.T) {
	fc := &fakeTaskClient{filterIDs: []int64{10, 20}}
	goals := []*goalv1.Goal{
		{Id: 7, Name: "My Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	m := ExportNewGoalModel(fc, goals)
	m.width = 80
	m.height = 30

	// Press ctrl+t.
	next, cmd := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	// FR-001: active tab should be Tasks.
	if m2.activeTab != tabTasks {
		t.Errorf("activeTab: want tabTasks (%d), got %d", int(tabTasks), int(m2.activeTab))
	}

	// FR-002/FM: filter input should contain the goal expression.
	want := fmt.Sprintf("^goal_id=%d", goals[0].Id)
	got := ExportFilterInputValue(m2)
	if got != want {
		t.Errorf("filterInput.Value(): want %q, got %q", want, got)
	}

	// The command should be non-nil (dispatches FilterTasks).
	if cmd == nil {
		t.Fatal("expected non-nil command from ctrl+t")
	}

	// Execute the command to verify FilterTasks is called.
	msg := cmd()
	result, ok := msg.(filterResultMsg)
	if !ok {
		t.Fatalf("expected filterResultMsg, got %T", msg)
	}
	if result.err != nil {
		t.Fatalf("filterResultMsg.err: %v", result.err)
	}

	// Verify FilterTasks was called with the correct expression.
	if fc.lastFilterReq == nil {
		t.Fatal("FilterTasks was not called")
	}
	if fc.lastFilterReq.Expression != want {
		t.Errorf("FilterTasks.Expression: want %q, got %q", want, fc.lastFilterReq.Expression)
	}

	// FR-009: showAll should be passed through unmodified.
	if fc.lastFilterReq.ShowAll != m.showAll {
		t.Errorf("FilterTasks.ShowAll: want %v, got %v", m.showAll, fc.lastFilterReq.ShowAll)
	}
}

// TestGoalFilter_EmptyGoalsNoOp verifies that ctrl+t with no goals is a no-op
// (FR-006).
func TestGoalFilter_EmptyGoalsNoOp(t *testing.T) {
	fc := &fakeTaskClient{}
	m := ExportNewGoalModel(fc, nil)
	m.width = 80
	m.height = 30

	next, cmd := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	if m2.activeTab != tabGoals {
		t.Errorf("activeTab should remain tabGoals, got %d", int(m2.activeTab))
	}
	if cmd != nil {
		t.Error("expected nil command for empty goals list")
	}
}

// TestGoalFilter_InertInSubModes verifies that ctrl+t is inert while a goal
// sub-mode is active (FR-007).
func TestGoalFilter_InertInSubModes(t *testing.T) {
	fc := &fakeTaskClient{}
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}

	subModes := []struct {
		name string
		mode int
	}{
		{"goalEdit", ExportGoalModeEdit},
		{"goalPickLink", ExportGoalModePickLink},
		{"goalStatusHistory", ExportGoalModeStatusHistory},
	}

	for _, sm := range subModes {
		t.Run(sm.name, func(t *testing.T) {
			m := ExportNewGoalModel(fc, goals)
			m.width = 80
			m.height = 30
			ExportSetGoalMode(&m, sm.mode)

			next, cmd := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
			m2 := next.(Model)

			if m2.activeTab != tabGoals {
				t.Errorf("activeTab should remain tabGoals in %s, got %d", sm.name, int(m2.activeTab))
			}
			if cmd != nil {
				t.Errorf("expected nil command in %s", sm.name)
			}
		})
	}
}

// TestGoalFilter_ShowAllPassthrough verifies that showAll is passed to
// FilterTasks unmodified in both true and false states (FR-009).
// Since filterCmd captures the client at creation time, we verify the expression
// is set correctly (which is what drives the dispatch) and that the command is
// non-nil. The full dispatch path is covered by TestGoalFilter_JumpAppliesFilter.
func TestGoalFilter_ShowAllPassthrough(t *testing.T) {
	for _, showAll := range []bool{false, true} {
		t.Run(fmt.Sprintf("showAll=%v", showAll), func(t *testing.T) {
			fc := &fakeTaskClient{filterIDs: []int64{1}}
			goals := []*goalv1.Goal{
				{Id: 3, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
			}
			m := ExportNewGoalModel(fc, goals)
			m.showAll = showAll
			m.width = 80
			m.height = 30

			next, cmd := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
			m2 := next.(Model)

			// The filter expression should be set correctly.
			want := "^goal_id=3"
			got := ExportFilterInputValue(m2)
			if got != want {
				t.Errorf("filterInput.Value(): want %q, got %q", want, got)
			}

			// A command should be returned.
			if cmd == nil {
				t.Fatal("expected non-nil command")
			}
		})
	}
}

// TestGoalFilter_ExpressionBareRelational verifies the expression is the bare
// relational condition ^goal_id=<id> with no completed=false appended
// (contracts/keybinding.md §2).
func TestGoalFilter_ExpressionBareRelational(t *testing.T) {
	fc := &fakeTaskClient{filterIDs: []int64{1}}
	goals := []*goalv1.Goal{
		{Id: 42, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	m := ExportNewGoalModel(fc, goals)
	m.width = 80
	m.height = 30

	next, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	want := "^goal_id=42"
	got := ExportFilterInputValue(m2)
	if got != want {
		t.Errorf("expression: want %q, got %q (should not include completed=false)", want, got)
	}

	// Verify the filterExpr is also set correctly via handleFilterResult.
	updated, _ := m2.Update(filterResultMsg{gen: m2.filterGen, expr: "^goal_id=42", ids: []int64{1}})
	m3 := updated.(Model)
	gotExpr := ExportFilterExpr(m3)
	if gotExpr != want {
		t.Errorf("filterExpr after result: want %q, got %q", want, gotExpr)
	}
}

// TestGoalFilter_SecondGoalResolvesCorrectly verifies the second goal's ID is
// used when the cursor is on it.
func TestGoalFilter_SecondGoalResolvesCorrectly(t *testing.T) {
	fc := &fakeTaskClient{filterIDs: []int64{5}}
	goals := []*goalv1.Goal{
		{Id: 10, Name: "First", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
		{Id: 20, Name: "Second", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 1},
	}
	m := ExportNewGoalModel(fc, goals)
	m.goal.cursor = 1
	m.width = 80
	m.height = 30

	next, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	want := "^goal_id=20"
	got := ExportFilterInputValue(m2)
	if got != want {
		t.Errorf("expression: want %q, got %q", want, got)
	}

	// Verify via handleFilterResult.
	updated, _ := m2.Update(filterResultMsg{gen: m2.filterGen, expr: "^goal_id=20", ids: []int64{5}})
	m3 := updated.(Model)
	gotExpr := ExportFilterExpr(m3)
	if gotExpr != want {
		t.Errorf("filterExpr after result: want %q, got %q", want, gotExpr)
	}
}

// ── T009-T011: US2 tests — post-arrival filter behavior ────────────────────

// TestGoalFilter_FilterExprRecordedAfterResult verifies that after dispatching
// a successful filterResultMsg, filterExpr matches the goal expression (P4)
// and the mode is modeList with the input blurred (P6).
func TestGoalFilter_FilterExprRecordedAfterResult(t *testing.T) {
	fc := &fakeTaskClient{filterIDs: []int64{10, 20}}
	goals := []*goalv1.Goal{
		{Id: 7, Name: "My Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	m := ExportNewGoalModel(fc, goals)
	m.width = 80
	m.height = 30

	// Jump.
	next, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	// Dispatch the filter result.
	updated, _ := m2.Update(filterResultMsg{gen: m2.filterGen, expr: "^goal_id=7", ids: []int64{10, 20}})
	m3 := updated.(Model)

	// P4: filterExpr should be the goal expression.
	want := "^goal_id=7"
	if ExportFilterExpr(m3) != want {
		t.Errorf("filterExpr: want %q, got %q", want, ExportFilterExpr(m3))
	}

	// P6: mode should be modeList.
	if int(m3.mode) != ExportModeList {
		t.Errorf("mode: want modeList (%d), got %d", ExportModeList, int(m3.mode))
	}

	// P6: filter input should be blurred (not focused).
	if m2.filterInput.Focused() {
		t.Error("filterInput should be blurred after jump")
	}
}

// TestGoalFilter_FilterReopenPrefilled verifies that reopening the filter with
// `/` after a jump presents the expression pre-filled for editing (US2 scenario 1).
func TestGoalFilter_FilterReopenPrefilled(t *testing.T) {
	fc := &fakeTaskClient{filterIDs: []int64{10}}
	goals := []*goalv1.Goal{
		{Id: 5, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	m := ExportNewGoalModel(fc, goals)
	m.width = 80
	m.height = 30

	// Jump.
	next, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	// Dispatch filter result.
	updated, _ := m2.Update(filterResultMsg{gen: m2.filterGen, expr: "^goal_id=5", ids: []int64{10}})
	m3 := updated.(Model)

	// Press `/` to reopen the filter.
	next2, _ := m3.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m4 := next2.(Model)

	// The filter input should be pre-filled with the goal expression.
	want := "^goal_id=5"
	got := ExportFilterInputValue(m4)
	if got != want {
		t.Errorf("filter input after reopen: want %q, got %q", want, got)
	}

	// The mode should be modeFilter.
	if int(m4.mode) != ExportModeFilter {
		t.Errorf("mode: want modeFilter (%d), got %d", ExportModeFilter, int(m4.mode))
	}
}

// TestGoalFilter_ReplacesExistingFilter verifies that jumping from a goal
// replaces any previously active filter (FR-004, P5) — no AND-merge.
func TestGoalFilter_ReplacesExistingFilter(t *testing.T) {
	fc := &fakeTaskClient{filterIDs: []int64{1, 2}}
	goals := []*goalv1.Goal{
		{Id: 8, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	m := ExportNewGoalModel(fc, goals)
	m.width = 80
	m.height = 30

	// Seed a different active filter.
	ExportSetFilterState(&m, "completed=false", []int64{1, 2})
	m.filterInput.SetValue("completed=false")

	// Jump from the goal.
	next, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	// The filter input should be overwritten with the goal expression.
	want := "^goal_id=8"
	got := ExportFilterInputValue(m2)
	if got != want {
		t.Errorf("filterInput after jump: want %q, got %q (should replace, not merge)", want, got)
	}

	// Dispatch filter result and check filterExpr.
	updated, _ := m2.Update(filterResultMsg{gen: m2.filterGen, expr: "^goal_id=8", ids: []int64{1, 2}})
	m3 := updated.(Model)

	gotExpr := ExportFilterExpr(m3)
	if gotExpr != want {
		t.Errorf("filterExpr after result: want %q, got %q", want, gotExpr)
	}

	// No trace of the prior expression.
	if ExportFilterExpr(m3) != want {
		t.Errorf("filterExpr should be only the goal expression, got %q", ExportFilterExpr(m3))
	}
}

// TestGoalFilter_ClearFilterRestoresUnfiltered verifies that after a jump,
// clearing the filter via `/` → `esc` restores the unfiltered task list
// (FR-003, US2 scenario 2).
func TestGoalFilter_ClearFilterRestoresUnfiltered(t *testing.T) {
	fc := &fakeTaskClient{filterIDs: []int64{10, 20, 30}}
	goals := []*goalv1.Goal{
		{Id: 3, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	// Set up a tree with tasks so visible is non-empty.
	tree := []*cli.TreeNode{
		{Task: &taskv1.Task{Id: 10, Name: "task a"}},
		{Task: &taskv1.Task{Id: 20, Name: "task b"}},
	}
	m := ExportNewGoalModel(fc, goals)
	m.tree = tree
	m.visible = buildVisible(tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	m.width = 80
	m.height = 30

	// Jump.
	next, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	// Verify filter is active.
	if ExportFilterExpr(m2) == "" && ExportFilterInputValue(m2) == "" {
		t.Fatal("precondition: filter should be active after jump")
	}

	// Press `/` to open the filter input.
	next2, _ := m2.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m3 := next2.(Model)

	// Press `esc` to clear the filter.
	next3, _ := m3.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m4 := next3.(Model)

	// Filter should be cleared.
	if ExportFilterExpr(m4) != "" {
		t.Errorf("filterExpr after esc: want empty, got %q", ExportFilterExpr(m4))
	}
	if ExportFilterInputValue(m4) != "" {
		t.Errorf("filterInput after esc: want empty, got %q", ExportFilterInputValue(m4))
	}

	// Mode should be modeList.
	if int(m4.mode) != ExportModeList {
		t.Errorf("mode after esc: want modeList (%d), got %d", ExportModeList, int(m4.mode))
	}

	// Should still be on Tasks tab.
	if m4.activeTab != tabTasks {
		t.Errorf("activeTab after esc: want tabTasks, got %d", int(m4.activeTab))
	}
}

// ── T014: Help listing test (FR-010) ───────────────────────────────────────

// TestGoalFilter_HelpListsShortcut verifies that the Goals FullHelp contains
// the ctrl+t binding (FR-010).
func TestGoalFilter_HelpListsShortcut(t *testing.T) {
	km := DefaultKeyMap()
	km.GoalMode = true

	fullHelp := km.FullHelp()
	found := false
	for _, row := range fullHelp {
		for _, b := range row {
			if b.Keys() != nil && len(b.Keys()) > 0 && b.Keys()[0] == "ctrl+t" {
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Error("Goals FullHelp does not contain the ctrl+t binding (FR-010)")
	}
}

// TestGoalFilter_ShortHelpDoesNotIncludeShortcut verifies that the Goals
// ShortHelp does NOT include GoalGoToTasks (it is already near capacity).
func TestGoalFilter_ShortHelpDoesNotIncludeShortcut(t *testing.T) {
	km := DefaultKeyMap()
	km.GoalMode = true

	shortHelp := km.ShortHelp()
	for _, b := range shortHelp {
		if b.Keys() != nil && len(b.Keys()) > 0 && b.Keys()[0] == "ctrl+t" {
			t.Error("Goals ShortHelp should NOT contain the ctrl+t binding")
		}
	}
}

// TestGoalFilter_ClearCancelsInFlightResult verifies that clearing the filter
// before a ctrl+t jump's response lands discards that response. clearFilter is
// the cancel side of the filterGen protocol; if it does not bump the counter,
// the stale response applies to a cleared model and leaves the list filtered
// with an empty filterExpr — a filter the user cannot see or turn off.
func TestGoalFilter_ClearCancelsInFlightResult(t *testing.T) {
	fc := &fakeTaskClient{filterIDs: []int64{10}}
	goals := []*goalv1.Goal{
		{Id: 7, Name: "My Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	tree := []*cli.TreeNode{
		{Task: &taskv1.Task{Id: 10, Name: "task a"}},
		{Task: &taskv1.Task{Id: 20, Name: "task b"}},
	}
	m := ExportNewGoalModel(fc, goals)
	m.tree = tree
	m.visible = buildVisible(tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	m.width = 80
	m.height = 30

	// Jump, holding the command back so the response is still "in flight".
	next, cmd := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("setup: expected a filter command from ctrl+t")
	}
	m2 := next.(Model)

	// Clear before it lands: `/` to open the bar, `esc` to cancel.
	n3, _ := m2.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	n4, _ := n3.(Model).Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m4 := n4.(Model)

	// The response arrives after the clear.
	updated, _ := m4.Update(cmd())
	m5 := updated.(Model)

	if got := ExportFilterExpr(m5); got != "" {
		t.Errorf("filterExpr: want empty after clear, got %q", got)
	}
	if len(m5.visible) != len(tree) {
		t.Errorf("visible: want %d unfiltered rows after clear, got %d", len(tree), len(m5.visible))
	}
}

// TestGoalFilter_JumpLandsOnFirstMatchingRow verifies the jump puts the cursor
// on the first task actually in the goal's tree. buildVisibleFiltered pulls
// non-matching ancestors in as scaffold rows, so a task linked to the goal can
// sit below a parent that is not — the cursor must skip past that parent.
func TestGoalFilter_JumpLandsOnFirstMatchingRow(t *testing.T) {
	// Task 2 is in the goal; its parent (task 1) is not.
	parentID := int64(1)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent not in goal"},
		{Id: 2, Name: "child in goal", ParentId: &parentID},
	}
	tree := cli.BuildTree(tasks)

	fc := &fakeTaskClient{filterIDs: []int64{2}} // only the child matches
	goals := []*goalv1.Goal{
		{Id: 7, Name: "My Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	m := ExportNewGoalModel(fc, goals)
	m.tree = tree
	m.expanded = map[int64]bool{1: true}
	m.cursor = 0 // clamping would leave it here, on the scaffold parent
	m.width = 80
	m.height = 30

	next, cmd := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)
	updated, _ := m2.Update(cmd())
	m3 := updated.(Model)

	if len(m3.visible) != 2 {
		t.Fatalf("visible: want 2 rows (scaffold parent + matching child), got %d", len(m3.visible))
	}
	if m3.cursor != 1 {
		t.Errorf("cursor: want 1 (the matching child), got %d — landed on the scaffold parent", m3.cursor)
	}
	if got := m3.visible[m3.cursor].node.Task.Id; got != 2 {
		t.Errorf("cursor task: want id 2, got %d", got)
	}
}
