# Phase 1 Data Model: Auto-Assign Enhancements

No new persisted entities, database tables, or proto messages. This feature operates only on the in-memory plan-entry list already loaded in the TUI. The model below documents the shapes and derived concepts the slot math reasons over.

## Existing types (unchanged)

### PlanEntry (`planv1.PlanEntry`, already defined)

Relevant fields used by auto-assign:

| Field | Meaning for auto-assign |
|-------|-------------------------|
| `Id` (int32) | Identifies the entry; used as `excludeID` so a task does not block itself. |
| `TaskId` (int32) | `0` ⇒ event (ignored by `a`); non-zero ⇒ task (eligible). |
| `StartMinute` (*int32, optional) | Minute-of-day start. `nil` ⇒ unscheduled. |
| `DurationMinute` (int32) | Length in minutes; `0` ⇒ treated as a 30-minute block for fit-finding only (stored value unchanged). |

No fields are added, removed, or repurposed.

## Derived concepts (computation only — not stored)

### Minute-of-day

All times are integer minutes from midnight, `0..1440`. 8 AM = `480`; end of day = `1440`.

### 15-minute boundary

A minute `m` where `m % 15 == 0` (`:00`, `:15`, `:30`, `:45`). Operations:

- **Round down**: `floor15(m) = (m / 15) * 15` — applied to the day floor by the TUI handler.
- **Round up**: `ceil15(m) = ((m + 14) / 15) * 15` — applied to every candidate gap start by `AutoScheduleSlot`.

### Obstacle interval

A `[start, end)` pair derived from each timed entry except the excluded one; overlapping/adjacent intervals are merged. `end = StartMinute + DurationMinute`.

### Free gap

A contiguous span not covered by any obstacle, bounded below by the (down-rounded) day floor or a preceding obstacle's end, and above by the next obstacle's start or `1440`. A task fits a gap when `ceil15(gapStart) + neededDuration <= gapEnd`.

### Bump floor

For the bump path: the `end` of the first obstacle whose `start >= currentStart`. This is the lower bound of the next distinct free gap. Undefined (⇒ "no room") when no obstacle begins at/after `currentStart`.

## Validation / invariants

- Auto-assigned `StartMinute` is always a 15-minute boundary (FR-001).
- Auto-assigned `StartMinute >= 480` (never before 8 AM, FR-003).
- Placed `[start, start+duration)` never overlaps another timed entry (FR-004/FR-005).
- A task with `StartMinute == nil` or an off-boundary start is treated as not-in-place and re-placed onto the grid (spec Edge Cases / Assumptions).

## State transitions

A single keypress moves a task between at most two states; no intermediate persisted state:

```
unscheduled (StartMinute == nil)
        │  press `a`, slot found
        ▼
scheduled at a 15-min boundary  ──press `a` again (already earliest)──▶  scheduled at next gap's boundary
        │                                                                 │ (no later gap)
        │ no slot fits                                                    ▼
        ▼                                                          stays put + "no room" notice
   stays put + "no room" notice
```
