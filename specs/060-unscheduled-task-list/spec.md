# Feature Specification: Unscheduled Tasks as a List

**Feature Branch**: `060-unscheduled-task-list`

**Created**: 2026-07-02

**Status**: Draft

**Input**: User description: "unscheduled tasks were originally intended to be a holding area for tasks that needed scheduling. In practice, however, they're more likely to be quick tasks that don't need a block of time. For example, \"schedule doctor appointment\" can be done any time there's a 5-minute gap in the schedule. As such, they should be displayed like tasks in the tasks tab, instead of like the blocks down below. The functionality shouldn't change, only the way they're presented. In the TUI, unscheduled tasks should have a list above the day planner grid and should be visually separate from it. The web app plan view is due for an overhaul, so let's not change anything there right now."

## Clarifications

### Session 2026-07-02

- Q: How should each unscheduled task row be presented in the Plan tab list? → A: Checkbox + name — a completion checkbox (☐/☑) followed by the task name, no visible ID (matches the Tasks tab).
- Q: How should the unscheduled list be visually separated from the day grid? → A: Divider line only — a horizontal divider between the list and the grid, no section header.
- Q: When there are more unscheduled tasks than fit above the grid, what should happen? → A: List grows, grid shrinks — the list expands to show all unscheduled tasks, taking vertical space from the grid.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See unscheduled tasks as a simple list (Priority: P1)

A person planning their day opens the Plan tab in the TUI. Their unscheduled tasks — the quick, "do it in any spare moment" items like *schedule doctor appointment* — appear as a plain list above the day planner grid, presented the same way tasks appear on the Tasks tab, rather than as time-block boxes. This makes clear that these items don't own a slot on the calendar; they are things to knock out whenever a gap appears.

**Why this priority**: This is the entire point of the feature. Presenting unscheduled tasks as blocks miscommunicates their nature (that they don't need a reserved block of time). Changing the presentation to a list is the core value and delivers the improvement on its own.

**Independent Test**: Open the Plan tab for a day that has at least one unscheduled task and at least one scheduled block. Confirm the unscheduled tasks render as list rows (in the Tasks-tab style) sitting above the grid, and the scheduled items still render as blocks on the grid. Delivers the full presentation change with nothing else required.

**Acceptance Scenarios**:

1. **Given** a day with one or more unscheduled tasks, **When** the Plan tab is viewed in the TUI, **Then** the unscheduled tasks appear as a list of rows above the day planner grid, styled like rows on the Tasks tab (not as time-block boxes).
2. **Given** a day with unscheduled tasks and scheduled blocks, **When** the Plan tab is viewed, **Then** the unscheduled list and the day grid are visually separated so it is obvious which items are unscheduled and which occupy time slots.
3. **Given** a day with no unscheduled tasks, **When** the Plan tab is viewed, **Then** no unscheduled list is shown and the day grid occupies the space as it did before this change.

---

### User Story 2 - Interact with unscheduled tasks unchanged (Priority: P1)

A person uses the same keyboard actions on unscheduled tasks that they could before — selecting one, completing it, editing it, scheduling it (giving it a time), reordering it within the unscheduled list, and viewing its details — with no change in what those actions do. Only how the tasks look has changed.

**Why this priority**: The user explicitly required that functionality not change. If any existing interaction breaks or behaves differently, the feature has regressed. This must ship together with the visual change, so it shares top priority.

**Independent Test**: On the Plan tab, move the selection onto an unscheduled task in the new list and perform each previously available action (select, complete/uncomplete, edit, schedule, reorder, open details). Confirm each produces the same result it did before the presentation change.

**Acceptance Scenarios**:

1. **Given** the unscheduled list has focus, **When** the user moves the selection through it, **Then** selection highlights the same tasks and moves between the unscheduled list and the grid exactly as before.
2. **Given** an unscheduled task is selected, **When** the user completes it, **Then** its completion state changes and it displays as completed, matching the prior behavior for completing an unscheduled task.
3. **Given** an unscheduled task is selected, **When** the user assigns it a start time (schedules it), **Then** it becomes a scheduled block on the grid and leaves the unscheduled list, as before.
4. **Given** multiple unscheduled tasks, **When** the user reorders one within the list, **Then** the order changes and persists exactly as it did before this change.
5. **Given** an unscheduled task is selected, **When** the user opens its details, **Then** the same detail information is shown as before.

---

### Edge Cases

- **Many unscheduled tasks**: When the unscheduled list is long enough that the list plus the grid exceed the available height, the list grows to show all unscheduled tasks and the grid shrinks (scrolls/compresses) to yield the space; the layout must remain intact and usable.
- **Completed unscheduled tasks**: A completed unscheduled task is presented in the list with the same "completed" treatment used elsewhere (e.g., a checked/struck-through row), consistent with existing behavior for whether completed items are shown or hidden.
- **Only unscheduled tasks (empty grid)**: A day with unscheduled tasks but no scheduled blocks shows the list above an otherwise empty grid, still visually separated.
- **No unscheduled tasks**: The unscheduled section is entirely absent (no empty header/separator), leaving the grid unchanged from today's no-unscheduled layout.
- **Long task names**: A task name too wide for the list row is handled gracefully (wrapped or truncated) consistent with how the Tasks tab presents long names.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: In the TUI Plan tab, the system MUST present unscheduled tasks (tasks with no assigned start time) as a list of rows rather than as time-block boxes.
- **FR-002**: The unscheduled task list MUST appear above the day planner grid.
- **FR-003**: The unscheduled task list MUST be visually separated from the day planner grid by a horizontal divider line between the two regions; no section header/title is shown above the list.
- **FR-004**: Each unscheduled task row MUST be presented in the same style used for tasks on the Tasks tab — a completion checkbox (unchecked ☐ / checked ☑) followed by the task name, with no visible task ID — rather than the block/box style used for scheduled entries.
- **FR-005**: The system MUST preserve all existing behavior for unscheduled tasks — selecting, completing/uncompleting, editing, scheduling (assigning a time), reordering within the list, and viewing details — with no functional change.
- **FR-006**: When a day has no unscheduled tasks, the system MUST omit the unscheduled list (including any header or separator) and render the day grid as it does today.
- **FR-007**: When an unscheduled task is scheduled (given a start time), the system MUST move it from the unscheduled list onto the grid; when a scheduled entry loses its time, it MUST return to the unscheduled list — matching current behavior.
- **FR-008**: When there are more unscheduled tasks than fit above the grid, the unscheduled list MUST grow to show all of them, taking vertical space from the day grid (the grid shrinks/scrolls accordingly); the layout MUST NOT break.
- **FR-009**: The system MUST NOT change the web app's Plan view as part of this feature.
- **FR-010**: The system MUST apply the existing visibility rules for completed unscheduled tasks (e.g., showing or hiding completed items) unchanged.

### Key Entities *(include if data involved)*

- **Unscheduled task**: A planned task for a given day that has no assigned start time. It is the item being re-presented as a list row. Attributes relevant to presentation: name, completion state, and its position/order among other unscheduled tasks.
- **Scheduled entry (block)**: A planned item that has a start time and occupies a slot on the day grid. Unchanged by this feature; referenced only to contrast with unscheduled tasks.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On the Plan tab, 100% of unscheduled tasks render as list rows (Tasks-tab style) and 0% render as time-block boxes.
- **SC-002**: The unscheduled list is positioned above the day grid and is visually distinguishable from it, confirmed by a viewer identifying the two regions without ambiguity.
- **SC-003**: Every unscheduled-task interaction available before the change (select, complete, edit, schedule, reorder, view details) produces the same outcome after the change — zero behavioral regressions.
- **SC-004**: For a day with no unscheduled tasks, the Plan view is unchanged from the pre-feature layout.
- **SC-005**: The web Plan view is unchanged by this feature.

## Assumptions

- "Displayed like tasks in the tasks tab" means adopting the Tasks-tab row presentation (a completion checkbox followed by the task name) for each unscheduled task; it does not require full parity with every Tasks-tab affordance (e.g., nested tree/subtask hierarchy) beyond what unscheduled tasks already support today.
- The unscheduled task list remains in its current location relative to the grid (above it); visual separation is provided by a horizontal divider line between the list and the grid.
- "Functionality shouldn't change" means the set of actions and their results stay identical; only the rendering of unscheduled tasks changes.
- Scope is limited to the TUI Plan tab. The Tasks tab, the day grid rendering of scheduled entries, and the web app are out of scope.
- Existing rules for whether completed unscheduled tasks are shown or hidden are retained as-is.
