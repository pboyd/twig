package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
)

// setUpGoalNewTask returns a Model positioned on the Goals tab, in
// goalNewTask mode, with a single visible goal under the cursor.
func setUpGoalNewTask(tc *fakeTaskClient, pc *fakePlanClient) Model {
	goals := []*goalv1.Goal{
		{Id: 7, Name: "Learn woodworking", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
	}
	m := ExportNewGoalModelWithPlan(tc, pc, goals)
	m.goal.mode = goalNewTask
	m.mode = modeNewRoot
	return m
}

// TestGoalNewTask_SnoozeApplied verifies that a snooze date entered on the
// Goals tab's new-task form is carried onto the created task, matching the
// Tasks tab create path.
func TestGoalNewTask_SnoozeApplied(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := setUpGoalNewTask(tc, pc)

	next, cmd := m.Update(editSavedMsg{name: "New task", snoozeStr: "2026-08-01"})
	m2 := next.(Model)
	if m2.goal.err != nil {
		t.Fatalf("unexpected goal.err: %v", m2.goal.err)
	}
	if cmd == nil {
		t.Fatal("expected a command")
	}
	cmd()

	if tc.lastCreateReq == nil {
		t.Fatal("CreateTask was not called")
	}
	if tc.lastCreateReq.SnoozeUntil == nil {
		t.Fatal("SnoozeUntil: want set, got nil")
	}
	got := tc.lastCreateReq.SnoozeUntil.AsTime().UTC().Format("2006-01-02")
	if got != "2026-08-01" {
		t.Errorf("SnoozeUntil: want %q, got %q", "2026-08-01", got)
	}
}

// TestGoalNewTask_PlanDayCallsAddPlanTask verifies that choosing a plan day
// on the Goals tab's new-task form schedules the created task, matching the
// Tasks tab create path.
func TestGoalNewTask_PlanDayCallsAddPlanTask(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := setUpGoalNewTask(tc, pc)

	next, cmd := m.Update(editSavedMsg{name: "New task", planDay: "2026-08-01"})
	m2 := next.(Model)
	if m2.goal.err != nil {
		t.Fatalf("unexpected goal.err: %v", m2.goal.err)
	}
	if cmd == nil {
		t.Fatal("expected a command")
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, sub := range batch {
			sub()
		}
	}

	if pc.addTaskReq == nil {
		t.Fatal("AddPlanTask was not called")
	}
	if pc.addTaskReq.Day != "2026-08-01" {
		t.Errorf("Day: want %q, got %q", "2026-08-01", pc.addTaskReq.Day)
	}
	if pc.addTaskReq.TaskId != 42 {
		t.Errorf("TaskId: want 42, got %d", pc.addTaskReq.TaskId)
	}
	if pc.addTaskReq.StartMinute != nil {
		t.Errorf("StartMinute: want nil, got %v", *pc.addTaskReq.StartMinute)
	}
}

// TestGoalNewTask_LinksToCursorGoal verifies the created task is still linked
// to the goal under the cursor via SetTaskGoal.
func TestGoalNewTask_LinksToCursorGoal(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := setUpGoalNewTask(tc, pc)

	_, cmd := m.Update(editSavedMsg{name: "New task"})
	if cmd == nil {
		t.Fatal("expected a command")
	}
	cmd()

	if tc.lastSetTaskGoalReq == nil {
		t.Fatal("SetTaskGoal was not called")
	}
	if tc.lastSetTaskGoalReq.TaskId != 42 {
		t.Errorf("TaskId: want 42, got %d", tc.lastSetTaskGoalReq.TaskId)
	}
	if tc.lastSetTaskGoalReq.GoalId == nil || *tc.lastSetTaskGoalReq.GoalId != 7 {
		t.Errorf("GoalId: want 7, got %v", tc.lastSetTaskGoalReq.GoalId)
	}
}

// TestGoalNewTask_NoPlanDayNoAddPlanTask is the control: leaving the plan
// selector untouched must not schedule the task.
func TestGoalNewTask_NoPlanDayNoAddPlanTask(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := setUpGoalNewTask(tc, pc)

	_, cmd := m.Update(editSavedMsg{name: "New task"})
	if cmd == nil {
		t.Fatal("expected a command")
	}
	cmd()

	if pc.addTaskReq != nil {
		t.Errorf("AddPlanTask: want not called, got %+v", pc.addTaskReq)
	}
}

// TestGoalNewTask_InvalidSnoozeKeepsFormOpen verifies that a malformed
// snooze value is rejected before any RPC, leaving the create form open with
// an error rather than surfacing an error only after a failed request.
func TestGoalNewTask_InvalidSnoozeKeepsFormOpen(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := setUpGoalNewTask(tc, pc)

	next, cmd := m.Update(editSavedMsg{name: "New task", snoozeStr: "not-a-date"})
	m2 := next.(Model)

	if m2.goal.err == nil {
		t.Fatal("expected goal.err to be set for invalid snooze")
	}
	if m2.goal.mode != goalNewTask {
		t.Errorf("goal.mode: want goalNewTask, got %v", m2.goal.mode)
	}
	if m2.mode != modeNewRoot {
		t.Errorf("mode: want modeNewRoot, got %v", m2.mode)
	}
	if cmd != nil {
		if msg := cmd(); msg != nil {
			t.Errorf("expected no RPC command to run, got %T", msg)
		}
	}
	if tc.lastCreateReq != nil {
		t.Error("CreateTask must not be called when snooze is invalid")
	}
}
