# Data Model: Complete Tasks

This feature extends the existing `tasks` table from feature 001 (Task CRUD)
and the per-user scoping added in feature 003 (User Auth). It introduces no
new tables.

## Schema delta

```sql
-- 000004_complete.up.sql
ALTER TABLE tasks
    ADD COLUMN completed_at TIMESTAMPTZ;
```

```sql
-- 000004_complete.down.sql
ALTER TABLE tasks
    DROP COLUMN completed_at;
```

No index is added. The column is read in two ways:

1. By `id` (already the primary key) — for `CompleteTask` and the parent-of
   check in `CreateTask` / `UpdateTask`.
2. As an output column on `ListTasks` — already scanned via `SELECT *`.

Neither path benefits from a dedicated index at this scale.

## Updated entity

### Task

| Field           | Type             | Nullable | Notes |
|-----------------|------------------|----------|-------|
| `id`            | `BIGINT`         | no       | Identity column (unchanged) |
| `name`          | `VARCHAR(255)`   | no       | Trimmed non-empty (unchanged) |
| `description`   | `TEXT`           | no       | Defaults to `''` (unchanged) |
| `due`           | `TIMESTAMPTZ`    | yes      | Optional due date (unchanged) |
| `parent_id`     | `BIGINT`         | yes      | FK to `tasks(id)` `ON DELETE CASCADE` (unchanged) |
| `user_id`       | `BIGINT`         | no       | Owner — from feature 003 (unchanged) |
| **`completed_at`** | **`TIMESTAMPTZ`** | **yes**  | **NEW. `NULL` = incomplete; non-null = the moment of completion (UTC).** |

### Derived state

- `is_complete(task) := task.completed_at IS NOT NULL`
- `is_terminal_complete_subtree(task) := is_complete(task) AND
   ∀ d ∈ descendants(task) : is_complete(d)`

## Invariants

The following invariants are maintained by the handler, not the database (no
trigger, no check constraint — they would duplicate logic with no payoff at
this scale):

- **I-1 — Completion downward-closure**: If a task is complete, every one of
  its descendants is complete. Enforced by:
  - `CompleteTask` refuses to mark a task complete when any descendant is
    incomplete (recursive CTE).
  - `CreateTask` refuses to create a child under a complete parent
    (FR-013).
  - `UpdateTask` refuses to re-parent a task under a complete parent
    (FR-014).
- **I-2 — Completion is terminal**: Once `completed_at` is set on a row, it is
  never modified again (no reopen). Enforced by the SQL
  `SET completed_at = COALESCE(completed_at, NOW())` in `CompleteTask` and by
  the fact that no other code path updates the column.
- **I-3 — Per-user isolation**: Every read and write of `completed_at` happens
  through a query already scoped by `user_id = $n` (inherited from feature
  003). Completion does not introduce any cross-user visibility.

## Lifecycle

A task lives in exactly one of two states with respect to completion:

```text
                ┌───────────────────────────────┐
                │                               │
   created  ──► │  Incomplete (completed_at NULL)│
                │                               │
                └────────────┬──────────────────┘
                             │  CompleteTask
                             │  (only if every descendant is complete
                             │   or the task has no descendants)
                             ▼
                ┌───────────────────────────────┐
                │                               │
                │  Complete (completed_at = t₀) │  (terminal — no transition out)
                │                               │
                └───────────────────────────────┘
```

Re-invoking `CompleteTask` on a task already in the Complete state is a no-op
that returns the row unchanged (preserving `t₀`).

## Query additions

The following query definitions are added to `db/queries/task.sql` and consumed
via `sqlc`-generated bindings. The shapes shown here are the contract;
implementation may rename parameters but not change semantics.

```sql
-- name: CompleteTask :one
-- Marks the task complete if it is not already; returns the resulting row.
-- Idempotent: re-completing preserves the original timestamp.
UPDATE tasks
SET completed_at = COALESCE(completed_at, NOW())
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: HasIncompleteDescendants :one
-- True if any descendant of (id, user_id) has completed_at IS NULL.
WITH RECURSIVE descendants AS (
    SELECT id, completed_at
      FROM tasks
     WHERE parent_id = $1 AND user_id = $2
    UNION ALL
    SELECT t.id, t.completed_at
      FROM tasks t
      JOIN descendants d ON t.parent_id = d.id
     WHERE t.user_id = $2
)
SELECT EXISTS (
    SELECT 1 FROM descendants WHERE completed_at IS NULL
) AS has_incomplete;

-- name: GetParentCompletion :one
-- Reads only the parent's completed_at, used by CreateTask / UpdateTask
-- to enforce FR-013 / FR-014. Returns NULL when the parent is incomplete.
SELECT completed_at FROM tasks WHERE id = $1 AND user_id = $2;
```

The existing `CreateTask`, `GetTask`, `ListTasks`, `UpdateTask`,
`DeleteTask`, and `TaskExists` queries are unchanged in shape; the
`RETURNING *` clauses now include the new column automatically via the
regenerated `db/models.go`.
