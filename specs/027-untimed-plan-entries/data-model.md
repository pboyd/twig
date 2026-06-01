# Phase 1 Data Model: Untimed Plan Entries

## Entity: Plan Entry

A single segment of one day's plan. Identity within a user's data is `(user_id, day, id)`. The only change for this feature is that **`start_minute` becomes optional**.

| Field | Type | Nullability | Rule |
|-------|------|-------------|------|
| `user_id` | BIGINT | NOT NULL | FK → `users(id)`; implicit from auth, never on the wire |
| `day` | DATE | NOT NULL | `YYYY-MM-DD` |
| `id` | INT | NOT NULL | Per-day sequential, server-assigned, never renumbered; defines creation order |
| `task_id` | BIGINT | NULL | FK → `tasks(id)`. NULL ⇒ event; non-NULL ⇒ task entry |
| `name` | VARCHAR(255) | NULL | Display override; falls back to the linked task's name when NULL |
| `start_minute` | SMALLINT | **NULL (changed)** | Minutes since local midnight, `0..1439`. **NULL ⇒ untimed.** |
| `duration_minute` | SMALLINT | NOT NULL | `> 0` and (when timed) `start_minute + duration_minute <= 1440` |
| `completed` | bool (derived) | — | Read-only: `task_id IS NOT NULL AND tasks.completed_at IS NOT NULL` |

### Derived classification

- **Timed**: `start_minute IS NOT NULL` → rendered on the day grid.
- **Untimed**: `start_minute IS NULL` → rendered in the untimed pane. Only task entries may be untimed.

### Constraints (after migration)

Retained (apply only when `start_minute` is non-NULL — a NULL makes the predicate unknown, which passes):
- `CHECK (start_minute BETWEEN 0 AND 1439)`
- `CHECK (start_minute + duration_minute <= 1440)`
- `CHECK (duration_minute > 0 AND duration_minute <= 1440)`
- `CHECK (task_id IS NOT NULL OR name IS NOT NULL)`

New:
- `CHECK (task_id IS NOT NULL OR start_minute IS NOT NULL)` — **events must be timed**. (An event is `task_id IS NULL`; this forbids an untimed event.)

Changed:
- `start_minute` column: `NOT NULL` → nullable.

### State transitions

```
                schedule (set start_minute)
   ┌──────────┐ ───────────────────────────▶ ┌────────┐
   │ Untimed  │                               │ Timed  │
   │ (task)   │ ◀─────────────────────────── │        │
   └──────────┘  unschedule (start_minute←NULL)└────────┘
```

- **Add untimed**: `AddPlanTask` with no start → row with `start_minute = NULL`, default duration.
- **Schedule**: `MovePlanEntry` with a start → `start_minute` set, overlap-checked, duration preserved (or overridden).
- **Unschedule**: `MovePlanEntry` with no/`null` start → `start_minute = NULL`, duration preserved, no overlap check.
- Events can never enter the Untimed state (DB CHECK + required start in `AddPlanEvent`).

## Migration

**Up** (`000007_plan_untimed.up.sql`):
```sql
ALTER TABLE plan_entries ALTER COLUMN start_minute DROP NOT NULL;
ALTER TABLE plan_entries
  ADD CONSTRAINT plan_entries_event_timed
  CHECK (task_id IS NOT NULL OR start_minute IS NOT NULL);
```

**Down** (`000007_plan_untimed.down.sql`):
```sql
ALTER TABLE plan_entries DROP CONSTRAINT plan_entries_event_timed;
-- Will fail if untimed rows exist; acceptable for a rollback.
ALTER TABLE plan_entries ALTER COLUMN start_minute SET NOT NULL;
```

## Query changes (`db/queries/plan.sql` → `sqlc generate`)

- **`ListPlanEntriesForDay`**: `ORDER BY plan_entries.start_minute` → `ORDER BY plan_entries.start_minute ASC NULLS FIRST, plan_entries.id ASC`.
- **`InsertPlanEntry`**: `start_minute` param becomes nullable (`pgtype.Int2` with `Valid` toggled).
- **`UpdatePlanEntryTime`**: `start_minute` param becomes nullable so a move can set a time **or** clear it (unschedule) in one query.
- **`DeletePlanEntriesFromMinute`**: unchanged — `start_minute >= $3` already excludes NULL (untimed) rows.

Generated `internal/db/` code is produced by `sqlc generate`; `gen/` proto code by `make proto`. Neither is hand-edited.
