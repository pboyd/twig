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
	"time"

	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/twig/services/twig/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/services/twig/gen/task/v1/taskv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fakeTaskService is an in-memory implementation of TaskServiceHandler for testing.
type fakeTaskService struct {
	taskv1connect.UnimplementedTaskServiceHandler
	mu     sync.Mutex
	tasks  map[int64]*taskv1.Task
	nextID int64
}

func newFakeTaskService() *fakeTaskService {
	return &fakeTaskService{
		tasks:  make(map[int64]*taskv1.Task),
		nextID: 1,
	}
}

func (s *fakeTaskService) CreateTask(_ context.Context, req *connect.Request[taskv1.CreateTaskRequest]) (*connect.Response[taskv1.CreateTaskResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	if req.Msg.ParentId != nil {
		parent, ok := s.tasks[*req.Msg.ParentId]
		if !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent task not found"))
		}
		if parent.CompletedAt != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("Task %d is already crossed off — no new sub-tasks for a finished job.", *req.Msg.ParentId))
		}
	}

	id := s.nextID
	s.nextID++
	task := &taskv1.Task{
		Id:          id,
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
		Due:         req.Msg.Due,
		ParentId:    req.Msg.ParentId,
	}
	s.tasks[id] = task
	return connect.NewResponse(&taskv1.CreateTaskResponse{Task: task}), nil
}

func (s *fakeTaskService) GetTask(_ context.Context, req *connect.Request[taskv1.GetTaskRequest]) (*connect.Response[taskv1.GetTaskResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found"))
	}
	return connect.NewResponse(&taskv1.GetTaskResponse{Task: task}), nil
}

func (s *fakeTaskService) ListTasks(_ context.Context, req *connect.Request[taskv1.ListTasksRequest]) (*connect.Response[taskv1.ListTasksResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks := make([]*taskv1.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	return connect.NewResponse(&taskv1.ListTasksResponse{Tasks: tasks}), nil
}

func (s *fakeTaskService) UpdateTask(_ context.Context, req *connect.Request[taskv1.UpdateTaskRequest]) (*connect.Response[taskv1.UpdateTaskResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found"))
	}

	if req.Msg.ParentId != nil {
		// Check for cycle: walk up from the proposed parent to see if we hit req.Msg.Id
		pid := *req.Msg.ParentId
		for pid != 0 {
			if pid == req.Msg.Id {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent_id would create a cycle"))
			}
			p, ok := s.tasks[pid]
			if !ok {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent task not found"))
			}
			if p.ParentId == nil {
				break
			}
			pid = *p.ParentId
		}
		if parent := s.tasks[*req.Msg.ParentId]; parent != nil && parent.CompletedAt != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("Task %d is already crossed off — nothing moves under a finished job.", *req.Msg.ParentId))
		}
	}

	task.Name = req.Msg.Name
	task.Description = req.Msg.Description
	task.Due = req.Msg.Due
	task.ParentId = req.Msg.ParentId
	return connect.NewResponse(&taskv1.UpdateTaskResponse{Task: task}), nil
}

func (s *fakeTaskService) DeleteTask(_ context.Context, req *connect.Request[taskv1.DeleteTaskRequest]) (*connect.Response[taskv1.DeleteTaskResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[req.Msg.Id]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found"))
	}

	// Cascade delete children
	s.deleteSubtree(req.Msg.Id)
	return connect.NewResponse(&taskv1.DeleteTaskResponse{}), nil
}

func (s *fakeTaskService) CompleteTask(_ context.Context, req *connect.Request[taskv1.CompleteTaskRequest]) (*connect.Response[taskv1.CompleteTaskResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found"))
	}

	// Check for incomplete descendants.
	for _, t := range s.tasks {
		if t.CompletedAt == nil && s.isDescendant(req.Msg.Id, t.Id) {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("Whoa there — sub-tasks %d still need doing first.", t.Id))
		}
	}

	if task.CompletedAt == nil {
		now := timestamppb.New(time.Now())
		task.CompletedAt = now
	}
	return connect.NewResponse(&taskv1.CompleteTaskResponse{Task: task}), nil
}

func (s *fakeTaskService) UncompleteTask(_ context.Context, req *connect.Request[taskv1.UncompleteTaskRequest]) (*connect.Response[taskv1.UncompleteTaskResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found"))
	}
	task.CompletedAt = nil
	return connect.NewResponse(&taskv1.UncompleteTaskResponse{Task: task}), nil
}

// isDescendant returns true if candidateID is a descendant of ancestorID. Caller must hold mu.
func (s *fakeTaskService) isDescendant(ancestorID, candidateID int64) bool {
	current := candidateID
	for {
		t, ok := s.tasks[current]
		if !ok || t.ParentId == nil {
			return false
		}
		if *t.ParentId == ancestorID {
			return true
		}
		current = *t.ParentId
	}
}

// deleteSubtree removes a task and all its descendants. Caller must hold mu.
func (s *fakeTaskService) deleteSubtree(id int64) {
	delete(s.tasks, id)
	for _, t := range s.tasks {
		if t.ParentId != nil && *t.ParentId == id {
			s.deleteSubtree(t.Id)
		}
	}
}

// testHarness starts an httptest server backed by fakeTaskService and returns
// a client pointed at it.
type testHarness struct {
	svc    *fakeTaskService
	server *httptest.Server
	client taskv1connect.TaskServiceClient
	addr   string
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	svc := newFakeTaskService()
	mux := http.NewServeMux()
	path, handler := taskv1connect.NewTaskServiceHandler(svc)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := taskv1connect.NewTaskServiceClient(srv.Client(), srv.URL)
	return &testHarness{
		svc:    svc,
		server: srv,
		client: client,
		addr:   srv.URL,
	}
}

// runCmd runs a CLI subcommand function and captures stdout/stderr.
func runCmd(fn func(taskv1connect.TaskServiceClient, string, []string) int, h *testHarness, args []string) (stdout, stderr string, code int) {
	oldOut := os.Stdout
	oldErr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	code = fn(h.client, h.addr, args)

	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)
	return bufOut.String(), bufErr.String(), code
}

// --- Tests for add (US1) ---

func TestAddNameOnly(t *testing.T) {
	h := newTestHarness(t)
	stdout, stderr, code := runCmd(runAdd, h, []string{"my task"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "created task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestAddWithDue(t *testing.T) {
	h := newTestHarness(t)
	stdout, stderr, code := runCmd(runAdd, h, []string{"--due", "2026-06-01T17:00:00Z", "my task"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "created task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestAddWithParent(t *testing.T) {
	h := newTestHarness(t)
	// Create parent first
	_, _, code := runCmd(runAdd, h, []string{"parent task"})
	if code != 0 {
		t.Fatal("failed to create parent")
	}
	stdout, stderr, code := runCmd(runAdd, h, []string{"--parent", "1", "child task"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "created task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestAddMissingName(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runAdd, h, []string{})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "usage") {
		t.Errorf("expected usage error, got: %s", stderr)
	}
}

func TestAddUnknownParent(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runAdd, h, []string{"--parent", "999", "task"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestAddMalformedDue(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runAdd, h, []string{"--due", "not-a-date", "task"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "--due") {
		t.Errorf("expected --due error, got: %s", stderr)
	}
}

func TestAddMalformedParent(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runAdd, h, []string{"--parent", "abc", "task"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "integer") {
		t.Errorf("expected integer error, got: %s", stderr)
	}
}

// --- Tests for list (US2) ---

func TestListEmpty(t *testing.T) {
	h := newTestHarness(t)
	stdout, stderr, code := runCmd(runList, h, []string{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "no tasks") {
		t.Errorf("expected 'no tasks', got: %s", stdout)
	}
}

func TestListMultiLevel(t *testing.T) {
	h := newTestHarness(t)
	// Build: root1 -> child1 -> grandchild, root2
	runCmd(runAdd, h, []string{"root1"})                       // id=1
	runCmd(runAdd, h, []string{"root2"})                       // id=2
	runCmd(runAdd, h, []string{"--parent", "1", "child1"})     // id=3
	runCmd(runAdd, h, []string{"--parent", "3", "grandchild"}) // id=4

	stdout, stderr, code := runCmd(runList, h, []string{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	// Roots should appear
	if !strings.Contains(stdout, "root1") || !strings.Contains(stdout, "root2") {
		t.Errorf("expected both roots in output: %s", stdout)
	}
	// Children should appear with connectors
	if !strings.Contains(stdout, "child1") || !strings.Contains(stdout, "grandchild") {
		t.Errorf("expected children in output: %s", stdout)
	}
	// tree connector glyphs
	if !strings.Contains(stdout, "──") {
		t.Errorf("expected tree connector glyphs in output: %s", stdout)
	}
}

func TestListNoDueDate(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"task without deadline"})
	stdout, stderr, code := runCmd(runList, h, []string{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	// Should not show "(due ...)" for this task
	if strings.Contains(stdout, "(due ") {
		t.Errorf("unexpected due date in output: %s", stdout)
	}
}

func TestListSiblingsAscendingOrder(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"alpha"}) // id=1
	runCmd(runAdd, h, []string{"beta"})  // id=2
	runCmd(runAdd, h, []string{"gamma"}) // id=3

	stdout, _, _ := runCmd(runList, h, []string{})
	idx1 := strings.Index(stdout, "alpha")
	idx2 := strings.Index(stdout, "beta")
	idx3 := strings.Index(stdout, "gamma")
	if idx1 > idx2 || idx2 > idx3 {
		t.Errorf("expected ascending id order, got:\n%s", stdout)
	}
}

// --- Tests for mod (US3) ---

func TestModRename(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"original"})
	stdout, stderr, code := runCmd(runMod, h, []string{"1", "renamed"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "updated task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestModChangeDue(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"task"})
	_, stderr, code := runCmd(runMod, h, []string{"1", "--due", "2026-07-01"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
}

func TestModReparent(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"parent"}) // id=1
	runCmd(runAdd, h, []string{"child"})  // id=2
	_, stderr, code := runCmd(runMod, h, []string{"2", "--parent", "1"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	_ = stderr
}

func TestModUnflaggedFieldsPreserved(t *testing.T) {
	h := newTestHarness(t)
	// Add with due date
	runCmd(runAdd, h, []string{"--due", "2026-06-01T00:00:00Z", "task"})
	// Mod only the name — due should be preserved
	runCmd(runMod, h, []string{"1", "renamed"})
	// Check list shows due date
	stdout, _, _ := runCmd(runList, h, []string{})
	if !strings.Contains(stdout, "due") {
		t.Errorf("expected due date preserved after mod: %s", stdout)
	}
}

func TestModUnknownID(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runMod, h, []string{"999", "new name"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestModSelfAncestorCycle(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"task"}) // id=1
	_, stderr, code := runCmd(runMod, h, []string{"1", "--parent", "1"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "cycle") {
		t.Errorf("expected cycle error, got: %s", stderr)
	}
}

// --- Tests for rm (US4) ---

func TestRmLeaf(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"task"})
	stdout, stderr, code := runCmd(runRm, h, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "deleted task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestRmParentCascades(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"parent"})                 // id=1
	runCmd(runAdd, h, []string{"--parent", "1", "child"}) // id=2
	_, _, code := runCmd(runRm, h, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0 removing parent")
	}
	// Both should be gone
	stdout, _, _ := runCmd(runList, h, []string{})
	if strings.Contains(stdout, "parent") || strings.Contains(stdout, "child") {
		t.Errorf("expected both deleted, got: %s", stdout)
	}
}

func TestRmUnknownID(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runRm, h, []string{"999"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestRmMalformedID(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runRm, h, []string{"abc"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "integer") {
		t.Errorf("expected integer error, got: %s", stderr)
	}
}

// --- Tests for complete (US1) ---

func TestCompleteLeaf(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"leaf task"}) // id=1
	stdout, stderr, code := runCmd(runComplete, h, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "completed task 1") {
		t.Errorf("expected 'completed task 1', got: %s", stdout)
	}
}

func TestCompleteIdempotent(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"leaf"}) // id=1
	runCmd(runComplete, h, []string{"1"})
	// Small sleep so the second call timestamp is definitely after the first.
	time.Sleep(5 * time.Millisecond)
	stdout, stderr, code := runCmd(runComplete, h, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0 on re-complete, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "already complete") {
		t.Errorf("expected 'already complete' message, got: %s", stdout)
	}
}

func TestCompleteMalformedID(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runComplete, h, []string{"abc"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "integer") {
		t.Errorf("expected integer error, got: %s", stderr)
	}
}

func TestCompleteNotFound(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runComplete, h, []string{"999"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestCompleteBlockedByIncompleteDescendant(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"parent"})                 // id=1
	runCmd(runAdd, h, []string{"--parent", "1", "child"}) // id=2
	_, stderr, code := runCmd(runComplete, h, []string{"1"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "still need doing first") {
		t.Errorf("expected 'still need doing first' message, got: %s", stderr)
	}
}

// --- Tests for list filter flags (US3) ---

func TestListFlagsMutuallyExclusive(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runList, h, []string{"--completed", "--all"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("expected mutually exclusive message, got: %s", stderr)
	}
}

func TestListDefaultHidesCompleted(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"done"})    // id=1
	runCmd(runAdd, h, []string{"pending"}) // id=2
	runCmd(runComplete, h, []string{"1"})

	stdout, stderr, code := runCmd(runList, h, []string{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if strings.Contains(stdout, "done") {
		t.Errorf("completed task should not appear in default list: %s", stdout)
	}
	if !strings.Contains(stdout, "pending") {
		t.Errorf("incomplete task should appear in default list: %s", stdout)
	}
}

func TestListCompletedFlag(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"done"})    // id=1
	runCmd(runAdd, h, []string{"pending"}) // id=2
	runCmd(runComplete, h, []string{"1"})

	stdout, stderr, code := runCmd(runList, h, []string{"--completed"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "done") {
		t.Errorf("expected completed task in --completed list: %s", stdout)
	}
	if strings.Contains(stdout, "pending") {
		t.Errorf("incomplete task should not appear in --completed list: %s", stdout)
	}
}

func TestListAllFlag(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"done"})    // id=1
	runCmd(runAdd, h, []string{"pending"}) // id=2
	runCmd(runComplete, h, []string{"1"})

	stdout, stderr, code := runCmd(runList, h, []string{"--all"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "done") || !strings.Contains(stdout, "pending") {
		t.Errorf("expected both tasks in --all list: %s", stdout)
	}
}

func TestListNoCheckboxPrefix(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"incomplete"}) // id=1
	runCmd(runAdd, h, []string{"complete"})   // id=2
	runCmd(runComplete, h, []string{"2"})

	stdout, _, _ := runCmd(runList, h, []string{"--all"})
	if strings.Contains(stdout, "[ ]") || strings.Contains(stdout, "[x]") {
		t.Errorf("checkbox prefixes must not appear in output: %s", stdout)
	}
	// Tasks should still appear with their id prefix
	if !strings.Contains(stdout, "[1]") || !strings.Contains(stdout, "[2]") {
		t.Errorf("expected task id prefixes in output: %s", stdout)
	}
}

func TestCompleteParentBlockedCLI(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"parent"}) // id=1
	runCmd(runComplete, h, []string{"1"})
	_, stderr, code := runCmd(runAdd, h, []string{"--parent", "1", "late child"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "already crossed off") {
		t.Errorf("expected 'already crossed off' message, got: %s", stderr)
	}
}

func TestModCompleteParentBlockedCLI(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"parent"}) // id=1
	runCmd(runAdd, h, []string{"orphan"}) // id=2
	runCmd(runComplete, h, []string{"1"})
	_, stderr, code := runCmd(runMod, h, []string{"2", "--parent", "1"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "already crossed off") {
		t.Errorf("expected 'already crossed off' message, got: %s", stderr)
	}
}

// --- T004: runMod arg-parsing table tests (US1) ---

func TestModArgParsing(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantCode    int
		wantErrMsg  string
		checkUpdate func(t *testing.T, h *testHarness)
	}{
		{
			name:       "id only — nothing to update",
			args:       []string{"1"},
			wantCode:   1,
			wantErrMsg: "nothing to update",
		},
		{
			name:     "name only — name change",
			args:     []string{"1", "new name"},
			wantCode: 0,
			checkUpdate: func(t *testing.T, h *testHarness) {
				h.svc.mu.Lock()
				defer h.svc.mu.Unlock()
				if h.svc.tasks[1].Name != "new name" {
					t.Errorf("name not updated: %q", h.svc.tasks[1].Name)
				}
			},
		},
		{
			name:     "parent flag only — parent change, name preserved",
			args:     []string{"1", "--parent", "2"},
			wantCode: 0,
			checkUpdate: func(t *testing.T, h *testHarness) {
				h.svc.mu.Lock()
				defer h.svc.mu.Unlock()
				task := h.svc.tasks[1]
				if task.Name != "original" {
					t.Errorf("name should be preserved, got %q", task.Name)
				}
				if task.ParentId == nil || *task.ParentId != 2 {
					t.Errorf("parent not updated")
				}
			},
		},
		{
			name:     "parent flag before name — both updated",
			args:     []string{"1", "--parent", "2", "updated"},
			wantCode: 0,
			checkUpdate: func(t *testing.T, h *testHarness) {
				h.svc.mu.Lock()
				defer h.svc.mu.Unlock()
				task := h.svc.tasks[1]
				if task.Name != "updated" {
					t.Errorf("name not updated: %q", task.Name)
				}
				if task.ParentId == nil || *task.ParentId != 2 {
					t.Errorf("parent not updated")
				}
			},
		},
		{
			name:     "name before parent flag — both updated",
			args:     []string{"1", "updated", "--parent", "2"},
			wantCode: 0,
			checkUpdate: func(t *testing.T, h *testHarness) {
				h.svc.mu.Lock()
				defer h.svc.mu.Unlock()
				task := h.svc.tasks[1]
				if task.Name != "updated" {
					t.Errorf("name not updated: %q", task.Name)
				}
				if task.ParentId == nil || *task.ParentId != 2 {
					t.Errorf("parent not updated")
				}
			},
		},
		{
			name:       "empty name — rejected",
			args:       []string{"1", ""},
			wantCode:   1,
			wantErrMsg: "empty",
		},
		{
			name:       "extra positionals — usage error",
			args:       []string{"1", "a", "b", "c"},
			wantCode:   1,
			wantErrMsg: "usage",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHarness(t)
			runCmd(runAdd, h, []string{"original"}) // id=1
			runCmd(runAdd, h, []string{"second"})   // id=2

			_, stderr, code := runCmd(runMod, h, tc.args)
			if code != tc.wantCode {
				t.Fatalf("exit code %d, want %d; stderr: %s", code, tc.wantCode, stderr)
			}
			if tc.wantErrMsg != "" && !strings.Contains(stderr, tc.wantErrMsg) {
				t.Errorf("expected %q in stderr, got: %s", tc.wantErrMsg, stderr)
			}
			if tc.checkUpdate != nil && code == 0 {
				tc.checkUpdate(t, h)
			}
		})
	}
}

// --- Tests for uncomplete ---

func TestUncompleteLeaf(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"leaf task"}) // id=1
	runCmd(runComplete, h, []string{"1"})
	stdout, stderr, code := runCmd(runUncomplete, h, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "back on your list") {
		t.Errorf("expected 'back on your list', got: %s", stdout)
	}
}

func TestUncompleteIdempotent(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h, []string{"leaf task"}) // id=1
	stdout, stderr, code := runCmd(runUncomplete, h, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0 on already-incomplete, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "already incomplete") {
		t.Errorf("expected 'already incomplete', got: %s", stdout)
	}
}

func TestUncompleteMalformedID(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runUncomplete, h, []string{"abc"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "integer") {
		t.Errorf("expected integer error, got: %s", stderr)
	}
}

func TestUncompleteNotFound(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runUncomplete, h, []string{"999"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}
