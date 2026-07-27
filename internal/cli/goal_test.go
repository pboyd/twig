package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	goalv1connect "github.com/pboyd/twig/api/gen/goal/v1/goalv1connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/goal"
)

// fakeGoalService is an in-memory GoalServiceHandler for testing.
type fakeGoalService struct {
	goalv1connect.UnimplementedGoalServiceHandler
	mu     sync.Mutex
	goals  map[int64]*goalv1.Goal
	nextID int64
}

func newFakeGoalService() *fakeGoalService {
	return &fakeGoalService{goals: make(map[int64]*goalv1.Goal), nextID: 1}
}

func (s *fakeGoalService) CreateGoal(_ context.Context, req *connect.Request[goalv1.CreateGoalRequest]) (*connect.Response[goalv1.CreateGoalResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	id := s.nextID
	s.nextID++
	g := &goalv1.Goal{
		Id:          id,
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
		Due:         req.Msg.Due,
		State:       goalv1.GoalState_GOAL_STATE_INCUBATING,
		Position:    int64(len(s.goals)),
	}
	s.goals[id] = g
	return connect.NewResponse(&goalv1.CreateGoalResponse{Goal: g}), nil
}

func (s *fakeGoalService) GetGoal(_ context.Context, req *connect.Request[goalv1.GetGoalRequest]) (*connect.Response[goalv1.GetGoalResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.goals[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("goal not found"))
	}
	return connect.NewResponse(&goalv1.GetGoalResponse{Goal: g}), nil
}

func (s *fakeGoalService) ListGoals(_ context.Context, _ *connect.Request[goalv1.ListGoalsRequest]) (*connect.Response[goalv1.ListGoalsResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	goals := make([]*goalv1.Goal, 0, len(s.goals))
	for _, g := range s.goals {
		goals = append(goals, g)
	}
	return connect.NewResponse(&goalv1.ListGoalsResponse{Goals: goals}), nil
}

func (s *fakeGoalService) UpdateGoal(_ context.Context, req *connect.Request[goalv1.UpdateGoalRequest]) (*connect.Response[goalv1.UpdateGoalResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.goals[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("goal not found"))
	}
	g.Name = req.Msg.Name
	g.Description = req.Msg.Description
	g.Due = req.Msg.Due
	return connect.NewResponse(&goalv1.UpdateGoalResponse{Goal: g}), nil
}

func (s *fakeGoalService) SetGoalState(_ context.Context, req *connect.Request[goalv1.SetGoalStateRequest]) (*connect.Response[goalv1.SetGoalStateResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.goals[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("goal not found"))
	}
	g.State = req.Msg.State
	return connect.NewResponse(&goalv1.SetGoalStateResponse{Goal: g}), nil
}

func (s *fakeGoalService) DeleteGoal(_ context.Context, req *connect.Request[goalv1.DeleteGoalRequest]) (*connect.Response[goalv1.DeleteGoalResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.goals[req.Msg.Id]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("goal not found"))
	}
	delete(s.goals, req.Msg.Id)
	return connect.NewResponse(&goalv1.DeleteGoalResponse{}), nil
}

type goalTestHarness struct {
	goalSvc  *fakeGoalService
	taskSvc  *fakeTaskService
	server   *httptest.Server
	goalClient goalv1connect.GoalServiceClient
	taskClient taskv1connect.TaskServiceClient
	addr     string
}

func newGoalTestHarness(t *testing.T) *goalTestHarness {
	t.Helper()
	goalSvc := newFakeGoalService()
	taskSvc := newFakeTaskService()

	mux := http.NewServeMux()
	gPath, gHandler := goalv1connect.NewGoalServiceHandler(goalSvc)
	tPath, tHandler := taskv1connect.NewTaskServiceHandler(taskSvc)
	mux.Handle(gPath, gHandler)
	mux.Handle(tPath, tHandler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &goalTestHarness{
		goalSvc:    goalSvc,
		taskSvc:    taskSvc,
		server:     srv,
		goalClient: goalv1connect.NewGoalServiceClient(srv.Client(), srv.URL),
		taskClient: taskv1connect.NewTaskServiceClient(srv.Client(), srv.URL),
		addr:       srv.URL,
	}
}

// captureOutput runs fn, capturing and restoring os.Stdout and os.Stderr.
func captureOutput(fn func()) (stdout, stderr string) {
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	var bOut, bErr bytes.Buffer
	bOut.ReadFrom(rOut)
	bErr.ReadFrom(rErr)
	return bOut.String(), bErr.String()
}

func TestGoalList_Empty(t *testing.T) {
	h := newGoalTestHarness(t)
	var code int
	stdout, _ := captureOutput(func() {
		code = runGoalList(h.goalClient, h.addr, nil)
	})
	if code != 0 {
		t.Fatalf("expected exit 0")
	}
	if !strings.Contains(stdout, "No goals on the horizon") {
		t.Errorf("empty list: want playful message, got: %s", stdout)
	}
}

func TestGoalList_GroupsAndOrder(t *testing.T) {
	h := newGoalTestHarness(t)
	// Add committed and incubating goals.
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "Incubating goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0}
	h.goalSvc.goals[2] = &goalv1.Goal{Id: 2, Name: "In progress goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0}
	h.goalSvc.nextID = 3
	h.goalSvc.mu.Unlock()

	var code int
	stdout, _ := captureOutput(func() {
		code = runGoalList(h.goalClient, h.addr, nil)
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	inProgressIdx := strings.Index(stdout, "In Progress")
	incubatingIdx := strings.Index(stdout, "Incubating")
	if inProgressIdx < 0 || incubatingIdx < 0 {
		t.Fatalf("expected both group headers, got: %s", stdout)
	}
	if inProgressIdx >= incubatingIdx {
		t.Errorf("In Progress (%d) should appear before Incubating (%d)", inProgressIdx, incubatingIdx)
	}
}

func TestGoalList_HidesCompletedAndArchivedByDefault(t *testing.T) {
	h := newGoalTestHarness(t)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "active", State: goalv1.GoalState_GOAL_STATE_INCUBATING}
	h.goalSvc.goals[2] = &goalv1.Goal{Id: 2, Name: "done", State: goalv1.GoalState_GOAL_STATE_COMPLETED}
	h.goalSvc.goals[3] = &goalv1.Goal{Id: 3, Name: "archived", State: goalv1.GoalState_GOAL_STATE_ARCHIVED}
	h.goalSvc.nextID = 4
	h.goalSvc.mu.Unlock()

	stdout, _ := captureOutput(func() {
		runGoalList(h.goalClient, h.addr, nil)
	})
	if strings.Contains(stdout, "done") {
		t.Error("completed goal should be hidden by default")
	}
	if strings.Contains(stdout, "archived") {
		t.Error("archived goal should be hidden by default")
	}
}

func TestGoalList_ShowsHoldByDefault(t *testing.T) {
	h := newGoalTestHarness(t)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "incubating goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING}
	h.goalSvc.goals[2] = &goalv1.Goal{Id: 2, Name: "parked goal", State: goalv1.GoalState_GOAL_STATE_HOLD}
	h.goalSvc.nextID = 3
	h.goalSvc.mu.Unlock()

	stdout, _ := captureOutput(func() {
		runGoalList(h.goalClient, h.addr, nil)
	})
	if !strings.Contains(stdout, "parked goal") {
		t.Errorf("hold goal should be visible by default, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Hold") {
		t.Errorf("expected 'Hold' group header, got: %s", stdout)
	}
	incubatingIdx := strings.Index(stdout, "Incubating")
	holdIdx := strings.Index(stdout, "Hold")
	if incubatingIdx < 0 || holdIdx < 0 || incubatingIdx >= holdIdx {
		t.Errorf("expected Incubating (%d) before Hold (%d)", incubatingIdx, holdIdx)
	}
}

func TestGoalList_ShowAllFlag(t *testing.T) {
	h := newGoalTestHarness(t)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "done", State: goalv1.GoalState_GOAL_STATE_COMPLETED}
	h.goalSvc.nextID = 2
	h.goalSvc.mu.Unlock()

	stdout, _ := captureOutput(func() {
		runGoalList(h.goalClient, h.addr, []string{"--all"})
	})
	if !strings.Contains(stdout, "done") {
		t.Errorf("--all: completed goal should be visible, got: %s", stdout)
	}
}

func TestGoalAdd(t *testing.T) {
	h := newGoalTestHarness(t)
	var code int
	stdout, _ := captureOutput(func() {
		code = runGoalAdd(h.goalClient, h.addr, []string{"Learn sailing"})
	})
	if code != 0 {
		t.Fatalf("expected exit 0")
	}
	if !strings.Contains(stdout, "created goal") {
		t.Errorf("expected 'created goal', got: %s", stdout)
	}
	h.goalSvc.mu.Lock()
	defer h.goalSvc.mu.Unlock()
	if len(h.goalSvc.goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(h.goalSvc.goals))
	}
	if h.goalSvc.goals[1].Name != "Learn sailing" {
		t.Errorf("goal name: want 'Learn sailing', got %q", h.goalSvc.goals[1].Name)
	}
}

func TestGoalAdd_EmptyNameRejected(t *testing.T) {
	h := newGoalTestHarness(t)
	var code int
	_, stderr := captureOutput(func() {
		code = runGoalAdd(h.goalClient, h.addr, []string{""})
	})
	if code == 0 {
		t.Fatal("expected non-zero exit for empty name")
	}
	if !strings.Contains(stderr, "name") {
		t.Errorf("expected name error, got: %s", stderr)
	}
}

func TestGoalMod(t *testing.T) {
	h := newGoalTestHarness(t)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "Old name", State: goalv1.GoalState_GOAL_STATE_INCUBATING}
	h.goalSvc.nextID = 2
	h.goalSvc.mu.Unlock()

	var code int
	stdout, _ := captureOutput(func() {
		code = runGoalMod(h.goalClient, h.addr, []string{"1", "New name"})
	})
	if code != 0 {
		t.Fatalf("expected exit 0")
	}
	if !strings.Contains(stdout, "updated goal") {
		t.Errorf("expected 'updated goal', got: %s", stdout)
	}
	h.goalSvc.mu.Lock()
	defer h.goalSvc.mu.Unlock()
	if h.goalSvc.goals[1].Name != "New name" {
		t.Errorf("want 'New name', got %q", h.goalSvc.goals[1].Name)
	}
}

func TestGoalMod_NotFound(t *testing.T) {
	h := newGoalTestHarness(t)
	var code int
	_, stderr := captureOutput(func() {
		code = runGoalMod(h.goalClient, h.addr, []string{"42", "irrelevant"})
	})
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "42") || !strings.Contains(stderr, "twig:") {
		t.Errorf("expected playful not-found message mentioning id 42, got: %s", stderr)
	}
}

func TestGoalState(t *testing.T) {
	h := newGoalTestHarness(t)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING}
	h.goalSvc.nextID = 2
	h.goalSvc.mu.Unlock()

	var code int
	stdout, _ := captureOutput(func() {
		code = runGoalState(h.goalClient, h.addr, []string{"1", "in-progress"})
	})
	if code != 0 {
		t.Fatalf("expected exit 0")
	}
	if !strings.Contains(stdout, "in progress") {
		t.Errorf("expected 'in progress' in output, got: %s", stdout)
	}
	h.goalSvc.mu.Lock()
	defer h.goalSvc.mu.Unlock()
	if h.goalSvc.goals[1].State != goalv1.GoalState_GOAL_STATE_IN_PROGRESS {
		t.Errorf("state not updated")
	}
}

func TestGoalState_InvalidState(t *testing.T) {
	h := newGoalTestHarness(t)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING}
	h.goalSvc.nextID = 2
	h.goalSvc.mu.Unlock()

	var code int
	_, stderr := captureOutput(func() {
		code = runGoalState(h.goalClient, h.addr, []string{"1", "bogusstate"})
	})
	if code == 0 {
		t.Fatal("expected non-zero exit for invalid state")
	}
	// Error message should list valid states.
	for _, valid := range goal.StateNames() {
		if !strings.Contains(stderr, valid) {
			t.Errorf("expected valid state %q listed in error, got: %s", valid, stderr)
		}
	}
}

func TestGoalState_NotFound(t *testing.T) {
	h := newGoalTestHarness(t)
	var code int
	_, stderr := captureOutput(func() {
		code = runGoalState(h.goalClient, h.addr, []string{"99", "in-progress"})
	})
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "99") || !strings.Contains(stderr, "twig:") {
		t.Errorf("expected playful not-found message mentioning id 99, got: %s", stderr)
	}
}

func TestGoalRm(t *testing.T) {
	h := newGoalTestHarness(t)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING}
	h.goalSvc.nextID = 2
	h.goalSvc.mu.Unlock()

	var code int
	stdout, _ := captureOutput(func() {
		code = runGoalRm(h.goalClient, h.addr, []string{"1"})
	})
	if code != 0 {
		t.Fatalf("expected exit 0")
	}
	if !strings.Contains(stdout, "deleted goal") {
		t.Errorf("expected 'deleted goal', got: %s", stdout)
	}
	h.goalSvc.mu.Lock()
	defer h.goalSvc.mu.Unlock()
	if len(h.goalSvc.goals) != 0 {
		t.Error("goal not deleted")
	}
}

func TestGoalRm_NotFound(t *testing.T) {
	h := newGoalTestHarness(t)
	var code int
	_, stderr := captureOutput(func() {
		code = runGoalRm(h.goalClient, h.addr, []string{"42"})
	})
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "42") || !strings.Contains(stderr, "twig:") {
		t.Errorf("expected playful not-found message mentioning id 42, got: %s", stderr)
	}
}

func TestGoalShow_NoTasks(t *testing.T) {
	h := newGoalTestHarness(t)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: 1, Name: "Build a boat", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Description: "A small sailboat"}
	h.goalSvc.nextID = 2
	h.goalSvc.mu.Unlock()

	var code int
	stdout, _ := captureOutput(func() {
		code = runGoalShow(h.goalClient, h.taskClient, h.addr, []string{"1"})
	})
	if code != 0 {
		t.Fatalf("expected exit 0")
	}
	if !strings.Contains(stdout, "Build a boat") {
		t.Errorf("expected goal name in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "in progress") {
		t.Errorf("expected state in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "A small sailboat") {
		t.Errorf("expected description in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "No tasks attached yet") {
		t.Errorf("expected no-tasks copy, got: %s", stdout)
	}
}

func TestGoalShow_WithTasks(t *testing.T) {
	h := newGoalTestHarness(t)
	goalID := int64(1)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[1] = &goalv1.Goal{Id: goalID, Name: "Ship the feature", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	h.goalSvc.nextID = 2
	h.goalSvc.mu.Unlock()

	// Add a task linked to the goal.
	h.taskSvc.mu.Lock()
	h.taskSvc.tasks[10] = &taskv1.Task{Id: 10, Name: "Write tests", GoalId: &goalID}
	h.taskSvc.mu.Unlock()

	var code int
	stdout, _ := captureOutput(func() {
		code = runGoalShow(h.goalClient, h.taskClient, h.addr, []string{"1"})
	})
	if code != 0 {
		t.Fatalf("expected exit 0")
	}
	if !strings.Contains(stdout, "Write tests") {
		t.Errorf("expected linked task in output, got: %s", stdout)
	}
}

func TestGoalShow_NotFound(t *testing.T) {
	h := newGoalTestHarness(t)
	var code int
	_, stderr := captureOutput(func() {
		code = runGoalShow(h.goalClient, h.taskClient, h.addr, []string{"99"})
	})
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "99") || !strings.Contains(stderr, "twig:") {
		t.Errorf("expected playful not-found message mentioning id 99, got: %s", stderr)
	}
}

// TestTaskShow_WithGoal verifies that `twig task show <id>` prints task fields
// and a Goal: line when the task has a direct goal association.
func TestTaskShow_WithGoal(t *testing.T) {
	h := newGoalTestHarness(t)

	goalID := int64(1)
	h.goalSvc.mu.Lock()
	h.goalSvc.goals[goalID] = &goalv1.Goal{Id: goalID, Name: "Buy a new car", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	h.goalSvc.mu.Unlock()

	h.taskSvc.mu.Lock()
	h.taskSvc.tasks[1] = &taskv1.Task{Id: 1, Name: "Research insurance", GoalId: &goalID}
	h.taskSvc.mu.Unlock()

	var code int
	stdout, stderr := captureOutput(func() {
		code = runTaskShow(h.taskClient, h.goalClient, h.addr, []string{"1"})
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Research insurance") {
		t.Errorf("expected task name in output; got: %s", stdout)
	}
	if !strings.Contains(stdout, "Goal:") || !strings.Contains(stdout, "Buy a new car") {
		t.Errorf("expected 'Goal: Buy a new car' in output; got: %s", stdout)
	}
}

// TestTaskShow_NoGoal verifies that `twig task show <id>` omits the Goal line
// for tasks with no goal association.
func TestTaskShow_NoGoal(t *testing.T) {
	h := newGoalTestHarness(t)
	h.taskSvc.mu.Lock()
	h.taskSvc.tasks[1] = &taskv1.Task{Id: 1, Name: "Buy milk"}
	h.taskSvc.mu.Unlock()

	var code int
	stdout, stderr := captureOutput(func() {
		code = runTaskShow(h.taskClient, h.goalClient, h.addr, []string{"1"})
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Buy milk") {
		t.Errorf("expected task name in output; got: %s", stdout)
	}
	if strings.Contains(stdout, "Goal:") {
		t.Errorf("expected no Goal: line for task with no goal; got: %s", stdout)
	}
}

// TestTaskShow_NotFound verifies `twig task show <id>` returns a clear error for unknown IDs.
func TestTaskShow_NotFound(t *testing.T) {
	h := newGoalTestHarness(t)
	var code int
	_, stderr := captureOutput(func() {
		code = runTaskShow(h.taskClient, h.goalClient, h.addr, []string{"99"})
	})
	if code == 0 {
		t.Fatal("expected non-zero exit for unknown task")
	}
	if !strings.Contains(stderr, "99") {
		t.Errorf("expected error mentioning id 99; got: %s", stderr)
	}
}
