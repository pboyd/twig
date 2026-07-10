# Data Model: Task Filters

**Feature**: 062-task-filters | **Date**: 2026-07-10

## Database

**No schema changes.** Filtering is a read-only operation over the existing `tasks` table, loaded through the existing `ListTasks` sqlc query. No new tables, columns, indexes, or migrations.

Existing `Task` attributes consumed by the evaluator (all already present):

| Attribute | Source column | Used by |
|-----------|--------------|---------|
| `id` | `tasks.id` | identity, `parent_id`/`^parent_id` resolution |
| `name`, `description` | `tasks.name`, `tasks.description` | free-text matching |
| `completed_at` | `tasks.completed_at` (nullable) | `completed` boolean + date comparisons |
| `snooze_until` | `tasks.snooze_until` (nullable) | `snoozed` boolean (vs. request `today`) |
| `parent_id` | `tasks.parent_id` (nullable) | tree structure, transitivity |
| `goal_id` | `tasks.goal_id` (nullable, association root only) | `goal_id` / `^goal_id` |
| `user_id` | `tasks.user_id` | scoping — evaluator only ever sees the caller's tasks |

## Filter Expression AST (`services/twig/internal/filter`)

```text
Expression
└── Conditions []Condition        # implicit AND of one or more

Condition (interface) — one of:
├── TextCondition
│   └── Term string               # case-insensitive substring vs name + description
├── BoolCondition
│   ├── Field  {completed, snoozed}
│   ├── Op     {=, !=}
│   └── Value  bool
├── DateCondition                 # only field: completed
│   ├── Op     {=, !=, <, <=, >, >=}
│   └── Day    civil date (YYYY-MM-DD, compared at UTC midnight)
└── RelCondition
    ├── Field      {parent_id, goal_id}
    ├── Transitive bool           # true when prefixed with ^
    └── ID         int64          # value must be a positive integer
```

**Validation rules** (parse-time, produce `InvalidArgument`):
- Empty expression → invalid (the TUI treats an empty input as "no filter" and never sends it).
- Unknown field name, unsupported operator for a field's type, malformed date (must be a real calendar date), non-integer ID, unterminated quoted string → invalid.
- `^` on anything except `parent_id`/`goal_id` → invalid.
- Multiple conditions joined by anything other than `AND` → invalid (reserved for future `OR`/`NOT`).

**Evaluation inputs** (per request): the caller's task rows, `show_all bool`, `today` civil date.

**Evaluation invariants**:
- Default-visibility rule (FR-008): if no `completed` condition exists in the expression, an implicit `completed=false` is applied when `show_all` is off; independently, if no `snoozed` condition exists, an implicit `snoozed=false` is applied when `show_all` is off. Explicit conditions replace only their own implicit counterpart.
- A `DateCondition` on `completed` implies `completed=true` (unset `completed_at` never satisfies a date comparison).
- Output is the set of task IDs whose rows satisfy every (explicit + implicit) condition. Ancestor inclusion is **not** part of evaluation — it is client presentation.

## Derived structures (built per evaluation, in memory)

| Structure | Purpose |
|-----------|---------|
| `children map[int64][]int64` | expand `^parent_id=N` to all descendants |
| `byID map[int64]*task` | ancestor walks, `goal` root lookup |
| goal roots `map[int64][]int64` (goalID → association-root task IDs) | expand `^goal_id=N` to subtree sets |

## TUI state additions (`internal/tui/model.go`)

| Field | Type | Meaning |
|-------|------|---------|
| `filterInput` | `textinput.Model` | the `/` input widget |
| `filterFocused` | `bool` | keystrokes go to the input vs. the list |
| `filterExpr` | `string` | last expression sent (pre-fills on re-open) |
| `filterMatches` | `map[int64]bool` | matched IDs from the last **valid** response; `nil` = no filter active |
| `filterInvalid` | `bool` | last response was `InvalidArgument` (indicator on, matches retained) |
| `filterGen` | `int` | generation counter; stale responses discarded |

**State transitions**:

```text
inactive ──'/'──▶ editing(empty) ──type──▶ editing(pending gen N)
editing ──response(gen N, ok)──▶ editing(matches updated)
editing ──response(gen N, invalid)──▶ editing(invalid indicator, matches kept)
editing ──Enter──▶ applied (focus → list, expression + matches retained)
editing ──Esc──▶ inactive (input cleared, matches nil)
applied ──'/'──▶ editing (input pre-filled with filterExpr)
applied ──Esc──▶ inactive
applied ──task mutation──▶ applied (expression re-fired after refresh)
```
