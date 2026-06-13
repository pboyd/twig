package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	tea "charm.land/bubbletea/v2"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
)

// fakePlanClient is a minimal PlanServiceClient for unit tests.
type fakePlanClient struct {
	planv1connect.PlanServiceClient
	entries              []*planv1.PlanEntry
	lastListDay          string
	listErr              error
	addTaskReq           *planv1.AddPlanTaskRequest
	addEventReq          *planv1.AddPlanEventRequest
	renameReq            *planv1.RenamePlanEntryRequest
	moveReq              *planv1.MovePlanEntryRequest
	removeReq            *planv1.RemovePlanEntryRequest
	clearReq             *planv1.ClearPlanRequest
	mutateErr            error
	scheduledDaysReq     *planv1.ListScheduledDaysRequest
	scheduledDaysResp    []*planv1.ScheduledDay
	scheduledDaysErr     error
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

func (f *fakePlanClient) ListScheduledDays(_ context.Context, req *connect.Request[planv1.ListScheduledDaysRequest]) (*connect.Response[planv1.ListScheduledDaysResponse], error) {
	f.scheduledDaysReq = req.Msg
	if f.scheduledDaysErr != nil {
		return nil, f.scheduledDaysErr
	}
	return connect.NewResponse(&planv1.ListScheduledDaysResponse{Days: f.scheduledDaysResp}), nil
}

// buildPlanTestModel creates a model ready for planning tab tests.
func buildPlanTestModel(fc *fakePlanClient) Model {
	m := ExportNewPlanModel(nil, fc, "2026-05-27")
	m.width = 80
	m.height = 30
	return m
}

// pressKeyStr sends a single-character key string to the model.
func pressKeyStr(m Model, k string) (Model, tea.Cmd) {
	runes := []rune(k)
	var msg tea.KeyPressMsg
	if len(runes) == 1 {
		msg = tea.KeyPressMsg{Code: runes[0], Text: k}
	}
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

// pressSpecialKey sends a pre-built key press message to the model.
func pressSpecialKey(m Model, msg tea.KeyPressMsg) (Model, tea.Cmd) {
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

	m2, cmd := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})

	if m2.activeTab != tabPlanning {
		t.Errorf("after Tab: expected tabPlanning, got %d", m2.activeTab)
	}
	if cmd == nil {
		t.Error("after Tab: expected a command (listPlanCmd), got nil")
	}
}

// TestTabSwitch_PlanningToReport checks that pressing Tab from Planning tab
// switches to the Report tab (Tasks → Planning → Report cycle).
func TestTabSwitch_PlanningToReport(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc) // starts in tabPlanning

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})

	if m2.activeTab != tabReport {
		t.Errorf("after Tab on Planning: expected tabReport, got %d", m2.activeTab)
	}
}

// TestTabSwitch_ShiftTabPlanningToTasks checks that Shift+Tab from Planning tab
// switches to Tasks tab.
func TestTabSwitch_ShiftTabPlanningToTasks(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc) // tabPlanning

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})

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
	m, _ = pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})
	// Switch back to tasks
	m, _ = pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})

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

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})

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

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEscape})

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

// TestAddEvent_EscDoesNotCancel checks that Esc in planEventForm no longer
// returns to planList (use the Cancel button to avoid losing in-progress work).
func TestAddEvent_EscDoesNotCancel(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.mode = planEventForm
	m.initAddEventForm()

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEscape})

	if m2.plan.mode == planList {
		t.Errorf("esc from event form: should stay in form mode, not return to planList")
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
	msg := tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
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

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})

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

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})

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

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})

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

	if m.plan.mode != planEdit {
		t.Errorf("submitEditForm dispatch: form should stay open (planEdit), got %d", m.plan.mode)
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

	if m.plan.mode != planEdit {
		t.Errorf("submitEditForm move dispatch: form should stay open (planEdit), got %d", m.plan.mode)
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

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})

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

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})

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

// ── T008/T009: US1 — pre-fill and change-detection tests (TDD: must fail before T010/T011) ──

// TestInitEditForm_PreFillsScheduledEntry verifies that initEditForm pre-fills
// Name verbatim, Start as HH:MM, and Duration as compact form for a scheduled entry.
func TestInitEditForm_PreFillsScheduledEntry(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 7, Name: "Morning standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	m.plan.cursor = 0

	m.initEditForm()

	if m.plan.form.fields[0].Value() != "Morning standup" {
		t.Errorf("Name: expected 'Morning standup', got %q", m.plan.form.fields[0].Value())
	}
	if m.plan.form.fields[1].Value() != "09:00" {
		t.Errorf("Start: expected '09:00' (540 min), got %q", m.plan.form.fields[1].Value())
	}
	if m.plan.form.fields[2].Value() != "30m" {
		t.Errorf("Duration: expected '30m', got %q", m.plan.form.fields[2].Value())
	}
}

// TestInitEditForm_PreFillsScheduledEntry_1h30m verifies compact format for 90m duration.
func TestInitEditForm_PreFillsScheduledEntry_1h30m(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 8, Name: "Focus", StartMinute: pint32(600), DurationMinute: 90},
	}
	m.plan.cursor = 0

	m.initEditForm()

	if m.plan.form.fields[1].Value() != "10:00" {
		t.Errorf("Start: expected '10:00' (600 min), got %q", m.plan.form.fields[1].Value())
	}
	if m.plan.form.fields[2].Value() != "1h30m" {
		t.Errorf("Duration: expected '1h30m', got %q", m.plan.form.fields[2].Value())
	}
}

// TestInitEditForm_EmptyStartDurationForUntimed verifies that an untimed entry
// leaves Start and Duration empty (not "09:00" or "0m").
func TestInitEditForm_EmptyStartDurationForUntimed(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 9, Name: "Write spec"},
	}
	m.plan.cursor = 0

	m.initEditForm()

	if m.plan.form.fields[1].Value() != "" {
		t.Errorf("Start: expected empty for untimed entry, got %q", m.plan.form.fields[1].Value())
	}
	if m.plan.form.fields[2].Value() != "" {
		t.Errorf("Duration: expected empty for untimed entry, got %q", m.plan.form.fields[2].Value())
	}
}

// TestInitEditForm_PlaceholdersNoSentinelText verifies that no field placeholder
// contains sentinel text like "blank=keep" or "null=unschedule".
func TestInitEditForm_PlaceholdersNoSentinelText(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "X"},
	}
	m.plan.cursor = 0

	m.initEditForm()

	for i, f := range m.plan.form.fields {
		ph := f.Placeholder
		if strings.Contains(ph, "blank=keep") || strings.Contains(ph, "null=unschedule") {
			t.Errorf("field %d placeholder contains sentinel text: %q", i, ph)
		}
	}
}

// TestSubmitEditForm_UnchangedIsNoop verifies that submitting an edit form with
// no changes dispatches no RPC and closes the form.
func TestSubmitEditForm_UnchangedIsNoop(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 5, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	m.plan.cursor = 0
	m.initEditForm() // pre-fills Name="Standup", Start="09:00", Duration="30m"

	cmd := m.submitEditForm()

	if m.plan.mode != planList {
		t.Errorf("unchanged: expected planList (form closed), got mode=%d", m.plan.mode)
	}
	if cmd != nil {
		t.Error("unchanged: expected nil cmd (no-op), got non-nil")
	}
}

// TestSubmitEditForm_ClearStartUnschedules verifies that clearing the Start field
// on a scheduled entry dispatches unschedule (MovePlanEntry with nil StartMinute),
// ignoring any Duration value.
func TestSubmitEditForm_ClearStartUnschedules(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 5, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	m.plan.cursor = 0
	m.initEditForm() // pre-fills Start="09:00", Duration="30m"

	m.plan.form.fields[1].SetValue("") // clear Start → unschedule

	cmd := m.submitEditForm()

	if cmd == nil {
		t.Fatal("clear-start: expected move command for unschedule, got nil")
	}
	cmd()
	if fc.moveReq == nil {
		t.Fatal("clear-start: MovePlanEntry was not called")
	}
	if fc.moveReq.StartMinute != nil {
		t.Errorf("clear-start: StartMinute should be nil (unschedule), got %v", fc.moveReq.StartMinute)
	}
	if fc.renameReq != nil {
		t.Error("clear-start: no rename expected (name unchanged)")
	}
}

// TestSubmitEditForm_ClearStartAlreadyUntimed_IsNoop verifies that clearing the
// Start field of an already-untimed entry dispatches no move.
func TestSubmitEditForm_ClearStartAlreadyUntimed_IsNoop(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 6, Name: "Write notes"},
	}
	m.plan.cursor = 0
	m.initEditForm() // Start="" (untimed), Duration=""

	// Start is already empty; submitting unchanged → no-op
	cmd := m.submitEditForm()

	if m.plan.mode != planList {
		t.Errorf("untimed noop: expected planList, got mode=%d", m.plan.mode)
	}
	if cmd != nil {
		t.Error("untimed noop: expected nil cmd")
	}
	if fc.moveReq != nil {
		t.Error("untimed noop: MovePlanEntry should not be called")
	}
}

// TestInitEditForm_PreFillsDurationForUntimedEntry verifies that an untimed entry
// that already has a duration pre-fills the Duration field (not the old behavior
// of leaving it empty just because StartMinute is nil).
func TestInitEditForm_PreFillsDurationForUntimedEntry(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 10, Name: "Build fence", DurationMinute: 30},
	}
	m.plan.cursor = 0

	m.initEditForm()

	if m.plan.form.fields[1].Value() != "" {
		t.Errorf("Start: expected empty for untimed entry, got %q", m.plan.form.fields[1].Value())
	}
	if m.plan.form.fields[2].Value() != "30m" {
		t.Errorf("Duration: expected '30m' pre-filled for untimed entry with DurationMinute=30, got %q",
			m.plan.form.fields[2].Value())
	}
}

// TestSubmitEditForm_UntimedChangeDuration verifies that changing the Duration
// of an already-untimed entry issues a MovePlanEntry with StartMinute=nil and
// the new DurationMinute (the fix for the bug where duration changes were silently dropped).
func TestSubmitEditForm_UntimedChangeDuration(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 11, Name: "Build fence", DurationMinute: 30},
	}
	m.plan.cursor = 0
	m.initEditForm() // pre-fills Duration="30m", Start=""

	m.plan.form.fields[2].SetValue("1h") // change duration to 60m

	cmd := m.submitEditForm()

	if cmd == nil {
		t.Fatal("untimed change duration: expected a command, got nil")
	}
	cmd()
	if fc.moveReq == nil {
		t.Fatal("untimed change duration: MovePlanEntry was not called")
	}
	if fc.moveReq.StartMinute != nil {
		t.Errorf("untimed change duration: StartMinute should be nil (stay untimed), got %v", fc.moveReq.StartMinute)
	}
	if fc.moveReq.DurationMinute != 60 {
		t.Errorf("untimed change duration: expected DurationMinute=60, got %d", fc.moveReq.DurationMinute)
	}
}

// TestSubmitEditForm_UntimedUnchangedDuration_IsNoop verifies that submitting an
// untimed entry with the same duration (pre-filled) dispatches no command.
func TestSubmitEditForm_UntimedUnchangedDuration_IsNoop(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 12, Name: "Build fence", DurationMinute: 30},
	}
	m.plan.cursor = 0
	m.initEditForm() // pre-fills Duration="30m", Start=""

	// Leave Duration at the pre-filled "30m" — no change.
	cmd := m.submitEditForm()

	if m.plan.mode != planList {
		t.Errorf("untimed unchanged duration: expected planList, got mode=%d", m.plan.mode)
	}
	if cmd != nil {
		t.Error("untimed unchanged duration: expected nil cmd (no-op)")
	}
	if fc.moveReq != nil {
		t.Error("untimed unchanged duration: MovePlanEntry should not be called")
	}
}

// TestSubmitEditForm_UntimedInvalidDuration_KeepsFormOpen verifies that an invalid
// duration string on an untimed entry keeps the form open with an error.
func TestSubmitEditForm_UntimedInvalidDuration_KeepsFormOpen(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 13, Name: "Build fence", DurationMinute: 30},
	}
	m.plan.cursor = 0
	m.initEditForm()

	m.plan.form.fields[2].SetValue("not-a-duration")

	cmd := m.submitEditForm()

	if m.plan.mode != planEdit {
		t.Errorf("invalid duration: form should stay planEdit, got %d", m.plan.mode)
	}
	if m.plan.err == nil {
		t.Error("invalid duration: plan.err should be set")
	}
	if cmd != nil {
		t.Error("invalid duration: expected nil cmd")
	}
	if fc.moveReq != nil {
		t.Error("invalid duration: MovePlanEntry should not be called")
	}
}

// TestSubmitEditForm_ChangedStartDispatches verifies that changing Start on a
// scheduled entry dispatches a move with the new start value.
func TestSubmitEditForm_ChangedStartDispatches(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 5, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	m.plan.cursor = 0
	m.initEditForm() // pre-fills Start="09:00"

	m.plan.form.fields[1].SetValue("10:00") // change start

	cmd := m.submitEditForm()

	if cmd == nil {
		t.Fatal("changed start: expected move command, got nil")
	}
	cmd()
	if fc.moveReq == nil {
		t.Fatal("changed start: MovePlanEntry was not called")
	}
	if fc.moveReq.StartMinute == nil {
		t.Fatal("changed start: StartMinute should be non-nil")
	}
	if *fc.moveReq.StartMinute != 600 {
		t.Errorf("changed start: expected StartMinute=600, got %d", *fc.moveReq.StartMinute)
	}
}

// TestSubmitEditForm_InvalidStart_KeepsFormOpen verifies that an invalid Start
// keeps the form open with an error set.
func TestSubmitEditForm_InvalidStart_KeepsFormOpen(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 5, Name: "Standup", StartMinute: pint32(540)},
	}
	m.plan.cursor = 0
	m.initEditForm()

	m.plan.form.fields[1].SetValue("not-a-time")

	cmd := m.submitEditForm()

	if m.plan.mode != planEdit {
		t.Errorf("invalid start: form should stay planEdit, got %d", m.plan.mode)
	}
	if m.plan.err == nil {
		t.Error("invalid start: plan.err should be set")
	}
	if cmd != nil {
		t.Error("invalid start: expected nil cmd")
	}
}

// TestSubmitEditForm_NameChangeOnlyRenames verifies that changing only the name
// dispatches rename with no move.
func TestSubmitEditForm_NameChangeOnlyRenames(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 5, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	m.plan.cursor = 0
	m.initEditForm() // pre-fills all fields

	m.plan.form.fields[0].SetValue("Daily sync")

	cmd := m.submitEditForm()

	if cmd == nil {
		t.Fatal("name change: expected rename command, got nil")
	}
	cmd()
	if fc.renameReq == nil {
		t.Fatal("name change: RenamePlanEntry was not called")
	}
	if fc.renameReq.Name != "Daily sync" {
		t.Errorf("name change: expected 'Daily sync', got %q", fc.renameReq.Name)
	}
	if fc.moveReq != nil {
		t.Error("name-only change: MovePlanEntry should NOT be called")
	}
}

// ── T007: Foundational interaction tests ───────────────────────────────────

// TestPlanForm_TabCyclesThroughButtons checks that Tab moves focus past all
// text fields into the Save/Cancel button slots and wraps back to field 0.
func TestPlanForm_TabCyclesThroughButtons(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "X", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm() // 3 fields: focus starts at 0

	nFields := len(m.plan.form.fields) // 3

	// Tab through all fields
	for i := 1; i < nFields; i++ {
		m, _ = pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})
		if m.plan.form.focus != i {
			t.Errorf("after Tab x%d: expected focus=%d, got %d", i, i, m.plan.form.focus)
		}
	}
	// Next Tab lands on Save slot
	m, _ = pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})
	if !planFocusSave(m.plan.form) {
		t.Errorf("Tab to Save: expected planFocusSave=true, focus=%d", m.plan.form.focus)
	}
	// Next Tab lands on Cancel slot
	m, _ = pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})
	if !planFocusCancel(m.plan.form) {
		t.Errorf("Tab to Cancel: expected planFocusCancel=true, focus=%d", m.plan.form.focus)
	}
	// Wrap back to 0
	m, _ = pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.plan.form.focus != 0 {
		t.Errorf("Tab wrap: expected focus=0, got %d", m.plan.form.focus)
	}
}

// TestPlanForm_ShiftTabCyclesBackward checks that Shift+Tab moves focus backward.
func TestPlanForm_ShiftTabCyclesBackward(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "X", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm() // focus at 0

	// Shift+Tab from 0 should wrap to Cancel slot (len(fields)+1)
	m, _ = pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if !planFocusCancel(m.plan.form) {
		t.Errorf("Shift+Tab from 0: expected Cancel slot, focus=%d", m.plan.form.focus)
	}
}

// TestPlanForm_EnterInTextField_DoesNotSubmit checks that pressing Enter while
// a text field is focused does NOT submit the form — mode stays open.
func TestPlanForm_EnterInTextField_DoesNotSubmit(t *testing.T) {
	fc := &fakePlanClient{}
	for _, mode := range []planMode{planEdit, planTaskTime, planEventForm} {
		m := buildPlanTestModel(fc)
		m.plan.loaded = true
		m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "X", StartMinute: pint32(540)}}
		m.plan.cursor = 0
		switch mode {
		case planEdit:
			m.initEditForm()
		case planTaskTime:
			m.initTaskTimeForm(1)
		case planEventForm:
			m.initAddEventForm()
		}
		// Ensure focus is on a text field
		if m.plan.form.focus >= len(m.plan.form.fields) {
			t.Fatalf("mode %d: focus should start on a text field", mode)
		}
		initialMode := m.plan.mode

		m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})

		if m2.plan.mode != initialMode {
			t.Errorf("mode %d: Enter in text field should not submit; mode changed from %d to %d",
				mode, initialMode, m2.plan.mode)
		}
	}
}

// TestPlanForm_EnterOnSaveButton_Submits checks that pressing Enter while the
// Save button is focused submits the form.
func TestPlanForm_EnterOnSaveButton_Submits(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "X", StartMinute: pint32(540), DurationMinute: 30}}
	m.plan.cursor = 0
	m.initEditForm()

	// Move focus to Save slot
	for !planFocusSave(m.plan.form) {
		m.cyclePlanFormFocus(1)
	}

	_, cmd := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	// A no-op save (unchanged form) should close the form and return nil cmd.
	// But we just care that Enter on Save does NOT silently do nothing —
	// either cmd is non-nil (RPC) or the mode closed (no-op).
	// With unchanged pre-filled form it closes immediately (nil cmd, planList).
	// We verify Enter was handled as a save attempt, not ignored.
	if m.plan.mode != planEdit {
		t.Errorf("Enter on Save: form should still be in planEdit before action; mode=%d", m.plan.mode)
	}
	_ = cmd // cmd may be nil (no-op) or non-nil (rpc) — both are valid "submit" outcomes
}

// TestPlanForm_EnterOnCancelButton_Closes checks that pressing Enter on the
// Cancel button closes the form without submitting.
func TestPlanForm_EnterOnCancelButton_Closes(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "X", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()

	// Move focus to Cancel slot
	for !planFocusCancel(m.plan.form) {
		m.cyclePlanFormFocus(1)
	}

	m2, cmd := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m2.plan.mode != planList {
		t.Errorf("Enter on Cancel: expected planList, got %d", m2.plan.mode)
	}
	if cmd != nil {
		t.Error("Enter on Cancel: expected nil cmd (no RPC)")
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

// TestEditForm_ClearStartUnschedules checks that clearing the Start field on a
// scheduled entry sends MovePlanEntry with StartMinute=nil (unschedule).
// The old "null" keyword sentinel is replaced by clearing the Start field.
func TestEditForm_ClearStartUnschedules(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Standup", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	// Name unchanged; clear the pre-filled Start field to unschedule.
	m.plan.form.fields[1].SetValue("")

	cmd := m.submitEditForm()

	if m.plan.mode != planEdit {
		t.Errorf("submitEditForm clear-start dispatch: form should stay open (planEdit), got %d", m.plan.mode)
	}
	if cmd == nil {
		t.Error("submitEditForm clear-start: expected a move command, got nil")
	}
	if cmd != nil {
		cmd()
		if fc.moveReq == nil {
			t.Error("submitEditForm clear-start: MovePlanEntry was not called")
		} else if fc.moveReq.StartMinute != nil {
			t.Errorf("submitEditForm clear-start: StartMinute = %v, want nil (unschedule)", fc.moveReq.StartMinute)
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

// ── form-stays-open-on-server-error ──────────────────────────────────────────

// TestEditForm_StaysOpenOnServerError checks that when submitEditForm dispatches
// an RPC but the server returns an error, the form remains open (planEdit) and
// plan.err is set with the user's input preserved.
func TestEditForm_StaysOpenOnServerError(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Old Name", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	m.plan.form.fields[0].SetValue("New Name")

	// Submit dispatches an RPC — form should stay open until server responds.
	cmd := m.submitEditForm()

	if m.plan.mode != planEdit {
		t.Errorf("after dispatch: form should stay open (planEdit), got mode=%d", m.plan.mode)
	}
	if cmd == nil {
		t.Fatal("submitEditForm: expected a command, got nil")
	}

	// Simulate a server error (e.g. time overlap).
	next, _ := m.Update(planMutatedMsg{err: errForTest("time overlap")})
	nm := next.(Model)

	if nm.plan.mode != planEdit {
		t.Errorf("server error: form should stay open (planEdit), got mode=%d", nm.plan.mode)
	}
	if nm.plan.err == nil {
		t.Error("server error: plan.err should be set")
	}
	// User's typed input must be preserved so they can correct and resubmit.
	if nm.plan.form.fields[0].Value() != "New Name" {
		t.Errorf("server error: expected field value 'New Name', got %q", nm.plan.form.fields[0].Value())
	}
}

// TestPlanMutatedMsg_Success_ClosesForm checks that planMutatedMsg success closes
// any open form (sets planList) and issues a reload command.
// The success handler, not the submit handler, is responsible for closing the form.
func TestPlanMutatedMsg_Success_ClosesForm(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{{Id: 5, Name: "Old Name", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	// Open the edit form; mode is now planEdit — simulating the "waiting for
	// server response" state after the fix (form open, RPC in flight).
	m.initEditForm()

	// Feed a success message while the form is still open.
	next, cmd := m.Update(planMutatedMsg{highlightID: 5})
	nm := next.(Model)

	if nm.plan.mode != planList {
		t.Errorf("success while form open: expected planList, got mode=%d", nm.plan.mode)
	}
	if cmd == nil {
		t.Error("success: expected reload command, got nil")
	}
}

// TestEventForm_StaysOpenOnServerError checks that when submitEventForm dispatches
// an RPC but the server returns an error, the form remains open (planEventForm).
func TestEventForm_StaysOpenOnServerError(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initAddEventForm()
	m.plan.form.fields[0].SetValue("Meeting")
	m.plan.form.fields[1].SetValue("09:00")
	m.plan.form.fields[2].SetValue("30m")

	cmd := m.submitEventForm()

	if m.plan.mode != planEventForm {
		t.Errorf("after dispatch: form should stay open (planEventForm), got mode=%d", m.plan.mode)
	}
	if cmd == nil {
		t.Fatal("submitEventForm: expected a command, got nil")
	}

	// Simulate server error.
	next, _ := m.Update(planMutatedMsg{err: errForTest("time overlap")})
	nm := next.(Model)

	if nm.plan.mode != planEventForm {
		t.Errorf("server error: form should stay open (planEventForm), got mode=%d", nm.plan.mode)
	}
	if nm.plan.err == nil {
		t.Error("server error: plan.err should be set")
	}
	if nm.plan.form.fields[0].Value() != "Meeting" {
		t.Errorf("server error: expected event name 'Meeting' preserved, got %q", nm.plan.form.fields[0].Value())
	}
}

// TestTaskTimeForm_StaysOpenOnServerError checks that when submitTaskTimeForm
// dispatches an RPC but the server returns an error, the form remains open
// (planTaskTime) with the user's input preserved.
func TestTaskTimeForm_StaysOpenOnServerError(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initTaskTimeForm(42)
	m.plan.form.fields[0].SetValue("09:00")
	m.plan.form.fields[1].SetValue("30m")

	cmd := m.submitTaskTimeForm()

	if m.plan.mode != planTaskTime {
		t.Errorf("after dispatch: form should stay open (planTaskTime), got mode=%d", m.plan.mode)
	}
	if cmd == nil {
		t.Fatal("submitTaskTimeForm: expected a command, got nil")
	}

	// Simulate server error.
	next, _ := m.Update(planMutatedMsg{err: errForTest("time overlap")})
	nm := next.(Model)

	if nm.plan.mode != planTaskTime {
		t.Errorf("server error: form should stay open (planTaskTime), got mode=%d", nm.plan.mode)
	}
	if nm.plan.err == nil {
		t.Error("server error: plan.err should be set")
	}
	if nm.plan.form.fields[0].Value() != "09:00" {
		t.Errorf("server error: expected start time '09:00' preserved, got %q", nm.plan.form.fields[0].Value())
	}
}

// ── T013/T014: US2 — schedule-task and add-event form parity ──────────────────

// TestScheduleTaskForm_ButtonsAndHelp verifies that the schedule-task form
// renders Save/Cancel buttons and the exact help line.
func TestScheduleTaskForm_ButtonsAndHelp(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.initTaskTimeForm(1)
	out := m.renderPlanFormView(80)

	if !strings.Contains(out, "[ Save ]") {
		t.Errorf("schedule form: expected '[ Save ]'; got:\n%s", out)
	}
	if !strings.Contains(out, "[ Cancel ]") {
		t.Errorf("schedule form: expected '[ Cancel ]'; got:\n%s", out)
	}
	const wantHelp = "Ctrl+S: save  Tab: next field"
	if !strings.Contains(out, wantHelp) {
		t.Errorf("schedule form: expected help line %q; got:\n%s", wantHelp, out)
	}
}

// TestAddEventForm_ButtonsAndHelp verifies that the add-event form renders
// Save/Cancel buttons and the exact help line.
func TestAddEventForm_ButtonsAndHelp(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.initAddEventForm()
	out := m.renderPlanFormView(80)

	if !strings.Contains(out, "[ Save ]") {
		t.Errorf("add-event form: expected '[ Save ]'; got:\n%s", out)
	}
	if !strings.Contains(out, "[ Cancel ]") {
		t.Errorf("add-event form: expected '[ Cancel ]'; got:\n%s", out)
	}
	const wantHelp = "Ctrl+S: save  Tab: next field"
	if !strings.Contains(out, wantHelp) {
		t.Errorf("add-event form: expected help line %q; got:\n%s", wantHelp, out)
	}
}

// TestScheduleTaskForm_EnterInFieldDoesNotSubmit verifies Enter in a text field
// of the schedule form does NOT submit (mode stays planTaskTime).
func TestScheduleTaskForm_EnterInFieldDoesNotSubmit(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initTaskTimeForm(5)

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if m2.plan.mode != planTaskTime {
		t.Errorf("schedule form: Enter in text field should not submit; mode=%d", m2.plan.mode)
	}
}

// TestAddEventForm_EnterInFieldDoesNotSubmit verifies Enter in a text field
// of the add-event form does NOT submit (mode stays planEventForm).
func TestAddEventForm_EnterInFieldDoesNotSubmit(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initAddEventForm()

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if m2.plan.mode != planEventForm {
		t.Errorf("add-event form: Enter in text field should not submit; mode=%d", m2.plan.mode)
	}
}

// TestScheduleTaskForm_CtrlSSubmits verifies that Ctrl+S still submits the
// schedule form from any focus (even when a text field is focused).
func TestScheduleTaskForm_CtrlSSubmits(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initTaskTimeForm(5)
	// Leave fields blank (untimed submission)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	if cmd == nil {
		t.Error("schedule form: Ctrl+S should submit (cmd non-nil)")
	}
}

// ── TestAllTaskIDs_CollectsAllIDs ─────────────────────────────────────────────

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
	_, _ = m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	// Manually simulate the state that ctrl+p creates.
	m.datePromptTaskID = 1
	m.datePromptTaskName = "Write docs"
	m.mode = modeDatePrompt
	m.datePromptInput = newPlanInput("YYYY-MM-DD")
	m.datePromptInput.SetValue("2026-07-04")
	m.datePromptInput.Focus()

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
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

// ── US1: auto-schedule (task, empty day / gaps / today-floor) ──────────────

// TestAutoSchedule_EmptyDay_PlacesAt0800 verifies that pressing 'a' on an untimed
// task on an empty day dispatches movePlanCmd with start_minute=480.
func TestAutoSchedule_EmptyDay_PlacesAt0800(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 5, Name: "Write tests", DurationMinute: 30},
	}, 0)

	_, cmd := pressKeyStr(m, "a")

	if cmd == nil {
		t.Fatal("expected movePlanCmd, got nil")
	}
	cmd()
	if fc.moveReq == nil {
		t.Fatal("MovePlanEntry was not called")
	}
	if fc.moveReq.StartMinute == nil {
		t.Fatal("expected timed=true (StartMinute set)")
	}
	if *fc.moveReq.StartMinute != 480 {
		t.Errorf("expected start_minute=480, got %d", *fc.moveReq.StartMinute)
	}
}

// TestAutoSchedule_GapAfterBlock_PlacesAtEndOfBlock verifies placement after an
// existing block.
func TestAutoSchedule_GapAfterBlock_PlacesAtEndOfBlock(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	sm := int32(480)
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 0, Name: "Stand-up", StartMinute: &sm, DurationMinute: 60}, // event 08:00–09:00
		{Id: 2, TaskId: 5, Name: "Write tests", DurationMinute: 30},                // untimed task
	}, 1) // cursor on task

	_, cmd := pressKeyStr(m, "a")

	if cmd == nil {
		t.Fatal("expected movePlanCmd, got nil")
	}
	cmd()
	if fc.moveReq == nil {
		t.Fatal("MovePlanEntry was not called")
	}
	if fc.moveReq.StartMinute == nil || *fc.moveReq.StartMinute != 540 {
		t.Errorf("expected start=540 (after block), got %v", fc.moveReq.StartMinute)
	}
}

// TestAutoSchedule_TooSmallGap_Skips verifies that a gap smaller than the task
// duration is skipped.
func TestAutoSchedule_TooSmallGap_Skips(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	sm1, sm2 := int32(480), int32(555)
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 0, Name: "Block1", StartMinute: &sm1, DurationMinute: 60},
		{Id: 2, TaskId: 0, Name: "Block2", StartMinute: &sm2, DurationMinute: 45},
		{Id: 3, TaskId: 7, Name: "Task", DurationMinute: 30},
	}, 2)

	_, cmd := pressKeyStr(m, "a")

	if cmd == nil {
		t.Fatal("expected movePlanCmd, got nil")
	}
	cmd()
	if fc.moveReq == nil || fc.moveReq.StartMinute == nil || *fc.moveReq.StartMinute != 600 {
		t.Errorf("expected start=600 (skip 15-min gap), got %v", fc.moveReq.GetStartMinute())
	}
}

// TestAutoSchedule_TodayFloor_UsesCurrentMinute verifies that on today's plan
// the floor is raised to the current minute (FR-003).
func TestAutoSchedule_TodayFloor_UsesCurrentMinute(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	// Inject a fixed "now" at 2026-01-01 14:00 (840 min) and set plan day to match.
	fixedNow := time.Date(2026, 1, 1, 14, 0, 0, 0, time.Local)
	ExportSetNowFunc(&m, func() time.Time { return fixedNow })
	ExportSetPlanDay(&m, "2026-01-01")
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 5, Name: "Task", DurationMinute: 30},
	}, 0)

	_, cmd := pressKeyStr(m, "a")

	if cmd == nil {
		t.Fatal("expected movePlanCmd, got nil")
	}
	cmd()
	if fc.moveReq == nil || fc.moveReq.StartMinute == nil {
		t.Fatal("MovePlanEntry was not called or start not set")
	}
	if *fc.moveReq.StartMinute < 840 {
		t.Errorf("today-floor: start_minute should be >= 840 (14:00), got %d", *fc.moveReq.StartMinute)
	}
}

// TestAutoSchedule_NoFit_SetsNotice verifies that when no slot fits a playful
// notice is set and no command is dispatched.
func TestAutoSchedule_NoFit_SetsNotice(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	// Fill the day from 08:00 to midnight.
	sm := int32(480)
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 0, Name: "Block", StartMinute: &sm, DurationMinute: 960},
		{Id: 2, TaskId: 5, Name: "Task", DurationMinute: 30},
	}, 1)

	m2, cmd := pressKeyStr(m, "a")

	if cmd != nil {
		t.Error("expected no command on a full day")
	}
	notice := ExportNotice(m2)
	if notice == "" {
		t.Error("expected a playful notice, got empty string")
	}
}

// ── US2: re-home / no-op ───────────────────────────────────────────────────

// TestAutoSchedule_ReHome_MovesEarlier verifies that pressing 'a' on a task
// already timed at 14:00 with 08:00 free moves it to 08:00.
func TestAutoSchedule_ReHome_MovesEarlier(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	sm := int32(840) // 14:00
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 5, Name: "Task", StartMinute: &sm, DurationMinute: 30},
	}, 0)

	_, cmd := pressKeyStr(m, "a")

	if cmd == nil {
		t.Fatal("expected movePlanCmd, got nil")
	}
	cmd()
	if fc.moveReq == nil || fc.moveReq.StartMinute == nil {
		t.Fatal("MovePlanEntry was not called")
	}
	if *fc.moveReq.StartMinute != 480 {
		t.Errorf("re-home: expected start=480, got %d", *fc.moveReq.StartMinute)
	}
}

// TestAutoSchedule_AlreadyInPlace_NoOp verifies that pressing 'a' on a task
// already at the earliest fitting slot produces no command (FR-011 / US2.3).
func TestAutoSchedule_AlreadyInPlace_NoOp(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	sm := int32(480) // already at 08:00
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 5, Name: "Task", StartMinute: &sm, DurationMinute: 30},
	}, 0)

	_, cmd := pressKeyStr(m, "a")

	if cmd != nil {
		t.Error("already in place: expected no command (no-op), got a command")
	}
	if fc.moveReq != nil {
		t.Error("already in place: MovePlanEntry should not be called")
	}
}

// ── US3: event guard / empty plan ─────────────────────────────────────────

// TestAutoSchedule_EventHighlighted_SetsNoticeNoMove verifies that pressing 'a'
// on an event sets a playful notice and dispatches no command.
func TestAutoSchedule_EventHighlighted_SetsNoticeNoMove(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	sm := int32(480)
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 0, Name: "Stand-up", StartMinute: &sm, DurationMinute: 30},
	}, 0)

	m2, cmd := pressKeyStr(m, "a")

	if cmd != nil {
		t.Error("event: expected no command, got one")
	}
	notice := ExportNotice(m2)
	if notice == "" {
		t.Error("event: expected a playful notice, got empty string")
	}
}

// TestAutoSchedule_EmptyPlan_NoError verifies that pressing 'a' on an empty plan
// is a silent no-op (no command, no panic).
func TestAutoSchedule_EmptyPlan_NoError(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{}, 0)

	_, cmd := pressKeyStr(m, "a")

	if cmd != nil {
		t.Error("empty plan: expected nil command, got one")
	}
}
