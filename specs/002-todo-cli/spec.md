# Feature Specification: Todo CLI Tool

**Feature Branch**: `002-todo-cli`

**Created**: 2026-05-18

**Status**: Draft

**Input**: User description: "this app needs a CLI tool — `todo task list` (no filters yet, show everything; display a tree of tasks with parentless tasks at the root, similar to `tree`; show id, name and due date), `todo task add [--parent id] [--due timestamp] <name>`, `todo task rm <id>`, `todo task mod <id> [--parent id] [--due timestamp] <name>`. At this stage we don't need to support description; there are no filters or any way to complete a task, so those can't be implemented."

## Clarifications

### Session 2026-05-18

- Q: How should `task list` render the task hierarchy? → A: `tree`-style ASCII connectors (`├──`, `└──`, `│`), matching the Unix `tree` command
- Q: How granular should the failure exit status be? → A: Single non-zero code — exit `1` for every failure
- Q: Which `--due` input formats are accepted? → A: Full RFC 3339 date-time and bare calendar dates; a bare date (`2026-06-01`) is interpreted as midnight UTC (`00:00:00Z`) that day
- Q: What do `rm` and `mod` print on success? → A: A brief confirmation line naming the affected task identifier (e.g. `Updated task 7`, `Deleted task 7`)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add a task from the command line (Priority: P1)

A person working in a terminal wants to capture something they have to do without leaving the command line. They run an `add` command, giving the task a name, and optionally a due date and a parent task to nest it under. The tool records the task and confirms it was created, reporting the identifier it was given.

**Why this priority**: Capturing tasks is the foundational action — without a way to record tasks from the command line, the rest of the tool has nothing to operate on. It is the minimum slice that delivers value on its own.

**Independent Test**: Run the `add` command with just a name, then with a name plus a due date, then with a name plus a parent identifier; confirm each invocation reports success and a new task identifier, and that the task is afterwards visible through the backend.

**Acceptance Scenarios**:

1. **Given** a reachable backend, **When** the user runs `add` with only a name, **Then** a task is created with that name, no due date, and no parent, and the tool reports the new task's identifier.
2. **Given** a reachable backend, **When** the user runs `add` with a name and a due date, **Then** a task is created carrying that due date and the tool reports the new identifier.
3. **Given** a reachable backend, **When** the user runs `add` with a name and a `--parent` identifier that matches an existing task, **Then** a task is created nested under that parent.
4. **Given** a reachable backend, **When** the user runs `add` without supplying a name, **Then** the command fails with a clear usage error and no task is created.
5. **Given** a reachable backend, **When** the user runs `add` with a `--parent` identifier that matches no existing task, **Then** the command fails with a clear error and no task is created.
6. **Given** a reachable backend, **When** the user supplies a `--due` value that is not a recognized timestamp, **Then** the command fails with a clear error explaining the expected format and no task is created.

---

### User Story 2 - View all tasks as a tree (Priority: P2)

A person wants to see everything they have recorded at a glance. They run a `list` command and the tool prints every task, arranged as a tree: top-level tasks at the root and subtasks indented beneath their parents, to any depth. Each line shows the task's identifier, its name, and its due date.

**Why this priority**: Reviewing recorded work is the natural counterpart to capturing it. It depends on tasks existing but is independently valuable as soon as any task is present.

**Independent Test**: With several tasks already recorded — some top-level, some nested several levels deep — run the `list` command and confirm the output shows every task, with each subtask indented under its correct parent and each line showing identifier, name, and due date.

**Acceptance Scenarios**:

1. **Given** several recorded tasks with no parents, **When** the user runs `list`, **Then** all of them appear as separate root entries, each showing identifier, name, and due date.
2. **Given** tasks nested two or more levels deep, **When** the user runs `list`, **Then** each task is displayed indented beneath its parent, reflecting the full hierarchy.
3. **Given** a task with no due date, **When** the user runs `list`, **Then** that task's line still appears, with the due date shown as empty or absent rather than causing an error.
4. **Given** no tasks recorded at all, **When** the user runs `list`, **Then** the tool reports that there are no tasks rather than failing.
5. **Given** recorded tasks, **When** the user runs `list`, **Then** entries sharing the same parent appear in identifier order (creation order, oldest first).

---

### User Story 3 - Modify an existing task (Priority: P3)

A person needs to correct or update a task they already recorded — rename it, change its due date, or move it under a different parent. They run a `mod` command, identifying the task and supplying its new name plus any fields they want to change.

**Why this priority**: Editing builds on captured tasks and is valuable, but the tool is already usable for capture and review without it.

**Independent Test**: Record a task, then run `mod` to change its name, its due date, and its parent in separate invocations, confirming after each that the change took effect and the task kept the same identifier.

**Acceptance Scenarios**:

1. **Given** an existing task, **When** the user runs `mod` with that task's identifier and a new name, **Then** the task is renamed and keeps the same identifier.
2. **Given** an existing task with a due date, **When** the user runs `mod` supplying a different `--due` value, **Then** the task's due date is updated to the new value.
3. **Given** an existing task, **When** the user runs `mod` supplying a `--parent` identifier of another existing task, **Then** the task is moved beneath that parent.
4. **Given** an existing task with a due date, **When** the user runs `mod` without supplying `--due`, **Then** the task's existing due date is left unchanged.
5. **Given** an identifier that matches no task, **When** the user runs `mod` for it, **Then** the command fails with a clear not-found error and nothing is changed.
6. **Given** an existing task, **When** the user runs `mod` with a `--parent` that would make the task its own ancestor, **Then** the command fails with a clear error and the task is left unchanged.

---

### User Story 4 - Remove a task (Priority: P4)

A person wants to discard a task that is finished or no longer relevant. They run an `rm` command with the task's identifier and the tool deletes it.

**Why this priority**: Removal keeps the list relevant over time, but the tool delivers value for capturing, reviewing, and editing tasks without it.

**Independent Test**: Record a task, run `rm` with its identifier, and confirm it no longer appears in the `list` output.

**Acceptance Scenarios**:

1. **Given** an existing task with no children, **When** the user runs `rm` with its identifier, **Then** the task is deleted and no longer appears in `list` output.
2. **Given** an existing task that has child tasks, **When** the user runs `rm` with its identifier, **Then** the task and all of its descendants are deleted.
3. **Given** an identifier that matches no task, **When** the user runs `rm` for it, **Then** the command fails with a clear not-found error.

---

### Edge Cases

- When the backend cannot be reached, every command fails with a clear connection error rather than a confusing crash, and returns a non-zero exit status.
- When a command fails for any reason (usage error, not-found, validation, connection), the tool returns a non-zero exit status so it composes correctly in scripts.
- A `--due` value that cannot be parsed as a timestamp is rejected with an error naming the expected format.
- An identifier argument (`<id>`, `--parent`) that is not a valid integer is rejected with a clear usage error.
- Running a command with no arguments, or an unrecognized subcommand, prints usage help.
- `rm` of a task with descendants removes the whole subtree (the backend cascades the delete); the user is operating on more than the single named task.
- A task name that exceeds the backend's maximum length is rejected; the tool surfaces the backend's validation error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST provide a `task` command group with four subcommands: `list`, `add`, `rm`, and `mod`.
- **FR-002**: `task list` MUST retrieve every task and display them as a tree drawn with `tree`-style ASCII connector glyphs (`├──`, `└──`, `│`, matching the Unix `tree` command), with tasks that have no parent at the root level and each subtask nested beneath its parent, to unrestricted depth.
- **FR-003**: `task list` MUST show, for each task, its identifier, its name, and its due date.
- **FR-004**: `task list` MUST order tasks that share the same parent by identifier ascending (creation order, oldest first).
- **FR-005**: `task list` MUST report an empty result clearly (e.g., a "no tasks" message) rather than failing when no tasks exist.
- **FR-006**: `task add` MUST accept a required task name as a positional argument and create a task with that name.
- **FR-007**: `task add` MUST accept an optional `--due` flag specifying the task's due date and an optional `--parent` flag specifying the identifier of the task to nest the new task under.
- **FR-008**: `task add` MUST report the identifier assigned to the newly created task on success.
- **FR-009**: `task rm` MUST accept a required task identifier as a positional argument and delete the matching task, reporting a brief confirmation line naming the deleted task's identifier on success.
- **FR-010**: `task mod` MUST accept a required task identifier and a required new name, update the matching task, and report a brief confirmation line naming the updated task's identifier on success.
- **FR-011**: `task mod` MUST accept an optional `--due` flag and an optional `--parent` flag to change those fields of the task.
- **FR-012**: `task mod` MUST leave a task's existing due date and parent unchanged when the corresponding flag is not supplied — only fields explicitly provided on the command line are changed.
- **FR-013**: The tool MUST validate that identifier arguments (`<id>`, `--parent`) are integers and that `--due` values are timestamps, rejecting malformed input with a clear error before contacting the backend.
- **FR-014**: The tool MUST surface backend errors — not-found, validation failure, and the prevention of parent cycles — to the user as clear, distinguishable messages.
- **FR-015**: The tool MUST return a zero exit status on success and exit status `1` on any failure (usage error, not-found, validation, cycle, or connection failure all share the single non-zero code).
- **FR-016**: The tool MUST report a clear error when the backend cannot be reached.
- **FR-017**: Every command MUST provide usage help describing its arguments and flags when invoked incorrectly or when help is requested.

### Key Entities

- **Task**: A unit of work managed by the backend. As surfaced by this tool, a task has an identifier (system-assigned integer), a name, an optional due date, and an optional parent task. The tool does not create or display a task's description; it neither sets nor relies on that field. Tasks form a hierarchy of unrestricted depth derived from parent references.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can add a task and then see it in `list` output, with the name, due date, and parent placement they supplied, 100% of the time.
- **SC-002**: `list` output represents the full task hierarchy: every task appears exactly once, indented under its correct parent, regardless of nesting depth.
- **SC-003**: A user modifying one field of a task (name, due date, or parent) leaves the task's other fields unchanged, 100% of the time.
- **SC-004**: Every failure case (missing name, malformed identifier or timestamp, unknown task, unknown parent, cycle, unreachable backend) produces a clear message identifying the cause and a non-zero exit status.
- **SC-005**: A new user can discover how to use any command from its built-in help without consulting external documentation.
- **SC-006**: Each command completes and returns its result within 2 seconds under normal single-user conditions against a local backend.

## Assumptions

- The tool is a client of the existing Task CRUD API backend (feature `001-task-crud-api`); that backend must be running and reachable for any command to succeed. The tool does not access the database directly.
- The backend address is configurable (for example, via an environment variable or a flag) and defaults to the local development server. Configuring it is a minor concern and not detailed further in this spec.
- `--due` values are supplied in one of two accepted forms: a full RFC 3339 date-time (e.g. `2026-06-01T17:00:00Z`), or a bare calendar date (e.g. `2026-06-01`) which is interpreted as midnight UTC (`00:00:00Z`) on that day. Due dates are displayed as absolute RFC 3339 timestamps. Tasks without a due date display an empty or blank due column.
- `task mod` uses fetch-then-update semantics: the tool reads the task's current state, applies the name and any supplied flags on top, and submits the complete state. This deliberately preserves the task's description (which this tool does not expose) and any field not named on the command line, avoiding accidental data loss from the backend's full-replace update behavior.
- Because only explicitly supplied fields are changed, this version provides no way to clear an existing due date or detach a task from its parent via `mod`; adding that capability is out of scope here.
- Task description, task completion/status, and any filtering or searching of the list are out of scope: the backend does not yet support completion or filtering, and description is intentionally excluded.
- The tool operates for a single trusted user; authentication and authorization are out of scope, consistent with the backend.
- `rm` performs the backend's cascade delete: deleting a task with descendants deletes the whole subtree. The tool does not prompt for additional confirmation.
