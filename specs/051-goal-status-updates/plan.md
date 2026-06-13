# Implementation Plan: Goal Status Updates

**Branch**: `051-goal-status-updates` | **Date**: 2026-06-12 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/051-goal-status-updates/spec.md`

**Design**: [docs/plans/2026-06-12-goal-status-updates-design.md](../../docs/plans/2026-06-12-goal-status-updates-design.md)

## Summary

Let users record timestamped, potentially long markdown notes ("status
updates") against a goal and read them back later. A goal accumulates a history
of updates; the newest is shown as the "latest status". TUI only — no CLI or web
surface. Technical approach: extend the existing `goal/v1` `GoalService` with
four status-update RPCs and a read-only `latest_status_update` field on `Goal`
(populated by `ListGoals`/`GetGoal` so the detail pane needs no extra fetch);
one migration adding a `goal_status_updates` table with `ON DELETE CASCADE` from
`goals`; sqlc queries scoped to the owning user via a join to `goals`. The TUI
Goals tab gains a "Latest status" section in the detail pane (`s` opens a
scrollable status-history view: master list → full reader; `S` quick-adds via
`$EDITOR`). Composing and editing reuse the existing `$EDITOR` flow
(`openEditorCmd`) used for descriptions.

## Technical Context

**Language/Version**: Go 1.26 (three existing modules: root CLI/TUI, `api/`, `services/twig/`)

**Primary Dependencies**: Bubble Tea / charm.land bubbles v2 (TUI), ConnectRPC (`connectrpc.com/connect`), buf (proto gen), sqlc + pgx/v5 (server queries), golang-migrate (schema), internal `markdown` renderer (feature 050)

**Storage**: PostgreSQL — one new migration (`000011_goal_status_updates`): `goal_status_updates` table, FK `goal_id → goals(id) ON DELETE CASCADE`

**Testing**: `go test ./...` per module; handler tests via `export_test.go` shims (no DB required, per repo convention); TUI tests table-style with string assertions like `goal_view_test.go`; migration up/down

**Target Platform**: Linux/macOS terminals (TUI); Linux server container

**Project Type**: Multi-module client/server — TUI client + ConnectRPC server. CLI and web app explicitly out of scope (spec)

**Performance Goals**: Detail-pane latest status adds zero round-trips beyond the existing `ListGoals` fetch (latest embedded in `Goal`); full history is one `ListGoalStatusUpdates` call when the history view opens

**Constraints**: Empty/whitespace body rejected client- and server-side (FR-005); deleting a goal cascades to its updates (FR-012); long bodies fully readable via the reader's scroll (FR-010, SC-004); markdown rendered via the existing renderer (FR-008); playful copy on all new user-facing text (Principle IV)

**Scale/Scope**: Personal data — tens of goals, a handful-to-dozens of updates per goal; full-list fetch per goal is well within the app's established patterns

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One migration, one new table, four RPCs on the existing `GoalService` (no new service); no denormalized `user_id` (scope via join); reuse `openEditorCmd`, the markdown renderer, and existing two-pane/help patterns. The reader's scroll is the only genuinely new TUI mechanic and is required by the spec (long bodies). |
| II. API-First Design | ✅ | `contracts/status-update-rpc.md` (goal/v1 proto additions) committed before implementation; tasks must reference it. |
| III. UI/UX Consistency | ✅ | New keys (`s`, `S`) join the Goals-tab keymap with help entries; history/reader use the shared `theme.go`, the same full-view replacement pattern as other views, and the established `$EDITOR` compose flow. |
| IV. Playful User Messages | ✅ | Empty-history state, delete confirmation, empty-body discard notice, and not-found errors specified with playful copy; tone review is a quality gate. |

**Post-design re-check (after Phase 1)**: all four principles still ✅ — the
design artifacts introduced no new violations; Complexity Tracking left empty.

## Project Structure

### Documentation (this feature)

```text
specs/051-goal-status-updates/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── status-update-rpc.md
├── checklists/
│   └── requirements.md  # from /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
api/
├── proto/goal/v1/goal.proto         # + StatusUpdate msg, 4 RPCs, Goal.latest_status_update
└── gen/goal/v1/                      # regenerated (make proto)

services/twig/
├── db/migrations/000011_goal_status_updates.{up,down}.sql   # new table
├── db/queries/goal_status_update.sql                        # sqlc queries
├── internal/db/                                             # regenerated (sqlc generate)
├── internal/handler/goal.go                                 # + status-update RPCs, latest population
└── internal/handler/goal_test.go                            # + handler tests

internal/tui/                          # root module
├── goal_view.go                        # detail-pane "Latest status" section
├── goal_status.go (new)                # history list + reader render + key handlers
├── update.go                           # new modes, msgs, RPC command factories
├── keymap.go                           # GoalStatusHistory (s), GoalAddStatus (S) bindings
├── client.go                           # (if a goal client accessor is needed)
└── *_test.go                           # goal_status_test.go, keymap/help, view tests
```

**Structure Decision**: Multi-module client/server, mirroring feature 049. No
new packages — the feature extends the existing `goal/v1` proto, the
`services/twig` handler/db layers, and the root `internal/tui` package. A single
new TUI file (`goal_status.go`) holds the history view to keep `update.go` and
`goal_view.go` focused.

## Complexity Tracking

> No Constitution Check violations — table intentionally empty.
