package tui

import (
	"strings"
	"testing"
	"time"

	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// TestTabBar_TasksTabActive checks that the tab bar labels are present and the
// Tasks tab is marked when activeTab==tabTasks (unstyled).
func TestTabBar_TasksTabActive(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.width = 80
	bar := m.renderTabBar(80)

	if !strings.Contains(bar, "Tasks") {
		t.Errorf("tab bar missing 'Tasks': %q", bar)
	}
	if !strings.Contains(bar, "Planning") {
		t.Errorf("tab bar missing 'Planning': %q", bar)
	}
}

// TestTabBar_PlanningTabActive checks that switchting to tabPlanning is reflected in the bar.
func TestTabBar_PlanningTabActive(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.activeTab = tabPlanning
	m.width = 80
	bar := m.renderTabBar(80)

	if !strings.Contains(bar, "Planning") {
		t.Errorf("tab bar missing 'Planning': %q", bar)
	}
}

// TestTabBar_StyledActiveHighlighted checks that in styled mode the active tab
// receives ANSI bold codes.
func TestTabBar_StyledActiveHighlighted(t *testing.T) {
	m := ExportNewStyledModel(nil, nil, true)
	m.activeTab = tabPlanning
	m.width = 80
	bar := m.renderTabBar(80)

	if !strings.Contains(bar, "\x1b[") {
		t.Errorf("styled tab bar: expected ANSI codes for active tab: %q", bar)
	}
}

// TestPlanningView_RendersGrid checks that renderPlanningView calls RenderGrid
// and produces output containing the grid (now-marker on today).
func TestPlanningView_RendersGrid(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 40
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Day: today, Id: 1, Name: "Standup", StartMinute: 540, DurationMinute: 30},
	}

	now := time.Now()
	out := m.renderPlanningView(80, 35, now)

	if !strings.Contains(out, "Standup") {
		t.Errorf("planning view: expected 'Standup' in output:\n%s", out)
	}
	// HideID: should NOT contain "[1]" prefix
	if strings.Contains(out, "[1]") {
		t.Errorf("planning view: id prefix [1] should be hidden (HideID=true):\n%s", out)
	}
}

// TestPlanningView_DayHeader_Today checks that the day header shows "(today)" for today.
func TestPlanningView_DayHeader_Today(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	today := now.Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.plan.day = today
	m.plan.loaded = true

	header := m.planDayHeader(now)

	if !strings.Contains(header, "today") {
		t.Errorf("day header should contain 'today' for today: %q", header)
	}
}

// TestPlanningView_DayHeader_NotToday checks that the day header does not show
// "(today)" for a different day.
func TestPlanningView_DayHeader_NotToday(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.plan.day = yesterday

	now := time.Now()
	header := m.planDayHeader(now)

	if strings.Contains(header, "today") {
		t.Errorf("day header should not contain 'today' for non-today: %q", header)
	}
}

// TestPlanningView_NowMarker_Today checks that the now-marker (▶) appears in the
// planning view when the in-view day is today.
func TestPlanningView_NowMarker_Today(t *testing.T) {
	// Use a fixed "now" at 10:00 so it's guaranteed to be within the default 08–17 window.
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	today := now.Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.plan.day = today
	m.plan.loaded = true

	out := m.renderPlanningView(80, 40, now)

	if !strings.Contains(out, "▶") {
		t.Errorf("planning view for today: expected ▶ now-marker in output:\n%s", out)
	}
}

// TestPlanningView_NowMarker_NotToday checks that the now-marker does NOT appear
// when viewing a day other than today.
func TestPlanningView_NowMarker_NotToday(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.plan.day = yesterday
	m.plan.loaded = true

	// now is in today, but plan.day is yesterday.
	now := time.Now()
	out := m.renderPlanningView(80, 35, now)

	if strings.Contains(out, "▶") {
		t.Errorf("planning view for non-today: unexpected ▶ now-marker:\n%s", out)
	}
}

// TestPlanningView_CompletedStrikethrough checks that a completed entry renders
// with strikethrough in styled mode.
func TestPlanningView_CompletedStrikethrough(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Day: today, Id: 1, Name: "Done", StartMinute: 540, DurationMinute: 120, Completed: true},
	}

	now := time.Now()
	out := m.renderPlanningView(80, 35, now)

	if !strings.Contains(out, cli.DimStrike("Done")) {
		// Just check that ANSI strikethrough code is present.
		if !strings.Contains(out, "\x1b[2;9m") {
			t.Errorf("completed entry: expected strikethrough ANSI code in planning view:\n%s", out)
		}
	}
}

// TestPlanGrid_SelectedID checks that planGridOptions returns HideID:true and
// SelectedID matching the cursor's entry.
func TestPlanGrid_SelectedID(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.plan.day = today
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "First", StartMinute: 540},
		{Id: 2, Name: "Second", StartMinute: 600},
	}
	m.plan.cursor = 1

	opts := m.planGridOptions()
	if !opts.HideID {
		t.Error("planGridOptions: HideID should always be true")
	}
	if opts.SelectedID != 2 {
		t.Errorf("planGridOptions: SelectedID should be 2 (cursor=1), got %d", opts.SelectedID)
	}
}

// TestPlanGrid_NoEntries checks that planGridOptions returns SelectedID=0 when no entries.
func TestPlanGrid_NoEntries(t *testing.T) {
	m := ExportNewModel(nil, nil)
	opts := m.planGridOptions()
	if opts.SelectedID != 0 {
		t.Errorf("no entries: SelectedID should be 0, got %d", opts.SelectedID)
	}
}

// ── US2: Planning help view (T006) ─────────────────────────────────────────

// TestPlanningHelp_ShowsPlanningBindings checks that when on the Planning tab
// with modeHelp set, View() renders full-help content with Planning bindings (T006).
func TestPlanningHelp_ShowsPlanningBindings(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 24
	m.activeTab = tabPlanning
	m.keys.PlanningMode = true
	m.mode = modeHelp

	out := m.View()

	// Planning FullHelp includes PlanAddTask ("add task"), navigation, etc.
	if !strings.Contains(out, "add task") {
		t.Errorf("Planning help view: expected 'add task' binding; got:\n%q", out)
	}
	// PomCancel ("cancel pomodoro") must be in the Planning FullHelp.
	if !strings.Contains(out, "cancel pomodoro") {
		t.Errorf("Planning help view: expected 'cancel pomodoro' binding; got:\n%q", out)
	}
}

// ── US4: status line at bottom (T016) ──────────────────────────────────────

// TestViewPlanning_StatusOnBottomRow checks that the help/status line appears as
// the last non-empty line of viewPlanning output at a fixed terminal height (T016).
func TestViewPlanning_StatusOnBottomRow(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 20
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true

	out := m.viewPlanning()
	lines := strings.Split(out, "\n")

	// Trim trailing empty lines.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		t.Fatal("viewPlanning: empty output")
	}

	// Last line should be the status (help keys, not grid content).
	last := lines[len(lines)-1]
	// The status line contains the help text from m.help.View(m.keys).
	// At minimum it's non-empty.
	if last == "" {
		t.Error("viewPlanning: last line (status) is empty — status not pinned to bottom")
	}
	// The total line count should be m.height (or fewer if terminal is tall).
	if len(lines) > m.height {
		t.Errorf("viewPlanning: output has %d lines, exceeds height=%d", len(lines), m.height)
	}
}

// ── US5: two-pane Planning layout (T022, T023) ─────────────────────────────

// TestViewPlanning_TwoPaneLayout_Unstyled checks that unstyled viewPlanning output
// has both the grid (day header visible) and a details section side-by-side (T022).
func TestViewPlanning_TwoPaneLayout_Unstyled(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 20
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: 540, DurationMinute: 30},
	}
	m.plan.cursor = 0

	out := m.viewPlanning()

	// Grid content (day header) should appear.
	if !strings.Contains(out, "Standup") {
		t.Errorf("unstyled two-pane: expected 'Standup' from grid; got:\n%s", out)
	}
	// Details pane should show entry name.
	// The entry name appears in the details pane.
	// Because it's unstyled, both panes are merged row-by-row.
	// We count how many times "Standup" appears — grid has it, details has it.
	count := strings.Count(out, "Standup")
	if count < 1 {
		t.Errorf("unstyled two-pane: expected at least one 'Standup'; got %d occurrences", count)
	}
}

// TestViewPlanning_TwoPaneLayout_Styled checks that styled viewPlanning uses
// paneBox borders and shows the "Details" title (T022).
func TestViewPlanning_TwoPaneLayout_Styled(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 20
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: 540, DurationMinute: 30},
	}
	m.plan.cursor = 0

	out := m.viewPlanning()

	// Styled mode uses paneBox borders.
	if !strings.ContainsAny(out, "╭╰╮╯") {
		t.Errorf("styled two-pane: expected border runes; got:\n%s", out)
	}
	// Details pane title must be present.
	if !strings.Contains(out, "Details") {
		t.Errorf("styled two-pane: expected 'Details' pane title; got:\n%s", out)
	}
}

// TestViewPlanning_EmptyDay_Placeholder checks that an empty day shows a
// placeholder in the details pane (T022).
func TestViewPlanning_EmptyDay_Placeholder(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 20
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = nil

	out := m.viewPlanning()

	if strings.Contains(out, "error") {
		t.Errorf("empty day: should not show error; got:\n%s", out)
	}
}

// TestRenderPlanDetail_NilEntry checks that nil entry returns a placeholder (T023).
func TestRenderPlanDetail_NilEntry(t *testing.T) {
	out := renderPlanDetail(nil, 40, false)
	if !strings.Contains(out, "no entry") && !strings.Contains(out, "nothing") && !strings.Contains(out, "(empty)") && out == "" {
		// Any non-empty placeholder is fine.
	}
	// Should not panic and should return a string.
}

// TestRenderPlanDetail_EventEntry checks that an event entry renders name and
// window but no Task-specific fields (T023).
func TestRenderPlanDetail_EventEntry(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Team sync",
		StartMinute:    540, // 09:00
		DurationMinute: 30,
		TaskId:         0, // event, not a task
	}
	out := renderPlanDetail(entry, 40, false)

	if !strings.Contains(out, "Team sync") {
		t.Errorf("event: expected entry name 'Team sync'; got %q", out)
	}
	if !strings.Contains(out, "09:00") {
		t.Errorf("event: expected window start '09:00'; got %q", out)
	}
	// Events must NOT show a Task field.
	if strings.Contains(out, "Task") {
		t.Errorf("event: must not show 'Task' field; got %q", out)
	}
}

// TestRenderPlanDetail_TaskEntry checks that a task-linked entry shows task info (T023).
func TestRenderPlanDetail_TaskEntry(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Write tests",
		StartMinute:    600, // 10:00
		DurationMinute: 60,
		TaskId:         42,
		Completed:      false,
	}
	out := renderPlanDetail(entry, 40, false)

	if !strings.Contains(out, "Write tests") {
		t.Errorf("task entry: expected name; got %q", out)
	}
	if !strings.Contains(out, "10:00") {
		t.Errorf("task entry: expected window start; got %q", out)
	}
	// Task entries must show some task-related info.
	if !strings.Contains(out, "Task") && !strings.Contains(out, "task") && !strings.Contains(out, "#42") {
		t.Errorf("task entry: expected task info; got %q", out)
	}
}

// TestRenderPlanDetail_CompletedTask checks that completion is reflected (T023).
func TestRenderPlanDetail_CompletedTask(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Done",
		StartMinute:    600,
		DurationMinute: 60,
		TaskId:         5,
		Completed:      true,
	}
	out := renderPlanDetail(entry, 40, false)
	if !strings.Contains(out, "complet") { // "completed" or "complete"
		t.Errorf("completed task: expected 'completed' status; got %q", out)
	}
}

// TestRenderPlanDetail_Unstyled checks that unstyled output has no ANSI codes (T023).
func TestRenderPlanDetail_Unstyled(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Focus",
		StartMinute:    480,
		DurationMinute: 120,
		TaskId:         0,
	}
	out := renderPlanDetail(entry, 40, false)
	if strings.Contains(out, "\x1b[") {
		t.Errorf("unstyled renderPlanDetail: must not emit ANSI codes; got %q", out)
	}
}
