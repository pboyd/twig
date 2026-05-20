# Data Model — Pomodoro Tracking (005)

## Schema changes

### `tasks` (modified)

Add one column:

```sql
ALTER TABLE tasks
  ADD COLUMN estimate SMALLINT NOT NULL DEFAULT 0
    CHECK (estimate BETWEEN 0 AND 10);
```

- **`estimate`** — user's predicted number of pomodoros to complete the task.
- `0` is the default and means "no estimate" (FR-003).
- Domain enforced at the database level (FR-002). The handler additionally returns a friendly "break it down further" message for out-of-range values (FR-005) before reaching the database.
- No backfill needed — `DEFAULT 0` populates existing rows.

All existing queries that `RETURNING *` from `tasks` (CreateTask, UpdateTask, GetTask, ListTasks, CompleteTask) automatically pick up `estimate` after sqlc regeneration. No query rewrite required.

### `pomodoros` (new)

```sql
CREATE TABLE pomodoros (
    id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id   BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id   BIGINT      NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    start_at  TIMESTAMPTZ NOT NULL,
    end_at    TIMESTAMPTZ,
    complete  BOOLEAN     NOT NULL DEFAULT FALSE,
    CHECK (end_at IS NULL OR end_at >= start_at),
    CHECK (NOT complete OR end_at IS NOT NULL)
);

CREATE UNIQUE INDEX pomodoros_one_active_per_user
    ON pomodoros (user_id) WHERE end_at IS NULL;

CREATE INDEX pomodoros_task_id_idx ON pomodoros (task_id);
```

#### Fields

| Field | Type | Notes |
|---|---|---|
| `id` | `BIGINT` | Server-assigned, positive, increasing, never reused. |
| `user_id` | `BIGINT` | Owner. Cascade-deletes with the user (matches `tasks.user_id`'s policy for owner deletion, even though users are not actively deleted in this app). |
| `task_id` | `BIGINT` | Target task. `ON DELETE CASCADE` so that deleting a task removes all its pomodoros (FR-025, clarification Q1). |
| `start_at` | `TIMESTAMPTZ` | Moment the pomodoro began (server's `NOW()` at StartPomodoro). UTC by convention. |
| `end_at` | `TIMESTAMPTZ?` | `NULL` while active. Set to `start_at + 25 minutes` on completion, or `NOW()` on cancellation. |
| `complete` | `BOOLEAN` | `false` for active and canceled records; `true` only for completed records. |

#### Invariants

1. **Single active pomodoro per user (SC-003 / FR-014)**: enforced by `pomodoros_one_active_per_user`. Any second insert with `end_at IS NULL` for the same `user_id` fails with `unique_violation`, which the handler maps to Connect `AlreadyExists`.
2. **`end_at >= start_at`** (CHECK): a pomodoro cannot end before it started.
3. **`complete ⇒ end_at IS NOT NULL`** (CHECK): a completed pomodoro must have an end time. (The converse does not hold — canceled pomodoros also have an end time.)
4. **Completed end_at exactly = start_at + 25 min (FR-012)**: enforced at the handler level by the `CompleteActivePomodoro` query, which writes `end_at = start_at + INTERVAL '25 minutes'` rather than `NOW()`. Not a database constraint because the duration constant lives in application code (`pomodoro.Length`).
5. **Canceled `end_at` <= start_at + 25 min (FR-013)**: a natural consequence — cancel writes `NOW()`, and the API never lets the row sit active for more than the client's resume policy (the CLI auto-completes anything older than 25 min on resume; a server in isolation will still accept later cancellation, but the test suite asserts the FR-013 boundary via the client paths).

#### Derived values

- **Completed-pomodoro count for a task**: `SELECT count(*) FROM pomodoros WHERE task_id = $1 AND user_id = $2 AND complete`.
- **Pomodoro history for a task**: `SELECT id, task_id, start_at, end_at, complete FROM pomodoros WHERE task_id = $1 AND user_id = $2 ORDER BY start_at`.
- **Remaining time for an active pomodoro** (CLI-only computation): `max(0, (start_at + 25 min) - now())`.

## Pomodoro state machine

```text
                                    (start_at + 25 min, COMPLETE)
                ┌─── CompletePomodoro ──────────────────────────► COMPLETED
                │
   (no row)  ──┴──── StartPomodoro ──►  ACTIVE
                                          │
                                          └── CancelPomodoro ──► CANCELED
                                              (end_at = NOW(),
                                               complete = false)
```

- A row is created in **ACTIVE** state by `StartPomodoro` (only if no other active row exists for the user).
- **ACTIVE → COMPLETED** is a one-way transition triggered by `CompletePomodoro` (or by the CLI's stale-active auto-completion on resume).
- **ACTIVE → CANCELED** is a one-way transition triggered by `CancelPomodoro`.
- **COMPLETED** and **CANCELED** are terminal. No edits, no re-opens. Rows are only removed by their task being deleted (cascade).

## Lifecycle examples

- *Happy path*: `StartPomodoro(task=7)` → ACTIVE → 25 minutes elapse → CLI calls `CompletePomodoro` → COMPLETED (`end_at = start_at + 25m`, `complete = true`).
- *User canceled*: `StartPomodoro(task=7)` → ACTIVE → user presses `c` → CLI calls `CancelPomodoro` → CANCELED (`end_at = NOW()`, `complete = false`).
- *Quit and resume*: `StartPomodoro(task=7)` → ACTIVE → user presses `q` (no API call) → user later runs `task pom resume` → CLI sees `start_at < 25 min ago`, polls until expiry → calls `CompletePomodoro` → COMPLETED.
- *Stale active*: `StartPomodoro(task=7)` → ACTIVE → 2 hours pass → user runs `task pom resume` → CLI sees `now - start_at > 25 min` → calls `CompletePomodoro` immediately → COMPLETED (with `end_at = start_at + 25m`, *not* `now`).
- *Task deleted*: `DeleteTask(7)` → all pomodoros for task 7 are removed by FK cascade, regardless of state.

## Index rationale

| Index | Purpose |
|---|---|
| `pomodoros_pkey` (id) | Standard primary key. |
| `pomodoros_one_active_per_user` (user_id) WHERE end_at IS NULL | Enforces single-active invariant *and* makes `GetActivePomodoro` a one-row lookup. |
| `pomodoros_task_id_idx` (task_id) | Backs `ListPomodorosForTask` and `CountCompletedPomodorosForTask`, both used by `GetTask`. |
