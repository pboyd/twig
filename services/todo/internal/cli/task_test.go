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
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
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
		if _, ok := s.tasks[*req.Msg.ParentId]; !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent task not found"))
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
func runCmd(fn func(taskv1connect.TaskServiceClient, []string) int, client taskv1connect.TaskServiceClient, args []string) (stdout, stderr string, code int) {
	oldOut := os.Stdout
	oldErr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	code = fn(client, args)

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
	stdout, stderr, code := runCmd(runAdd, h.client, []string{"my task"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "created task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestAddWithDue(t *testing.T) {
	h := newTestHarness(t)
	stdout, stderr, code := runCmd(runAdd, h.client, []string{"--due", "2026-06-01T17:00:00Z", "my task"})
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
	_, _, code := runCmd(runAdd, h.client, []string{"parent task"})
	if code != 0 {
		t.Fatal("failed to create parent")
	}
	stdout, stderr, code := runCmd(runAdd, h.client, []string{"--parent", "1", "child task"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "created task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestAddMissingName(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runAdd, h.client, []string{})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "usage") {
		t.Errorf("expected usage error, got: %s", stderr)
	}
}

func TestAddUnknownParent(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runAdd, h.client, []string{"--parent", "999", "task"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestAddMalformedDue(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runAdd, h.client, []string{"--due", "not-a-date", "task"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "--due") {
		t.Errorf("expected --due error, got: %s", stderr)
	}
}

func TestAddMalformedParent(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runAdd, h.client, []string{"--parent", "abc", "task"})
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
	stdout, stderr, code := runCmd(runList, h.client, []string{})
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
	runCmd(runAdd, h.client, []string{"root1"})                       // id=1
	runCmd(runAdd, h.client, []string{"root2"})                       // id=2
	runCmd(runAdd, h.client, []string{"--parent", "1", "child1"})     // id=3
	runCmd(runAdd, h.client, []string{"--parent", "3", "grandchild"}) // id=4

	stdout, stderr, code := runCmd(runList, h.client, []string{})
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
	runCmd(runAdd, h.client, []string{"task without deadline"})
	stdout, stderr, code := runCmd(runList, h.client, []string{})
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
	runCmd(runAdd, h.client, []string{"alpha"}) // id=1
	runCmd(runAdd, h.client, []string{"beta"})  // id=2
	runCmd(runAdd, h.client, []string{"gamma"}) // id=3

	stdout, _, _ := runCmd(runList, h.client, []string{})
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
	runCmd(runAdd, h.client, []string{"original"})
	stdout, stderr, code := runCmd(runMod, h.client, []string{"1", "renamed"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "updated task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestModChangeDue(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h.client, []string{"task"})
	_, stderr, code := runCmd(runMod, h.client, []string{"--due", "2026-07-01", "1", "task"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
}

func TestModReparent(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h.client, []string{"parent"}) // id=1
	runCmd(runAdd, h.client, []string{"child"})  // id=2
	_, stderr, code := runCmd(runMod, h.client, []string{"--parent", "1", "2", "child"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	_ = stderr
}

func TestModUnflaggedFieldsPreserved(t *testing.T) {
	h := newTestHarness(t)
	// Add with due date
	runCmd(runAdd, h.client, []string{"--due", "2026-06-01T00:00:00Z", "task"})
	// Mod only the name — due should be preserved
	runCmd(runMod, h.client, []string{"1", "renamed"})
	// Check list shows due date
	stdout, _, _ := runCmd(runList, h.client, []string{})
	if !strings.Contains(stdout, "due") {
		t.Errorf("expected due date preserved after mod: %s", stdout)
	}
}

func TestModUnknownID(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runMod, h.client, []string{"999", "new name"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestModSelfAncestorCycle(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h.client, []string{"task"}) // id=1
	_, stderr, code := runCmd(runMod, h.client, []string{"--parent", "1", "1", "task"})
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
	runCmd(runAdd, h.client, []string{"task"})
	stdout, stderr, code := runCmd(runRm, h.client, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "deleted task") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

func TestRmParentCascades(t *testing.T) {
	h := newTestHarness(t)
	runCmd(runAdd, h.client, []string{"parent"})                 // id=1
	runCmd(runAdd, h.client, []string{"--parent", "1", "child"}) // id=2
	_, _, code := runCmd(runRm, h.client, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0 removing parent")
	}
	// Both should be gone
	stdout, _, _ := runCmd(runList, h.client, []string{})
	if strings.Contains(stdout, "parent") || strings.Contains(stdout, "child") {
		t.Errorf("expected both deleted, got: %s", stdout)
	}
}

func TestRmUnknownID(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runRm, h.client, []string{"999"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestRmMalformedID(t *testing.T) {
	h := newTestHarness(t)
	_, stderr, code := runCmd(runRm, h.client, []string{"abc"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "integer") {
		t.Errorf("expected integer error, got: %s", stderr)
	}
}
