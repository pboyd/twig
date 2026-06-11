package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pboyd/twig/internal/report"
)

var reportNow = time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)

// buildReportModel creates a model with the Report tab active and the given state.
func buildReportModel(data reportState) Model {
	m := ExportNewReportModel(nil)
	m.width = 80
	m.height = 30
	ExportSetReportData(&m, data)
	ExportSetNowFunc(&m, func() time.Time { return reportNow })
	return m
}

func makeRecentPeriod() report.Period {
	p, _ := report.ParsePeriod("recent", "", "", reportNow)
	return p
}

func makeQuarterPeriod() report.Period {
	p, _ := report.ParsePeriod("quarter", "", "", reportNow)
	return p
}

// TestReportView_DefaultPresetIsRecent verifies that a freshly loaded model's
// period label is for the "recent" preset.
func TestReportView_DefaultPresetIsRecent(t *testing.T) {
	m := ExportNewReportModel(nil)
	ExportSetNowFunc(&m, func() time.Time { return reportNow })
	presets := report.PresetOrder()
	if presets[m.reportData.presetIdx] != "recent" {
		t.Errorf("default preset = %q, want 'recent'", presets[m.reportData.presetIdx])
	}
}

// TestReportView_LoadingState checks that an unloaded state shows "Loading...".
func TestReportView_LoadingState(t *testing.T) {
	m := buildReportModel(reportState{loaded: false})
	body := m.renderReportBody(80, 20)
	if !strings.Contains(body, "Loading...") {
		t.Errorf("expected 'Loading...' in body, got: %q", body)
	}
}

// TestReportView_EmptyState checks the playful empty state message.
func TestReportView_EmptyState(t *testing.T) {
	p := makeRecentPeriod()
	m := buildReportModel(reportState{
		loaded:    true,
		period:    p,
		dayGroups: nil,
		totals:    report.Totals{},
	})
	lines := m.renderReportLines(80)
	content := strings.Join(lines, "\n")
	if !strings.Contains(content, "Nothing checked off") {
		t.Errorf("expected empty state copy, got: %q", content)
	}
	if !strings.Contains(content, "potential") {
		t.Errorf("empty state should be playful (contain 'potential'), got: %q", content)
	}
}

// TestReportView_DayGroupedLayout verifies day-grouped rendering for ≤14 day periods.
func TestReportView_DayGroupedLayout(t *testing.T) {
	p := makeRecentPeriod()

	jun11 := time.Date(2025, 6, 11, 10, 0, 0, 0, time.Local)
	jun10 := time.Date(2025, 6, 10, 15, 0, 0, 0, time.Local)

	groups := []report.DayGroup{
		{
			Day: time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local),
			Entries: []report.Entry{
				{TaskID: 1, Name: "Write summary", CompletedAt: jun11},
			},
		},
		{
			Day: time.Date(2025, 6, 10, 0, 0, 0, 0, time.Local),
			Entries: []report.Entry{
				{TaskID: 2, Name: "Fix proxy", CompletedAt: jun10},
				{TaskID: 3, Name: "Wire download page", CompletedAt: jun10, ParentName: "Web app polish"},
			},
		},
	}

	m := buildReportModel(reportState{
		loaded:    true,
		period:    p,
		dayGroups: groups,
		totals:    report.Totals{TasksCompleted: 3, PomodorosCompleted: 5},
	})

	lines := m.renderReportLines(80)
	content := strings.Join(lines, "\n")

	// Header.
	if !strings.Contains(content, "Yesterday & Today") {
		t.Errorf("expected period label in header, got: %q", content)
	}

	// Task names.
	if !strings.Contains(content, "Write summary") {
		t.Errorf("expected 'Write summary' in output, got: %q", content)
	}
	if !strings.Contains(content, "Fix proxy") {
		t.Errorf("expected 'Fix proxy' in output, got: %q", content)
	}

	// Parent context.
	if !strings.Contains(content, "Web app polish") {
		t.Errorf("expected parent context 'Web app polish', got: %q", content)
	}

	// Totals.
	if !strings.Contains(content, "3 tasks done") {
		t.Errorf("expected totals in output, got: %q", content)
	}
	if !strings.Contains(content, "5 pomodoros burned") {
		t.Errorf("expected pomodoro total in output, got: %q", content)
	}
}

// TestReportView_DayGroupedOrdering verifies most-recent-day-first ordering.
func TestReportView_DayGroupedOrdering(t *testing.T) {
	p := makeRecentPeriod()

	groups := []report.DayGroup{
		{Day: time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local), Entries: []report.Entry{{Name: "Jun11 task"}}},
		{Day: time.Date(2025, 6, 10, 0, 0, 0, 0, time.Local), Entries: []report.Entry{{Name: "Jun10 task"}}},
	}

	m := buildReportModel(reportState{
		loaded:    true,
		period:    p,
		dayGroups: groups,
		totals:    report.Totals{},
	})

	lines := m.renderReportLines(80)
	content := strings.Join(lines, "\n")

	idx11 := strings.Index(content, "Jun 11")
	idx10 := strings.Index(content, "Jun 10")
	if idx11 < 0 || idx10 < 0 {
		t.Fatalf("missing day headers in output: %q", content)
	}
	if idx11 > idx10 {
		t.Errorf("Jun 11 should appear before Jun 10 (most recent first)")
	}
}

// TestReportView_TotalsLine checks the totals grammar (singular/plural).
func TestReportView_TotalsLine(t *testing.T) {
	p := makeRecentPeriod()

	t.Run("plural tasks and pomodoros", func(t *testing.T) {
		m := buildReportModel(reportState{
			loaded: true,
			period: p,
			dayGroups: []report.DayGroup{
				{Day: time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local), Entries: []report.Entry{{Name: "t1"}, {Name: "t2"}}},
			},
			totals: report.Totals{TasksCompleted: 2, PomodorosCompleted: 3},
		})
		content := strings.Join(m.renderReportLines(80), "\n")
		if !strings.Contains(content, "2 tasks done") {
			t.Errorf("expected '2 tasks done', got: %q", content)
		}
		if !strings.Contains(content, "3 pomodoros burned") {
			t.Errorf("expected '3 pomodoros burned', got: %q", content)
		}
	})

	t.Run("singular task and pomodoro", func(t *testing.T) {
		m := buildReportModel(reportState{
			loaded: true,
			period: p,
			dayGroups: []report.DayGroup{
				{Day: time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local), Entries: []report.Entry{{Name: "t1"}}},
			},
			totals: report.Totals{TasksCompleted: 1, PomodorosCompleted: 1},
		})
		content := strings.Join(m.renderReportLines(80), "\n")
		if !strings.Contains(content, "1 task done") {
			t.Errorf("expected '1 task done', got: %q", content)
		}
		if !strings.Contains(content, "1 pomodoro burned") {
			t.Errorf("expected '1 pomodoro burned', got: %q", content)
		}
	})
}

// TestReportView_TabBarHasReport verifies the tab bar renders "Report".
func TestReportView_TabBarHasReport(t *testing.T) {
	m := buildReportModel(reportState{loaded: false})
	bar := m.renderTabBar(80)
	if !strings.Contains(bar, "Report") {
		t.Errorf("tab bar missing 'Report': %q", bar)
	}
}

// TestReportView_AccomplishmentLayout verifies the accomplishment-grouped layout for >14 day periods.
func TestReportView_AccomplishmentLayout(t *testing.T) {
	p := makeQuarterPeriod()
	if p.ReportLayout() != report.AccomplishmentGrouped {
		t.Skip("quarter period unexpectedly short; skip")
	}

	jun2 := time.Date(2025, 6, 2, 12, 0, 0, 0, time.Local)
	finished := []report.AccomplishmentGroup{
		{
			TopLevelID:          1,
			TopLevelName:        "Ship alternate profiles",
			TopLevelCompletedAt: jun2,
			Finished:            true,
			Entries: []report.Entry{
				{TaskID: 2, Name: "Config schema", CompletedAt: jun2, Depth: 1},
			},
		},
	}
	ongoing := []report.AccomplishmentGroup{
		{
			TopLevelID:   3,
			TopLevelName: "Web app polish",
			Finished:     false,
			Entries: []report.Entry{
				{TaskID: 4, Name: "Wire up download page", CompletedAt: jun2, Depth: 1},
			},
		},
	}

	m := buildReportModel(reportState{
		loaded:   true,
		period:   p,
		finished: finished,
		ongoing:  ongoing,
		totals:   report.Totals{TasksCompleted: 2, PomodorosCompleted: 4},
	})

	content := strings.Join(m.renderReportLines(80), "\n")

	if !strings.Contains(content, "Finished") {
		t.Errorf("expected 'Finished' section header, got: %q", content)
	}
	if !strings.Contains(content, "Progress on ongoing work") {
		t.Errorf("expected 'Progress on ongoing work' section header, got: %q", content)
	}
	if !strings.Contains(content, "Ship alternate profiles") {
		t.Errorf("expected top-level name in Finished, got: %q", content)
	}
	if !strings.Contains(content, "Ship alternate profiles  (done Jun 2)") {
		t.Errorf("expected '(done Jun 2)' on the finished headline, got: %q", content)
	}
	if !strings.Contains(content, "Web app polish") {
		t.Errorf("expected ongoing group name, got: %q", content)
	}
}

// TestReportKey_QuitGuardConfirmable verifies that the quit confirmation on the
// Report tab can actually be answered: y quits, n dismisses (regression: the
// prompt used to dead-end because handleReportKey never processed it).
func TestReportKey_QuitGuardConfirmable(t *testing.T) {
	newModelWithPom := func() Model {
		m := buildReportModel(reportState{loaded: true, period: makeRecentPeriod()})
		m.pom = &activePom{taskID: 1, taskName: "task-one", startAt: reportNow}
		return m
	}

	t.Run("q prompts instead of quitting", func(t *testing.T) {
		m := newModelWithPom()
		next, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
		nm := next.(Model)
		if !nm.confirmingQuit {
			t.Error("expected confirmingQuit after q with active pom")
		}
		if cmd != nil {
			t.Error("should not quit immediately")
		}
	})

	t.Run("y quits while confirming", func(t *testing.T) {
		m := newModelWithPom()
		m.confirmingQuit = true
		_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
		mustQuit(t, cmd)
	})

	t.Run("n dismisses the prompt", func(t *testing.T) {
		m := newModelWithPom()
		m.confirmingQuit = true
		next, cmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
		nm := next.(Model)
		if nm.confirmingQuit {
			t.Error("confirmingQuit should be cleared after n")
		}
		if cmd != nil {
			t.Error("should not quit after n")
		}
	})
}

// TestReportKey_ScrollClamped verifies that scrolling past the end of the
// content stops at the last page instead of growing unbounded (regression:
// the view used to snap back to the top once scroll exceeded the line count).
func TestReportKey_ScrollClamped(t *testing.T) {
	groups := []report.DayGroup{
		{Day: time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local), Entries: []report.Entry{
			{Name: "t1"}, {Name: "t2"}, {Name: "t3"}, {Name: "t4"}, {Name: "t5"},
		}},
	}
	m := buildReportModel(reportState{
		loaded:    true,
		period:    makeRecentPeriod(),
		dayGroups: groups,
		totals:    report.Totals{TasksCompleted: 5},
	})
	m.height = 8 // small viewport so content overflows

	lineCount := len(m.renderReportLines(m.width))
	maxScroll := lineCount - m.reportBodyHeight()
	if maxScroll < 1 {
		t.Fatalf("test setup: content must overflow viewport (lines=%d, height=%d)", lineCount, m.reportBodyHeight())
	}

	cur := m
	for i := 0; i < lineCount+10; i++ {
		next, _ := cur.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
		cur = next.(Model)
	}

	if cur.reportData.scroll != maxScroll {
		t.Errorf("scroll = %d after over-scrolling, want clamped at %d", cur.reportData.scroll, maxScroll)
	}
	body := cur.renderReportBody(cur.width, cur.reportBodyHeight())
	if strings.Contains(body, "What you got done") {
		t.Errorf("over-scrolled view should not snap back to the header, got: %q", body)
	}
}
