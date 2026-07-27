package goal_test

import (
	"testing"

	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"

	"github.com/pboyd/twig/internal/goal"
)

func ptr[T any](v T) *T { return &v }

func TestStateDisplayOrder(t *testing.T) {
	order := goal.StateDisplayOrder()
	want := []goalv1.GoalState{
		goalv1.GoalState_GOAL_STATE_IN_PROGRESS,
		goalv1.GoalState_GOAL_STATE_INCUBATING,
		goalv1.GoalState_GOAL_STATE_HOLD,
		goalv1.GoalState_GOAL_STATE_COMPLETED,
		goalv1.GoalState_GOAL_STATE_ARCHIVED,
	}
	if len(order) != len(want) {
		t.Fatalf("StateDisplayOrder len = %d, want %d", len(order), len(want))
	}
	for i, s := range want {
		if order[i] != s {
			t.Errorf("StateDisplayOrder[%d] = %v, want %v", i, order[i], s)
		}
	}
}

func TestDefaultVisible(t *testing.T) {
	tests := []struct {
		state   goalv1.GoalState
		visible bool
	}{
		{goalv1.GoalState_GOAL_STATE_IN_PROGRESS, true},
		{goalv1.GoalState_GOAL_STATE_INCUBATING, true},
		{goalv1.GoalState_GOAL_STATE_HOLD, true},
		{goalv1.GoalState_GOAL_STATE_COMPLETED, false},
		{goalv1.GoalState_GOAL_STATE_ARCHIVED, false},
		{goalv1.GoalState_GOAL_STATE_UNSPECIFIED, false},
	}
	for _, tt := range tests {
		got := goal.DefaultVisible(tt.state)
		if got != tt.visible {
			t.Errorf("DefaultVisible(%v) = %v, want %v", tt.state, got, tt.visible)
		}
	}
}

func TestStateName(t *testing.T) {
	tests := []struct {
		state goalv1.GoalState
		name  string
	}{
		{goalv1.GoalState_GOAL_STATE_INCUBATING, "incubating"},
		{goalv1.GoalState_GOAL_STATE_IN_PROGRESS, "in progress"},
		{goalv1.GoalState_GOAL_STATE_HOLD, "hold"},
		{goalv1.GoalState_GOAL_STATE_COMPLETED, "completed"},
		{goalv1.GoalState_GOAL_STATE_ARCHIVED, "archived"},
	}
	for _, tt := range tests {
		got := goal.StateName(tt.state)
		if got != tt.name {
			t.Errorf("StateName(%v) = %q, want %q", tt.state, got, tt.name)
		}
	}
}

func TestParseState(t *testing.T) {
	roundTrips := []goalv1.GoalState{
		goalv1.GoalState_GOAL_STATE_INCUBATING,
		goalv1.GoalState_GOAL_STATE_IN_PROGRESS,
		goalv1.GoalState_GOAL_STATE_HOLD,
		goalv1.GoalState_GOAL_STATE_COMPLETED,
		goalv1.GoalState_GOAL_STATE_ARCHIVED,
	}
	for _, s := range roundTrips {
		name := goal.StateName(s)
		got, err := goal.ParseState(name)
		if err != nil {
			t.Errorf("ParseState(%q) error: %v", name, err)
			continue
		}
		if got != s {
			t.Errorf("ParseState(%q) = %v, want %v", name, got, s)
		}
	}

	// Unknown value returns an error.
	_, err := goal.ParseState("bogus")
	if err == nil {
		t.Error("ParseState(\"bogus\") expected error, got nil")
	}
}

// buildTasks creates a flat task list for testing.
// Each element is (id, parentID, goalID) where 0 means nil.
func buildTasks(specs []struct{ id, parentID, goalID int64 }) []*taskv1.Task {
	tasks := make([]*taskv1.Task, len(specs))
	for i, s := range specs {
		t := &taskv1.Task{Id: s.id}
		if s.parentID != 0 {
			t.ParentId = ptr(s.parentID)
		}
		if s.goalID != 0 {
			t.GoalId = ptr(s.goalID)
		}
		tasks[i] = t
	}
	return tasks
}

func TestEffectiveGoalID(t *testing.T) {
	// Task tree:
	//   1 (goal_id=10)
	//   └─ 2
	//      └─ 3
	//   4 (no goal)
	tasks := buildTasks([]struct{ id, parentID, goalID int64 }{
		{1, 0, 10},
		{2, 1, 0},
		{3, 2, 0},
		{4, 0, 0},
	})

	tests := []struct {
		name   string
		taskID int64
		wantID int64
		wantOK bool
	}{
		{"self has goal_id", 1, 10, true},
		{"parent has goal_id", 2, 10, true},
		{"grandparent has goal_id", 3, 10, true},
		{"no goal in ancestry", 4, 0, false},
		{"unknown task", 99, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := goal.EffectiveGoalID(tasks, tt.taskID)
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Errorf("EffectiveGoalID(_, %d) = (%d, %v), want (%d, %v)",
					tt.taskID, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestAssociationRoots(t *testing.T) {
	// Three tasks: two associated with goal 10, one with goal 20.
	tasks := buildTasks([]struct{ id, parentID, goalID int64 }{
		{1, 0, 10},
		{2, 0, 10},
		{3, 0, 20},
	})

	roots := goal.AssociationRoots(tasks, 10)
	if len(roots) != 2 {
		t.Fatalf("AssociationRoots(_, 10) len = %d, want 2", len(roots))
	}
	ids := map[int64]bool{roots[0].Id: true, roots[1].Id: true}
	if !ids[1] || !ids[2] {
		t.Errorf("AssociationRoots(_, 10) ids = %v, want {1, 2}", ids)
	}

	roots20 := goal.AssociationRoots(tasks, 20)
	if len(roots20) != 1 || roots20[0].Id != 3 {
		t.Errorf("AssociationRoots(_, 20) = %v, want [{Id:3}]", roots20)
	}

	rootsNone := goal.AssociationRoots(tasks, 99)
	if len(rootsNone) != 0 {
		t.Errorf("AssociationRoots(_, 99) = %v, want []", rootsNone)
	}
}

func TestSubtreeForGoal(t *testing.T) {
	// Tree structure:
	//   1 (goal_id=10)
	//   ├─ 2 (child of 1)
	//   │  └─ 4 (grandchild of 1)
	//   5 (goal_id=10, second root)
	//   └─ 6 (child of 5)
	//   3 (goal_id=20, different goal)
	tasks := buildTasks([]struct{ id, parentID, goalID int64 }{
		{1, 0, 10},
		{2, 1, 0},
		{4, 2, 0},
		{5, 0, 10},
		{6, 5, 0},
		{3, 0, 20},
	})

	subtree := goal.SubtreeForGoal(tasks, 10)

	// Should include tasks 1, 2, 4, 5, 6 — not 3.
	got := make(map[int64]bool, len(subtree))
	for _, tt := range subtree {
		got[tt.Id] = true
	}

	wantIn := []int64{1, 2, 4, 5, 6}
	for _, id := range wantIn {
		if !got[id] {
			t.Errorf("SubtreeForGoal(_, 10) missing task %d", id)
		}
	}

	if got[3] {
		t.Error("SubtreeForGoal(_, 10) should not include task 3 (different goal)")
	}

	if len(subtree) != 5 {
		t.Errorf("SubtreeForGoal(_, 10) len = %d, want 5", len(subtree))
	}
}

func TestCLINameRoundTripsThroughParseState(t *testing.T) {
	for _, s := range goal.StateDisplayOrder() {
		name := goal.CLIName(s)
		got, err := goal.ParseState(name)
		if err != nil {
			t.Errorf("ParseState(CLIName(%v)) = %v, want no error", s, err)
			continue
		}
		if got != s {
			t.Errorf("ParseState(CLIName(%v)) = %v, want %v", s, got, s)
		}
	}
}
