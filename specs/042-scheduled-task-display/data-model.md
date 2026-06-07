# Phase 1 Data Model: Show Scheduled Days in Task Details

This feature is **read-only**. It introduces no new persisted entities and no schema migration. It adds one read query over the existing `plan_entries` table and one in-memory client structure.

## Existing storage (unchanged)

### `plan_entries` (existing table)

| Column | Type | Notes |
|---|---|---|
| `user_id` | bigint | Owner; all queries scoped to the authenticated caller. |
| `day` | date | The scheduled day (local-midnight semantics). Rendered as `YYYY-MM-DD`. |
| `id` | int | Per-day sequential entry id. |
| `task_id` | bigint NULL | Links the entry to a task. `NULL` for event entries. |
| `name` | text NULL | Display override; irrelevant to this feature. |
| `start_minute` | smallint NULL | `NULL` = untimed. **Irrelevant** — an untimed entry still counts as scheduled for its day. |
| `duration_minute` | smallint | Irrelevant to this feature. |

A task is **scheduled for a day** iff there exists a `plan_entries` row with that `task_id` (non-null) and `day`, regardless of `start_minute`.

## New read query

### `ListScheduledDaysForTasks` (sqlc, `services/twig/db/queries/plan.sql`)

```sql
-- name: ListScheduledDaysForTasks :many
SELECT DISTINCT task_id, day
FROM plan_entries
WHERE user_id = $1
  AND task_id IS NOT NULL
  AND day >= $2
ORDER BY task_id, day;
```

- `$1` = authenticated user id (from context, never the wire).
- `$2` = `from_day` (the caller's local current day as a `date`).
- `DISTINCT` collapses multiple entries for the same task on the same day → satisfies **FR-008** (each day at most once).
- `ORDER BY task_id, day` gives ascending days per task → satisfies **FR-004** without client-side sorting.
- `day >= $2` enforces today-or-future → satisfies **FR-005 / FR-006**.

## Derived transport entity

### `ScheduledDay` (proto message — see `contracts/`)

| Field | Type | Notes |
|---|---|---|
| `task_id` | int64 | The scheduled task. |
| `day` | string | `YYYY-MM-DD`. |

The response is a flat, ascending list of `(task_id, day)` pairs. The client groups by `task_id`.

## Client in-memory structure (TUI `Model`)

| Field | Type | Notes |
|---|---|---|
| `scheduledDays` | `map[int64][]string` | task id → ascending `YYYY-MM-DD` days (today-or-future). Built from the response; missing key or empty slice ⇒ render no `Scheduled for:` line. |

**Lifecycle**: populated by a `listScheduledDaysCmd` issued at init, on `Ctrl-R`, on switching to the Tasks tab, and after a `ctrl+p` send-to-plan. Read synchronously by `renderDetails`. No persistence.

## Validation rules

- `from_day` MUST be a valid `YYYY-MM-DD`; the handler rejects malformed input with `InvalidArgument` (reuse the existing `parseDay`).
- Results MUST be isolated to the authenticated user (enforced by `user_id = $1`).
