# Feature Specification: User-Configurable Task Order

**Feature Branch**: `037-task-ordering`

**Created**: 2026-06-05

**Status**: Draft

**Input**: User description: "tasks should have a user-configurable order. The order of root-level tasks should be adjustable, and so should the order of the sub-tasks under any parent task. The order should persist across all the clients. In the TUI, `{` should rank the highlighted task higher and `}` should rank it lower. In the web app, users should be able to drag-and-drop the tasks in the order they want. The CLI does not need to rank tasks right now (though it should respect the order defined elsewhere)."

## Clarifications

### Session 2026-06-05

- Q: In the web app, should drag-and-drop move a task to a different parent (re-parenting), or only reorder within its current sibling group? → A: Reorder only — dragging changes position among current siblings; changing a task's parent stays with the existing move-task capability and is out of scope here.
- Q: When completed tasks are hidden, how should TUI `{`/`}` move a highlighted task relative to hidden siblings interleaved in the group? → A: Move past hidden ones — `{`/`}` swaps with the next *visible* sibling, skipping any hidden tasks in between, so a keypress always produces a visible change unless the task is at a visible boundary.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reorder tasks in the TUI (Priority: P1)

A user working in the terminal interface wants their task list to reflect their own sense of priority rather than a fixed default order. They highlight a task and press `{` to move it earlier (higher) among its siblings, or `}` to move it later (lower). This works for top-level tasks and for the sub-tasks beneath any parent.

**Why this priority**: The TUI is the primary day-to-day interface and the explicit keyboard ranking is the most direct expression of the user's intent. Delivering this alone gives users full control over ordering from the tool they use most.

**Independent Test**: Open the TUI with several sibling tasks, press `{`/`}` on a highlighted task, and confirm it visibly changes position relative to its siblings and that the new position is retained after the list is refreshed or the TUI is restarted.

**Acceptance Scenarios**:

1. **Given** three root-level tasks in order A, B, C with B highlighted, **When** the user presses `{`, **Then** the order becomes B, A, C and B remains highlighted.
2. **Given** three root-level tasks in order A, B, C with B highlighted, **When** the user presses `}`, **Then** the order becomes A, C, B and B remains highlighted.
3. **Given** a parent task with sub-tasks X, Y, Z and Y highlighted, **When** the user presses `{`, **Then** the sub-task order becomes Y, X, Z while sub-tasks remain under the same parent.
4. **Given** the first task among its siblings is highlighted, **When** the user presses `{`, **Then** the order is unchanged (it is already first) and no error is shown.
5. **Given** the last task among its siblings is highlighted, **When** the user presses `}`, **Then** the order is unchanged (it is already last) and no error is shown.
6. **Given** a task has been reordered, **When** the TUI is closed and reopened, **Then** the task appears in its newly assigned position.

---

### User Story 2 - Reorder tasks by drag-and-drop in the web app (Priority: P2)

A user in the web application wants to arrange their tasks visually. They click and drag a task to a new position among its siblings and drop it there. The list reflects the new order immediately and the change is saved.

**Why this priority**: The web app is a secondary but important interface; drag-and-drop is the natural ordering gesture there. It builds on the same persisted-order capability established for the TUI.

**Independent Test**: In the web app, drag a task above or below a sibling, release it, reload the page, and confirm the order is preserved.

**Acceptance Scenarios**:

1. **Given** root-level tasks A, B, C, **When** the user drags C and drops it above A, **Then** the displayed order becomes C, A, B.
2. **Given** a parent with sub-tasks X, Y, Z, **When** the user drags Z above Y, **Then** the sub-task order becomes X, Z, Y and Z remains a child of the same parent.
3. **Given** a task has been dragged to a new position, **When** the user reloads the page, **Then** the task is shown in its new position.
4. **Given** a drag begins but the user drops the task back on its original position (or cancels the drag), **When** the drag ends, **Then** the order is unchanged.

---

### User Story 3 - Consistent order across all clients (Priority: P3)

A user who reorders tasks in one client expects to see the same order everywhere — in the TUI, in the web app, and in the read-only CLI listing. The CLI does not need controls to change the order, but it must display tasks in the user-defined order.

**Why this priority**: This is what makes the ordering feel like a property of the user's data rather than a per-client preference. It depends on ordering being persisted centrally, so it follows the client-specific editing stories.

**Independent Test**: Reorder tasks in the TUI, then list the same tasks via the CLI and open them in the web app; confirm all three present siblings in the identical user-defined order.

**Acceptance Scenarios**:

1. **Given** tasks reordered in the web app, **When** the user lists tasks via the CLI, **Then** the CLI prints them in the user-defined order.
2. **Given** tasks reordered in the TUI, **When** the user opens the web app, **Then** the web app shows the same order.
3. **Given** a CLI listing of tasks, **When** the user inspects sub-tasks under any parent, **Then** those sub-tasks also appear in the user-defined order.

---

### Edge Cases

- **Already at boundary**: Ranking the first sibling higher or the last sibling lower is a no-op; the action is accepted without error and the highlight/selection is preserved.
- **Newly created tasks**: A newly created task is placed at the end of its sibling group's order by default (see Assumptions); existing siblings keep their relative order.
- **Moving a task to a new parent**: When a task changes parent (existing move capability), it is placed at the end of the destination sibling group's order; the source group's remaining tasks keep their relative order.
- **Completed / hidden tasks**: Ordering is defined over the full sibling group; hiding completed tasks does not change the relative order of the visible tasks, and unhiding restores them to their stored positions. In the TUI, `{`/`}` on a visible task swaps it with the next *visible* sibling, skipping over any hidden tasks in between (so the move is always visible unless the task is at a visible boundary).
- **Single-item or empty groups**: Reordering within a group of zero or one task is always a no-op.
- **Concurrent edits**: If two clients reorder the same sibling group at nearly the same time, the system applies the changes and converges to a single consistent order without losing tasks (no task is dropped or duplicated).
- **Tasks never manually ordered**: Before any manual reordering, tasks appear in the system's existing default order (see Assumptions); manual ordering overrides that default from then on.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Each task MUST have a position relative to its siblings (the other tasks sharing the same parent, or the set of root-level tasks for tasks with no parent).
- **FR-002**: The system MUST persist task order centrally so that it is shared across all clients and survives client restarts and sessions.
- **FR-003**: Users MUST be able to change a task's position among its siblings, both for root-level tasks and for sub-tasks under any parent.
- **FR-004**: Reordering a task MUST affect only its position within its own sibling group; it MUST NOT change the task's parent or reorder unrelated groups.
- **FR-005**: In the TUI, pressing `{` MUST move the highlighted task one position earlier (higher) within its sibling group, and pressing `}` MUST move it one position later (lower). When some siblings are hidden (e.g., completed-task hiding), `{`/`}` MUST move the task past hidden siblings so that it swaps with the next *visible* sibling, always producing a visible change unless the task is at a visible boundary.
- **FR-006**: In the TUI, the highlighted/selected task MUST remain highlighted after a reorder action so the user can continue ranking it.
- **FR-007**: In the TUI, a reorder action at a boundary (first task with `{`, last task with `}`) MUST be a no-op that produces no error.
- **FR-008**: In the web app, users MUST be able to reorder tasks within a sibling group via drag-and-drop, including both root-level tasks and sub-tasks under any parent. Drag-and-drop reorders within the current sibling group only; it MUST NOT move a task to a different parent (re-parenting remains the responsibility of the existing move-task capability and is out of scope for this feature).
- **FR-009**: In the web app, the displayed order MUST update to reflect a completed drag-and-drop, and a cancelled drag (or drop on the original position) MUST leave the order unchanged.
- **FR-010**: All clients that display tasks (TUI, web app, and CLI) MUST present each sibling group in the user-defined order.
- **FR-011**: The CLI MUST respect and display the user-defined order but is NOT required to provide controls to change it in this feature.
- **FR-012**: A newly created task MUST be assigned a defined position within its sibling group without disturbing the relative order of existing siblings.
- **FR-013**: When a task's parent changes, the task MUST be assigned a defined position within its new sibling group, and the remaining tasks in the old group MUST keep their relative order.
- **FR-014**: The system MUST keep ordering consistent when tasks are hidden/shown (e.g., completed-task hiding) such that the relative order of remaining tasks is preserved.
- **FR-015**: Concurrent reordering from multiple clients MUST converge to a single consistent order without dropping or duplicating any task.

### Key Entities *(include if feature involves data)*

- **Task**: An item of work that may have a parent task and may have sub-tasks. Gains an ordering attribute that determines its position relative to its siblings.
- **Sibling group**: The set of tasks sharing the same parent, or the set of all root-level (parentless) tasks. Order is defined and adjusted within a sibling group, independently of other groups.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can move any task to any position within its sibling group using only the controls described (TUI `{`/`}`, web drag-and-drop) without consulting documentation.
- **SC-002**: Each TUI `{` or `}` press moves the highlighted task exactly one position (or is a no-op at a boundary), with the change visible immediately.
- **SC-003**: After reordering in any one client, 100% of the other clients display the same sibling order on their next load of those tasks.
- **SC-004**: A user-defined order is retained across client restarts and sessions with no loss of ordering 100% of the time.
- **SC-005**: After any sequence of reorder actions, every task appears exactly once in its sibling group (no tasks lost or duplicated).
- **SC-006**: Reordering feels responsive: the user sees the updated order in under 1 second after a TUI keypress or a web drop.

## Assumptions

- **Default position for new tasks**: Newly created tasks are appended to the end of their sibling group's order. (No explicit requirement was given; end-of-list is the least surprising default and matches typical task tools.)
- **Default order before manual ordering**: Tasks that have never been manually ordered appear in the system's current default order (e.g., creation order). Manual reordering overrides this going forward.
- **Move-to-new-parent placement**: When an existing move-task capability changes a task's parent, the moved task lands at the end of the destination group's order.
- **Scope of CLI changes**: The CLI is read-only with respect to ordering in this feature; it only needs to display tasks in the stored order. Adding CLI ranking commands is out of scope.
- **Reuse of existing task model**: This feature extends the existing task data and synchronization mechanisms; it does not introduce a separate ordering system or new client–server transport beyond what is needed to read and update a task's position.
- **Granularity of TUI ranking**: `{` and `}` move a task by a single position per keypress (rather than to the extreme top/bottom); repeated presses move it further.
