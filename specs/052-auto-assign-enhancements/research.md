# Phase 0 Research: Auto-Assign Enhancements

All technical context was resolvable from the existing codebase (feature 043). No external research or NEEDS CLARIFICATION items remained after the spec's clarification session.

## Existing behavior (baseline)

The `a` key handler lives in `internal/tui/update.go` (`case key.Matches(msg, m.keys.PlanAutoSchedule)`):

1. Skips when the highlighted entry is an event (`entry.TaskId == 0`).
2. Computes `floorMin = 480` (8 AM); when planning today and the current minute exceeds it, `floorMin = nowMin` (the **exact** current minute).
3. Calls `cli.AutoScheduleSlot(timed, durationMin, floorMin, excludeID)` (`internal/cli/plan_grid.go:662`) to find the earliest free slot ≥ floor that fits.
4. If `!ok` → "Day's packed — no room left to squeeze this one in." notice.
5. If the task is already at the returned `startMin` → **no-op** (`return m, nil`).
6. Otherwise issues `movePlanCmd(...)` → `PlanService.MovePlanEntry`.

`AutoScheduleSlot` builds merged obstacle intervals (excluding `excludeID`), then scans free intervals from `floorMin` to 1440, returning the first gap start that fits the (≥30-min) duration.

## Decision 1 — Where the 15-minute rounding lives

**Decision**: Split the rounding by ownership of information.

- The **TUI handler** rounds the day floor **down** to the nearest 15-minute boundary: `floorMin = (floorMin / 15) * 15`. Because `floorMin = max(480, nowMin)` and 480 is itself a boundary, the result is never below 8 AM (FR-003). This is where the "even if in the past" rule belongs because the handler is the only place that knows `now`.
- `AutoScheduleSlot` rounds every **candidate gap start up** to the next boundary before testing fit: `gapStart = ceilTo15(freeStart)`. For obstacle-derived starts this prevents overlap (FR-004); for the already-down-rounded floor it is a no-op.

**Rationale**: Down-rounding only ever applies to the soft day floor (nothing sits below it to overlap). Up-rounding applies to hard lower bounds (a preceding entry's end). Keeping each rule where its precondition is known avoids passing a "direction" flag and keeps both functions simple (Principle I). A boundary floor passed into an up-rounding scan is idempotent, so composition is clean.

**Alternatives considered**:
- *Round everything down inside `AutoScheduleSlot`*: rejected — down-rounding an obstacle-derived start would push the task into the preceding entry (overlap).
- *Round only the floor, leave post-entry starts unrounded*: rejected by the spec clarification (every start must be boundary-aligned).
- *Add a `roundDir` parameter to `AutoScheduleSlot`*: rejected as unnecessary complexity; the split above needs no new parameter.

## Decision 2 — How the bump finds the next gap

**Decision**: Add a pure helper next to `AutoScheduleSlot`:

```go
// NextGapFloor returns the end minute of the earliest timed entry that begins
// at or after fromMin — i.e. the start of the next free gap after the stretch
// containing fromMin. ok is false when no entry begins at/after fromMin (the
// current stretch runs to end of day, so there is no next gap).
func NextGapFloor(timed []*planv1.PlanEntry, fromMin int, excludeID int32) (floorMin int, ok bool)
```

The handler's bump path: `bumpFloor, ok := cli.NextGapFloor(timed, currentStart, entry.Id)`; if `ok`, call `AutoScheduleSlot(timed, dur, bumpFloor, entry.Id)` (which up-rounds `bumpFloor` to a boundary and finds the next fitting slot); if `!ok`, show the existing "no room" notice and leave the task in place.

**Rationale**: Reuses the merged-interval geometry already proven in `AutoScheduleSlot` and reuses `AutoScheduleSlot` itself for the actual placement, so the bump is "skip past one obstacle, then run the normal search." This satisfies FR-006/FR-007/FR-008 with a single small, independently testable helper (Principle I).

**Alternatives considered**:
- *Bump to the next 15-minute increment within the same gap (`floor = currentStart + 15`)*: rejected by the spec clarification (bump goes to the next **distinct** gap, not the next increment).
- *Inline the next-obstacle scan in the handler*: rejected — keeping it as a pure `cli` helper makes it unit-testable without a Bubble Tea model and matches where `AutoScheduleSlot` already lives.

## Decision 3 — When the bump triggers

**Decision**: The bump replaces the existing no-op branch and triggers **only** when the highlighted task already has a start time equal to the earliest-fitting slot just computed (`entry.StartMinute != nil && *entry.StartMinute == startMin`). Every other case keeps the existing earliest-fit move.

**Rationale**: This preserves the established re-home-to-an-earlier-slot behavior (feature 043, US2): if a cleaner earlier slot exists, the earliest-fitting slot differs from the current position and the task simply moves there — no bump. The bump only addresses the previously useless no-op (FR-009).

**Alternatives considered**:
- *Always bump forward from the current position*: rejected — would break the documented "move to an earlier open slot" behavior.

## Decision 4 — Test impact

**Decision**: Update the two existing tests whose expectations encode the old behavior, and add new cases.

- `internal/tui/plan_update_test.go`:
  - `TestAutoSchedule_TodayFloor_UsesCurrentMinute` — now expects the floor rounded **down** to a boundary instead of the exact current minute. Rename/adjust to `..._RoundsFloorDownTo15`.
  - `TestAutoSchedule_AlreadyInPlace_NoOp` — the no-op contract is replaced; convert into a bump test (task already in earliest slot is bumped to the next gap), plus a new "no later gap ⇒ no-room notice" case.
- `internal/cli/plan_grid_test.go`: existing `AutoScheduleSlot` tests use boundary-aligned minutes (480/540/600) so the internal up-rounding is a no-op for them and they keep passing. Add cases for an obstacle ending off-boundary (start rounds up) and for `NextGapFloor`.

**Rationale**: Minimal churn; only assertions tied to the two changed behaviors move. Other regression tests (events ignored, packed day, highlight follows) are unchanged (FR-010/FR-012/FR-013, US3).
