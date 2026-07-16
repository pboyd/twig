package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
)

// buildFilterModel builds a model with 5 tasks (ids 1–5) and seeds an active
// filter matching a subset. Returns the model and the full tree.
func buildFilterModel(filteredIDs []int64) Model {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "A"},
		{Id: 2, Name: "B"},
		{Id: 3, Name: "C"},
		{Id: 4, Name: "D"},
		{Id: 5, Name: "E"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)
	ExportSetFilterState(&m, "^goal_id=1", filteredIDs)
	return m
}

func TestFilterPersist_NavPreservesFilter(t *testing.T) {
	m := buildFilterModel([]int64{2, 3})
	filteredCount := len(m.visible)
	if filteredCount != 2 {
		t.Fatalf("setup: want 2 filtered rows, got %d", filteredCount)
	}

	m = pressKey(m, "down")
	if ExportFilterExpr(m) != "^goal_id=1" {
		t.Errorf("after down: filterExpr cleared, want ^goal_id=1")
	}
	if len(m.visible) != filteredCount {
		t.Errorf("after down: visible count = %d, want %d (filtered)", len(m.visible), filteredCount)
	}

	m = pressKey(m, "up")
	if ExportFilterExpr(m) != "^goal_id=1" {
		t.Errorf("after up: filterExpr cleared, want ^goal_id=1")
	}
	if len(m.visible) != filteredCount {
		t.Errorf("after up: visible count = %d, want %d (filtered)", len(m.visible), filteredCount)
	}
}

func TestFilterPersist_ReorderPreservesFilter(t *testing.T) {
	m := buildFilterModel([]int64{2, 3})
	filteredCount := len(m.visible)
	if filteredCount != 2 {
		t.Fatalf("setup: want 2 filtered rows, got %d", filteredCount)
	}

	newSiblings := []*taskv1.Task{
		{Id: 3, Name: "C", Position: 0},
		{Id: 2, Name: "B", Position: 1},
	}
	result := ExportHandleReorderResult(m, reorderResultMsg{taskID: 3, siblings: newSiblings})

	if ExportFilterExpr(result) != "^goal_id=1" {
		t.Error("after reorder: filterExpr cleared, want ^goal_id=1")
	}
	if len(result.visible) != filteredCount {
		t.Errorf("after reorder: visible count = %d, want %d (filtered)", len(result.visible), filteredCount)
	}
}

func TestFilterPersist_EscClearsInModeList(t *testing.T) {
	m := buildFilterModel([]int64{2, 3})
	if ExportFilterExpr(m) == "" {
		t.Fatal("setup: filterExpr should be set")
	}

	m = pressKey(m, "esc")

	if ExportFilterExpr(m) != "" {
		t.Errorf("after esc: filterExpr = %q, want empty", ExportFilterExpr(m))
	}
	if len(m.visible) != 5 {
		t.Errorf("after esc: visible count = %d, want 5 (full tree)", len(m.visible))
	}
}

func TestFilterPersist_EscNoOpWithoutFilter(t *testing.T) {
	m := buildTestModel()
	origCursor := m.cursor
	origVisible := len(m.visible)

	result, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	rm := result.(Model)

	if cmd != nil {
		t.Error("esc without filter should return nil cmd")
	}
	if ExportFilterExpr(rm) != "" {
		t.Errorf("esc without filter should not set filterExpr, got %q", ExportFilterExpr(rm))
	}
	if rm.cursor != origCursor {
		t.Errorf("esc without filter moved cursor from %d to %d", origCursor, rm.cursor)
	}
	if len(rm.visible) != origVisible {
		t.Errorf("esc without filter changed visible from %d to %d", origVisible, len(rm.visible))
	}
}

func TestFilterPersist_ExpandCollapseInertUnderFilter(t *testing.T) {
	parent := &taskv1.Task{Id: 1, Name: "parent"}
	child := &taskv1.Task{Id: 2, Name: "child", ParentId: &parent.Id}
	tree := cli.BuildTree([]*taskv1.Task{parent, child})
	m := ExportNewModel(nil, tree)
	ExportSetFilterState(&m, "test", []int64{1})
	origVisible := len(m.visible)
	origExpanded := len(m.expanded)

	m = pressKey(m, "left")
	if len(m.visible) != origVisible {
		t.Errorf("collapse under filter changed visible from %d to %d", origVisible, len(m.visible))
	}
	if len(m.expanded) != origExpanded {
		t.Errorf("collapse under filter changed expanded map")
	}

	m = pressKey(m, "right")
	if len(m.visible) != origVisible {
		t.Errorf("expand under filter changed visible from %d to %d", origVisible, len(m.visible))
	}
}

func TestFilterPersist_IndicatorRendered(t *testing.T) {
	m := buildFilterModel([]int64{2, 3})
	m.activeTab = tabTasks
	m.mode = modeList

	status := m.renderStatus()

	if !strings.Contains(status, "^goal_id=1") {
		t.Errorf("renderStatus should contain filter expr, got: %s", status)
	}
	if !strings.Contains(status, "esc") {
		t.Errorf("renderStatus should contain esc hint, got: %s", status)
	}
}

func TestFilterPersist_PlanJumpClearsFilter(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "A"},
		{Id: 2, Name: "B"},
		{Id: 3, Name: "C"},
	}
	tree := cli.BuildTree(tasks)
	fc := &fakePlanClient{}
	m := buildPlanTestModel(fc)
	m.tree = tree
	m.visible = buildVisible(tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())

	ExportSetFilterState(&m, "^goal_id=1", []int64{1})
	if ExportFilterExpr(m) != "^goal_id=1" {
		t.Fatal("setup: filterExpr should be set")
	}

	m.activeTab = tabPlanning
	m.keys.PlanningMode = true
	m.keys.GoalMode = false
	ExportSetPlanEntries(&m, []*planv1.PlanEntry{
		{Id: 1, TaskId: 2, Name: "B", DurationMinute: 30},
	}, 0)

	msg := tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl}
	result, cmd := m.Update(msg)
	rm := result.(Model)

	if ExportFilterExpr(rm) != "" {
		t.Errorf("after PlanGoToTask: filterExpr = %q, want empty", ExportFilterExpr(rm))
	}
	if int(rm.activeTab) != ExportTabTasks {
		t.Errorf("after PlanGoToTask: activeTab = %d, want tabTasks (%d)", rm.activeTab, ExportTabTasks)
	}
	if len(rm.visible) == 0 {
		t.Fatal("after PlanGoToTask: no visible rows")
	}
	if rm.visible[rm.cursor].node.Task.Id != 2 {
		t.Errorf("after PlanGoToTask: cursor on task %d, want task 2", rm.visible[rm.cursor].node.Task.Id)
	}
	if cmd != nil {
		t.Errorf("after PlanGoToTask: expected nil cmd, got %v", cmd)
	}
}
