# Phase 0 Research: Todo CLI Tool

**Feature**: 002-todo-cli | **Date**: 2026-05-18

All decisions below are resolved. The `/speckit-clarify` session of 2026-05-18
already settled the four spec-level ambiguities (tree rendering style, exit-code
granularity, accepted `--due` formats, success output), so no
`NEEDS CLARIFICATION` markers remain. This document records the implementation
decisions the plan depends on.

## Decision 1 — Where the CLI lives

**Decision**: Add the CLI to the existing `services/todo` Go module as a second
binary, `cmd/todo`, with command logic in a new `internal/cli` package.

**Rationale**: The CLI must call the backend through the buf-generated Connect
client (`gen/task/v1/taskv1connect`). Hosting the CLI in the same module lets it
import that package by path with zero extra wiring. Go modules routinely hold
multiple `cmd/` binaries; `cmd/server` and `cmd/todo` sit side by side.

**Alternatives considered**:
- *Separate module* (`cli/` or `services/cli` with its own `go.mod`) — rejected:
  it would need a `go.work` file or a `replace` directive to see the generated
  client locally. More moving parts for no benefit (Principle I, YAGNI).
- *Publishing the generated package* — rejected: it is internal to this repo.

## Decision 2 — Argument parsing

**Decision**: Use the standard library `flag` package. The root layer matches
the `task` group and dispatches on the subcommand word; each subcommand owns a
`flag.FlagSet` that parses its `--parent`/`--due` flags and exposes positional
arguments via `FlagSet.Args()`.

**Rationale**: The surface is tiny — one command group, four subcommands, two
optional flags. `flag` covers it, prints per-subcommand flag help, and adds no
dependency. The constitution forbids speculative abstraction; a CLI framework
would be exactly that here.

**Alternatives considered**:
- *cobra*, *urfave/cli*, *kong* — rejected: each is a new third-party
  dependency carried for a four-command tool. `flag` plus a hand-written usage
  line per subcommand satisfies FR-017 (usage help) without them.

## Decision 3 — Transport to the backend

**Decision**: Build the client with `taskv1connect.NewTaskServiceClient` over a
stock `&http.Client{}`. No HTTP/2 or h2c configuration on the client side.

**Rationale**: Connect's own unary protocol is plain HTTP/1.1 POST requests. The
backend is served through `h2c.NewHandler`, which still answers ordinary
HTTP/1.1 requests, so a default `http.Client` reaches it. The CLI makes only
unary calls (no streaming), so HTTP/1.1 is sufficient.

**Alternatives considered**:
- *h2c client* (`http2.Transport` with `AllowHTTP` and a custom `DialTLSContext`)
  — rejected: needed only for HTTP/2-specific features the CLI does not use; it
  is avoidable complexity (Principle I).
- *gRPC client option* (`connect.WithGRPC()`) — rejected: gRPC requires HTTP/2;
  the default Connect protocol does not.

## Decision 4 — Error mapping and exit codes

**Decision**: Inspect RPC errors with `connect.CodeOf(err)`:
- `connect.CodeNotFound` → "task not found" style message naming the id.
- `connect.CodeInvalidArgument` → surface the backend's `(*connect.Error).Message()`
  text verbatim (it already distinguishes "parent task not found",
  "parent_id would create a cycle", and name-validation failures).
- A transport/dial failure (connection refused, DNS, timeout — not a
  `*connect.Error`) → a clear "cannot reach backend at <addr>" message.
- Malformed input caught before any RPC (non-integer id, unparseable `--due`,
  missing name) → a usage error printed locally.

Every failure path writes the message to stderr and returns process exit
status `1`; success returns `0` (clarified: a single non-zero code).

**Rationale**: FR-014 asks for "clear, distinguishable" messages. The backend
maps unknown-parent, cycle, and name errors all to `invalid_argument`, so the
distinguishing information is the error *message*, not the code. Passing that
message through verbatim is the faithful and simplest behavior.

**Alternatives considered**:
- *Distinct exit codes per error class* — rejected: the clarify session chose a
  single code `1`.
- *Re-deriving cycle vs. unknown-parent on the client* — rejected: the backend
  already produced a precise message; re-classifying it would duplicate logic.

## Decision 5 — `mod` fetch-then-update

**Decision**: `mod` first calls `GetTask` for the target id, then constructs an
`UpdateTaskRequest` carrying the *complete* desired state: the new name (always
supplied), the existing `description`, and `due`/`parent_id` taken from the
supplied flag when present or from the fetched task when absent. It then calls
`UpdateTask`.

**Rationale**: The backend's `UpdateTask` is full-replace — an omitted optional
field is cleared. A naive `mod` that sent only the changed fields would wipe the
task's description (which this tool never exposes) and any unflagged field. The
spec's Assumptions section mandates this fetch-then-update approach; FR-012
requires unsupplied fields to stay unchanged.

**Consequence**: A `mod` is two RPCs. With the backend's ~1 s per-RPC budget the
pair stays within the 2 s ceiling of SC-006. If `GetTask` returns
`not_found`, `mod` reports it and stops without calling `UpdateTask`.

**Alternatives considered**:
- *Field-mask / partial update* — rejected: the backend contract has no field
  mask; changing it is out of scope for this feature.

## Decision 6 — `--due` parsing and display

**Decision**: Parse `--due` by trying `time.Parse(time.RFC3339, v)` first; on
failure, try `time.Parse("2006-01-02", v)` and, if it succeeds, treat it as
`00:00:00Z` on that date. If both fail, reject with a usage error naming both
accepted forms. Convert the result to `*timestamppb.Timestamp`. Display due
dates in `list` as `t.UTC().Format(time.RFC3339)`.

**Rationale**: Matches the clarified behavior — full RFC 3339 date-times and
bare calendar dates both accepted, bare dates pinned to midnight UTC.

**Alternatives considered**:
- *Accepting relative expressions* ("tomorrow", "+3d") — rejected: out of scope;
  the spec specifies absolute timestamps only.

## Decision 7 — Backend address configuration

**Decision**: Resolve the backend base URL from the `TODO_ADDR` environment
variable, defaulting to `http://localhost:8080` when unset.

**Rationale**: The spec calls configuration "a minor concern, not detailed
further" and expects a default to the local dev server. A single environment
variable is the least machinery that satisfies that.

**Alternatives considered**:
- *A `--addr` global flag* — rejected for now: a flag would have to thread
  through every subcommand's `FlagSet`; the env var covers the stated need with
  less code. Can be added later if required.

## Decision 8 — Test strategy

**Decision**: Two layers, both under `go test ./...` with no external services:
- *Unit tests* (`render_test.go`) — table-driven tests for tree rendering,
  `--due` parsing/formatting, and error-message mapping (pure functions).
- *Command tests* (`task_test.go`) — an in-memory fake `TaskService` (a struct
  implementing `taskv1connect.TaskServiceHandler`, backed by a map) is mounted
  with `taskv1connect.NewTaskServiceHandler` and served by `httptest.NewServer`.
  Each subcommand is run with its base URL pointed at that server; assertions
  cover stdout, stderr, and the returned exit code.

**Rationale**: The fake exercises the real client, real request building, and
real error mapping over a real HTTP loopback, without PostgreSQL or the backend
binary. Connection-failure behavior is tested by pointing at a closed port.

**Alternatives considered**:
- *Integration tests against the real server + PostgreSQL* — rejected as the
  primary strategy: heavy, slow, and it tests the backend more than the CLI. The
  `quickstart.md` documents a manual end-to-end check against the real stack.
