package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
)

func buildCompleteTestModel(fc *fakePlanClient, entries []*planv1.PlanEntry, cursor int) Model {
	m := buildPlanTestModel(fc)
	m.plan.entries = entries
	m.plan.loaded = true
	m.plan.cursor = cursor
	return m
}

// TestComplete_UntimedEntry_SetsPendingComplete verifies that completing a highlighted
// untimed entry sets m.plan.pendingComplete to its id.
func TestComplete_UntimedEntry_SetsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	entry := &planv1.PlanEntry{Id: 5, Name: "Write spec", TaskId: 10, Completed: false}
	m := buildCompleteTestModel(fc, []*planv1.PlanEntry{entry}, 0)

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	if m2.plan.pendingComplete == nil {
		t.Fatal("completing untimed entry: expected pendingComplete to be set")
	}
	if *m2.plan.pendingComplete != 5 {
		t.Errorf("pendingComplete: expected 5, got %d", *m2.plan.pendingComplete)
	}
}

// TestComplete_TimedEntry_DoesNotSetPendingComplete verifies that completing a timed
// entry never sets pendingComplete.
func TestComplete_TimedEntry_DoesNotSetPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	entry := &planv1.PlanEntry{Id: 7, Name: "Standup", TaskId: 20, StartMinute: pint32(540), Completed: false}
	m := buildCompleteTestModel(fc, []*planv1.PlanEntry{entry}, 0)

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	if m2.plan.pendingComplete != nil {
		t.Errorf("completing timed entry: pendingComplete should stay nil, got %d", *m2.plan.pendingComplete)
	}
}

// TestComplete_ReloadRetainsPendingEntry verifies that after completing an untimed entry,
// a subsequent handlePlanEntriesMsg with the entry still present keeps it (pendingComplete set).
func TestComplete_ReloadRetainsPendingEntry(t *testing.T) {
	fc := &fakePlanClient{}
	entry := &planv1.PlanEntry{Id: 5, Name: "Write spec", TaskId: 10, Completed: false}
	m := buildCompleteTestModel(fc, []*planv1.PlanEntry{entry}, 0)

	// Complete the entry: sets pendingComplete=5
	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	// Simulate reload: server returns entry as completed
	completedEntry := &planv1.PlanEntry{Id: 5, Name: "Write spec", TaskId: 10, Completed: true}
	m3 := m2.handlePlanEntriesMsg(planEntriesMsg{entries: []*planv1.PlanEntry{completedEntry}}, 0)

	// Entry should still be present (pendingComplete retains it)
	found := false
	for _, e := range m3.plan.entries {
		if e.Id == 5 {
			found = true
		}
	}
	if !found {
		t.Error("pending complete entry should remain in displayed list after reload")
	}
}

// TestCursorUp_ClearsPendingComplete verifies that pressing Up clears pendingComplete
// and the entry is dropped from the displayed plan.
func TestCursorUp_ClearsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(2)
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Task A"},
		{Id: 2, Name: "Done task", Completed: true, TaskId: 5}, // pending complete
	}
	m := buildCompleteTestModel(fc, entries, 1) // cursor on entry 2
	m.plan.pendingComplete = &pending

	m2, _ := pressKeyStr(m, "k")

	if m2.plan.pendingComplete != nil {
		t.Errorf("cursor Up: pendingComplete should be cleared, got %d", *m2.plan.pendingComplete)
	}
	// Entry 2 (completed untimed) should now be absent
	for _, e := range m2.plan.entries {
		if e.Id == 2 {
			t.Error("cursor Up: completed untimed entry should be removed after clearing pendingComplete")
		}
	}
}

// TestCursorDown_ClearsPendingComplete verifies that pressing Down clears pendingComplete.
func TestCursorDown_ClearsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(1)
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", Completed: true, TaskId: 5},
		{Id: 2, Name: "Task B"},
	}
	m := buildCompleteTestModel(fc, entries, 0)
	m.plan.pendingComplete = &pending

	m2, _ := pressKeyStr(m, "j")

	if m2.plan.pendingComplete != nil {
		t.Errorf("cursor Down: pendingComplete should be cleared, got %d", *m2.plan.pendingComplete)
	}
	for _, e := range m2.plan.entries {
		if e.Id == 1 {
			t.Error("cursor Down: completed untimed entry should be removed after clearing pendingComplete")
		}
	}
}

// TestCursorDown_AfterComplete_LandsOnNextEntry verifies that pressing Down after
// completing the highlighted untimed entry lands on the entry directly below it,
// not the one after that (regression: cursor was incremented before the completed
// entry was dropped, shifting the target index).
func TestCursorDown_AfterComplete_LandsOnNextEntry(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(1)
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "A", Completed: true, TaskId: 5}, // pending complete, lingering
		{Id: 2, Name: "B", TaskId: 6},
		{Id: 3, Name: "C", TaskId: 7},
	}
	m := buildCompleteTestModel(fc, entries, 0)
	m.plan.pendingComplete = &pending

	m2, _ := pressKeyStr(m, "j")

	if len(m2.plan.entries) != 2 || m2.plan.entries[0].Id != 2 || m2.plan.entries[1].Id != 3 {
		t.Fatalf("expected entries [B, C] after drop, got %+v", m2.plan.entries)
	}
	if m2.plan.cursor != 0 || m2.plan.entries[m2.plan.cursor].Id != 2 {
		t.Errorf("cursor Down after complete: expected to land on entry B (id 2), got cursor=%d entries=%+v", m2.plan.cursor, m2.plan.entries)
	}
}

// TestCursorDown_AfterCompletingLastEntry verifies Down from the last (pending-complete)
// entry clamps onto the new last entry after it is dropped.
func TestCursorDown_AfterCompletingLastEntry(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(2)
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "A", TaskId: 5},
		{Id: 2, Name: "B", Completed: true, TaskId: 6}, // pending complete, lingering, at cursor
	}
	m := buildCompleteTestModel(fc, entries, 1)
	m.plan.pendingComplete = &pending

	m2, _ := pressKeyStr(m, "j")

	if len(m2.plan.entries) != 1 || m2.plan.entries[0].Id != 1 {
		t.Fatalf("expected entries [A] after drop, got %+v", m2.plan.entries)
	}
	if m2.plan.cursor != 0 {
		t.Errorf("cursor Down after completing last entry: expected cursor 0, got %d", m2.plan.cursor)
	}
}

// TestCursorUp_AfterComplete_LandsOnPrevEntry verifies Up after completing the
// highlighted untimed entry lands on the entry directly above it.
func TestCursorUp_AfterComplete_LandsOnPrevEntry(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(3)
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "A", TaskId: 5},
		{Id: 2, Name: "B", TaskId: 6},
		{Id: 3, Name: "C", Completed: true, TaskId: 7}, // pending complete, lingering, at cursor
	}
	m := buildCompleteTestModel(fc, entries, 2)
	m.plan.pendingComplete = &pending

	m2, _ := pressKeyStr(m, "k")

	if len(m2.plan.entries) != 2 || m2.plan.entries[0].Id != 1 || m2.plan.entries[1].Id != 2 {
		t.Fatalf("expected entries [A, B] after drop, got %+v", m2.plan.entries)
	}
	if m2.plan.cursor != 1 || m2.plan.entries[m2.plan.cursor].Id != 2 {
		t.Errorf("cursor Up after complete: expected to land on entry B (id 2), got cursor=%d entries=%+v", m2.plan.cursor, m2.plan.entries)
	}
}

// TestPlanPrevDay_ClearsPendingComplete verifies that day navigation clears pendingComplete.
func TestPlanPrevDay_ClearsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(1)
	m := buildPlanTestModel(fc)
	m.plan.day = "2026-06-09"
	m.plan.loaded = true
	m.plan.pendingComplete = &pending

	m2, _ := pressKeyStr(m, "[")

	if m2.plan.pendingComplete != nil {
		t.Errorf("PlanPrevDay: pendingComplete should be cleared, got %d", *m2.plan.pendingComplete)
	}
}

// TestPlanNextDay_ClearsPendingComplete verifies next-day navigation clears pendingComplete.
func TestPlanNextDay_ClearsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(1)
	m := buildPlanTestModel(fc)
	m.plan.day = "2026-06-09"
	m.plan.loaded = true
	m.plan.pendingComplete = &pending

	m2, _ := pressKeyStr(m, "]")

	if m2.plan.pendingComplete != nil {
		t.Errorf("PlanNextDay: pendingComplete should be cleared, got %d", *m2.plan.pendingComplete)
	}
}

// TestPlanToday_ClearsPendingComplete verifies today-navigation clears pendingComplete.
func TestPlanToday_ClearsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(1)
	m := buildPlanTestModel(fc)
	m.plan.day = "2026-01-01"
	m.plan.loaded = true
	m.plan.pendingComplete = &pending

	m2, _ := pressKeyStr(m, ".")

	if m2.plan.pendingComplete != nil {
		t.Errorf("PlanToday: pendingComplete should be cleared, got %d", *m2.plan.pendingComplete)
	}
}

// TestTabSwitch_ClearsPendingComplete verifies tab switch clears pendingComplete.
func TestTabSwitch_ClearsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(1)
	m := buildPlanTestModel(fc)
	m.plan.pendingComplete = &pending

	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeyTab})

	if m2.plan.pendingComplete != nil {
		t.Errorf("tab switch: pendingComplete should be cleared, got %d", *m2.plan.pendingComplete)
	}
}

// ── T012: US3 — uncomplete clears pendingComplete ─────────────────────────────

// TestUncomplete_ClearsPendingComplete verifies that uncompleting an entry clears
// pendingComplete so the (now-incomplete) entry renders normally and is not struck.
func TestUncomplete_ClearsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(5)
	entry := &planv1.PlanEntry{Id: 5, Name: "Done task", TaskId: 10, Completed: true}
	m := buildCompleteTestModel(fc, []*planv1.PlanEntry{entry}, 0)
	m.plan.pendingComplete = &pending

	// Toggle: entry is Completed=true so this uncompletes it
	m2, _ := pressSpecialKey(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	if m2.plan.pendingComplete != nil {
		t.Errorf("uncomplete: pendingComplete should be nil, got %d", *m2.plan.pendingComplete)
	}
}

// TestUncomplete_EntryAppearsAfterReload verifies that after uncompleting, a reload
// with the entry as Completed=false shows it in the plan (no longer filtered out).
func TestUncomplete_EntryAppearsAfterReload(t *testing.T) {
	fc := &fakePlanClient{}
	entry := &planv1.PlanEntry{Id: 5, Name: "Reopened task", TaskId: 10, Completed: true}
	m := buildCompleteTestModel(fc, []*planv1.PlanEntry{entry}, 0)
	m.plan.pendingComplete = nil // entry was hidden (pendingComplete cleared elsewhere)

	// Reload with entry now incomplete
	reopened := &planv1.PlanEntry{Id: 5, Name: "Reopened task", TaskId: 10, Completed: false}
	m2 := m.handlePlanEntriesMsg(planEntriesMsg{entries: []*planv1.PlanEntry{reopened}}, 0)

	found := false
	for _, e := range m2.plan.entries {
		if e.Id == 5 {
			found = true
		}
	}
	if !found {
		t.Error("reopened entry (Completed=false) should appear in displayed plan")
	}
}

// TestPlanGoToTask_ClearsPendingComplete verifies go-to-task clears pendingComplete.
func TestPlanGoToTask_ClearsPendingComplete(t *testing.T) {
	fc := &fakePlanClient{}
	pending := int32(1)
	m := buildPlanTestModel(fc)
	m.plan.loaded = true
	m.plan.pendingComplete = &pending
	// Set a task-linked entry at cursor
	m.plan.entries = []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", Completed: true, TaskId: 5},
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m2 := next.(Model)

	if m2.plan.pendingComplete != nil {
		t.Errorf("go-to-task: pendingComplete should be cleared, got %d", *m2.plan.pendingComplete)
	}
}
