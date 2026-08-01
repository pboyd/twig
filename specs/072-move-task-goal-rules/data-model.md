# Phase 1 Data Model: Goal Handling When Moving Tasks

No schema migration. This feature changes which rows are written, not the shape of any table.

## Entities

### Task (`tasks`)

Existing table. Fields relevant to this feature:

| Field | Type | Notes |
|---|---|---|
| `id` | `bigint` | Primary key. |
| `user_id` | `bigint` | Every query in this feature is scoped by it. |
| `parent_id` | `bigint NULL` | Self-reference. `NULL` = top-level task. The field this feature keys off. |
| `goal_id` | `bigint NULL` | Link to a goal. **Invariant: non-NULL only when `parent_id IS NULL`.** |
| `position` | `bigint` | Order within the sibling group sharing a `parent_id`. |
| `completed_at` | `timestamptz NULL` | A completed task cannot become a move destination. |

### Goal (`goals`)

Existing table, read-only in this feature. Relevant only through its state: a goal may be
in progress, incubating, on hold, completed, or archived. The last two are "closed" — `SetTaskGoal`
refuses new links to them, but promotion may still carry an existing link to one (FR-007).

### Effective goal (derived, not stored)

For a task `T`:

- if `T.parent_id IS NULL` → `T.goal_id`
- otherwise → the `goal_id` of the nearest ancestor of `T` that has one, or none

Computed client-side today by `internal/goal.EffectiveGoalID` (`internal/goal/goal.go:111`) and
server-side by the new `NearestAncestorGoal` query. This feature's whole purpose is to keep the
stored `goal_id` consistent with this derivation across moves.

## The invariant

> A task with a parent never holds its own `goal_id`.

Not enforced by a database constraint — the spec chose opportunistic repair over a backfill, and a
`CHECK` constraint would reject writes against existing violating rows. Enforced instead in the
`UpdateTask` handler (this feature) and in `SetTaskGoal` (already, via `AncestorHasGoal` /
`DescendantHasGoal`).

Rows violating it may exist from before this change. They are legal to read, the clients render them
correctly by falling back to the derivation above, and they are corrected when the task or an
ancestor is next moved.

## State transitions on a parent change

Let `T` be the moved task, `old` its stored `parent_id`, `new` the requested one.

| Transition | Condition | Effect on `goal_id` |
|---|---|---|
| **No move** | `old == new` | Unchanged, for `T` and all descendants. |
| **Descent** | `new` is non-NULL and `new != old` | `T.goal_id → NULL`. Every descendant of `T`, any depth: `goal_id → NULL`. |
| **Promotion** | `new IS NULL` and `old` is non-NULL | If `T.goal_id IS NULL`: set it to `NearestAncestorGoal(T)` evaluated **before** the parent is rewritten; leave NULL if there is none. If `T.goal_id` is already set, leave it. Descendants untouched. |

Ordering matters in two places:

1. `NearestAncestorGoal(T)` must run **before** `UpdateTask` rewrites `parent_id` — afterwards `T`
   has no ancestors and the answer is always empty.
2. `ClearSubtreeGoals(T)` may run before or after the parent rewrite; the descendant set is
   unaffected by `T`'s own parent changing. Running it inside the same transaction is what matters.

## Validation rules

Carried over unchanged from the current handler, all still rejecting the move (FR-009):

- Destination task must exist for this user.
- Destination must not be `T` itself or any descendant of `T` (cycle).
- Destination must not be completed.

Removed:

- The nested-goal precondition. No move is refused for a goal reason (FR-001).

Unchanged elsewhere:

- `SetTaskGoal` still refuses to link a goal to a task that has a parent, that has a goal-linked
  ancestor, or that has a goal-linked descendant, and still refuses closed goals (FR-010).

## Query surface

New in `services/twig/db/queries/task.sql`:

- **`NearestAncestorGoal(id, user_id) → bigint NULL`** — recursive walk up `parent_id` from `id`,
  tracking depth, returning the `goal_id` of the closest ancestor having one. Strict ancestors only;
  `id`'s own `goal_id` is not considered.
- **`ClearSubtreeGoals(parent_id, user_id)`** — recursive `UPDATE tasks SET goal_id = NULL` over all
  descendants of `parent_id`. Does not touch the row identified by `parent_id` itself.

Unchanged and still used by `SetTaskGoal`: `AncestorHasGoal`, `DescendantHasGoal`, `SetTaskGoal`,
`ClearTaskGoal`, `GetMaxSiblingPosition`, `UpdateTaskPosition`.

Regenerate `services/twig/internal/db/task.sql.go` with `sqlc generate` from `services/twig/`.
