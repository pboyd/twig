# Data Model: Daily Planning

## Entity: PlanEntry

A single scheduled segment of one user's one calendar day. Identity is `(user_id, day, id)`. The CLI surface exposes only `(day, id)` because `user_id` is implicit from the authenticated caller.

### Storage — `plan_entries` table

| Column | Type | Constraints |
|---|---|---|
| `user_id` | `BIGINT` | NOT NULL · REFERENCES `users(id)` ON DELETE CASCADE |
| `day` | `DATE` | NOT NULL |
| `id` | `INT` | NOT NULL · positive · per-day sequential |
| `task_id` | `BIGINT` | NULLABLE · REFERENCES `tasks(id)` ON DELETE CASCADE |
| `name` | `VARCHAR(255)` | NULLABLE |
| `start_minute` | `SMALLINT` | NOT NULL · CHECK `BETWEEN 0 AND 1439` |
| `duration_minute` | `SMALLINT` | NOT NULL · CHECK `> 0 AND <= 1440` |

**Composite primary key**: `(user_id, day, id)`.

**Additional table-level constraints**:
- `CHECK (start_minute + duration_minute <= 1440)` — no entry crosses midnight.
- `CHECK (task_id IS NOT NULL OR name IS NOT NULL)` — an entry without a linked task must carry an explicit name (FR-006).

**Index**: `plan_entries_task_id_idx ON plan_entries (task_id)` — supports cascade-on-task-delete and any future task-anchored lookups.

### Why integer minutes on a date

The spec mandates local-timezone semantics on the user's machine and forbids cross-midnight entries. Storing the time component as integer minutes within a named day removes all timezone math from the server: a row says nothing more than "on this calendar day, this many minutes after midnight, for this many minutes." The CLI is responsible for translating between human time formats (`13:15`, `1:15pm`, `01:15 PM`) and the minute count. See research.md §1.

### Half-open interval semantics

The interval represented by an entry is `[start_minute, start_minute + duration_minute)`. Two entries A and B on the same `(user_id, day)` overlap iff `A.start < B.end AND B.start < A.end`. Two entries that touch — `A.end == B.start` or vice versa — do not overlap. (Clarification 2026-05-21.) This is the convention used by the `internal/plan.Overlap` helper, the handler's overlap check, and the handler tests.

### Per-day id allocation

When a new entry is inserted for `(user_id, day)`, its `id` is `COALESCE(MAX(id), 0) + 1` over the existing rows for that `(user_id, day)`. Removing entries does not renumber the remaining rows (FR-003): after `1, 2, 3` and `rm 2`, the next insert is `4`.

## Relationships

```
users (1) ──< (many) plan_entries          ON DELETE CASCADE  (existing convention)
tasks (1) ──< (0..many) plan_entries       ON DELETE CASCADE  (mirrors pomodoros)
```

A plan entry is owned by exactly one user. A task may be referenced by zero, one, or many plan entries on a given day (clarification 2026-05-21 — multi-entry per task is allowed).

## Lifecycle

```
                     (handler insert in tx)
       (nothing)  ────────────────────────────────►  Active
                                                       │
                                                       │  RenamePlanEntry / MovePlanEntry
                                                       │  ◄──────── (still Active) ────────
                                                       ▼
                                              RemovePlanEntry  /  ClearPlan past it
                                                       │
                                                       ▼
                                                   (deleted)
```

There is no soft-delete state. A `ClearPlan` that straddles an entry shortens its `duration_minute` (the entry stays Active) and removes every later entry on the day (those are deleted).

## Computed display fields

These are produced by the read path (`ListPlanEntries`) but are not stored:

- **Display name**: `name` if non-null; otherwise the linked task's `name`. The handler joins to `tasks` only when at least one returned row has `name IS NULL`.
- **End-of-day rendering range**: the grid renderer takes `min(start_minute)` across the day's entries as the grid floor and `max(start_minute + duration_minute)` as the grid ceiling. Hours before the floor and after the ceiling are omitted; gaps in the middle are rendered as empty rows.

## Migration

`db/migrations/000006_plan_entries.up.sql`:

```sql
CREATE TABLE plan_entries (
    user_id         BIGINT   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day             DATE     NOT NULL,
    id              INT      NOT NULL,
    task_id         BIGINT            REFERENCES tasks(id) ON DELETE CASCADE,
    name            VARCHAR(255),
    start_minute    SMALLINT NOT NULL CHECK (start_minute BETWEEN 0 AND 1439),
    duration_minute SMALLINT NOT NULL CHECK (duration_minute > 0 AND duration_minute <= 1440),
    CHECK (start_minute + duration_minute <= 1440),
    CHECK (task_id IS NOT NULL OR name IS NOT NULL),
    PRIMARY KEY (user_id, day, id)
);

CREATE INDEX plan_entries_task_id_idx ON plan_entries (task_id);
```

`db/migrations/000006_plan_entries.down.sql`:

```sql
DROP TABLE IF EXISTS plan_entries;
```
