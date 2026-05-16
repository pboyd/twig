# Phase 1 Data Model: Task CRUD API

**Feature**: 001-task-crud-api | **Date**: 2026-05-16

## Entity: Task

A single unit of work. Tasks form a hierarchy of unrestricted depth via an optional self-reference.

| Field | Storage type | Proto type | Required | Notes |
|---|---|---|---|---|
| id | `BIGINT GENERATED ALWAYS AS IDENTITY` | `int64` | server-assigned | Primary key. Positive, increasing, never reused (FR-004). |
| name | `VARCHAR(255) NOT NULL` | `string` | yes | Non-empty after trimming; ≤ 255 chars (FR-002, FR-010). |
| description | `TEXT NOT NULL DEFAULT ''` | `string` | no | Empty string when absent (FR-003). |
| due | `TIMESTAMPTZ` (nullable) | `google.protobuf.Timestamp` | no | `NULL` / `nil` when absent. Stored and returned in UTC. |
| parent_id | `BIGINT REFERENCES tasks(id) ON DELETE CASCADE` (nullable) | `optional int64` | no | `NULL` ⇒ top-level task (FR-013). |

### Relationships

- **Self-reference (parent/child)**: `parent_id` → `tasks.id`. A task has at most one parent (FR-013) and any number of children. Depth is unbounded (FR-015).
- The foreign key enforces that a referenced parent exists (FR-014).
- `ON DELETE CASCADE`: deleting a task deletes its children, recursively deleting the whole subtree (FR-017).

### Validation rules

| Rule | Enforced by | Requirement |
|---|---|---|
| Name present and non-blank (trimmed) | Handler (primary) + `CHECK (btrim(name) <> '')` (backstop) | FR-002 |
| Name length ≤ 255 (trimmed) | Handler (primary) + `VARCHAR(255)` (backstop) | FR-010 |
| `parent_id` references an existing task | Handler pre-check (clean error) + FK constraint (backstop) | FR-014 |
| No cycle: a task is never its own ancestor | Handler recursive-CTE check on update | FR-016 |
| Identifier never reused | Identity sequence | FR-004 |

### Lifecycle

`created` (CreateTask) → `updated` 0..n times (UpdateTask, full replace of name/description/due/parent_id) → `deleted` (DeleteTask, hard delete; cascades to descendants). There is no status/completion field — out of scope for this feature.

## Migration

`services/todo/db/migrations/000002_tasks.up.sql`:

```sql
CREATE TABLE tasks (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(255) NOT NULL CHECK (btrim(name) <> ''),
    description TEXT NOT NULL DEFAULT '',
    due         TIMESTAMPTZ,
    parent_id   BIGINT REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX tasks_parent_id_idx ON tasks (parent_id);
```

`services/todo/db/migrations/000002_tasks.down.sql`:

```sql
DROP TABLE IF EXISTS tasks;
```

The `tasks_parent_id_idx` index keeps the cascade delete and the recursive cycle-check query efficient as the hierarchy grows.

## sqlc queries

`services/todo/db/queries/task.sql` — sqlc regenerates `internal/db/` (adds `Task` to `models.go`, creates `task.sql.go`).

```sql
-- name: CreateTask :one
INSERT INTO tasks (name, description, due, parent_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1;

-- name: ListTasks :many
SELECT * FROM tasks ORDER BY id;

-- name: UpdateTask :one
UPDATE tasks
SET name = $2, description = $3, due = $4, parent_id = $5
WHERE id = $1
RETURNING *;

-- name: DeleteTask :one
DELETE FROM tasks WHERE id = $1
RETURNING id;

-- name: TaskExists :one
SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1) AS exists;

-- name: ParentChainContains :one
-- Walks the ancestor chain starting at the proposed parent ($1) and reports
-- whether the candidate task id ($2) appears in it. A true result means
-- attaching $2 under $1 would create a cycle.
WITH RECURSIVE chain AS (
    SELECT id, parent_id FROM tasks WHERE id = $1
    UNION ALL
    SELECT t.id, t.parent_id
    FROM tasks t
    JOIN chain c ON t.id = c.parent_id
)
SELECT EXISTS (SELECT 1 FROM chain WHERE id = $2) AS found;
```

### Query usage by RPC

| RPC | Queries | Behavior |
|---|---|---|
| CreateTask | `TaskExists` (if `parent_id` set), `CreateTask` | Validate name; reject unknown parent with `InvalidArgument`; insert; return row. |
| GetTask | `GetTask` | `pgx.ErrNoRows` ⇒ `NotFound`. |
| ListTasks | `ListTasks` | Returns flat list ordered by `id`; empty list when none. |
| UpdateTask | `GetTask` or rely on `RETURNING`, `TaskExists` + `ParentChainContains` (if `parent_id` set), `UpdateTask` | Validate name; reject unknown parent; reject cycle (`ParentChainContains` true ⇒ `InvalidArgument`); update; `UpdateTask` returning 0 rows ⇒ `NotFound`. |
| DeleteTask | `DeleteTask` | `DeleteTask` returning 0 rows ⇒ `NotFound`. FK `ON DELETE CASCADE` removes descendants. |

> `ParentChainContains($1 = new parent, $2 = task being updated)` returning `found = true` means the task is the proposed parent itself or one of its ancestors — i.e., the move would create a cycle and must be rejected.
