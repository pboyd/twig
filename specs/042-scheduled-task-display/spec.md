# Feature Specification: Show Scheduled Days in Task Details

**Feature Branch**: `042-scheduled-task-display`

**Created**: 2026-06-07

**Status**: Draft

**Input**: User description: "In the TUI, a user should be able to tell from the tasks tab if a task has been scheduled. This information should be shown in the details pane: `Scheduled for: 2026-06-07`. If it's scheduled for multiple days, all of them should be listed: `Scheduled for: 2026-06-07, 2026-06-08`. Don't show days in the past."

## Clarifications

### Session 2026-06-07

- Q: When deciding whether a scheduled day is "in the past" (and therefore hidden), which clock defines the current day? → A: The user's **local** current date. This matches the planning model (plan days are defined relative to local midnight). Note: this intentionally differs from the existing Snooze line in the same pane, which formats its date in UTC.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See that a task is scheduled (Priority: P1)

A user is reviewing their task list in the TUI tasks tab. They have placed some tasks onto specific days in their daily plan. When they select a task that has been scheduled for an upcoming day, the details pane tells them which day (or days) the task is scheduled for, so they can tell at a glance that the task already has a home on their calendar and does not need to be re-planned.

**Why this priority**: This is the entire purpose of the feature — surfacing existing scheduling information that is otherwise invisible from the tasks tab. Without it, a user cannot tell whether a task has been planned without leaving the tasks tab and inspecting each day's plan manually. It delivers the full value on its own.

**Independent Test**: In the TUI tasks tab, select a task that has been scheduled for today or a future day and confirm the details pane shows a `Scheduled for:` line listing that day. Select a task that has not been scheduled for any current/future day and confirm no such line appears.

**Acceptance Scenarios**:

1. **Given** a task that is scheduled for a single future day, **When** the user selects it in the tasks tab, **Then** the details pane shows `Scheduled for: <YYYY-MM-DD>` for that day.
2. **Given** a task that is scheduled for today, **When** the user selects it in the tasks tab, **Then** the details pane includes today's date in the `Scheduled for:` line.
3. **Given** a task that has never been scheduled for any day, **When** the user selects it in the tasks tab, **Then** the details pane shows no `Scheduled for:` line at all.

---

### User Story 2 - See every day a task is scheduled for (Priority: P2)

A user has scheduled the same task across more than one day (for example, work they expect to continue over consecutive days). When they select that task, the details pane lists all of the current-or-future days it is scheduled for, in date order, so they get a complete picture of where the task lives on their calendar.

**Why this priority**: Multi-day scheduling is a real but less common case. It extends the core value of Story 1 to a fuller picture; the feature is still useful with only single-day display, so this is secondary.

**Independent Test**: Schedule one task on two different future days, select it in the tasks tab, and confirm the details pane lists both days, comma-separated, in ascending date order.

**Acceptance Scenarios**:

1. **Given** a task scheduled for two or more current/future days, **When** the user selects it, **Then** the `Scheduled for:` line lists every such day, separated by `, ` (comma and space).
2. **Given** a task scheduled for multiple days, **When** the days are displayed, **Then** they appear in ascending chronological order regardless of the order in which they were scheduled.
3. **Given** a task scheduled for the same day more than once (multiple plan entries on one day), **When** the user selects it, **Then** that day appears only once in the list.

---

### User Story 3 - Past scheduling is not shown (Priority: P1)

A user has tasks that were scheduled for days that have already passed. When they select such a task, the details pane does not clutter the view with stale past dates; only days that are today or later are shown. If a task's only scheduling is in the past, no `Scheduled for:` line appears at all.

**Why this priority**: Suppressing past days is an explicit requirement and is essential to keeping the information relevant and uncluttered. It is tightly coupled to Stories 1 and 2 and must hold for the feature to behave correctly.

**Independent Test**: Schedule a task only for a day before the current day, select it, and confirm no `Scheduled for:` line appears. Schedule a task for one past day and one future day, and confirm only the future day is shown.

**Acceptance Scenarios**:

1. **Given** a task scheduled only for one or more days strictly before today, **When** the user selects it, **Then** the details pane shows no `Scheduled for:` line.
2. **Given** a task scheduled for a mix of past and current/future days, **When** the user selects it, **Then** only the current/future days are listed and past days are omitted.
3. **Given** a task scheduled for a past day and for today, **When** the user selects it, **Then** today is shown and the past day is not.

---

### Edge Cases

- **All scheduling in the past**: No `Scheduled for:` line is shown (treated as "not scheduled" for display purposes).
- **Day boundary**: "Today" is determined by the user's local current date so the display stays correct as the date rolls over; a day that was "today" yesterday becomes a past day and drops off.
- **Completed task that is scheduled**: The `Scheduled for:` line still reflects its current/future scheduled days; completion does not suppress it.
- **Snoozed task that is scheduled**: The `Scheduled for:` line is shown alongside the existing snooze information when both apply.
- **Long list of days**: The line follows the same text-wrapping behavior as other details-pane content when it exceeds the pane width.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tasks-tab details pane MUST display a line, labeled consistently with the existing detail labels, that lists the days a selected task is scheduled for, using the form `Scheduled for: <days>`.
- **FR-002**: Each scheduled day MUST be displayed in `YYYY-MM-DD` format.
- **FR-003**: When a task is scheduled for multiple qualifying days, the days MUST be listed on a single labeled line, separated by `, ` (comma followed by a space).
- **FR-004**: Listed days MUST appear in ascending chronological order.
- **FR-005**: A day that is strictly before the user's current **local** date MUST NOT be shown.
- **FR-006**: The current day (today, by the user's local date) MUST be shown when the task is scheduled for it.
- **FR-007**: When a task has no qualifying (current-or-future) scheduled days, the details pane MUST omit the `Scheduled for:` line entirely.
- **FR-008**: A day MUST appear at most once in the list even if the task is scheduled more than once on that day.
- **FR-009**: The feature MUST treat "scheduled" as a task being present in the user's daily plan for a given day (i.e., a plan entry linked to that task on that day).
- **FR-010**: The display MUST be consistent in both the plain (unstyled) and styled rendering paths of the details pane.

### Key Entities *(include if feature involves data)*

- **Task**: The item shown in the tasks tab; the subject whose scheduled days are being displayed.
- **Scheduled Day**: A calendar day (identified as `YYYY-MM-DD`) on which the task has been placed in the user's daily plan. A task may have zero, one, or many scheduled days. Only days that are today or later are relevant to this display.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can determine whether a selected task is scheduled, and on which upcoming day(s), entirely from the tasks-tab details pane without opening any day's plan.
- **SC-002**: For a task scheduled across N current/future days, all N days — and no past days — are shown, in ascending order, with no duplicate days.
- **SC-003**: For a task with no current/future scheduling, the details pane shows no scheduling line, so an unscheduled (or only-past-scheduled) task is visually indistinguishable from one that was never scheduled.
- **SC-004**: The displayed scheduling reflects the task's actual current plan placement at the time the task is viewed.

## Assumptions

- "Scheduled" refers to a task being placed in the user's **daily plan** for a day (a plan entry that links the task to that day). This matches the existing planning model where tasks are added to specific days. Other notions a user might call "scheduling" (e.g., the task's due date or its snooze-until day) are out of scope for this line; due dates and snooze already have their own display.
- "Don't show days in the past" means days strictly before the current local date; the current day is shown. This is consistent with the example, which lists `2026-06-07` (the current date) as a valid scheduled day.
- The current date is the user's **local** current date (per the Session 2026-06-07 clarification). This is consistent with the planning model, in which plan days are defined relative to local midnight. It intentionally differs from the existing Snooze line's UTC-based date formatting in the same pane.
- The label and alignment follow the existing details-pane conventions (e.g., the `Due:`, `Completed:`, and `Snooze:` lines), and the line appears in a position consistent with those existing labels.
- This feature is read-only display; it does not add any new way to schedule, reschedule, or unschedule a task.
- Scope is limited to the TUI tasks tab. The CLI and web clients are out of scope.
