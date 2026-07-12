package filter

import (
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Task represents the subset of a task row the evaluator needs.
type Task struct {
	ID          int64
	Name        string
	Description string
	CompletedAt pgtype.Timestamptz
	SnoozeUntil pgtype.Timestamptz
	ParentID    pgtype.Int8
	GoalID      pgtype.Int8
}

// evalCtx carries per-evaluation lookup structures needed by transitive
// relation matching (^parent_id, ^goal_id).
type evalCtx struct {
	byID map[int64]*Task
}

// Evaluate evaluates the expression against the given tasks and returns
// the matching task IDs in ascending order.
func Evaluate(e Expression, tasks []Task, showAll bool, today time.Time) ([]int64, error) {
	todayDay := todayToDay(today)

	byID := make(map[int64]*Task, len(tasks))
	for i := range tasks {
		byID[tasks[i].ID] = &tasks[i]
	}
	ctx := &evalCtx{byID: byID}

	// Determine implicit visibility conditions (FR-008).
	hasCompleted := false
	hasSnoozed := false
	for _, c := range e.Conditions {
		switch cc := c.(type) {
		case *BoolCondition:
			if cc.Field == BoolFieldCompleted {
				hasCompleted = true
			}
			if cc.Field == BoolFieldSnoozed {
				hasSnoozed = true
			}
		case *DateCondition:
			if cc.Field == DateFieldCompleted {
				hasCompleted = true
			}
		case *RelCondition:
			// RelCondition on parent_id/goal_id does not count as mentioning completed/snoozed.
		}
	}

	var result []int64
	for i := range tasks {
		t := &tasks[i]

		// Apply explicit conditions.
		if !matchAll(ctx, e.Conditions, t, todayDay) {
			continue
		}

		// Apply implicit default-visibility (FR-008).
		if !showAll {
			if !hasCompleted && isCompleted(t) {
				continue
			}
			if !hasSnoozed && isSnoozed(t, todayDay) {
				continue
			}
		}

		result = append(result, t.ID)
	}

	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func matchAll(ctx *evalCtx, conditions []Condition, t *Task, todayDay day) bool {
	for _, c := range conditions {
		if !matchCondition(ctx, c, t, todayDay) {
			return false
		}
	}
	return true
}

func matchCondition(ctx *evalCtx, c Condition, t *Task, todayDay day) bool {
	switch cc := c.(type) {
	case *TextCondition:
		return matchText(cc.Term, t)
	case *BoolCondition:
		return matchBool(cc, t, todayDay)
	case *DateCondition:
		return matchDate(cc, t)
	case *RelCondition:
		return matchRel(ctx, cc, t)
	}
	return false
}

func matchText(term string, t *Task) bool {
	term = strings.ToLower(term)
	return strings.Contains(strings.ToLower(t.Name), term) ||
		strings.Contains(strings.ToLower(t.Description), term)
}

func matchBool(c *BoolCondition, t *Task, todayDay day) bool {
	var actual bool
	switch c.Field {
	case BoolFieldCompleted:
		actual = isCompleted(t)
	case BoolFieldSnoozed:
		actual = isSnoozed(t, todayDay)
	}
	return applyBoolOp(actual, c.Op, c.Value)
}

func matchDate(c *DateCondition, t *Task) bool {
	switch c.Field {
	case DateFieldCompleted:
		if !t.CompletedAt.Valid {
			return false
		}
		completedDay := timeToDay(t.CompletedAt.Time.UTC())
		cmpDay, err := parseDay(c.Day)
		if err != nil {
			return false
		}
		return compareDays(completedDay, c.Op, cmpDay)
	}
	return false
}

func matchRel(ctx *evalCtx, c *RelCondition, t *Task) bool {
	if c.Transitive {
		var eq bool
		switch c.Field {
		case RelFieldParentID:
			eq = hasAncestor(ctx, t, c.ID)
		case RelFieldGoalID:
			eq = hasGoalInLineage(ctx, t, c.ID)
		}
		if c.Op == OpNe {
			return !eq
		}
		return eq
	}

	switch c.Field {
	case RelFieldParentID:
		if !t.ParentID.Valid {
			return false
		}
		return applyIntOp(t.ParentID.Int64, c.Op, c.ID)
	case RelFieldGoalID:
		if !t.GoalID.Valid {
			return false
		}
		return applyIntOp(t.GoalID.Int64, c.Op, c.ID)
	}
	return false
}

// hasAncestor reports whether task id appears anywhere on t's parent chain
// (t itself does not count). Guards against cycles with a visited set.
func hasAncestor(ctx *evalCtx, t *Task, id int64) bool {
	visited := make(map[int64]bool)
	cur := t
	for cur.ParentID.Valid {
		pid := cur.ParentID.Int64
		if visited[pid] {
			return false // cycle guard
		}
		visited[pid] = true
		if pid == id {
			return true
		}
		parent, ok := ctx.byID[pid]
		if !ok {
			return false
		}
		cur = parent
	}
	return false
}

// hasGoalInLineage reports whether t or any of its ancestors is the
// association root for goal id.
func hasGoalInLineage(ctx *evalCtx, t *Task, id int64) bool {
	visited := make(map[int64]bool)
	cur := t
	for {
		if cur.GoalID.Valid && cur.GoalID.Int64 == id {
			return true
		}
		if !cur.ParentID.Valid {
			return false
		}
		pid := cur.ParentID.Int64
		if visited[pid] {
			return false // cycle guard
		}
		visited[pid] = true
		parent, ok := ctx.byID[pid]
		if !ok {
			return false
		}
		cur = parent
	}
}

func isCompleted(t *Task) bool {
	return t.CompletedAt.Valid
}

func isSnoozed(t *Task, todayDay day) bool {
	if !t.SnoozeUntil.Valid {
		return false
	}
	snoozeDay := timeToDay(t.SnoozeUntil.Time.UTC())
	return snoozeDay.after(todayDay)
}

func applyBoolOp(actual bool, op Op, expected bool) bool {
	switch op {
	case OpEq:
		return actual == expected
	case OpNe:
		return actual != expected
	}
	return false
}

func applyIntOp(actual int64, op Op, expected int64) bool {
	switch op {
	case OpEq:
		return actual == expected
	case OpNe:
		return actual != expected
	case OpLt:
		return actual < expected
	case OpLe:
		return actual <= expected
	case OpGt:
		return actual > expected
	case OpGe:
		return actual >= expected
	}
	return false
}

// day is a simple year/month/day tuple for date-only comparisons.
type day struct {
	year, month, day int
}

func (d day) after(other day) bool {
	if d.year != other.year {
		return d.year > other.year
	}
	if d.month != other.month {
		return d.month > other.month
	}
	return d.day > other.day
}

func todayToDay(t time.Time) day {
	return day{year: t.Year(), month: int(t.Month()), day: t.Day()}
}

func timeToDay(t time.Time) day {
	return day{year: t.Year(), month: int(t.Month()), day: t.Day()}
}

func parseDay(s string) (day, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return day{}, err
	}
	return day{year: t.Year(), month: int(t.Month()), day: t.Day()}, nil
}

func compareDays(a day, op Op, b day) bool {
	switch op {
	case OpEq:
		return a == b
	case OpNe:
		return a != b
	case OpLt:
		return !a.after(b) && a != b
	case OpLe:
		return !a.after(b)
	case OpGt:
		return a.after(b)
	case OpGe:
		return a.after(b) || a == b
	}
	return false
}
