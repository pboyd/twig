# Feature Specification: Change Task Parent in TUI

**Feature Branch**: `017-tui-move-task`

**Created**: 2026-05-28

**Status**: Draft

**Input**: User description: "changing a task's parent in the TUI

Pressing `m` (for move) in the TUI should bring up a dialog box listing incomplete tasks. The user can select a new parent task from the list and save it. The existing parent task should be pre-selected."

## Clarifications

### Session 2026-05-28

- Q: How are candidate parent tasks arranged in the move dialog? → A: Hierarchical tree (indented by parent/child), matching the main TUI
- Q: Which tasks are listed as candidate parents — all incomplete tasks, or only those visible in the current TUI view? → A: All incomplete tasks across the user's account
- Q: Which keys confirm and cancel the move dialog? → A: Enter confirms, Esc cancels

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Move a task under a new parent (Priority: P1)

While viewing the task list in the TUI, the user wants to reorganize their work by changing which task is the parent of a currently selected task. They press `m` to open a move dialog, browse the list of incomplete tasks, choose the new parent, and confirm — the task is reparented and the list re-renders to reflect the new hierarchy.

**Why this priority**: This is the core capability of the feature; without it the user has to leave the TUI (or fall back to deleting/recreating) to restructure their task tree.

**Independent Test**: With a task tree containing at least two incomplete tasks, select a child, press `m`, pick a different incomplete task in the dialog, confirm, and verify the child now appears under the chosen parent in the TUI and persists after refresh.

**Acceptance Scenarios**:

1. **Given** a task with an existing parent is selected in the TUI, **When** the user presses `m`, **Then** a dialog opens listing the incomplete tasks and the current parent is highlighted as pre-selected.
2. **Given** the move dialog is open with the current parent pre-selected, **When** the user navigates to a different incomplete task and confirms, **Then** the dialog closes and the selected task is now displayed as a child of the newly chosen parent.
3. **Given** the move dialog is open, **When** the user cancels (without confirming), **Then** the dialog closes and the task's parent is unchanged.

---

### User Story 2 - Promote a task to top-level (Priority: P2)

A user has a sub-task that should no longer be nested under another task — they want to promote it to a top-level task. They press `m`, pick the "no parent" option in the dialog, and confirm.

**Why this priority**: A natural counterpart to reparenting; without it, users can move tasks sideways but cannot detach them from a parent without leaving the TUI.

**Independent Test**: Select a child task, press `m`, choose the "no parent / top-level" entry in the dialog, confirm, and verify the task now appears at the root of the list.

**Acceptance Scenarios**:

1. **Given** a child task is selected, **When** the user opens the move dialog and chooses the top-level/no-parent option, **Then** after confirming, the task appears at the root of the task tree.
2. **Given** a task that is already top-level is selected, **When** the user opens the move dialog, **Then** the top-level/no-parent option is pre-selected.

---

### Edge Cases

- What happens when the user tries to set a task's parent to itself or to one of its own descendants? The system must reject this to prevent cycles, and surface a clear in-dialog message rather than silently failing.
- What happens when the only incomplete task is the one being moved? The dialog still opens and offers the "no parent" option; no other parent choices are listed.
- What happens when the parent list is long? The dialog must be scrollable / navigable from the keyboard so any incomplete task can be reached.
- What happens when the user presses `m` while no task is selected? The dialog does not open and the TUI state is unchanged.
- What happens when the save fails (e.g., the server rejects the update)? The dialog surfaces an error and leaves the task's parent unchanged; the user can retry or cancel.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The TUI MUST bind the `m` key, while a task is selected, to open a "move task" dialog for that task.
- **FR-002**: The move dialog MUST display all incomplete tasks across the user's account as candidate parents (independent of any filter applied to the main TUI view), plus an explicit entry representing "no parent" (top-level).
- **FR-002a**: Candidate parents MUST be rendered as a hierarchical tree (indented to show parent/child relationships), matching how the main TUI presents the task list.
- **FR-003**: The move dialog MUST pre-select the task's current parent on open; if the task is currently top-level, the "no parent" entry MUST be pre-selected.
- **FR-004**: The dialog MUST allow keyboard navigation through the candidate list and confirm the selection when the user presses Enter.
- **FR-005**: The dialog MUST allow the user to cancel without changing the task's parent by pressing Esc.
- **FR-006**: On confirm, the system MUST persist the new parent so the change survives restarting the TUI.
- **FR-007**: The system MUST prevent reparenting a task to itself or to any of its descendants, and MUST surface the rejection to the user within the dialog without losing their selection state.
- **FR-008**: The task being moved MUST NOT appear as a selectable parent candidate in the dialog.
- **FR-009**: Completed tasks MUST NOT appear in the candidate-parent list.
- **FR-010**: After a successful move, the TUI MUST re-render the task list so the moved task appears under its new parent (or at the root) without requiring a manual refresh.

### Key Entities

- **Task**: The unit being reparented. Relevant attributes for this feature: identity, completion status, and a reference to its parent task (which may be empty for top-level tasks).
- **Move dialog selection**: A transient choice representing the prospective new parent for the task being moved; resolves to either an existing incomplete task or the "no parent" sentinel.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can reparent a selected task to any other incomplete task in under 10 seconds without leaving the TUI.
- **SC-002**: 100% of attempts to create a parent cycle (assigning a task to itself or one of its descendants) are rejected with a visible message and leave the task's parent unchanged.
- **SC-003**: After confirming a move, the new parent relationship is visible in the TUI immediately and persists across TUI restarts in 100% of cases.
- **SC-004**: When the move dialog opens, the currently effective parent (or "no parent" if top-level) is pre-selected in 100% of cases.

## Assumptions

- The TUI already has a notion of a currently selected task; `m` operates on that selection.
- "Incomplete tasks" means tasks not marked complete, consistent with the existing completion model used elsewhere in the TUI.
- The candidate-parent list is drawn from the same task set the TUI already loads; no new server endpoint is assumed beyond what is needed to update a task's parent.
- Mouse interaction is out of scope; the dialog is keyboard-driven, matching the rest of the TUI.
- Sorting/grouping of candidate parents in the dialog follows the same ordering the TUI uses elsewhere when listing tasks (within the hierarchical tree layout).
