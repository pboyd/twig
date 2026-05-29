# Feature Specification: Planning Tab Polish

**Feature Branch**: `021-planning-tab-polish`

**Created**: 2026-05-29

**Status**: Draft

**Input**: User description: "The new planning tab is functional, but it has some consistency issues with the rest of the TUI. A few simple misses from the spec (pomodoro can't be cancelled from the Planning tab; no help screen for the Planning tab; the add-task dialog doesn't show sub-tasks the way the move-task dialog does). And the two tabs have a very different feel: Tasks has a blue color scheme while Planning is monotone; Tasks has a 2-pane view (list left, details right) that Planning could adopt; the Planning status line isn't pinned to the bottom; and the tab bar is less polished than the Tasks screen, including a redundant doubled \"Tasks\" word from an unneeded panel title."

## Clarifications

### Session 2026-05-29

- Q: In the Planning tab's two-pane layout, what should the right-hand details pane do for the selected entry? → A: Read-only display (name, time window, linked task, completion); editing stays via the existing rename/move prompts, mirroring the Tasks tab's details pane.
- Q: Beyond the Planning tab's redundant title, should the Tasks tab's in-pane "Tasks" title also be removed? → A: Remove the redundant in-pane title on both tabs; the tab bar already names each view.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cancel a running pomodoro from the Planning tab (Priority: P1)

A user has a pomodoro running and switches to the Planning tab to look at their day. The shared status bar shows the running timer and advertises that `x` cancels it. The user presses the cancel key and the pomodoro stops, exactly as it would from the Tasks tab.

**Why this priority**: This is a broken, advertised control — the status bar tells the user the key works but it does nothing. Restoring it fixes a correctness bug and an explicit parity guarantee from the original Planning tab spec (the shared status bar and its controls must function identically on both tabs).

**Independent Test**: Start a pomodoro, switch to the Planning tab, press the cancel key, and confirm the pomodoro ends and the status bar updates. Fully testable with a stub pomodoro service.

**Acceptance Scenarios**:

1. **Given** a pomodoro is running and the Planning tab is active, **When** the user presses the cancel key shown in the status bar, **Then** the pomodoro is cancelled and the status bar reflects that no pomodoro is running.
2. **Given** no pomodoro is running and the Planning tab is active, **When** the user presses the cancel key, **Then** nothing is cancelled and no error is shown (the key is inert, matching the Tasks tab).
3. **Given** a pomodoro is running, **When** the user cancels it from the Planning tab and switches to the Tasks tab, **Then** the Tasks tab also shows no running pomodoro.

---

### User Story 2 - Get help on the Planning tab (Priority: P1)

A user on the Planning tab presses the help key. A help screen appears listing the Planning tab's keys and actions, in the same style as the Tasks tab's help screen. The user dismisses it and returns to the grid.

**Why this priority**: The status bar already advertises `? help`, but there is no Planning help screen, so the advertised key is broken. Help is the primary way a user discovers the Planning tab's actions, and its absence is an explicit gap against the parity guarantee.

**Independent Test**: On the Planning tab, press the help key, confirm a help screen listing Planning actions appears, dismiss it, and confirm the grid returns unchanged. Testable in isolation.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active, **When** the user presses the help key, **Then** a help screen appears describing the Planning tab's available keys and actions.
2. **Given** the Planning help screen is open, **When** the user presses the dismiss/help key, **Then** the help screen closes and the Planning grid returns with its prior selection intact.
3. **Given** either tab is active, **When** the user opens help, **Then** the help screen's visual style (framing, layout, dismissal) matches the other tab's help screen.

---

### User Story 3 - See sub-tasks when scheduling a task (Priority: P2)

From the Planning tab the user presses add-task to schedule an existing task. The picker shows the user's tasks as a tree, including nested sub-tasks, identical to the way the move-task dialog on the Tasks tab presents them. The user can find and select a sub-task, not just top-level tasks.

**Why this priority**: Sub-tasks are common scheduling targets, and they are currently invisible in the add-task picker, so a whole class of tasks can't be planned. The move-task dialog already renders the task tree correctly, so this is a consistency fix that should reuse existing presentation rather than diverge.

**Independent Test**: With a task that has sub-tasks, open the Planning add-task picker and confirm the sub-tasks appear nested under their parent (matching the move-task dialog), and that a sub-task can be selected and scheduled. Testable with a stub task list.

**Acceptance Scenarios**:

1. **Given** the user has tasks with sub-tasks, **When** they open the add-task picker on the Planning tab, **Then** sub-tasks are displayed nested under their parents, consistent with the move-task dialog.
2. **Given** the add-task picker is open, **When** the user selects a sub-task and provides a time, **Then** an entry for that sub-task is scheduled on the grid.
3. **Given** the move-task dialog and the add-task picker are both shown the same task tree, **When** compared, **Then** they present the hierarchy in the same way (no divergent formatting or omitted levels).

---

### User Story 4 - Planning tab matches the Tasks tab's look and feel (Priority: P2)

A user moving between the Tasks and Planning tabs experiences one coherent application rather than two differently-styled screens. The Planning tab uses the same blue accent color scheme as the Tasks tab. The tab bar is polished and clearly indicates the active tab. The redundant panel title is gone (the tab bar already names the view, so the duplicated word is removed). The status line (help text) is pinned to the bottom of the screen, exactly as on the Tasks tab.

**Why this priority**: Visual inconsistency makes the Planning tab feel unfinished and unrelated to the rest of the TUI. It does not block functionality, so it ranks below the broken-control fixes, but it is the headline of the request and strongly affects perceived quality.

**Independent Test**: Switch between tabs and confirm the Planning tab uses the same color accents, a polished active-tab indicator, no duplicated/redundant panel title, and a bottom-pinned status line. Verifiable by visual comparison against the Tasks tab.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active, **When** it renders, **Then** it uses the same accent color scheme (e.g. blue highlights, selection styling) as the Tasks tab rather than a monotone palette.
2. **Given** either tab is active, **When** the tab bar renders, **Then** the active tab is clearly and consistently indicated, and the bar is styled to the same standard as the Tasks tab.
3. **Given** the Planning tab is active, **When** it renders, **Then** the previously duplicated panel title is removed so the view is not labeled redundantly next to the tab bar.
4. **Given** the Planning tab is active at any supported terminal height, **When** it renders, **Then** the status line appears pinned to the bottom row of the screen, matching the Tasks tab.

---

### User Story 5 - Review entry details beside the plan (Priority: P3)

The user wants to see details about a scheduled entry without losing sight of the whole day. The Planning tab adopts a two-pane layout like the Tasks tab: the calendar grid on the left and a details pane on the right showing the currently selected entry's information. As the user moves the selection on the grid, the details pane updates to describe that entry.

**Why this priority**: The two-pane layout improves situational awareness and brings the Planning tab in line with the Tasks tab's interaction model, but the Planning tab is fully usable without it. It is the largest change and depends on the shared styling work, so it is ranked last.

**Independent Test**: On the Planning tab, move the selection between entries and confirm a right-hand details pane updates to describe the selected entry while the grid stays visible on the left. Testable with a stub plan service.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active with entries on the grid, **When** it renders, **Then** the calendar grid occupies a left pane and a details pane occupies the right, consistent with the Tasks tab's split layout.
2. **Given** the two-pane Planning layout, **When** the user moves the selection to a different entry, **Then** the right pane updates to show that entry's details (name, time window, and—for a task-linked entry—its linked task and completion state).
3. **Given** the in-view day has no entries, **When** the Planning tab renders, **Then** the details pane shows an empty/placeholder state without error, consistent with the Tasks tab's empty details behavior.
4. **Given** the two-pane layout is active, **When** the terminal is narrow, **Then** the layout degrades gracefully without corrupting the grid or details, consistent with the existing TUI's narrow-terminal handling.

---

### Edge Cases

- **Cancel key with no pomodoro**: pressing cancel on the Planning tab with nothing running is inert (no error), matching the Tasks tab.
- **Help opened mid-context**: opening help while an entry is selected returns to the same selection on dismissal; help cannot be opened on top of an open modal/prompt if the Tasks tab also forbids it (behavior matches Tasks).
- **Add-task picker with deeply nested sub-tasks**: nesting renders consistently with the move-task dialog at the same depth, without truncating levels differently.
- **Add-task picker when the user has no tasks**: the picker shows an empty state that can be cancelled, unchanged from current behavior.
- **Narrow terminal with two panes**: the grid/details split collapses gracefully (consistent with the Tasks tab) rather than overflowing.
- **Selection details for an event vs a task entry**: an event entry's details omit task-only fields (no linked task / completion state) rather than showing blank or misleading task fields.
- **Theme on a non-color terminal**: when ANSI styling is unavailable, the Planning tab degrades to plain text just as the Tasks tab does, with no broken escape sequences.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The pomodoro-cancel key advertised in the shared status bar MUST cancel a running pomodoro while the Planning tab is active, with the same effect as cancelling from the Tasks tab.
- **FR-002**: When no pomodoro is running, the cancel key on the Planning tab MUST be inert (no cancellation, no error), matching the Tasks tab.
- **FR-003**: The Planning tab MUST provide a help screen, reachable via the same help key advertised in the status bar, that lists the Planning tab's keys and actions.
- **FR-004**: The Planning help screen MUST match the Tasks tab help screen in visual style and dismissal behavior, and dismissing it MUST return the user to the Planning grid with prior selection intact.
- **FR-005**: The add-task picker on the Planning tab MUST display the user's tasks as a hierarchy including nested sub-tasks, presented consistently with the move-task dialog on the Tasks tab.
- **FR-006**: The add-task picker MUST allow selecting a sub-task (not only top-level tasks) and scheduling it onto the in-view day. Where practical, it MUST reuse the move-task dialog's task-tree presentation rather than introduce a separate one.
- **FR-007**: The Planning tab MUST use the same accent color scheme as the Tasks tab (e.g. blue highlights and selection styling) instead of a monotone palette.
- **FR-008**: The tab bar MUST be styled to the same standard as the Tasks tab and MUST clearly indicate the active tab.
- **FR-009**: The redundant in-pane panel title MUST be removed on BOTH the Planning and Tasks tabs, so neither view is labeled twice next to the tab bar that already names it.
- **FR-010**: The Planning tab's status line (help text) MUST be pinned to the bottom row of the screen at all supported terminal heights, matching the Tasks tab.
- **FR-011**: The Planning tab MUST present a two-pane layout — the calendar grid on the left and a details pane on the right — consistent with the Tasks tab's split layout.
- **FR-012**: The Planning details pane MUST display (read-only) the currently selected entry's details (at minimum its name and time window; for a task-linked entry, its linked task and completion state) and MUST update as the selection changes. Editing remains via the existing rename/move prompts; the details pane is not an inline editor.
- **FR-013**: When the in-view day has no entries, the Planning details pane MUST show an empty/placeholder state without error, consistent with the Tasks tab.
- **FR-014**: All Planning tab styling and layout changes MUST degrade gracefully on narrow terminals and on terminals without ANSI styling, consistent with the existing TUI.
- **FR-015**: All previously specified Planning tab behaviors (grid rendering, now-marker, add/edit/remove/clear, day navigation, refresh, completion strike-through) MUST continue to function unchanged; this feature refines presentation and parity, not core planning behavior.

### Key Entities

- **Plan entry**: an item on the day's grid — either a scheduled task (linked to a task, can be complete/incomplete) or an event (free-text, never complete). Carries a name and a time window. The details pane describes this entity.
- **Task tree node**: a task and its nested sub-tasks as shown in the add-task picker, mirroring the move-task dialog's hierarchy.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of controls advertised in the Planning tab's status bar (including pomodoro-cancel and help) perform their advertised action.
- **SC-002**: A user can cancel a running pomodoro from the Planning tab in a single keypress, with the same result as from the Tasks tab.
- **SC-003**: A user can locate and schedule any sub-task from the add-task picker, with sub-tasks visible at the same nesting depth shown by the move-task dialog.
- **SC-004**: In a side-by-side comparison, reviewers judge the Planning and Tasks tabs to share one coherent visual style (color scheme, tab bar, bottom-pinned status line) rather than two distinct looks.
- **SC-005**: No view in the TUI labels itself redundantly (e.g. a panel title duplicating the tab name).
- **SC-006**: While viewing the Planning tab, a user can read the selected entry's details without the calendar grid leaving the screen.
- **SC-007**: All existing Planning tab acceptance scenarios from the prior feature continue to pass after this work (no functional regressions).

## Assumptions

- This feature extends the existing Planning tab (feature 020); the planning data model, server interactions, and core actions are already in place and are not changing.
- The Tasks tab is the reference for "consistent" styling and layout: where this spec says "matching the Tasks tab," the Tasks tab's current behavior defines the target.
- The Planning details pane is read-only display in this feature (mirroring the Tasks details pane); existing edit flows (rename, move, etc.) remain via their current prompts and are not moved into the details pane.
- Removing the redundant panel title does not remove the day label/date header, which conveys distinct information (the in-view day), not a redundant view name.
- "Blue color scheme" refers to reusing the TUI's existing accent/theme tokens already applied on the Tasks tab, not introducing a new palette.
- Keybindings for cancel and help are the existing shared bindings already shown in the status bar; this feature does not introduce new keys.
