# Contract Delta: plan.v1.PlanService — Untimed Polish

**No proto/message change.** Request and response shapes for `AddPlanTask`, `MovePlanEntry`, and `ListPlanEntries` are exactly as defined in feature 027. This document records the **behavioral** delta only: a new rejection that callers must expect.

## New behavior: reject duplicate untimed entries

### `AddPlanTask`

`AddPlanTaskRequest { day, task_id, start_minute (optional), duration_minute }`

- **When `start_minute` is omitted (untimed add)**: if the target `day` already contains an untimed entry (`start_minute` unset) linked to the same `task_id`, the server MUST reject the request.
  - Error: `CodeFailedPrecondition`, with a playful, actionable message stating the task is already on that day's plan without a time.
  - No entry is created.
- **When `start_minute` is present (timed add)**: unchanged — multiple timed entries for the same task remain allowed; a timed add is never blocked by an existing untimed entry for the same task.

### `MovePlanEntry`

`MovePlanEntryRequest { day, id, start_minute (optional), duration_minute }`

- **When `start_minute` is omitted (clear start / unschedule)**: if the target `day` already contains a **different** untimed entry (id ≠ this request's `id`) linked to the same `task_id` as the entry being moved, the server MUST reject the request.
  - Error: `CodeFailedPrecondition`, same message family as above.
  - The entry keeps its current start time (no mutation).
- **When `start_minute` is present (schedule / reschedule)**: unchanged — existing overlap rules only.

### `ListPlanEntries`

Unchanged. Untimed entries continue to be returned with unset `start_minute`, ordered untimed-first then by creation order.

## Invariant guaranteed to clients

> For any `(day, task_id)`, the plan holds at most one untimed entry, and any number of timed entries.

## Concurrency

The check is performed inside the existing serializable transaction after locking the day's entries, so two simultaneous untimed adds for the same `(day, task_id)` cannot both succeed.

## Backward compatibility

- Clients that only ever created **timed** entries, or a single untimed entry per task, see no behavior change.
- The only new outcome is a `FailedPrecondition` where a second untimed duplicate would previously have been (incorrectly) created. CLI surfaces it as an error message; the TUI surfaces it in the status bar (replacing the would-be success notice).
