# Implementation Plan: Goals

**Branch**: `049-goals` | **Date**: 2026-06-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/049-goals/spec.md`

## Summary

Add goals — long-term aims that aren't actionable today — as a first-class
object so they stop cluttering the task list. Goals have name/description/due,
four states (incubating, committed, completed, archived; any transition
allowed), per-state-group ranking, and optional task association with subtree
inheritance. Technical approach: a new `goal/v1` proto package with its own
`GoalService` (CRUD + SetGoalState + ReorderGoal, mirroring existing service
shapes), one migration adding a `goals` table and a nullable
`tasks.goal_id` FK (`ON DELETE SET NULL`), and a dedicated
`TaskService.SetTaskGoal` RPC so the full-replace `UpdateTask` path — and the
untouched web app — can never clobber associations. TUI gains a Goals tab,
first in the tab bar (startup remains the Tasks tab), with the established
two-pane layout; CLI gains `twig goal` mirroring `twig task` conventions.
Web app explicitly unchanged.

## Technical Context

**Language/Version**: Go 1.26 (three existing modules: root CLI/TUI, `api/`, `services/twig/`)

**Primary Dependencies**: Bubble Tea / charm.land bubbles v2 (TUI), ConnectRPC (`connectrpc.com/connect`), buf (proto gen), sqlc + pgx/v5 (server queries), golang-migrate (schema)

**Storage**: PostgreSQL — one new migration (`000010_goals`): `goals` table + `tasks.goal_id BIGINT NULL REFERENCES goals(id) ON DELETE SET NULL`

**Testing**: `go test ./...` per module; handler tests `DATABASE_URL`-gated; CLI/TUI tests via `export_test.go` shims and string assertions; pure goal logic (effective goal, grouping) table-driven in the root module

**Target Platform**: Linux/macOS terminals (CLI/TUI); Linux server container

**Project Type**: Multi-module client/server — CLI+TUI client, ConnectRPC server; web app explicitly out of scope (spec) and protected from regressions by D2

**Performance Goals**: Goal list render < 1 round-trip beyond the existing ListTasks fetch (SC-002: visible in < 30 s is trivially met); no N+1 — goal detail composes client-side from ListGoals + ListTasks

**Constraints**: Goals never nest (FR-004); no-nested-goal-associations invariant enforced server-side (FR-006a); completed/archived hidden by default everywhere (FR-011); playful copy on all new user-facing text (Principle IV)

**Scale/Scope**: Personal todo data — tens of goals, thousands of tasks per user; client-side composition over full lists is the app's established pattern

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One proto package, one migration, one new RPC on TaskService; no join table (nullable FK suffices); association stored once, inheritance computed client-side; state as TEXT+CHECK not a PG enum. No speculative abstractions — TUI reuses the existing edit form and task picker components. |
| II. API-First Design | ✅ | `contracts/goal-rpc.md` (goal/v1 proto + task.proto additions) and `contracts/goal-cli.md` (CLI/TUI surface) committed before implementation; tasks must reference them. |
| III. UI/UX Consistency | ✅ | Goals tab joins the existing tab cycle with shared `theme.go`, two-pane layout matching Plan/Tasks, and reused bindings (`n`, `e`, `ctrl+d`, `{`/`}`, `c`); `twig goal` copies `twig task` subcommand/flag/listing conventions. |
| IV. Playful User Messages | ✅ | Empty states, not-found errors, nesting-conflict error, and state-change notices specified with playful copy in `contracts/goal-cli.md`; tone review is a quality gate. |

**Post-design re-check (after Phase 1)**: all four principles still ✅ — the
design artifacts introduced no new violations; Complexity Tracking left empty.

## Project Structure

### Documentation (this feature)

```text
specs/049-goals/
├── plan.md              # This file
├── research.md          # Phase 0 output — decisions D1–D10
├── data-model.md        # Phase 1 output — Goal entity, states, validation
├── quickstart.md        # Phase 1 output — manual verification walkthrough
├── contracts/
│   ├── goal-rpc.md      # goal/v1 GoalService + task.v1 SetTaskGoal contract
│   └── goal-cli.md      # twig goal + TUI Goals tab surface contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
api/
├── proto/goal/v1/goal.proto            # NEW: GoalService + Goal messages
└── proto/task/v1/task.proto            # + Task.goal_id field + SetTaskGoal RPC
    api/gen/…                           # regenerated (make proto) — do not edit

services/twig/
├── db/migrations/000010_goals.{up,down}.sql   # NEW: goals table + tasks.goal_id
├── db/queries/goal.sql                 # NEW: goal CRUD, state, reorder queries
├── db/queries/task.sql                 # + SetTaskGoal / ancestry-check queries
│   internal/db/…                       # regenerated (sqlc generate) — do not edit
├── internal/handler/goal.go            # NEW: GoalService implementation
├── internal/handler/goal_test.go       # NEW: DATABASE_URL-gated tests
├── internal/handler/task.go            # + SetTaskGoal + re-parent nesting guard
├── internal/handler/task_test.go       # + association/nesting cases
└── cmd/server/main.go                  # register GoalService handler

internal/goal/                          # NEW package (root module): shared logic
├── goal.go                             # state grouping/order, effective-goal calc
└── goal_test.go                        # table-driven unit tests

internal/cli/
├── cli.go                              # dispatch "goal", root usage, help
├── goal.go                             # NEW: twig goal subcommands + rendering
├── goal_test.go
├── task.go                             # + --goal flag on add/mod, Goal: line
└── task_test.go

internal/tui/
├── model.go                            # + tabGoals (first), goalState struct
├── keymap.go                           # goal bindings + help entries
├── update.go                           # goal-tab key handling + RPC commands
├── view.go                             # tab bar gains Goals; routes goal view
├── goal_view.go                        # NEW: two-pane rendering, state groups
├── goal_view_test.go
├── edit.go                             # task edit form + Goal field
└── details.go                          # task detail pane + Goal: line
```

**Structure Decision**: Follows the established repo layout exactly — new proto
package beside `plan/v1`, migration + sqlc queries + handler in
`services/twig/`, all client logic in the root module. The single new package
`internal/goal` mirrors the `internal/report` precedent: pure domain logic
shared by the CLI and TUI surfaces, with rendering kept in `internal/cli` and
`internal/tui` respectively.

## Complexity Tracking

No Constitution Check violations — table intentionally left empty.
