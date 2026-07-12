# Implementation Plan: Task Filters

**Branch**: `062-task-filters` | **Date**: 2026-07-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/062-task-filters/spec.md`

## Summary

Add an expressive, SQL-like filter language for tasks, evaluated **server-side** so the web app can adopt it later, surfaced in the **TUI Tasks tab** behind a `/` key that progressively re-filters as the user types. A new `FilterTasks` RPC on the existing `task.v1.TaskService` parses and evaluates the expression against the caller's tasks and returns the set of matching task IDs; the TUI renders those matches plus their ancestor chains from the task tree it already holds. No database schema changes — evaluation is in-memory over the existing task rows. The "show all" toggle is passed with each request so default-visibility semantics (FR-008) live entirely on the server.

## Technical Context

**Language/Version**: Go 1.26.0 (both the CLI/TUI root module and the `services/twig` server module)

**Primary Dependencies**: Bubble Tea v2 + `charm.land/bubbles/v2/textinput` (TUI), ConnectRPC + protobuf (`api/` module, regenerated via `make proto`), pgx/v5 + sqlc (server, unchanged queries). No new third-party dependencies: the filter language gets a small hand-written lexer + recursive-descent parser.

**Storage**: PostgreSQL — **no schema changes**. The evaluator reads the user's tasks via the existing `ListTasks` sqlc query; all matching (text, status, dates, transitivity) happens in memory in the handler layer.

**Testing**: `go test ./...` in each module. Parser/evaluator gets table-driven unit tests in `services/twig/internal/filter`; handler tests follow the existing `export_test.go` shim convention; TUI behavior tests follow existing `internal/tui/*_test.go` patterns. No integration-test infrastructure (per repo convention, tests must not require a running database).

**Target Platform**: Linux server (HTTP/2, port 8080) + terminal TUI client

**Project Type**: Client/server — Go TUI client + ConnectRPC service (existing structure, three Go modules)

**Performance Goals**: Filter round-trip (keystroke → updated list) under 300 ms for task lists up to 1,000 tasks (SC-002). In-memory evaluation of ≤1,000 tasks is microseconds; the budget is dominated by the HTTP round-trip, so no debounce is required — a per-keystroke request with a generation counter (discard stale responses) suffices.

**Constraints**: Filter semantics must be 100% server-side (FR-011) so a future web client gets identical results; the TUI never applies its own status-eligibility logic while a filter is active. Invalid expressions must be non-fatal per keystroke (FR-009). Existing `ListTasks` contract must remain untouched (web app and CLI depend on it).

**Scale/Scope**: Single-user task lists in the hundreds to low thousands; one new RPC; one new server package (`internal/filter`); TUI changes confined to the Tasks tab.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | No new dependencies; hand-written ~small parser; in-memory evaluation instead of SQL generation; no schema changes; no debounce machinery (generation counter only). No speculative web-UI work. |
| II. API-First Design | ✅ | `contracts/filter-tasks.md` (RPC contract) and `contracts/filter-grammar.md` (language semantics) are committed with this plan, before implementation begins. Proto change lands in `api/proto/task/v1/task.proto` per contract. |
| III. UI/UX Consistency | ✅ | Filter input reuses `charm.land/bubbles/v2/textinput` (same widget as plan-entry inputs and task edit form), styled from `internal/tui/theme.go`; `/` follows universal terminal search convention; new binding added to `keymap.go`/`help.go` alongside existing bindings. |
| IV. Playful User Messages | ✅ | Two new user-facing strings — the no-matches empty state and the invalid-expression indicator — get warm, playful copy (drafted in quickstart.md, reviewed at implementation). |

**Post-design re-check (after Phase 1)**: still ✅ on all four — the design added no abstractions beyond the single `internal/filter` package (parser + evaluator, which cannot live in the handler without bloating it), and both contracts exist.

## Project Structure

### Documentation (this feature)

```text
specs/062-task-filters/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── filter-tasks.md      # FilterTasks RPC contract (proto source of truth)
│   └── filter-grammar.md    # Filter language grammar + evaluation semantics
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
api/
├── proto/task/v1/task.proto        # + FilterTasks RPC, Filter{Tasks}Request/Response messages
└── gen/                            # regenerated via `make proto` (do not edit)

services/twig/
├── internal/filter/                # NEW package: filter language
│   ├── lexer.go                    # tokenizer (idents, operators, strings, dates, ints, AND, ^)
│   ├── parser.go                   # recursive-descent parser → expression AST
│   ├── eval.go                     # evaluator: AST × task set × (showAll, today) → matched IDs
│   └── *_test.go                   # table-driven grammar + semantics tests
└── internal/handler/
    ├── task.go                     # + FilterTasks handler (loads tasks, delegates to filter pkg)
    └── task_test.go                # + handler tests

internal/tui/                       # root module (TUI client)
├── model.go                        # + filter state (input, applied expression, matched IDs, generation)
├── update.go                       # + '/' binding, filter-input key handling, response handling
├── tree.go                         # buildVisible honors matched-ID set (matches + ancestors, expansion ignored)
├── view.go                         # + filter bar rendering (active/invalid states)
├── client.go                       # + FilterTasks call helper
├── keymap.go / help.go             # + '/' and filter-mode bindings in help
└── *_test.go                       # TUI behavior tests
```

**Structure Decision**: Extends the existing three-module layout. The only new directory is `services/twig/internal/filter` — the language implementation must be a separate package so the handler stays thin and the parser/evaluator are unit-testable without ConnectRPC scaffolding. All other work modifies existing files in place.

## Complexity Tracking

No Constitution Check violations — table intentionally empty.
