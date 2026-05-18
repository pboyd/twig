# Implementation Plan: Todo CLI Tool

**Branch**: `002-todo-cli` | **Date**: 2026-05-18 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-todo-cli/spec.md`

## Summary

Add a `todo` command-line tool that is a client of the existing `TaskService`
(feature `001-task-crud-api`). It exposes a `task` command group with four
subcommands — `list`, `add`, `rm`, `mod` — that call the backend's Connect-Go
RPCs. `list` renders every task as a `tree`-style hierarchy; `add`/`mod` build
requests from positional name plus `--parent`/`--due` flags; `mod` uses
fetch-then-update to survive the backend's full-replace `UpdateTask`. The tool
ships as a second binary (`cmd/todo`) inside the existing `services/todo` Go
module so it can import the generated `taskv1connect` client directly — no new
module, no new third-party dependency. Argument parsing uses the standard
library `flag` package.

## Technical Context

**Language/Version**: Go 1.25 (same module as the backend, `services/todo`)

**Primary Dependencies**: Connect-Go (`connectrpc.com/connect` v1.19.2) and the
already-generated `gen/task/v1` (`taskv1`) and `gen/task/v1/taskv1connect`
packages; `google.golang.org/protobuf` (`timestamppb`); standard library
`net/http`, `flag`, `time`. No new third-party dependency is added.

**Storage**: N/A — the tool is a stateless client; the backend owns all storage.

**Testing**: `go test` — table-driven unit tests for the pure helpers (tree
rendering, `--due` parsing/formatting, error mapping); command-level tests that
run each subcommand against an in-memory fake `TaskService` served by
`httptest.NewServer`. No PostgreSQL and no running server are required for tests.

**Target Platform**: Single statically-linked CLI binary (`todo`) run from a
Linux/macOS terminal.

**Project Type**: CLI tool — one new binary plus a supporting internal package,
added to the existing single Go service module.

**Performance Goals**: Each command completes and returns within 2 seconds under
normal single-user load against a local backend (spec SC-006).

**Constraints**: No new Go module and no new third-party dependency; the tool
reuses the generated Connect client. Connect's unary protocol runs over plain
HTTP/1.1, so a stock `http.Client` reaches the backend's h2c server with no
HTTP/2 client configuration.

**Scale/Scope**: Single trusted user; one binary, one `task` command group, four
subcommands, ~hundreds of tasks expected in `list` output.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I — Simplicity / YAGNI

| Check | Status |
|---|---|
| Simplest solution that satisfies requirements | PASS — standard-library `flag` parsing; a stock `http.Client`; the CLI lives in the existing module so it imports the generated client directly. No CLI framework, no new module, no h2c client plumbing. |
| No speculative abstractions | PASS — three source files (dispatch, subcommands, formatting helpers); no command-registry, no plugin layer, no config file beyond one env var. |
| Reuse over invention | PASS — the backend's generated `taskv1connect` client and `taskv1` message types are reused as-is; no hand-rolled HTTP/JSON. |
| Complexity justified in tracking table | PASS — no violations; Complexity Tracking is empty. |

### Principle II — API-First Design

| Check | Status |
|---|---|
| Data contract defined before implementation | PASS — the backend contract `task.proto` already exists and is committed under `specs/001-task-crud-api/contracts/`; the CLI consumes it unchanged. |
| Contract committed under `specs/002-todo-cli/contracts/` | PASS — the tool's user-facing command contract is authored as `contracts/cli.md` in Phase 1, before any CLI code. |
| Implementation conforms to the contract | PASS — request/response types are the buf-generated types from `task.proto`; the CLI cannot diverge from the wire contract. |

**Result**: PASS (initial). Re-evaluated after Phase 1 design — see "Post-Design Constitution Re-Check" below.

## Project Structure

### Documentation (this feature)

```text
specs/002-todo-cli/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── cli.md           # Phase 1 output — CLI command contract
├── checklists/
│   └── requirements.md  # Spec quality checklist (from /speckit-specify)
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/todo/
├── cmd/
│   ├── server/
│   │   └── main.go          # unchanged — backend entrypoint
│   └── todo/
│       └── main.go          # NEW — CLI entrypoint: os.Exit(cli.Run(os.Args[1:]))
└── internal/
    └── cli/
        ├── cli.go           # NEW — root dispatch, server-address resolution, exit code
        ├── task.go          # NEW — list/add/rm/mod: flag parsing, RPC calls, output
        ├── render.go        # NEW — tree rendering, --due parse/format, error mapping
        ├── task_test.go     # NEW — subcommand tests against an in-memory fake TaskService
        └── render_test.go   # NEW — unit tests for tree rendering and --due handling
```

**Structure Decision**: The CLI is added to the existing `services/todo` module
as a second `cmd/` binary (`cmd/todo`), mirroring how Go projects host multiple
binaries in one module. All command logic lives in `internal/cli`; `cmd/todo/main.go`
is a thin entrypoint. This lets the CLI import the buf-generated `taskv1connect`
client by package path with no `go.work` file and no `replace` directive — the
simplest arrangement that satisfies the spec (Principle I). No new top-level
directory is created.

## Complexity Tracking

> No Constitution Check violations. No entries required.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _(none)_  | —          | —                                   |

## Post-Design Constitution Re-Check

Re-evaluated after Phase 1 artifacts (`research.md`, `data-model.md`,
`contracts/cli.md`, `quickstart.md`) were produced:

- **Principle I** — The design adds one entrypoint file and three small source
  files in a single new package, with two test files. The only non-trivial logic
  is recursive tree rendering and two-format date parsing; both are the minimum
  needed for FR-002 and the clarified `--due` behavior. No CLI framework, no new
  module, no new dependency. **PASS.**
- **Principle II** — The backend wire contract (`task.proto`) is unchanged and
  already committed; `contracts/cli.md` defines the user-facing command surface
  before implementation. **PASS.**

No violations introduced by the design. Ready for `/speckit-tasks`.

## Phase 0 — Research

See [research.md](./research.md). All technical decisions are resolved; no
`NEEDS CLARIFICATION` items remain (the spec's `/speckit-clarify` session on
2026-05-18 settled tree rendering, exit codes, `--due` formats, and success
output).

## Phase 1 — Design & Contracts

- [data-model.md](./data-model.md) — the `Task` entity as the CLI consumes it,
  the in-memory tree it builds for `list`, and the per-command input models.
- [contracts/cli.md](./contracts/cli.md) — the CLI command contract: every
  command, its arguments and flags, exit codes, and stdout/stderr behavior.
- [quickstart.md](./quickstart.md) — how to build the `todo` binary, point it at
  a backend, and exercise each command.
- Agent context: the `<!-- SPECKIT START -->` block in `CLAUDE.md` is updated to
  point to this plan.

## Phase 2 — Next Step

Run `/speckit-tasks` to generate `tasks.md`, the dependency-ordered
implementation task list.
