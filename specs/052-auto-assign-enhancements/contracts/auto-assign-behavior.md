# Behavioral Contract: Auto-Assign Enhancements

This feature changes no network/proto contract. The relevant contracts are the internal Go functions the TUI consumes and the `a`-key handler behavior. They are documented here to satisfy the constitution's "contract before implementation" gate.

## Consumed network contract (unchanged)

`PlanService.MovePlanEntry` (via `internal/tui/plan_update.go: movePlanCmd`) — already exists; called with the computed boundary-aligned `start` minute and the entry's existing duration. No change to the request/response shape.

## Function: `AutoScheduleSlot` (revised — `internal/cli/plan_grid.go`)

```go
func AutoScheduleSlot(timed []*planv1.PlanEntry, durationMin, floorMin int, excludeID int32) (startMin int, ok bool)
```

**Unchanged**: signature; builds merged obstacle intervals excluding `excludeID`; treats `durationMin <= 0` as 30; scans from `floorMin` to 1440; returns `ok=false` when nothing fits.

**New guarantee**: every candidate gap start is aligned **up** to the next 15-minute boundary (`ceil15`) before the fit test, so any returned `startMin` satisfies `startMin % 15 == 0` and `[startMin, startMin+needed)` does not overlap any obstacle.

**Caller responsibility**: `floorMin` is expected to already be a 15-minute boundary for the "earliest allowed start" semantics (the TUI handler round-rounds the day floor **down** before calling). Up-rounding an already-boundary `floorMin` is a no-op.

| Given (excluding the highlighted task) | floorMin | duration | Returns |
|---|---|---|---|
| empty day | 480 | 30 | 480, true |
| entry 480–540 (08:00–09:00) | 480 | 30 | 540, true |
| entry 480–547 (08:00–09:07, off-boundary end) | 480 | 30 | 555 (09:15, rounded up), true |
| entries 480–540 and 555–600 | 480 | 30 | 600, true (15-min gap too small) |
| day full to 1440 | 480 | 30 | _, false |

## Function: `NextGapFloor` (new — `internal/cli/plan_grid.go`)

```go
// NextGapFloor returns the end minute of the earliest timed entry that begins
// at or after fromMin — the lower bound of the next free gap after the stretch
// containing fromMin. ok is false when no entry begins at/after fromMin.
func NextGapFloor(timed []*planv1.PlanEntry, fromMin int, excludeID int32) (floorMin int, ok bool)
```

Builds the same merged obstacle intervals (excluding `excludeID`) and returns the `end` of the first interval whose `start >= fromMin`.

| Given (excluding the task) | fromMin | Returns |
|---|---|---|
| entry 540–600 (09:00–10:00) | 480 | 600, true |
| entries 540–600 and 660–720 | 480 | 600, true (first one) |
| entry 420–480 only (before fromMin) | 480 | _, false |
| no entries | 480 | _, false |

## Handler decision table (`a` key — `internal/tui/update.go`)

Let `cur = *entry.StartMinute` (when scheduled). `floor = floor15(max(480, todayNowMin or 480))`. `start, ok = AutoScheduleSlot(timed, dur, floor, entry.Id)`.

| Condition | Action |
|---|---|
| highlighted entry is an event (`TaskId == 0`) | no move; event notice (unchanged) |
| `!ok` | no move; "Day's packed…" notice (unchanged) |
| task unscheduled, or `cur != start` | `movePlanCmd(..., start, ...)` (place / re-home, possibly earlier) |
| task scheduled and `cur == start` (already in earliest slot) | **bump**: `bumpFloor, ok2 = NextGapFloor(timed, cur, entry.Id)`; if `ok2`, `start2, ok3 = AutoScheduleSlot(timed, dur, bumpFloor, entry.Id)` and `movePlanCmd(..., start2, ...)` when `ok3`; otherwise "Day's packed…" notice and no move |
| after any successful move | highlight follows the task to its new position (unchanged) |

## Invariants verified by tests

- Returned start is always a 15-minute boundary and ≥ 480.
- Bump always yields a start strictly later than `cur`, or a "no room" notice — never a silent no-op.
- Re-home-to-earlier (cur ≠ start, start < cur) still moves the task earlier.
- Events and packed-day handling are byte-for-byte the same as feature 043.
