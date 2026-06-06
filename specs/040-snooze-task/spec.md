# Feature Specification: Snooze a Task

**Feature Branch**: `040-snooze-task`

**Created**: 2026-06-06

**Status**: Draft

**Input**: User description: "Snooze a task. Looking at the task list should feel empowering. But a big list of things that need to be done, but can't be done yet, is instead overwhelming and frustrating. ... Let's give users the ability to hide a task until some day in the future when it can actually be done. In the TUI, this should be another field on the add task and edit task forms. The CLI and web apps don't need the ability to snooze a task, but they should respect the hint to not show them. This has implications for the existing 'Show completed' and 'Hide completed' filters. 'Show completed' should be changed to 'Show all' and include completed and snoozed tasks (ideally, snoozed tasks would be visually indicated--💤 perhaps). 'Hide completed' will need to change too, ('Show only pending' perhaps). The TUI simply says 'Toggle completed' in the help page, it can remain triggered by the `c` key, but the help text will need to be updated."

## Clarifications

### Session 2026-06-06

- Q: Where should snoozed-task visibility be decided (client-side vs server-side)? → A: Client-side — `ListTasks` keeps returning all tasks (including the snooze day); each client hides future-snoozed tasks in its default view using its own local current day. Matches the existing client-side completed filter; no new list parameters.
- Q: Which exact wording should the two-state TUI filter use? → A: "Show all" (completed + snoozed + pending) ↔ "Show only pending".
- Q: When a parent task is snoozed, what happens to its descendants in the default view? → A: Hide the whole subtree (parent + all descendants) together, mirroring completed-subtree behavior. *(carried over from the `/speckit-specify` clarification)*

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Snooze a task until a future day (Priority: P1)

A user has a task that genuinely cannot be acted on yet — for example, they need to talk to a colleague who is out of the office for a week. Instead of leaving it in the list to nag them every day, the user opens the task in the TUI add or edit form, sets a "snooze until" day in the future, and the task disappears from their working view until that day arrives. On that day, the task automatically reappears so it is not forgotten.

**Why this priority**: This is the core value of the feature — turning an overwhelming, partly-undoable list into a focused list of things that can actually be done now, without losing track of deferred work. It delivers the primary benefit on its own.

**Independent Test**: In the TUI, edit a task and set its snooze-until day to a date in the future; confirm the task is no longer shown in the default (pending) view. Advance/observe at/after that day and confirm the task reappears in the default view.

**Acceptance Scenarios**:

1. **Given** a task with no snooze date, **When** the user edits it and sets a snooze-until day in the future, **Then** the task is hidden from the default (pending) view from that point until the chosen day.
2. **Given** a task snoozed until a future day, **When** that day arrives, **Then** the task reappears automatically in the default (pending) view without further user action.
3. **Given** the add-task form, **When** the user enters a snooze-until day while creating a task, **Then** the new task is created already snoozed and does not appear in the default view until the chosen day.
4. **Given** a snoozed task, **When** the user edits it and clears the snooze-until day, **Then** the task immediately returns to the default (pending) view.

---

### User Story 2 - Reveal and review snoozed tasks (Priority: P2)

A user occasionally wants to see everything, including work they have deferred — to reconsider a snooze, wake a task early, or just confirm nothing important is hidden. Using the existing completed-tasks filter (now broadened), the user reveals all tasks. Snoozed tasks are shown alongside everything else and are visually marked (e.g., with 💤) so it is obvious why they were hidden.

**Why this priority**: Deferring work is only safe if the user can retrieve it on demand. It builds directly on P1 but is secondary to simply decluttering the default view.

**Independent Test**: With a snoozed task present and the default view active, toggle the filter to "Show all" and confirm the snoozed task appears with a snooze indicator; toggle back and confirm it is hidden again.

**Acceptance Scenarios**:

1. **Given** snoozed (and/or completed) tasks exist and the view is set to "Show only pending", **When** the user activates the filter toggle, **Then** the view switches to "Show all" and both completed and snoozed tasks become visible.
2. **Given** the view is "Show all", **When** the user activates the filter toggle again, **Then** the view returns to "Show only pending" and both completed and snoozed tasks are hidden.
3. **Given** the view is "Show all", **When** a snoozed task is displayed, **Then** it is visually distinguished as snoozed (e.g., a 💤 indicator).
4. **Given** a snoozed task is visible in "Show all", **When** the user edits it and clears or changes the snooze date, **Then** the view updates to reflect the new state.

---

### User Story 3 - CLI and web honor the snooze (Priority: P2)

A user who relies on the CLI or web app to glance at their tasks should get the same focused, decluttered list. Those clients do not let the user set a snooze, but they must respect a task's snooze and keep snoozed tasks out of the default view, so the deferral the user set in the TUI is honored everywhere.

**Why this priority**: A snooze that only applies in one client would be misleading — the task would still "taunt" the user from the CLI or web. Honoring it everywhere is essential to the feature's promise, though it depends on P1 existing first.

**Independent Test**: Snooze a task in the TUI, then open the CLI task listing and the web task list; confirm the snoozed task does not appear in their default views.

**Acceptance Scenarios**:

1. **Given** a task snoozed until a future day, **When** the user lists tasks in the CLI default view, **Then** the snoozed task is not shown.
2. **Given** a task snoozed until a future day, **When** the user opens the web task list in its default view, **Then** the snoozed task is not shown.
3. **Given** a task whose snooze day has passed, **When** the user lists tasks in the CLI or web default view, **Then** the task is shown as a normal pending task.

---

### Edge Cases

- **Snooze day is today or in the past**: A snooze-until day that is the current day or earlier has no hiding effect — the task is treated as pending and shown. (Setting a past date is effectively a no-op / immediate un-snooze.)
- **Snoozed and completed at once**: A task could be both snoozed and completed. In "Show only pending" it is hidden (either condition hides it); in "Show all" it is shown. Visual indicators for completed and snoozed should not conflict.
- **Snoozed parent with visible children**: When a parent task is snoozed, its entire subtree (the parent and all its descendants) is hidden together in the default view, so children never appear "orphaned" under a hidden parent. This mirrors how a completed subtree hides together.
- **Whole list snoozed/completed**: When every task is hidden in the default view but snoozed/completed tasks exist, the interface must make clear that hidden tasks exist and how to reveal them, rather than implying the list is empty.
- **Waking early**: A user can remove a snooze before its day by editing the task and clearing the snooze-until field; the task returns to the pending view immediately.
- **Timezone boundary**: "The day arrives" is evaluated against the user's local day boundary, so a task snoozed until a given calendar day becomes visible at the start of that day for the user.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A task MUST support an optional "snooze until" day that defers it; absence of a value means the task is not snoozed.
- **FR-002**: In the TUI, the add-task and edit-task forms MUST provide a field for setting (and, when editing, changing or clearing) a task's snooze-until day.
- **FR-003**: A task whose snooze-until day is in the future MUST be hidden from the default ("pending") view in all clients (TUI, CLI, web). Each client decides this itself by comparing the snooze-until day against its own local current day; snoozed tasks are still retrieved from the server (the task list is not filtered server-side) so a client can choose to reveal them.
- **FR-004**: A task whose snooze-until day is the current local day or has passed MUST be treated as not snoozed and shown in the default view.
- **FR-005**: A snoozed task MUST reappear automatically once its snooze-until day arrives, with no user action required beyond normally viewing/refreshing the list.
- **FR-006**: The TUI filter toggle (bound to the `c` key) MUST switch between two states: **"Show only pending"** (hiding completed and snoozed tasks) and **"Show all"** (showing completed and snoozed tasks alongside pending tasks).
- **FR-007**: The TUI MUST update the filter's user-facing wording from "Show completed"/"Hide completed" to **"Show all"** / **"Show only pending"**, and MUST update the help-page entry currently reading "toggle completed" to reflect the broadened behavior.
- **FR-008**: When snoozed tasks are shown (in "Show all"), the TUI MUST visually distinguish snoozed tasks from pending and completed tasks (e.g., a 💤 indicator).
- **FR-009**: The CLI and web clients MUST honor a task's snooze and keep snoozed tasks out of their default views, but are NOT required to provide any control for setting a snooze.
- **FR-010**: Snoozing MUST only affect visibility; it MUST NOT delete, complete, or otherwise alter the underlying task, and a snoozed task MUST remain fully retrievable and editable.
- **FR-011**: A user MUST be able to remove or shorten a snooze before its day arrives by editing the task's snooze-until field, returning the task to the pending view.
- **FR-012**: When a parent task is snoozed, the system MUST hide its entire subtree (the parent and all of its descendants) from the default view together, so no descendant is left visibly orphaned under a hidden parent.
- **FR-013**: When the default view is empty because all remaining tasks are snoozed and/or completed, the interface MUST indicate that hidden tasks exist and how to reveal them rather than appearing as an empty list.

### Key Entities *(include if feature involves data)*

- **Task**: A unit of work with a name, optional description, optional due date, optional parent (forming a tree), a completion state, and now an optional **snooze-until day**. The snooze-until day, when set to a future day, hides the task from default views until that day.
- **Snooze-until day**: A future calendar day associated with a task before which the task is hidden from default ("pending") views. Cleared/absent means not snoozed; a current-or-past value is inert.
- **View filter state (TUI)**: A two-state setting — "Show only pending" (default) or "Show all" — controlling whether completed and snoozed tasks are displayed. Toggled with the `c` key.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can snooze a task to a future day from the TUI add or edit form in a single form interaction, and the task disappears from the pending view immediately.
- **SC-002**: 100% of tasks snoozed to a future day are absent from the default view in all three clients (TUI, CLI, web); 100% reappear in the default view on/after their snooze day.
- **SC-003**: A user can reveal all snoozed (and completed) tasks in the TUI with a single key press (`c`), and every snoozed task shown is visually identifiable as snoozed.
- **SC-004**: After a task's snooze day passes, the task reappears in the default view with no manual action other than viewing the list (no reload-only-fixes-it surprises).
- **SC-005**: When the pending view is empty due to snoozed/completed tasks, 100% of users can locate the control to reveal hidden tasks (the view is never a dead-end empty list).

## Assumptions

- **Snooze granularity is a whole day** (a calendar date), not a precise time — matching the user's "hide until some day" framing. The task becomes visible at the start of that day in the user's local timezone.
- The snooze-until day is stored on the task itself (server-side) so the deferral is consistent across the TUI, CLI, and web clients and across devices. **Filtering is performed client-side** (decided during clarification): the server returns all tasks including their snooze day, and each client hides future-snoozed tasks using its own local current day — consistent with how the existing completed-task filter already works.
- The TUI snooze-until field reuses the existing date-input convention already used for the due field (bare `YYYY-MM-DD` accepted), so no new input format is introduced.
- The TUI filter remains a single two-state toggle on the `c` key (pending-only ↔ all); no third intermediate state (e.g., "completed but not snoozed") is introduced.
- Snoozing a parent task hides its whole subtree together (decided during clarification); the same subtree-hides-together rule already applied to completed tasks is reused for snoozed tasks.
- In the CLI and web clients, snoozed tasks are simply excluded from the default listing; whether those clients later expose a "show all" affordance is out of scope here (they at minimum honor the hide).
- Snooze is independent of the existing "due" date — a task may have a due date, a snooze date, both, or neither; they serve different purposes (when it's needed vs. when it can first be started).
- Setting a snooze-until day equal to today or earlier is permitted and simply results in no hiding (treated as un-snoozed), rather than being rejected as invalid.
- Snoozing affects display/visibility only and reuses the existing per-task data model and list-retrieval flow; no archival or separate storage concept is introduced.
