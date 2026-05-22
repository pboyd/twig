# Feature Specification: CLI Help System

**Feature Branch**: `008-cli-help`

**Created**: 2026-05-22

**Status**: Draft

**Input**: User description: "Add help to the CLI. Add a help command and ensure that each subcommand has appropriate help information."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Get top-level help (Priority: P1)

A user unfamiliar with the tool (or returning after a break) runs `todo help` or `todo --help` to see all available commands and a brief description of each.

**Why this priority**: The root help is the entry point for everything else. It must exist before per-command help is meaningful.

**Independent Test**: Run `todo help` (and `todo --help`) and confirm it prints a summary of all top-level commands (`task`, `pom`, `plan`) with a one-line description each.

**Acceptance Scenarios**:

1. **Given** the CLI is installed, **When** the user runs `todo help`, **Then** the output lists all top-level commands with a brief description and exits with code 0.
2. **Given** the CLI is installed, **When** the user runs `todo --help`, **Then** the same help text is shown as `todo help`.
3. **Given** the CLI is installed, **When** the user runs `todo` with no arguments, **Then** the root help is shown and the process exits with a non-zero code (unchanged from current behavior).

---

### User Story 2 - Get help for a specific command (Priority: P1)

A user wants to know the subcommands and flags for `task`, `pom`, or `plan`. They run `todo help <command>` (or `todo <command> --help`) and receive a detailed usage summary for that command.

**Why this priority**: Per-command help is the primary daily use of a help system. Equal priority to root help since both are needed for a coherent experience.

**Independent Test**: Run `todo help task`, `todo help pom`, and `todo help plan` individually and confirm each prints its subcommands, their arguments, and their flags, then exits with code 0.

**Acceptance Scenarios**:

1. **Given** the CLI, **When** the user runs `todo help task`, **Then** the output describes the `task` command, its subcommands (`add`, `rm`, `mod`, `complete`), and their flags (`--parent`, `--due`, `--completed`, `--all`).
2. **Given** the CLI, **When** the user runs `todo help pom`, **Then** the output describes the `pom` command and its subcommands (`estimate`, `start`, `resume`, `cancel`, `status`).
3. **Given** the CLI, **When** the user runs `todo help plan`, **Then** the output describes the `plan` command and its subcommands (`task`, `event`, `rm`, `rename`, `mv`, `clear`), including time and duration format examples.
4. **Given** the CLI, **When** the user runs `todo task --help`, **Then** the same help text is shown as `todo help task`.
5. **Given** the CLI, **When** the user runs `todo help unknown`, **Then** the CLI prints an error indicating the unknown command and exits non-zero.

---

### User Story 3 - Discover help from error messages (Priority: P2)

When a user types an invalid command or subcommand, the error message tells them how to get help, rather than just printing "unknown command."

**Why this priority**: Helps users self-rescue after mistyping. Lower priority than explicit help commands since the help commands themselves must exist first.

**Independent Test**: Run `todo badcmd` and `todo task badsubcmd`; confirm the error message references how to get help (e.g., "Run 'todo help' for usage.").

**Acceptance Scenarios**:

1. **Given** the CLI, **When** the user runs `todo badcmd`, **Then** the error message includes a hint pointing to `todo help`.
2. **Given** the CLI, **When** the user runs `todo task badsubcmd`, **Then** the error message includes a hint pointing to `todo help task`.

---

### Edge Cases

- `todo help help`: should either show root help or a brief description of the help command itself; must not crash.
- Help output should go to stdout (not stderr) when invoked via `todo help` or `--help` flags, so it can be piped or captured.
- When the terminal does not support styling, help output must remain fully readable as plain text.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI MUST accept `todo help` as a valid command and display top-level usage information, exiting with code 0.
- **FR-002**: The CLI MUST accept `todo --help` (at the root level) as equivalent to `todo help`.
- **FR-003**: The CLI MUST accept `todo help <command>` where `<command>` is `task`, `pom`, or `plan`, and display detailed usage for that command, exiting with code 0.
- **FR-004**: The CLI MUST accept `todo <command> --help` as equivalent to `todo help <command>`.
- **FR-005**: Help output for `todo help` (or `--help`) MUST include each top-level command name and a one-line description.
- **FR-006**: Help output for `todo help task` MUST document all `task` subcommands (`add`, `rm`, `mod`, `complete`) with their arguments and flags.
- **FR-007**: Help output for `todo help pom` MUST document all `pom` subcommands (`estimate`, `start`, `resume`, `cancel`, `status`) with their arguments and flags.
- **FR-008**: Help output for `todo help plan` MUST document all `plan` subcommands and include time/duration format examples.
- **FR-009**: When the user runs an unknown top-level command, the error message MUST include a hint directing the user to `todo help`.
- **FR-010**: When the user runs an unknown subcommand under `task`, `pom`, or `plan`, the error message MUST include a hint directing the user to `todo help <command>`.
- **FR-011**: Help output produced by `todo help` or `todo help <command>` MUST be written to stdout.
- **FR-012**: Help output produced when no arguments are given (current behavior: exits non-zero) MUST continue to exit non-zero; the help text MAY be written to stderr in that case.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can retrieve usage information for every command and subcommand in one invocation (`todo help` or `todo help <command>`) without needing to read source code or external documentation.
- **SC-002**: Every error path for an unknown command or subcommand includes a reference to the appropriate help command, reducing the need to guess or search.
- **SC-003**: `todo help` and `todo <command> --help` produce identical output to `todo help <command>`, so users can discover help through any natural path.
- **SC-004**: Help output is fully readable when captured to a file or piped (no raw escape codes, content goes to stdout).

## Assumptions

- The CLI has a single user (the project author), so no internationalization or accessibility requirements beyond plain-text degradation.
- Existing usage text in `printRootUsage`, `printTaskUsage`, `printPomUsage`, and `printPlanUsage` will be retained and improved in-place rather than replaced by a third-party help library.
- `--help` at the subcommand level (e.g., `todo task add --help`) is out of scope; only top-level `--help` and `todo <command> --help` are required.
- No man-page or external documentation generation is in scope.
