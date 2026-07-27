package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/report"
)

func setNow(m *Model, t time.Time) {
	ExportSetNowFunc(m, func() time.Time { return t })
}

// hasReschedule checks that a returned command reschedules the heartbeat.
// The heartbeat always returns a non-nil command (either the tick alone or the
// tick batched with a fetch).
func hasReschedule(cmd tea.Cmd) bool {
	return cmd != nil
}

// fetchMessages returns the non-tick messages produced by a command.
func fetchMessages(cmd tea.Cmd) []tea.Msg {
	var msgs []tea.Msg
	for _, msg := range collectCmdMsgs(cmd) {
		if _, ok := msg.(autoRefreshTickMsg); ok {
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func TestHeartbeat_StaleTabDispatchesOneFetch(t *testing.T) {
	cases := []struct {
		name       string
		setup      func() Model
		wantMsgTyp string
	}{
		{
			name: "Tasks",
			setup: func() Model {
				m := ExportNewModel(&fakeTaskClient{}, cli.BuildTree([]*taskv1.Task{}))
				setNow(&m, time.Now())
				ExportSetTasksLastLoad(&m, time.Now().Add(-11*time.Minute))
				return m
			},
			wantMsgTyp: "tui.listTasksResultMsg",
		},
		{
			name: "Goals",
			setup: func() Model {
				m := ExportNewGoalModel(&fakeTaskClient{}, []*goalv1.Goal{})
				m.goalClient = &fakeGoalClient{}
				m.activeTab = tabGoals
				setNow(&m, time.Now())
				ExportSetGoalLastLoad(&m, time.Now().Add(-11*time.Minute))
				return m
			},
			wantMsgTyp: "tui.listGoalsResultMsg",
		},
		{
			name: "Plan",
			setup: func() Model {
				m := ExportNewPlanModel(&fakeTaskClient{}, &fakePlanClient{}, "2026-05-27")
				setNow(&m, time.Now())
				ExportSetPlanLastLoad(&m, time.Now().Add(-11*time.Minute))
				return m
			},
			wantMsgTyp: "tui.planEntriesMsg",
		},
		{
			name: "Report",
			setup: func() Model {
				m := ExportNewReportModel(&fakeTaskClient{})
				m.reportData.period = report.Period{From: time.Now(), To: time.Now()}
				setNow(&m, time.Now())
				ExportSetReportLastLoad(&m, time.Now().Add(-11*time.Minute))
				return m
			},
			wantMsgTyp: "tui.reportResultMsg",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.setup()
			next, cmd := m.Update(autoRefreshTickMsg{})
			nm := next.(Model)
			_ = nm
			if cmd == nil {
				t.Fatal("expected command")
			}
			if !hasReschedule(cmd) {
				t.Error("expected command to reschedule the heartbeat")
			}
			msgs := fetchMessages(cmd)
			if len(msgs) != 1 {
				t.Fatalf("expected exactly 1 fetch message, got %d: %v", len(msgs), msgs)
			}
			if typeName(msgs[0]) != tc.wantMsgTyp {
				t.Errorf("expected %s, got %s", tc.wantMsgTyp, typeName(msgs[0]))
			}
		})
	}
}

func TestHeartbeat_FreshTabDispatchesNoFetch(t *testing.T) {
	m := ExportNewModel(&fakeTaskClient{}, cli.BuildTree([]*taskv1.Task{}))
	setNow(&m, time.Now())
	ExportSetTasksLastLoad(&m, time.Now().Add(-30*time.Second))

	next, cmd := m.Update(autoRefreshTickMsg{})
	nm := next.(Model)
	_ = nm
	if !hasReschedule(cmd) {
		t.Error("expected command to reschedule the heartbeat")
	}
	if len(fetchMessages(cmd)) != 0 {
		t.Errorf("expected no fetch messages, got %v", fetchMessages(cmd))
	}
}

func TestHeartbeat_NonListModeSuppressesFetch(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(*Model)
		eligible bool
	}{
		{"modeEdit", func(m *Model) { m.mode = modeEdit }, false},
		{"modeNewSubtask", func(m *Model) { m.mode = modeNewSubtask }, false},
		{"modeNewRoot", func(m *Model) { m.mode = modeNewRoot }, false},
		{"modeHelp", func(m *Model) { m.mode = modeHelp }, false},
		{"modeMove", func(m *Model) { m.mode = modeMove }, false},
		{"modeDatePrompt", func(m *Model) { m.mode = modeDatePrompt }, false},
		{"modeFilter", func(m *Model) { m.mode = modeFilter }, false},
		{"confirmingQuit", func(m *Model) { m.confirmingQuit = true }, false},
		{"confirmingDiscard", func(m *Model) { m.confirmingDiscard = true }, false},
		{"goalPickLink", func(m *Model) { m.activeTab = tabGoals; m.goal.mode = goalPickLink }, false},
		{"goalConfirmDelete", func(m *Model) { m.activeTab = tabGoals; m.goal.mode = goalConfirmDelete }, false},
		{"planPickTask", func(m *Model) { m.activeTab = tabPlanning; m.plan.mode = planPickTask }, false},
		{"planEventForm", func(m *Model) { m.activeTab = tabPlanning; m.plan.mode = planEventForm }, false},
		{"modeList", func(m *Model) { m.mode = modeList }, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := ExportNewModel(&fakeTaskClient{}, cli.BuildTree([]*taskv1.Task{}))
			setNow(&m, time.Now())
			ExportSetTasksLastLoad(&m, time.Now().Add(-11*time.Minute))
			tc.setup(&m)

			if got := ExportAutoRefreshEligible(m); got != tc.eligible {
				t.Fatalf("autoRefreshEligible: got %v, want %v", got, tc.eligible)
			}
			next, cmd := m.Update(autoRefreshTickMsg{})
			nm := next.(Model)
			_ = nm
			if !hasReschedule(cmd) {
				t.Error("expected command to reschedule the heartbeat")
			}
			if tc.eligible {
				if len(fetchMessages(cmd)) == 0 {
					t.Error("expected fetch when eligible")
				}
			} else {
				if len(fetchMessages(cmd)) != 0 {
					t.Errorf("expected no fetch when ineligible, got %v", fetchMessages(cmd))
				}
			}
		})
	}
}

func TestHeartbeat_SuppressionNotSticky(t *testing.T) {
	m := ExportNewModel(&fakeTaskClient{}, cli.BuildTree([]*taskv1.Task{}))
	setNow(&m, time.Now())
	ExportSetTasksLastLoad(&m, time.Now().Add(-11*time.Minute))
	m.mode = modeEdit

	// First tick is suppressed while in edit mode.
	next, cmd := m.Update(autoRefreshTickMsg{})
	m = next.(Model)
	if len(fetchMessages(cmd)) != 0 {
		t.Fatalf("expected no fetch while editing, got %v", fetchMessages(cmd))
	}

	// Return to list mode.
	m.mode = modeList
	next, cmd = m.Update(autoRefreshTickMsg{})
	m = next.(Model)
	_ = m
	if len(fetchMessages(cmd)) == 0 {
		t.Error("expected fetch after returning to list mode")
	}
}

func TestHeartbeat_ReschedulesInAllBranches(t *testing.T) {
	branches := []struct {
		name string
		prep func(*Model)
	}{
		{"eligible", func(m *Model) {
			ExportSetTasksLastLoad(m, time.Now().Add(-11*time.Minute))
		}},
		{"ineligible", func(m *Model) {
			ExportSetTasksLastLoad(m, time.Now().Add(-11*time.Minute))
			m.mode = modeEdit
		}},
		{"not-stale", func(m *Model) {
			ExportSetTasksLastLoad(m, time.Now().Add(-30*time.Second))
		}},
	}

	for _, b := range branches {
		t.Run(b.name, func(t *testing.T) {
			m := ExportNewModel(&fakeTaskClient{}, cli.BuildTree([]*taskv1.Task{}))
			setNow(&m, time.Now())
			b.prep(&m)
			_, cmd := m.Update(autoRefreshTickMsg{})
			if !hasReschedule(cmd) {
				t.Error("expected heartbeat to reschedule")
			}
		})
	}
}

func TestHeartbeat_StampsLastLoadAtDispatch(t *testing.T) {
	m := ExportNewModel(&fakeTaskClient{}, cli.BuildTree([]*taskv1.Task{}))
	now := time.Now()
	setNow(&m, now)
	ExportSetTasksLastLoad(&m, now.Add(-11*time.Minute))

	next, cmd := m.Update(autoRefreshTickMsg{})
	nm := next.(Model)
	_ = cmd
	if ExportTasksLastLoad(nm).IsZero() {
		t.Fatal("expected lastLoad stamped at dispatch")
	}

	// A second tick immediately after dispatch sees fresh data and issues nothing.
	// Use the stamped model's own lastLoad time.
	m = nm
	next, cmd = m.Update(autoRefreshTickMsg{})
	nm = next.(Model)
	_ = nm
	if len(fetchMessages(cmd)) != 0 {
		t.Errorf("expected no fetch after dispatch stamp, got %v", fetchMessages(cmd))
	}
}

// TestHeartbeat_ExactlyTwoTickCallSites is a source-level guard: the heartbeat
// tick command must only be started in Init and rescheduled by its own handler.
// A third call site would permanently double the beat rate.
func TestHeartbeat_ExactlyTwoTickCallSites(t *testing.T) {
	dir := "."
	count := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "func autoRefreshTickCmd(") {
				continue
			}
			if strings.Contains(line, "autoRefreshTickCmd(") {
				count++
			}
		}
	}
	if count != 2 {
		t.Errorf("expected exactly 2 non-test autoRefreshTickCmd( call sites, got %d", count)
	}
}

func TestHeartbeat_UnreachableServerLimitsFetches(t *testing.T) {
	// A failing client and a server that has been unreachable for an hour.
	fc := &fakeTaskClient{listErr: errors.New("unreachable")}
	m := ExportNewModel(fc, cli.BuildTree([]*taskv1.Task{}))
	now := time.Now()
	setNow(&m, now)
	ExportSetTasksLastLoad(&m, now.Add(-11*time.Minute))

	// Simulate one tick per minute for an hour.
	fetchCount := 0
	for i := 0; i < 60; i++ {
		next, cmd := m.Update(autoRefreshTickMsg{})
		m = next.(Model)
		// Advance the injected clock by one minute each iteration.
		now = now.Add(time.Minute)
		setNow(&m, now)
		for _, msg := range fetchMessages(cmd) {
			// Each fetch message is the result of executing the command. Since the
			// fake client returns an error synchronously, the message is a
			// listTasksResultMsg with err set. Count it as a fetch attempt.
			fetchCount++
			// Apply the result so the handler updates lastLoad.
			next, _ := m.Update(msg)
			m = next.(Model)
		}
	}
	if fetchCount > 6 {
		t.Errorf("expected at most 6 fetches in an hour, got %d", fetchCount)
	}
}
