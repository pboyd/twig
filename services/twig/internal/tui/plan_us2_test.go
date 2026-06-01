package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	planv1 "github.com/pboyd/twig/services/twig/gen/plan/v1"
)

// ── US2: splitPlanEntries (T018) ──────────────────────────────────────────────

// TestSplitPlanEntries_SeparatesUntimedAndTimed verifies that nil StartMinute
// entries land in the untimed bucket and non-nil ones land in timed.
func TestSplitPlanEntries_SeparatesUntimedAndTimed(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Untimed task"},
		{Id: 2, Name: "Morning standup", StartMinute: pint32(540)},
		{Id: 3, Name: "Another untimed"},
		{Id: 4, Name: "Afternoon focus", StartMinute: pint32(840)},
	}

	untimed, timed := ExportSplitPlanEntries(entries)

	if len(untimed) != 2 {
		t.Fatalf("splitPlanEntries: expected 2 untimed, got %d", len(untimed))
	}
	if untimed[0].Id != 1 || untimed[1].Id != 3 {
		t.Errorf("splitPlanEntries: untimed IDs should be [1,3], got [%d,%d]", untimed[0].Id, untimed[1].Id)
	}
	if len(timed) != 2 {
		t.Fatalf("splitPlanEntries: expected 2 timed, got %d", len(timed))
	}
	if timed[0].Id != 2 || timed[1].Id != 4 {
		t.Errorf("splitPlanEntries: timed IDs should be [2,4], got [%d,%d]", timed[0].Id, timed[1].Id)
	}
}

// TestSplitPlanEntries_AllTimed verifies all-timed list returns empty untimed bucket.
func TestSplitPlanEntries_AllTimed(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, StartMinute: pint32(540)},
		{Id: 2, StartMinute: pint32(600)},
	}
	untimed, timed := ExportSplitPlanEntries(entries)
	if len(untimed) != 0 {
		t.Errorf("all-timed: expected 0 untimed, got %d", len(untimed))
	}
	if len(timed) != 2 {
		t.Errorf("all-timed: expected 2 timed, got %d", len(timed))
	}
}

// TestSplitPlanEntries_AllUntimed verifies all-untimed list returns empty timed bucket.
func TestSplitPlanEntries_AllUntimed(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1},
		{Id: 2},
	}
	untimed, timed := ExportSplitPlanEntries(entries)
	if len(untimed) != 2 {
		t.Errorf("all-untimed: expected 2 untimed, got %d", len(untimed))
	}
	if len(timed) != 0 {
		t.Errorf("all-untimed: expected 0 timed, got %d", len(timed))
	}
}

// ── US2: untimed pane rendering (T019) ───────────────────────────────────────

// TestUntimedPane_RendersBoxesAboveGrid checks that renderPlanningView includes
// untimed entry names above the timed grid content.
func TestUntimedPane_RendersBoxesAboveGrid(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 40
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Must-do task"},
		{Id: 2, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}

	now := time.Now()
	out := m.renderPlanningView(80, 35, now)

	if !strings.Contains(out, "Must-do task") {
		t.Errorf("untimed pane: expected 'Must-do task' in output:\n%s", out)
	}
	if !strings.Contains(out, "Standup") {
		t.Errorf("untimed pane: expected 'Standup' in grid output:\n%s", out)
	}
	untimedPos := strings.Index(out, "Must-do task")
	timedPos := strings.Index(out, "Standup")
	if untimedPos >= timedPos {
		t.Errorf("untimed pane: 'Must-do task' (pos %d) should appear before 'Standup' (pos %d)", untimedPos, timedPos)
	}
}

// TestUntimedPane_HiddenWhenEmpty checks that renderPlanningView omits the
// untimed pane section when all entries are timed.
func TestUntimedPane_HiddenWhenEmpty(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 40
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}

	now := time.Now()
	out := m.renderPlanningView(80, 35, now)
	if !strings.Contains(out, "Standup") {
		t.Errorf("no untimed entries: expected 'Standup' still in output:\n%s", out)
	}
}

// TestUntimedPane_OnlyUntimedEntries checks output when the day has only untimed entries.
func TestUntimedPane_OnlyUntimedEntries(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 40
	m.plan.day = "2026-01-02"
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Write report", DurationMinute: 60},
		{Id: 2, Name: "Code review", DurationMinute: 30},
	}

	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	out := m.renderPlanningView(80, 35, now)

	if !strings.Contains(out, "Write report") {
		t.Errorf("only-untimed: expected 'Write report' in output:\n%s", out)
	}
	if !strings.Contains(out, "Code review") {
		t.Errorf("only-untimed: expected 'Code review' in output:\n%s", out)
	}
}

// TestRenderPlanDetail_UntimedEntry checks that an untimed entry shows "untimed"
// instead of a 00:00-based window.
func TestRenderPlanDetail_UntimedEntry(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Write spec",
		DurationMinute: 60,
		TaskId:         7,
	}
	out := renderPlanDetail(entry, 40, false)

	if !strings.Contains(out, "Write spec") {
		t.Errorf("untimed detail: expected entry name; got %q", out)
	}
	if !strings.Contains(out, "untimed") {
		t.Errorf("untimed detail: expected 'untimed' window label; got %q", out)
	}
	if strings.Contains(out, "00:00") {
		t.Errorf("untimed detail: must not show '00:00' window; got %q", out)
	}
}

// ── US2: unified Up/Down cursor (T020) ───────────────────────────────────────

// TestNavigation_CursorCrossesUntimedAndTimed checks that j/k navigate over
// both untimed and timed entries in the unified m.plan.entries slice.
func TestNavigation_CursorCrossesUntimedAndTimed(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	// entries ordered untimed-first (as returned by server ORDER BY NULLS FIRST)
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Untimed task"},
		{Id: 2, Name: "Standup", StartMinute: pint32(540)},
		{Id: 3, Name: "Focus", StartMinute: pint32(600)},
	}
	m.plan.cursor = 0

	m2, _ := pressKeyStr(m, "j")
	if m2.plan.cursor != 1 {
		t.Errorf("down from untimed to timed: expected cursor=1, got %d", m2.plan.cursor)
	}
	m3, _ := pressKeyStr(m2, "j")
	if m3.plan.cursor != 2 {
		t.Errorf("down to second timed: expected cursor=2, got %d", m3.plan.cursor)
	}
	m4, _ := pressKeyStr(m3, "k")
	if m4.plan.cursor != 1 {
		t.Errorf("up back to first timed: expected cursor=1, got %d", m4.plan.cursor)
	}
	m5, _ := pressKeyStr(m4, "k")
	if m5.plan.cursor != 0 {
		t.Errorf("up back to untimed: expected cursor=0, got %d", m5.plan.cursor)
	}
}

// TestPlanGridOptions_SelectedIDReflectsUntimedCursor checks that planGridOptions
// returns the ID of an untimed entry when the cursor points to it.
func TestPlanGridOptions_SelectedIDReflectsUntimedCursor(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewPlanModel(nil, nil, today)
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 10, Name: "Untimed"},
		{Id: 20, Name: "Timed", StartMinute: pint32(540)},
	}, 0)

	opts := m.planGridOptions()
	if opts.SelectedID != 10 {
		t.Errorf("planGridOptions with cursor on untimed: SelectedID should be 10, got %d", opts.SelectedID)
	}
}

// ── US2: optional-start task form (T021) ─────────────────────────────────────

// TestTaskTimeForm_BlankStartCreatesUntimedEntry checks that submitting the
// task-time form with a blank Start sends AddPlanTask with nil StartMinute.
func TestTaskTimeForm_BlankStartCreatesUntimedEntry(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initTaskTimeForm(42)
	// Leave start blank; set only duration
	m.plan.form.fields[1].SetValue("30m")

	cmd := m.submitTaskTimeForm()

	if m.plan.mode != planList {
		t.Errorf("blank start: form should close (planList), got mode=%d", m.plan.mode)
	}
	if m.plan.err != nil {
		t.Errorf("blank start: expected no error, got %v", m.plan.err)
	}
	if cmd == nil {
		t.Fatal("blank start: expected addPlanTaskCmd, got nil")
	}
	cmd()
	if fc.addTaskReq == nil {
		t.Fatal("blank start: AddPlanTask was not called")
	}
	if fc.addTaskReq.StartMinute != nil {
		t.Errorf("blank start: expected nil StartMinute (untimed), got %v", fc.addTaskReq.StartMinute)
	}
}

// TestTaskTimeForm_TimedStartSendsStartMinute checks that a filled Start sends
// a non-nil StartMinute to AddPlanTask.
func TestTaskTimeForm_TimedStartSendsStartMinute(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initTaskTimeForm(42)
	m.plan.form.fields[0].SetValue("09:00")
	m.plan.form.fields[1].SetValue("30m")

	cmd := m.submitTaskTimeForm()

	if m.plan.err != nil {
		t.Errorf("timed start: expected no error, got %v", m.plan.err)
	}
	if cmd == nil {
		t.Fatal("timed start: expected addPlanTaskCmd, got nil")
	}
	cmd()
	if fc.addTaskReq == nil {
		t.Fatal("timed start: AddPlanTask was not called")
	}
	if fc.addTaskReq.StartMinute == nil {
		t.Error("timed start: expected non-nil StartMinute, got nil")
	}
	if *fc.addTaskReq.StartMinute != 540 {
		t.Errorf("timed start: expected StartMinute=540 (09:00), got %d", *fc.addTaskReq.StartMinute)
	}
}

// TestTaskTimeForm_InvalidTimeSetsError checks that an invalid start keeps the
// form open and sets plan.err (regression: blank now allowed, bad time is not).
func TestTaskTimeForm_InvalidTimeSetsError(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initTaskTimeForm(42)
	m.plan.form.fields[0].SetValue("not-a-time")

	cmd := m.submitTaskTimeForm()

	if m.plan.mode != planTaskTime {
		t.Errorf("invalid time: form should stay planTaskTime, got %d", m.plan.mode)
	}
	if m.plan.err == nil {
		t.Error("invalid time: expected plan.err to be set")
	}
	if cmd != nil {
		t.Error("invalid time: expected nil cmd")
	}
}

// TestTaskTimeForm_AllBlankCreatesUntimedEntry checks that both start and
// duration blank still creates an untimed entry (default duration applied server-side).
func TestTaskTimeForm_AllBlankCreatesUntimedEntry(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.initTaskTimeForm(42)
	// Leave both fields blank

	cmd := m.submitTaskTimeForm()

	if m.plan.err != nil {
		t.Errorf("all-blank: expected no error, got %v", m.plan.err)
	}
	if cmd == nil {
		t.Fatal("all-blank: expected addPlanTaskCmd, got nil")
	}
	// Run the command to verify RPC call.
	result := cmd()
	// The result is a planMutatedMsg (not an error from the fake client).
	if _, ok := result.(planMutatedMsg); !ok {
		t.Errorf("all-blank: expected planMutatedMsg, got %T: %v", result, result)
	}
	if fc.addTaskReq == nil {
		t.Fatal("all-blank: AddPlanTask was not called")
	}
	if fc.addTaskReq.StartMinute != nil {
		t.Errorf("all-blank: expected nil StartMinute (untimed), got %v", fc.addTaskReq.StartMinute)
	}
}

// TestPickerToTaskForm_BlankStartFlow exercises the full picker → task-time-form →
// submit flow with a blank start, verifying mode transitions and the RPC call.
func TestPickerToTaskForm_BlankStartFlow(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.mode = planPickTask

	// Simulate selecting a task from the picker (taskID=7) by initialising the form.
	m.initTaskTimeForm(7)

	if m.plan.mode != planTaskTime {
		t.Fatalf("after initTaskTimeForm: expected planTaskTime, got %d", m.plan.mode)
	}

	// Submit with blank start → untimed entry.
	cmd := m.submitTaskTimeForm()

	if m.plan.mode != planList {
		t.Errorf("after blank submit: expected planList, got %d", m.plan.mode)
	}
	if cmd == nil {
		t.Fatal("after blank submit: expected a command")
	}
	cmd()
	if fc.addTaskReq == nil {
		t.Fatal("AddPlanTask was not called")
	}
	if fc.addTaskReq.TaskId != 7 {
		t.Errorf("expected TaskId=7, got %d", fc.addTaskReq.TaskId)
	}
	if fc.addTaskReq.StartMinute != nil {
		t.Errorf("blank start: expected nil StartMinute, got %v", fc.addTaskReq.StartMinute)
	}
}

// TestViewPlanning_UntimedPane_Styled checks that viewPlanning in styled mode
// renders the untimed entry name when there is an untimed entry.
func TestViewPlanning_UntimedPane_Styled(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 24
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Review PR"},
		{Id: 2, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	m.plan.cursor = 0

	out := m.viewPlanning()

	if !strings.Contains(out, "Review PR") {
		t.Errorf("styled untimed pane: expected 'Review PR'; got:\n%s", out)
	}
}

// TestPlanFieldLabel_TaskTimeStartIsOptional checks that planFieldLabel for
// planTaskTime index 0 contains "optional".
func TestPlanFieldLabel_TaskTimeStartIsOptional(t *testing.T) {
	label := planFieldLabel(planTaskTime, 0)
	if !strings.Contains(strings.ToLower(label), "optional") {
		t.Errorf("planFieldLabel(planTaskTime, 0) = %q; want 'optional' in label", label)
	}
}

// TestTaskTimeForm_EscCancels checks that Esc in planTaskTime mode returns to planList.
func TestTaskTimeForm_EscCancels(t *testing.T) {
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.initTaskTimeForm(5)

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if m2.(Model).plan.mode != planList {
		t.Errorf("esc from task-time form: expected planList, got %d", m2.(Model).plan.mode)
	}
}
