# Feature Specification: Complete Tasks

**Feature Branch**: `004-complete-tasks`

**Created**: 2026-05-20

**Status**: Draft

**Input**: User description: "users need to be able to mark a task complete. This will need to be done through the CLI: `todo task complete 1`. Record a timestamp for when the task was completed. We'll also need to filter the task list for completed or incomplete tasks. By default the CLI should only show incomplete tasks, but it should have a flag to show everything or completed tasks. The system should not allow a user to complete a task with an incomplete subtask."

## Clarifications

### Session 2026-05-20

- Q: When the default list filter hides completed tasks, how should it interact with the tree rendering? → A: Hide only completed task rows; an incomplete parent stays visible even if some children are complete. A branch where every task (parent and all descendants) is complete disappears entirely.
- Q: How should completed tasks be visually distinguished in list output? → A: Checkbox marker `[ ]` for incomplete and `[x]` for complete, prefixed to each task name. Completed tasks also show their completion timestamp at the end of the line.
- Q: What should happen if a user tries to add a subtask under a parent that is already complete? → A: Reject with a clear error. A completed task may not gain new (incomplete) descendants; the invariant "no incomplete descendant under a complete task" always holds.
- Q: In which order should tasks appear in `--completed` and `--all` listings? → A: Keep id order (creation order, oldest first) within each parent for every list mode; only the filter differs across modes, never the ordering rules.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Mark a task complete (Priority: P1)

A user has a task they have finished and wants to record that it is done. From the CLI they run `todo task complete <task-id>` and the task is marked complete with the moment of completion recorded.

**Why this priority**: Marking tasks complete is the core action that gives the todo app its purpose. Without it, the app is only a capture tool. This unlocks all subsequent filtering and reporting.

**Independent Test**: Create a task, run `todo task complete <id>`, then inspect the task — it shows as complete with a completion timestamp. Delivers a working completion workflow on its own.

**Acceptance Scenarios**:

1. **Given** an incomplete task with id `1`, **When** the user runs `todo task complete 1`, **Then** the task is marked complete and the completion timestamp is recorded as the current time.
2. **Given** a task that is already complete, **When** the user runs `todo task complete <id>` against it, **Then** the system reports that the task is already complete and the original completion timestamp is preserved.
3. **Given** a task id that does not exist, **When** the user runs `todo task complete <id>`, **Then** the system reports that the task was not found and exits with a non-zero status.

---

### User Story 2 - Default to showing only incomplete tasks (Priority: P1)

A user wants to see what still needs to be done. When they list tasks without any flags, only incomplete tasks appear, so the list reflects outstanding work.

**Why this priority**: Without sensible default filtering, completed tasks accumulate in the list and obscure outstanding work — making completion tracking pointless. This is paired with P1 because completion is only valuable if it changes what the user sees.

**Independent Test**: Create several tasks, complete some of them, run the list command with no flags, and confirm only the incomplete ones appear.

**Acceptance Scenarios**:

1. **Given** a mix of complete and incomplete tasks, **When** the user lists tasks with no filter flag, **Then** only incomplete tasks are shown.
2. **Given** all tasks are complete, **When** the user lists tasks with no filter flag, **Then** the result is an empty list (with a clear indication that no incomplete tasks remain).

---

### User Story 3 - Filter the task list by completion status (Priority: P2)

A user wants to review what they have finished, or see the full picture of all tasks regardless of status. They use a flag on the list command to show completed tasks only, or all tasks.

**Why this priority**: Useful for review, reporting, and recovering tasks the user wants to revisit, but not required for the basic complete-and-hide flow.

**Independent Test**: Create a mix of complete and incomplete tasks, list with each filter flag in turn, and confirm the correct subset appears for each.

**Acceptance Scenarios**:

1. **Given** a mix of complete and incomplete tasks, **When** the user lists tasks with the flag to show completed tasks only, **Then** only completed tasks are shown, including their completion timestamps.
2. **Given** a mix of complete and incomplete tasks, **When** the user lists tasks with the flag to show all tasks, **Then** every task is shown with its completion status visible.
3. **Given** the user passes both the "completed only" and "show all" flags together, **When** the list command runs, **Then** the system reports the conflict and exits without showing results.

---

### User Story 4 - Block completion when subtasks are incomplete (Priority: P2)

A user attempts to mark a parent task complete while it still has unfinished subtasks. The system refuses, explaining that the subtasks must be completed first, preserving the integrity of the task hierarchy.

**Why this priority**: Important for data integrity and trust in the completion status, but the feature still delivers value for users with flat task lists if this is implemented later.

**Independent Test**: Create a parent task with at least one incomplete subtask, attempt to complete the parent, and confirm the system rejects the attempt with a clear message and does not record a completion timestamp.

**Acceptance Scenarios**:

1. **Given** a parent task with one or more incomplete subtasks, **When** the user runs `todo task complete <parent-id>`, **Then** the system rejects the request, identifies which subtasks remain incomplete, and the parent task remains incomplete.
2. **Given** a parent task whose subtasks are all complete, **When** the user runs `todo task complete <parent-id>`, **Then** the parent task is marked complete with a current timestamp.
3. **Given** a parent task with no subtasks, **When** the user runs `todo task complete <parent-id>`, **Then** the parent task is marked complete normally.

---

### Edge Cases

- The user attempts to complete a task whose id is malformed (e.g., not a number) — the system reports an input error and does not modify any data.
- A subtask itself has further nested subtasks that are incomplete — completion is blocked by any incomplete descendant in the subtask chain, not just direct children.
- A task is completed, then later the user wants to reopen it — reopening is out of scope for this feature and is not provided here.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI MUST accept the command `todo task complete <task-id>` to mark the identified task as complete.
- **FR-002**: When a task is marked complete, the system MUST record the date and time of completion and persist it with the task.
- **FR-003**: The system MUST reject completion of a task that has any incomplete subtask (at any depth in the subtask hierarchy) and MUST NOT record a completion timestamp on the parent in that case.
- **FR-004**: When completion is rejected due to incomplete subtasks, the system MUST identify which subtasks are blocking completion.
- **FR-005**: The CLI list command MUST, by default (with no filter flag), show only tasks that are not complete. The tree structure is preserved by omitting only the rows for completed tasks: an incomplete parent task remains visible even when some of its children are complete, and a branch is omitted entirely only when the parent task and all of its descendants are complete.
- **FR-006**: The CLI list command MUST support a flag to show only completed tasks.
- **FR-007**: The CLI list command MUST support a flag to show all tasks regardless of completion status.
- **FR-008**: When a completed task appears in the list output, the system MUST include that task's completion timestamp at the end of its line.
- **FR-009**: Every task line in list output MUST be prefixed with a checkbox marker: `[ ]` for incomplete tasks and `[x]` for completed tasks. This marker is shown in every list mode (default, completed-only, all).
- **FR-010**: Attempting to complete a task that is already complete MUST be a no-op that preserves the original completion timestamp and reports the current status to the user.
- **FR-011**: Attempting to complete a non-existent task MUST produce a clear error message and a non-zero exit status, with no changes to stored data.
- **FR-012**: Conflicting list filter flags (e.g., "completed only" together with "show all") MUST be rejected with a clear error.
- **FR-013**: The system MUST reject any attempt to add a new subtask under a task that is already complete, with a clear error explaining that the parent task is complete. No new task is created in that case. (Combined with FR-003, this preserves the invariant that a completed task has no incomplete descendants.)
- **FR-014**: The system MUST also reject moving an existing task so that it becomes a descendant of a completed task, by the same rule as FR-013.
- **FR-015**: Across every list mode (default, completed-only, all), tasks MUST appear in id order (creation order, oldest first) within each parent. The filter changes which tasks are included; it does not change ordering rules.

### Key Entities *(include if feature involves data)*

- **Task**: Gains a completion state (complete vs. incomplete) and an optional completion timestamp that is set when the task is marked complete and unset (or absent) otherwise. Retains its existing relationship to any parent task and subtasks.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can mark a single task complete with one command invocation in under 5 seconds end-to-end.
- **SC-002**: After completing one or more tasks, the default list command shows zero completed tasks; 100% of completed tasks are hidden by default.
- **SC-003**: 100% of attempts to complete a task with any incomplete subtask are rejected and produce a message naming at least one blocking subtask.
- **SC-004**: Every completed task surfaced via the completed or all-tasks filter shows an accurate completion timestamp matching the moment the task was marked complete.
- **SC-005**: A new user can discover how to complete a task and how to view completed tasks using only the CLI's built-in help, without consulting external docs.

## Assumptions

- The existing `todo` CLI and its task-listing command established in the prior task-CRUD and todo-CLI features will be extended; no new top-level interface is introduced.
- Task identifiers used at the command line are the existing numeric ids surfaced by the current list command.
- Subtasks are an existing concept in the data model (referenced from the user description); their structure is reused as-is rather than introduced here.
- Reopening a completed task (un-completing it) is out of scope for this feature.
- Bulk completion (completing multiple tasks in one command) is out of scope for this feature.
- Completion timestamps are recorded in the system's standard time representation; users are not asked to provide a custom completion time.
- Authentication and ownership rules established by the user-auth feature continue to apply: a user can only complete their own tasks.
- Following the existing CLI exit-status convention (single non-zero code `1` for every failure, `0` for success), completing an already-complete task is treated as a successful no-op and exits with status `0`. Genuine errors (non-existent task id, blocked by incomplete subtasks, conflicting flags, malformed id) exit with status `1`.
- Completion timestamps are recorded in UTC using RFC 3339 format, matching the convention established for due-date timestamps in the prior CLI feature.
