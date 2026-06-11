# Data Model: Activity Report

**Feature**: 048-activity-report | **Date**: 2026-06-11

No database schema changes. All report structures are in-memory, computed on
demand in the new `internal/report` package (root module) from `ListTasks` output
plus one pomodoro count.

## Period

A resolved reporting window.

| Field | Type | Notes |
|---|---|---|
| `From` | local calendar day | First day, inclusive |
| `To` | local calendar day | Last day, inclusive; `To >= From` (else validation error) |
| `Label` | string | Human label ("Yesterday", "Mar 1 – Mar 31", …) |

Derived:
- `StartUTC = time.Date(From, 00:00 local).UTC()`
- `EndUTC = time.Date(To+1 day, 00:00 local).UTC()` — half-open `[StartUTC, EndUTC)`
- `Days = calendar-day span (To - From + 1)`
- `Layout = DayGrouped` if `Days <= 14`, else `AccomplishmentGrouped`

### Presets

| Preset | Resolution (local timezone, weeks Mon–Sun) |
|---|---|
| `recent` (default) | yesterday … today |
| `today` | today … today |
| `yesterday` | yesterday … yesterday |
| `week` | Monday of current week … today |
| `last-week` | Monday … Sunday of previous week |
| `month` | 1st of current month … today |
| `quarter` | 1st day of current calendar quarter … today |
| `year` | Jan 1 of current year … today |
| explicit | `--from` … `--to` (CLI only), `YYYY-MM-DD` |

Validation rules (FR-013): dates must parse as `YYYY-MM-DD`; `to` must not precede
`from`; `--from`/`--to` must both be present and are mutually exclusive with a
preset; failures produce a playful, actionable error and no partial report.

## Entry

One completed task occurrence within the period.

| Field | Type | Notes |
|---|---|---|
| `TaskID` | int64 | From `Task.id` |
| `Name` | string | `Task.name` |
| `CompletedAt` | time.Time | `Task.completed_at`, rendered in local time |
| `ParentName` | string | Immediate parent's name; empty for top-level tasks (FR-002 context) |
| `TopLevelID` | int64 | Root ancestor's id (used by accomplishment grouping) |
| `Depth` | int | Distance from top-level ancestor (indent in accomplishment layout) |

Inclusion rule (FR-011): a task is an Entry iff `completed_at` is set **and**
`StartUTC <= completed_at < EndUTC` at generation time. Un-completed tasks have a
cleared `completed_at` and therefore drop out automatically.

## DayGroup (layout for periods ≤ 14 days)

| Field | Type | Notes |
|---|---|---|
| `Day` | local calendar day | Grouping key from `CompletedAt` in local time (FR-003, SC-005) |
| `Entries` | []Entry | Ordered by `CompletedAt` descending |

Groups ordered most recent day first. Days with no entries are omitted (the
empty state only applies when the whole period has no entries).

## AccomplishmentGroup (layout for periods > 14 days)

| Field | Type | Notes |
|---|---|---|
| `TopLevel` | Entry-like header | The top-level ancestor task (name, completion state) |
| `Finished` | bool | True when the top-level task itself completed within the period |
| `Entries` | []Entry | Completed-in-period descendants, by `Depth` then `CompletedAt` |

Report sections (FR-004, research D4):
1. **Finished** — groups with `Finished = true`, newest top-level completion first.
2. **Progress on ongoing work** — groups whose top-level task is incomplete or
   completed outside the period but which contain in-period entries.

Invariant (SC-004): every Entry appears in exactly one group in exactly one section.

## Totals

| Field | Type | Source |
|---|---|---|
| `TasksCompleted` | int | `len(all entries)` |
| `PomodorosCompleted` | int64 | `CountCompletedPomodoros(StartUTC, EndUTC)` RPC (FR-014) |

Shown in both layouts. The pomodoro total counts every pomodoro completed in the
period regardless of its task's completion state (spec clarification).

## Server-side query (new, no migration)

`CountCompletedPomodorosInRange(user_id, start, end)` over the existing
`pomodoros` table: `complete = TRUE AND end_at >= start AND end_at < end`.
Exposed via the new `CountCompletedPomodoros` RPC (see `contracts/report-rpc.md`).

## State transitions

None persisted. Reports are pure functions of (task list, pomodoro count, period,
clock, local timezone); regenerating after any task change reflects current state.
