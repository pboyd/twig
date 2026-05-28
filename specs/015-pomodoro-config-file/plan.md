# Implementation Plan: Pomodoro Config File

**Branch**: `015-pomodoro-config-file` | **Date**: 2026-05-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/015-pomodoro-config-file/spec.md`

## Summary

Introduce a user-level config file for the `todo` CLI/TUI that supplies (a) the API URL and auth token, replacing the strict `TODO_API_KEY`/`TODO_ADDR` env-var requirement (env vars still win when set), and (b) three pomodoro lifecycle hooks — `on_start`, `on_cancel`, `on_complete` — replacing the existing `--exec` flag, which is removed. The config is parsed once at CLI startup and threaded through to both the CLI subcommand path and the TUI pomodoro path so hooks fire identically in both.

## Technical Context

**Language/Version**: Go 1.23+ (existing module at `services/todo/`)

**Primary Dependencies**: `connectrpc.com/connect`, `github.com/charmbracelet/bubbletea` (TUI), `github.com/jackc/pgx/v5` (server-side, unaffected). For TOML parsing: `github.com/BurntSushi/toml` (Go-standard, single-file dependency, BSD).

**Storage**: N/A — config is a flat file on disk; no DB interaction in this feature.

**Testing**: `go test ./...` with table-driven tests; hook execution tests will use a temporary shell script as the configured command so we can assert it ran without mocking `exec.Command`.

**Target Platform**: Linux/macOS CLI (TTY + TUI). Windows is not currently supported by this project.

**Project Type**: CLI/TUI client of an HTTP/2 ConnectRPC server. This feature touches only the client side (`services/todo/cmd/todo` + `internal/cli` + `internal/tui` + a new `internal/config` package).

**Performance Goals**: Config load adds <10 ms to CLI startup (single file read + TOML parse).

**Constraints**: Backward compatibility with existing env-var workflows (FR-009). No new server-side changes. No new persistent state.

**Scale/Scope**: Single user, single machine. ~5 config keys, 3 hook events.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Principle I — Simplicity / YAGNI**: PASS.
- One new internal package (`internal/config`) with a single `Load()` function returning a `Config` struct.
- No interfaces, no plugin system, no per-task hooks. Three hook events, plain shell-string commands, exactly what the spec calls for.
- `--exec` is *removed*, not aliased or deprecated. Removing code is cheaper than maintaining two paths.
- Existing helpers (`execHook`, env-var lookup) are reused; only the *source* of values changes.

**Principle II — API-First Design**: PASS (with note).
- This feature touches no RPC contracts — it is purely a client-side configuration change. No `proto/`, no `gen/`, no `internal/handler/` work.
- The "contract" for this feature is the config-file schema, documented in `contracts/config-schema.md`.

No violations; Complexity Tracking table is empty.

## Project Structure

### Documentation (this feature)

```text
specs/015-pomodoro-config-file/
├── plan.md              # This file
├── spec.md              # Feature spec
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (config struct + precedence rules)
├── quickstart.md        # Phase 1 output (user-facing config walkthrough)
├── contracts/
│   └── config-schema.md # The config-file schema (the "API" for this feature)
└── checklists/
    └── requirements.md  # From /speckit-specify
```

### Source Code (repository root)

```text
services/todo/
├── cmd/
│   └── todo/                          # CLI entry point (no change expected)
├── internal/
│   ├── config/                        # NEW package
│   │   ├── config.go                  # Config struct, Load(), precedence resolution
│   │   └── config_test.go             # Table-driven tests for load/precedence/errors
│   ├── cli/
│   │   ├── cli.go                     # MODIFIED: read config; use config-or-env for addr/key
│   │   ├── pom.go                     # MODIFIED: remove --exec parsing; call config hooks
│   │   ├── pom_test.go                # MODIFIED: tests now cover config-driven hooks
│   │   ├── plan.go                    # MODIFIED: addr/key from config helper
│   │   └── task.go                    # MODIFIED: addr/key from config helper
│   └── tui/
│       ├── client.go                  # MODIFIED: addr/key from config helper
│       └── pomodoro.go                # MODIFIED: invoke hooks on start/cancel/complete
```

**Structure Decision**: Single Go module, single binary client. A small new `internal/config` package owns parsing and precedence; CLI and TUI both call into it. Hook execution stays where it already lives (`internal/cli/pom.go::execHook`), but now the *trigger* and *command source* come from config rather than a CLI flag. The TUI's pomodoro path also calls the same hook function so behavior is identical (FR-012).

## Complexity Tracking

> Empty — no Constitution Check violations.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| —         | —          | —                                    |
