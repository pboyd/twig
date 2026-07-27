package tui

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/report"
)

// TestBackgroundLoad_PreservesCursorIdentity checks that a background load keeps
// the highlight on the same item even when a new item is inserted above it.
func TestBackgroundLoad_PreservesCursorIdentity(t *testing.T) {
	t.Run("Tasks", func(t *testing.T) {
		tasks := []*taskv1.Task{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
		m := ExportNewModel(&fakeTaskClient{listTasksResp: tasks}, cli.BuildTree(tasks))
		m.cursor = 1 // on task b

		newTasks := []*taskv1.Task{{Id: 3, Name: "c"}, {Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
		next, _ := m.Update(listTasksResultMsg{tree: cli.BuildTree(newTasks), bg: true})
		nm := next.(Model)
		if nm.cursor < 0 || nm.cursor >= len(nm.visible) || nm.visible[nm.cursor].node.Task.Id != 2 {
			t.Errorf("expected cursor on task b (id 2), got cursor %d", nm.cursor)
		}
	})

	t.Run("Goals", func(t *testing.T) {
		goals := []*goalv1.Goal{
			{Id: 1, Name: "a", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
			{Id: 2, Name: "b", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
		}
		m := ExportNewGoalModel(&fakeTaskClient{}, goals)
		m.goal.cursor = 1 // on goal b
		m.goal.loaded = true

		newGoals := []*goalv1.Goal{
			{Id: 3, Name: "c", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
			{Id: 1, Name: "a", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
			{Id: 2, Name: "b", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
		}
		next, _ := m.Update(listGoalsResultMsg{goals: newGoals, bg: true})
		nm := next.(Model)
		visible := visibleGoals(nm.goal.goals, nm.goal.showAll)
		if nm.goal.cursor < 0 || nm.goal.cursor >= len(visible) || visible[nm.goal.cursor].Id != 2 {
			t.Errorf("expected goal cursor on goal b (id 2), got cursor %d", nm.goal.cursor)
		}
	})

	t.Run("Plan", func(t *testing.T) {
		entries := []*planv1.PlanEntry{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
		m := ExportNewPlanModel(&fakeTaskClient{}, &fakePlanClient{}, "2026-05-27")
		ExportSetPlanEntries(&m, entries, 1)

		newEntries := []*planv1.PlanEntry{{Id: 3, Name: "c"}, {Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
		next, _ := m.Update(planEntriesMsg{entries: newEntries, bg: true, bgDay: "2026-05-27"})
		nm := next.(Model)
		if nm.plan.cursor != 2 {
			t.Errorf("expected plan cursor 2 (still on entry b), got %d", nm.plan.cursor)
		}
	})

	t.Run("Report", func(t *testing.T) {
		// Report has no cursor; verify the scroll is not disturbed.
		m := ExportNewReportModel(&fakeTaskClient{})
		m.reportData.scroll = 5
		m.reportData.loaded = true

		next, _ := m.Update(reportResultMsg{bg: true})
		nm := next.(Model)
		if nm.reportData.scroll != 5 {
			t.Errorf("expected report scroll 5, got %d", nm.reportData.scroll)
		}
	})
}

// TestBackgroundLoad_CursorClampsWhenItemMissing checks that the cursor lands on
// a valid index when the previously selected item is absent from the reloaded data.
func TestBackgroundLoad_CursorClampsWhenItemMissing(t *testing.T) {
	t.Run("Tasks", func(t *testing.T) {
		tasks := []*taskv1.Task{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
		m := ExportNewModel(&fakeTaskClient{listTasksResp: tasks}, cli.BuildTree(tasks))
		m.cursor = 1

		next, _ := m.Update(listTasksResultMsg{tree: cli.BuildTree([]*taskv1.Task{{Id: 3, Name: "c"}}), bg: true})
		nm := next.(Model)
		if nm.cursor != 0 {
			t.Errorf("expected cursor clamped to 0, got %d", nm.cursor)
		}
	})

	t.Run("Goals", func(t *testing.T) {
		goals := []*goalv1.Goal{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
		m := ExportNewGoalModel(&fakeTaskClient{}, goals)
		m.goal.cursor = 1
		m.goal.loaded = true

		next, _ := m.Update(listGoalsResultMsg{goals: []*goalv1.Goal{{Id: 3, Name: "c"}}, bg: true})
		nm := next.(Model)
		if nm.goal.cursor != 0 {
			t.Errorf("expected goal cursor clamped to 0, got %d", nm.goal.cursor)
		}
	})

	t.Run("Plan", func(t *testing.T) {
		entries := []*planv1.PlanEntry{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
		m := ExportNewPlanModel(&fakeTaskClient{}, &fakePlanClient{}, "2026-05-27")
		ExportSetPlanEntries(&m, entries, 1)

		next, _ := m.Update(planEntriesMsg{entries: []*planv1.PlanEntry{{Id: 3, Name: "c"}}, bg: true, bgDay: "2026-05-27"})
		nm := next.(Model)
		if nm.plan.cursor != 0 {
			t.Errorf("expected plan cursor clamped to 0, got %d", nm.plan.cursor)
		}
	})
}

// TestBackgroundLoad_ErrorDiscarded checks that a background load carrying an
// error leaves the tab's data and err field untouched, while a user-initiated
// load carrying the same error sets the err field.
func TestBackgroundLoad_ErrorDiscarded(t *testing.T) {
	boom := errors.New("boom")

	t.Run("Tasks", func(t *testing.T) {
		m := ExportNewModel(&fakeTaskClient{}, nil)
		m.err = errors.New("existing")

		next, _ := m.Update(listTasksResultMsg{err: boom, bg: true})
		nm := next.(Model)
		if nm.err == nil || nm.err.Error() != "existing" {
			t.Errorf("bg error should not replace existing err, got %v", nm.err)
		}

		next, _ = m.Update(listTasksResultMsg{err: boom, bg: false})
		nm = next.(Model)
		if nm.err == nil || nm.err.Error() != "boom" {
			t.Errorf("user error should set err, got %v", nm.err)
		}
	})

	t.Run("Goals", func(t *testing.T) {
		m := ExportNewGoalModel(&fakeTaskClient{}, nil)
		m.goal.err = errors.New("existing")

		next, _ := m.Update(listGoalsResultMsg{err: boom, bg: true})
		nm := next.(Model)
		if nm.goal.err == nil || nm.goal.err.Error() != "existing" {
			t.Errorf("bg error should not replace existing goal err, got %v", nm.goal.err)
		}

		next, _ = m.Update(listGoalsResultMsg{err: boom, bg: false})
		nm = next.(Model)
		if nm.goal.err == nil || nm.goal.err.Error() != "boom" {
			t.Errorf("user error should set goal err, got %v", nm.goal.err)
		}
	})

	t.Run("Plan", func(t *testing.T) {
		m := ExportNewPlanModel(&fakeTaskClient{}, &fakePlanClient{}, "2026-05-27")
		m.plan.entries = []*planv1.PlanEntry{{Id: 1, Name: "a"}}
		m.plan.loaded = true
		m.plan.err = errors.New("existing")

		next, _ := m.Update(planEntriesMsg{err: boom, bg: true, bgDay: "2026-05-27"})
		nm := next.(Model)
		if nm.plan.err == nil || nm.plan.err.Error() != "existing" {
			t.Errorf("bg error should not replace existing plan err, got %v", nm.plan.err)
		}
		if len(nm.plan.entries) != 1 {
			t.Errorf("bg error should not change plan entries, got %d", len(nm.plan.entries))
		}

		next, _ = m.Update(planEntriesMsg{err: boom, bg: false, bgDay: ""})
		nm = next.(Model)
		if nm.plan.err == nil || nm.plan.err.Error() != "boom" {
			t.Errorf("user error should set plan err, got %v", nm.plan.err)
		}
	})

	t.Run("Report", func(t *testing.T) {
		m := ExportNewReportModel(&fakeTaskClient{})
		m.reportData.err = errors.New("existing")
		m.reportData.scroll = 5

		next, _ := m.Update(reportResultMsg{err: boom, bg: true})
		nm := next.(Model)
		if nm.reportData.err == nil || nm.reportData.err.Error() != "existing" {
			t.Errorf("bg error should not replace existing report err, got %v", nm.reportData.err)
		}
		if nm.reportData.scroll != 5 {
			t.Errorf("bg error should not change report scroll, got %d", nm.reportData.scroll)
		}

		next, _ = m.Update(reportResultMsg{err: boom, bg: false})
		nm = next.(Model)
		if nm.reportData.err == nil || nm.reportData.err.Error() != "boom" {
			t.Errorf("user error should set report err, got %v", nm.reportData.err)
		}
	})
}

// TestBackgroundLoad_SuccessKeepsExistingError checks that a successful
// background load does not clear an error that is already displayed.
func TestBackgroundLoad_SuccessKeepsExistingError(t *testing.T) {
	t.Run("Tasks", func(t *testing.T) {
		m := ExportNewModel(&fakeTaskClient{}, nil)
		m.err = errors.New("existing")

		next, _ := m.Update(listTasksResultMsg{tree: cli.BuildTree([]*taskv1.Task{}), bg: true})
		nm := next.(Model)
		if nm.err == nil || nm.err.Error() != "existing" {
			t.Errorf("expected existing err preserved, got %v", nm.err)
		}
	})

	t.Run("Goals", func(t *testing.T) {
		m := ExportNewGoalModel(&fakeTaskClient{}, nil)
		m.goal.err = errors.New("existing")

		next, _ := m.Update(listGoalsResultMsg{goals: []*goalv1.Goal{}, bg: true})
		nm := next.(Model)
		if nm.goal.err == nil || nm.goal.err.Error() != "existing" {
			t.Errorf("expected existing goal err preserved, got %v", nm.goal.err)
		}
	})

	t.Run("Plan", func(t *testing.T) {
		m := ExportNewPlanModel(&fakeTaskClient{}, &fakePlanClient{}, "2026-05-27")
		m.plan.err = errors.New("existing")

		next, _ := m.Update(planEntriesMsg{entries: []*planv1.PlanEntry{}, bg: true, bgDay: "2026-05-27"})
		nm := next.(Model)
		if nm.plan.err == nil || nm.plan.err.Error() != "existing" {
			t.Errorf("expected existing plan err preserved, got %v", nm.plan.err)
		}
	})

	t.Run("Report", func(t *testing.T) {
		m := ExportNewReportModel(&fakeTaskClient{})
		m.reportData.err = errors.New("existing")

		next, _ := m.Update(reportResultMsg{bg: true})
		nm := next.(Model)
		if nm.reportData.err == nil || nm.reportData.err.Error() != "existing" {
			t.Errorf("expected existing report err preserved, got %v", nm.reportData.err)
		}
	})
}

// TestBackgroundLoad_ReportScrollPreserved checks that a background report load
// preserves the scroll offset while a user-initiated load resets it.
func TestBackgroundLoad_ReportScrollPreserved(t *testing.T) {
	m := ExportNewReportModel(&fakeTaskClient{})
	m.reportData.scroll = 5

	next, _ := m.Update(reportResultMsg{bg: true})
	nm := next.(Model)
	if nm.reportData.scroll != 5 {
		t.Errorf("bg load: expected scroll 5, got %d", nm.reportData.scroll)
	}

	next, _ = nm.Update(reportResultMsg{bg: false})
	nm = next.(Model)
	if nm.reportData.scroll != 0 {
		t.Errorf("user load: expected scroll 0, got %d", nm.reportData.scroll)
	}
}

// TestBackgroundLoad_PlanWrongDayDiscarded checks that a background plan load
// whose day no longer matches the current view is discarded.
func TestBackgroundLoad_PlanWrongDayDiscarded(t *testing.T) {
	entries := []*planv1.PlanEntry{{Id: 1, Name: "a"}}
	m := ExportNewPlanModel(&fakeTaskClient{}, &fakePlanClient{}, "2026-05-27")
	ExportSetPlanEntries(&m, entries, 0)

	newEntries := []*planv1.PlanEntry{{Id: 2, Name: "b"}}
	next, _ := m.Update(planEntriesMsg{entries: newEntries, bg: true, bgDay: "2026-05-28"})
	nm := next.(Model)
	if len(nm.plan.entries) != 1 || nm.plan.entries[0].Id != 1 {
		t.Errorf("plan entries should not be overwritten by wrong-day bg load, got %v", nm.plan.entries)
	}
}

// TestLoad_PreservesMode checks that no load handler modifies the interactive mode.
func TestLoad_PreservesMode(t *testing.T) {
	cases := []struct {
		name     string
		msg      tea.Msg
		setMode  func(*Model)
		getMode  func(Model) int
		wantMode int
	}{
		{
			name: "listTasksResultMsg",
			msg:  listTasksResultMsg{tree: cli.BuildTree([]*taskv1.Task{}), bg: true},
			setMode: func(m *Model) {
				m.mode = modeEdit
			},
			getMode:  func(m Model) int { return int(m.mode) },
			wantMode: int(modeEdit),
		},
		{
			name: "listGoalsResultMsg",
			msg:  listGoalsResultMsg{goals: []*goalv1.Goal{}, bg: true},
			setMode: func(m *Model) {
				m.activeTab = tabGoals
				m.mode = modeEdit
				m.goal.mode = goalEdit
			},
			getMode:  func(m Model) int { return int(m.goal.mode) },
			wantMode: int(goalEdit),
		},
		{
			name: "planEntriesMsg",
			msg:  planEntriesMsg{entries: []*planv1.PlanEntry{}, bg: true, bgDay: "2026-05-27"},
			setMode: func(m *Model) {
				m.activeTab = tabPlanning
				m.mode = modeEdit
				m.plan.mode = planEdit
			},
			getMode:  func(m Model) int { return int(m.plan.mode) },
			wantMode: int(planEdit),
		},
		{
			name: "reportResultMsg",
			msg:  reportResultMsg{bg: true},
			setMode: func(m *Model) {
				m.activeTab = tabReport
				m.mode = modeHelp
			},
			getMode:  func(m Model) int { return int(m.mode) },
			wantMode: int(modeHelp),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := ExportNewPlanModel(&fakeTaskClient{}, &fakePlanClient{}, "2026-05-27")
			m.client = &fakeTaskClient{}
			m.goalClient = &fakeGoalClient{}
			m.reportData.period = reportPeriodForTest()
			tc.setMode(&m)

			next, _ := m.Update(tc.msg)
			nm := next.(Model)
			if tc.getMode(nm) != tc.wantMode {
				t.Errorf("mode changed: got %d, want %d", tc.getMode(nm), tc.wantMode)
			}
		})
	}
}

// TestLoad_PreservesExpandedFilterAndScroll checks that a Tasks load preserves
// expansion and the active filter's effect on the visible list, and leaves the
// scroll offset valid.
func TestLoad_PreservesExpandedFilterAndScroll(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
		{Id: 3, Name: "c"},
		{Id: 4, Name: "d"},
		{Id: 5, Name: "e"},
	}
	m := ExportNewModel(&fakeTaskClient{}, cli.BuildTree(tasks))
	m.height = 40
	m.expanded = map[int64]bool{1: true}
	m.filterExpr = "completed=false"
	m.filteredIDs = map[int64]bool{2: true, 4: true}
	m.cursor = 0
	m.listScroll = 0

	newTasks := append([]*taskv1.Task{{Id: 6, Name: "f"}}, tasks...)
	next, _ := m.Update(listTasksResultMsg{tree: cli.BuildTree(newTasks), bg: true})
	nm := next.(Model)
	if !nm.expanded[1] {
		t.Error("expected expanded state preserved")
	}
	if nm.filterExpr != "completed=false" {
		t.Errorf("expected filterExpr preserved, got %q", nm.filterExpr)
	}
	// The filter must still be honored: only the filtered tasks (2, 4) should
	// appear, not the unfiltered set (id 6 must not have reappeared either).
	if len(nm.visible) != 2 {
		t.Fatalf("expected 2 filtered rows, got %d", len(nm.visible))
	}
	for _, row := range nm.visible {
		id := row.node.Task.Id
		if id != 2 && id != 4 {
			t.Errorf("expected only filtered tasks (2, 4) visible, got task %d", id)
		}
	}
}

// TestLoad_ShrinkingListReconcilesScroll checks that a background load which
// shrinks the list keeps listScroll pointing at a valid window, so the Tasks
// pane never renders blank because the offset ran past the end of the list.
func TestLoad_ShrinkingListReconcilesScroll(t *testing.T) {
	var tasks []*taskv1.Task
	for i := int64(1); i <= 50; i++ {
		tasks = append(tasks, &taskv1.Task{Id: i, Name: "t"})
	}
	m := ExportNewModel(&fakeTaskClient{}, cli.BuildTree(tasks))
	m.height = 40 // small viewport so listScroll can be nonzero
	m.cursor = 49
	m.listScroll = 40

	shortTasks := tasks[:5]
	next, _ := m.Update(listTasksResultMsg{tree: cli.BuildTree(shortTasks), bg: true})
	nm := next.(Model)
	if len(nm.visible) != 5 {
		t.Fatalf("expected 5 visible tasks, got %d", len(nm.visible))
	}
	if nm.listScroll < 0 || nm.listScroll >= len(nm.visible) {
		t.Errorf("listScroll %d out of bounds for %d visible rows", nm.listScroll, len(nm.visible))
	}
	if rendered := nm.renderList(80); rendered == "" {
		t.Error("expected renderList to produce non-empty output after a shrinking background load")
	}
}

// TestGoalLoad_ErrorStillEndsLoadingState checks that a failed user-initiated
// goals load still marks the tab as loaded, so the Goals pane renders the
// error instead of being stuck on "Loading..." forever.
func TestGoalLoad_ErrorStillEndsLoadingState(t *testing.T) {
	m := ExportNewGoalModel(&fakeTaskClient{}, nil)
	m.goal.loaded = false // e.g. startup, or after ctrl+r resets it

	next, _ := m.Update(listGoalsResultMsg{err: errors.New("boom"), bg: false})
	nm := next.(Model)
	if !nm.goal.loaded {
		t.Error("expected goal.loaded to be true after a failed load, so the pane shows the error instead of \"Loading...\"")
	}
	if nm.goal.err == nil {
		t.Error("expected goal.err to be set")
	}
}

// TestReportLoad_DiscardsStalePeriod checks that a background report result
// for a period the user has since navigated away from does not overwrite the
// currently displayed period or data (contract C3.7).
func TestReportLoad_DiscardsStalePeriod(t *testing.T) {
	periodA := report.Period{
		From:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		To:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Label: "A",
	}
	periodB := report.Period{
		From:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		To:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Label: "B",
	}
	m := ExportNewReportModel(&fakeTaskClient{})
	ExportSetReportData(&m, reportState{period: periodB, loaded: true})

	// A background result for the old period A arrives after the user moved on to B.
	next, _ := m.Update(reportResultMsg{period: periodA, bg: true})
	nm := next.(Model)
	if nm.reportData.period.Label != "B" {
		t.Errorf("expected period to remain B, got %q", nm.reportData.period.Label)
	}
}

func reportPeriodForTest() report.Period {
	return report.Period{
		From: time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC),
	}
}
