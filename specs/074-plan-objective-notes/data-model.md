# Phase 1 Data Model: Plan Objectives and Notes

**Feature**: 074-plan-objective-notes | **Date**: 2026-08-06

## Entity: Plan day

A single user's single calendar day, as a record in its own right. Until now a "plan day"
existed only as the set of `plan_entries` rows sharing a `(user_id, day)`. This feature
gives the day two attributes of its own, which exist whether or not the day has entries
(spec FR-001).

### Table: `plan_days`

Migration `000015_plan_days.up.sql`:

```sql
CREATE TABLE plan_days (
    user_id   BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day       DATE         NOT NULL,
    objective VARCHAR(255) NOT NULL DEFAULT '',
    notes     TEXT         NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, day)
);
```

Down migration drops the table.

| Column | Type | Notes |
|---|---|---|
| `user_id` | `BIGINT NOT NULL` | FK to `users(id)`, `ON DELETE CASCADE` — matches `plan_entries` |
| `day` | `DATE NOT NULL` | The calendar day, user-local |
| `objective` | `VARCHAR(255) NOT NULL DEFAULT ''` | Markdown source. `''` means not set |
| `notes` | `TEXT NOT NULL DEFAULT ''` | Markdown source. `''` means not set |

No index beyond the primary key: every access is a point lookup by `(user_id, day)`.

No foreign key to `plan_entries`, and no requirement that entries exist. The two tables
are independent views of the same `(user_id, day)` key.

### Validation rules

| Rule | Source | Enforced |
|---|---|---|
| `day` must parse as `YYYY-MM-DD` | FR-003 | Handler, via the existing `parseDay` |
| Objective and notes are trimmed of surrounding whitespace before storage | FR-005 | Handler |
| A whitespace-only value stores as `''` and reads as absent | FR-005 | Handler + all readers |
| Objective is at most 255 runes after trimming | D1 | Handler, `CodeInvalidArgument` with a warm, actionable message |
| Notes have no length limit | D1 | — |
| Both fields store verbatim markdown source — no reformatting, no rendering | FR-004 | Handler |
| A day row is created on first write and never deleted | D1 | `INSERT … ON CONFLICT DO UPDATE` |

### Lifecycle

There are no state transitions. Both fields move freely between "set" and "not set":

```
(no row)  --SetPlanObjective/SetPlanNotes-->  (row, field = value)
(row, field = value)  --Set… with "" or whitespace-->  (row, field = '')
```

Reads of a day with no row return `objective = ""`, `notes = ""` — identical to a row whose
fields are empty. No caller distinguishes the two (FR-005).

Deleting a user cascades the row away. Deleting plan entries does not.

### Queries (`services/twig/db/queries/plan.sql`)

```sql
-- name: GetPlanDay :one
SELECT * FROM plan_days WHERE user_id = $1 AND day = $2;

-- name: UpsertPlanDayObjective :one
INSERT INTO plan_days (user_id, day, objective) VALUES ($1, $2, $3)
ON CONFLICT (user_id, day) DO UPDATE SET objective = EXCLUDED.objective
RETURNING *;

-- name: UpsertPlanDayNotes :one
INSERT INTO plan_days (user_id, day, notes) VALUES ($1, $2, $3)
ON CONFLICT (user_id, day) DO UPDATE SET notes = EXCLUDED.notes
RETURNING *;
```

`GetPlanDay` returns `pgx.ErrNoRows` for an untouched day; callers translate that to the
zero-value `PlanDay` rather than an error. Each upsert touches exactly one column, so the
two setters can never clobber each other's field.

Regenerate with `sqlc generate` from `services/twig/`; do not hand-edit `internal/db/`.

## Wire representation

```proto
message PlanDay {
  string day = 1;        // YYYY-MM-DD
  string objective = 2;  // markdown source; "" means not set
  string notes = 3;      // markdown source; "" means not set
}
```

Carried on `ListPlanEntriesResponse.day` (always populated) and returned by both setters.
See `contracts/plan-day.md`.

## Client-side state

**TUI** — `planState` (`internal/tui/model.go:115`) gains:

| Field | Type | Purpose |
|---|---|---|
| `objective` | `string` | The displayed day's objective, as loaded |
| `notes` | `string` | The displayed day's notes, as loaded |
| `objectiveInput` | `textinput.Model` | Draft while `mode == planObjectiveEdit` |
| `notesInput` | `textarea.Model` | Draft while `mode == planNotesEdit` |

`planMode` gains `planObjectiveEdit` and `planNotesEdit`. `planEntriesMsg` gains
`day *planv1.PlanDay`, applied by `handlePlanEntriesMsg` under the existing stale-day rules,
and never over an open editor's draft.

Both strings are replaced wholesale on every load for the displayed day; there is no
client-side merge and no local cache across days. Navigating to another day loads that
day's values with its entries (FR-022).

**CLI** — stateless. Reads come from `ListPlanEntriesResponse.day.objective`; writes go
straight to `SetPlanObjective`.

## Out of scope

The web app is untouched (FR-030). `plan_days` is additive: no existing table, query, or
message changes shape, so every current code path behaves exactly as it does today on days
where neither field is set.
