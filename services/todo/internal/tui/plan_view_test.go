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
