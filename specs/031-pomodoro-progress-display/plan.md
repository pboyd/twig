# Implementation Plan: Pomodoro Progress Display

**Branch**: `031-pomodoro-progress-display` | **Date**: 2026-06-02 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/031-pomodoro-progress-display/spec.md`

## Summary

Render a task's pomodoro estimate and completion as a single row of tomato glyphs (🍅) in the TUI: dimmed for estimated-but-not-done, bold red for completed-within-estimate, yellow+bold for completed-beyond-estimate. The row appears in the task-tree details pane and the planning-tab linked-task details pane, drawn from a single shared renderer for consistency. Separately, the active (running) pomodoro glyph in the status bar becomes red and bold.

The one non-trivial piece is data availability: completed-pomodoro counts are **not** currently carried on the bare `Task` rows that `ListTasks` returns (the TUI tree is built from `ListTasks`). The chosen approach adds a per-task completed count to `ListTasks` so the in-memory tree carries everything the details panes need, and rendering stays synchronous (no per-selection RPC, no cache). See [research.md](./research.md) for the alternatives weighed.

## Technical Context

**Language/Version**: Go 1.x (module at `services/twig/`)

**Primary Dependencies**: Bubble Tea + Lipgloss (TUI), ConnectRPC over HTTP/2, pgx/v5 + sqlc (DB), protobuf/buf (contracts)

**Storage**: PostgreSQL (`pomodoros`, `tasks` tables) — read-only for this feature (counting completed pomodoros)

**Testing**: `go test ./...`; table-driven unit tests for the glyph renderer; existing TUI view/details test patterns (`details_test.go`, `plan_view_test.go`, `export_test.go` shims)

**Target Platform**: Terminal (TTY) — TUI launched by `twig` with no args; styling gated by `cli.WantStyled`

**Project Type**: Single Go module, two binaries (`cmd/server`, `cmd/twig`); TUI lives in `internal/tui/`

**Performance Goals**: No perceptible latency change; one extra aggregate query per `ListTasks` call. Rendering remains synchronous.

**Constraints**: Must degrade gracefully without color (glyph count still correct); must not break the existing web client (additive proto field only)

**Scale/Scope**: Personal task lists (tens–hundreds of tasks); estimate is server-bounded to 0..10, so glyph rows are short in normal use

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One shared renderer (`renderPomodoroRow`), one additive proto field, one aggregate SQL query. No caching layer, no new abstraction. Rejected the lazy-fetch+cache alternative precisely because it was more complex (see research.md). |
| II. API-First Design | ✅ | The `ListTasks` contract change (per-task completed count) is defined in `contracts/` and committed before implementation. Proto regenerated via `make proto`. |
| III. UI/UX Consistency | ✅ | Both tabs call the same `renderPomodoroRow`; new colors (`pomodoroDone` red, `pomodoroOver` yellow) are added to the shared `theme.go` palette — no ad-hoc styles. Active-glyph styling reuses the same red token. |
| IV. Playful User Messages | ✅ | Feature adds no prose; it renders glyphs. Empty state is an omission (no text). No dry messages introduced; existing tone untouched. |

## Project Structure

### Documentation (this feature)

```text
specs/031-pomodoro-progress-display/
├── plan.md              # This file
├── research.md          # Phase 0 output — data-availability decision
├── data-model.md        # Phase 1 output — entities & rendering model
├── quickstart.md        # Phase 1 output — how to exercise & verify
├── contracts/
│   ├── listtasks-completed-count.md   # ListTasks proto/contract change
│   └── pomodoro-row-rendering.md      # Visual/render contract (the rules)
└── checklists/
    └── requirements.md  # Spec quality checklist (already passing)
```

### Source Code (repository root)

```text
services/twig/
├── proto/task/v1/task.proto          # + Task.completed_pomodoro_count (additive)
├── gen/task/v1/                      # regenerated via `make proto` (do not hand-edit)
├── db/queries/pomodoro.sql           # + CountCompletedPomodorosByTask (per-user, grouped)
├── internal/db/                      # regenerated via `sqlc generate` (do not hand-edit)
├── internal/handler/task.go          # ListTasks: attach per-task completed counts
└── internal/tui/
    ├── theme.go                      # + pomodoroDone (red), pomodoroOver (yellow) tokens
    ├── pomodoro_row.go               # NEW: renderPomodoroRow(estimate, completed, styled)
    ├── pomodoro_row_test.go          # NEW: table-driven render tests
    ├── details.go                    # task-tree details: use glyph row
    ├── plan_view.go                  # planning-tab details: use glyph row for linked task
    ├── view.go                       # active-pomodoro glyph → red+bold; pass task to plan detail
    └── export_test.go                # expose new helper(s) to tests as needed
```

**Structure Decision**: Single Go module. Backend touch is minimal and additive (one proto field, one query, one handler edit). All rendering changes live in `internal/tui/`, centered on a new shared `renderPomodoroRow` helper consumed by both the tasks and planning details panes.

## Phase 0: Research

See [research.md](./research.md). Key decision: **extend `ListTasks` with a per-task completed-pomodoro count** rather than lazily fetching `GetTask` per selection. This keeps the details panes rendering synchronously from the in-memory tree, gives a single source of truth, and avoids cache-invalidation complexity. After a pomodoro completes, the tree is reloaded (`listTasksCmd`) so counts refresh.

## Phase 1: Design & Contracts

- **data-model.md** — Task pomodoro state (estimate 0..10, completed count ≥ 0), the derived glyph-row model (count = max(estimate, completed); segment styling rules), and the active-pomodoro state.
- **contracts/listtasks-completed-count.md** — the additive `Task.completed_pomodoro_count` field and `ListTasks` population semantics; the new `CountCompletedPomodorosByTask` query shape.
- **contracts/pomodoro-row-rendering.md** — the precise visual contract: glyph, counts, color/emphasis per segment, empty-state omission, plain-mode behavior. This is the testable spec for `renderPomodoroRow`.
- **quickstart.md** — manual + automated verification steps, including the worked example from the spec.
- **Agent context** — update the `CLAUDE.md` SPECKIT block to point at this plan.

### Post-Design Constitution Re-Check

All four principles remain ✅ after design. No complexity requiring justification was introduced; the Complexity Tracking table below is intentionally empty.

## Complexity Tracking

> No Constitution violations. No entries required.
