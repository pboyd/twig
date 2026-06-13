# Feature Specification: Auto-Assign Enhancements

**Feature Branch**: `052-auto-assign-enhancements`

**Created**: 2026-06-13

**Status**: Draft

**Input**: User description: "auto-assign enhancements. The auto-assign feature in the TUI's planning tab needs a couple enhancements: (1) Round the start time to 15 minutes, even if that's in the past — e.g. if it's 9:03 AM and the task will fit in the slot beginning at 9:00 AM, then that's where it should go. (2) Auto-assigning an already assigned slot should bump it into the next available slot."

## Clarifications

### Session 2026-06-13

- Q: When a slot opens right after an existing entry (e.g. a meeting ends at 9:07), does 15-minute rounding apply there too, or only to the day's starting floor? → A: Every auto-assigned start snaps to a 15-minute boundary (:00/:15/:30/:45). A slot bounded below by an existing entry rounds **up** to the next boundary so it never overlaps; the day's starting floor rounds **down** (and may land before the current time).
- Q: When the highlighted task is already sitting in its earliest-fitting slot, how far does pressing `a` move it? → A: It skips past the entry that closes its current free stretch and lands in the next distinct free gap of the day (not merely the next 15-minute boundary within the same gap).

## Overview

This feature extends the existing auto-schedule action (`a`) in the TUI planning tab. Today, pressing `a` on a highlighted task moves it to the earliest free stretch of the day — no earlier than 8 AM, and no earlier than the current minute when planning today — that is long enough to hold the task. (See feature `043-auto-schedule-task`.) Two refinements make the placement times tidier and make the keypress always do something useful:

1. **Round placements to 15-minute boundaries** so the day stays on a clean quarter-hour grid.
2. **Bump an already-placed task forward** so that pressing `a` on a task that is already in its earliest slot advances it rather than doing nothing.

All other established behavior — events are ignored, tasks with no duration assume a 30-minute block, the moved task keeps the highlight, and a "no room" notice appears when nothing fits — is unchanged.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Placements land on tidy quarter-hour times (Priority: P1)

A user is planning their day in the TUI planning tab. It is mid-morning and they highlight a task and press `a`. Rather than receiving an odd start time tied to the exact current minute (e.g. 9:03), the task is placed at the nearest clean quarter-hour at or before that point that still fits — 9:00 — so the plan reads as a tidy grid of quarter-hour blocks.

**Why this priority**: This is the headline refinement. Clean quarter-hour start times are what make an auto-built plan look intentional instead of jittery, and the rounding rule touches every single auto-assignment, so it delivers the most pervasive value.

**Independent Test**: On today's plan at a time like 9:03 with an open morning, highlight a task and press `a`; confirm it is placed at 9:00 (a quarter-hour boundary), not 9:03.

**Acceptance Scenarios**:

1. **Given** today's plan with an open morning and the current time of 9:03, **When** the user highlights a 30-minute task and presses `a`, **Then** the task is placed starting at 9:00 — the quarter-hour boundary at or just before the current time — even though 9:00 is three minutes in the past.
2. **Given** today's plan and a current time that is already exactly on a quarter-hour (e.g. 9:15), **When** the user presses `a` with an open morning, **Then** the task is placed starting at 9:15 (no change, since it is already on a boundary).
3. **Given** a day with an existing entry ending at 9:07 and a highlighted task that fits in the stretch after it, **When** the user presses `a`, **Then** the task is placed starting at 9:15 — the next quarter-hour boundary after the entry ends — so it does not overlap the entry.
4. **Given** a future day (not today) with an empty morning, **When** the user presses `a`, **Then** the task is placed starting at 8:00 (the 8 AM floor, already a quarter-hour boundary).

---

### User Story 2 - Pressing `a` on an already-placed task moves it on (Priority: P2)

A user has a task that is already sitting in the earliest spot it can occupy. They press `a` again — perhaps expecting it to relocate, or simply because they want it out of the current stretch. Instead of the keypress doing nothing, the task is bumped past the entry that closes off its current free stretch and dropped into the next free gap of the day.

**Why this priority**: Today this case is a silent no-op, which feels broken — the user presses a key and nothing happens. Making `a` always advance an already-placed task turns it into a reliable "move this along" control, but it builds on and is secondary to the core placement behavior.

**Independent Test**: Place a task in its earliest-fitting slot with a later free gap also available, highlight it, press `a`, and confirm it jumps forward into that later gap rather than staying put.

**Acceptance Scenarios**:

1. **Given** a 30-minute task placed at 8:00 (its earliest-fitting slot), an entry from 9:00–10:00, and free time after 10:00, **When** the user presses `a`, **Then** the task is bumped to 10:00 — the start of the next free gap after the entry that closed its stretch.
2. **Given** a task that is **not** in its earliest-fitting slot but a clean earlier slot exists (e.g. placed at 14:00 with an open 8:00 slot), **When** the user presses `a`, **Then** the task moves to the earlier 8:00 slot (the existing re-home behavior — the bump only applies when the task is already in its earliest slot).
3. **Given** a task already placed in the last/only free stretch of an otherwise full day, with no later gap available, **When** the user presses `a`, **Then** the task stays where it is and a "no room" notice is shown, because there is no next gap to bump into.

---

### User Story 3 - Existing guarantees still hold (Priority: P3)

A user relies on the auto-schedule behavior they already know: events are left untouched, packed days produce a clear notice, and the highlight follows the moved task. These behaviors continue to work exactly as before alongside the two new refinements.

**Why this priority**: This is a regression guardrail rather than new value. It matters for correctness and user trust but introduces no new capability.

**Independent Test**: Re-run the existing auto-schedule scenarios (event highlighted, packed day, highlight-follows-task) and confirm unchanged outcomes.

**Acceptance Scenarios**:

1. **Given** an event (not a task) is highlighted, **When** the user presses `a`, **Then** nothing moves and the day is unchanged.
2. **Given** a day with no free stretch long enough to hold the highlighted task, **When** the user presses `a`, **Then** the task is not moved and a "no room" notice is shown.
3. **Given** a successful auto-assignment, **When** the task moves to its new time, **Then** the planning-tab highlight follows the task to its new position.

---

### Edge Cases

- **Floor rounding never crosses 8 AM**: On today, the earliest-allowed start is the later of 8 AM and the current time; rounding that floor down to a quarter-hour never produces a start before 8 AM. (E.g. at 8:07 the floor rounds down to 8:00, not 7:45.)
- **Task currently on a non-boundary time**: A task already placed at an off-grid time (e.g. 9:07, perhaps set manually) and highlighted for `a` is treated as not being in its rounded earliest slot, so it is re-placed onto a quarter-hour boundary rather than bumped.
- **Bump when the current stretch reaches end of day**: If the task already sits in a free stretch that extends to end of day with no following entry, there is no next gap; the task stays and a "no room" notice is shown.
- **Zero-duration task**: A task with no duration continues to assume a 30-minute block for fit-finding (existing behavior) and its placement is rounded to a quarter-hour like any other.
- **Empty plan**: Pressing `a` on a day with no entries at all is a no-op (existing behavior).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: When auto-assigning a task, the system MUST place it so its start time falls exactly on a 15-minute boundary (:00, :15, :30, or :45).
- **FR-002**: The earliest-allowed start for the day (8 AM, or the later of 8 AM and the current time when planning today) MUST be rounded **down** to the nearest 15-minute boundary, even when the resulting boundary is earlier than the current time.
- **FR-003**: The rounded-down floor MUST NOT produce a start earlier than 8 AM.
- **FR-004**: A candidate slot whose earliest possible start is bounded by a preceding entry MUST round its start **up** to the next 15-minute boundary at or after that entry ends, so the placed task never overlaps the preceding entry.
- **FR-005**: The system MUST choose the earliest 15-minute-aligned start, at or after the rounded floor, where the task's full duration fits without overlapping any other timed entry — preserving the existing "earliest free slot" placement behavior on the quarter-hour grid.
- **FR-006**: When the highlighted task is already positioned at its earliest-fitting 15-minute-aligned slot, pressing `a` MUST bump it forward into the next distinct free gap of the day rather than leaving it in place.
- **FR-007**: The bump in FR-006 MUST skip entirely past the entry that closes the task's current free stretch and place the task at the earliest 15-minute-aligned start within the next free gap where it fits.
- **FR-008**: When there is no later free gap large enough to hold the task being bumped, the system MUST leave the task in place and show a "no room" notice.
- **FR-009**: When the highlighted task is NOT already in its earliest-fitting slot, pressing `a` MUST place it in the earliest-fitting slot (which may be earlier than its current position), as in existing behavior — the bump applies only to the already-in-earliest-slot case.
- **FR-010**: The system MUST continue to ignore events (non-task entries) when `a` is pressed, leaving the day unchanged.
- **FR-011**: The system MUST continue to treat a task with no/zero duration as a 30-minute block for fit-finding without changing the task's stored duration, and round its placement to a 15-minute boundary like any other task.
- **FR-012**: The system MUST continue to keep the planning-tab highlight on the moved task after a successful auto-assignment.
- **FR-013**: The system MUST continue to show a "no room" notice and leave the day unchanged when no qualifying slot exists.

### Key Entities

- **Plan entry**: A scheduled item on a day, either a *task* (eligible for auto-assign) or an *event* (ignored by auto-assign). Relevant attributes: an optional start time, a duration, and whether it is a task or event.
- **Free gap**: A contiguous stretch of a day not occupied by any timed entry, bounded below by either the day floor or a preceding entry's end, and above by the next entry's start or end of day. Auto-assign searches these gaps for a fitting, quarter-hour-aligned start.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of tasks placed by auto-assign have a start time on a 15-minute boundary.
- **SC-002**: When planning today at a non-quarter-hour time with an open day, auto-assign places the task at the quarter-hour at or before the current time (e.g. 9:03 → 9:00), confirmed in 100% of such cases.
- **SC-003**: Pressing `a` on a task already in its earliest-fitting slot moves the task to a later free gap (or surfaces a "no room" notice when none exists) in 100% of cases — never a silent no-op.
- **SC-004**: All previously passing auto-schedule scenarios (events ignored, packed-day notice, highlight follows task, re-home to an earlier slot) continue to pass with no regressions.

## Assumptions

- The 15-minute boundary is fixed at quarter-hour marks (:00, :15, :30, :45); the rounding interval is not user-configurable.
- "Already in its earliest-fitting slot" means the task's current start equals the start the auto-assign rule would otherwise compute on the 15-minute grid; a task sitting on an off-grid start time is therefore treated as not-in-place and is re-placed onto the grid.
- The day floor remains 8 AM as established in feature `043-auto-schedule-task`; this feature only changes how that floor (and slot starts) are rounded, not the floor value itself.
- "Next available slot" for the bump means the next distinct free gap (past the entry that closes the current stretch), not merely the next 15-minute increment inside the same gap.
- This feature only affects the TUI planning tab's `a` action; CLI and web planning surfaces are out of scope.
