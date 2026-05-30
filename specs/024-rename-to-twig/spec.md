# Feature Specification: Rename App to Twig

**Feature Branch**: `024-rename-to-twig`

**Created**: 2026-05-30

**Status**: Draft

**Input**: User description: "Rename the app from "todo" to "twig" — warm/playful task manager where tasks are the soul of the app, with a nested tree structure, Pomodoro timer, and daily planning features."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Use the renamed CLI binary (Priority: P1)

A user who has the app installed runs `twig` from the command line to manage their tasks, start Pomodoro sessions, and plan their day. The old `todo` command no longer exists and all documentation references the new name.

**Why this priority**: The CLI binary name is the primary interface — it's the first thing users type and the most disorienting thing if it differs from documentation.

**Independent Test**: Install the built binary and run `twig task list` — it returns results. Running `todo` returns "command not found."

**Acceptance Scenarios**:

1. **Given** the app is installed, **When** a user runs `twig`, **Then** the CLI responds correctly with help output or task list.
2. **Given** the app is installed, **When** a user runs `twig pom`, **Then** the Pomodoro timer starts as expected.
3. **Given** the app is installed, **When** a user runs `twig plan`, **Then** the daily planning flow starts as expected.

---

### User Story 2 - Configure the app using the new environment variable names (Priority: P2)

A user sets `TWIG_API_KEY` and `TWIG_ADDR` in their shell environment (or config file) to connect the CLI to the server. The old `TODO_API_KEY` and `TODO_ADDR` variable names no longer work.

**Why this priority**: Without correct env var names, the CLI cannot authenticate — making this a blocker for all CLI use after upgrade.

**Independent Test**: Set `TWIG_API_KEY=test` and run `twig task list` against a test server — the key is sent. Setting only `TODO_API_KEY` has no effect.

**Acceptance Scenarios**:

1. **Given** `TWIG_API_KEY` is set, **When** a user runs a CLI command, **Then** the API key is used for authentication.
2. **Given** `TWIG_ADDR` is set, **When** a user runs a CLI command, **Then** the CLI connects to the specified address.
3. **Given** only legacy `TODO_API_KEY` is set (no `TWIG_API_KEY`), **When** a user runs a CLI command, **Then** the CLI reports that `TWIG_API_KEY` is not set.

---

### User Story 3 - Use the config file at the new path (Priority: P3)

A user's config file lives at `~/.config/twig/config.toml`. The app no longer reads from `~/.config/todo/config.toml`.

**Why this priority**: Affects users who rely on the config file rather than environment variables. Existing config files at the old path will be silently ignored, which could cause confusing behavior, but this is lower priority than the binary and env var renames.

**Independent Test**: Place a `config.toml` at `~/.config/twig/config.toml` with a valid API key, run `twig task list` — the key from the config file is used. A file at `~/.config/todo/config.toml` is ignored.

**Acceptance Scenarios**:

1. **Given** a config file at `~/.config/twig/config.toml`, **When** the CLI starts, **Then** it reads the API key and address from that file.
2. **Given** a config file only at the old path `~/.config/todo/config.toml`, **When** the CLI starts with no env vars set, **Then** the CLI treats the config as absent.

---

### User Story 4 - See the new name in the web UI (Priority: P4)

A user visiting the web interface sees "Twig" as the app name in the page title, header, and any branding elements. No "Todo" branding appears in the UI.

**Why this priority**: Purely cosmetic — the app functions correctly regardless of displayed name. Still important for brand coherence.

**Independent Test**: Load the web app — the browser tab and header say "Twig," not "Todo."

**Acceptance Scenarios**:

1. **Given** the web app is open, **When** a user looks at the page title or app header, **Then** they see "Twig."
2. **Given** the web app is open, **Then** no visible text reads "Todo" as a product name.

---

### Edge Cases

- What happens to users who have `TODO_API_KEY` set in their shell? They will see an "API key not set" error and need to rename the variable — the error message should name `TWIG_API_KEY` clearly.
- What happens to users with config at `~/.config/todo/config.toml`? They must move it — no automatic migration.
- How does the system handle the `todo-server` binary being referenced in deployment scripts? This is out of scope for the spec; deployment tooling changes are handled separately.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI binary MUST be named `twig`.
- **FR-002**: The server binary MUST be named `twig-server`.
- **FR-003**: The CLI MUST read its API key from the `TWIG_API_KEY` environment variable.
- **FR-004**: The CLI MUST read the server address from the `TWIG_ADDR` environment variable.
- **FR-005**: The CLI MUST read configuration from `~/.config/twig/config.toml` (respecting `$XDG_CONFIG_HOME`).
- **FR-006**: The CLI MUST NOT read from the legacy environment variables `TODO_API_KEY` or `TODO_ADDR`.
- **FR-007**: The CLI MUST NOT read from the legacy config path `~/.config/todo/config.toml`.
- **FR-008**: Error messages that reference missing configuration MUST use the new variable and path names (`TWIG_API_KEY`, `TWIG_ADDR`, `~/.config/twig/config.toml`).
- **FR-009**: The web app MUST display "Twig" as the application name in all user-visible branding (page title, app header).
- **FR-010**: All user-facing help text, usage strings, and error messages in the CLI MUST reference `twig`, not `todo`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A fresh install of the app allows a user to authenticate and run commands using only `TWIG_API_KEY` and `TWIG_ADDR`, with no reference to the old names needed.
- **SC-002**: Running `twig --help` displays usage information that contains no references to "todo" as a product name.
- **SC-003**: The web app page title and visible header contain "Twig" and no legacy "Todo" branding.
- **SC-004**: All existing automated tests pass after the rename with no test referencing the old binary name `todo` as the production name.

## Assumptions

- The Go module path (`services/todo/`) and internal package names are internal implementation details — they are not user-facing and are out of scope for this rename. Only user-facing names (binary, env vars, config path, UI text) change.
- The PostgreSQL database name and connection strings are infrastructure concerns managed separately; they are out of scope.
- No automatic migration of existing user config files or environment variable names is provided. Users must update their own configurations.
- The web app service directory name (`services/todo-web/`) is an internal directory name and is out of scope.
- The `specs/` directory contents and existing spec files are not renamed; only new work uses the "twig" name.
