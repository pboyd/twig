package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
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
