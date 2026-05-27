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

	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	planv1connect "github.com/pboyd/todo/services/todo/gen/plan/v1/planv1connect"
)

// fakePlanService is an in-memory PlanServiceHandler for CLI tests.
type fakePlanService struct {
	planv1connect.UnimplementedPlanServiceHandler
	mu           sync.Mutex
	entries      map[string][]*planv1.PlanEntry // key: day
	nextIDPerDay map[string]int32
	// Captured request fields for assertion.
	lastListDay      string
	lastClearRequest *planv1.ClearPlanRequest
	lastMvRequest    *planv1.MovePlanEntryRequest
}

func newFakePlanService() *fakePlanService {
	return &fakePlanService{
		entries:      make(map[string][]*planv1.PlanEntry),
		nextIDPerDay: make(map[string]int32),
	}
}

func (s *fakePlanService) nextID(day string) int32 {
	s.nextIDPerDay[day]++
	return s.nextIDPerDay[day]
}

func (s *fakePlanService) ListPlanEntries(_ context.Context, req *connect.Request[planv1.ListPlanEntriesRequest]) (*connect.Response[planv1.ListPlanEntriesResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastListDay = req.Msg.Day
	entries := s.entries[req.Msg.Day]
	if entries == nil {
		entries = []*planv1.PlanEntry{}
	}
	return connect.NewResponse(&planv1.ListPlanEntriesResponse{Entries: entries}), nil
}

func (s *fakePlanService) AddPlanTask(_ context.Context, req *connect.Request[planv1.AddPlanTaskRequest]) (*connect.Response[planv1.AddPlanTaskResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Msg.TaskId == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found"))
	}
	e := &planv1.PlanEntry{
		Day:            req.Msg.Day,
		Id:             s.nextID(req.Msg.Day),
		TaskId:         req.Msg.TaskId,
		StartMinute:    req.Msg.StartMinute,
		DurationMinute: req.Msg.DurationMinute,
	}
	s.entries[req.Msg.Day] = append(s.entries[req.Msg.Day], e)
	return connect.NewResponse(&planv1.AddPlanTaskResponse{Entry: e}), nil
}

func (s *fakePlanService) AddPlanEvent(_ context.Context, req *connect.Request[planv1.AddPlanEventRequest]) (*connect.Response[planv1.AddPlanEventResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := &planv1.PlanEntry{
		Day:            req.Msg.Day,
		Id:             s.nextID(req.Msg.Day),
		Name:           req.Msg.Name,
		StartMinute:    req.Msg.StartMinute,
		DurationMinute: req.Msg.DurationMinute,
	}
	s.entries[req.Msg.Day] = append(s.entries[req.Msg.Day], e)
	return connect.NewResponse(&planv1.AddPlanEventResponse{Entry: e}), nil
}

func (s *fakePlanService) RemovePlanEntry(_ context.Context, req *connect.Request[planv1.RemovePlanEntryRequest]) (*connect.Response[planv1.RemovePlanEntryResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.entries[req.Msg.Day] {
		if e.Id == req.Msg.Id {
			s.entries[req.Msg.Day] = append(s.entries[req.Msg.Day][:i], s.entries[req.Msg.Day][i+1:]...)
			return connect.NewResponse(&planv1.RemovePlanEntryResponse{}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("entry not found"))
}

func (s *fakePlanService) RenamePlanEntry(_ context.Context, req *connect.Request[planv1.RenamePlanEntryRequest]) (*connect.Response[planv1.RenamePlanEntryResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries[req.Msg.Day] {
		if e.Id == req.Msg.Id {
			e.Name = req.Msg.Name
			return connect.NewResponse(&planv1.RenamePlanEntryResponse{Entry: e}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("entry not found"))
}

func (s *fakePlanService) MovePlanEntry(_ context.Context, req *connect.Request[planv1.MovePlanEntryRequest]) (*connect.Response[planv1.MovePlanEntryResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastMvRequest = req.Msg
	for _, e := range s.entries[req.Msg.Day] {
		if e.Id == req.Msg.Id {
			e.StartMinute = req.Msg.StartMinute
			if req.Msg.DurationMinute != 0 {
				e.DurationMinute = req.Msg.DurationMinute
			}
			return connect.NewResponse(&planv1.MovePlanEntryResponse{Entry: e}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("entry not found"))
}

func (s *fakePlanService) ClearPlan(_ context.Context, req *connect.Request[planv1.ClearPlanRequest]) (*connect.Response[planv1.ClearPlanResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastClearRequest = req.Msg
	return connect.NewResponse(&planv1.ClearPlanResponse{DeletedCount: 2, TrimmedStraddlingEntry: true}), nil
}

// planTestHarness wraps a fake plan service in an httptest server.
type planTestHarness struct {
	svc    *fakePlanService
	server *httptest.Server
	client planv1connect.PlanServiceClient
}

func newPlanTestHarness(t *testing.T) *planTestHarness {
	t.Helper()
	svc := newFakePlanService()
	mux := http.NewServeMux()
	path, h := planv1connect.NewPlanServiceHandler(svc)
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := planv1connect.NewPlanServiceClient(srv.Client(), srv.URL)
	return &planTestHarness{svc: svc, server: srv, client: client}
}

// runPlanCmd runs a plan subcommand function and captures stdout/stderr.
func runPlanCmd(fn func(planv1connect.PlanServiceClient, string, []string) int, client planv1connect.PlanServiceClient, day string, args []string) (stdout, stderr string, code int) {
	oldOut := os.Stdout
	oldErr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	code = fn(client, day, args)
	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr
	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)
	return bufOut.String(), bufErr.String(), code
}

func runPlanShowCmd(fn func(planv1connect.PlanServiceClient, string) int, client planv1connect.PlanServiceClient, day string) (stdout, stderr string, code int) {
	oldOut := os.Stdout
	oldErr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	code = fn(client, day)
	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr
	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)
	return bufOut.String(), bufErr.String(), code
}

// ---- T023: runPlanTask / runPlanEvent CLI tests ----

func TestRunPlanTask_HappyPath(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	stdout, stderr, code := runPlanCmd(runPlanTask, h.client, day, []string{"5", "9:00am"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "→ entry 1") {
		t.Errorf("expected '→ entry 1', got: %q", stdout)
	}
}

func TestRunPlanEvent_HappyPath(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	stdout, stderr, code := runPlanCmd(runPlanEvent, h.client, day, []string{"Lunch", "12:00pm", "45m"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "→ entry 1") {
		t.Errorf("expected '→ entry 1', got: %q", stdout)
	}
}

func TestRunPlanTask_ArgParseFailures(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"

	// non-integer task_id
	_, stderr, code := runPlanCmd(runPlanTask, h.client, day, []string{"abc", "9:00am"})
	if code == 0 {
		t.Error("expected non-zero exit for non-integer task_id")
	}
	if !strings.Contains(stderr, "integer") {
		t.Errorf("expected 'integer' in stderr, got: %q", stderr)
	}

	// malformed start
	_, _, code = runPlanCmd(runPlanTask, h.client, day, []string{"5", "not-a-time"})
	if code == 0 {
		t.Error("expected non-zero exit for malformed start")
	}

	// malformed duration
	_, _, code = runPlanCmd(runPlanTask, h.client, day, []string{"5", "9:00am", "bad"})
	if code == 0 {
		t.Error("expected non-zero exit for malformed duration")
	}
}

func TestRunPlanTask_NotFound(t *testing.T) {
	h := newPlanTestHarness(t)
	// task_id=0 triggers NotFound in our fake service.
	_, _, code := runPlanCmd(runPlanTask, h.client, "2026-05-21", []string{"0", "9:00am"})
	if code == 0 {
		t.Error("expected non-zero exit for NotFound task")
	}
}

func TestRunPlanTask_DateFlag(t *testing.T) {
	h := newPlanTestHarness(t)
	overrideDay := "2026-06-15"
	runPlanCmd(runPlanTask, h.client, overrideDay, []string{"5", "9:00am"})
	h.svc.mu.Lock()
	defer h.svc.mu.Unlock()
	if _, ok := h.svc.entries[overrideDay]; !ok {
		t.Errorf("entry not stored for day %s; stored days: %v", overrideDay, h.svc.entries)
	}
}

// ---- T029: runPlanShow CLI tests ----

func TestRunPlanShow_HappyPath(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	h.svc.mu.Lock()
	h.svc.entries[day] = []*planv1.PlanEntry{
		{Day: day, Id: 1, Name: "Work", StartMinute: 480, DurationMinute: 60},
		{Day: day, Id: 2, Name: "Lunch", StartMinute: 720, DurationMinute: 60},
	}
	h.svc.mu.Unlock()

	stdout, stderr, code := runPlanShowCmd(runPlanShow, h.client, day)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "08:00") {
		t.Errorf("expected 08:00 row, got:\n%s", stdout)
	}
}

func TestRunPlanShow_EmptyPlan(t *testing.T) {
	h := newPlanTestHarness(t)
	stdout, _, code := runPlanShowCmd(runPlanShow, h.client, "2026-05-21")
	if code != 0 {
		t.Error("expected exit 0 for empty plan")
	}
	// Empty day renders the default 08:00–17:00 calendar grid.
	if !strings.Contains(stdout, "08:00") {
		t.Errorf("expected calendar grid for empty plan, got: %q", stdout)
	}
}

func TestRunPlanShow_DateForwarded(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2099-12-25"
	runPlanShowCmd(runPlanShow, h.client, day)
	if h.svc.lastListDay != day {
		t.Errorf("ListPlanEntries called with day %q, want %q", h.svc.lastListDay, day)
	}
}

// ---- T039: runPlanRm / runPlanRename / runPlanMv CLI tests ----

func TestRunPlanRm_HappyPath(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	h.svc.mu.Lock()
	h.svc.entries[day] = []*planv1.PlanEntry{{Day: day, Id: 1, Name: "X", StartMinute: 480, DurationMinute: 30}}
	h.svc.mu.Unlock()

	stdout, stderr, code := runPlanCmd(runPlanRm, h.client, day, []string{"1"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "removed entry 1") {
		t.Errorf("expected removal message, got: %q", stdout)
	}
}

func TestRunPlanRm_NotFound(t *testing.T) {
	h := newPlanTestHarness(t)
	_, stderr, code := runPlanCmd(runPlanRm, h.client, "2026-05-21", []string{"99"})
	if code == 0 {
		t.Error("expected non-zero exit for not found")
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("expected 'not found' in stderr, got: %q", stderr)
	}
}

func TestRunPlanRename_HappyPath(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	h.svc.mu.Lock()
	h.svc.entries[day] = []*planv1.PlanEntry{{Day: day, Id: 1, Name: "Old", StartMinute: 480, DurationMinute: 30}}
	h.svc.mu.Unlock()

	stdout, stderr, code := runPlanCmd(runPlanRename, h.client, day, []string{"1", "New"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "renamed entry 1") {
		t.Errorf("expected rename message, got: %q", stdout)
	}
}

func TestRunPlanMv_HappyPath(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	h.svc.mu.Lock()
	h.svc.entries[day] = []*planv1.PlanEntry{{Day: day, Id: 1, Name: "X", StartMinute: 480, DurationMinute: 60}}
	h.svc.mu.Unlock()

	stdout, stderr, code := runPlanCmd(runPlanMv, h.client, day, []string{"1", "10:00am", "45m"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "moved entry 1") {
		t.Errorf("expected move message, got: %q", stdout)
	}
}

func TestRunPlanMv_DurationOmitted_Forwards_Zero(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	h.svc.mu.Lock()
	h.svc.entries[day] = []*planv1.PlanEntry{{Day: day, Id: 1, Name: "X", StartMinute: 480, DurationMinute: 60}}
	h.svc.mu.Unlock()

	runPlanCmd(runPlanMv, h.client, day, []string{"1", "10:00am"})
	h.svc.mu.Lock()
	defer h.svc.mu.Unlock()
	if h.svc.lastMvRequest == nil {
		t.Fatal("no mv request captured")
	}
	if h.svc.lastMvRequest.DurationMinute != 0 {
		t.Errorf("expected duration_minute=0 (keep current), got %d", h.svc.lastMvRequest.DurationMinute)
	}
}

// ---- T043: runPlanClear CLI tests ----

func TestRunPlanClear_ExplicitStart(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	stdout, stderr, code := runPlanCmd(runPlanClear, h.client, day, []string{"9:00am"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Cleared 2 entries") {
		t.Errorf("expected clear message, got: %q", stdout)
	}
	if !strings.Contains(stdout, "trimmed 1") {
		t.Errorf("expected trimmed count, got: %q", stdout)
	}
	h.svc.mu.Lock()
	defer h.svc.mu.Unlock()
	if h.svc.lastClearRequest.StartMinute != 540 {
		t.Errorf("start_minute = %d, want 540 (9:00am)", h.svc.lastClearRequest.StartMinute)
	}
}

func TestRunPlanClear_NoArg_UsesCurrentTime(t *testing.T) {
	h := newPlanTestHarness(t)
	day := "2026-05-21"
	runPlanCmd(runPlanClear, h.client, day, nil)
	h.svc.mu.Lock()
	defer h.svc.mu.Unlock()
	if h.svc.lastClearRequest == nil {
		t.Fatal("no clear request captured")
	}
	// Just verify it's a valid minute (0-1439); exact value depends on clock.
	m := h.svc.lastClearRequest.StartMinute
	if m < 0 || m >= 1440 {
		t.Errorf("start_minute %d out of valid range", m)
	}
}
