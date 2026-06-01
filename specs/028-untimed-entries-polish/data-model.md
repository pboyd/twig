# Data Model: Untimed Plan Entries — Polish & Bugfixes

**No schema change.** This feature adds a behavioral invariant and a TUI-only display-state field. The `plan_entries` table and the `plan.v1` proto messages are unchanged from feature 027.

## Entity: Plan Entry (unchanged shape, new invariant)

`plan_entries` — composite PK `(user_id, day, id)`; relevant columns: `task_id` (nullable), `start_minute` (nullable `int2`, NULL ⇒ untimed), `duration_minute`.

### New invariant (enforced, not schema-constrained)

> For a given `(user_id, day, task_id)`, **at most one** entry may have `start_minute IS NULL`.

- Applies only to **task-linked** entries (untimed events are already impossible — events require a start).
- **Timed** entries are unconstrained: any number of `start_minute IS NOT NULL` entries may exist for the same `(day, task_id)`, alongside the single untimed one.
- Enforced in `internal/handler/plan.go` within the serializable transaction (see contracts), **not** by a DB constraint. The table definition is untouched.

### Enforcement points

| Operation | Path | Check |
|-----------|------|-------|
| Create untimed | `AddPlanTask` with `start_minute == nil` | Reject if the day already has an untimed entry for this `task_id`. |
| Clear start (timed → untimed) | `MovePlanEntry` with `start_minute == nil` | Reject if the day already has a **different** untimed entry for this entry's `task_id`. |
| Create/keep timed | `AddPlanTask` / `MovePlanEntry` with `start_minute != nil` | No new check (existing overlap rules only). |

## TUI display state (in-memory only)

| Field | Location | Purpose |
|-------|----------|---------|
| `notice string` | `internal/tui/model.go` `Model` | Transient, non-error status message (e.g. send confirmation). Rendered by `renderStatus` when set and no error is active. Cleared/replaced on the next action. Not persisted. |

## Ordering & display

Unchanged from 027: untimed entries sort before timed entries, creation order within the untimed group. The rendering fix changes only the *box geometry* of untimed entries (to match the grid), not their order, count, or data.
