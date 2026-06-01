# Feature Specification: Hide Completed Tasks

**Feature Branch**: `026-hide-completed-tasks`

**Created**: 2026-05-31

**Status**: Draft

**Input**: User description: "users of the web app need a way to hide completed tasks. By default, completed tasks should not be shown, but the user should be able to show them when needed."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Completed tasks hidden by default (Priority: P1)

When a user opens the task list in the web app, tasks they have already completed are not shown. The view focuses on outstanding work, so the user sees only what still needs doing without manually scrolling past finished items.

**Why this priority**: This is the core value of the feature — a focused, uncluttered task list. It delivers the primary benefit on its own, even before any toggle exists, because the most common need is simply not seeing finished work.

**Independent Test**: With a mix of completed and incomplete tasks present, load the task list and confirm only incomplete tasks appear. Can be fully tested by completing a task and observing it disappear from the default view.

**Acceptance Scenarios**:

1. **Given** a task list containing both completed and incomplete tasks, **When** the user opens the task list, **Then** only the incomplete tasks are displayed.
2. **Given** the default view is showing, **When** the user marks an incomplete task as complete, **Then** that task is removed from the view.
3. **Given** an incomplete parent task with both completed and incomplete sub-tasks, **When** the user views the list, **Then** the parent and its incomplete sub-tasks remain visible while the completed sub-tasks are hidden.

---

### User Story 2 - Reveal completed tasks on demand (Priority: P2)

A user occasionally needs to see what they have already finished — to review progress, reopen a task, or confirm something was done. The user can toggle the view to show completed tasks alongside incomplete ones, then return to the focused default view.

**Why this priority**: Hiding completed work is only safe and useful if the user can retrieve it when needed. Without this, completed tasks would feel lost. It builds directly on P1 but is secondary to simply decluttering the default view.

**Independent Test**: With completed tasks hidden, activate the "show completed" control and confirm completed tasks appear; deactivate it and confirm they are hidden again.

**Acceptance Scenarios**:

1. **Given** completed tasks are hidden, **When** the user activates the show-completed control, **Then** completed tasks become visible alongside incomplete tasks.
2. **Given** completed tasks are visible, **When** the user deactivates the show-completed control, **Then** completed tasks are hidden again and only incomplete tasks remain.
3. **Given** completed tasks are visible, **When** the user views a completed task, **Then** it is clearly distinguishable from incomplete tasks (e.g., visually marked as done).
4. **Given** completed tasks are visible, **When** the user reopens (uncompletes) a task, **Then** the task remains visible and is shown as incomplete.

---

### User Story 3 - Preference is remembered (Priority: P3)

A user who chooses to show completed tasks does not have to re-enable it every time they reload the page or return later in the same browser. Their last show/hide choice is remembered, with hidden-by-default applying to a first-time user.

**Why this priority**: A quality-of-life improvement that reduces repetitive interaction. The feature is fully functional without it, so it is the lowest priority.

**Independent Test**: Activate show-completed, reload the page, and confirm completed tasks are still shown. Reset the preference (or use a fresh browser session) and confirm the default is hidden.

**Acceptance Scenarios**:

1. **Given** the user has set the view to show completed tasks, **When** they reload the page or reopen the app in the same browser, **Then** completed tasks are still shown.
2. **Given** a first-time user with no saved preference, **When** they open the task list, **Then** completed tasks are hidden by default.

---

### Edge Cases

- **All tasks complete**: When every task is completed and the view is in the default (hide) state, the list appears empty. The interface MUST distinguish this from a genuinely empty list so the user understands completed tasks exist but are hidden, and how to reveal them.
- **Completed parent of completed sub-tasks**: Because a parent can only be completed after all its sub-tasks are complete, hiding completed tasks hides the entire completed subtree (parent and descendants together).
- **Completing the last visible task**: When the user completes a task in the default view and it disappears, the user MUST still have an obvious way to access the show-completed control (it does not vanish along with the tasks).
- **No completed tasks exist**: The show/hide control behaves consistently and does not error; toggling it simply has no visible effect.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The task list MUST hide completed tasks by default when first shown to a user with no saved preference.
- **FR-002**: Users MUST be able to toggle the task list between hiding and showing completed tasks via a clearly labeled control.
- **FR-003**: When completed tasks are hidden, the system MUST continue to display all incomplete tasks, including incomplete tasks whose sibling tasks are completed.
- **FR-004**: When an incomplete parent task has completed sub-tasks, the system MUST keep the parent and its incomplete sub-tasks visible while hiding only the completed sub-tasks (in the hide state).
- **FR-005**: When completed tasks are shown, the system MUST visually distinguish completed tasks from incomplete tasks.
- **FR-006**: When a task's completion state changes (completed or reopened), the task list MUST update to reflect the current show/hide setting without requiring a manual page reload.
- **FR-007**: The system MUST remember the user's most recent show/hide choice for the duration it would normally retain such preferences in the same browser, and apply it on subsequent visits.
- **FR-008**: When the default (hide) view results in no visible tasks but completed tasks exist, the system MUST communicate that completed tasks are hidden and provide a way to reveal them, rather than implying the task list is empty.
- **FR-009**: The show/hide control MUST remain accessible at all times, including when no tasks are currently visible.

### Key Entities *(include if feature involves data)*

- **Task**: A unit of work with a name, optional description, an optional parent (forming a tree), and a completion state (complete or incomplete, derived from whether it has a completion timestamp). The completion state determines whether a task is subject to hiding.
- **Show-completed preference**: A per-browser user setting capturing whether completed tasks are currently shown or hidden. Defaults to hidden.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On opening the task list with no saved preference, 100% of completed tasks are hidden and 100% of incomplete tasks are shown.
- **SC-002**: A user can switch between hiding and showing completed tasks in a single interaction (one tap/click).
- **SC-003**: After toggling to show completed tasks and reloading the page in the same browser, completed tasks remain visible without any further action.
- **SC-004**: When a user marks a task complete in the default view, the task is removed from view within the same interaction (no manual refresh needed).
- **SC-005**: When all tasks are complete and hidden, 100% of users can locate and use the control to reveal them (the view never appears as a dead-end empty list).

## Assumptions

- The web app's existing task list (task tree) is the surface for this feature; the CLI and TUI clients are out of scope.
- A task's completion state is determined by its existing completion timestamp; no new completion concept is introduced.
- "Completed" includes any task currently marked done, regardless of when it was completed; there is no time-window or "recently completed" distinction.
- The show/hide preference is stored per browser (not synced across devices), consistent with how the app already persists view state such as expanded/collapsed tasks.
- Hiding affects only display; it does not delete, archive, or otherwise change the underlying tasks, and completed tasks remain fully retrievable.
- Default behavior for a first-time user (no stored preference) is to hide completed tasks.
