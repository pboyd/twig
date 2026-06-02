# Feature Specification: CLI Task Uncomplete

**Feature Branch**: `032-cli-task-uncomplete`

**Created**: 2026-06-02

**Status**: Draft

**Input**: User description: "recently we added the ability to uncomplete a task in the TUI and web UI, but it was never surfaced in the CLI. `twig task uncomplete <id>` should mark a completed task incomplete."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reopen a completed task from the command line (Priority: P1)

A user who completed a task by mistake, or whose completed task needs more work, wants to mark it incomplete again using the CLI without switching to the TUI or web UI.

**Why this priority**: This is the entire feature — a single missing command that brings the CLI to parity with other interfaces. It stands fully on its own and delivers all intended value.

**Independent Test**: Complete a task via `twig task complete <id>`, then run `twig task uncomplete <id>` and confirm the task no longer appears as completed.

**Acceptance Scenarios**:

1. **Given** a completed task with id 42, **When** the user runs `twig task uncomplete 42`, **Then** the task is marked incomplete and the CLI outputs a confirmation message such as `uncompleted task 42`.
2. **Given** an incomplete task with id 7, **When** the user runs `twig task uncomplete 7`, **Then** the command succeeds and notifies the user the task was already incomplete (idempotent, no error).
3. **Given** a non-existent task id, **When** the user runs `twig task uncomplete 999`, **Then** the command exits with a non-zero status and prints a clear error to stderr.
4. **Given** a non-integer argument, **When** the user runs `twig task uncomplete abc`, **Then** the command exits with a non-zero status and prints a usage error to stderr.
5. **Given** no arguments, **When** the user runs `twig task uncomplete`, **Then** the command exits with a non-zero status and prints usage instructions to stderr.

---

### Edge Cases

- What happens when the task id belongs to a deleted or archived task?
- What happens when the user's API key does not have permission to modify the task?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI MUST expose a `twig task uncomplete <id>` subcommand.
- **FR-002**: The command MUST accept exactly one positional argument: an integer task id.
- **FR-003**: Running the command on a completed task MUST mark that task incomplete and print a confirmation to stdout.
- **FR-004**: Running the command on an already-incomplete task MUST succeed without error and inform the user the task was already incomplete.
- **FR-005**: Passing a non-integer value for `<id>` MUST print a usage error to stderr and exit with a non-zero status.
- **FR-006**: Omitting `<id>` MUST print usage instructions to stderr and exit with a non-zero status.
- **FR-007**: Server-side errors (task not found, network failure, auth failure) MUST be surfaced as human-readable messages on stderr with a non-zero exit code.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can mark a completed task incomplete with a single command in under 2 seconds.
- **SC-002**: All error conditions (missing argument, invalid id, unknown task) produce actionable, human-readable output on stderr.
- **SC-003**: The command behaves consistently with `twig task complete` in output style, argument handling, and error reporting.

## Assumptions

- The `UncompleteTask` RPC endpoint already exists on the server and is accessible to CLI clients — this feature adds only the CLI surface.
- The existing `twig task complete <id>` command serves as the direct implementation model; `uncomplete` mirrors its structure exactly.
- No new flags or options are needed beyond the required `<id>` argument.
- Output format follows the same pattern as `complete`: a short confirmation line to stdout on success.
