# Implementation Plan: TUI Completed Task Display

**Branch**: `016-tui-completed-task-display` | **Date**: 2026-05-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/016-tui-completed-task-display/spec.md`

## Summary

Two related TUI rendering changes:

1. **Linger-on-complete (Story 1)**: When the user marks a task complete from the tree, the row stays in place, gets strikethrough styling, and remains selected. The next selection-changing action (`up`/`down`/jump-to) drops it.
2. **Completion styling in tree (Story 2)**: Every completed task that is visible in the tree (today only happens with the "show completed" filter on, or for a lingering just-completed row) renders with strikethrough on its name. The detail pane's name no longer carries strikethrough (the explicit `Completed: <timestamp>` line already conveys state).

Technical approach: the codebase already has a `pendingComplete *int64` field on the TUI model and `buildVisible` already honors it (the just-completed row is kept visible regardless of the `showCompleted` filter), but `pendingComplete` is never *set* — it's plumbed but inert. The implementation sets it in the `Complete` key handler, ensures it is cleared on `up`/`down` (already done) and on refresh, and updates `view.go::renderList` to apply strikethrough to any row whose task is completed. `renderDetails` drops its `cli.DimStrike` call on the name.

## Technical Context

**Language/Version**: Go 1.x (matches existing module at `services/todo/`)

**Primary Dependencies**: `bubbletea` (TUI loop), `lipgloss` (style rendering), existing `internal/cli` helpers (`DimStrike` and ANSI rendering utilities — a sibling `Strike`-only helper will be added)

**Storage**: N/A (view-only feature; no database changes, no API changes, no proto changes)

**Testing**: Standard `go test`; package already has `view_test`-style coverage via `tree_test.go`, `update_test.go`, and `export_test.go` shims. New tests live next to the modified files.

**Target Platform**: Terminal (ANSI-capable); behavior is gated by the existing TTY detection in `internal/cli/render.go`.

**Project Type**: Single Go module (TUI client binary `cmd/todo` + server). Only the client TUI package is affected.

**Performance Goals**: No new perf constraints. Each render pass already iterates `m.visible`; adding a strikethrough wrap per row is O(visible-rows) and indistinguishable from current cost.

**Constraints**: No new dependencies; no protobuf or sqlc regen; no DB migration; no API change. Behavior must be unchanged in non-TTY mode (strikethrough already no-ops via `DimStrike`/`Strike` helpers when `isTTY=false`).

**Scale/Scope**: ~3 files modified (`internal/tui/update.go`, `internal/tui/view.go`, `internal/tui/details.go`), 1 helper added to `internal/cli/render.go`, plus tests.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Simplicity / YAGNI** — ✅ Pass. The change reuses an existing-but-inert field (`pendingComplete`) and an existing helper pattern (`DimStrike`). One new sibling helper (`Strike`, strikethrough-only) is added because Q3 in the spec mandated strikethrough without a color change, and the existing `DimStrike` couples both. No new abstractions, no new files beyond tests.
- **II. API-First Design** — ✅ Pass / N/A. There are no external interfaces in scope: no proto changes, no HTTP/Connect endpoint changes, no CLI command surface changes. The "contract" surface in this feature is internal TUI rendering, captured in `contracts/tui-rendering.md` (see Phase 1).

No violations to log in Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/016-tui-completed-task-display/
├── plan.md              # This file (/speckit-plan command output)
├── spec.md              # Feature spec (already exists)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (view-model only — no persisted entities)
├── quickstart.md        # Phase 1 output (how to verify the feature locally)
├── contracts/
│   └── tui-rendering.md # Internal "contract" for tree row + detail pane rendering
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks command)
```

### Source Code (repository root)

```text
services/todo/
├── internal/
│   ├── cli/
│   │   ├── render.go            # MODIFIED: add Strike (strikethrough-only) helper alongside DimStrike
│   │   └── render_test.go       # MODIFIED: cover Strike
│   └── tui/
│       ├── update.go            # MODIFIED: set m.pendingComplete = id in Complete handler; ensure it's cleared on refresh
│       ├── update_test.go       # MODIFIED: cover lingering + clear-on-navigation
│       ├── view.go              # MODIFIED: renderList wraps completed rows in Strike
│       ├── details.go           # MODIFIED: drop DimStrike on name (Q2)
│       └── tree_test.go         # MODIFIED if any expectations on rendering change
└── cmd/                          # unchanged
```

**Structure Decision**: Modify only `services/todo/internal/tui/*` (and a small helper in `services/todo/internal/cli/render.go`). No new packages, no new binaries, no proto or sqlc regeneration.

## Complexity Tracking

> No violations — table omitted.
