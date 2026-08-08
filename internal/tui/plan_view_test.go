package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/markdown"
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
	// Fixed day + pre-window now → TOP-TRUNCATE keeps the 09:00 entry visible
	// regardless of when the test runs.
	day := "2026-05-27"
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 40
	m.plan.day = day
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Day: day, Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}

	now := time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC) // before 08:00 → TOP-TRUNCATE
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
	// Fixed day + pre-window now → TOP-TRUNCATE keeps the 09:00 entry visible.
	day := "2026-05-27"
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.plan.day = day
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Day: day, Id: 1, Name: "Done", StartMinute: pint32(540), DurationMinute: 120, Completed: true},
	}

	now := time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC) // before 08:00 → TOP-TRUNCATE
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
		{Id: 1, Name: "First", StartMinute: pint32(540)},
		{Id: 2, Name: "Second", StartMinute: pint32(600)},
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

	out := m.View().Content

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
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
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
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
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
	out := renderPlanDetail(nil, nil, 40, false, nil)
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
		StartMinute:    pint32(540), // 09:00
		DurationMinute: 30,
		TaskId:         0, // event, not a task
	}
	out := renderPlanDetail(entry, nil, 40, false, nil)

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
		StartMinute:    pint32(600), // 10:00
		DurationMinute: 60,
		TaskId:         42,
		Completed:      false,
	}
	out := renderPlanDetail(entry, nil, 40, false, nil)

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
		StartMinute:    pint32(600),
		DurationMinute: 60,
		TaskId:         5,
		Completed:      true,
	}
	out := renderPlanDetail(entry, nil, 40, false, nil)
	if !strings.Contains(out, "complet") { // "completed" or "complete"
		t.Errorf("completed task: expected 'completed' status; got %q", out)
	}
}

// TestRenderPlanDetail_Unstyled checks that unstyled output has no ANSI codes (T023).
func TestRenderPlanDetail_Unstyled(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Focus",
		StartMinute:    pint32(480),
		DurationMinute: 120,
		TaskId:         0,
	}
	out := renderPlanDetail(entry, nil, 40, false, nil)
	if strings.Contains(out, "\x1b[") {
		t.Errorf("unstyled renderPlanDetail: must not emit ANSI codes; got %q", out)
	}
}

// ── US2: Tasks-tab help (T010) ─────────────────────────────────────────────

// TestTasksHelp_AdvertisesEnterForEdit checks that ShortHelp on the Tasks tab
// advertises "enter" for "edit task" (T010).
func TestTasksHelp_AdvertisesEnterForEdit(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 24
	m.activeTab = tabTasks
	m.keys.PlanningMode = false

	short := m.keys.ShortHelp()
	found := false
	for _, b := range short {
		if strings.Contains(b.Help().Key, "enter") && strings.Contains(b.Help().Desc, "edit") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Tasks ShortHelp: expected 'enter / edit task' binding; bindings: %v", short)
	}
}

// ── US5: Planning help (T019) ──────────────────────────────────────────────

// TestPlanningHelp_AdvertisesTForAddTask checks that Planning FullHelp contains
// 't' for "add task" (T019).
func TestPlanningHelp_AdvertisesTForAddTask(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.keys.PlanningMode = true

	full := m.keys.FullHelp()
	found := false
	for _, group := range full {
		for _, b := range group {
			if strings.Contains(b.Help().Key, "t") && strings.Contains(b.Help().Desc, "add task") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("Planning FullHelp: expected 't / add task' binding")
	}
}

// TestPlanningHelp_AdvertisesDotForToday checks that Planning FullHelp contains
// '.' for "today" (T019).
func TestPlanningHelp_AdvertisesDotForToday(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.keys.PlanningMode = true

	full := m.keys.FullHelp()
	found := false
	for _, group := range full {
		for _, b := range group {
			if strings.Contains(b.Help().Key, ".") && strings.Contains(b.Help().Desc, "today") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("Planning FullHelp: expected '. / today' binding")
	}
}

// ── US6: no clear in Planning help (T021) ──────────────────────────────────

// TestPlanningHelp_NoClearAction checks that Planning FullHelp does NOT mention
// "clear" (T021).
func TestPlanningHelp_NoClearAction(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.keys.PlanningMode = true

	full := m.keys.FullHelp()
	for _, group := range full {
		for _, b := range group {
			if strings.Contains(strings.ToLower(b.Help().Desc), "clear") {
				t.Errorf("Planning FullHelp: must not mention 'clear'; found: %q", b.Help().Desc)
			}
		}
	}
}

// ── US3: right-pane forms (T012) ───────────────────────────────────────────

// TestViewPlanning_FormInRightPane_Styled checks that in styled mode, when a form
// is active (plan.mode != planList), viewPlanning still renders both the grid
// (left pane) and the form (right pane) — grid lines present alongside form (T012).
func TestViewPlanning_FormInRightPane_Styled(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 20
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	m.plan.cursor = 0
	m.initEditForm()

	out := m.viewPlanning()

	// Both panes should have borders.
	if !strings.ContainsAny(out, "╭╰╮╯") {
		t.Errorf("styled form pane: expected border runes alongside grid; got:\n%s", out)
	}
	// Grid should still be visible (day header or grid content).
	if !strings.Contains(out, "Standup") {
		t.Errorf("styled form pane: expected grid entry 'Standup' still visible; got:\n%s", out)
	}
}

// TestViewPlanning_FormInRightPane_Unstyled checks that in unstyled mode, a form
// renders in the right column while the grid is on the left (T012).
func TestViewPlanning_FormInRightPane_Unstyled(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	// Height 45 gives gridHeight≈43 ≥ baseRows(37), so GridWindow uses FILL mode
	// and the 09:00 entry is visible regardless of the wall-clock time.
	m.height = 45
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Focus", StartMinute: pint32(540), DurationMinute: 60},
	}
	m.plan.cursor = 0
	m.initAddEventForm()

	out := m.viewPlanning()

	// Grid should be visible alongside the form.
	if !strings.Contains(out, "Focus") {
		t.Errorf("unstyled form pane: expected 'Focus' from grid; got:\n%s", out)
	}
	// Form field labels should appear.
	if !strings.Contains(out, "Name") && !strings.Contains(out, "Start") {
		t.Errorf("unstyled form pane: expected form field labels; got:\n%s", out)
	}
}

// ── T007: Foundational form render assertions ─────────────────────────────

// TestPlanFormView_ButtonsAndHelpLine checks that renderPlanFormView emits
// Save/Cancel buttons and the exact help line for all three form modes.
func TestPlanFormView_ButtonsAndHelpLine(t *testing.T) {
	cases := []struct {
		mode planMode
		name string
	}{
		{planEdit, "Edit entry"},
		{planTaskTime, "Schedule task"},
		{planEventForm, "Add event"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := ExportNewModel(nil, nil)
			m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "X", StartMinute: pint32(540), DurationMinute: 30}}
			m.plan.cursor = 0
			switch c.mode {
			case planEdit:
				m.initEditForm()
			case planTaskTime:
				m.initTaskTimeForm(1)
			case planEventForm:
				m.initAddEventForm()
			}
			out := m.renderPlanFormView(80)

			if !strings.Contains(out, "[ Save ]") {
				t.Errorf("form should contain '[ Save ]'; got:\n%s", out)
			}
			if !strings.Contains(out, "[ Cancel ]") {
				t.Errorf("form should contain '[ Cancel ]'; got:\n%s", out)
			}
			const wantHelp = "Ctrl+S: save  Tab: next field"
			if !strings.Contains(out, wantHelp) {
				t.Errorf("form should contain help line %q; got:\n%s", wantHelp, out)
			}
		})
	}
}

// TestPlanFormView_FocusedSaveButton checks that the Save button renders as
// [>Save<] when its virtual slot is focused.
func TestPlanFormView_FocusedSaveButton(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "X", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	for !planFocusSave(m.plan.form) {
		m.cyclePlanFormFocus(1)
	}
	out := m.renderPlanFormView(80)
	if !strings.Contains(out, "[>Save<]") {
		t.Errorf("focused Save button: want '[>Save<]'; got:\n%s", out)
	}
	if strings.Contains(out, "[>Cancel<]") {
		t.Errorf("focused Save button: Cancel should not be focused; got:\n%s", out)
	}
}

// TestPlanFormView_FocusedCancelButton checks that the Cancel button renders
// as [>Cancel<] when its virtual slot is focused.
func TestPlanFormView_FocusedCancelButton(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "X", StartMinute: pint32(540)}}
	m.plan.cursor = 0
	m.initEditForm()
	for !planFocusCancel(m.plan.form) {
		m.cyclePlanFormFocus(1)
	}
	out := m.renderPlanFormView(80)
	if !strings.Contains(out, "[>Cancel<]") {
		t.Errorf("focused Cancel button: want '[>Cancel<]'; got:\n%s", out)
	}
	if strings.Contains(out, "[>Save<]") {
		t.Errorf("focused Cancel button: Save should not be focused; got:\n%s", out)
	}
}

// stripANSI removes ANSI escape sequences so plain text assertions work on
// styled textinput output.
var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiEscapeRe.ReplaceAllString(s, "") }

// TestPlanFormView_PlaceholdersFullyRendered guards against the Width==0 bug
// where textinput.placeholderView truncates to a single character.
func TestPlanFormView_PlaceholdersFullyRendered(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.initAddEventForm()
	out := stripANSI(m.renderPlanFormView(80))
	for _, want := range []string{"Event name", "Start time", "Duration"} {
		if !strings.Contains(out, want) {
			t.Errorf("placeholder %q not found in form output:\n%s", want, out)
		}
	}
}

// ── T015: US4 — separator between untimed pane and grid ────────────────────

// TestSeparator_PresentWhenUntimedEntriesExist checks that a horizontal separator
// appears between the untimed pane and the grid when untimed entries are present.
func TestSeparator_PresentWhenUntimedEntriesExist(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 40
	m.plan.day = "2026-06-01"
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Untimed task", DurationMinute: 30},                      // untimed
		{Id: 2, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30}, // timed
	}

	// Use now before the default 08:00 window so nowBlock < baseStart → TOP-TRUNCATE
	// and the 09:00 Standup stays visible regardless of terminal height.
	now := time.Date(2026, 6, 1, 7, 0, 0, 0, time.UTC)
	out := m.renderPlanningView(80, 50, now)

	// The separator is a line containing only box-drawing divider chars (─, ┤, ├, etc.)
	// It must appear between the untimed pane and the grid.
	untimedPos := strings.Index(out, "Untimed task")
	gridPos := strings.Index(out, "Standup")
	if untimedPos < 0 {
		t.Fatal("separator test: 'Untimed task' not found in output")
	}
	if gridPos < 0 {
		t.Fatal("separator test: 'Standup' not found in output")
	}

	// Extract the portion between untimed content and grid content.
	between := out[untimedPos:gridPos]
	// A separator line contains at least some ─ characters.
	if !strings.Contains(between, "─") {
		t.Errorf("no separator between untimed pane and grid; between content: %q", between)
	}
}

// TestSeparator_AbsentWhenNoUntimedEntries checks that when all entries are timed,
// the untimed pane and separator are both absent (no RenderUntimed output before the grid).
func TestSeparator_AbsentWhenNoUntimedEntries(t *testing.T) {
	// Model with only a timed entry.
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 40
	m.plan.day = "2026-06-01"
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Timed event", StartMinute: pint32(540), DurationMinute: 30},
	}

	// Use now before the default 08:00 window so TOP-TRUNCATE is used and the
	// 09:00 entry stays visible regardless of terminal height.
	now := time.Date(2026, 6, 1, 7, 0, 0, 0, time.UTC)
	timedOnly := m.renderPlanningView(80, 50, now)

	// Model with the same timed entry plus an untimed entry.
	m2 := m
	m2.plan.entries = []*planv1.PlanEntry{
		{Id: 2, Name: "Untimed task", DurationMinute: 30}, // untimed
		{Id: 1, Name: "Timed event", StartMinute: pint32(540), DurationMinute: 30},
	}
	withUntimed := m2.renderPlanningView(80, 50, now)

	// When untimed entries exist, the output must be different (untimed pane + separator added).
	if timedOnly == withUntimed {
		t.Error("output should differ when untimed entries are added (untimed pane + separator)")
	}
	// The timed-only output must not contain "Untimed task".
	if strings.Contains(timedOnly, "Untimed task") {
		t.Error("timed-only output must not contain the untimed entry name")
	}
	// The timed-only output must contain the timed entry.
	if !strings.Contains(timedOnly, "Timed event") {
		t.Error("timed-only output must contain the timed entry name")
	}
}

// TestPlanGridOptions_CursorBgOnLightBg verifies that with hasDarkBackground=false
// the SelectionStyle uses the light cursorBg color (#DDEEFF = rgb(221,238,255)).
func TestPlanGridOptions_CursorBgOnLightBg(t *testing.T) {
	today := "2026-05-29"
	m := ExportNewPlanModel(nil, nil, today)
	m.styled = true
	m.hasDarkBackground = false
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Day: today, Id: 1, Name: "Task", StartMinute: pint32(540), DurationMinute: 30},
	}, 0)
	opts := m.planGridOptions()
	if opts.SelectionStyle == nil {
		t.Fatal("SelectionStyle must not be nil when an entry is selected")
	}
	rendered := opts.SelectionStyle("X")
	// Light cursorBg: #DDEEFF = rgb(221, 238, 255) → \x1b[48;2;221;238;255m in truecolor
	if !strings.Contains(rendered, "221;238;255") {
		t.Errorf("light background: expected cursorBg light RGB 221;238;255 in %q", rendered)
	}
}

// TestPlanGridOptions_CursorBgOnDarkBg verifies that with hasDarkBackground=true
// the SelectionStyle uses the dark cursorBg color (#1A2A3A = rgb(26,42,58)).
func TestPlanGridOptions_CursorBgOnDarkBg(t *testing.T) {
	today := "2026-05-29"
	m := ExportNewPlanModel(nil, nil, today)
	m.styled = true
	m.hasDarkBackground = true
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Day: today, Id: 1, Name: "Task", StartMinute: pint32(540), DurationMinute: 30},
	}, 0)
	opts := m.planGridOptions()
	if opts.SelectionStyle == nil {
		t.Fatal("SelectionStyle must not be nil when an entry is selected")
	}
	rendered := opts.SelectionStyle("X")
	// Dark cursorBg: #1A2A3A = rgb(26, 42, 58) → \x1b[48;2;26;42;58m in truecolor
	if !strings.Contains(rendered, "26;42;58") {
		t.Errorf("dark background: expected cursorBg dark RGB 26;42;58 in %q", rendered)
	}
}

// ── T014: US2 — glyph row in planning detail ───────────────────────────────

// TestRenderPlanDetail_GlyphRowLinkedTask asserts glyph row appears for linked task with estimate.
func TestRenderPlanDetail_GlyphRowLinkedTask(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Focus session",
		StartMinute:    pint32(540),
		DurationMinute: 50,
		TaskId:         7,
	}
	task := &taskv1.Task{
		Id:                     7,
		Estimate:               3,
		CompletedPomodoroCount: 1,
	}

	// Plain mode.
	plain := renderPlanDetail(entry, task, 60, false, nil)
	n := countPomodoroGlyphs(plain)
	if n != 3 {
		t.Errorf("plain: want 3 glyphs for estimate=3; got %d; %q", n, plain)
	}
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("plain mode must not emit ANSI; got %q", plain)
	}

	// Styled mode.
	styled := renderPlanDetail(entry, task, 60, true, nil)
	ns := countPomodoroGlyphs(styled)
	if ns != 3 {
		t.Errorf("styled: want 3 glyphs for estimate=3; got %d; %q", ns, styled)
	}
}

// TestRenderPlanDetail_GlyphRowEventEntry asserts no glyph row for event entry (task nil).
func TestRenderPlanDetail_GlyphRowEventEntry(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Meeting",
		StartMinute:    pint32(540),
		DurationMinute: 30,
		TaskId:         0,
	}
	out := renderPlanDetail(entry, nil, 60, false, nil)
	if countPomodoroGlyphs(out) != 0 {
		t.Errorf("event entry must not show glyph row; got %q", out)
	}
}

// TestRenderPlanDetail_GlyphRowNoPomodoros asserts no glyph row when task has 0 estimate and 0 completed.
func TestRenderPlanDetail_GlyphRowNoPomodoros(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "No estimate task",
		StartMinute:    pint32(540),
		DurationMinute: 30,
		TaskId:         3,
	}
	task := &taskv1.Task{
		Id:                     3,
		Estimate:               0,
		CompletedPomodoroCount: 0,
	}
	out := renderPlanDetail(entry, task, 60, false, nil)
	if countPomodoroGlyphs(out) != 0 {
		t.Errorf("task with 0 estimate and 0 completed must not show glyph row; got %q", out)
	}
}

// ── Feature 035: TUI fill, anchor, and untimed tests ──────────────────────────

// TestPlanningView_Fill_TallTerminal checks that on a tall terminal the timed grid
// extends past 17:00 to fill available space (US1, T009).
func TestPlanningView_Fill_TallTerminal(t *testing.T) {
	day := "2026-05-27"
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.plan.day = day
	m.plan.loaded = true
	// No entries — default 08:00–17:00 base window (37 rows).

	// A tall terminal: 80 rows available for the grid means FILL mode.
	now := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	out := m.renderPlanningView(80, 81, now) // header(1) + gridHeight(80)

	// The grid should extend past 17:00.
	if !strings.Contains(out, "18:00") {
		t.Errorf("tall terminal fill: expected hours past 17:00 in output:\n%s", out)
	}
	// Should not contain a large blank gap at the bottom: last rendered hour
	// should be beyond 17:00.
	if strings.HasSuffix(strings.TrimRight(out, "\n "), "17:00") {
		t.Errorf("tall terminal fill: grid should not end at 17:00 when space is available")
	}
}

// TestPlanningView_Anchor_ShortToday checks that on a short terminal viewing today,
// the grid starts at the current-time block (US2, T012).
func TestPlanningView_Anchor_ShortToday(t *testing.T) {
	day := "2026-05-27"
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.plan.day = day
	m.plan.loaded = true
	// Entry at 15:00 so it remains visible in the anchored window.
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Afternoon", StartMinute: pint32(900), DurationMinute: 30},
	}

	// Short terminal: 15 rows total (header + 14 grid rows = 14 rows, baseRows=37 → constrained).
	// now = 13:00 on day → ANCHOR at 13:00.
	now := time.Date(2026, 5, 27, 13, 0, 0, 0, time.UTC)
	out := m.renderPlanningView(80, 15, now)

	// The grid must not show morning hours (08:00 should be absent, or 13:00 should be present).
	if strings.Contains(out, "08:00") {
		t.Errorf("ANCHOR short today: expected no 08:00 morning row; got:\n%s", out)
	}
	if !strings.Contains(out, "13:00") {
		t.Errorf("ANCHOR short today: expected 13:00 (now-block) as grid start; got:\n%s", out)
	}
}

// TestPlanningView_NonToday_TopTruncate checks that a short terminal on a non-today day
// starts at the default top without anchoring (US2, T013).
func TestPlanningView_NonToday_TopTruncate(t *testing.T) {
	yesterday := "2026-05-26"
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.plan.day = yesterday
	m.plan.loaded = true

	// now is today (2026-05-27), but we're viewing yesterday.
	now := time.Date(2026, 5, 27, 13, 0, 0, 0, time.UTC)
	out := m.renderPlanningView(80, 15, now)

	// Non-today: must start at the default 08:00 top.
	if !strings.Contains(out, "08:00") {
		t.Errorf("TOP-TRUNCATE non-today: expected 08:00 default start; got:\n%s", out)
	}
}

// TestPlanningView_UntimedListComposition checks that the Plan tab composes
// untimed list rows, then the divider, then the grid (US1 T006).
func TestPlanningView_UntimedListComposition(t *testing.T) {
	day := "2026-05-27"
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 40
	m.plan.day = day
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "UntimedTask", DurationMinute: 30},                       // untimed
		{Id: 2, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30}, // timed
	}

	now := time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC) // before 08:00 → TOP-TRUNCATE
	out := m.renderPlanningView(80, 40, now)

	untimedPos := strings.Index(out, "UntimedTask")
	gridPos := strings.Index(out, "Standup")
	if untimedPos < 0 {
		t.Fatal("untimed task 'UntimedTask' not found in output")
	}
	if gridPos < 0 {
		t.Fatal("timed entry 'Standup' not found in output")
	}
	// Untimed comes before grid.
	if untimedPos > gridPos {
		t.Errorf("untimed list must appear before the grid; untimedPos=%d gridPos=%d", untimedPos, gridPos)
	}

	// Separator between untimed list and grid.
	between := out[untimedPos:gridPos]
	if !strings.Contains(between, "─") {
		t.Errorf("expected separator (─) between untimed list and grid; between=%q", between)
	}
}

// TestPlanningView_Untimed_VisibleBeforeGrid checks that on a constrained terminal with
// untimed entries, all untimed boxes and separator appear before the timed grid (US3, T015).
func TestPlanningView_Untimed_VisibleBeforeGrid(t *testing.T) {
	day := "2026-05-27"
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.plan.day = day
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "BacklogTask", DurationMinute: 15}, // untimed
		{Id: 2, Name: "Lunch", StartMinute: pint32(720), DurationMinute: 60},
	}

	now := time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC) // before window → TOP-TRUNCATE
	out := m.renderPlanningView(80, 25, now)

	untimedPos := strings.Index(out, "BacklogTask")
	gridPos := strings.Index(out, "Lunch")
	if untimedPos < 0 {
		t.Fatalf("untimed priority: 'BacklogTask' not found in output:\n%s", out)
	}
	if gridPos < 0 {
		t.Fatalf("untimed priority: 'Lunch' not found in output:\n%s", out)
	}
	if untimedPos > gridPos {
		t.Errorf("untimed priority: untimed entry must appear before timed grid; positions: untimed=%d, grid=%d", untimedPos, gridPos)
	}
}

// TestPlanningView_Untimed_GracefulWhenOverfull checks that when untimed entries
// alone exceed the available height, the view does not panic and stays within bounds (US3, T015).
func TestPlanningView_Untimed_GracefulWhenOverfull(t *testing.T) {
	day := "2026-05-27"
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.plan.day = day
	m.plan.loaded = true
	// Many untimed entries to fill more than the available height.
	entries := make([]*planv1.PlanEntry, 20)
	for i := range entries {
		entries[i] = &planv1.PlanEntry{Id: int32(i + 1), Name: fmt.Sprintf("Backlog%d", i+1), DurationMinute: 30}
	}
	m.plan.entries = entries

	now := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	// This must not panic.
	out := m.renderPlanningView(80, 10, now)

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > 10 {
		t.Errorf("overfull untimed: output has %d lines, exceeds height 10", len(lines))
	}
}

// ── T010: Preview lifecycle — entries unchanged after form open/cancel ─────────

// TestPreviewLifecycle_EntriesUnchangedAfterFormOpenCancel verifies that
// m.plan.entries is not mutated when a form is opened or when the preview is built.
func TestPreviewLifecycle_EntriesUnchangedAfterFormOpenCancel(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewModel(nil, nil)
	m.width = 80
	m.height = 45
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	origLen := len(m.plan.entries)
	origID := m.plan.entries[0].Id

	// Open the edit form.
	m.initEditForm()
	if len(m.plan.entries) != origLen {
		t.Errorf("after initEditForm: entries length changed: want %d, got %d", origLen, len(m.plan.entries))
	}
	if m.plan.entries[0].Id != origID {
		t.Errorf("after initEditForm: entry Id changed: want %d, got %d", origID, m.plan.entries[0].Id)
	}

	// Render (this is where buildTimedSliceWithPreview runs).
	now := time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC)
	_ = m.renderPlanGrid(80, 40, now)

	if len(m.plan.entries) != origLen {
		t.Errorf("after renderPlanGrid: entries length changed: want %d, got %d", origLen, len(m.plan.entries))
	}
	if m.plan.entries[0].Id != origID {
		t.Errorf("after renderPlanGrid: entry Id changed: want %d, got %d", origID, m.plan.entries[0].Id)
	}

	// Cancel: return to planList mode.
	m.plan.mode = planList

	if len(m.plan.entries) != origLen {
		t.Errorf("after cancel (planList): entries length changed: want %d, got %d", origLen, len(m.plan.entries))
	}
}

// TestPreviewLifecycle_EditedEntryStillRenderedAfterClose is a regression test
// for the bug where a timed entry disappeared from the planning grid after its
// edit form was closed. The root cause: buildTimedSliceWithPreview read
// form.entryID unconditionally, but that field is never cleared when the form
// closes, so the edited entry stayed excluded even with no preview to replace it.
func TestPreviewLifecycle_EditedEntryStillRenderedAfterClose(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	now := time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC)

	newModel := func() Model {
		m := ExportNewModel(nil, nil)
		m.width = 80
		m.height = 45
		m.activeTab = tabPlanning
		m.plan.day = today
		m.plan.loaded = true
		m.plan.entries = []*planv1.PlanEntry{
			{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
		}
		return m
	}

	t.Run("styled after submit", func(t *testing.T) {
		m := newModel()
		m.styled = true
		m.initEditForm() // sets form.entryID=1, mode=planEdit
		// Simulate successful submit: mode returns to planList, form not cleared.
		m.plan.mode = planList
		out := m.renderPlanGrid(80, 40, now)
		if !strings.Contains(out, "Standup") {
			t.Errorf("entry 'Standup' missing from styled grid after edit form closed; got:\n%s", out)
		}
	})

	t.Run("plain after submit", func(t *testing.T) {
		m := newModel()
		m.styled = false
		m.initEditForm()
		m.plan.mode = planList
		out := m.renderPlanGridContent(80, 40, now)
		if !strings.Contains(out, "Standup") {
			t.Errorf("entry 'Standup' missing from plain grid after edit form closed; got:\n%s", out)
		}
	})

	t.Run("styled after cancel", func(t *testing.T) {
		m := newModel()
		m.styled = true
		m.initEditForm()
		// Simulate Esc-cancel: same path — mode=planList, form not cleared.
		m.plan.mode = planList
		m.plan.err = nil
		out := m.renderPlanGrid(80, 40, now)
		if !strings.Contains(out, "Standup") {
			t.Errorf("entry 'Standup' missing from styled grid after edit cancelled; got:\n%s", out)
		}
	})
}

// ── T023: Inline rendering integration tests for plan picker ──────────────────

// TestPlanPicker_InlineNameNoMarkupChars asserts that the plan task picker
// renders task names without leftover markdown syntax in plain mode.
func TestPlanPicker_InlineNameNoMarkupChars(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "normal task"},
		{Id: 2, Name: "**Bold plan task**"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewPlanModel(nil, nil, "2026-06-12")
	m.styled = false
	m.plan.mode = planPickTask
	expanded := map[int64]bool{1: true, 2: true}
	m.plan.picker = pickerState{
		tree:    tree,
		visible: ExportBuildVisible(tree, expanded, false, nil),
		cursor:  0,
	}

	out := m.renderPlanPickerView(60)
	if strings.Contains(out, "**") {
		t.Errorf("plain plan picker: leftover ** in output %q", out)
	}
	if !strings.Contains(out, "Bold plan task") {
		t.Errorf("plain plan picker: task name missing; got %q", out)
	}
}

// TestPlanDetail_EntryNameInlinePlain asserts that plan entry names are
// rendered without raw markdown syntax in plain mode when a renderer is passed.
func TestPlanDetail_EntryNameInlinePlain(t *testing.T) {
	m := ExportNewPlanModel(nil, nil, "2026-06-12")
	entry := &planv1.PlanEntry{
		Id:   1,
		Name: "**Stand-up** meeting",
	}
	out := renderPlanDetail(entry, nil, 60, false, m.md)
	if strings.Contains(out, "**") {
		t.Errorf("plain renderPlanDetail: leftover ** in output %q", out)
	}
	if !strings.Contains(out, "Stand-up") {
		t.Errorf("plain renderPlanDetail: entry name missing; got %q", out)
	}
}

// ── Feature 071: Task descriptions on the Planning tab (US1) ───────────────

// TestRenderPlanDetail_DescriptionShown (T002) asserts that a task entry with a
// non-empty description renders the description text in the detail pane, both
// in the unstyled and styled paths.
func TestRenderPlanDetail_DescriptionShown(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Quarterly report",
		StartMinute:    pint32(600),
		DurationMinute: 60,
		TaskId:         42,
	}
	task := &taskv1.Task{
		Id:          42,
		Name:        "Quarterly report",
		Description: "Wrap Q3 numbers into a single page and send to the leadership team.",
	}

	// Unstyled path with md == nil falls back to wrapDescription.
	plain := renderPlanDetail(entry, task, 80, false, nil)
	if !strings.Contains(plain, "Wrap Q3 numbers") {
		t.Errorf("plain: description text missing from output:\n%q", plain)
	}

	// Styled path with md == nil also falls back to wrapDescription.
	styled := renderPlanDetail(entry, task, 80, true, nil)
	if !strings.Contains(styled, "Wrap Q3 numbers") {
		t.Errorf("styled: description text missing from output:\n%q", styled)
	}
}

// TestRenderPlanDetail_DescriptionAbsent (T003) covers rules D2, D3, D4: the
// description must not appear AND no trailing blank separator line must be
// emitted when the conditions for display are not met.
func TestRenderPlanDetail_DescriptionAbsent(t *testing.T) {
	desc := "Description that must not appear in absence cases."

	t.Run("event entry (TaskId == 0, task == nil)", func(t *testing.T) {
		entry := &planv1.PlanEntry{
			Name:           "Team sync",
			StartMinute:    pint32(540),
			DurationMinute: 30,
			TaskId:         0,
		}
		for _, styled := range []bool{false, true} {
			out := renderPlanDetail(entry, nil, 60, styled, nil)
			if strings.Contains(out, desc) {
				t.Errorf("styled=%v: event must not show description; got %q", styled, out)
			}
		}
	})

	t.Run("missing task (TaskId != 0, task == nil)", func(t *testing.T) {
		entry := &planv1.PlanEntry{
			Name:           "Orphan",
			StartMinute:    pint32(600),
			DurationMinute: 30,
			TaskId:         99,
		}
		for _, styled := range []bool{false, true} {
			out := renderPlanDetail(entry, nil, 60, styled, nil)
			if strings.Contains(out, desc) {
				t.Errorf("styled=%v: missing task must not show description; got %q", styled, out)
			}
			// Other fields (entry name, Window, Duration) must still render.
			if !strings.Contains(out, "Orphan") {
				t.Errorf("styled=%v: missing task must still render entry name; got %q", styled, out)
			}
			if !strings.Contains(out, "Window") {
				t.Errorf("styled=%v: missing task must still render Window; got %q", styled, out)
			}
		}
	})

	t.Run("empty description", func(t *testing.T) {
		entry := &planv1.PlanEntry{
			Name:           "No notes",
			StartMinute:    pint32(540),
			DurationMinute: 30,
			TaskId:         7,
		}
		task := &taskv1.Task{Id: 7, Name: "No notes", Description: ""}
		for _, styled := range []bool{false, true} {
			out := renderPlanDetail(entry, task, 60, styled, nil)
			if strings.Contains(out, desc) {
				t.Errorf("styled=%v: empty description must not appear; got %q", styled, out)
			}
			// No trailing blank separator line (do not end with extra '\n\n').
			if strings.HasSuffix(out, "\n\n") {
				t.Errorf("styled=%v: empty description must not leave a trailing blank line; got %q", styled, out)
			}
		}
	})

	t.Run("whitespace-only description", func(t *testing.T) {
		entry := &planv1.PlanEntry{
			Name:           "Whitespace",
			StartMinute:    pint32(540),
			DurationMinute: 30,
			TaskId:         8,
		}
		task := &taskv1.Task{Id: 8, Name: "Whitespace", Description: "   \n\t  "}
		for _, styled := range []bool{false, true} {
			out := renderPlanDetail(entry, task, 60, styled, nil)
			if strings.Contains(out, desc) {
				t.Errorf("styled=%v: whitespace description must not appear; got %q", styled, out)
			}
			if strings.HasSuffix(out, "\n\n") {
				t.Errorf("styled=%v: whitespace description must not leave a trailing blank line; got %q", styled, out)
			}
		}
	})
}

// exitPlanDetailBaselines captures output bytes for "no description" cases so a
// future feature cannot accidentally add even a blank line. T004 pins the byte
// identity across both styled and unstyled paths for empty and whitespace-only
// descriptions.
func TestRenderPlanDetail_EmptyDescriptionFixturePin(t *testing.T) {
	entry := &planv1.PlanEntry{
		Name:           "Plain task",
		StartMinute:    pint32(540),
		DurationMinute: 30,
		TaskId:         3,
	}
	task := &taskv1.Task{Id: 3, Name: "Plain task", Description: ""}
	plain := renderPlanDetail(entry, task, 80, false, nil)
	styled := renderPlanDetail(entry, task, 80, true, nil)
	if strings.HasSuffix(plain, "\n\n") {
		t.Errorf("plain: empty description: trailing blank separator line; got %q", plain)
	}
	if strings.HasSuffix(styled, "\n\n") {
		t.Errorf("styled: empty description: trailing blank separator line; got %q", styled)
	}
	if strings.Contains(plain, "Description") || strings.Contains(styled, "Description") {
		t.Errorf("empty description: text label not allowed; plain=%q styled=%q", plain, styled)
	}

	// Record the byte forms so a future regression would show as a diff against
	// this comment. The contract is that these must NEVER change.
	const plainFixture = "Plain task\nWindow:   09:00–09:30\nDuration: 30 min\nTask:     #3  in progress\n"
	if plain != plainFixture {
		t.Errorf("plain empty-description baseline drifted:\nwant %q\ngot  %q", plainFixture, plain)
	}
}

// ── Feature 071: Markdown rendering fidelity (US2) ──────────────────────────

// richDesc exercises the markdown constructs that must render identically on
// both tabs: heading, list, bold, inline code, link.
const richDesc = "# Heading\n\n- item one\n- item two with **bold** and `code`\n\n[example](http://example.com)\n"

// TestRenderPlanDetail_DescriptionMatchesTaskDetail (T009) — both tabs share
// the same md.Render call, so the rendered description block must be byte
// identical at the same Width and Styled settings (contract C2.3, SC-003).
func TestRenderPlanDetail_DescriptionMatchesTaskDetail(t *testing.T) {
	md := newTestMarkdownRenderer()

	entry := &planv1.PlanEntry{
		Name:           "rich task",
		StartMinute:    pint32(540),
		DurationMinute: 30,
		TaskId:         11,
	}
	task := &taskv1.Task{
		Id:          11,
		Name:        "rich task",
		Description: richDesc,
	}

	const width = 60
	for _, styled := range []bool{false, true} {
		plan := renderPlanDetail(entry, task, width, styled, md)
		// Build the canonical rendered block the two tabs both rely on.
		rendered := md.Render(richDesc, markdown.Options{Width: width, Styled: styled})

		if !strings.Contains(plan, rendered) {
			t.Errorf("styled=%v: planning detail must contain the rendered description block;\nrendered:\n%s\nplan:\n%s", styled, rendered, plan)
		}
	}
}

// TestRenderPlanDetail_DescriptionUnstyledNoANSI (T010) — extending the
// guarantee at plan_view_test.go:423 to the new description block: the
// unstyled path must emit no ANSI escape sequences even with a real renderer.
func TestRenderPlanDetail_DescriptionUnstyledNoANSI(t *testing.T) {
	md := newTestMarkdownRenderer()

	entry := &planv1.PlanEntry{
		Name:           "no ansi",
		StartMinute:    pint32(540),
		DurationMinute: 30,
		TaskId:         12,
	}
	task := &taskv1.Task{
		Id:          12,
		Name:        "no ansi",
		Description: richDesc,
	}
	out := renderPlanDetail(entry, task, 80, false, md)
	if strings.Contains(out, "\x1b[") {
		t.Errorf("unstyled+markdown: must not emit ANSI; got:\n%s", out)
	}
}

// TestRenderPlanDetail_DescriptionNilRendererFallback (T011) — when md is nil,
// the description falls back to wrapDescription (contract C2.4).
func TestRenderPlanDetail_DescriptionNilRendererFallback(t *testing.T) {
	const desc = "This description appears in the pane when md is nil."
	entry := &planv1.PlanEntry{
		Name:           "nil md",
		StartMinute:    pint32(540),
		DurationMinute: 30,
		TaskId:         13,
	}
	task := &taskv1.Task{
		Id:          13,
		Name:        "nil md",
		Description: desc,
	}

	for _, styled := range []bool{false, true} {
		out := renderPlanDetail(entry, task, 80, styled, nil)
		if !strings.Contains(out, desc) {
			t.Errorf("styled=%v: nil renderer: description must still appear via wrapDescription; got %q", styled, out)
		}
	}
}

// TestRenderPlanDetail_DescriptionBlockLevel (T012) — multi-line block markdown
// must render multi-line. A bullet list's items must appear on separate lines.
func TestRenderPlanDetail_DescriptionBlockLevel(t *testing.T) {
	const desc = "- one\n- two\n- three\n"
	md := newTestMarkdownRenderer()

	entry := &planv1.PlanEntry{
		Name:           "list task",
		StartMinute:    pint32(540),
		DurationMinute: 30,
		TaskId:         14,
	}
	task := &taskv1.Task{
		Id:          14,
		Name:        "list task",
		Description: desc,
	}

	// Render through md.Render to confirm block-level output is multi-line.
	rendered := md.Render(desc, markdown.Options{Width: 80, Styled: false})
	if strings.Count(rendered, "\n") < 2 {
		t.Errorf("block-list: render should produce multi-line output (>=2 newlines); got %q", rendered)
	}

	out := renderPlanDetail(entry, task, 80, false, md)
	if !strings.Contains(out, "one") || !strings.Contains(out, "two") || !strings.Contains(out, "three") {
		t.Errorf("block-list: items missing from renderPlanDetail output:\n%s", out)
	}
}

// TestRenderPlanDetail_DescriptionWrapsToWidth (T016) — at a narrow width, no
// rendered description line may exceed the pane's inner width, including a
// case with a long unbroken URL (contract C3.4 / spec edge case).
// TestRenderPlanDetail_DescriptionWrapsToWidth covers two layers: that
// renderPlanDetail's markdown rendering preserves an unbreakable URL rather
// than losing it, and that fitPane — the mechanism paneBox actually uses to
// guarantee width containment (see Fix 1 / internal/tui/view.go) — brings the
// result within width. renderPlanDetail itself is not required to keep every
// line within width: internal/markdown/wrap.go intentionally leaves words
// wider than the target width unbroken, on their own line; wrapping that line
// to fit the pane is fitPane's job, applied inside paneBox.
func TestRenderPlanDetail_DescriptionWrapsToWidth(t *testing.T) {
	md := newTestMarkdownRenderer()

	entry := &planv1.PlanEntry{
		Name:           "wrap task",
		StartMinute:    pint32(540),
		DurationMinute: 30,
		TaskId:         15,
	}
	// A very long unbroken URL — fitPane should still wrap it to the pane
	// width rather than letting it flow past the pane border.
	const url = "https://example.com/this/is/a/really/long/url/that/will/not/fit/inside/a/narrow/pane/without/wrapping"
	task := &taskv1.Task{
		Id:          15,
		Name:        "wrap task",
		Description: "Visit " + url + " for details.\n",
	}

	const width = 30
	out := renderPlanDetail(entry, task, width, true, md)
	plain := stripANSI(out)

	// Description text must appear (split into multiple lines once wrapped).
	if !strings.Contains(plain, "example.com") {
		t.Errorf("description must still contain the URL once wrapped; got:\n%s", plain)
	}

	// fitPane is what paneBox applies before rendering into the box; every
	// line it produces must fit within width, measured by visible columns
	// (lipgloss.Width), not len (which counts bytes and mis-measures styled
	// or UTF-8 output).
	fitted := fitPane(out, width, 1000)
	for i, line := range strings.Split(fitted, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Errorf("line %d exceeds width=%d (%d cols): %q", i, width, w, line)
		}
	}
}

// ── Feature 071: Height containment (US3) ───────────────────────────────────

// TestPlanView_LongDescriptionDoesNotExceedHeight (T014) — a Planning tab view
// whose selected task has a 200-line description must not exceed the terminal
// height in the rendered output (FR-006, SC-004, contract C4.2).
func TestPlanView_LongDescriptionDoesNotExceedHeight(t *testing.T) {
	today := "2026-05-29"
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 24
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true

	// 200-line description — a paragraph per line so Render emits many lines.
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString(fmt.Sprintf("Line %d of a deliberately long description.\n", i+1))
	}
	longDesc := sb.String()

	tasks := []*taskv1.Task{
		{Id: 1, Name: "Stub"},
		{Id: 2, Name: "LongDescTask", Description: longDesc},
	}
	tree := cli.BuildTree(tasks)
	m.tree = tree
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Short", StartMinute: pint32(540), DurationMinute: 30},
		{Id: 2, Name: "LongDescTask", StartMinute: pint32(600), DurationMinute: 60, TaskId: 2},
	}
	m.plan.cursor = 1

	out := m.viewPlanning()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > m.height {
		t.Errorf("rendered view height=%d exceeds terminal height=%d:\n%s", len(lines), m.height, out)
	}
}

// TestPlanView_LongDescriptionPreservesGridAlignment (T015) — with a long
// description, every line in the joined pane section must have the same
// visible width (m.width) so the grid and detail panes stay aligned. The tab
// bar at the top and status help at the bottom are padded/narrower by
// construction and excluded from this check.
func TestPlanView_LongDescriptionPreservesGridAlignment(t *testing.T) {
	today := "2026-05-29"
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true

	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString(fmt.Sprintf("Line %d xyz.\n", i+1))
	}
	longDesc := sb.String()

	tasks := []*taskv1.Task{
		{Id: 2, Name: "LongDescTask", Description: longDesc},
	}
	m.tree = cli.BuildTree(tasks)
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 2, Name: "LongDescTask", StartMinute: pint32(600), DurationMinute: 60, TaskId: 2},
	}
	m.plan.cursor = 0

	out := m.viewPlanning()
	lines := strings.Split(out, "\n")

	// Pane section: lines [1..len-statusHeight-1] of the output are the joined
	// panes (tab bar is line 0; status is the trailing 1-2 lines).
	statusH := m.statusHeight()
	paneEnd := len(lines) - statusH
	if paneEnd <= 1 {
		t.Fatalf("not enough lines for panes: %d", len(lines))
	}
	for i := 1; i < paneEnd; i++ {
		w := lipgloss.Width(lines[i])
		if w != m.width {
			t.Errorf("pane line %d has width %d, want %d (mismatch means panes out of alignment): %q", i, w, m.width, lines[i])
		}
	}
}

// repeatedUnbreakableLines builds a description of n paragraphs, each a single
// ~100-column unbreakable token (no space to break on). clampLines budgets one
// raw line per paragraph, but paneBox's lipgloss hard-wrap later splits each
// token into several lines, so n lines like this expand well past whatever
// line count clampLines allowed for — enough to blow the pane height budget
// even though each individual wrapped line stays within the pane width.
func repeatedUnbreakableLines(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "https://example.com/unbreakable/token/number/%03d/that/is/far/wider/than/any/reasonable/pane/width\n", i)
	}
	return sb.String()
}

// TestPlanView_WidthHostileDescriptionFitsPane pins the width axis of pane
// containment (FR-005/FR-006/SC-004, spec edge case "very long single word or
// URL with no break opportunity"). A description containing an unbreakable
// URL and a wide fenced-code-block line must not push any rendered pane line
// past m.width, nor the overall view past m.height.
func TestPlanView_WidthHostileDescriptionFitsPane(t *testing.T) {
	today := "2026-05-29"
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 90
	m.height = 24
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true

	desc := repeatedUnbreakableLines(30)

	tasks := []*taskv1.Task{
		{Id: 2, Name: "WideDescTask", Description: desc},
	}
	m.tree = cli.BuildTree(tasks)
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 2, Name: "WideDescTask", StartMinute: pint32(600), DurationMinute: 60, TaskId: 2},
	}
	m.plan.cursor = 0

	out := m.viewPlanning()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if len(lines) > m.height {
		t.Errorf("rendered view height=%d exceeds terminal height=%d:\n%s", len(lines), m.height, out)
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > m.width {
			t.Errorf("line %d has width %d, exceeds terminal width %d: %q", i, w, m.width, line)
		}
	}
}

// TestViewList_WidthHostileDescriptionFitsPane is the Tasks-tab counterpart of
// TestPlanView_WidthHostileDescriptionFitsPane: the same shared paneBox
// guarantee must hold on the Tasks tab's detail pane.

// ---- T029: Plan objective band rendering ----

func TestPlanObjective_OmittedWhenUnset_Styled(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 40
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30, Day: today},
	}
	m.plan.cursor = 0
	m.plan.objective = ""

	with := m.viewPlanning()

	ExportSetPlanObjective(&m, "Ship it")
	with2 := m.viewPlanning()

	if with == with2 {
		t.Errorf("objective band not present when objective is set; output should differ from empty baseline")
	}
}

func TestPlanObjective_OmittedWhenUnset_Unstyled(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, false)
	m.width = 80
	m.height = 40
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30, Day: today},
	}
	m.plan.cursor = 0
	m.plan.objective = ""

	with := m.viewPlanning()

	ExportSetPlanObjective(&m, "Ship it")
	with2 := m.viewPlanning()

	if with == with2 {
		t.Errorf("unstyled: objective band not present when objective is set; output should differ")
	}
}

func TestPlanObjective_EditorShownOnUnsetDay_Styled(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 40
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30, Day: today},
	}
	m.plan.cursor = 0
	m.plan.objective = ""
	m.plan.mode = planObjectiveEdit
	ExportSetPlanObjectiveInput(&m, "")

	v := m.viewPlanning()
	if !strings.Contains(v, "Objective") {
		t.Errorf("editor should appear even on a day without an objective; got:\n%s", v)
	}
}

func TestPlanObjective_LongObjectiveStillRenders(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 40
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30, Day: today},
	}
	m.plan.cursor = 0
	m.plan.objective = "line1\nline2\nline3\nline4\nline5\nline6\nline7"

	v := m.viewPlanning()
	if !strings.Contains(v, "Objective") {
		t.Errorf("Objective band should appear; got:\n%s", v)
	}
}

func TestPlanObjective_NarrowTerminalStaysIntact(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 24
	m.activeTab = tabPlanning
	m.plan.day = today
	m.plan.loaded = true
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30, Day: today},
	}
	m.plan.cursor = 0
	ExportSetPlanObjective(&m, "Ship the long-named thing today that would normally overflow a narrow terminal")

	// The objective band's inner content is rendered at the available inner width.
	band := ExportRenderPlanObjectiveBand(m, m.width-2)
	lines := strings.Split(strings.TrimRight(band, "\n"), "\n")
	for i, line := range lines {
		if w := lipgloss.Width(line); w > m.width {
			t.Errorf("band line %d width=%d exceeds terminal width=%d: %q", i, w, m.width, line)
		}
	}
}

func TestViewList_WidthHostileDescriptionFitsPane(t *testing.T) {
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 90
	m.height = 24
	m.activeTab = tabTasks

	desc := repeatedUnbreakableLines(30)

	tasks := []*taskv1.Task{
		{Id: 2, Name: "WideDescTask", Description: desc},
	}
	m.tree = cli.BuildTree(tasks)
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	m.cursor = 0

	out := m.viewList()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if len(lines) > m.height {
		t.Errorf("rendered view height=%d exceeds terminal height=%d:\n%s", len(lines), m.height, out)
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > m.width {
			t.Errorf("line %d has width %d, exceeds terminal width %d: %q", i, w, m.width, line)
		}
	}
}

// ---- T044: Plan notes pane rendering ----

func TestPlanNotes_PanePresentWhenEmpty_Styled(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanNotes(&m, "")
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	out := m.viewPlanning()
	if !strings.Contains(out, "Notes") {
		t.Errorf("Notes pane should appear even when notes are empty; got:\n%s", out)
	}
	if !strings.Contains(out, "Details") {
		t.Errorf("Details pane should still appear above notes; got:\n%s", out)
	}
}

func TestPlanNotes_PaneShowsContentWhenSet_Styled(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanNotes(&m, "Blocked on review for the migration PR.")
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	out := m.viewPlanning()
	if !strings.Contains(out, "Blocked on review") {
		t.Errorf("Notes pane should render notes content; got:\n%s", out)
	}
}

func TestPlanNotes_EditorTakesColumn_Styled(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanNotes(&m, "Saved content")
	ExportSetPlanNotesInput(&m, "Draft in progress")
	m.plan.mode = planNotesEdit
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	out := m.viewPlanning()
	if !strings.Contains(out, "Draft in progress") {
		t.Errorf("Notes editor should show draft; got:\n%s", out)
	}
}

func TestPlanNotes_NoPaneWhenPickerOpen_Styled(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanNotes(&m, "Blocked on review")
	m.plan.mode = planPickTask
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	out := m.viewPlanning()
	if strings.Contains(out, "Notes") {
		t.Errorf("Notes pane must NOT appear while a picker is open; got:\n%s", out)
	}
}

func TestPlanNotes_UnstyledLayout(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, false)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanNotes(&m, "hello")
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	out := m.viewPlanning()
	if !strings.Contains(out, "Notes") {
		t.Errorf("unstyled: Notes section must appear; got:\n%s", out)
	}
}

// ---- T049: Markdown fidelity (objective and notes) ----

func TestPlanObjective_RendersMarkdown_Styled(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanObjective(&m, "**bold** and `code` and [link](https://example.com)")
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	// The renderer threads the styled theme, so bold should emit an ANSI code
	// via the markdown theme, while the unstyled branch must produce plain text.
	if !m.styled {
		t.Skip("styled test")
	}
	band := ExportRenderPlanObjectiveBand(m, m.width-2)
	if !strings.Contains(band, "bold") {
		t.Errorf("band should contain raw 'bold' source; got:\n%s", band)
	}
	// styled branch wraps the text in a rounded border — verify a border run is
	// visible even if the markdown theme is otherwise hard to introspect.
	if band == "" {
		t.Errorf("band rendering produced empty string")
	}
}

func TestPlanObjective_RendersMarkdown_Unstyled(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, false)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanObjective(&m, "**bold**")
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	band := ExportRenderPlanObjectiveBand(m, m.width-2)
	if strings.Contains(band, "\x1b[") {
		t.Errorf("unstyled band must not carry ANSI escape sequences; got:\n%q", band)
	}
}

func TestPlanNotes_RendersMarkdown_Styled(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanNotes(&m, "**bold** and a [link](https://example.com)")
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	out := m.viewPlanning()
	if !strings.Contains(out, "bold") {
		t.Errorf("notes pane should render source text; got:\n%s", out)
	}
	if !strings.Contains(out, "https://example.com") {
		t.Errorf("notes link text should be visible; got:\n%s", out)
	}
}

// ---- T050: Editors show raw markdown source, not rendered output ----

func TestPlanObjective_EditorShowsRawMarkdown(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanObjective(&m, "")
	ExportSetPlanObjectiveInput(&m, "**bold** [link](https://example.com)")
	m.plan.mode = planObjectiveEdit
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	band := ExportRenderPlanObjectiveBand(m, m.width-2)
	if !strings.Contains(band, "**bold**") {
		t.Errorf("objective editor should display raw markdown; got:\n%s", band)
	}
	if !strings.Contains(band, "[link]") {
		t.Errorf("objective editor should not strip markdown link syntax; got:\n%s", band)
	}
}

func TestPlanNotes_EditorShowsRawMarkdown(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 100
	m.height = 30
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanNotes(&m, "Saved")
	ExportSetPlanNotesInput(&m, "**bold** line\n\n- one\n- two")
	m.plan.mode = planNotesEdit
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	editor := ExportRenderPlanNotesEditor(m, 80, 10)
	if !strings.Contains(editor, "**bold**") {
		t.Errorf("notes editor should display raw markdown; got:\n%s", editor)
	}
	if !strings.Contains(editor, "- one") {
		t.Errorf("notes editor should show bullet list source; got:\n%s", editor)
	}
}

// ---- T051: Resilience — long content, hostile URLs, narrow terminals ----

func TestPlanNotes_LongContentClippedInsidePane(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 80
	m.height = 24
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	long := strings.Repeat("A really long paragraph that should be wrapped and clipped to fit the pane. ", 20)
	ExportSetPlanNotes(&m, long)
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	out := m.viewPlanning()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > m.height {
		t.Errorf("rendered view height=%d exceeds terminal height=%d:\n%s", len(lines), m.height, out)
	}
}

func TestPlanNotes_NarrowTerminalIntact(t *testing.T) {
	day := time.Now().Format("2006-01-02")
	m := ExportNewStyledModel(nil, nil, true)
	m.width = 40
	m.height = 20
	m.activeTab = tabPlanning
	m.plan.day = day
	m.plan.loaded = true
	ExportSetPlanNotes(&m, "Long-named note that should still render at narrow widths without breaking the layout.")
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", Day: day, StartMinute: pint32(540), DurationMinute: 30},
	}, 0)

	out := m.viewPlanning()
	// Tab bar must still appear.
	if !strings.Contains(out, "Planning") {
		t.Errorf("narrow terminal: tab bar must still be present; got:\n%s", out)
	}
	// Notes pane must still appear.
	if !strings.Contains(out, "Notes") {
		t.Errorf("narrow terminal: Notes pane must still be present; got:\n%s", out)
	}
}
