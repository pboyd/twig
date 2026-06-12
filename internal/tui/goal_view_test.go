package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
)

// ── T017: Goals tab unit tests ───────────────────────────────────────────────

// TestGoal_StartupTabIsTasks verifies that the default active tab after
// newModel is Tasks, not Goals — even though tabGoals == 0 (first iota).
func TestGoal_StartupTabIsTasks(t *testing.T) {
	m := ExportNewModel(nil, nil)
	got := int(m.activeTab)
	if got != int(tabTasks) {
		t.Errorf("startup tab: want tabTasks (%d), got %d", int(tabTasks), got)
	}
	if got == int(tabGoals) {
		t.Errorf("startup tab must NOT be tabGoals (%d)", int(tabGoals))
	}
}

// TestGoal_TabOrderShiftTabFromTasksReachesGoals verifies that Shift+Tab from
// the Tasks tab activates the Goals tab.
func TestGoal_TabOrderShiftTabFromTasksReachesGoals(t *testing.T) {
	m := ExportNewModel(nil, nil)
	// Confirm we start on Tasks.
	if m.activeTab != tabTasks {
		t.Fatalf("precondition: expected tabTasks, got %d", m.activeTab)
	}

	// Shift+Tab from Tasks → Goals.
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m2 := next.(Model)

	if m2.activeTab != tabGoals {
		t.Errorf("Shift+Tab from Tasks: want tabGoals (%d), got %d", int(tabGoals), int(m2.activeTab))
	}
}

// TestGoal_GroupOrderCommittedBeforeIncubating verifies that Committed goals
// appear before Incubating goals in the rendered left pane.
func TestGoal_GroupOrderCommittedBeforeIncubating(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Plant a garden", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Run a marathon", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	// Render the goal list (unstyled).
	out := m.renderGoalList(40)

	committedIdx := strings.Index(out, "Committed")
	incubatingIdx := strings.Index(out, "Incubating")

	if committedIdx < 0 {
		t.Fatal("renderGoalList: 'Committed' header not found")
	}
	if incubatingIdx < 0 {
		t.Fatal("renderGoalList: 'Incubating' header not found")
	}
	if committedIdx >= incubatingIdx {
		t.Errorf("group order: Committed (%d) should appear before Incubating (%d)", committedIdx, incubatingIdx)
	}
}

// TestGoal_CompletedHiddenByDefault verifies that completed goals are not
// rendered when showAll is false, and are revealed when 'c' is pressed.
func TestGoal_CompletedHiddenByDefault(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Active goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
		{Id: 2, Name: "Done goal", State: goalv1.GoalState_GOAL_STATE_COMPLETED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	// showAll should be false by default.
	if m.goal.showAll {
		t.Fatal("precondition: goal.showAll should be false by default")
	}

	out := m.renderGoalList(40)
	if strings.Contains(out, "Done goal") {
		t.Error("completed goal should be hidden when showAll=false")
	}

	// Press 'c' to toggle showAll.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m2 := next.(Model)

	if !m2.goal.showAll {
		t.Error("after 'c': goal.showAll should be true")
	}

	out2 := m2.renderGoalList(40)
	if !strings.Contains(out2, "Done goal") {
		t.Error("completed goal should be visible after pressing 'c'")
	}
}

// TestGoal_ArchivedHiddenByDefault verifies that archived goals follow the same
// hide/reveal behavior as completed goals.
func TestGoal_ArchivedHiddenByDefault(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Active goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Old idea", State: goalv1.GoalState_GOAL_STATE_ARCHIVED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	out := m.renderGoalList(40)
	if strings.Contains(out, "Old idea") {
		t.Error("archived goal should be hidden when showAll=false")
	}

	// Toggle show-all with 'c'.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m2 := next.(Model)

	out2 := m2.renderGoalList(40)
	if !strings.Contains(out2, "Old idea") {
		t.Error("archived goal should be visible after pressing 'c'")
	}
}

// TestGoal_EmptyStateCopyWhenNoVisibleGoals verifies the empty-state message
// shown when there are no visible goals.
func TestGoal_EmptyStateCopyWhenNoVisibleGoals(t *testing.T) {
	m := ExportNewGoalModel(nil, nil) // no goals at all

	out := m.renderGoalList(40)

	want := "A blank canvas! Press 'n' to plant your first goal."
	if !strings.Contains(out, want) {
		t.Errorf("empty-state copy: want %q in output, got:\n%q", want, out)
	}
}

// TestGoal_EmptyStateCopyWhenAllGoalsHidden verifies the empty-state message
// when goals exist but all are hidden by the showAll filter.
func TestGoal_EmptyStateCopyWhenAllGoalsHidden(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Finished goal", State: goalv1.GoalState_GOAL_STATE_COMPLETED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	// showAll=false by default, so the completed goal is hidden.

	out := m.renderGoalList(40)

	want := "A blank canvas! Press 'n' to plant your first goal."
	if !strings.Contains(out, want) {
		t.Errorf("empty-state copy with all-hidden goals: want %q in output, got:\n%q", want, out)
	}
}

// TestGoal_DeleteConfirmCopyMentionsTasksStickAround verifies that the delete
// confirmation notice contains the expected copy about tasks.
func TestGoal_DeleteConfirmCopyMentionsTasksStickAround(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Learn woodworking", State: goalv1.GoalState_GOAL_STATE_COMMITTED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	// Press Ctrl+D to trigger goal delete confirmation.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	m2 := next.(Model)

	notice := m2.notice
	if !strings.Contains(notice, "Tasks attached to it will stick around.") {
		t.Errorf("delete confirm notice: want substring 'Tasks attached to it will stick around.', got:\n%q", notice)
	}
}
