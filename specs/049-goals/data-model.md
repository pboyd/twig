# Data Model: Goals

**Feature**: 049-goals | **Date**: 2026-06-11

## Entities

### Goal (new)

| Field | Type | Constraints |
|---|---|---|
| `id` | BIGINT identity PK | server-assigned, positive, never reused |
| `user_id` | BIGINT FK → users | owner; all queries scoped by it |
| `name` | VARCHAR(255) | required, non-blank after trim (CHECK `btrim(name) <> ''`) |
| `description` | TEXT | optional, default `''` |
| `due` | TIMESTAMPTZ | optional (nullable); no enforcement when past |
| `state` | TEXT | CHECK in (`incubating`, `committed`, `completed`, `archived`); default `incubating` |
| `position` | INTEGER | rank within the `(user_id, state)` group; lower sorts first |

Index: `goals_user_state_position_idx ON goals (user_id, state, position)`.

Goals do not nest — no parent column, by design (FR-004).

### Task (existing, extended)

| New field | Type | Constraints |
|---|---|---|
| `goal_id` | BIGINT NULL, FK → goals(id) `ON DELETE SET NULL` | at most one goal; only the association root row stores it |

**Effective goal** (derived, client-side): a task's effective goal is the
`goal_id` of its nearest self-or-ancestor that has one set. Stored on exactly
one task per subtree; descendants inherit (clarification 2026-06-11).

## Proto mapping

`goal.v1.Goal`: `id`, `name`, `description`, `due` (Timestamp, unset when
absent), `state` (enum `GOAL_STATE_{INCUBATING,COMMITTED,COMPLETED,ARCHIVED}`),
`position` (read-only). `task.v1.Task` gains `optional int64 goal_id = 11`
(read-only: populated on reads, ignored on Create/UpdateTask writes, like
`position`).

## State machine

```
            ┌────────────┐
   create → │ incubating │ ⇄ committed ⇄ completed
            └────────────┘       ⇅           ⇅
                  ⇵           archived ⇄ ────┘
```

All 12 directed transitions between the four states are permitted (FR-003).
There are no transition side effects on tasks. The only side effect is rank
placement: the goal is appended to the bottom of its new `(user, state)` group
(FR-013a).

Visibility rule: `incubating` and `committed` are shown by default;
`completed` and `archived` are hidden until the user asks (FR-011).

## Validation rules

| Rule | Where enforced |
|---|---|
| Goal name required, non-blank, ≤ 255 chars | handler + DB CHECK (mirrors tasks) |
| State must be one of the four values | proto enum + DB CHECK |
| `SetTaskGoal(task, goal)`: goal must exist and belong to the caller | handler (`NotFound`) |
| `SetTaskGoal`: rejected if any ancestor of the task has a `goal_id` | handler (`FailedPrecondition`) — descendants inherit; no nested differing goals |
| `SetTaskGoal`: rejected if any descendant of the task has a `goal_id` (when setting; clearing always allowed) | handler (`FailedPrecondition`) |
| `ReorderGoal`: anchor must be a goal in the same `(user, state)` group | handler (`InvalidArgument`/`NotFound`, mirrors ReorderTask) |
| Delete goal → associations cleared, tasks untouched | DB `ON DELETE SET NULL` |
| All goal rows scoped to the calling user | every sqlc query filters on `user_id` |

## Rank/position semantics

- `CreateGoal`: `position = COALESCE(MAX(position)+1, 0)` within the user's
  `incubating` group.
- `SetGoalState`: same formula against the destination group; positions of
  other goals are untouched.
- `ReorderGoal`: renumbers the affected group contiguously `0..n-1`
  (same approach as `ReorderTask`).
- All listings (TUI, CLI) order by state group, then `position` ascending.

## Lifecycle interactions (spec FR-007)

| Event | Effect on associated tasks |
|---|---|
| Goal completed / archived / any state change | none |
| Goal deleted | `tasks.goal_id` → NULL (cascade rule); tasks otherwise untouched |
| Task deleted | row gone; descendants cascade-delete as today; goal untouched |
| Task re-parented (UpdateTask/move) such that a goal-bearing task would gain a goal-bearing ancestor or descendant | rejected with `FailedPrecondition` — the no-nested-goals invariant is absolute; the user must clear one association first |
