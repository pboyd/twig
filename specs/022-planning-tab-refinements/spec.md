# Feature Specification: Planning Tab Refinements

**Feature Branch**: `022-planning-tab-refinements`

**Created**: 2026-05-29

**Status**: Draft

**Input**: User description: "Further refinements for the planning tab: (1) the selected-cell highlight should mirror the Tasks tab — set the cell background to the same blue, keep the text color, make the text bold, and only restyle the entry cell (not the hour marker or the hour-slot border); (2) editing or adding an entry should render the form in the right pane instead of a full-screen takeover, matching the Tasks tab; (3) `t` should add a task; (4) remove the `clear` action from the TUI; (5) merge rename and move into a single Edit form shown in the right pane, opened with Enter; (6) also use Enter (instead of `e`) to edit a task on the Tasks tab."

## Clarifications

### Session 2026-05-29

- Q: `t` is being reassigned from "jump to today" to "add task" on the Planning tab. What should happen to the jump-to-today action? → A: Rebind jump-to-today to `.`, next to the `[`/`]` prev/next-day keys; keep the feature.
- Q: The Tasks-tab selection highlight is bold + blue background + forced white text. Should the Planning selected cell keep the entry's existing text color, or match the Tasks tab exactly? → A: Match the Tasks tab exactly — reuse the same highlight style (bold text + blue accent background + white foreground), including overriding the entry's text color to white.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Edit a scheduled entry with one Enter-driven form (Priority: P1)

On the Planning tab a user selects a scheduled entry and presses Enter. A single Edit form opens in the right-hand pane (where the entry details normally show), letting the user change both the entry's name/label and its time window in one place. The user saves and the grid updates; pressing cancel discards changes and restores the details view.

**Why this priority**: Today the same intent is split across two separate prompts (`rename` and `move`) that each take over the screen. Merging them into one Enter-driven form is the headline interaction change of this round and removes redundant, hard-to-discover controls.

**Independent Test**: With an entry selected on the Planning grid, press Enter, confirm a single form appears in the right pane with fields for the entry's name and its time, change both, save, and confirm the grid reflects the new name and time. Testable with a stub plan service.

**Acceptance Scenarios**:

1. **Given** an entry is selected on the Planning grid, **When** the user presses Enter, **Then** a single Edit form opens in the right-hand pane (not a full-screen view) containing the entry's current name and time, with the grid still visible on the left.
2. **Given** the Edit form is open, **When** the user changes the name and/or the time and saves, **Then** the entry is updated on the grid to reflect both changes.
3. **Given** the Edit form is open, **When** the user cancels, **Then** no change is made and the right pane returns to showing the selected entry's details.
4. **Given** no entry is selected (empty grid or empty slot), **When** the user presses Enter, **Then** no Edit form opens and no error is shown.

---

### User Story 2 - Edit a task with Enter on the Tasks tab (Priority: P1)

On the Tasks tab a user moves the cursor to a task and presses Enter to open the edit form, instead of pressing `e`. The edit form behaves exactly as before; only the key that opens it changes.

**Why this priority**: Enter is the natural "open/act on the selected row" key, and aligning both tabs on Enter for editing gives the TUI one consistent editing gesture. It is a small, high-value consistency fix.

**Independent Test**: On the Tasks tab, select a task, press Enter, confirm the existing edit form opens; press `e` and confirm it no longer opens the edit form. Testable with a stub task list.

**Acceptance Scenarios**:

1. **Given** a task is selected on the Tasks tab, **When** the user presses Enter, **Then** the task edit form opens (the same form previously bound to `e`).
2. **Given** a task is selected on the Tasks tab, **When** the user presses `e`, **Then** the edit form does not open (the `e` binding for editing is removed).
3. **Given** the Tasks list is empty, **When** the user presses Enter, **Then** nothing happens and no error is shown.
4. **Given** the help screen is shown on the Tasks tab, **When** it lists how to edit, **Then** it advertises Enter (not `e`).

---

### User Story 3 - Add entries without a full-screen takeover (Priority: P2)

When the user adds a task or an event to the plan, the entry form appears in the right-hand pane beside the calendar grid, exactly like the Tasks tab shows its forms, rather than replacing the whole screen.

**Why this priority**: It removes a jarring inconsistency (the Planning tab is the only place a form blanks the screen) and keeps the day visible while the user fills in the form. It depends on the same pane-rendering work as US1.

**Independent Test**: On the Planning tab, trigger add-task and add-event, and confirm each form renders in the right pane with the grid still visible on the left, mirroring the Tasks tab. Testable with stub task/plan services.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active, **When** the user starts adding a task (including the task picker and the time-entry form), **Then** the form/picker renders in the right-hand pane with the calendar grid still shown on the left.
2. **Given** the Planning tab is active, **When** the user starts adding an event, **Then** the event form renders in the right-hand pane with the grid still shown on the left.
3. **Given** any Planning add/edit form is open in the right pane, **When** the user saves or cancels, **Then** the right pane returns to the entry-details view and the grid remains intact.
4. **Given** a narrow terminal, **When** a Planning form is shown in the right pane, **Then** the split degrades gracefully without corrupting the grid or form, consistent with the Tasks tab.

---

### User Story 4 - Planning selection highlight matches the Tasks tab (Priority: P2)

The selected entry on the Planning grid is highlighted the same way a selected task is on the Tasks tab: the cell reuses the exact same selection style — bold text on the blue accent background, with the same (white) foreground. Only the entry cell is restyled — the hour marker and the hour-slot border keep their normal styling.

**Why this priority**: The current monotone selection makes the Planning tab feel unfinished next to the Tasks tab; matching the highlight is the most visible polish in this request but does not block any function.

**Independent Test**: On the Planning grid, move the selection between entries and confirm the selected entry's cell reuses the Tasks-tab selection style (bold text, blue accent background, white foreground) and that the hour marker and slot border are not restyled. Verify against the Tasks tab's selection styling.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active with at least one entry, **When** an entry is selected, **Then** that entry's cell is rendered with the exact same selection style used for a selected task on the Tasks tab (bold text, blue accent background, white foreground).
2. **Given** an entry is selected, **When** it is rendered, **Then** its appearance is indistinguishable from a selected Tasks-tab row in color and weight.
3. **Given** an entry is selected, **When** the row is rendered, **Then** the hour marker (e.g. `07:00`) and the hour-slot border are not restyled by the selection.
4. **Given** the terminal does not support color, **When** the Planning grid renders, **Then** selection degrades to plain text consistent with the Tasks tab, with no broken escape sequences.

---

### User Story 5 - Add a task with `t` on the Planning tab (Priority: P3)

On the Planning tab the user presses `t` to add (schedule) a task, which feels more memorable than the previous key. The status bar and help screen advertise `t` for adding a task.

**Why this priority**: It is a discoverability tweak based on real use. It is valuable but small, and it requires reassigning the key currently used for the "jump to today" action.

**Independent Test**: On the Planning tab, press `t` and confirm the add-task flow begins; confirm the status bar/help advertise `t` for add task. Testable with a stub task list.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active, **When** the user presses `t`, **Then** the add-task flow begins (the task picker opens).
2. **Given** the Planning tab is active, **When** the status bar or help screen renders, **Then** `t` is advertised as "add task".
3. **Given** the Planning tab is active, **When** the user presses `.`, **Then** the grid jumps to today, and the help screen advertises `.` for "today".

---

### User Story 6 - Remove the clear action from the TUI (Priority: P3)

The `clear` action (clearing scheduled entries from a time onward) is removed from the Planning tab. Users delete entries individually instead. The status bar and help no longer mention clear.

**Why this priority**: The maintainer judged clear unnecessary in the TUI given how easy it is to delete entries directly; removing it simplifies the surface. It is the lowest-impact change.

**Independent Test**: On the Planning tab, confirm the clear key does nothing and that neither the status bar nor the help screen advertises a clear action, while per-entry delete still works. Testable with a stub plan service.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active, **When** the user presses the key formerly bound to clear, **Then** no clear action runs (the binding is removed).
2. **Given** the Planning tab is active, **When** the help screen and status bar render, **Then** they do not mention a clear action.
3. **Given** the Planning tab is active, **When** the user removes a single entry, **Then** per-entry delete still works as before.

---

### Edge Cases

- **Enter on an empty slot or empty grid (Planning)**: no Edit form opens and no error is shown.
- **Enter on an empty Tasks list**: nothing happens; no edit form, no error.
- **Editing a task-linked entry vs an event**: the merged Edit form exposes the fields appropriate to the entry type (e.g. an event has its own name; a task-linked entry edits its label/time) without showing misleading empty fields.
- **Cancel mid-edit**: cancelling any Planning add/edit form restores the right-pane details view with the prior selection intact.
- **Narrow terminal with a right-pane form**: the grid/form split collapses gracefully rather than overflowing, consistent with the Tasks tab.
- **Non-color terminal**: the selection highlight and right-pane forms degrade to plain text with no broken escape sequences.
- **Key reassignment collisions**: after reassigning `t` to add-task and removing clear, no two advertised Planning actions share a key.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Planning tab MUST provide a single Edit form that lets the user change both a selected entry's name/label and its time window, replacing the previously separate rename and move prompts.
- **FR-002**: Pressing Enter on the Planning tab with an entry selected MUST open the Edit form for that entry; pressing Enter with no entry selected MUST do nothing and show no error.
- **FR-003**: The Planning Edit form, the add-task flow (picker and time-entry form), and the add-event form MUST render in the right-hand pane beside the calendar grid, with the grid remaining visible, mirroring the Tasks tab's form layout. None of these may take over the full screen.
- **FR-004**: Saving a Planning add/edit form MUST apply the change and return the right pane to the entry-details view; cancelling MUST discard changes and return to the details view with the prior selection intact.
- **FR-005**: On the Tasks tab, pressing Enter on a selected task MUST open the task edit form, and the previous `e` binding for editing MUST be removed.
- **FR-006**: The selected entry's cell on the Planning grid MUST be highlighted by reusing the exact same selection style as a selected task on the Tasks tab (bold text on the blue accent background, with the same white foreground).
- **FR-007**: The Planning selection highlight MUST restyle only the entry cell; the hour marker and the hour-slot border MUST NOT be restyled by selection.
- **FR-008**: The Planning tab MUST bind `t` to the add-task action, and the status bar and help screen MUST advertise `t` for "add task".
- **FR-009**: The "jump to today" action MUST be rebound from `t` to `.`, and the help screen MUST advertise `.` for "today"; no advertised Planning action may share a key with another.
- **FR-010**: The `clear` action MUST be removed from the Planning tab: its key MUST no longer trigger clear, and neither the status bar nor the help screen may mention it. Per-entry deletion MUST remain available.
- **FR-011**: All advertised Planning and Tasks key bindings MUST remain mutually non-conflicting after the reassignments in this feature.
- **FR-012**: When the terminal does not support color, the Planning selection highlight and right-pane forms MUST degrade to plain text with no broken escape sequences, consistent with the Tasks tab.

### Key Entities

- **Plan entry**: a scheduled item on the day grid — either a task-linked entry (references a task, has a label and a time window) or an event (has its own name and time window). The Edit form acts on a single selected entry.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can edit a scheduled entry's name and time in a single form opened with one keypress (Enter); the edit gesture is Enter on both the Planning and Tasks tabs.
- **SC-002**: 100% of Planning add/edit interactions (add task, add event, edit entry) render beside the grid in the right pane; none blank the screen.
- **SC-003**: The Planning grid's selected-entry highlight is visually indistinguishable in color and weight from the Tasks tab's selected-row highlight, and applies only to the entry cell.
- **SC-004**: After the changes, every action advertised in the Planning status bar and help screen maps to a working key, with no duplicate key advertised for two different actions.
- **SC-005**: The clear action is absent from the Planning tab's status bar, help screen, and behavior, while per-entry delete still succeeds.

## Assumptions

- "Add a task" with `t` applies to the Planning tab's schedule-a-task action (the bullet is part of the Planning-tab refinements); the Tasks tab's create-task keys (`n` new subtask, `ctrl+n` new root) are unchanged.
- The merged Edit form replaces the separate rename (`r`) and move (`m`) entry prompts; those letter keys are freed and need not be rebound to other actions.
- The add-event key (`e`) on the Planning tab is unaffected by removing `e`-to-edit on the Tasks tab (the two tabs maintain independent bindings).
- The two-pane Planning layout from the prior feature (021) is the baseline; this feature reuses that right pane to host forms.
- The Planning selection reuses the existing Tasks-tab selection style already defined in the TUI theme (bold + blue accent background + white foreground); no new color or style is introduced.
