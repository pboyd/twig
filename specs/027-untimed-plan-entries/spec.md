# Feature Specification: Untimed Plan Entries

**Feature Branch**: `027-untimed-plan-entries`

**Created**: 2026-06-01

**Status**: Draft

**Input**: User description: "users need the ability to add a task to a daily plan without a time. There are a few reasons for this: (1) Some tasks are quick enough to fit in the gaps and don't require dedicated focus time, but still need to be done at some point in the day (e.g. make a doctor's appointment); (2) A typical morning workflow is picking the tasks that need to be done that day and then scheduling them in a time slot — a related problem is scheduling tasks for another day. Plan entries can be tasks or events, but events are illogical without a time. Untimed entries display above the day planner (a new top-left pane in the TUI), are highlightable/editable like any other entry, and still have a duration that renders like any other entry. In the CLI, the start argument to `twig plan task` and `twig plan mv` should be optional, and `null` (case-insensitive) may be passed as the start so the user can still specify the end/duration. In the TUI, the start time is optional when adding or editing, a user on the Tasks tab can send a task to the day's plan with `p` or pick the day with `ctrl+p`, and the existing edit flow can add a time to an entry that lacks one."

## Clarifications

### Session 2026-06-01

- Q: How does `ctrl+p` on the Tasks tab let the user choose the target day? → A: Free-form absolute date entry (`YYYY-MM-DD`), pre-filled with the day after today (tomorrow) as the default.
- Q: How does the highlight move between the untimed pane and the day planner grid in the Planning view? → A: A single unified Up/Down highlight cycle that flows through the untimed pane first, then into the grid, and wraps back — no separate focus-switch key.
- Q: What does the untimed pane look like on a day with no untimed entries? → A: The pane is hidden entirely; the day grid uses the full area and the pane appears as soon as an untimed entry exists.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Capture a must-do-today task without committing to a time (Priority: P1)

A user wants to make sure a small, schedule-flexible task (e.g. "make a doctor's appointment") gets done today, but it does not warrant a dedicated time block. They add the task to today's plan without giving it a start time. The entry is recorded on the day and is visible as part of that day's plan, separate from the time-gridded schedule.

**Why this priority**: This is the core capability the whole feature exists to provide — a plan entry that belongs to a day but has no start time. Everything else (TUI pane, quick-send shortcuts, scheduling later) builds on it. Without this, none of the other stories are possible.

**Independent Test**: Add a task to a day's plan with no start time via the CLI, then list that day's plan and confirm the entry appears, is associated with the correct task, has a duration, and carries no start time.

**Acceptance Scenarios**:

1. **Given** an existing task, **When** the user adds it to today's plan without specifying a start time, **Then** the plan records an untimed entry linked to that task with a default duration.
2. **Given** an untimed entry on a day, **When** the user lists that day's plan, **Then** the untimed entry is returned and clearly distinguishable from timed entries.
3. **Given** the user wants an untimed entry with a non-default duration, **When** they add the task with the start explicitly set to `null` (any casing) and a duration supplied, **Then** the entry is created untimed with the requested duration.

---

### User Story 2 - See and manage untimed entries in the TUI (Priority: P2)

A user working in the TUI Planning view sees their untimed entries grouped in a dedicated pane at the top-left, above the day planner grid. Each untimed entry looks like any other plan entry (dark box, one line per 15 minutes of duration) and can be highlighted and edited (rename, change duration, delete) using the same interactions as gridded entries.

**Why this priority**: Most day-to-day planning happens in the TUI. Surfacing untimed entries where they can be reviewed and edited makes the feature usable in the primary interface, but it depends on US1 existing first.

**Independent Test**: With at least one untimed entry on the displayed day, open the Planning view and confirm the untimed pane renders the entry, that it can be highlighted, and that editing (rename/duration/delete) affects only that entry.

**Acceptance Scenarios**:

1. **Given** a day with one or more untimed entries, **When** the user opens the Planning view, **Then** the untimed entries appear in a pane at the top-left, above the day planner.
2. **Given** the untimed pane and the grid both contain entries, **When** the user presses Up/Down, **Then** the highlight moves in a single unified cycle through the untimed entries and the gridded entries (untimed first, then grid, wrapping around) with no separate focus-switch key.
3. **Given** an untimed entry is highlighted, **When** the user edits its name or duration or deletes it, **Then** the change applies to that entry and the pane re-renders to reflect it.
4. **Given** an untimed entry with a duration of N minutes, **When** it is rendered, **Then** it occupies one line per 15 minutes of duration, matching the visual style of gridded entries.

---

### User Story 3 - Schedule an untimed entry later (and unschedule a timed one) (Priority: P2)

After picking the tasks for the day, the user assigns times to them. They take an untimed entry and give it a start time, which moves it out of the untimed pane and onto the day planner grid at that time. Conversely, they can clear an entry's start time to move it back into the untimed pane.

**Why this priority**: This completes the "pick first, schedule later" morning workflow that motivates the feature. It depends on untimed entries existing (US1) and being visible (US2), so it ranks alongside US2 rather than above it.

**Independent Test**: Take an untimed entry, assign it a start time via the edit flow (TUI) or the move command (CLI), and confirm it leaves the untimed pane and appears on the grid at that time; then clear the start time and confirm it returns to the untimed pane.

**Acceptance Scenarios**:

1. **Given** an untimed entry, **When** the user assigns it a start time through the TUI edit flow, **Then** the entry moves from the untimed pane to the day planner grid at the chosen time.
2. **Given** an untimed entry, **When** the user moves it via the CLI with a start time, **Then** the entry becomes a timed entry at that start.
3. **Given** a timed entry, **When** the user moves it with the start set to `null` (any casing) or omitted, **Then** the entry becomes untimed and appears in the untimed pane, keeping its duration.
4. **Given** an entry being scheduled, **When** the user supplies only a start time, **Then** the entry keeps its existing duration.

---

### User Story 4 - Send tasks from the Tasks tab to a plan (Priority: P3)

While browsing the Tasks tab in the TUI, the user finds a task they want to act on today and presses `p` to send it straight to today's plan as an untimed entry, without leaving the Tasks tab. If they want it on a different day (e.g. scheduling work for tomorrow), they press `ctrl+p`, choose the day, and the task is added to that day's plan as an untimed entry.

**Why this priority**: This is a workflow accelerator that makes the morning "pick the day's tasks" routine fast, but the same outcome is achievable through US1/US3, so it is the lowest priority slice.

**Independent Test**: From the Tasks tab, highlight a task and press `p`; confirm an untimed entry for that task appears on today's plan. Repeat with `ctrl+p`, choose another day, and confirm the entry lands on that day.

**Acceptance Scenarios**:

1. **Given** a highlighted task on the Tasks tab, **When** the user presses `p`, **Then** an untimed entry linked to that task is added to today's plan.
2. **Given** a highlighted task on the Tasks tab, **When** the user presses `ctrl+p`, a date prompt appears pre-filled with tomorrow's date, **And** the user accepts or edits it to a valid `YYYY-MM-DD` date, **Then** an untimed entry linked to that task is added to that day's plan.
3. **Given** the date prompt from `ctrl+p`, **When** the user cancels it, **Then** no entry is added.
4. **Given** the date prompt from `ctrl+p`, **When** the user enters a value that is not a valid `YYYY-MM-DD` date, **Then** no entry is added and the user is informed the date is invalid.

---

### Edge Cases

- **Events cannot be untimed**: Creating an event without a start time is rejected; events always require a time. Only task entries may be untimed.
- **Empty untimed pane**: When a day has no untimed entries, the untimed pane is hidden entirely and the day planner grid uses the full area; the pane appears as soon as the day has at least one untimed entry.
- **CLI start = `null` without a duration**: A literal `null` start with no duration given creates an untimed entry with the default duration (same default the server uses today).
- **End time supplied for an untimed entry**: An end-of-day clock time is meaningless without a start; when the start is null/omitted, the trailing argument is interpreted as a duration, not a wall-clock end time.
- **Long-duration untimed entry**: An untimed entry with a large duration still renders one line per 15 minutes within the untimed pane.
- **Completed task entries**: Untimed entries that reference completed tasks follow the same completed-display rules as timed entries.
- **Duplicate sends**: Pressing `p`/`ctrl+p` (or re-running the CLI add) for a task already present on the plan follows the same behavior as adding any entry today (a new entry is created); the feature does not introduce de-duplication.
- **Ordering**: Untimed entries on a day appear in a stable, predictable order (creation order) since they have no time to sort by.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A plan task entry MUST be able to exist on a day without a start time ("untimed").
- **FR-002**: An untimed entry MUST retain a duration; when none is supplied, the system MUST apply the same default duration it uses for timed task entries.
- **FR-003**: Event entries MUST always have a start time; the system MUST reject any attempt to create an untimed event.
- **FR-004**: Listing a day's plan MUST return untimed entries alongside timed entries, with untimed entries distinguishable from timed ones.
- **FR-005**: Untimed entries on a day MUST be returned/displayed in a stable, predictable order (creation order) given the absence of a start time.
- **FR-006**: In the CLI, the start argument to `twig plan task` MUST be optional; omitting it MUST create an untimed entry.
- **FR-007**: In the CLI, the literal value `null` (case-insensitive) MUST be accepted in the start position for `twig plan task` and `twig plan mv`, designating an untimed entry while still allowing a duration to be supplied as the following argument.
- **FR-008**: In the CLI, the start argument to `twig plan mv` MUST be optional; omitting it or passing `null` MUST make the target entry untimed while preserving its duration.
- **FR-009**: In the CLI, `twig plan mv` with a start time supplied MUST schedule a previously untimed entry at that time.
- **FR-010**: The TUI MUST display untimed entries in a dedicated pane at the top-left, above the day planner grid, when the day has at least one untimed entry; when the day has none, the pane MUST be hidden so the day grid uses the full area.
- **FR-011**: Untimed entries in the TUI MUST be rendered with the same visual treatment as gridded entries (dark box, one display line per 15 minutes of duration).
- **FR-012**: Untimed entries in the TUI MUST be individually highlightable and editable (rename, change duration, delete) using the same interactions available for gridded entries.
- **FR-012a**: The Planning view MUST move the highlight in a single unified Up/Down cycle that traverses the untimed pane and the day grid (untimed entries first, then gridded entries, wrapping around), without requiring a separate key to switch focus between the two areas.
- **FR-013**: When adding or editing a plan entry in the TUI, the start time MUST be optional; leaving it empty MUST produce or preserve an untimed entry.
- **FR-014**: The TUI edit flow MUST allow adding a start time to an entry that lacks one, moving it from the untimed pane to the day planner grid at that time.
- **FR-015**: The TUI edit flow MUST allow clearing an entry's start time, moving a timed entry into the untimed pane while preserving its duration.
- **FR-016**: On the TUI Tasks tab, pressing `p` while a task is highlighted MUST add an untimed entry for that task to today's plan.
- **FR-017**: On the TUI Tasks tab, pressing `ctrl+p` while a task is highlighted MUST present a date entry prompt pre-filled with tomorrow's date (the day after today); accepting or editing it to a valid `YYYY-MM-DD` date MUST add an untimed entry for that task to that day's plan.
- **FR-018**: Cancelling the `ctrl+p` date prompt, or entering a value that is not a valid `YYYY-MM-DD` date, MUST add no entry and leave the plan unchanged (an invalid date MUST inform the user).
- **FR-019**: Existing timed-entry behavior (creation, listing, move, rename, remove, clear) MUST remain unchanged for entries that have a start time.

### Key Entities *(include if feature involves data)*

- **Plan Entry**: A segment of one day's plan, identified within a user's data by (day, id). May be linked to a task or stand alone as an event. Has an optional display name, a duration, and — newly — an *optional* start time. An entry with a start time is "timed" and appears on the day planner grid; an entry without one is "untimed" and appears in the untimed pane. Events are constrained to always be timed; only task-linked entries may be untimed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can add a task to a day's plan without a start time and see it persisted in 1 CLI command (or 1 keypress from the Tasks tab).
- **SC-002**: From the Tasks tab, a user can send a highlighted task to today's plan in a single keypress (`p`), and to a chosen other day in two steps (`ctrl+p`, then pick the day).
- **SC-003**: 100% of untimed entries appear in the dedicated untimed pane and never on the time grid; 100% of timed entries appear on the grid and never in the untimed pane.
- **SC-004**: A user can convert an entry between untimed and timed (and back) without losing its duration or its link to the underlying task.
- **SC-005**: Attempting to create an untimed event is rejected 100% of the time, with the user informed that events require a time.
- **SC-006**: Untimed entries render at one display line per 15 minutes of duration, visually matching gridded entries, so a user cannot tell the two apart by style alone.

## Assumptions

- **"End" for an untimed entry means duration**: Because a wall-clock end time is meaningless without a start, when the CLI start argument is omitted or `null`, the trailing `[duration|end]` argument is interpreted as a duration.
- **`p` sends an *untimed* entry**: The quick-send shortcuts create untimed entries (consistent with the "pick now, schedule later" workflow) rather than guessing a time.
- **`p` targets today; `ctrl+p` targets a chosen day**: "Today" is the current local date; `ctrl+p` opens a `YYYY-MM-DD` date prompt defaulting to tomorrow, so the common "schedule for tomorrow" case is a single confirmation.
- **Default duration is unchanged**: Untimed task entries use the existing server default duration logic (derived from the task's estimate, falling back to the standard default) when no duration is supplied.
- **Completed-entry display is inherited**: Untimed entries that reference completed tasks follow whatever completed-task display rules already apply to plan entries; this feature does not change those rules.
- **No de-duplication**: Sending or adding a task that is already on the plan creates an additional entry, matching current add behavior.
- **Ordering is by creation**: With no start time to sort on, untimed entries are ordered by their per-day id (creation order).
- **The untimed pane occupies the top-left**: Placing it above the day planner is a layout change within the existing Planning view; the day planner grid remains the primary element and reclaims the full area whenever there are no untimed entries.
