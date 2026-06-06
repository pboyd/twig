package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	tea "charm.land/bubbletea/v2"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── fake client for pomodoro tests ────────────────────────────────────────

type fakePomClient struct {
	taskv1connect.TaskServiceClient
	// StartPomodoro config
	startErr  error
	startResp *taskv1.Pomodoro
	// GetActivePomodoro config
	activeResp *taskv1.Pomodoro
	activeErr  error
	// CancelPomodoro config
	cancelErr error
	// CompletePomodoro config
	completeErr error
	// GetTask config
	getTaskResp *taskv1.Task
	getTaskErr  error
	// call tracking
	startCallCount  int
	cancelCallCount int
}

func (f *fakePomClient) StartPomodoro(_ context.Context, req *connect.Request[taskv1.StartPomodoroRequest]) (*connect.Response[taskv1.StartPomodoroResponse], error) {
	f.startCallCount++
	if f.startErr != nil {
		return nil, f.startErr
	}
	return connect.NewResponse(&taskv1.StartPomodoroResponse{Pomodoro: f.startResp}), nil
}

func (f *fakePomClient) GetActivePomodoro(_ context.Context, _ *connect.Request[taskv1.GetActivePomodoroRequest]) (*connect.Response[taskv1.GetActivePomodoroResponse], error) {
	if f.activeErr != nil {
		return nil, f.activeErr
	}
	return connect.NewResponse(&taskv1.GetActivePomodoroResponse{Pomodoro: f.activeResp}), nil
}

func (f *fakePomClient) CancelPomodoro(_ context.Context, _ *connect.Request[taskv1.CancelPomodoroRequest]) (*connect.Response[taskv1.CancelPomodoroResponse], error) {
	f.cancelCallCount++
	if f.cancelErr != nil {
		return nil, f.cancelErr
	}
	return connect.NewResponse(&taskv1.CancelPomodoroResponse{}), nil
}

func (f *fakePomClient) CompletePomodoro(_ context.Context, _ *connect.Request[taskv1.CompletePomodoroRequest]) (*connect.Response[taskv1.CompletePomodoroResponse], error) {
	if f.completeErr != nil {
		return nil, f.completeErr
	}
	return connect.NewResponse(&taskv1.CompletePomodoroResponse{}), nil
}

func (f *fakePomClient) GetTask(_ context.Context, req *connect.Request[taskv1.GetTaskRequest]) (*connect.Response[taskv1.GetTaskResponse], error) {
	if f.getTaskErr != nil {
		return nil, f.getTaskErr
	}
	return connect.NewResponse(&taskv1.GetTaskResponse{Task: f.getTaskResp}), nil
}

func (f *fakePomClient) ListTasks(_ context.Context, _ *connect.Request[taskv1.ListTasksRequest]) (*connect.Response[taskv1.ListTasksResponse], error) {
	return connect.NewResponse(&taskv1.ListTasksResponse{}), nil
}

// ── helpers ───────────────────────────────────────────────────────────────

func buildPomTestModel(client taskv1connect.TaskServiceClient) Model {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "task-one"},
		{Id: 2, Name: "task-two"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(client, tree)
	m.width = 80
	m.height = 24
	return m
}

func mustQuit(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd, got nil")
	}
	if cmd() != tea.Quit() {
		t.Error("expected tea.Quit")
	}
}

// ── T012: startPomCmd ─────────────────────────────────────────────────────

func TestStartPomCmd_Success(t *testing.T) {
	startAt := time.Now().Add(-5 * time.Minute)
	fc := &fakePomClient{
		startResp: &taskv1.Pomodoro{TaskId: 1, StartAt: timestamppb.New(startAt)},
	}
	cmd := startPomCmd(fc, 1, "task-one")
	msg := cmd()
	sm, ok := msg.(pomStartedMsg)
	if !ok {
		t.Fatalf("expected pomStartedMsg, got %T", msg)
	}
	if sm.err != nil {
		t.Fatalf("expected no error, got %v", sm.err)
	}
	if sm.taskID != 1 {
		t.Errorf("taskID: want 1, got %d", sm.taskID)
	}
	if sm.taskName != "task-one" {
		t.Errorf("taskName: want task-one, got %q", sm.taskName)
	}
	if !sm.startAt.Equal(startAt) {
		t.Errorf("startAt mismatch")
	}
}

func TestStartPomCmd_SameTaskAlreadyExists_Attaches(t *testing.T) {
	startAt := time.Now().Add(-3 * time.Minute)
	// AlreadyExists carrying same task id.
	alreadyExistsErr := connect.NewError(connect.CodeAlreadyExists, errors.New("already running"))
	detail, _ := connect.NewErrorDetail(&taskv1.StartPomodoroRequest{TaskId: 1})
	alreadyExistsErr.AddDetail(detail)

	fc := &fakePomClient{
		startErr:   alreadyExistsErr,
		activeResp: &taskv1.Pomodoro{TaskId: 1, StartAt: timestamppb.New(startAt)},
	}
	cmd := startPomCmd(fc, 1, "task-one")
	msg := cmd()
	sm, ok := msg.(pomStartedMsg)
	if !ok {
		t.Fatalf("expected pomStartedMsg, got %T", msg)
	}
	if sm.err != nil {
		t.Fatalf("expected no error on same-task attach, got %v", sm.err)
	}
	// Should not have called cancel.
	if fc.cancelCallCount != 0 {
		t.Errorf("cancel should not be called for same-task attach, called %d time(s)", fc.cancelCallCount)
	}
	if !sm.startAt.Equal(startAt) {
		t.Errorf("startAt should come from GetActivePomodoro")
	}
}

func TestStartPomCmd_DifferentTaskAlreadyExists_CancelsThenStarts(t *testing.T) {
	alreadyExistsErr := connect.NewError(connect.CodeAlreadyExists, errors.New("already running"))
	// Detail carries task id=1 (different from requested task id=2).
	detail, _ := connect.NewErrorDetail(&taskv1.StartPomodoroRequest{TaskId: 1})
	alreadyExistsErr.AddDetail(detail)

	newStartAt := time.Now()

	// Wrap: first StartPomodoro call returns AlreadyExists (task 1 active);
	// subsequent calls succeed.
	wrapped := &sequentialPomClient{
		calls: []pomClientCall{
			{startErr: alreadyExistsErr},
			{startResp: &taskv1.Pomodoro{TaskId: 2, StartAt: timestamppb.New(newStartAt)}},
		},
	}

	cmd2 := startPomCmd(wrapped, 2, "task-two")
	msg := cmd2()
	sm, ok := msg.(pomStartedMsg)
	if !ok {
		t.Fatalf("expected pomStartedMsg, got %T", msg)
	}
	if sm.err != nil {
		t.Fatalf("expected no error after cancel+start, got %v", sm.err)
	}
	if wrapped.cancelCount != 1 {
		t.Errorf("expected 1 cancel call, got %d", wrapped.cancelCount)
	}
	if sm.taskID != 2 {
		t.Errorf("taskID: want 2, got %d", sm.taskID)
	}
}

// sequentialPomClient feeds pre-configured responses for StartPomodoro calls in order.
type pomClientCall struct {
	startErr  error
	startResp *taskv1.Pomodoro
}

type sequentialPomClient struct {
	taskv1connect.TaskServiceClient
	calls       []pomClientCall
	callIdx     int
	cancelCount int
}

func (s *sequentialPomClient) StartPomodoro(_ context.Context, _ *connect.Request[taskv1.StartPomodoroRequest]) (*connect.Response[taskv1.StartPomodoroResponse], error) {
	if s.callIdx >= len(s.calls) {
		return nil, errors.New("unexpected StartPomodoro call")
	}
	c := s.calls[s.callIdx]
	s.callIdx++
	if c.startErr != nil {
		return nil, c.startErr
	}
	return connect.NewResponse(&taskv1.StartPomodoroResponse{Pomodoro: c.startResp}), nil
}

func (s *sequentialPomClient) CancelPomodoro(_ context.Context, _ *connect.Request[taskv1.CancelPomodoroRequest]) (*connect.Response[taskv1.CancelPomodoroResponse], error) {
	s.cancelCount++
	return connect.NewResponse(&taskv1.CancelPomodoroResponse{}), nil
}

func (s *sequentialPomClient) GetActivePomodoro(_ context.Context, _ *connect.Request[taskv1.GetActivePomodoroRequest]) (*connect.Response[taskv1.GetActivePomodoroResponse], error) {
	return connect.NewResponse(&taskv1.GetActivePomodoroResponse{}), nil
}

func (s *sequentialPomClient) ListTasks(_ context.Context, _ *connect.Request[taskv1.ListTasksRequest]) (*connect.Response[taskv1.ListTasksResponse], error) {
	return connect.NewResponse(&taskv1.ListTasksResponse{}), nil
}

// TestCancelPomCmd_ReturnsMsg checks that cancelPomCmd returns pomCancelledMsg.
func TestCancelPomCmd_ReturnsMsg(t *testing.T) {
	fc := &fakePomClient{}
	msg := cancelPomCmd(fc)()
	cm, ok := msg.(pomCancelledMsg)
	if !ok {
		t.Fatalf("expected pomCancelledMsg, got %T", msg)
	}
	if cm.err != nil {
		t.Fatalf("expected no error, got %v", cm.err)
	}
}

// TestCancelPomCmd_ErrorPropagates checks error is carried in pomCancelledMsg.
func TestCancelPomCmd_ErrorPropagates(t *testing.T) {
	fc := &fakePomClient{cancelErr: errors.New("rpc fail")}
	msg := cancelPomCmd(fc)()
	cm, ok := msg.(pomCancelledMsg)
	if !ok {
		t.Fatalf("expected pomCancelledMsg, got %T", msg)
	}
	if cm.err == nil {
		t.Error("expected error in pomCancelledMsg")
	}
}

// TestStartPomCmd_ErrorPropagates checks that an RPC error returns pomStartedMsg with err.
func TestStartPomCmd_ErrorPropagates(t *testing.T) {
	fc := &fakePomClient{startErr: errors.New("network error")}
	msg := startPomCmd(fc, 1, "task-one")()
	sm, ok := msg.(pomStartedMsg)
	if !ok {
		t.Fatalf("expected pomStartedMsg, got %T", msg)
	}
	if sm.err == nil {
		t.Error("expected error in pomStartedMsg")
	}
}

// TestPomTickMsg_IgnoredWhenNoPom checks that pomTickMsg is a no-op when m.pom==nil.
func TestPomTickMsg_IgnoredWhenNoPom(t *testing.T) {
	m := buildPomTestModel(nil)
	if m.pom != nil {
		t.Fatal("precondition: pom should be nil")
	}
	next, cmd := m.Update(pomTickMsg{})
	nm := next.(Model)
	if nm.pom != nil {
		t.Error("pom should remain nil")
	}
	if cmd != nil {
		t.Error("no cmd should be returned when pom is nil")
	}
}

// TestPomTickMsg_ReschedulesWhenActive checks that a tick reschedules while pom is active.
func TestPomTickMsg_ReschedulesWhenActive(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:   1,
		taskName: "task-one",
		startAt:  time.Now().Add(-1 * time.Minute),
	}
	_, cmd := m.Update(pomTickMsg{})
	if cmd == nil {
		t.Error("expected a reschedule cmd while pom is active")
	}
}

// TestPomTickMsg_IgnoredWhenCompleted checks that completed pom does not reschedule.
func TestPomTickMsg_IgnoredWhenCompleted(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:    1,
		taskName:  "task-one",
		startAt:   time.Now().Add(-30 * time.Minute),
		completed: true,
		banner:    "done",
	}
	_, cmd := m.Update(pomTickMsg{})
	if cmd != nil {
		t.Error("no cmd should be returned when pom is completed (tick ignored)")
	}
}

// TestPomStartedMsg_SetsActivePom checks that pomStartedMsg on success sets m.pom.
func TestPomStartedMsg_SetsActivePom(t *testing.T) {
	m := buildPomTestModel(nil)
	startAt := time.Now()
	next, _ := m.Update(pomStartedMsg{taskID: 1, taskName: "task-one", startAt: startAt})
	nm := next.(Model)
	if nm.pom == nil {
		t.Fatal("expected pom to be set")
	}
	if nm.pom.taskID != 1 {
		t.Errorf("taskID: want 1, got %d", nm.pom.taskID)
	}
	if nm.pom.taskName != "task-one" {
		t.Errorf("taskName: want task-one, got %q", nm.pom.taskName)
	}
	if nm.err != nil {
		t.Errorf("err should be nil on success, got %v", nm.err)
	}
}

// TestPomStartedMsg_ErrorSetsErr verifies that a pomStartedMsg with error sets m.err
// and does NOT set m.pom.
func TestPomStartedMsg_ErrorSetsErr(t *testing.T) {
	m := buildPomTestModel(nil)
	next, _ := m.Update(pomStartedMsg{err: errors.New("rpc fail")})
	nm := next.(Model)
	if nm.err == nil {
		t.Error("expected m.err to be set")
	}
	if nm.pom != nil {
		t.Error("m.pom should remain nil on error")
	}
}

// TestPomCancelledMsg_ClearsPom checks that pomCancelledMsg clears m.pom.
func TestPomCancelledMsg_ClearsPom(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now()}
	next, _ := m.Update(pomCancelledMsg{})
	nm := next.(Model)
	if nm.pom != nil {
		t.Error("expected pom to be nil after cancel")
	}
}

// TestPomCancelledMsg_ErrorSetsErr verifies that a pomCancelledMsg with error sets m.err
// and leaves the model usable.
func TestPomCancelledMsg_ErrorSetsErr(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now()}
	next, _ := m.Update(pomCancelledMsg{err: errors.New("rpc fail")})
	nm := next.(Model)
	if nm.err == nil {
		t.Error("expected m.err to be set")
	}
	// Mode should remain usable (modeList).
	if nm.mode != modeList {
		t.Errorf("mode should remain modeList, got %v", nm.mode)
	}
}

// ── T013: view tests for active pomodoro ─────────────────────────────────

func TestView_PomActiveStatusTwoLines(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:   1,
		taskName: "task-one",
		startAt:  time.Now().Add(-10 * time.Minute),
	}
	m.styled = false
	status := m.renderStatus()
	lines := splitLines(status, 2)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 status lines, got %d", len(lines))
	}
	if !containsTimePattern(lines[0]) {
		t.Errorf("first status line should contain mm:ss timer, got %q", lines[0])
	}
	if lines[1] == "" {
		t.Errorf("second status line (help) should not be empty")
	}
}

func TestView_PomIdleStatusOneLine(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = nil
	status := m.renderStatus()
	// Should not contain a newline (just one line).
	if countNewlines(status) > 0 {
		t.Errorf("idle status should be one line, got: %q", status)
	}
}

func TestView_PomTimerContainsTaskName(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:   1,
		taskName: "my-special-task",
		startAt:  time.Now().Add(-5 * time.Minute),
	}
	m.styled = false
	status := m.renderStatus()
	if !contains(status, "my-special-task") {
		t.Errorf("status should contain task name; got: %q", status)
	}
}

func TestView_PomTimerUnstyledNoAnsi(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:   1,
		taskName: "task-one",
		startAt:  time.Now().Add(-5 * time.Minute),
	}
	m.styled = false
	status := m.renderStatus()
	if contains(status, "\x1b[") {
		t.Errorf("unstyled status must not contain ANSI escapes; got: %q", status)
	}
}

func TestView_PomTimerInModeList(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:   1,
		taskName: "task-one",
		startAt:  time.Now().Add(-5 * time.Minute),
	}
	m.styled = false
	out := m.viewList()
	if !contains(out, "task-one") {
		t.Errorf("modeList view should show task name in timer; got %q", out[:min(len(out), 200)])
	}
}

func TestView_PomTimerInModeEdit(t *testing.T) {
	tasks := []*taskv1.Task{{Id: 1, Name: "task-one"}}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)
	m.width = 80
	m.height = 24
	m.styled = false
	m.pom = &activePom{
		taskID:   1,
		taskName: "task-one",
		startAt:  time.Now().Add(-5 * time.Minute),
	}
	m.mode = modeEdit
	m.edit = NewEditForm(tasks[0], 0)
	out := m.viewWithForm()
	if !contains(out, "task-one") {
		t.Errorf("modeEdit view should show timer; got:\n%q", out[:min(len(out), 200)])
	}
}

// ── T017: completion tests ────────────────────────────────────────────────

// TestPomTick_CrossingZeroTriggersCompletionOnce verifies exactly-once completion.
func TestPomTick_CrossingZeroTriggersCompletionOnce(t *testing.T) {
	m := buildPomTestModel(nil)
	// Set startAt far in the past so remaining == 0.
	m.pom = &activePom{
		taskID:   1,
		taskName: "task-one",
		startAt:  time.Now().Add(-30 * time.Minute),
	}
	next, cmd := m.Update(pomTickMsg{})
	nm := next.(Model)
	if !nm.pom.completed {
		t.Error("pom should be marked completed when remaining == 0")
	}
	if cmd == nil {
		t.Error("expected completePomCmd to be returned")
	}
	// Second tick: completed, so no re-fire.
	next2, cmd2 := nm.Update(pomTickMsg{})
	nm2 := next2.(Model)
	if cmd2 != nil {
		t.Error("second tick after completion should return nil cmd")
	}
	if !nm2.pom.completed {
		t.Error("pom should remain completed")
	}
}

// TestPomCompletedMsg_SetsBanner checks that pomCompletedMsg sets the banner.
func TestPomCompletedMsg_SetsBanner(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now(), completed: true}
	m.styled = false
	next, _ := m.Update(pomCompletedMsg{taskName: "task-one"})
	nm := next.(Model)
	if nm.pom == nil {
		t.Fatal("pom should not be nil after completion msg")
	}
	if nm.pom.banner == "" {
		t.Error("banner should be set after pomCompletedMsg")
	}
	if !contains(nm.pom.banner, "task-one") {
		t.Errorf("banner should contain task name; got %q", nm.pom.banner)
	}
}

// TestPomCompletedMsg_ErrorSetsErr verifies RPC error in pomCompletedMsg sets m.err.
func TestPomCompletedMsg_ErrorSetsErr(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now(), completed: true}
	next, _ := m.Update(pomCompletedMsg{taskName: "task-one", err: errors.New("rpc fail")})
	nm := next.(Model)
	if nm.err == nil {
		t.Error("expected m.err to be set")
	}
	if nm.mode != modeList {
		t.Errorf("mode should remain modeList, got %v", nm.mode)
	}
}

// TestBannerClearsOnKeypress checks banner is cleared on the next keypress, which
// still performs its normal action (F2: keypress is not swallowed).
func TestBannerClearsOnKeypress(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:    1,
		taskName:  "task-one",
		startAt:   time.Now().Add(-30 * time.Minute),
		completed: true,
		banner:    "Pomodoro complete! task-one",
	}
	m.cursor = 0
	// Press 'j' (down) — should clear banner AND move cursor.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	nm := next.(Model)
	if nm.pom != nil {
		t.Error("pom should be nil after banner-clearing keypress")
	}
	if nm.cursor != 1 {
		t.Errorf("cursor should have moved to 1 after 'j', got %d", nm.cursor)
	}
}

// TestBannerClearsOnExpiry checks that pomBannerExpireMsg clears the pom.
func TestBannerClearsOnExpiry(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:    1,
		taskName:  "task-one",
		startAt:   time.Now().Add(-30 * time.Minute),
		completed: true,
		banner:    "Pomodoro complete! task-one",
	}
	next, _ := m.Update(pomBannerExpireMsg{})
	nm := next.(Model)
	if nm.pom != nil {
		t.Error("pom should be nil after pomBannerExpireMsg")
	}
}

// TestPomHookErrMsg_SetsErr checks that a hook error is surfaced in m.err.
func TestPomHookErrMsg_SetsErr(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now(), completed: true}
	next, _ := m.Update(pomHookErrMsg{err: errors.New("hook failed")})
	nm := next.(Model)
	if nm.err == nil {
		t.Error("expected m.err to be set from pomHookErrMsg")
	}
	if nm.pom == nil {
		t.Error("pom should still be set (hook error doesn't clear it)")
	}
}

// ── T020: auto-attach at launch ───────────────────────────────────────────

// TestPomActiveMsg_SeedsModel checks that a pomActiveMsg with a pom sets m.pom and starts tick.
func TestPomActiveMsg_SeedsModel(t *testing.T) {
	m := buildPomTestModel(nil)
	startAt := time.Now().Add(-10 * time.Minute)
	pom := &activePom{
		taskID:   1,
		taskName: "task-one",
		startAt:  startAt,
	}
	next, cmd := m.Update(pomActiveMsg{pom: pom})
	nm := next.(Model)
	if nm.pom == nil {
		t.Fatal("expected pom to be seeded from pomActiveMsg")
	}
	if nm.pom.taskID != 1 {
		t.Errorf("taskID: want 1, got %d", nm.pom.taskID)
	}
	if !nm.pom.startAt.Equal(startAt) {
		t.Error("startAt mismatch")
	}
	if cmd == nil {
		t.Error("expected pomTickCmd to be returned")
	}
}

// TestPomActiveMsg_NilPomLeavesIdle checks that pomActiveMsg with nil pom leaves m.pom nil.
func TestPomActiveMsg_NilPomLeavesIdle(t *testing.T) {
	m := buildPomTestModel(nil)
	next, _ := m.Update(pomActiveMsg{pom: nil})
	nm := next.(Model)
	if nm.pom != nil {
		t.Error("expected pom to remain nil")
	}
}

// ── T023: quit guard ──────────────────────────────────────────────────────

// TestQuitGuard_ActivePomSetsConfirmingQuit checks that q with active pom prompts.
func TestQuitGuard_ActivePomSetsConfirmingQuit(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now()}
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	nm := next.(Model)
	if !nm.confirmingQuit {
		t.Error("expected confirmingQuit to be set when pom is active")
	}
	if cmd != nil {
		t.Error("should not quit immediately")
	}
}

// TestQuitGuard_YQuitsWhenConfirming checks that y quits when confirmingQuit is set.
func TestQuitGuard_YQuitsWhenConfirming(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now()}
	m.confirmingQuit = true
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	mustQuit(t, cmd)
}

// TestQuitGuard_NClearsConfirmingQuit checks that n clears the quit confirm state.
func TestQuitGuard_NClearsConfirmingQuit(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now()}
	m.confirmingQuit = true
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	nm := next.(Model)
	if nm.confirmingQuit {
		t.Error("confirmingQuit should be cleared after 'n'")
	}
	if cmd != nil {
		t.Error("should not quit after 'n'")
	}
}

// TestQuitGuard_EscClearsConfirmingQuit checks that esc clears confirmingQuit.
func TestQuitGuard_EscClearsConfirmingQuit(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: time.Now()}
	m.confirmingQuit = true
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	nm := next.(Model)
	if nm.confirmingQuit {
		t.Error("confirmingQuit should be cleared after esc")
	}
	if cmd != nil {
		t.Error("should not quit after esc")
	}
}

// TestQuitGuard_NoPomQuitsImmediately checks that q with no pom quits immediately.
func TestQuitGuard_NoPomQuitsImmediately(t *testing.T) {
	m := buildPomTestModel(nil)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	mustQuit(t, cmd)
}

// TestQuitGuard_CompletedPomQuitsImmediately checks that q with a completed pom quits.
func TestQuitGuard_CompletedPomQuitsImmediately(t *testing.T) {
	m := buildPomTestModel(nil)
	m.pom = &activePom{
		taskID:    1,
		taskName:  "task-one",
		startAt:   time.Now().Add(-30 * time.Minute),
		completed: true,
		banner:    "done",
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	mustQuit(t, cmd)
}

// ── test helpers ──────────────────────────────────────────────────────────

func containsTimePattern(s string) bool {
	// Looks for digit:digit pattern like "15:00".
	for i := 0; i+4 < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' &&
			s[i+1] >= '0' && s[i+1] <= '9' &&
			s[i+2] == ':' &&
			s[i+3] >= '0' && s[i+3] <= '9' &&
			s[i+4] >= '0' && s[i+4] <= '9' {
			return true
		}
	}
	return false
}

func countNewlines(s string) int {
	n := 0
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
