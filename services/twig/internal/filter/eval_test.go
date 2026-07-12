package filter

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func ts(year, month, day int) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC),
		Valid: true,
	}
}

func today(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func TestEvalTextMatch(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Buy groceries", Description: "milk, eggs"},
		{ID: 2, Name: "Review code", Description: "PR #42"},
		{ID: 3, Name: "Write report", Description: "quarterly summary"},
		{ID: 4, Name: "Buy gift", Description: "birthday present"},
	}

	expr, err := Parse("buy")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// "buy" matches tasks 1 and 4 (both have "Buy" in name)
	// FR-008: show_all=false, no completed/snoozed conditions → completed=false AND snoozed=false
	// All tasks are incomplete and unsnoozed, so both should match.
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d: %v", len(ids), ids)
	}
	if ids[0] != 1 || ids[1] != 4 {
		t.Errorf("ids = %v, want [1 4]", ids)
	}
}

func TestEvalTextMatchDescription(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Task A", Description: "milk eggs"},
		{ID: 2, Name: "Task B", Description: "bread cheese"},
	}

	expr, err := Parse("milk")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalCaseInsensitive(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Groceries", Description: "milk"},
		{ID: 2, Name: "other", Description: "stuff"},
	}

	expr, err := Parse("groceries")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalShowAllOffDefaults(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Incomplete foo", CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "Completed foo", CompletedAt: ts(2026, 7, 1)},
	}

	expr, err := Parse("foo")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// show_all=false → implicit completed=false → only incomplete task 1
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalShowAllOn(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Incomplete foo", CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "Completed foo", CompletedAt: ts(2026, 7, 1)},
	}

	expr, err := Parse("foo")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// show_all=true → no implicit conditions → both match
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d: %v", len(ids), ids)
	}
}

func TestEvalSnoozedDefault(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Task foo", SnoozeUntil: pgtype.Timestamptz{}},
		{ID: 2, Name: "Task foo", SnoozeUntil: ts(2026, 7, 20)}, // snoozed into future
	}

	expr, err := Parse("foo")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// show_all=false → implicit snoozed=false → snoozed task 2 excluded
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalSnoozedExplicit(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "foo", SnoozeUntil: pgtype.Timestamptz{}},
		{ID: 2, Name: "foo", SnoozeUntil: ts(2026, 7, 20)},
		{ID: 3, Name: "foo", SnoozeUntil: ts(2026, 7, 20), CompletedAt: ts(2026, 7, 1)},
	}

	expr, err := Parse(`snoozed=true`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// snoozed=true → only task 2 matches (task 3 is completed, which is filtered by implicit completed=false)
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("ids = %v, want [2]", ids)
	}
}

func TestEvalBoolCompleted(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Task A", CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "Task B", CompletedAt: ts(2026, 7, 1)},
	}

	expr, err := Parse(`completed=true`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// completed=true → explicit mention of completed → no implicit snoozed filter
	// (show_all=false with completed mentioned → only snoozed=false added)
	// Task 2 is completed and not snoozed
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("ids = %v, want [2]", ids)
	}
}

func TestEvalDateCompleted(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Task A", CompletedAt: ts(2025, 12, 31)},
		{ID: 2, Name: "Task B", CompletedAt: ts(2026, 1, 1)},
		{ID: 3, Name: "Task C", CompletedAt: ts(2026, 6, 15)},
	}

	expr, err := Parse(`completed < 2026-01-01`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// completed < 2026-01-01 → only task 1 (completed 2025-12-31)
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalDateCompletedSameDayGranularity(t *testing.T) {
	// completed_at is 2026-01-01 12:00 UTC (midday, not midnight). Day-granularity
	// comparison means this task should match both <= and >= 2026-01-01, since its
	// UTC calendar date equals the comparison day.
	tasks := []Task{
		{ID: 1, Name: "Task A", CompletedAt: ts(2026, 1, 1)},
	}

	leExpr, err := Parse(`completed <= 2026-01-01`)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := Evaluate(leExpr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("<=: ids = %v, want [1] (same calendar day matches)", ids)
	}

	geExpr, err := Parse(`completed >= 2026-01-01`)
	if err != nil {
		t.Fatal(err)
	}
	ids, err = Evaluate(geExpr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf(">=: ids = %v, want [1] (same calendar day matches)", ids)
	}

	gtExpr, err := Parse(`completed > 2026-01-01`)
	if err != nil {
		t.Fatal(err)
	}
	ids, err = Evaluate(gtExpr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf(">: ids = %v, want [] (same calendar day is not strictly after)", ids)
	}
}

func TestEvalRelParentId(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Parent"},
		{ID: 2, Name: "Child", ParentID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 3, Name: "Other"},
	}

	expr, err := Parse(`parent_id=1`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("ids = %v, want [2]", ids)
	}
}

func TestEvalAscendingOrder(t *testing.T) {
	tasks := []Task{
		{ID: 5, Name: "foo bar"},
		{ID: 3, Name: "bar baz foo"},
		{ID: 1, Name: "foo qux"},
	}

	expr, err := Parse("foo")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}
	if ids[0] != 1 || ids[1] != 3 || ids[2] != 5 {
		t.Errorf("ids = %v, want [1 3 5] (ascending)", ids)
	}
}

func TestEvalNoMatch(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Task A"},
		{ID: 2, Name: "Task B"},
	}

	expr, err := Parse("nonexistent")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 0 {
		t.Errorf("expected empty result, got %v", ids)
	}
}

func TestEvalTransitiveParentId(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Root"},
		{ID: 2, Name: "Child", ParentID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 3, Name: "Grandchild", ParentID: pgtype.Int8{Int64: 2, Valid: true}},
		{ID: 4, Name: "Other"},
	}

	expr, err := Parse(`^parent_id=1`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Task 1 itself does not match; 2 (direct child) and 3 (grandchild) do.
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 3 {
		t.Errorf("ids = %v, want [2 3]", ids)
	}
}

func TestEvalTransitiveParentIdNe(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Root"},
		{ID: 2, Name: "Child", ParentID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 3, Name: "Grandchild", ParentID: pgtype.Int8{Int64: 2, Valid: true}},
		{ID: 4, Name: "Other"},
	}

	expr, err := Parse(`^parent_id!=1`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Complement of {2, 3}: {1, 4}.
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 4 {
		t.Errorf("ids = %v, want [1 4]", ids)
	}
}

func TestEvalTransitiveGoalId(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "Goal root", GoalID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 2, Name: "Child", ParentID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 3, Name: "Grandchild", ParentID: pgtype.Int8{Int64: 2, Valid: true}},
		{ID: 4, Name: "Unrelated"},
	}

	expr, err := Parse(`^goal_id=1`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// The association root and both descendants match; task 4 does not.
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("ids = %v, want [1 2 3]", ids)
	}
}

func TestEvalAndExpression(t *testing.T) {
	tasks := []Task{
		{ID: 1, Name: "foo task", CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "foo done", CompletedAt: ts(2026, 7, 1)},
		{ID: 3, Name: "bar task", CompletedAt: pgtype.Timestamptz{}},
	}

	expr, err := Parse(`foo AND completed=false`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// foo matches 1 and 2; completed=false matches 1 and 3
	// intersection: task 1 only
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalWorkedExampleCompletedAndParentId(t *testing.T) {
	// completed=true AND parent_id=1 (show_all=off)
	// → completed, unsnoozed direct children of task 1
	tasks := []Task{
		{ID: 1, Name: "Parent"},
		{ID: 2, Name: "Child done", CompletedAt: ts(2026, 7, 1), ParentID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 3, Name: "Child incomplete", CompletedAt: pgtype.Timestamptz{}, ParentID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 4, Name: "Grandchild done", CompletedAt: ts(2026, 7, 1), ParentID: pgtype.Int8{Int64: 2, Valid: true}},
		{ID: 5, Name: "Unrelated done", CompletedAt: ts(2026, 7, 1)},
	}

	expr, err := Parse(`completed=true AND parent_id=1`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Only task 2: completed AND direct child of 1
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("ids = %v, want [2]", ids)
	}
}

func TestEvalWorkedExampleCompletedAndTransitiveParent(t *testing.T) {
	// completed=false AND ^parent_id=1 (show_all=off)
	// → incomplete, unsnoozed descendants of task 1 (any depth)
	tasks := []Task{
		{ID: 1, Name: "Root", CompletedAt: ts(2026, 7, 1)},
		{ID: 2, Name: "Child inc", CompletedAt: pgtype.Timestamptz{}, ParentID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 3, Name: "Grandchild inc", CompletedAt: pgtype.Timestamptz{}, ParentID: pgtype.Int8{Int64: 2, Valid: true}},
		{ID: 4, Name: "Child done", CompletedAt: ts(2026, 7, 1), ParentID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 5, Name: "Other"},
	}

	expr, err := Parse(`completed=false AND ^parent_id=1`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Tasks 2 and 3: incomplete descendants of 1; task 4 is completed; task 1 is the root itself
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 3 {
		t.Errorf("ids = %v, want [2 3]", ids)
	}
}

func TestEvalWorkedExampleCompletedAndSnoozedAndGoalTransitive(t *testing.T) {
	// completed=false AND snoozed=false AND ^goal_id=1 (show_all=off)
	// → the whole incomplete, unsnoozed tree under goal 1
	tasks := []Task{
		{ID: 1, Name: "Goal root", GoalID: pgtype.Int8{Int64: 1, Valid: true}, CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "Child", ParentID: pgtype.Int8{Int64: 1, Valid: true}, CompletedAt: pgtype.Timestamptz{}},
		{ID: 3, Name: "Grandchild", ParentID: pgtype.Int8{Int64: 2, Valid: true}, CompletedAt: pgtype.Timestamptz{}},
		{ID: 4, Name: "Done child", ParentID: pgtype.Int8{Int64: 1, Valid: true}, CompletedAt: ts(2026, 7, 1)},
		{ID: 5, Name: "Snoozed child", ParentID: pgtype.Int8{Int64: 1, Valid: true}, SnoozeUntil: ts(2026, 7, 20)},
		{ID: 6, Name: "Unrelated"},
	}

	expr, err := Parse(`completed=false AND snoozed=false AND ^goal_id=1`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Tasks 1, 2, 3: incomplete and unsnoozed in the goal-1 subtree
	// Task 4 is completed, task 5 is snoozed, task 6 is unrelated
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("ids = %v, want [1 2 3]", ids)
	}
}

func TestEvalWorkedExampleTextAndTransitiveGoal(t *testing.T) {
	// groceries AND ^goal_id=2 (show_all=any)
	// → incomplete goal-2-tree tasks containing "groceries"
	tasks := []Task{
		{ID: 1, Name: "Goal 2 root", GoalID: pgtype.Int8{Int64: 2, Valid: true}, CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "groceries list", ParentID: pgtype.Int8{Int64: 1, Valid: true}, CompletedAt: pgtype.Timestamptz{}},
		{ID: 3, Name: "other task", ParentID: pgtype.Int8{Int64: 1, Valid: true}, CompletedAt: pgtype.Timestamptz{}},
		{ID: 4, Name: "groceries elsewhere", CompletedAt: pgtype.Timestamptz{}},
	}

	expr, err := Parse(`groceries AND ^goal_id=2`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Task 2: has "groceries" and is in goal-2 subtree
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("ids = %v, want [2]", ids)
	}
}

func TestEvalWorkedExampleQuotedAndReview(t *testing.T) {
	// "AND review" (show_all=off)
	// → incomplete, unsnoozed tasks containing the literal text "AND review"
	tasks := []Task{
		{ID: 1, Name: "AND review meeting", CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "review notes", CompletedAt: pgtype.Timestamptz{}},
		{ID: 3, Name: "AND review done", CompletedAt: ts(2026, 7, 1)},
		{ID: 4, Name: "AND review snoozed", SnoozeUntil: ts(2026, 7, 20), CompletedAt: pgtype.Timestamptz{}},
	}

	expr, err := Parse(`"AND review"`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Task 1: matches text, incomplete, unsnoozed
	// Task 2: doesn't contain "AND review"
	// Task 3: completed (implicit completed=false excludes it)
	// Task 4: snoozed (implicit snoozed=false excludes it)
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalFR008OverrideBothExplicit(t *testing.T) {
	// Both completed and snoozed explicitly mentioned → no implicit defaults
	tasks := []Task{
		{ID: 1, Name: "Task A", CompletedAt: pgtype.Timestamptz{}, SnoozeUntil: pgtype.Timestamptz{}},
		{ID: 2, Name: "Task B", CompletedAt: ts(2026, 7, 1), SnoozeUntil: ts(2026, 7, 20)},
		{ID: 3, Name: "Task C", CompletedAt: ts(2026, 7, 1), SnoozeUntil: pgtype.Timestamptz{}},
		{ID: 4, Name: "Task D", CompletedAt: pgtype.Timestamptz{}, SnoozeUntil: ts(2026, 7, 20)},
	}

	// completed=true AND snoozed=true with show_all=off
	// Since both are explicit, no implicit defaults are added.
	expr, err := Parse(`completed=true AND snoozed=true`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Task 2: completed AND snoozed
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("ids = %v, want [2]", ids)
	}
}

func TestEvalFR008OverrideCompletedExplicitSnoozedDefault(t *testing.T) {
	// Only completed is explicit → snoozed gets implicit snoozed=false
	tasks := []Task{
		{ID: 1, Name: "Completed unsnoozed", CompletedAt: ts(2026, 7, 1), SnoozeUntil: pgtype.Timestamptz{}},
		{ID: 2, Name: "Completed snoozed", CompletedAt: ts(2026, 7, 1), SnoozeUntil: ts(2026, 7, 20)},
		{ID: 3, Name: "Incomplete unsnoozed", CompletedAt: pgtype.Timestamptz{}, SnoozeUntil: pgtype.Timestamptz{}},
	}

	expr, err := Parse(`completed=true`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// completed=true → explicit mention of completed
	// show_all=off, snoozed not mentioned → implicit snoozed=false
	// Task 1 matches (completed, not snoozed); Task 2 excluded (snoozed)
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalFR008OverrideSnoozedExplicitCompletedDefault(t *testing.T) {
	// Only snoozed is explicit → completed gets implicit completed=false
	tasks := []Task{
		{ID: 1, Name: "Incomplete snoozed", CompletedAt: pgtype.Timestamptz{}, SnoozeUntil: ts(2026, 7, 20)},
		{ID: 2, Name: "Completed snoozed", CompletedAt: ts(2026, 7, 1), SnoozeUntil: ts(2026, 7, 20)},
		{ID: 3, Name: "Incomplete unsnoozed", CompletedAt: pgtype.Timestamptz{}, SnoozeUntil: pgtype.Timestamptz{}},
	}

	expr, err := Parse(`snoozed=true`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, false, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// snoozed=true → explicit mention of snoozed
	// show_all=off, completed not mentioned → implicit completed=false
	// Task 1 matches (incomplete, snoozed); Task 2 excluded (completed)
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalFR008ShowAllOnNoDefaults(t *testing.T) {
	// show_all=true → no implicit conditions at all
	tasks := []Task{
		{ID: 1, Name: "foo incomplete", CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "foo completed", CompletedAt: ts(2026, 7, 1)},
		{ID: 3, Name: "foo snoozed", SnoozeUntil: ts(2026, 7, 20)},
	}

	expr, err := Parse("foo")
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// show_all=true → no implicit completed/snoozed filters
	// All three tasks match
	if len(ids) != 3 {
		t.Errorf("ids = %v, want [1 2 3]", ids)
	}
}

func TestEvalDateImpliesCompleted(t *testing.T) {
	// A date comparison on completed implies the task is completed.
	// An incomplete task should never match a date comparison on completed.
	tasks := []Task{
		{ID: 1, Name: "Completed before", CompletedAt: ts(2025, 6, 1)},
		{ID: 2, Name: "Incomplete", CompletedAt: pgtype.Timestamptz{}},
		{ID: 3, Name: "Completed after", CompletedAt: ts(2026, 6, 1)},
	}

	expr, err := Parse(`completed < 2026-01-01`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Only task 1: completed before 2026-01-01; task 2 is incomplete (no completed_at)
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalNonexistentParentId(t *testing.T) {
	// parent_id referencing a non-existent task → empty result (not an error)
	tasks := []Task{
		{ID: 1, Name: "Task A", ParentID: pgtype.Int8{Int64: 999, Valid: true}},
		{ID: 2, Name: "Task B"},
	}

	expr, err := Parse(`parent_id=1`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// No task has parent_id=1
	if len(ids) != 0 {
		t.Errorf("expected empty result, got %v", ids)
	}
}

func TestEvalNonexistentGoalId(t *testing.T) {
	// goal_id referencing a non-existent goal → empty result
	tasks := []Task{
		{ID: 1, Name: "Task A"},
		{ID: 2, Name: "Task B", GoalID: pgtype.Int8{Int64: 5, Valid: true}},
	}

	expr, err := Parse(`goal_id=99`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 0 {
		t.Errorf("expected empty result, got %v", ids)
	}
}

func TestEvalCompletedNeTrue(t *testing.T) {
	// completed!=true → tasks where completed_at is NOT set
	tasks := []Task{
		{ID: 1, Name: "Incomplete", CompletedAt: pgtype.Timestamptz{}},
		{ID: 2, Name: "Completed", CompletedAt: ts(2026, 7, 1)},
	}

	expr, err := Parse(`completed!=true`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Only task 1: not completed
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ids = %v, want [1]", ids)
	}
}

func TestEvalSnoozedNeFalse(t *testing.T) {
	// snoozed!=false → tasks where snoozed is true (snooze_until > today)
	tasks := []Task{
		{ID: 1, Name: "Not snoozed", SnoozeUntil: pgtype.Timestamptz{}},
		{ID: 2, Name: "Snoozed", SnoozeUntil: ts(2026, 7, 20)},
		{ID: 3, Name: "Past snooze", SnoozeUntil: ts(2026, 7, 5)}, // snooze in the past
	}

	expr, err := Parse(`snoozed!=false`)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := Evaluate(expr, tasks, true, today(2026, 7, 10))
	if err != nil {
		t.Fatal(err)
	}

	// Only task 2: snoozed (snooze_until > today)
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("ids = %v, want [2]", ids)
	}
}
