# Implementation Plan: Pomodoro Tracking

**Branch**: `005-pomodoro-tracking` | **Date**: 2026-05-20 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/005-pomodoro-tracking/spec.md`

## Summary

Extend the existing `services/todo` backend and `todo` CLI with pomodoro tracking. The `tasks` table gains an `estimate SMALLINT NOT NULL DEFAULT 0` column (0–10, CHECK-enforced). A new `pomodoros` table records each focused-work session with `(id, user_id, task_id, start_at, end_at, complete)`; a partial unique index on `user_id WHERE end_at IS NULL` enforces "at most one active pomodoro per user". `task.proto` is extended with the `Task.estimate` field, a new `Pomodoro` message, and five new RPCs: `SetEstimate`, `StartPomodoro`, `CancelPomodoro`, `CompletePomodoro`, `GetActivePomodoro`. The existing `GetTask` is extended (in its response shape) to return the count and full list of pomodoros for the task; this avoids a separate "history" RPC and keeps the CLI's countdown UI single-call. Pomodoros cascade-delete with their task via FK ON DELETE CASCADE. The CLI gains a `task pom` subcommand group (`estimate`, `start`, `resume`, `cancel`, `status`); `start` and `resume` render a live terminal countdown driven by the server's stored `start_at` (no local stopwatch), poll `GetActivePomodoro` once per second so external cancellation/completion is detected, and accept keystrokes `c` (cancel) and `q` (quit-without-stopping). On timer expiry the CLI calls `CompletePomodoro`; if `--exec <cmd>` was given, it runs the command synchronously, streams stdout/stderr, and surfaces a non-zero exit as a warning (CLI itself exits 0). `cancel` and `resume` take no task_id and always operate on the user's single active pomodoro.

## Technical Context

**Language/Version**: Go 1.25 (existing `services/todo` module — no new module)

**Primary Dependencies**: Existing only — Connect-Go (`connectrpc.com/connect` v1.19.2), `pgx/v5`, `golang-migrate/v4`, `sqlc` v1.31.1, generated `taskv1`/`taskv1connect`. The countdown UI uses the Go standard library only (`os.Stdin` in raw mode via `golang.org/x/term` for keystroke reads — `x/term` is already permitted under the standard-library-adjacent rule established for features 003/004; if not yet vendored it is added in tasks). No TUI framework.

**Storage**: PostgreSQL via `pgx/v5`; schema managed by `golang-migrate` file migrations. New migration `000005_pomodoros`:
  - `ALTER TABLE tasks ADD COLUMN estimate SMALLINT NOT NULL DEFAULT 0 CHECK (estimate BETWEEN 0 AND 10);`
  - `CREATE TABLE pomodoros (id BIGSERIAL PK, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE, start_at TIMESTAMPTZ NOT NULL, end_at TIMESTAMPTZ, complete BOOLEAN NOT NULL DEFAULT FALSE, CHECK (end_at IS NULL OR end_at >= start_at), CHECK (NOT complete OR end_at IS NOT NULL));`
  - `CREATE UNIQUE INDEX pomodoros_one_active_per_user ON pomodoros (user_id) WHERE end_at IS NULL;`
  - `CREATE INDEX pomodoros_task_id_idx ON pomodoros (task_id);`

  `db/queries/task.sql` gains `SetTaskEstimate`. New `db/queries/pomodoro.sql` holds `StartPomodoro`, `CancelActivePomodoro`, `CompleteActivePomodoro`, `GetActivePomodoro`, `ListPomodorosForTask`, `CountCompletedPomodorosForTask`. The `RETURNING *` clauses on existing `CreateTask`/`UpdateTask`/`GetTask`/`ListTasks`/`CompleteTask` automatically include `estimate` after sqlc regenerates.

**Testing**: `go test` — table-driven handler tests against a test database covering:
  - `SetEstimate`: happy path (0–10), rejection at -1 and 11 (CodeInvalidArgument with "break it down" message), unknown task (NotFound), overwrites prior value.
  - `StartPomodoro`: success path, AlreadyExists when an active pomodoro already exists for the user, NotFound for unknown/foreign task.
  - `CancelPomodoro` / `CompletePomodoro`: success, FailedPrecondition when none active.
  - `GetActivePomodoro`: returns the row when active, returns a "none active" response (not an error) when none active.
  - Pomodoros cascade-delete on `DeleteTask` (assertion on `pomodoros` rowcount).
  - The single-active-pomodoro invariant under concurrent `StartPomodoro` calls (race test: two goroutines, exactly one succeeds).
  - `GetTask` returns the embedded count and list of pomodoros.

  CLI tests for: `runEstimate` (forwarding + validation), `runStart` flow with a stubbed clock and a fake terminal reader (timer expiry triggers `CompletePomodoro`; `c` triggers `CancelPomodoro`; `q` exits without an API call; external cancellation detected via polling exits cleanly), `runResume` (immediate-completion path when `start_at` is more than 25 min ago), `runCancel`, `runStatus`, and the `--exec` command runner (success path, non-zero exit produces a warning and CLI exit 0).

**Target Platform**: Same single Linux server binary (`cmd/server`) plus the `todo` CLI binary (`cmd/todo`), already established by features 001–004. The countdown UI runs in an interactive terminal; if `stdin` is not a TTY (e.g., piped), `start`/`resume` print a one-line warning and run in a non-interactive countdown that still polls and completes/cancels on the server but ignores keystrokes.

**Project Type**: Web service (Connect-Go backend) plus its companion CLI — both already exist in one Go module; this feature extends both.

**Performance Goals**: SC-002 — every API call returns in under 1 second locally. The countdown UI redraws once per second and polls `GetActivePomodoro` once per second on the same tick (one request/sec, comfortably within the existing scale envelope of 1–2 users/instance). External cancellation is therefore detected within 1–2 seconds, well under SC-005's 5-second bound.

**Constraints**: The single-active-pomodoro invariant (SC-003) is enforced at the database layer by `pomodoros_one_active_per_user` partial unique index; the handler converts the resulting unique-violation into Connect's `AlreadyExists`. The 25-minute pomodoro length is a single named constant (`pomodoro.Length = 25 * time.Minute`) in a new `internal/pomodoro` package; it is consumed by the handler's `CompletePomodoro` (to compute `end_at = start_at + Length`) and by the CLI's countdown renderer (to compute remaining time from the server's `start_at`). Completion timestamps are UTC. Idempotency: `CancelPomodoro` / `CompletePomodoro` on a no-longer-active row return `FailedPrecondition`, not silent success — the CLI translates this to a clear message for the user. CLI exit codes follow the established convention: `0` on success, `1` on user-facing error, `1` on `start` aborted by user-declined-cancel (FR-036), `0` when `q` quits the UI without altering state.

**Scale/Scope**: Continues feature 004's posture — one to two users per instance, modest history per user (tens of pomodoros per task at most). All queries are either keyed on `(user_id)` or `(task_id)`, both indexed.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I — Simplicity / YAGNI

| Check | Status |
|---|---|
| Simplest solution that satisfies requirements | PASS — one new column, one new table, five new RPCs (each a thin wrapper around one query), one CLI subcommand group. No background timer, no scheduler, no event bus; the API is stateless w.r.t. time and the CLI is the only timing actor. Polling-based external-cancel detection (vs. websockets/SSE) avoids any new transport. |
| No speculative abstractions | PASS — no "session" / "focus block" generalization, no per-task settings table, no policy registry, no notification system beyond the user-supplied `--exec` hook. The `pomodoro.Length` constant exists because two callers (handler completion, CLI countdown) genuinely share it. |
| Reuse over invention | PASS — uses existing `sqlc`, `pgx`, Connect error mapping, auth interceptor, and CLI client wiring. The 25-minute constant lives in one package and is imported by both consumers; no duplicated literal. |
| Complexity justified in tracking table | PASS — no violations; Complexity Tracking is empty. The partial unique index is the standard Postgres pattern for "at most one row matching predicate" and is materially simpler than a SELECT-then-INSERT guard. |

### Principle II — API-First Design

| Check | Status |
|---|---|
| Data contract defined before implementation | PASS — `contracts/pomodoro.md` is authored in Phase 1 and specifies the new `Task.estimate` field, the new `Pomodoro` message, the five new RPCs (request/response/error), and the `GetTask` response extension (added count + list of pomodoros). |
| Contract committed under `specs/005-pomodoro-tracking/contracts/` | PASS — `contracts/pomodoro.md` is committed with the feature branch. The proto edit lands in the implementation phase, matching the contract verbatim. |
| Implementation conforms to the contract | PASS — handler tests assert the new RPCs' request/response shapes; the proto delta is recorded in the contract file and the existing `proto/task/v1/task.proto` is amended to match it exactly. The countdown UI's display values are derived from the server response, never from a parallel local model. |

**Result**: PASS (initial). Re-evaluated after Phase 1 design — see "Post-Design Constitution Re-Check" below.

## Project Structure

### Documentation (this feature)

```text
specs/005-pomodoro-tracking/
├── plan.md                       # This file
├── research.md                   # Phase 0 output
├── data-model.md                 # Phase 1 output
├── quickstart.md                 # Phase 1 output
├── contracts/
│   └── pomodoro.md               # Phase 1 output — proto delta, RPC contracts, GetTask extension
├── checklists/
│   └── requirements.md           # Spec quality checklist (from /speckit-specify)
└── tasks.md                      # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/todo/
├── proto/task/v1/
│   └── task.proto                # MODIFIED — Task.estimate; Pomodoro message;
│                                 #   SetEstimate, StartPomodoro, CancelPomodoro,
│                                 #   CompletePomodoro, GetActivePomodoro RPCs;
│                                 #   GetTaskResponse gains completed_pomodoro_count
│                                 #   and repeated Pomodoro pomodoros
├── gen/task/v1/
│   ├── task.pb.go                # REGENERATED by buf
│   └── taskv1connect/...         # REGENERATED by buf
├── db/
│   ├── migrations/
│   │   ├── 000005_pomodoros.up.sql   # NEW — tasks.estimate column + pomodoros table
│   │   └── 000005_pomodoros.down.sql # NEW — reverse
│   └── queries/
│       ├── task.sql              # MODIFIED — new SetTaskEstimate query;
│       │                         #   existing RETURNING * picks up estimate
│       │                         #   via sqlc regeneration
│       └── pomodoro.sql          # NEW — StartPomodoro, CancelActivePomodoro,
│                                 #   CompleteActivePomodoro, GetActivePomodoro,
│                                 #   ListPomodorosForTask,
│                                 #   CountCompletedPomodorosForTask
├── internal/
│   ├── pomodoro/
│   │   └── pomodoro.go           # NEW — exports Length = 25 * time.Minute and
│   │                             #   Remaining(start time.Time, now time.Time) time.Duration
│   ├── db/
│   │   ├── models.go             # REGENERATED — Task gains Estimate; new Pomodoro model
│   │   ├── task.sql.go           # REGENERATED
│   │   └── pomodoro.sql.go       # REGENERATED — sqlc bindings for new queries
│   └── handler/
│       ├── task.go               # MODIFIED — dbTaskToProto sets Estimate;
│       │                         #   GetTask response includes pomodoro count + list;
│       │                         #   new SetEstimate method with 0–10 validation
│       ├── pomodoro.go           # NEW — StartPomodoro, CancelPomodoro,
│       │                         #   CompletePomodoro, GetActivePomodoro; maps
│       │                         #   unique-violation → AlreadyExists; computes
│       │                         #   end_at = start_at + pomodoro.Length on complete
│       ├── pomodoro_test.go      # NEW — handler tests for the five new RPCs and
│       │                         #   the cascade-delete and concurrency invariants
│       └── task_test.go          # MODIFIED — SetEstimate tests; GetTask test
│                                 #   asserting embedded pomodoro count/list
└── internal/cli/
    ├── cli.go                    # MODIFIED — wire `task pom` subgroup;
    │                             #   updated usage text
    ├── pom.go                    # NEW — runPom dispatcher; runEstimate,
    │                             #   runStart, runResume, runCancel, runStatus;
    │                             #   --exec runner
    ├── pom_countdown.go          # NEW — terminal countdown UI: render loop
    │                             #   (1Hz), keystroke reader, polling external-
    │                             #   cancel detector; abstracted behind a small
    │                             #   interface so tests can drive a fake clock
    │                             #   and fake input
    ├── pom_test.go               # NEW — runEstimate/runStart/runResume/runCancel/
    │                             #   runStatus tests + --exec runner test
    └── pom_countdown_test.go     # NEW — countdown loop tests with fake clock,
                                  #   fake input, and fake API client
```

**Structure Decision**: The feature extends the existing single `services/todo` module in place — no new module. Two new files in `internal/handler/` (`pomodoro.go`, `pomodoro_test.go`) and a small new `internal/pomodoro` package for the shared 25-minute constant are the only structural additions on the server side; everything else is additive within existing files. On the CLI side a new `pom.go` plus a `pom_countdown.go` keep the interactive timer code isolated from the rest of the CLI so the existing `task.go` stays focused on plain CRUD subcommands. Pomodoros are recorded in their own table (not as a JSON column on `tasks`) because they have their own lifecycle and history queries; that is the simpler model overall, even though it adds a table.

## Complexity Tracking

> No Constitution Check violations. No entries required.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| _(none)_  | —          | —                                    |

## Post-Design Constitution Re-Check

Re-evaluated after Phase 1 artifacts (`research.md`, `data-model.md`, `contracts/pomodoro.md`, `quickstart.md`) were produced:

- **Principle I** — Phase 1 confirmed: the data model is one new column and one new table with three indexes (PK, partial-unique, task_id). The handler additions are thin per-RPC functions; the CLI countdown is roughly 150 lines guarded behind a small interface for testability. No new dependency beyond `golang.org/x/term` for raw terminal input, which is the lightest viable choice (no TUI framework). The shared `pomodoro.Length` constant prevents a duplicated 25-minute literal between server and CLI. **PASS.**
- **Principle II** — `contracts/pomodoro.md` records the exact proto delta (one new field on `Task`, one new `Pomodoro` message, five new RPCs, two extra fields on `GetTaskResponse`) and the request/response/error contract for each RPC, including the error codes the CLI relies on. The CLI's countdown derives all displayed values (task name, estimate, completed-count, remaining time) from the `GetTask` + `GetActivePomodoro` responses — no parallel local model. **PASS.**

No violations introduced by the design. Ready for `/speckit-tasks`.

## Phase 0 — Research

See [research.md](./research.md). All technical decisions are resolved; no `NEEDS CLARIFICATION` items remain. Research records the notable implementation-level choices:
1. **Single-active-pomodoro enforcement** — partial unique index vs. handler-level lock vs. SELECT-then-INSERT.
2. **Where the 25-minute clock authority lives** — server-derived `end_at` at completion vs. client-asserted `end_at` in the request.
3. **External-cancel detection mechanism** — 1Hz polling on `GetActivePomodoro` vs. SSE/websocket push.
4. **Raw terminal input** — `golang.org/x/term` vs. a TUI library (bubbletea/tview) vs. line-buffered reads.
5. **Embedding pomodoro count/list in `GetTask`** vs. introducing a dedicated `ListPomodoros` RPC.

## Phase 1 — Design & Contracts

- [data-model.md](./data-model.md) — the `tasks.estimate` column (CHECK 0–10), the `pomodoros` table, the partial unique index that enforces the single-active invariant, and the lifecycle of a pomodoro (`Active → Completed` or `Active → Canceled`).
- [contracts/pomodoro.md](./contracts/pomodoro.md) — the proto delta (`Task.estimate`, `Pomodoro` message, five new RPCs, `GetTaskResponse` extension) and each RPC's request/response/error contract.
- [quickstart.md](./quickstart.md) — apply migration, set an estimate, start a pomodoro from the CLI, observe the countdown, cancel from another terminal and watch the countdown exit, and run a session with `--exec`.
- Agent context: the `<!-- SPECKIT START -->` block in `CLAUDE.md` is updated to point to this plan.

## Phase 2 — Next Step

Run `/speckit-tasks` to generate `tasks.md`, the dependency-ordered implementation task list.
