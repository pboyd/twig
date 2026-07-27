package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"connectrpc.com/connect"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	goalv1connect "github.com/pboyd/twig/api/gen/goal/v1/goalv1connect"
)

// collectCmdMsgs recursively executes a tea.Cmd and returns the concrete messages
// it produces, flattening any tea.BatchMsg it encounters. Tick commands (which
// may block for a duration in tests) are skipped after a short timeout.
func collectCmdMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := executeWithTimeout(cmd, 10*time.Millisecond)
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			msgs = append(msgs, collectCmdMsgs(c)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// executeWithTimeout runs a tea.Cmd with a timeout; nil means the command timed
// out (typical for tea.Tick commands in tests).
func executeWithTimeout(cmd tea.Cmd, timeout time.Duration) tea.Msg {
	if cmd == nil {
		return nil
	}
	type result struct {
		msg tea.Msg
	}
	ch := make(chan result, 1)
	go func() { ch <- result{cmd()} }()
	select {
	case r := <-ch:
		return r.msg
	case <-time.After(timeout):
		return nil
	}
}

func typeName(msg tea.Msg) string {
	return fmt.Sprintf("%T", msg)
}

func TestTabSwitch_RefreshDispatchedOnEveryTabEntry(t *testing.T) {
	goalClient := &fakeGoalClient{}
	cases := []struct {
		name       string
		startTab   tab
		key        tea.KeyPressMsg
		wantTab    tab
		wantMsgTyp []string
	}{
		{
			name:       "Tasks → Planning",
			startTab:   tabTasks,
			key:        tea.KeyPressMsg{Code: tea.KeyTab},
			wantTab:    tabPlanning,
			wantMsgTyp: []string{"tui.planEntriesMsg"},
		},
		{
			name:       "Planning → Report",
			startTab:   tabPlanning,
			key:        tea.KeyPressMsg{Code: tea.KeyTab},
			wantTab:    tabReport,
			wantMsgTyp: []string{"tui.reportResultMsg"},
		},
		{
			name:       "Report → Goals",
			startTab:   tabReport,
			key:        tea.KeyPressMsg{Code: tea.KeyTab},
			wantTab:    tabGoals,
			wantMsgTyp: []string{"tui.listGoalsResultMsg"},
		},
		{
			name:       "Goals → Tasks",
			startTab:   tabGoals,
			key:        tea.KeyPressMsg{Code: tea.KeyTab},
			wantTab:    tabTasks,
			wantMsgTyp: []string{"tui.listTasksResultMsg", "tui.scheduledDaysResultMsg"},
		},
		{
			name:       "Planning → Tasks",
			startTab:   tabPlanning,
			key:        tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
			wantTab:    tabTasks,
			wantMsgTyp: []string{"tui.listTasksResultMsg", "tui.scheduledDaysResultMsg"},
		},
		{
			name:       "Report → Planning",
			startTab:   tabReport,
			key:        tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
			wantTab:    tabPlanning,
			wantMsgTyp: []string{"tui.planEntriesMsg"},
		},
		{
			name:       "Goals → Report",
			startTab:   tabGoals,
			key:        tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
			wantTab:    tabReport,
			wantMsgTyp: []string{"tui.reportResultMsg"},
		},
		{
			name:       "Tasks → Goals",
			startTab:   tabTasks,
			key:        tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
			wantTab:    tabGoals,
			wantMsgTyp: []string{"tui.listGoalsResultMsg"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakePlanClient{}
			taskClient := &fakeTaskClient{}
			m := buildPlanTestModel(fc)
			m.client = taskClient
			m.goalClient = goalClient
			m.activeTab = tc.startTab
			m.plan.day = time.Now().Format("2006-01-02")
			m.keys.GoalMode = tc.startTab == tabGoals
			m.keys.PlanningMode = tc.startTab == tabPlanning
			m.keys.ReportMode = tc.startTab == tabReport

			next, cmd := m.Update(tc.key)
			nm := next.(Model)
			if nm.activeTab != tc.wantTab {
				t.Errorf("active tab: got %d, want %d", nm.activeTab, tc.wantTab)
			}
			if cmd == nil {
				t.Fatalf("expected non-nil command on tab entry")
			}
			msgs := fetchCmdMsgs(cmd)
			if len(msgs) != len(tc.wantMsgTyp) {
				t.Fatalf("expected %d fetch messages, got %d: %v", len(tc.wantMsgTyp), len(msgs), msgs)
			}
			for i, want := range tc.wantMsgTyp {
				got := typeName(msgs[i])
				if got != want {
					t.Errorf("message %d: got %s, want %s", i, got, want)
				}
			}
		})
	}
}

// fetchCmdMsgs returns only the non-timer fetch messages produced by a command.
func fetchCmdMsgs(cmd tea.Cmd) []tea.Msg {
	var msgs []tea.Msg
	for _, msg := range collectCmdMsgs(cmd) {
		if msg == nil {
			continue
		}
		switch msg.(type) {
		case planTickMsg, autoRefreshTickMsg:
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func TestTabSwitch_GoalsReloadsWhenAlreadyLoaded(t *testing.T) {
	goalClient := &fakeGoalClient{}
	m := buildPlanTestModel(&fakePlanClient{})
	m.client = &fakeTaskClient{}
	m.goalClient = goalClient
	m.activeTab = tabReport
	m.goal.loaded = true
	m.keys.ReportMode = true

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd == nil {
		t.Fatal("expected reload command when entering already-loaded Goals tab")
	}
	msgs := collectCmdMsgs(cmd)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if _, ok := msgs[0].(listGoalsResultMsg); !ok {
		t.Fatalf("expected listGoalsResultMsg, got %T", msgs[0])
	}
}

// fakeGoalClient is a minimal GoalServiceClient for unit tests.
type fakeGoalClient struct {
	goalv1connect.GoalServiceClient
	listCalls int
	listResp  []*goalv1.Goal
	listErr   error
}

func (f *fakeGoalClient) ListGoals(ctx context.Context, _ *connect.Request[goalv1.ListGoalsRequest]) (*connect.Response[goalv1.ListGoalsResponse], error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	return connect.NewResponse(&goalv1.ListGoalsResponse{Goals: f.listResp}), nil
}
