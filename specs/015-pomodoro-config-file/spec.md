# Feature Specification: Pomodoro Config File

**Feature Branch**: `015-pomodoro-config-file`

**Created**: 2026-05-28

**Status**: Draft

**Input**: User description: "Specifying an `--exec` command whenever starting or resuming a pomodoro is cumbersome, and there's no way to even set it in TUI. Users will probably want the same command to run every time, so it makes sense to configure the command in a config file. The `--exec` flag can be removed completely. In addition to a command that runs after the pomodoro ends, it would be helpful to add commands to run when the pomodoro begins and is canceled (the use-case is turning on and off do-not-disturb). Since there will be a config file, it should have options to set the API URL and auth token too."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Persistent pomodoro hook commands via config file (Priority: P1)

A user wants the same shell command to run every time a pomodoro completes (e.g., a notification or a system audio cue), without typing `--exec "..."` every time they start or resume a pomodoro, and without losing that capability in the TUI where there is no place to type CLI flags.

**Why this priority**: This is the core motivation for the change. Without it, the `--exec` flag remains the only way to attach a completion command, and TUI users have no way to use the feature at all.

**Independent Test**: Create a config file specifying a completion command, start a pomodoro from either CLI or TUI, let it run to completion, and confirm the configured command executed exactly once with the configured arguments. No `--exec` flag is supplied.

**Acceptance Scenarios**:

1. **Given** a config file with a completion command set, **When** a user starts a pomodoro from the CLI (`todo pom start <id>`), **Then** the configured command runs when the pomodoro completes.
2. **Given** a config file with a completion command set, **When** a user starts a pomodoro from the TUI, **Then** the configured command runs when the pomodoro completes.
3. **Given** a config file with a completion command set, **When** a user resumes an in-progress pomodoro (`todo pom resume`), **Then** the configured command runs when the pomodoro completes.
4. **Given** no config file exists, **When** a pomodoro completes, **Then** no hook command runs and no error is shown.
5. **Given** an invocation that includes the removed `--exec` flag, **When** the user runs the command, **Then** the CLI reports an unknown-flag error rather than silently ignoring it.

---

### User Story 2 - Hooks for pomodoro start and cancel (Priority: P2)

A user wants to automatically enable Do Not Disturb when a pomodoro begins and disable it when the pomodoro is canceled (or completes). They configure separate commands for the start, cancel, and completion events.

**Why this priority**: This delivers the secondary motivation (DND automation). It depends on User Story 1 being in place but extends its value substantially.

**Independent Test**: Configure distinct commands for start, cancel, and complete events. Start a pomodoro and verify the start command runs. Cancel it and verify the cancel command runs (and the complete command does not). Start another and let it complete; verify the complete command runs (and the cancel command does not).

**Acceptance Scenarios**:

1. **Given** a config file with a start-hook command, **When** a user starts a pomodoro, **Then** the start-hook command runs once immediately after the pomodoro begins.
2. **Given** a config file with a cancel-hook command, **When** a user cancels an active pomodoro (CLI or TUI), **Then** the cancel-hook command runs once and the completion-hook command does not run.
3. **Given** a config file with start, cancel, and complete hooks, **When** a pomodoro runs to completion, **Then** only the start and complete hooks run.
4. **Given** any hook command exits with a non-zero status, **When** the hook runs, **Then** the pomodoro itself continues normally and the failure is surfaced as a non-fatal warning to the user.
5. **Given** a hook command is not set in the config, **When** the corresponding event occurs, **Then** no command runs and no warning is shown.

---

### User Story 3 - Configure server address and auth token in the config file (Priority: P3)

A user wants to avoid setting `TODO_ADDR` and `TODO_API_KEY` environment variables in every shell. They put both values into the config file once and the CLI uses them automatically.

**Why this priority**: Quality-of-life improvement that piggybacks on the new config file. Independent of the hook work; the existing env-var mechanism already works for users who prefer it.

**Independent Test**: With `TODO_ADDR` and `TODO_API_KEY` unset in the environment but set in the config file, run any CLI command that talks to the server (e.g., `todo task list`) and confirm it connects successfully using the configured values.

**Acceptance Scenarios**:

1. **Given** a config file with `api_url` and `api_key` set and no corresponding environment variables, **When** the user runs any server-backed CLI command, **Then** the command authenticates and reaches the configured server.
2. **Given** both an environment variable (`TODO_API_KEY` or `TODO_ADDR`) and a config file value are present, **When** the user runs a CLI command, **Then** the environment variable takes precedence over the config value.
3. **Given** neither the environment variable nor the config file provides an API key, **When** the user runs a server-backed command, **Then** the CLI reports a clear error explaining where the key can be set.

---

### Edge Cases

- The config file exists but is malformed (syntactically invalid). The CLI must report a clear, actionable error pointing at the file path and refuse to start, rather than silently falling back to defaults.
- The config file references a hook command that is not on `$PATH`. The hook exec fails; the pomodoro itself continues and a non-fatal warning is shown.
- A start hook is configured but the pomodoro server-side start call fails. The start hook must NOT run, because no pomodoro actually began.
- Cancel happens because the user exits the TUI / terminates the CLI process before completion. The cancel hook runs on explicit user-initiated cancel; ungraceful termination (SIGKILL, terminal close, machine sleep) does not invoke any hook, matching today's behavior.
- The same command is configured for multiple events (start and cancel). Each event still triggers an independent invocation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST read a user-level configuration file at a well-known path on CLI startup. The file is optional; absence is not an error.
- **FR-002**: The configuration file MUST support, at minimum, the following settings: API server URL, API auth token, pomodoro start-hook command, pomodoro cancel-hook command, and pomodoro completion-hook command.
- **FR-003**: The `--exec` flag MUST be removed from `todo pom start` and `todo pom resume`. Passing `--exec` MUST result in an unknown-flag error.
- **FR-004**: When a pomodoro begins (start, or resume of a not-yet-running pomodoro from the user's perspective), the system MUST execute the configured start-hook command, if any, exactly once.
- **FR-005**: When a pomodoro is explicitly canceled by the user, the system MUST execute the configured cancel-hook command, if any, exactly once, and MUST NOT execute the completion-hook command.
- **FR-006**: When a pomodoro completes naturally (timer reaches zero / server reports completion), the system MUST execute the configured completion-hook command, if any, exactly once, and MUST NOT execute the cancel-hook command.
- **FR-007**: Hook commands MUST be executed via the user's shell so common shell syntax (pipes, quoting, environment expansion) works in the configured string.
- **FR-008**: A hook command failure (non-zero exit, command not found) MUST NOT abort or alter the pomodoro itself. The failure MUST be surfaced as a non-fatal warning to the user.
- **FR-009**: When both an environment variable and a config-file value provide the API URL or API key, the environment variable MUST take precedence. This preserves the current env-var workflow as an override.
- **FR-010**: When no API key is available from either the environment or the config file, the CLI MUST report an actionable error that names both possible sources.
- **FR-011**: A malformed config file MUST cause the CLI to fail fast with a clear error identifying the file path and the parse problem; it MUST NOT silently fall back to defaults.
- **FR-012**: Hooks MUST behave identically whether a pomodoro is driven from the CLI subcommands or from the interactive TUI.
- **FR-013**: The system MUST document the config file location, format, and supported keys in user-visible help text or documentation.

### Key Entities

- **Config file**: A single user-level file containing optional settings. Logical keys: API URL, API key, pomodoro on-start command, pomodoro on-cancel command, pomodoro on-complete command. Exact field names, file location, and format are implementation details for the planning phase.
- **Hook command**: A user-supplied shell command string associated with one of three pomodoro lifecycle events (start, cancel, complete).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can configure a completion command once in the config file and have it run on 100% of pomodoros completed thereafter (CLI and TUI) without any per-invocation flags.
- **SC-002**: Configuring start and cancel hooks enables a user to fully automate enabling/disabling Do Not Disturb across the pomodoro lifecycle with zero per-invocation input.
- **SC-003**: Users who put API URL and API key into the config file can run any server-backed CLI command in a fresh shell with no `TODO_*` environment variables set.
- **SC-004**: No existing scripted workflow that relies on `TODO_API_KEY` or `TODO_ADDR` environment variables breaks after this change (env vars continue to work and override config values).
- **SC-005**: A user encountering a malformed config file receives an error message that identifies the file path within 5 seconds of running any CLI command, allowing them to locate and fix the file unaided.

## Assumptions

- The config file lives at the standard per-user config location for the platform (e.g., `$XDG_CONFIG_HOME/todo/config.toml` or `~/.config/todo/config.toml` on Linux). Exact path and file format are implementation choices to be settled in planning.
- Hook commands are executed synchronously in the foreground of the CLI/TUI process. The start hook completes (or fails) before the countdown UI begins; the complete and cancel hooks run before the CLI/TUI returns the user to a prompt.
- "Cancel" means an explicit user-initiated cancel via the existing cancel pathway in CLI or TUI. Ungraceful termination does NOT trigger the cancel hook; this matches today's behavior, where no cleanup runs in those cases either.
- Existing `TODO_API_KEY` and `TODO_ADDR` environment variables continue to be supported and take precedence over config-file values, preserving backward compatibility for scripted setups.
- Only the CLI/TUI client reads this config file. The server has its own configuration mechanism and is out of scope.
- Hooks are global to the user in v1 — they are not configurable per task or per project.
