# Feature Specification: Auto-Schedule a Task

**Feature Branch**: `043-auto-schedule-task`

**Created**: 2026-06-08

**Status**: Draft

**Input**: User description: "auto-schedule a task. In the TUI planning tab, when the user presses `a` it should move the highlighted task to the next available gap it fits into, but no earlier than 8 AM. Events don't need this feature. The idea is to avoid some tedium by finding the next available slot automatically for the user."

## Clarifications

### Session 2026-06-08

- Q: For a task with no/zero duration, how should auto-schedule choose a slot? → A: Assume a default 30-minute block to find the gap; do not change the task's stored duration.
- Q: When the plan day is today and the current time is already past 8 AM, what is the earliest start auto-schedule may use? → A: The later of 8 AM and the current time (never place into a past time); future days use a flat 8 AM floor.
- Q: After a successful auto-schedule, where does the planning tab highlight go? → A: It follows the moved task to its new position.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Drop a task into the next open slot (Priority: P1)

A user is building their plan for a day in the TUI planning tab. They have a task on that day that does not yet have a time (or that they want to relocate), and rather than mentally scanning the day for a free stretch and typing in an exact start time, they highlight the task and press `a`. The task immediately jumps to the earliest free stretch of the day — no earlier than 8 AM — that is long enough to hold it, taking on a start time there.

**Why this priority**: This is the whole feature. Finding the next free slot by hand is the tedium the feature exists to remove; a single keypress that places the task correctly delivers the entire value on its own.

**Independent Test**: In the planning tab, highlight a task that has a duration and press `a`. Confirm it becomes a timed entry starting at the earliest free time at or after 8 AM where its full duration fits without overlapping any other timed entry.

**Acceptance Scenarios**:

1. **Given** a day whose first timed entry begins at 10:00 and a highlighted 30-minute task with no start time, **When** the user presses `a`, **Then** the task is placed starting at 08:00 (the earliest free time at or after 8 AM that fits its duration).
2. **Given** a day with a timed entry from 08:00–09:00 and a highlighted 30-minute task, **When** the user presses `a`, **Then** the task is placed starting at 09:00 (immediately after the blocking entry).
3. **Given** a day with timed entries at 08:00–09:00 and 09:15–10:00 and a highlighted 30-minute task, **When** the user presses `a`, **Then** the task is placed starting at 10:00 because the 09:00–09:15 gap is too short to hold it.
4. **Given** a completely empty day and a highlighted task of any duration that fits before end of day, **When** the user presses `a`, **Then** the task is placed starting at 08:00.

---

### User Story 2 - Re-home a task that no longer fits where it is (Priority: P2)

A user has a task already placed at a time, but they have since added other entries around it and want it moved to wherever it next fits cleanly. They highlight that task and press `a`. The task is lifted from its current position and placed at the earliest free slot at or after 8 AM that fits it — which may be earlier or later than where it was.

**Why this priority**: Reusing the same single keypress to relocate an already-timed task extends the convenience to mid-planning adjustments. It is valuable but secondary to placing an as-yet-unscheduled task, and it builds directly on the same mechanism.

**Independent Test**: Place a task at 14:00, leave an earlier free slot open at/after 8 AM big enough for it, highlight the task, press `a`, and confirm it moves to that earlier slot.

**Acceptance Scenarios**:

1. **Given** a task currently placed at 14:00 and an open slot at 08:00 large enough to hold it, **When** the user presses `a`, **Then** the task moves to 08:00.
2. **Given** a task being re-homed, **When** the search for a free slot runs, **Then** the task's own current position is treated as free (it does not block itself).
3. **Given** a task whose current placement is already the earliest free slot at or after 8 AM that fits it, **When** the user presses `a`, **Then** the task stays where it is.

---

### User Story 3 - Auto-schedule does not apply to events (Priority: P3)

A user highlights an event (not a task) in the planning tab and presses `a`. Because events represent fixed commitments rather than flexible work, nothing is moved; the day is left exactly as it was.

**Why this priority**: This is a guardrail that keeps the feature from doing something unwanted to fixed commitments. It matters for correctness but is a narrow no-op rule rather than the core value.

**Independent Test**: Highlight an event entry, press `a`, and confirm the event and the rest of the day are unchanged.

**Acceptance Scenarios**:

1. **Given** a highlighted event entry, **When** the user presses `a`, **Then** no entry is moved and the plan is unchanged.
2. **Given** a highlighted event entry, **When** the user presses `a`, **Then** the user is made aware that auto-scheduling applies only to tasks (rather than the keypress silently doing nothing).

---

### Edge Cases

- **No room left in the day**: If no free slot at or after the scheduling floor is long enough to hold the task before the end of the day, the task is left unchanged and the user is told it could not be placed.
- **Today, past 8 AM**: When viewing today's plan and it is already past 8 AM, the floor is the current time, not 8 AM, so the task is never placed into a slot that has already passed. Late enough in the day, this can become the "no room left" case.
- **Task already at the chosen slot**: If the earliest fitting slot is exactly where the task already sits, the result is a no-op (no spurious change is recorded).
- **Existing entries before the floor**: Entries that occupy time before the scheduling floor are ignored as obstacles only for the part before the floor; the search still never starts a placement earlier than the floor.
- **Overlapping existing entries**: When existing timed entries overlap each other, the occupied time is treated as their union when computing free slots.
- **Untimed entries on the day**: Entries that have no start time do not occupy any part of the timeline and therefore never block a slot.
- **A blocking entry runs to end of day**: If everything at or after 8 AM is occupied, this is the "no room left" case above.
- **Highlight is on nothing / empty day with no selectable task**: Pressing `a` with no task highlighted does nothing.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The planning tab MUST provide an action, bound to the `a` key, that auto-schedules the currently highlighted entry when that entry is a task.
- **FR-002**: Auto-scheduling MUST place the task at the earliest start time, at or after the scheduling floor (see FR-003), at which the task's full duration fits without overlapping any other timed entry on that day.
- **FR-003**: The scheduling floor MUST be 8:00 AM (local plan-day time) for any day other than the current local day. When the plan day being viewed is the current local day and the current time is later than 8:00 AM, the scheduling floor MUST instead be the current time, so the task is never placed to start in the past. Auto-scheduling MUST NOT place a task to start before the scheduling floor, even if earlier free time exists.
- **FR-004**: When evaluating free time, the system MUST treat all other timed entries on the day — both tasks and events — as obstacles, and MUST treat untimed entries as non-obstacles.
- **FR-005**: When the highlighted task is already timed, the system MUST exclude that task's own current placement from the obstacles (the task does not block itself) before searching for its new slot.
- **FR-006**: A candidate slot MUST be considered a fit only when the task's full duration ends at or before the end of the plan day.
- **FR-007**: When no qualifying slot exists for the task in the remainder of the day, the system MUST leave the plan unchanged and MUST inform the user that the task could not be auto-scheduled.
- **FR-008**: After a successful auto-schedule, the task MUST appear as a timed entry at the chosen start time, retaining its existing duration, and the planning view MUST reflect the new placement.
- **FR-014**: After a successful auto-schedule, the planning tab highlight/selection MUST follow the moved task to its new position so the moved task remains the selected entry.
- **FR-009**: Pressing the auto-schedule key while an event is highlighted MUST NOT move or alter any entry, and MUST inform the user that auto-scheduling applies only to tasks.
- **FR-010**: Pressing the auto-schedule key when no task is highlighted MUST do nothing and leave the plan unchanged.
- **FR-011**: When the earliest fitting slot equals the task's current placement, the system MUST treat the action as a no-op and MUST NOT register a spurious change.
- **FR-012**: The auto-schedule action MUST be discoverable through the planning tab's help/key listing alongside the other planning actions.
- **FR-013**: For a task that has no explicit duration (zero-length), the system MUST assume a default block length of 30 minutes when searching for a fitting slot, so the task is placed in the earliest gap at or after the scheduling floor that is at least 30 minutes long. The task's own stored duration is not changed by this assumption; the 30-minute block is used only to choose the start time.

### Key Entities *(include if feature involves data)*

- **Plan Entry**: An item placed on a given day's plan. It may be timed (has a start time and a duration) or untimed (no start time). It is either a *task* (linked to a task) or an *event* (a standalone commitment). Auto-scheduling acts only on task entries.
- **Plan Day**: The single day currently shown in the planning tab, whose timeline runs from its start (local midnight) to its end. The auto-schedule search is confined to this day, between the scheduling floor and the day's end.
- **Free Slot / Gap**: A stretch of the plan day's timeline, at or after the scheduling floor, not covered by any timed entry. A task "fits" a slot when its full duration lies within that slot and within the day.
- **Scheduling Floor**: The earliest start time auto-schedule may use on a given day: 8:00 AM for any day other than today, and the later of 8:00 AM and the current time when the plan day is today.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can schedule a task into the next open slot with a single keypress, with no need to read other entries' times or type a start time.
- **SC-002**: For any arrangement of existing timed entries, the task lands at the earliest slot at or after the scheduling floor where its duration fits, and never overlaps another timed entry.
- **SC-003**: No task is ever auto-placed to start before the scheduling floor — 8:00 AM on other days, and the current time when scheduling on today after 8:00 AM — so no task is placed into a past time.
- **SC-004**: Pressing the key on an event, on an unschedulable task, or with nothing highlighted never corrupts or silently changes the plan; the user always either gets a correct placement or a clear message explaining why nothing happened.
- **SC-005**: Auto-scheduling a task that already sits in the earliest fitting slot produces no change.

## Assumptions

- **"Highlighted task" means the currently selected entry in the planning tab**, and the action applies to whatever entry the selection is on, whether it is currently timed or untimed. The description's verb "move" is read to cover both placing an as-yet-untimed task and relocating an already-timed one.
- **8 AM is the lower bound of 08:00 in the plan day's local time for any day other than today; it is not user-configurable in this feature.** For today, once the current time is past 08:00, the lower bound rises to the current time so nothing is scheduled into the past (per the Session 2026-06-08 clarification).
- **The day's upper bound is local midnight (end of the plan day).** A task must finish by end of day to be placed; the search does not spill into the next day.
- **"Next available gap it fits into" means the earliest start time** (scanning forward from 8 AM) at which the task's full duration is free — not the smallest gap or a best-fit gap.
- **Obstacles are the day's timed entries (tasks and events alike).** Untimed entries are not on the timeline and never block placement.
- **The chosen start time is the exact minute the qualifying slot opens** (8:00, or the end of the preceding obstacle); start times are not snapped to a coarser granularity such as 5- or 15-minute boundaries.
- **Scope is the TUI planning tab only.** The CLI commands and the web client are out of scope for this feature.
- **The task keeps its existing duration** when auto-scheduled; auto-scheduling sets only the start time (and makes the entry timed).
- **A task with no duration is fitted as if it were a 30-minute block** (per FR-013) for the purpose of choosing a start time. The default block length is fixed at 30 minutes and is not user-configurable in this feature; the task's stored duration is left as-is.
