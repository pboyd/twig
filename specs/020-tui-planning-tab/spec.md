# Feature Specification: TUI Planning Tab

**Feature Branch**: `020-tui-planning-tab`

**Created**: 2026-05-28

**Status**: Draft

**Input**: User description: "Bring the daily planning mode into the interactive TUI as a new Planning tab alongside the existing Tasks tab, looking like the CLI day-planner grid and offering the same planning features."

## Clarifications

### Session 2026-05-28

- Q: When a modal/prompt is open (Tasks edit/move form, or a Planning picker/rename/move/time prompt), what should the tab-switch key do? → A: Block tab-switching while any modal/prompt is open; the user must confirm or cancel it first, and the active form keeps consuming the Tab keys.
- Q: When does the Planning grid refresh from the server? → A: Reload the day's entries each time the Planning tab is activated and after every successful mutation, plus offer a manual refresh key for parity with the Tasks tab.
- Q: Is the running pomodoro shown while the Planning tab is active? → A: Yes — the shared status bar (including a running pomodoro) is displayed identically on both tabs (already required by FR-004).
- Q: Does the now-marker advance on its own while the tab is open? → A: Yes — the now-marker MUST advance automatically over time via a periodic refresh, independent of any running pomodoro, so it stays current even with no user input.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View the day's plan inside the TUI (Priority: P1)

A user working in the interactive TUI wants to see their daily plan without dropping back to the command line. A tab bar at the top of the screen offers "Tasks" and "Planning". The user switches to the Planning tab and sees their day laid out as a calendar grid — the same visual style as the CLI day-planner: time rails down the side, hour dividers, and each scheduled item drawn as a bordered box spanning its time range with its time window and name. A marker shows the current time. Completed task entries appear struck through. The user switches back to the Tasks tab and their task view is exactly as they left it.

**Why this priority**: Seeing the plan inside the TUI is the foundational slice. Without it, no other planning interaction is possible. On its own it already delivers value: a user can review their schedule without leaving the interactive view, and the shared timer/help status bar keeps working across both tabs.

**Independent Test**: Launch the TUI, switch to the Planning tab, and confirm the day's entries render as a grid matching the CLI layout (minus the entry-number prefix), with the now-marker and completion styling present. Switch back and confirm the Tasks tab is unchanged. Fully testable with a stub plan service and a fixed clock.

**Acceptance Scenarios**:

1. **Given** the TUI is open on the Tasks tab, **When** the user presses the next-tab key, **Then** the Planning tab becomes active and the day's plan renders as a calendar grid.
2. **Given** the Planning tab is active, **When** the user presses the previous-tab key (or cycles forward again), **Then** the Tasks tab becomes active with its prior cursor and expansion state intact.
3. **Given** the Planning tab is showing a day with scheduled entries, **When** the grid renders, **Then** each entry appears as a bordered box across its time range showing its `HH:MM-HH:MM` window and name, with no entry-number prefix.
4. **Given** the viewed day is today, **When** the grid renders, **Then** a now-marker indicates the current time slot.
5. **Given** an entry references a completed task, **When** the grid renders, **Then** that entry is shown struck through; event entries are never shown as completed.
6. **Given** a pomodoro is running, **When** the user switches between tabs, **Then** the timer in the status bar keeps counting down and stays visible on both tabs.

---

### User Story 2 - Schedule tasks and block out events from the Planning tab (Priority: P2)

From the Planning tab, the user adds items to the day. To schedule an existing task, the user presses the add-task key, a picker listing their tasks (as the task tree) appears, they choose one, then enter a start time and an optional duration; the task is placed on the grid. To block out non-task time (a meeting, lunch, a break), the user presses the add-event key, types a name, then enters a start time and optional duration. Both new items appear immediately on the grid.

**Why this priority**: Building up the day is the primary reason to plan. It depends on the Planning tab and grid existing (Story 1), so it comes second. It is independently valuable: a user who can view and populate the plan has a complete planning loop even before in-place editing exists.

**Independent Test**: On the Planning tab, add a task via the picker with a start and duration and confirm it appears on the grid at the right place; add an event with a name and time and confirm it appears. Testable with a stub plan service and stub task list.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active, **When** the user invokes add-task, chooses a task from the picker, and enters a start time and duration, **Then** a new entry for that task appears on the grid spanning the entered window.
2. **Given** the Planning tab is active, **When** the user invokes add-task and enters only a start time (no duration), **Then** the entry is created with the system's default duration and appears on the grid.
3. **Given** the Planning tab is active, **When** the user invokes add-event, enters a name, start time, and optional duration, **Then** a new event entry with that name appears on the grid.
4. **Given** the user is entering a start time or duration, **When** they type a value in any accepted format (e.g. `13:15`, `1315`, `1:15pm`; `90m`, `1h30m`; or an end time in place of a duration), **Then** the value is accepted and applied.
5. **Given** the user is in the task picker or a time prompt, **When** they cancel, **Then** no entry is created and the grid is unchanged.
6. **Given** the user enters an invalid time or duration, **When** they confirm, **Then** a readable error is shown and no entry is created.

---

### User Story 3 - Edit and remove plan entries in place (Priority: P3)

The user reviews the grid and adjusts it. A highlight sits on one entry; up/down moves the highlight between entries in chronological order. With an entry highlighted, the user can rename it, move it to a new start time and/or duration, or remove it. The user can also clear the remainder of the day from a chosen time forward. Each change is reflected on the grid immediately.

**Why this priority**: In-place editing refines a plan that already exists. It depends on Stories 1 and 2 and is the least essential slice — a user could otherwise re-add entries — so it is prioritized last. It still completes parity with the CLI planning features.

**Independent Test**: With entries on the grid, move the highlight to a specific entry, rename it, move it to a new time, and remove it, confirming the grid updates after each; clear from a time and confirm later entries are gone. Testable with a stub plan service.

**Acceptance Scenarios**:

1. **Given** the grid shows two or more entries, **When** the user presses up/down, **Then** the highlight moves between entries in chronological order and the highlighted entry is visually distinct.
2. **Given** an entry is highlighted, **When** the user invokes rename and enters a new name, **Then** the entry's name updates on the grid.
3. **Given** an entry is highlighted, **When** the user invokes move and enters a new start time and/or duration, **Then** the entry relocates on the grid to the new window.
4. **Given** an entry is highlighted, **When** the user invokes remove, **Then** the entry disappears from the grid.
5. **Given** the grid has entries later in the day, **When** the user invokes clear from a chosen start time (defaulting to the current time), **Then** entries at or after that time are removed and earlier entries remain.

---

### User Story 4 - Plan a different day (Priority: P3)

The user plans ahead or reviews a past day. The Planning tab defaults to today and shows which day is in view. The user steps to the next or previous day, edits that day's plan with all the same actions, and can jump straight back to today. Tomorrow and today are the common targets, but any day — past or future — is reachable.

**Why this priority**: Most planning happens for today, so single-day support already delivers most of the value; multi-day navigation is an enhancement layered on top. It depends on the Planning tab existing and reuses every other action, so it is prioritized last alongside in-place editing.

**Independent Test**: On the Planning tab, step forward a day and confirm the displayed day and its (separate) entries change; add an entry and confirm it lands on that day; jump to today and confirm the view returns. Testable with a stub plan service keyed by day and a fixed clock.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active on today, **When** the user steps to the next day, **Then** the view shows the next day, its day label updates, and its own entries are displayed.
2. **Given** the user is viewing a day other than today, **When** they invoke any add/edit/remove/clear action, **Then** the change applies to the day in view.
3. **Given** the user is viewing a day other than today, **When** they press the jump-to-today key, **Then** the view returns to today.
4. **Given** the user is viewing a day that is not today, **When** the grid renders, **Then** no now-marker is shown.

---

### Edge Cases

- **Empty day**: The grid renders the default working-hours window with no entry boxes and (when viewing today) the now-marker.
- **Entries outside the default window**: The visible time window expands to contain every entry, matching the CLI grid behavior.
- **No entries to highlight**: Edit/move/remove actions are unavailable or no-ops when the day has no entries; the user is not shown an error for an empty selection.
- **No tasks to schedule**: Invoking add-task when the user has no tasks shows an empty picker that can be cancelled without creating an entry.
- **Server rejects a change** (e.g. invalid time range, overlapping precondition, entry not found): a readable error appears in the status bar and the grid reflects the last known good state.
- **Narrow terminal**: The grid and tab bar degrade gracefully on small widths without corrupting the layout, consistent with the existing TUI's handling.
- **Switching tabs mid-action**: While any modal/prompt is open (a Planning picker, rename, move, or time prompt, or a Tasks edit/move form), the tab-switch key is blocked; the user must confirm or cancel the prompt before switching tabs, so neither tab is ever left in a broken modal state.
- **Highlight after mutation**: After adding, moving, or removing an entry, the highlight settles on a sensible entry rather than disappearing or jumping arbitrarily.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The TUI MUST present a persistent tab bar offering at least "Tasks" and "Planning", with the active tab visually indicated.
- **FR-002**: Users MUST be able to switch tabs with a forward key and a backward key, and the active tab MUST own the full screen area below the tab bar.
- **FR-003**: Each tab MUST retain its own state when the other tab is active, so switching away and back does not reset cursor, selection, or in-view day.
- **FR-004**: The shared status bar (running pomodoro, help hints, error messages) and the quit-confirmation behavior MUST function identically on both tabs.
- **FR-005**: The Planning tab MUST render the in-view day's entries as a calendar grid matching the CLI day-planner's visual style — time rails, hour dividers, 15-minute resolution, and bordered boxes spanning each entry's time range with its `HH:MM-HH:MM` window and name.
- **FR-006**: The Planning grid MUST NOT display the per-entry number prefix used by the CLI; entries are identified by cursor selection instead.
- **FR-007**: The Planning tab MUST visually highlight one selected entry and MUST let the user move the selection between entries in chronological order.
- **FR-008**: When the in-view day is today, the grid MUST show a marker for the current time; when it is any other day, no such marker is shown.
- **FR-008a**: When the Planning tab is active on today, the now-marker MUST advance automatically as time passes, without requiring user input and independent of whether a pomodoro is running, so a long-open tab keeps showing the correct current time slot.
- **FR-009**: Entries linked to a completed task MUST render struck through; event entries MUST never render as completed.
- **FR-010**: Users MUST be able to schedule an existing task onto the in-view day by selecting it from a task picker and providing a start time and an optional duration.
- **FR-011**: Users MUST be able to add an event to the in-view day by providing a name, a start time, and an optional duration.
- **FR-012**: Users MUST be able to rename the selected entry.
- **FR-013**: Users MUST be able to move the selected entry to a new start time and/or duration.
- **FR-014**: Users MUST be able to remove the selected entry.
- **FR-015**: Users MUST be able to clear entries from a chosen start time forward, defaulting to the current time.
- **FR-016**: Time and duration inputs MUST accept the same formats as the existing CLI planning commands (clock times such as `13:15`, `1315`, `1:15pm`; durations such as `90m`, `1h30m`; and an end time supplied in place of a duration).
- **FR-017**: The Planning tab MUST default to today and MUST display which day is in view.
- **FR-018**: Users MUST be able to navigate to the previous day, the next day, and directly back to today; any day (past or future) MUST be reachable and editable.
- **FR-019**: The grid MUST reflect each successful add, rename, move, remove, and clear immediately.
- **FR-020**: Users MUST be able to cancel any planning prompt (picker, name entry, time entry) without changing the plan.
- **FR-021**: When a planning action fails validation or is rejected by the server, the system MUST show a readable error and leave the plan in its last known good state.
- **FR-022**: Planning actions MUST operate on the same stored plan data as the CLI, so a change made in the TUI is visible from the CLI and vice versa.
- **FR-023**: While any modal or prompt is open (a Tasks edit/move form or a Planning picker, rename, move, or time prompt), the system MUST NOT switch tabs; the open form retains control of all keys until the user confirms or cancels it.
- **FR-024**: The Planning tab MUST reload the in-view day's entries each time it becomes active and after every successful mutation, and MUST provide a manual refresh key (consistent with the Tasks tab) that reloads the current day on demand.

### Key Entities *(include if feature involves data)*

- **Plan entry**: A scheduled item on a given day. Has a day, a start time, a duration, and a display name. May optionally link to a task (a "task entry") or stand alone (an "event"). A task entry's completion is derived from its linked task; events are never complete.
- **Day**: The calendar date currently in view on the Planning tab, defaulting to today.
- **Task**: An existing to-do item from the Tasks tab that can be selected in the picker and placed onto the plan; supplies the entry's name when no override is given.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can view their day's plan from inside the TUI without exiting to the command line.
- **SC-002**: Every planning action available in the CLI day-planner (schedule task, add event, rename, move, remove, clear, view another day) is achievable from the Planning tab.
- **SC-003**: The Planning grid is visually recognizable as the same day-planner as the CLI, differing only by the omitted entry-number prefix and the on-screen selection highlight.
- **SC-004**: A user can schedule a task onto the plan — from opening the picker to seeing the entry on the grid — in under 15 seconds without consulting documentation.
- **SC-005**: Switching between the Tasks and Planning tabs preserves each tab's state and never interrupts a running pomodoro timer.
- **SC-006**: A change made on the Planning tab is immediately visible on the grid and is consistent with what the CLI shows for the same day.

## Assumptions

- The existing plan service and its operations (list, add task, add event, rename, move, remove, clear) are sufficient; no new server-side capability is required, and entry completion remains derived server-side from the linked task.
- There is no "complete entry" action on the Planning tab; toggling task completion remains a Tasks-tab concern.
- The task picker presents the same task tree shown on the Tasks tab; choosing a parent or child task is allowed wherever the CLI allows scheduling that task.
- The Planning tab edits one day at a time; there is no multi-day or week view in scope.
- Default working-hours window and entry duration defaults match the existing CLI day-planner behavior.
- Reordering of overlapping entries and any overlap rules are governed by the existing plan service; this feature does not introduce new scheduling-conflict logic beyond surfacing server errors.
- The feature targets the interactive TUI running on a TTY; the CLI planning commands remain available and unchanged.
