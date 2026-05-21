# Feature Specification: CLI UX Improvements

**Feature Branch**: `007-cli-ux-improvements`

**Created**: 2026-05-21

**Status**: Draft

**Input**: User description: "Miscellaneous UX improvements to the todo CLI: make `todo task` default to listing tasks (and remove the `list` subcommand), render completed tasks with gray strikethrough instead of `[x]`, fix `todo task mod` so the name is optional and flags are accepted after the id, and display estimated pomodoros in parentheses after the task name."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Re-parent a task without renaming it (Priority: P1)

A user wants to change a task's parent without altering its name. They run `todo task mod <id> --parent <new-parent-id>` and expect only the parent to change. Today the command treats `--parent` as the new name, which is surprising and destructive.

**Why this priority**: This is a correctness bug that silently destroys data (overwrites the task name). It's the most impactful issue in this batch because it can cause unintended loss of task content.

**Independent Test**: Run `todo task mod <id> --parent <new-parent-id>` against a task with a known name; confirm the name is unchanged and the parent is updated. Also verify flags placed after the id are parsed as flags rather than as positional arguments.

**Acceptance Scenarios**:

1. **Given** a task with id 3 named "Write docs" and parent 1, **When** the user runs `todo task mod 3 --parent 5`, **Then** task 3 keeps the name "Write docs" and its parent becomes 5.
2. **Given** a task with id 3, **When** the user runs `todo task mod 3 "New name" --parent 5`, **Then** task 3's name becomes "New name" and its parent becomes 5.
3. **Given** a task with id 3, **When** the user runs `todo task mod 3 --parent 5 "New name"` (flag before positional name), **Then** the command succeeds with name "New name" and parent 5.

---

### User Story 2 - List tasks with a shorter command (Priority: P2)

A user frequently lists tasks. They want `todo task` (with no subcommand) to show the task list, mirroring how `todo plan` works. The explicit `todo task list` subcommand is removed.

**Why this priority**: High-frequency ergonomic improvement that aligns the `task` and `plan` commands. Not a correctness issue, so ranked below the bug fix.

**Independent Test**: Run `todo task` with no arguments and confirm it produces the same output as today's `todo task list`. Confirm `todo task list` is no longer a recognized subcommand.

**Acceptance Scenarios**:

1. **Given** any task list state, **When** the user runs `todo task`, **Then** the CLI prints the task list (the output previously produced by `todo task list`).
2. **Given** the new CLI, **When** the user runs `todo task list`, **Then** the CLI reports an unknown subcommand (or equivalent help/error), because `list` no longer exists.

---

### User Story 3 - Visually distinguish completed tasks (Priority: P2)

When viewing the task list, a user wants completed tasks to be easy to skim past. Instead of the `[x]` checkbox marker, completed tasks should render in gray with a strikethrough so they recede visually while incomplete tasks stand out.

**Why this priority**: Improves daily readability of the most-used view. Independent of the other changes.

**Independent Test**: Create a mix of completed and incomplete tasks, run the task list command in a terminal that supports ANSI styling, and verify completed task lines appear in gray with strikethrough while incomplete ones use the default style.

**Acceptance Scenarios**:

1. **Given** a list containing both completed and incomplete tasks, **When** the user runs `todo task`, **Then** completed tasks appear in gray with strikethrough text and incomplete tasks appear in the default style.
2. **Given** the task list output, **When** rendered, **Then** the literal `[x]` / `[ ]` checkbox marker is no longer used to indicate completion status.

---

### User Story 4 - See estimated pomodoros next to task names (Priority: P3)

A user planning their day wants to see at a glance how many pomodoros each task is estimated to take. When a task has an estimate, it appears in parentheses after the task name (e.g., `Write docs (3)`).

**Why this priority**: Useful planning aid, but only meaningful for users who set estimates. Lower urgency than the other changes.

**Independent Test**: Create tasks with and without pomodoro estimates, run `todo task`, and confirm the estimate appears in parentheses after the name only for tasks that have one.

**Acceptance Scenarios**:

1. **Given** a task named "Write docs" with an estimate of 3 pomodoros, **When** the user runs `todo task`, **Then** that line displays `Write docs (3)` (with surrounding formatting consistent with the rest of the list).
2. **Given** a task with no pomodoro estimate, **When** the user runs `todo task`, **Then** no parenthesized count is appended to its name.

---

### Edge Cases

- `todo task mod <id>` with no name and no flags: should be a no-op or an error indicating nothing to change, consistent with existing CLI conventions for empty modifications.
- `todo task mod <id> ""`: an explicitly empty name should be rejected, not silently applied.
- Terminals that do not support ANSI styling (e.g., output piped to a file): completion styling should degrade gracefully and not insert raw escape codes that obscure the task text.
- Pomodoro estimate of 0: treat as "no estimate" and omit the parenthesized suffix (see Assumptions).
- `todo task mod 3 --parent 5 "Some name"` vs `todo task mod 3 "Some name" --parent 5`: both orderings must produce the same result.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Running `todo task` with no subcommand MUST produce the same output as the previous `todo task list` subcommand.
- **FR-002**: The CLI MUST no longer expose `list` as a subcommand of `task`; invoking it MUST result in standard "unknown subcommand" behavior consistent with the rest of the CLI.
- **FR-003**: In the task list output, completed tasks MUST be rendered with gray foreground color and strikethrough text decoration, and MUST NOT use the `[x]` checkbox marker.
- **FR-004**: In the task list output, incomplete tasks MUST be rendered in the default (non-gray, non-strikethrough) style and MUST NOT use the `[ ]` checkbox marker.
- **FR-005**: When the output is not a styled terminal (e.g., piped or redirected), the CLI MUST degrade gracefully so task text remains readable without raw escape sequences interfering with output, reusing existing CLI styling conventions.
- **FR-006**: The `todo task mod` command MUST treat the task name positional argument as optional; when omitted, the task's name MUST remain unchanged.
- **FR-007**: The `todo task mod` command MUST accept flags (e.g., `--parent`) in any position relative to the id and the optional name, including immediately after the id with no name supplied.
- **FR-008**: `todo task mod <id> --parent <p>` MUST update only the parent of the task and MUST NOT modify the task name.
- **FR-009**: `todo task mod <id>` with no name and no modifying flags MUST NOT silently rename the task and SHOULD report that no changes were specified.
- **FR-010**: In the task list output, when a task has an estimated number of pomodoros, the estimate MUST be displayed in parentheses immediately after the task name (e.g., `Task name (N)`).
- **FR-011**: In the task list output, when a task has no estimated number of pomodoros, no parenthesized count MUST be appended to its name.

### Key Entities

- **Task**: An existing entity in the system, with at minimum a name, a completion state (complete / incomplete), an optional parent reference, and an optional estimated number of pomodoros. This feature changes how the task is displayed and edited, not its underlying data shape.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can re-parent a task without altering its name in a single command invocation; 0% of `todo task mod <id> --parent <p>` invocations result in the task being renamed.
- **SC-002**: A user listing tasks types one fewer word (`todo task` vs `todo task list`) to reach the most-used view; the previous `list` subcommand no longer appears in help output.
- **SC-003**: In a terminal supporting styling, completed and incomplete tasks are visually distinguishable at a glance, with completed tasks de-emphasized via gray + strikethrough and no checkbox markers in the output.
- **SC-004**: For every task with a pomodoro estimate, the estimate appears in parentheses immediately after the name in the task list; for every task without one, no parentheses appear.

## Assumptions

- The terminal styling library already in use for the CLI supports gray foreground and strikethrough; if not, existing conventions for graceful degradation (e.g., when piped) are reused.
- A pomodoro estimate of 0 is equivalent to "no estimate" and should not render parentheses.
- Removing `todo task list` is acceptable as a breaking CLI change because this tool has a single user (the project author), matching the precedent set by the `plan` command.
- The flag-parsing fix for `todo task mod` applies only to that subcommand; no audit of other subcommands for similar positional-vs-flag ambiguity is in scope.
- "Gray" is interpreted as a dim/secondary foreground color consistent with whatever the CLI already uses for de-emphasized text; no new color palette is being introduced.
