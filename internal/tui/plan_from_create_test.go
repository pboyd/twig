package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/config"
)

// failingTaskClient is a minimal TaskServiceClient that always fails on CreateTask.
type failingTaskClient struct {
	taskv1connect.TaskServiceClient
}

func (f *failingTaskClient) CreateTask(_ context.Context, _ *connect.Request[taskv1.CreateTaskRequest]) (*connect.Response[taskv1.CreateTaskResponse], error) {
	return nil, connect.NewError(connect.CodeInternal, errors.New("create failed"))
}

// CT-04: AddPlanTask is called with no start_minute and duration_minute: 0.
func TestCT04_AddPlanTaskCalledWithNoStartMinute(t *testing.T) {
	taskID := int64(42)
	tc := &fakeTaskClient{createTaskID: taskID}
	pc := &fakePlanClient{}

	msg := editSavedMsg{
		name:    "new task",
		planDay: "2026-07-20",
	}
	cmd := ExportCreateTaskCmdWithPlan(tc, pc, msg)
	result := cmd()

	if _, ok := result.(refreshedMsg); !ok {
		t.Fatalf("expected refreshedMsg, got %T", result)
	}

	if pc.addTaskReq == nil {
		t.Fatal("AddPlanTask was not called")
	}
	if pc.addTaskReq.Day != "2026-07-20" {
		t.Errorf("Day: want %q, got %q", "2026-07-20", pc.addTaskReq.Day)
	}
	if pc.addTaskReq.DurationMinute != 0 {
		t.Errorf("DurationMinute: want 0, got %d", pc.addTaskReq.DurationMinute)
	}
	if pc.addTaskReq.StartMinute != nil {
		t.Errorf("StartMinute: want nil, got %v", *pc.addTaskReq.StartMinute)
	}
	if pc.addTaskReq.TaskId != taskID {
		t.Errorf("TaskId: want %d, got %d", taskID, pc.addTaskReq.TaskId)
	}
}

// CT-05: Untouched control ⇒ AddPlanTask never called.
func TestCT05_UntouchedControlAddPlanTaskNeverCalled(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 99}
	pc := &fakePlanClient{}

	msg := editSavedMsg{
		name:    "new task",
		planDay: "",
	}
	cmd := ExportCreateTaskCmdWithPlan(tc, pc, msg)
	cmd()

	if pc.addTaskReq != nil {
		t.Error("AddPlanTask should not be called when planDay is empty")
	}
}

// CT-08: CreateTask fails ⇒ AddPlanTask never called.
func TestCT08_CreateTaskFailsAddPlanTaskNeverCalled(t *testing.T) {
	tc := &failingTaskClient{}
	pc := &fakePlanClient{}

	msg := editSavedMsg{
		name:    "new task",
		planDay: "2026-07-20",
	}
	cmd := ExportCreateTaskCmdWithPlan(tc, pc, msg)
	result := cmd()

	rm, ok := result.(refreshedMsg)
	if !ok {
		t.Fatalf("expected refreshedMsg, got %T", result)
	}
	if rm.err == nil {
		t.Error("expected err in refreshedMsg when CreateTask fails")
	}
	if pc.addTaskReq != nil {
		t.Error("AddPlanTask should not be called when CreateTask fails")
	}
}

// CT-09: AddPlanTask fails ⇒ task stays created, partial-failure reported,
// and the tree is still refreshed so the created task is visible.
func TestCT09_AddPlanTaskFailsTaskStaysCreated(t *testing.T) {
	taskID := int64(42)
	tc := &fakeTaskClient{
		createTaskID: taskID,
		// The refresh after the partial failure lists the created task.
		listTasksResp: []*taskv1.Task{{Id: taskID, Name: "new task"}},
	}
	pc := &fakePlanClient{
		mutateErr: connect.NewError(connect.CodeFailedPrecondition, errors.New("already on plan")),
	}

	msg := editSavedMsg{
		name:    "new task",
		planDay: "2026-07-20",
	}
	cmd := ExportCreateTaskCmdWithPlan(tc, pc, msg)
	result := cmd()

	rm, ok := result.(refreshedMsg)
	if !ok {
		t.Fatalf("expected refreshedMsg, got %T", result)
	}
	if rm.err != nil {
		t.Errorf("expected no hard err when AddPlanTask fails (tree should still refresh), got %v", rm.err)
	}
	if rm.partialErr == nil {
		t.Fatal("expected partialErr in refreshedMsg when AddPlanTask fails")
	}
	if !strings.Contains(rm.partialErr.Error(), "plan") {
		t.Errorf("partialErr should mention plan, got %q", rm.partialErr.Error())
	}
	if rm.highlightID != taskID {
		t.Errorf("highlightID: want %d, got %d", taskID, rm.highlightID)
	}
	if len(rm.tree) != 1 || rm.tree[0].Task.Id != taskID {
		t.Errorf("tree should contain the created task, got %v", rm.tree)
	}
}

// Create-with-plan must also refresh the Plan tab's scheduled-days markers,
// not just the task tree — otherwise the newly scheduled day doesn't show up
// until some unrelated action happens to refresh it.
func TestHandleEditSaved_WithPlanDay_AlsoRefreshesScheduledDays(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := newModel(tc, pc, "", config.PomodoroConfig{}, config.PlanConfig{}, false, nil)

	msg := editSavedMsg{name: "new task", planDay: "2026-07-20"}
	_, cmd := m.handleEditSaved(msg)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg when a plan day was chosen, got %T", cmd())
	}
	if len(batch) != 2 {
		t.Fatalf("expected 2 batched commands, got %d", len(batch))
	}

	sawRefresh := false
	for _, sub := range batch {
		if _, ok := sub().(refreshedMsg); ok {
			sawRefresh = true
		}
	}
	if !sawRefresh {
		t.Error("expected one batched command to produce refreshedMsg (the task-tree refresh)")
	}
	if pc.scheduledDaysReq == nil {
		t.Error("expected ListScheduledDays to be called after create-with-plan")
	}
}

// Create without a plan day is unaffected: no scheduled-days refresh, and the
// result is the plain (non-batched) command it always was.
func TestHandleEditSaved_WithoutPlanDay_NoScheduledDaysRefresh(t *testing.T) {
	tc := &fakeTaskClient{createTaskID: 42}
	pc := &fakePlanClient{}
	m := newModel(tc, pc, "", config.PomodoroConfig{}, config.PlanConfig{}, false, nil)

	msg := editSavedMsg{name: "new task"}
	_, cmd := m.handleEditSaved(msg)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	if _, ok := cmd().(tea.BatchMsg); ok {
		t.Error("expected a single (non-batched) cmd when no plan day was chosen")
	}
	if pc.scheduledDaysReq != nil {
		t.Error("ListScheduledDays should not be called when no plan day was chosen")
	}
}
