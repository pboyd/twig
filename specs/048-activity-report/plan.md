# Implementation Plan: Activity Report

**Branch**: `048-activity-report` | **Date**: 2026-06-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/048-activity-report/spec.md`

## Summary

Add an on-demand activity report — "what did I get done?" — to the TUI (new Report
tab) and CLI (`twig report`). The report lists tasks completed within a chosen
period (presets or explicit range, local-timezone days, Monday-start weeks) plus
summary totals including pomodoros completed. Periods of ≤ 14 days group by day;
longer periods group by top-level accomplishment. Technical approach: compose the
report client-side from the existing `ListTasks` RPC (which already returns
completed tasks with `completed_at` and `parent_id`), adding a single new RPC —
`CountCompletedPomodoros(start, end)` — for the pomodoro total. Shared period and
grouping logic lives in a new root-module package `internal/report`. No schema
migration; no web app changes.

## Technical Context

**Language/Version**: Go 1.26 (three existing modules: root CLI/TUI, `api/`, `services/twig/`)

**Primary Dependencies**: Bubble Tea / charm.land bubbles v2 (TUI), ConnectRPC (`connectrpc.com/connect`), buf (proto gen), sqlc + pgx/v5 (server queries)

**Storage**: PostgreSQL — no schema changes; one new sqlc query over the existing `pomodoros` table

**Testing**: `go test ./...` per module; handler tests are `DATABASE_URL`-gated integration tests (skip when unset); CLI/TUI tests use `export_test.go` shims and string assertions

**Target Platform**: Linux/macOS terminals (CLI/TUI); Linux server container

**Project Type**: Multi-module client/server — CLI+TUI client, ConnectRPC server; web app explicitly out of scope

**Performance Goals**: Year-long report rendered in < 3 s (SC-003); report = one `ListTasks` + one count RPC, both single round-trips

**Constraints**: Local-timezone day boundaries with exact near-midnight/DST behavior (SC-005); plain-text output when piped (FR-009); playful copy on all new user-facing text (Principle IV)

**Scale/Scope**: Personal todo data — thousands of tasks/pomodoros per user; client-side filtering of the full task list is already the app's established pattern

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses `ListTasks` + existing tree-building; one new RPC, one new query, no migration, no new service/proto package. `internal/report` has two concrete consumers (CLI, TUI) — organization, not speculative abstraction. |
| II. API-First Design | ✅ | `contracts/report-rpc.md` (proto additions) and `contracts/report-cli.md` (CLI/TUI surface) committed before implementation; tasks must reference them. |
| III. UI/UX Consistency | ✅ | TUI Report tab joins the existing tab cycle, uses shared `theme.go` and existing bindings; CLI follows top-level-command conventions (`plan`, `pom`) and `render.go` TTY gating. |
| IV. Playful User Messages | ✅ | Empty state, validation errors, and headers specified with playful copy in `contracts/report-cli.md`; tone review is a quality gate. |

**Post-design re-check (after Phase 1)**: all four principles still ✅ — no new
violations introduced by the design artifacts; Complexity Tracking left empty.

## Project Structure

### Documentation (this feature)

```text
specs/048-activity-report/
├── plan.md              # This file
├── research.md          # Phase 0 output — decisions D1–D8
├── data-model.md        # Phase 1 output — Period, Entry, groups, totals
├── quickstart.md        # Phase 1 output — manual verification walkthrough
├── contracts/
│   ├── report-rpc.md    # CountCompletedPomodoros proto contract
│   └── report-cli.md    # twig report + TUI Report tab surface contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
api/
└── proto/task/v1/task.proto        # + CountCompletedPomodoros RPC + messages
    api/gen/…                       # regenerated (make proto) — do not edit

services/twig/
├── db/queries/pomodoro.sql         # + CountCompletedPomodorosInRange query
│   internal/db/…                   # regenerated (sqlc generate) — do not edit
└── internal/handler/pomodoro.go    # + CountCompletedPomodoros handler
    internal/handler/pomodoro_test.go  # + DATABASE_URL-gated handler test

internal/report/                    # NEW package (root module): shared logic
├── period.go                       # presets, parsing, local-day → UTC range math
├── period_test.go                  # ISO weeks, DST, midnight, validation cases
├── group.go                        # Entry building, day/accomplishment grouping
└── group_test.go

internal/cli/
├── cli.go                          # dispatch "report", root usage, help
├── report.go                       # NEW: arg parsing, fetch, render (TTY-gated)
└── report_test.go

internal/tui/
├── model.go                        # + tabReport, report state
├── keymap.go                       # report-tab help entries for ←/→ preset cycle
├── update.go                       # report-tab key handling + fetch commands
├── view.go                         # tab bar gains Report
├── report_view.go                  # NEW: themed rendering of both layouts
└── report_view_test.go
```

**Structure Decision**: Follows the established repo layout exactly — proto change
in `api/`, query+handler in `services/twig/`, all client logic in the root module.
The single new package `internal/report` mirrors the existing
`internal/pomodoro` precedent: domain logic shared by two surfaces, with rendering
kept in `internal/cli` and `internal/tui` respectively.

## Complexity Tracking

No Constitution Check violations — table intentionally left empty.
