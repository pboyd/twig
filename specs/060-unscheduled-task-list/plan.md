# Implementation Plan: Unscheduled Tasks as a List

**Branch**: `060-unscheduled-task-list` | **Date**: 2026-07-02 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/060-unscheduled-task-list/spec.md`

## Summary

Change how unscheduled (untimed) plan entries are presented in the Plan tab: render them as a
list of Tasks-tab-style rows — a completion checkbox (☐/☑) followed by the task name — sitting
above the day planner grid, separated from it by the existing horizontal divider. Today they are
drawn as time-block boxes (the same visual language as scheduled entries), which miscommunicates
that they don't own a slot on the calendar.

The change is localized to one rendering function, `RenderUntimed` in `internal/cli/plan_grid.go`,
which is shared by both the TUI Plan tab and the `twig plan` CLI command. No proto, server, DB, or
web-app code changes. The layout/height model and all interaction plumbing (selection, complete,
edit, schedule, reorder, details) already operate on plan-entry IDs and are unaffected.

## Technical Context

**Language/Version**: Go (module `github.com/pboyd/twig`, root module)

**Primary Dependencies**: `charmbracelet/lipgloss` + `bubbletea` (TUI rendering); no new dependencies

**Storage**: N/A — no persistence or schema change

**Testing**: `go test ./...` (table-driven string/golden assertions in `internal/cli` and `internal/tui`)

**Target Platform**: Terminal (interactive TUI on a TTY; plain-text CLI `twig plan` fallback)

**Project Type**: CLI/TUI — all affected code lives in the root module (`internal/cli`, `internal/tui`)

**Performance Goals**: Instant redraw; rendering a day's tasks is trivial (tens of entries)

**Constraints**: Presentation-only — no functional/behavioral change (FR-005); layout must not break at any width/height (FR-008); no-unscheduled days render byte-identical to today (FR-006)

**Scale/Scope**: A single day's plan entries (typically < 30); single shared render function + its tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Rewrites one function to emit simpler single-row output (drops duration→box geometry). Reuses existing `GridOptions` selection/completion plumbing. No new abstraction, type, or config flag. |
| II. API-First Design | ✅ | No proto/endpoint/DB contract. The user-facing surface contract (the untimed row format) is documented in `contracts/untimed-list-rendering.md` before implementation, satisfying the "contract first" intent for this CLI/TUI surface. |
| III. UI/UX Consistency | ✅ | Adopts the Tasks-tab checkbox row style (☐/☑) drawn from the existing shared theme; no ad-hoc glyphs/colors. Changing the shared renderer keeps the TUI Plan tab and `twig plan` CLI output coherent rather than divergent. |
| IV. Playful User Messages | ✅ | No new user-facing copy. Per FR-006 the section is omitted entirely when empty (no header/empty-state string), so no tone surface is added. |

**Result**: PASS — no violations; Complexity Tracking table left empty.

## Project Structure

### Documentation (this feature)

```text
specs/060-unscheduled-task-list/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── untimed-list-rendering.md   # UI render contract for the untimed list
└── tasks.md             # Phase 2 output (/speckit-tasks — not created here)
```

### Source Code (repository root)

```text
internal/
├── cli/
│   ├── plan_grid.go        # RenderUntimed — REWRITE box output → checkbox+name list rows
│   │                       #   (RenderUntimedSeparator unchanged: divider-only satisfies FR-003)
│   ├── plan_grid_test.go   # UPDATE assertions for the new untimed row format
│   ├── plan.go             # `twig plan` CLI — no code change; inherits new list rendering
│   └── plan_test.go        # UPDATE any assertions on untimed CLI output
└── tui/
    ├── plan_view.go        # No logic change: three render paths already pass
    │                       #   GridOptions (SelectedID/SelectionStyle) to RenderUntimed.
    │                       #   Height model already grows list / shrinks grid (FR-008).
    └── plan_view_test.go   # UPDATE assertions for the new untimed row format
```

**Structure Decision**: Single (root) Go module. The entire change is concentrated in
`internal/cli/plan_grid.go::RenderUntimed`. Because the TUI already threads selection and
completion state through `GridOptions`, and the height allocation already subtracts the untimed
pane's line count from the grid ("list grows, grid shrinks"), no changes are required in
`plan_view.go` beyond what already exists. The shared renderer means the `twig plan` CLI command
picks up the same list presentation for free (an intentional consistency choice — see research.md
Decision 4).

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.
