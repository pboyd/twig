package tui

import "testing"

// ── Pomodoro estimate persistence ──────────────────────────────────────────
//
// The edit/new-task form's Pomodoro estimate field is validated but, absent
// these fixes, never reaches the server: CreateTaskRequest and
// UpdateTaskRequest have no estimate field, so it must be set separately via
// SetEstimate.

// TestCreateTaskCmd_SetsEstimate verifies that a non-empty estimate on the
// create form is persisted via SetEstimate after CreateTask.
func TestCreateTaskCmd_SetsEstimate(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	msg := editSavedMsg{name: "New task", estimateStr: "3"}

	cmd := ExportCreateTaskCmd(tc, msg)
	cmd()

	if tc.lastSetEstimateReq == nil {
		t.Fatal("SetEstimate was not called")
	}
	if tc.lastSetEstimateReq.TaskId != 42 {
		t.Errorf("TaskId: want 42, got %d", tc.lastSetEstimateReq.TaskId)
	}
	if tc.lastSetEstimateReq.Estimate != 3 {
		t.Errorf("Estimate: want 3, got %d", tc.lastSetEstimateReq.Estimate)
	}
}

// TestCreateTaskCmd_NoEstimateSkipsSetEstimate verifies that leaving the
// estimate field blank on create does not call SetEstimate at all.
func TestCreateTaskCmd_NoEstimateSkipsSetEstimate(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	msg := editSavedMsg{name: "New task"}

	cmd := ExportCreateTaskCmd(tc, msg)
	cmd()

	if tc.lastSetEstimateReq != nil {
		t.Errorf("SetEstimate: want not called, got %+v", tc.lastSetEstimateReq)
	}
}

// TestUpdateTaskCmd_SetsEstimate verifies that editing a task's estimate is
// persisted via SetEstimate after UpdateTask.
func TestUpdateTaskCmd_SetsEstimate(t *testing.T) {
	tc := &fakeTaskClient{}
	taskID := int64(7)
	msg := editSavedMsg{taskID: &taskID, name: "task", estimateStr: "5"}

	cmd := ExportUpdateTaskCmd(tc, msg)
	cmd()

	if tc.lastSetEstimateReq == nil {
		t.Fatal("SetEstimate was not called")
	}
	if tc.lastSetEstimateReq.TaskId != 7 {
		t.Errorf("TaskId: want 7, got %d", tc.lastSetEstimateReq.TaskId)
	}
	if tc.lastSetEstimateReq.Estimate != 5 {
		t.Errorf("Estimate: want 5, got %d", tc.lastSetEstimateReq.Estimate)
	}
}

// TestUpdateTaskCmd_BlankEstimateClears verifies that blanking the estimate
// field on an edit clears it (full-replace semantics), matching how due and
// snooze already behave.
func TestUpdateTaskCmd_BlankEstimateClears(t *testing.T) {
	tc := &fakeTaskClient{}
	taskID := int64(7)
	msg := editSavedMsg{taskID: &taskID, name: "task", estimateStr: ""}

	cmd := ExportUpdateTaskCmd(tc, msg)
	cmd()

	if tc.lastSetEstimateReq == nil {
		t.Fatal("SetEstimate was not called")
	}
	if tc.lastSetEstimateReq.Estimate != 0 {
		t.Errorf("Estimate: want 0 (cleared), got %d", tc.lastSetEstimateReq.Estimate)
	}
}

// TestHandleEditSaved_InvalidEstimateKeepsFormOpen verifies that a
// non-integer or out-of-range estimate is rejected before any RPC, leaving
// the form open with an error.
func TestHandleEditSaved_InvalidEstimateKeepsFormOpen(t *testing.T) {
	tests := []struct {
		name        string
		estimateStr string
	}{
		{"non-integer", "abc"},
		{"out of range", "11"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &fakeTaskClient{}
			taskID := int64(7)
			m := buildTestModel()
			m.client = tc

			next, cmd := m.handleEditSaved(editSavedMsg{taskID: &taskID, name: "task", estimateStr: tt.estimateStr})
			m2 := next.(Model)

			if m2.err == nil {
				t.Fatal("expected m.err to be set for invalid estimate")
			}
			if cmd != nil {
				t.Fatal("expected no RPC command to run")
			}
			if tc.lastUpdateReq != nil {
				t.Error("UpdateTask must not be called when estimate is invalid")
			}
		})
	}
}

// TestGoalNewTask_SetsEstimate verifies that an estimate entered on the
// Goals-tab new-task form is carried onto the created task, matching the
// Tasks-tab create path.
func TestGoalNewTask_SetsEstimate(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := setUpGoalNewTask(tc, pc)

	next, cmd := m.Update(editSavedMsg{name: "New task", estimateStr: "4"})
	m2 := next.(Model)
	if m2.goal.err != nil {
		t.Fatalf("unexpected goal.err: %v", m2.goal.err)
	}
	if cmd == nil {
		t.Fatal("expected a command")
	}
	cmd()

	if tc.lastSetEstimateReq == nil {
		t.Fatal("SetEstimate was not called")
	}
	if tc.lastSetEstimateReq.Estimate != 4 {
		t.Errorf("Estimate: want 4, got %d", tc.lastSetEstimateReq.Estimate)
	}
}

// TestGoalNewTask_InvalidEstimateKeepsFormOpen verifies that a malformed
// estimate value on the Goals-tab new-task form is rejected before any RPC.
func TestGoalNewTask_InvalidEstimateKeepsFormOpen(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := setUpGoalNewTask(tc, pc)

	next, cmd := m.Update(editSavedMsg{name: "New task", estimateStr: "abc"})
	m2 := next.(Model)

	if m2.goal.err == nil {
		t.Fatal("expected goal.err to be set for invalid estimate")
	}
	if m2.goal.mode != goalNewTask {
		t.Errorf("goal.mode: want goalNewTask, got %v", m2.goal.mode)
	}
	if cmd != nil {
		if msg := cmd(); msg != nil {
			t.Errorf("expected no RPC command to run, got %T", msg)
		}
	}
	if tc.lastCreateReq != nil {
		t.Error("CreateTask must not be called when estimate is invalid")
	}
}
