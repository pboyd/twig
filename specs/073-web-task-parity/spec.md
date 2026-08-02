# Feature Specification: Task Parity for the Web App

**Feature Branch**: `073-web-task-parity`

**Created**: 2026-08-02

**Status**: Draft

**Input**: User description: "task parity for the web app — The web app is missing several capabilities that the TUI has, we need to bring task functionality in-line with what the TUI does and what the server already provides endpoints for. Specifically: delete a task, set a task's due date, link/unlink a task with a goal, set the snooze date on a task. Not in this round: move a task to a new parent, anything related to the Pomodoro timer, task filters."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Delete a task (Priority: P1)

A user viewing a task in the web app can delete it. Because deleting a task also removes all of its subtasks, the user is asked to confirm before the deletion happens. After deletion the user is returned to the task list, where the task and its subtasks no longer appear.

**Why this priority**: Deletion is the most conspicuous gap — today a task created by mistake in the web app can never be removed there, forcing a switch to the CLI/TUI. It is also the only irreversible action in this feature, so getting its confirmation flow right matters most.

**Independent Test**: Create a task with a subtask, delete the parent from the web app, confirm both disappear from the task list.

**Acceptance Scenarios**:

1. **Given** a task with no subtasks, **When** the user chooses to delete it and confirms, **Then** the task is removed and no longer appears anywhere in the web app.
2. **Given** a task with subtasks, **When** the user chooses to delete it, **Then** the confirmation warns that its subtasks will also be deleted, and confirming removes the task and every descendant.
3. **Given** the delete confirmation is shown, **When** the user cancels, **Then** nothing is deleted and the task view is unchanged.
4. **Given** a task that was already deleted in another session, **When** the user confirms deletion, **Then** the app shows a friendly error and refreshes the list rather than failing silently.

---

### User Story 2 - Set or clear a task's due date (Priority: P2)

A user editing a task in the web app can set a due date, change an existing due date, or clear it. The due date is visible when viewing the task.

**Why this priority**: Due dates already exist on tasks and are shown elsewhere; users just cannot manage them from the web today. It is a frequent, low-risk edit.

**Independent Test**: Edit a task, set a due date, save, reload the task and see the date; then clear it and see it gone.

**Acceptance Scenarios**:

1. **Given** a task without a due date, **When** the user sets a due date and saves, **Then** the task shows that due date after saving and after a page reload.
2. **Given** a task with a due date, **When** the user changes the date and saves, **Then** the new date replaces the old one.
3. **Given** a task with a due date, **When** the user clears the date and saves, **Then** the task has no due date.
4. **Given** the user edits only the due date, **When** they save, **Then** the task's other details (name, description, snooze, position in the tree) are unchanged.

---

### User Story 3 - Snooze a task (Priority: P2)

A user can set a "snoozed until" date on a task, change it, or clear it to un-snooze the task. A snoozed task is visibly marked as snoozed in the web app, consistent with how the rest of the product treats snoozing.

**Why this priority**: Snoozing is a daily-workflow feature in the TUI; without it web users cannot defer tasks. Same shape of edit as the due date, so it rides along naturally.

**Independent Test**: Snooze a task until a future date from the web app, verify the task is marked snoozed; clear the snooze and verify the mark disappears.

**Acceptance Scenarios**:

1. **Given** an active task, **When** the user sets a snooze date in the future and saves, **Then** the task is marked as snoozed.
2. **Given** a snoozed task, **When** the user clears the snooze date and saves, **Then** the task is no longer marked as snoozed.
3. **Given** the user edits only the snooze date, **When** they save, **Then** the task's other details are unchanged.

---

### User Story 4 - Link or unlink a task with a goal (Priority: P3)

A user viewing a task can associate it with one of their goals, change which goal it is associated with, or remove the association. Linking a task to a goal applies to the task and its whole subtree, matching how goals work everywhere else in the product. Where the product's rules forbid a link (an ancestor task already carries a goal, or a subtask has its own goal), the web app explains why instead of failing cryptically.

**Why this priority**: Valuable for goal-driven planning, but it has the most business rules and depends on goals existing; the other stories deliver value without it.

**Independent Test**: Link a task to an existing goal from the web app, see the goal reflected on the task and the task listed under the goal; unlink and see the association removed.

**Acceptance Scenarios**:

1. **Given** a task with no goal and at least one goal defined, **When** the user links the task to a goal, **Then** the task shows that goal, and the task appears under that goal's detail view.
2. **Given** a task linked to a goal, **When** the user switches it to a different goal, **Then** the new goal replaces the old one.
3. **Given** a task linked to a goal, **When** the user removes the association, **Then** the task shows no goal.
4. **Given** a task whose ancestor already has a goal, **When** the user attempts to link the task to a goal, **Then** the app shows an understandable message explaining the task already inherits a goal from its parent and no change is made.
5. **Given** a task with a subtask that has its own goal, **When** the user attempts to link the parent to a goal, **Then** the app shows an understandable message and no change is made.
6. **Given** the user has no goals defined, **When** they view the goal-linking control, **Then** the app makes clear there are no goals to choose from rather than presenting an empty broken control.

---

### Edge Cases

- Deleting a task that appears on today's plan: the deletion proceeds and the plan no longer shows the task (server cascade); the plan view must not break.
- Setting a due date or snooze date in the past: allowed, matching existing product behavior (a past snooze simply means the task is not snoozed anymore).
- Concurrent edits: another session changed or deleted the task between load and save — the app surfaces a clear error and refreshes rather than silently overwriting or crashing.
- Goal linked to a task that is then deleted: the goal remains, only the association disappears with the task.
- A save that includes both a due-date change and a snooze change applies both together.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Users MUST be able to delete a task from the web app.
- **FR-002**: Before a deletion is performed, the web app MUST require explicit confirmation and MUST state that all of the task's subtasks will be deleted with it.
- **FR-003**: After a successful deletion, the web app MUST navigate the user away from the deleted task and no longer display it or its descendants anywhere.
- **FR-004**: Users MUST be able to set, change, and clear a task's due date when editing a task in the web app.
- **FR-005**: Users MUST be able to set, change, and clear a task's snooze ("snoozed until") date when editing a task in the web app.
- **FR-006**: A task snoozed until a future date MUST be visibly distinguishable as snoozed in the web app's task views.
- **FR-007**: Users MUST be able to link a task to one of their goals, change the linked goal, and remove the link from the web app.
- **FR-008**: When the product's goal rules reject a link (ancestor already has a goal, or a descendant has its own goal), the web app MUST present the reason in plain language and leave the task unchanged.
- **FR-009**: Editing any one attribute (due date, snooze, name, description) MUST NOT alter any other attribute of the task, including its position in the task tree and its goal association.
- **FR-010**: All of these actions MUST reflect their outcome to the user (success or a human-readable error) and the views showing the affected task or goal MUST update without requiring a manual page reload.
- **FR-011**: Moving a task to a new parent, Pomodoro-related functionality, and task filtering remain out of scope and MUST NOT be added as part of this feature.

### Key Entities

- **Task**: A unit of work with a name, optional description, optional due date, optional snooze-until date, optional parent task (forming a tree), and an optional associated goal. Deleting a task deletes its whole subtree.
- **Goal**: A long-term objective a task subtree can be associated with. A task's effective goal may be inherited from an ancestor; only one goal may apply along any ancestor chain.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can delete a task, set a due date, snooze a task, and link a task to a goal entirely within the web app, with zero steps requiring the CLI or TUI.
- **SC-002**: Each of the four actions completes in three or fewer interactions from the task's detail view (excluding typing a date or choosing a goal).
- **SC-003**: 100% of rejected goal links (rule violations) show a human-readable explanation rather than a raw error.
- **SC-004**: After any of the four actions, the affected views reflect the change without a manual page reload.

## Assumptions

- The server already provides all needed capabilities (task deletion with cascade, due/snooze updates, goal association with subtree rules); no server or API changes are expected.
- Due dates and snooze dates follow existing product semantics — the web app introduces no new validation (past dates are allowed, as in the TUI).
- The confirmation for deletion is a simple confirm/cancel step; a soft-delete or undo mechanism is out of scope.
- Goal linking in the web app is offered from the task's perspective (choose a goal for a task); managing tasks from the goal's side beyond what already exists is out of scope.
- The existing web app visual conventions for snoozed/overdue tasks (as used in current list views) are reused rather than redesigned.
