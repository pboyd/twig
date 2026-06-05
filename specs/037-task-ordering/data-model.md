# Phase 1 Data Model: User-Configurable Task Order

## Entity: Task (modified)

Existing `tasks` table (PostgreSQL). This feature adds one column.

| Field | Type | Notes |
|---|---|---|
| id | BIGINT identity PK | unchanged |
| user_id | BIGINT | unchanged (owning user; scopes all queries) |
| name | VARCHAR(255) NOT NULL | unchanged |
| description | TEXT NOT NULL DEFAULT '' | unchanged |
| due | TIMESTAMPTZ | unchanged |
| parent_id | BIGINT → tasks(id) ON DELETE CASCADE | unchanged; NULL = root group |
| completed_at | TIMESTAMPTZ | unchanged |
| estimate | INT | unchanged |
| **position** | **INTEGER NOT NULL DEFAULT 0** | **NEW** — order within the task's `(user_id, parent_id)` sibling group |

### `position` semantics

- Meaningful **only relative to siblings** — tasks sharing the same `user_id` and `parent_id` (with `parent_id IS NULL` forming the root group).
- Siblings are ordered by `position ASC, id ASC`. The `id` tiebreaker guarantees a deterministic total order even if two rows transiently share a `position`.
- After any reorder, a group is renumbered to a contiguous `0..n-1` sequence. Between reorders, values are contiguous; `DEFAULT 0` only applies before backfill/assignment and is immediately resolved by create/backfill logic.
- `position` is **not** editable through `UpdateTask`. It is set by: backfill (migration), create (end of group), re-parent via `UpdateTask` (end of destination group), and `ReorderTask`.

### Validation / invariants

- Within a sibling group, every task has a defined position; no task is dropped or duplicated by any reorder (SC-005).
- A reorder never changes `parent_id` (FR-004).
- A `ReorderTask` anchor must be a sibling of the moved task (same `user_id`, same `parent_id`) and not the task itself.

### Index

- Add `tasks_user_parent_position_idx ON tasks (user_id, parent_id, position)` to support ordered sibling reads. (Existing `tasks_parent_id_idx` remains.)

## Migration: `000008_task_position`

**Up** (sketch):

```sql
ALTER TABLE tasks ADD COLUMN position INTEGER NOT NULL DEFAULT 0;

-- Backfill so existing order (id ascending within each sibling group) is preserved.
WITH ordered AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY user_id, parent_id
               ORDER BY id
           ) - 1 AS pos
    FROM tasks
)
UPDATE tasks t SET position = ordered.pos
FROM ordered WHERE ordered.id = t.id;

CREATE INDEX tasks_user_parent_position_idx ON tasks (user_id, parent_id, position);
```

**Down**:

```sql
DROP INDEX IF EXISTS tasks_user_parent_position_idx;
ALTER TABLE tasks DROP COLUMN position;
```

> Postgres groups all `NULL` `parent_id` rows together in `PARTITION BY`, so root-level tasks form one group — the desired behavior.

## State transitions (position lifecycle)

```
create task            → position = COALESCE(MAX(position) in (user,parent) group, -1) + 1   (end)
reparent via UpdateTask → position = end of NEW (user,parent) group; old group keeps order
ReorderTask before X    → remove task from group, reinsert immediately before X, renumber 0..n-1
ReorderTask after X     → remove task from group, reinsert immediately after X,  renumber 0..n-1
delete task            → siblings keep relative order (gap is harmless; next reorder renumbers)
complete/uncomplete    → position unchanged (hidden tasks retain their slot; FR-014)
```

## Derived / proto representation

- `task.v1.Task` gains `int64 position = 9;` — read-only on writes (ignored by `CreateTask`/`UpdateTask`), populated on every read so all clients can sort siblings.
- Tree builders sort siblings by `position` (then `id`):
  - `internal/cli/render.go` `SortNodes`
  - `services/twig-web/src/lib/tree.ts` `buildTree`
