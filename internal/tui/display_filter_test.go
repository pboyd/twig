package tui

import (
	"testing"

	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
)

// TestDisplayedPlanEntries_ExcludesCompletedUntimed verifies that completed untimed entries
// are filtered out when pendingComplete is nil.
func TestDisplayedPlanEntries_ExcludesCompletedUntimed(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", Completed: true},        // untimed, completed → drop
		{Id: 2, Name: "Pending task"},                       // untimed, incomplete → keep
		{Id: 3, Name: "Timed done", StartMinute: pint32(540), Completed: true}, // timed, completed → keep
		{Id: 4, Name: "Event", StartMinute: pint32(600)},   // timed event → keep
	}
	result := displayedPlanEntries(entries, nil)
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(result), result)
	}
	for _, e := range result {
		if e.Id == 1 {
			t.Error("completed untimed entry (id=1) should be excluded")
		}
	}
}

// TestDisplayedPlanEntries_KeepsPendingComplete verifies that the entry whose id
// matches *pendingComplete is retained even though it is completed and untimed.
func TestDisplayedPlanEntries_KeepsPendingComplete(t *testing.T) {
	id := int32(1)
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Just completed", Completed: true}, // untimed, completed, but pending
		{Id: 2, Name: "Other done", Completed: true},     // untimed, completed, not pending → drop
	}
	result := displayedPlanEntries(entries, &id)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry (pending), got %d", len(result))
	}
	if result[0].Id != 1 {
		t.Errorf("expected retained entry id=1, got %d", result[0].Id)
	}
}

// TestDisplayedPlanEntries_KeepsTimedCompleted verifies that timed completed entries
// are never dropped.
func TestDisplayedPlanEntries_KeepsTimedCompleted(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Timed done", StartMinute: pint32(540), Completed: true},
	}
	result := displayedPlanEntries(entries, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 timed completed entry to survive, got %d", len(result))
	}
}

// TestDisplayedPlanEntries_KeepsIncompleteUntimed verifies that incomplete untimed entries
// are always kept.
func TestDisplayedPlanEntries_KeepsIncompleteUntimed(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Todo"},
	}
	result := displayedPlanEntries(entries, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 incomplete untimed entry, got %d", len(result))
	}
}

// TestDisplayedPlanEntries_OrderPreserving verifies the output order matches the input order.
func TestDisplayedPlanEntries_OrderPreserving(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 10, Name: "A"},
		{Id: 20, Name: "B", StartMinute: pint32(540)},
		{Id: 30, Name: "C"},
	}
	result := displayedPlanEntries(entries, nil)
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	if result[0].Id != 10 || result[1].Id != 20 || result[2].Id != 30 {
		t.Errorf("order not preserved: got %d %d %d", result[0].Id, result[1].Id, result[2].Id)
	}
}

// TestDisplayedPlanEntries_Idempotent verifies calling twice gives the same result.
func TestDisplayedPlanEntries_Idempotent(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done", Completed: true},
		{Id: 2, Name: "Todo"},
	}
	first := displayedPlanEntries(entries, nil)
	second := displayedPlanEntries(first, nil)
	if len(first) != len(second) {
		t.Errorf("not idempotent: first=%d, second=%d", len(first), len(second))
	}
}

// ── T006: handlePlanEntriesMsg applies displayedPlanEntries filter ─────────────

// TestHandlePlanEntriesMsg_FiltersCompletedUntimed verifies that handlePlanEntriesMsg
// stores a displayed list that omits completed untimed entries when pendingComplete==nil.
func TestHandlePlanEntriesMsg_FiltersCompletedUntimed(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})
	m.plan.pendingComplete = nil

	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", Completed: true},       // untimed completed → drop
		{Id: 2, Name: "Todo task"},                         // untimed incomplete → keep
		{Id: 3, Name: "Timed done", StartMinute: pint32(540), Completed: true}, // timed → keep
	}
	m = m.handlePlanEntriesMsg(planEntriesMsg{entries: entries}, 0)

	if len(m.plan.entries) != 2 {
		t.Fatalf("expected 2 displayed entries, got %d", len(m.plan.entries))
	}
	for _, e := range m.plan.entries {
		if e.Id == 1 {
			t.Error("completed untimed entry (id=1) should be filtered from displayed list")
		}
	}
}

// TestHandlePlanEntriesMsg_KeepsTimedAndEvents verifies that timed entries (including
// completed ones) and event entries are never filtered.
func TestHandlePlanEntriesMsg_KeepsTimedAndEvents(t *testing.T) {
	m := buildPlanTestModel(&fakePlanClient{})

	entries := []*planv1.PlanEntry{
		{Id: 1, StartMinute: pint32(540), Completed: true},  // timed completed → keep
		{Id: 2, StartMinute: pint32(600)},                   // timed event → keep
	}
	m = m.handlePlanEntriesMsg(planEntriesMsg{entries: entries}, 0)

	if len(m.plan.entries) != 2 {
		t.Fatalf("expected both timed entries to survive, got %d", len(m.plan.entries))
	}
}
