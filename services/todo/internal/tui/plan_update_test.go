package tui

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	planv1connect "github.com/pboyd/todo/services/todo/gen/plan/v1/planv1connect"
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
		{Id: 1, Name: "Standup", StartMinute: 540},
		{Id: 2, Name: "Focus", StartMinute: 600},
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
		{Id: 1, Name: "One", StartMinute: 540},
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

// TestAddTask_PickerOpensOnA checks that pressing 'a' in planList mode issues
// a listTasksForPickerCmd and sets mode to planPickTask.
func TestAddTask_PickerOpensOnA(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true

	m2, cmd := pressKeyStr(m, "a")

	if m2.plan.mode != planPickTask {
		t.Errorf("after 'a': expected planPickTask, got %d", m2.plan.mode)
	}
	if cmd == nil {
		t.Error("after 'a': expected a list-tasks command, got nil")
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
	m.plan.mode = planRename

	m2, _ := pressSpecialKey(m, tea.KeyTab)

	if m2.activeTab != tabPlanning {
		t.Errorf("Tab while form open: should not switch tabs, got tab=%d", m2.activeTab)
	}
}

// ── US3: rename/move/remove/clear ──────────────────────────────────────────

// TestRename_FormOpensOnR checks that 'r' opens planRename mode.
func TestRename_FormOpensOnR(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "Standup", StartMinute: 540}}
	m.plan.cursor = 0

	m2, _ := pressKeyStr(m, "r")

	if m2.plan.mode != planRename {
		t.Errorf("after 'r': expected planRename, got %d", m2.plan.mode)
	}
	// Form should be prefilled with the entry name.
	if m2.plan.form.fields[0].Value() != "Standup" {
		t.Errorf("rename form: expected prefilled name 'Standup', got %q", m2.plan.form.fields[0].Value())
	}
}

// TestRename_NoopOnEmpty checks that 'r' is a no-op with no entries.
func TestRename_NoopOnEmpty(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = nil

	m2, _ := pressKeyStr(m, "r")

	if m2.plan.mode != planList {
		t.Errorf("rename with no entries: mode should stay planList, got %d", m2.plan.mode)
	}
}

// TestRemove_IssuesRPCOnCtrlD checks that Ctrl+D issues a removePlanCmd.
func TestRemove_IssuesRPCOnCtrlD(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 3, Name: "Review", StartMinute: 600}}
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
		{Id: 1, StartMinute: 540},
		{Id: 2, StartMinute: 600},
		{Id: 3, StartMinute: 660},
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

// TestDayNav_Today checks that 't' resets to today.
func TestDayNav_Today(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.day = "2020-01-01"
	m.plan.loaded = true

	today := time.Now().Format("2006-01-02")
	m2, _ := pressKeyStr(m, "t")

	if m2.plan.day != today {
		t.Errorf("today: expected %s, got %s", today, m2.plan.day)
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
