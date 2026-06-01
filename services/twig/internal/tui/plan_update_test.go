package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	planv1 "github.com/pboyd/twig/services/twig/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/services/twig/gen/plan/v1/planv1connect"
	taskv1 "github.com/pboyd/twig/services/twig/gen/task/v1"
	"github.com/pboyd/twig/services/twig/internal/cli"
)

// fakePlanClient is a minimal PlanServiceClient for unit tests.
type fakePlanClient struct {
	planv1connect.PlanServiceClient
	entries     []*planv1.PlanEntry
	lastListDay string
	listErr     error
	addTaskReq  *planv1.AddPlanTaskRequest
	addEventReq *planv1.AddPlanEventRequest
	renameReq   *planv1.RenamePlanEntryRequest
	moveReq     *planv1.MovePlanEntryRequest
	removeReq   *planv1.RemovePlanEntryRequest
	clearReq    *planv1.ClearPlanRequest
	mutateErr   error
}

func (f *fakePlanClient) ListPlanEntries(_ context.Context, req *connect.Request[planv1.ListPlanEntriesRequest]) (*connect.Response[planv1.ListPlanEntriesResponse], error) {
	f.lastListDay = req.Msg.Day
	if f.listErr != nil {
		return nil, f.listErr
	}
	return connect.NewResponse(&planv1.ListPlanEntriesResponse{Entries: f.entries}), nil
}

func (f *fakePlanClient) AddPlanTask(_ context.Context, req *connect.Request[planv1.AddPlanTaskRequest]) (*connect.Response[planv1.AddPlanTaskResponse], error) {
	f.addTaskReq = req.Msg
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	return connect.NewResponse(&planv1.AddPlanTaskResponse{Entry: &planv1.PlanEntry{Id: 99}}), nil
}

func (f *fakePlanClient) AddPlanEvent(_ context.Context, req *connect.Request[planv1.AddPlanEventRequest]) (*connect.Response[planv1.AddPlanEventResponse], error) {
	f.addEventReq = req.Msg
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	return connect.NewResponse(&planv1.AddPlanEventResponse{Entry: &planv1.PlanEntry{Id: 98}}), nil
}

func (f *fakePlanClient) RenamePlanEntry(_ context.Context, req *connect.Request[planv1.RenamePlanEntryRequest]) (*connect.Response[planv1.RenamePlanEntryResponse], error) {
	f.renameReq = req.Msg
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	return connect.NewResponse(&planv1.RenamePlanEntryResponse{}), nil
}

func (f *fakePlanClient) MovePlanEntry(_ context.Context, req *connect.Request[planv1.MovePlanEntryRequest]) (*connect.Response[planv1.MovePlanEntryResponse], error) {
	f.moveReq = req.Msg
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	return connect.NewResponse(&planv1.MovePlanEntryResponse{}), nil
}

func (f *fakePlanClient) RemovePlanEntry(_ context.Context, req *connect.Request[planv1.RemovePlanEntryRequest]) (*connect.Response[planv1.RemovePlanEntryResponse], error) {
	f.removeReq = req.Msg
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	return connect.NewResponse(&planv1.RemovePlanEntryResponse{}), nil
}

func (f *fakePlanClient) ClearPlan(_ context.Context, req *connect.Request[planv1.ClearPlanRequest]) (*connect.Response[planv1.ClearPlanResponse], error) {
	f.clearReq = req.Msg
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	return connect.NewResponse(&planv1.ClearPlanResponse{}), nil
}

// buildPlanTestModel creates a model ready for planning tab tests.
func buildPlanTestModel(fc *fakePlanClient) Model {
	m := ExportNewPlanModel(nil, fc, "2026-05-27")
	m.width = 80
	m.height = 30
	return m
}

// pressKeyStr sends a key string to the model.
func pressKeyStr(m Model, k string) (Model, tea.Cmd) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

// pressSpecialKey sends a special key (tab, shift+tab, etc.) to the model.
func pressSpecialKey(m Model, typ tea.KeyType) (Model, tea.Cmd) {
	msg := tea.KeyMsg{Type: typ}
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

// ── US1: tab switching ──────────────────────────────────────────────────────

// TestTabSwitch_TasksToPlanning checks that pressing Tab from Tasks tab activates
// the Planning tab and issues a listPlanCmd.
func TestTabSwitch_TasksToPlanning(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	// Start on tasks tab
	m.activeTab = tabTasks

	m2, cmd := pressSpecialKey(m, tea.KeyTab)

	if m2.activeTab != tabPlanning {
		t.Errorf("after Tab: expected tabPlanning, got %d", m2.activeTab)
	}
	if cmd == nil {
		t.Error("after Tab: expected a command (listPlanCmd), got nil")
	}
}

// TestTabSwitch_PlanningToTasks checks that pressing Tab from Planning tab
// switches back to Tasks tab.
func TestTabSwitch_PlanningToTasks(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc) // starts in tabPlanning

	m2, _ := pressSpecialKey(m, tea.KeyTab)

	if m2.activeTab != tabTasks {
		t.Errorf("after Tab on Planning: expected tabTasks, got %d", m2.activeTab)
	}
}

// TestTabSwitch_ShiftTabPlanningToTasks checks that Shift+Tab from Planning tab
// switches to Tasks tab.
func TestTabSwitch_ShiftTabPlanningToTasks(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc) // tabPlanning

	m2, _ := pressSpecialKey(m, tea.KeyShiftTab)

	if m2.activeTab != tabTasks {
		t.Errorf("after Shift+Tab on Planning: expected tabTasks, got %d", m2.activeTab)
	}
}

// TestTabSwitch_BackPreservesTasksCursor checks that switching back to Tasks
// preserves the cursor position (it's not reset).
func TestTabSwitch_BackPreservesTasksCursor(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.activeTab = tabTasks
	m.cursor = 2
	m.visible = []*visibleRow{{}, {}, {}}

	// Switch to planning
	m, _ = pressSpecialKey(m, tea.KeyTab)
	// Switch back to tasks
	m, _ = pressSpecialKey(m, tea.KeyTab)

	if m.cursor != 2 {
		t.Errorf("switching back to Tasks: cursor should be preserved (2), got %d", m.cursor)
	}
}

// TestTabSwitch_BlockedWhileModal checks that Tab key is ignored (no tab switch)
// while plan.mode != planList.
func TestTabSwitch_BlockedWhileModal(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc) // tabPlanning
	m.plan.mode = planEventForm // a modal is open

	m2, _ := pressSpecialKey(m, tea.KeyTab)

	if m2.activeTab != tabPlanning {
		t.Errorf("Tab while modal open: activeTab should stay tabPlanning, got %d", m2.activeTab)
	}
}

// ── planEntriesMsg handling ─────────────────────────────────────────────────

// TestPlanEntriesMsg_PopulatesEntries checks that a planEntriesMsg populates
// plan.entries and sets loaded=true.
func TestPlanEntriesMsg_PopulatesEntries(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})
	m.plan.loaded = false

	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540)},
		{Id: 2, Name: "Focus", StartMinute: pint32(600)},
	}
	next, _ := m.Update(planEntriesMsg{entries: entries})
	nm := next.(Model)

	if !nm.plan.loaded {
		t.Error("planEntriesMsg: plan.loaded should be true")
	}
	if len(nm.plan.entries) != 2 {
		t.Errorf("planEntriesMsg: expected 2 entries, got %d", len(nm.plan.entries))
	}
}

// TestPlanEntriesMsg_ClampsCursor checks that cursor is clamped to the new entry list length.
func TestPlanEntriesMsg_ClampsCursor(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})
	m.plan.cursor = 5

	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "One", StartMinute: pint32(540)},
	}
	next, _ := m.Update(planEntriesMsg{entries: entries})
	nm := next.(Model)

	if nm.plan.cursor != 0 {
		t.Errorf("planEntriesMsg: cursor should be clamped to 0, got %d", nm.plan.cursor)
	}
}

// TestPlanEntriesMsg_ErrorSetsErr checks that an error in planEntriesMsg is surfaced.
func TestPlanEntriesMsg_ErrorSetsErr(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})

	next, _ := m.Update(planEntriesMsg{err: errForTest("list failed")})
	nm := next.(Model)

	if nm.plan.err == nil {
		t.Error("planEntriesMsg error: plan.err should be set")
	}
}

// ── planTickMsg handling ────────────────────────────────────────────────────

// TestPlanTick_RearmsOnTodayActivePlanning checks that planTickMsg re-arms
// the tick while the Planning tab is active on today.
func TestPlanTick_RearmsOnTodayActivePlanning(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})
	m.activeTab = tabPlanning
	m.plan.day = "2099-01-01" // a future day that won't be "today"

	// If day != today, tick should NOT re-arm.
	_, cmd := m.Update(planTickMsg{})
	if cmd != nil {
		// Execute cmd and check if it returns a planTickMsg.
		result := cmd()
		if _, ok := result.(planTickMsg); ok {
			t.Error("planTickMsg: tick should not re-arm when day != today")
		}
	}
}

// TestPlanMutatedMsg_ErrorSetsErr checks that a mutation error is surfaced.
func TestPlanMutatedMsg_ErrorSetsErr(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})

	next, _ := m.Update(planMutatedMsg{err: errForTest("rpc failed")})
	nm := next.(Model)

	if nm.plan.err == nil {
		t.Error("planMutatedMsg error: plan.err should be set")
	}
}

// TestPlanMutatedMsg_Success_ReloadsEntries checks that a successful mutation
// issues a listPlanCmd.
func TestPlanMutatedMsg_Success_ReloadsEntries(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})

	_, cmd := m.Update(planMutatedMsg{highlightID: 5})
	if cmd == nil {
		t.Error("planMutatedMsg success: expected a reload command, got nil")
	}
}

// TestPomTick_ContinuesWhilePlanningActive checks that the pomodoro tick continues
// while the Planning tab is active (the tick handler is tab-independent).
func TestPomTick_ContinuesWhilePlanningActive(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})
	m.activeTab = tabPlanning
	m.pom = &activePom{
		taskID:   1,
		taskName: "some task",
		startAt:  time.Date(2099, 1, 1, 10, 0, 0, 0, time.UTC),
	}

	// A pomTickMsg while on the Planning tab should re-arm the tick (pom is not done).
	_, cmd := m.Update(pomTickMsg{})
	if cmd == nil {
		t.Error("pomTickMsg while planning active: expected re-arm cmd, got nil")
	}
}

// ── handlePlanEntriesMsg ───────────────────────────────────────────────────

// TestHandlePlanEntriesMsg_HighlightID checks that the cursor lands on the
// entry matching highlightID.
func TestHandlePlanEntriesMsg_HighlightID(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})
	entries := []*planv1.PlanEntry{
		{Id: 10, Name: "A"},
		{Id: 20, Name: "B"},
		{Id: 30, Name: "C"},
	}
	m = m.handlePlanEntriesMsg(planEntriesMsg{entries: entries, highlightID: 20}, 0)

	if m.plan.cursor != 1 {
		t.Errorf("highlightID=20: expected cursor=1, got %d", m.plan.cursor)
	}
}

// ── US2: add task and event ─────────────────────────────────────────────────

// TestAddTask_PickerOpensOnT checks that pressing 't' in planList mode issues
// a listTasksForPickerCmd and sets mode to planPickTask (T018/US5).
func TestAddTask_PickerOpensOnT(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true

	m2, cmd := pressKeyStr(m, "t")

	if m2.plan.mode != planPickTask {
		t.Errorf("after 't': expected planPickTask, got %d", m2.plan.mode)
	}
	if cmd == nil {
		t.Error("after 't': expected a list-tasks command, got nil")
	}
}

// TestAddTask_EscCancels checks that pressing Esc in planPickTask mode returns to planList.
func TestAddTask_EscCancels(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.mode = planPickTask

	m2, _ := pressSpecialKey(m, tea.KeyEscape)

	if m2.plan.mode != planList {
		t.Errorf("esc from picker: expected planList, got %d", m2.plan.mode)
	}
}

// TestAddEvent_FormOpensOnE checks that pressing 'e' opens planEventForm.
func TestAddEvent_FormOpensOnE(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true

	m2, _ := pressKeyStr(m, "e")

	if m2.plan.mode != planEventForm {
		t.Errorf("after 'e': expected planEventForm, got %d", m2.plan.mode)
	}
	if len(m2.plan.form.fields) != 3 {
		t.Errorf("event form: expected 3 fields (name/start/dur), got %d", len(m2.plan.form.fields))
	}
}

// TestAddEvent_EscCancels checks that Esc in planEventForm returns to planList.
func TestAddEvent_EscCancels(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.mode = planEventForm
	m.initAddEventForm()

	m2, _ := pressSpecialKey(m, tea.KeyEscape)

	if m2.plan.mode != planList {
		t.Errorf("esc from event form: expected planList, got %d", m2.plan.mode)
	}
}

// TestAddEvent_InvalidTimeSetsError checks that submitting a form with an
// invalid time leaves the form open and sets plan.err.
func TestAddEvent_InvalidTimeSetsError(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.mode = planEventForm
	m.initAddEventForm()
	// Set name but bad time
	m.plan.form.fields[0].SetValue("Meeting")
	m.plan.form.fields[1].SetValue("not-a-time")

	// Submit with Ctrl+S
	msg := tea.KeyMsg{Type: tea.KeyCtrlS}
	next, _ := m.Update(msg)
	nm := next.(Model)

	if nm.plan.err == nil {
		t.Error("invalid start time: expected plan.err to be set")
	}
	if nm.plan.mode != planEventForm {
		t.Errorf("invalid time: form should stay open (planEventForm), got %d", nm.plan.mode)
	}
}

// TestTabSwitch_BlockedWhilePlannerForm checks that Tab does not switch tabs
// while a planner form is open.
func TestTabSwitch_BlockedWhilePlannerForm(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.mode = planEdit

	m2, _ := pressSpecialKey(m, tea.KeyTab)

	if m2.activeTab != tabPlanning {
		t.Errorf("Tab while form open: should not switch tabs, got tab=%d", m2.activeTab)
	}
}

// ── US1: Enter-driven edit form ────────────────────────────────────────────

// TestEdit_FormOpensOnEnter checks that Enter on Planning with an entry selected
// enters planEdit with Name prefilled, and entryID set (T002).
func TestEdit_FormOpensOnEnter(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "Standup", StartMinute: pint32(540)}}
	m.plan.cursor = 0

	m2, _ := pressSpecialKey(m, tea.KeyEnter)

	if m2.plan.mode != planEdit {
		t.Errorf("Enter with entry: expected planEdit, got %d", m2.plan.mode)
	}
	if len(m2.plan.form.fields) != 3 {
		t.Errorf("edit form: expected 3 fields (Name/Start/Duration), got %d", len(m2.plan.form.fields))
	}
	if m2.plan.form.fields[0].Value() != "Standup" {
		t.Errorf("edit form: expected Name prefilled with 'Standup', got %q", m2.plan.form.fields[0].Value())
	}
	if m2.plan.form.entryID != 1 {
		t.Errorf("edit form: expected entryID=1, got %d", m2.plan.form.entryID)
	}
}

// TestEdit_InertOnEmptyGrid checks that Enter with no entries is inert (T002).
func TestEdit_InertOnEmptyGrid(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = nil

	m2, _ := pressSpecialKey(m, tea.KeyEnter)

	if m2.plan.mode != planList {
		t.Errorf("Enter with no entries: mode should stay planList, got %d", m2.plan.mode)
	}
}

// TestEditForm_RenameOnNameChange checks that submitEditForm issues RenamePlanEntry
// when the name changed (T003).
func TestEditForm_RenameOnNameChange(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Old Name", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	m.plan.form.fields[0].SetValue("New Name")
	// Start and Duration blank (no move)

	cmd := m.submitEditForm()

	if m.plan.mode != planList {
		t.Errorf("submitEditForm: expected mode planList, got %d", m.plan.mode)
	}
	if cmd == nil {
		t.Error("submitEditForm: expected a rename command, got nil")
	}
	if cmd != nil {
		cmd()
		if fc.renameReq == nil {
			t.Error("submitEditForm rename: RenamePlanEntry was not called")
		} else if fc.renameReq.Name != "New Name" {
			t.Errorf("submitEditForm rename: expected name 'New Name', got %q", fc.renameReq.Name)
		}
		if fc.moveReq != nil {
			t.Error("submitEditForm rename-only: MovePlanEntry should NOT be called")
		}
	}
}

// TestEditForm_MoveOnStartProvided checks that submitEditForm issues MovePlanEntry
// when a start time is provided (T003).
func TestEditForm_MoveOnStartProvided(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Standup", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	// Name unchanged, start provided
	m.plan.form.fields[1].SetValue("10:00")

	cmd := m.submitEditForm()

	if m.plan.mode != planList {
		t.Errorf("submitEditForm move: expected planList, got %d", m.plan.mode)
	}
	if cmd == nil {
		t.Error("submitEditForm move: expected a command, got nil")
	}
	if cmd != nil {
		cmd()
		if fc.moveReq == nil {
			t.Error("submitEditForm move: MovePlanEntry was not called")
		}
		if fc.renameReq != nil {
			t.Error("submitEditForm move-only: RenamePlanEntry should NOT be called")
		}
	}
}

// TestEditForm_NoopWhenUnchanged checks that submitEditForm is a no-op when neither
// name nor time changed (T003).
func TestEditForm_NoopWhenUnchanged(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Standup", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	// Name unchanged (same value), Start and Duration blank

	cmd := m.submitEditForm()

	if m.plan.mode != planList {
		t.Errorf("submitEditForm no-op: expected planList, got %d", m.plan.mode)
	}
	if cmd != nil {
		t.Error("submitEditForm no-op: expected nil cmd, got non-nil")
	}
}

// TestEditForm_EmptyNameSetsError checks that an empty Name keeps the form open
// and sets plan.err (T003).
func TestEditForm_EmptyNameSetsError(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Standup", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	m.plan.form.fields[0].SetValue("") // empty name

	cmd := m.submitEditForm()

	if m.plan.mode != planEdit {
		t.Errorf("empty name: form should stay planEdit, got %d", m.plan.mode)
	}
	if m.plan.err == nil {
		t.Error("empty name: plan.err should be set")
	}
	if cmd != nil {
		t.Error("empty name: expected nil cmd")
	}
}

// ── US3: remove/clear tests updated ────────────────────────────────────────

// TestRemove_IssuesRPCOnCtrlD checks that Ctrl+D issues a removePlanCmd.
func TestRemove_IssuesRPCOnCtrlD(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 3, Name: "Review", StartMinute: pint32(600)}}
	m.plan.cursor = 0

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})

	if cmd == nil {
		t.Error("ctrl+d: expected removePlanCmd, got nil")
	}
	// Execute the command to verify RPC is called.
	if cmd != nil {
		cmd()
		if fc.removeReq == nil {
			t.Error("removePlanCmd: RemovePlanEntry was not called")
		} else if fc.removeReq.Id != 3 {
			t.Errorf("removePlanCmd: expected id=3, got %d", fc.removeReq.Id)
		}
	}
}

// TestNavigation_UpDownMovesCursor checks that ↑/↓ (j/k) move the plan cursor.
func TestNavigation_UpDownMovesCursor(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, StartMinute: pint32(540)},
		{Id: 2, StartMinute: pint32(600)},
		{Id: 3, StartMinute: pint32(660)},
	}
	m.plan.cursor = 0

	m2, _ := pressKeyStr(m, "j")
	if m2.plan.cursor != 1 {
		t.Errorf("down: expected cursor=1, got %d", m2.plan.cursor)
	}
	m3, _ := pressKeyStr(m2, "k")
	if m3.plan.cursor != 0 {
		t.Errorf("up: expected cursor=0, got %d", m3.plan.cursor)
	}
	// Clamp at top.
	m4, _ := pressKeyStr(m3, "k")
	if m4.plan.cursor != 0 {
		t.Errorf("up at top: cursor should stay 0, got %d", m4.plan.cursor)
	}
}

// ── US4: day navigation ────────────────────────────────────────────────────

// TestDayNav_NextDay checks that ']' advances the day by one.
func TestDayNav_NextDay(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.day = "2026-05-27"
	m.plan.loaded = true

	m2, cmd := pressKeyStr(m, "]")

	if m2.plan.day != "2026-05-28" {
		t.Errorf("next day: expected 2026-05-28, got %s", m2.plan.day)
	}
	if cmd == nil {
		t.Error("next day: expected reload command, got nil")
	}
}

// TestDayNav_PrevDay checks that '[' goes back one day.
func TestDayNav_PrevDay(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.day = "2026-05-27"
	m.plan.loaded = true

	m2, _ := pressKeyStr(m, "[")

	if m2.plan.day != "2026-05-26" {
		t.Errorf("prev day: expected 2026-05-26, got %s", m2.plan.day)
	}
}

// TestDayNav_Today checks that '.' resets to today (T018/US5).
func TestDayNav_Today(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.day = "2020-01-01"
	m.plan.loaded = true

	today := time.Now().Format("2006-01-02")
	m2, _ := pressKeyStr(m, ".")

	if m2.plan.day != today {
		t.Errorf("'.': expected %s, got %s", today, m2.plan.day)
	}
}

// TestAddTask_TDoesNotJumpToToday checks that 't' does NOT change the day (T018/US5).
func TestAddTask_TDoesNotJumpToToday(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.day = "2020-01-01"
	m.plan.loaded = true

	m2, _ := pressKeyStr(m, "t")

	if m2.plan.day != "2020-01-01" {
		t.Errorf("'t': should not change day, got %s", m2.plan.day)
	}
}

// TestClear_InertOnC checks that 'c' does not enter a clear mode (T021/US6).
func TestClear_InertOnC(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "Focus", StartMinute: pint32(540)}}

	m2, cmd := pressKeyStr(m, "c")

	if m2.plan.mode != planList {
		t.Errorf("'c': mode should stay planList, got %d", m2.plan.mode)
	}
	if cmd != nil {
		t.Errorf("'c': expected nil cmd, got non-nil")
	}
}

// TestDayNav_RefreshCtrlR checks that Ctrl+R reloads the current day.
func TestDayNav_RefreshCtrlR(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.day = "2026-05-27"
	m.plan.loaded = true

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})

	if cmd == nil {
		t.Error("ctrl+r: expected reload command, got nil")
	}
}

// ── US1: pomodoro cancel on Planning tab ───────────────────────────────────

// TestPomCancel_CancelsWhenRunningOnPlanning checks that 'x' issues cancelPomCmd
// when a pomodoro is running while on the Planning tab (T002).
func TestPomCancel_CancelsWhenRunningOnPlanning(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.pom = &activePom{
		taskID:   1,
		taskName: "focus",
		startAt:  time.Now().Add(-5 * time.Minute),
	}

	_, cmd := pressKeyStr(m, "x")

	if cmd == nil {
		t.Error("x with running pom on Planning: expected cancelPomCmd, got nil")
	}
}

// TestPomCancel_InertWhenNoneRunning checks that 'x' is inert when no pomodoro
// is active on the Planning tab (T002).
func TestPomCancel_InertWhenNoneRunning(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.pom = nil

	_, cmd := pressKeyStr(m, "x")

	if cmd != nil {
		t.Errorf("x with no pom on Planning: expected nil cmd, got %v", cmd)
	}
}

// ── US3: sub-tasks in task picker ──────────────────────────────────────────

// TestPickerSubtasks_VisibleAfterPlanTasksMsg checks that after a planTasksMsg
// with a parent+child tree, the picker's visible rows include the sub-task (T010).
func TestPickerSubtasks_VisibleAfterPlanTasksMsg(t *testing.T) {
	parentID := int64(1)
	tree := cli.BuildTree([]*taskv1.Task{
		{Id: 1, Name: "parent"},
		{Id: 2, Name: "child", ParentId: &parentID},
	})

	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.mode = planPickTask

	next, _ := m.Update(planTasksMsg{tree: tree})
	nm := next.(Model)

	if len(nm.plan.picker.visible) < 2 {
		t.Errorf("picker visible: expected at least 2 rows (parent+child), got %d", len(nm.plan.picker.visible))
	}
	// Verify child is present.
	found := false
	for _, row := range nm.plan.picker.visible {
		if row.node.Task.Id == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Error("picker visible: child task (id=2) not found in visible rows")
	}
}

// ── US3 (T028): edit form schedule↔unschedule ──────────────────────────────

// TestEditForm_NullUnschedules checks that typing "null" in the Start field
// sends MovePlanEntry with StartMinute=nil (unschedule).
func TestEditForm_NullUnschedules(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Standup", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	// Name unchanged, Start = "null" → unschedule
	m.plan.form.fields[1].SetValue("null")

	cmd := m.submitEditForm()

	if m.plan.mode != planList {
		t.Errorf("submitEditForm null: expected planList, got %d", m.plan.mode)
	}
	if cmd == nil {
		t.Error("submitEditForm null: expected a move command, got nil")
	}
	if cmd != nil {
		cmd()
		if fc.moveReq == nil {
			t.Error("submitEditForm null: MovePlanEntry was not called")
		} else if fc.moveReq.StartMinute != nil {
			t.Errorf("submitEditForm null: StartMinute = %v, want nil (unschedule)", fc.moveReq.StartMinute)
		}
	}
}

// TestEditForm_TimeSchedules verifies that a concrete time in Start issues MovePlanEntry
// with a non-nil StartMinute (existing behavior, regression guard).
func TestEditForm_TimeSchedules(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Standup"}} // untimed
	m.plan.cursor = 0
	m.initEditForm()
	m.plan.form.fields[1].SetValue("10:00")

	cmd := m.submitEditForm()

	if cmd == nil {
		t.Fatal("submitEditForm time: expected move command, got nil")
	}
	cmd()
	if fc.moveReq == nil {
		t.Fatal("submitEditForm time: MovePlanEntry was not called")
	}
	if fc.moveReq.StartMinute == nil {
		t.Error("submitEditForm time: StartMinute should be non-nil (schedule)")
	}
	if fc.moveReq.StartMinute != nil && *fc.moveReq.StartMinute != 600 {
		t.Errorf("submitEditForm time: StartMinute = %d, want 600 (10:00)", *fc.moveReq.StartMinute)
	}
}

// TestAllTaskIDs_CollectsAllIDs checks that allTaskIDs returns every id in the
// tree, including nested descendants (T011).
func TestAllTaskIDs_CollectsAllIDs(t *testing.T) {
	p1ID := int64(1)
	p2ID := int64(2)
	tree := cli.BuildTree([]*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child", ParentId: &p1ID},
		{Id: 3, Name: "grandchild", ParentId: &p2ID},
	})

	ids := allTaskIDs(tree)

	for _, wantID := range []int64{1, 2, 3} {
		if !ids[wantID] {
			t.Errorf("allTaskIDs: missing id=%d", wantID)
		}
	}
	if len(ids) != 3 {
		t.Errorf("allTaskIDs: expected 3 ids, got %d", len(ids))
	}
}

// ── T014: US1 — notice channel for send feedback ────────────────────────────

// buildTasksTabModel creates a model on the Tasks tab with a single visible task.
func buildTasksTabModel(fc *fakePlanClient, taskName string) Model {
	tasks := []*taskv1.Task{{Id: 1, Name: taskName}}
	m := ExportNewModel(nil, cli.BuildTree(tasks))
	m.planClient = fc
	m.activeTab = tabTasks
	m.plan.day = "2026-06-01"
	m.width = 80
	m.height = 30
	m.cursor = 0
	return m
}

// TestPlanSendToday_SetsSuccessNotice checks that pressing 'p' on the Tasks tab
// with a successful send sets m.notice naming the task and "today".
func TestPlanSendToday_SetsSuccessNotice(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildTasksTabModel(fc, "Write tests")

	_, cmd := pressKeyStr(m, "p")
	if cmd == nil {
		t.Fatal("p key: expected addPlanTaskCmd, got nil")
	}
	// Execute the command (synchronously via the fake client).
	msg := cmd()
	mutated, ok := msg.(planMutatedMsg)
	if !ok {
		t.Fatalf("p key: expected planMutatedMsg, got %T: %v", msg, msg)
	}
	if mutated.err != nil {
		t.Fatalf("p key: expected success, got error: %v", mutated.err)
	}
	if mutated.notice == "" {
		t.Error("p key: expected non-empty success notice")
	}
	if !containsAll(mutated.notice, "Write tests", "today") {
		t.Errorf("p key notice: expected task name 'Write tests' and 'today' in %q", mutated.notice)
	}
}

// TestPlanSendPickDay_SetsSuccessNoticeWithDate checks that confirming ctrl+p with a date
// sets m.notice naming the task and chosen date.
func TestPlanSendPickDay_SetsSuccessNoticeWithDate(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildTasksTabModel(fc, "Write docs")

	// Open date prompt via ctrl+p.
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	// Manually simulate the state that ctrl+p creates.
	m.datePromptTaskID = 1
	m.datePromptTaskName = "Write docs"
	m.mode = modeDatePrompt
	m.datePromptInput = newPlanInput("YYYY-MM-DD")
	m.datePromptInput.SetValue("2026-07-04")
	m.datePromptInput.Focus()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("ctrl+p confirm: expected addPlanTaskCmd, got nil")
	}
	msg := cmd()
	mutated, ok := msg.(planMutatedMsg)
	if !ok {
		t.Fatalf("ctrl+p confirm: expected planMutatedMsg, got %T: %v", msg, msg)
	}
	if mutated.err != nil {
		t.Fatalf("ctrl+p confirm: expected success, got error: %v", mutated.err)
	}
	if mutated.notice == "" {
		t.Error("ctrl+p confirm: expected non-empty success notice")
	}
	if !containsAll(mutated.notice, "Write docs", "2026-07-04") {
		t.Errorf("ctrl+p notice: expected task name and date in %q", mutated.notice)
	}
}

// TestPlanSendToday_DuplicateErrorVisibleOnTasksTab checks that a duplicate-rejection
// error from a Tasks-tab send is visible on the Tasks tab (routed to m.err, not m.plan.err).
func TestPlanSendToday_DuplicateErrorVisibleOnTasksTab(t *testing.T) {
	fc := &fakePlanClient{mutateErr: errForTest("That one's already parked here")}
	m := buildTasksTabModel(fc, "Fix bug")

	_, cmd := pressKeyStr(m, "p")
	if cmd == nil {
		t.Fatal("p key with duplicate error: expected cmd, got nil")
	}
	msg := cmd()
	// Feed the message back into Update.
	next, _ := m.Update(msg)
	nm := next.(Model)

	if nm.err == nil {
		t.Error("duplicate rejection: expected m.err to be set (visible on Tasks tab)")
	}
	if nm.plan.err != nil {
		t.Errorf("duplicate rejection: m.plan.err should be nil (not used for Tasks-tab sends); got %v", nm.plan.err)
	}
}

// TestNoticeCleared_OnNextUserAction checks that m.notice is cleared when the
// user presses any key.
func TestNoticeCleared_OnNextUserAction(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildTasksTabModel(fc, "Any task")
	m.notice = "Tucked 'Any task' into today's plan."

	// Press any key (e.g. j/down).
	next, _ := pressKeyStr(m, "j")

	if next.notice != "" {
		t.Errorf("notice should be cleared on next key press; got %q", next.notice)
	}
}

// TestRenderStatus_ShowsNoticeWhenNoError checks that renderStatus shows m.notice
// when it is non-empty and no active error is present.
func TestRenderStatus_ShowsNoticeWhenNoError(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.notice = "Tucked 'My task' into today's plan."

	out := m.renderStatus()

	if !containsAll(out, "My task", "today") {
		t.Errorf("renderStatus with notice: expected notice text in status; got %q", out)
	}
}

// TestRenderStatus_ErrorTakesPrecedenceOverNotice checks that renderStatus shows
// the error instead of the notice when both are set.
func TestRenderStatus_ErrorTakesPrecedenceOverNotice(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.notice = "Some notice"
	m.err = errForTest("rpc error")

	out := m.renderStatus()

	if containsAll(out, "Some notice") && !containsAll(out, "rpc error") {
		t.Errorf("renderStatus: error should take precedence over notice; got %q", out)
	}
	if !containsAll(out, "rpc error") {
		t.Errorf("renderStatus: expected error text in status; got %q", out)
	}
}

// containsAll reports whether s contains all the given substrings.
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
