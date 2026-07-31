# Feature Specification: Task Descriptions on the Planning Tab

**Feature Branch**: `071-plan-task-descriptions`

**Created**: 2026-07-31

**Status**: Draft

**Input**: User description: "Show task descriptions on the TUI planning tab. Currently, the description is only shown on the Tasks tab, but it would be useful on the Planning tab. Note that the description contains markdown which should be rendered (same as elsewhere)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read a scheduled task's description without leaving the plan (Priority: P1)

While working through the day's plan, a person selects a scheduled entry and wants to recall what the task actually involves — the checklist, the link, the acceptance notes they wrote when creating it. Today that information lives only on the Tasks tab, so they must leave the plan, find the task in the tree, read it, and navigate back. With this feature, selecting a plan entry that is linked to a task shows that task's description in the entry's detail view, alongside the window, duration, and status already shown.

**Why this priority**: This is the entire point of the feature. Without it, nothing changes for the user. It is also self-contained: showing the description delivers the full value even if no other story ships.

**Independent Test**: Schedule a task that has a non-empty description, select that entry on the Planning tab, and confirm the description text appears in the detail view without leaving the tab.

**Acceptance Scenarios**:

1. **Given** a plan entry linked to a task whose description is non-empty, **When** the entry is selected on the Planning tab, **Then** the description is shown in the entry's detail view below the existing entry fields.
2. **Given** a plan entry linked to a task whose description is empty, **When** the entry is selected, **Then** no description text, label, or reserved blank area is shown, and the detail view looks exactly as it does today.
3. **Given** a plan entry that is an event rather than a task, **When** the entry is selected, **Then** no description is shown, because events carry no task description.
4. **Given** the selection moves from one entry to another, **When** the new entry is selected, **Then** the detail view shows the newly selected entry's description (or none), never the previous entry's.
5. **Given** no entry is selected (empty day), **When** the Planning tab is viewed, **Then** the existing placeholder is shown unchanged.

---

### User Story 2 - Description renders as formatted text, not raw markup (Priority: P1)

Descriptions are authored in markdown. A person reading a description on the Planning tab expects headings, lists, emphasis, links, and code to appear the way they do everywhere else in the application — not as raw `**asterisks**` and `- dashes`.

**Why this priority**: Shipping story 1 with unformatted markup would be a visible regression against how descriptions look on the Tasks tab, and the user called this out explicitly. It is inseparable in practice from story 1, but it is independently testable.

**Independent Test**: Give a task a description containing a heading, a bullet list, bold text, and inline code; select its plan entry; confirm the rendering matches what the same description produces on the Tasks tab.

**Acceptance Scenarios**:

1. **Given** a description containing markdown constructs (headings, lists, emphasis, inline code, links), **When** it is shown on the Planning tab, **Then** it is rendered with the same formatting rules and visual treatment used for descriptions on the Tasks tab.
2. **Given** the same description and the same available text width, **When** it is shown on the Planning tab and on the Tasks tab, **Then** the rendered output is equivalent.
3. **Given** the application is running without visual styling (plain-text output mode), **When** a description is shown on the Planning tab, **Then** it is presented as readable wrapped plain text consistent with the plain-text presentation used elsewhere.

---

### User Story 3 - Description fits the available space without breaking the layout (Priority: P2)

The Planning tab shows the calendar grid and the entry details side by side. A long description must not push the grid around, overflow into the neighbouring pane, or leave the layout misaligned.

**Why this priority**: A correct but layout-breaking implementation would make the Planning tab unusable, so this matters — but it is a robustness concern on top of the core value delivered by stories 1 and 2.

**Independent Test**: Select an entry whose task has a description far longer and wider than the detail area, and confirm the pane borders, the grid, and the status line all remain intact.

**Acceptance Scenarios**:

1. **Given** a description longer than the vertical space available in the detail area, **When** the entry is selected, **Then** the visible portion is shown, the surrounding layout stays intact, and no content spills outside the detail area.
2. **Given** a description with lines wider than the detail area, **When** the entry is selected, **Then** the text wraps within the available width and does not bleed into the calendar grid.
3. **Given** a narrow terminal, **When** an entry with a description is selected, **Then** the layout remains readable and undamaged.

---

### Edge Cases

- **Entry with no linked task (event)**: no description section is rendered.
- **Linked task not present in the loaded task data** (e.g. hidden by a filter, or not yet loaded): the entry's other detail fields still render normally and the description section is simply absent — no error, no placeholder.
- **Description that is only whitespace**: treated the same as empty; nothing is shown.
- **Description containing malformed or unsupported markdown**: it is displayed as text rather than causing an error or blank pane.
- **Very long single word or URL with no break opportunity**: does not cause the detail area to exceed its width.
- **Description edited elsewhere in the application**: the Planning tab reflects the current description the next time the entry's details are drawn from refreshed data, using the same freshness behaviour as the other task-derived fields already shown on this tab (such as the pomodoro row).
- **Detail area of zero or near-zero width**: renders without error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Planning tab's entry detail view MUST display the description of the task linked to the selected plan entry.
- **FR-002**: The description MUST be rendered as formatted markdown using the same rendering behaviour applied to task descriptions elsewhere in the interface, including the plain-text fallback when visual styling is unavailable.
- **FR-003**: The description MUST be omitted entirely — with no label, heading, or reserved blank space — when the linked task has no description, when the description is only whitespace, when the entry is an event with no linked task, or when the linked task's data is unavailable.
- **FR-004**: The description MUST appear after the entry fields the detail view already shows (name, window, duration, task status, pomodoro progress), separated from them so it reads as a distinct block.
- **FR-005**: The description MUST be wrapped to the width available in the detail area and MUST NOT render outside that area's horizontal or vertical bounds.
- **FR-006**: When the description exceeds the vertical space available, the detail view MUST show as much as fits and MUST NOT distort the calendar grid, pane borders, tab bar, or status line.
- **FR-007**: The detail view MUST always show the description belonging to the currently selected entry, updating as the selection changes.
- **FR-008**: The description MUST be read-only in this view; the Planning tab MUST NOT gain any description editing capability.
- **FR-009**: All existing Planning tab behaviour — entry selection, scheduling, editing, event creation, the task picker, and the calendar grid — MUST be unchanged.
- **FR-010**: The description MUST NOT be shown while a picker or entry form occupies the detail area; it appears only in the read-only entry detail view.

### Key Entities

- **Plan entry**: a scheduled item on a day's plan. Either linked to a task or a standalone event. Already carries name, start time, duration, and completion status.
- **Task**: the work item a plan entry may link to. Already carries a name, a markdown description, a pomodoro estimate, and completion state. The description is the field this feature surfaces.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A person can read the full context of a scheduled task's description without leaving the Planning tab, eliminating the tab switch and task lookup previously required — a reduction from at least three navigation actions to zero.
- **SC-002**: For every scheduled task with a description, selecting its plan entry shows that description; 100% of such entries display it and 0% of entries without a description display any related empty space.
- **SC-003**: Descriptions containing headings, lists, emphasis, inline code, and links display identically on the Planning tab and the Tasks tab at equal width in 100% of tested cases.
- **SC-004**: With descriptions of any length or width, the Planning tab layout remains intact — grid alignment, pane boundaries, and status line unaffected in 100% of tested cases, including the narrowest supported terminal size.
- **SC-005**: Adding the description introduces no perceptible delay when moving the selection between plan entries.

## Assumptions

- The Planning tab's existing read-only entry detail view is the correct place for the description. It already presents per-entry information beside the calendar grid, so no new pane, popup, or toggle is introduced, and the calendar grid rows themselves are left untouched.
- Descriptions belong to tasks, not to plan entries. Events have no description, and none is invented for them.
- The description is placed at the end of the detail view because it is the only free-form, variable-length field; the short labelled fields stay together above it.
- Overflow is handled by showing what fits, consistent with how the detail views elsewhere in the interface already behave. Adding scrolling to the detail area is out of scope.
- Rendering reuses the interface's existing markdown presentation rather than defining new formatting, so the Planning tab automatically inherits any future changes to how descriptions are displayed.
- The task data already loaded to support the detail view's existing task-derived fields is sufficient to supply the description; no new data source is assumed.
- Web and CLI surfaces are out of scope. This feature concerns the terminal interface's Planning tab only.
