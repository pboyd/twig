# Phase 1: Data Model — Plan Calendar Grid View

This feature is overwhelmingly a presentation change. The only persisted/transported data model change is one new boolean field on the existing `PlanEntry` proto.

## Modified entity: `PlanEntry` (wire)

| Field | Type | Number | Origin | Notes |
|-------|------|--------|--------|-------|
| `day` | string (YYYY-MM-DD) | 1 | existing | unchanged |
| `id` | int32 | 2 | existing | unchanged |
| `task_id` | int64 | 3 | existing | unchanged; `0` means "no task" |
| `name` | string | 4 | existing | unchanged |
| `start_minute` | int32 | 5 | existing | unchanged |
| `duration_minute` | int32 | 6 | existing | unchanged |
| **`completed`** | **bool** | **7** | **new** | **`true` iff `task_id != 0` and the referenced task has a non-null `completed_at`. `false` for events and for tasks not yet completed.** |

### Server-side population

`ListPlanEntries` returns entries from a query that LEFT JOINs `tasks` on `plan_entries.task_id = tasks.id`. The new column projection is:

```sql
(plan_entries.task_id IS NOT NULL
 AND plan_entries.task_id <> 0
 AND tasks.completed_at IS NOT NULL) AS completed
```

(Exact SQL lives in `db/queries/plan.sql`; this document is the wire-level contract.)

### Backwards compatibility

- Old clients calling a new server: ignore the unknown field. No behavioral change for them — they were not rendering completion state anyway.
- New clients calling an old server: receive default `false`. They will render every entry as not-completed — degraded but correct.

## Render-only state (not persisted)

The calendar renderer derives the following at render time; none of it is stored or transported:

| Derived value | Source | Rule |
|---|---|---|
| `windowStartMin` | min over entries of `start_minute` | `min(8*60, floor(minStart/60)*60)`; defaults to `8*60` if no entries |
| `windowEndMin` | max over entries of `start_minute + duration_minute` | `max(17*60, ceil(maxEnd/60)*60)`; defaults to `17*60` if no entries |
| `snappedStartMin` | per entry | `round_nearest_15(start_minute)` |
| `snappedEndMin` | per entry | `round_nearest_15(start_minute + duration_minute)`; clamped to `>= snappedStartMin + 15` to guarantee at least one row |
| `width` | terminal | from `x/term.GetSize(stdout)`; floor 60, default 80 when not a TTY |
| `nowRow` | clock | only if `day == today`; the 15-minute row index of the current local time within the visible window |

## Entities NOT changed

- `tasks` table — already has the `completed_at` column required for the join. No schema migration.
- Plan-related RPCs other than `ListPlanEntries` — unchanged; `completed` is irrelevant for add/move/remove/rename.
