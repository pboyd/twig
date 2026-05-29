# Implementation Plan: Planning Tab Refinements

**Branch**: `022-planning-tab-refinements` | **Date**: 2026-05-29 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/022-planning-tab-refinements/spec.md`

## Summary

A second polish pass on the TUI Planning tab, tightening its interaction model and selection styling to match the Tasks tab. Six changes, all confined to `internal/tui` plus one additive change to the shared `cli` grid renderer:

1. **Single Enter-driven Edit form** — collapse the separate `rename` and `move` prompts into one form (Name + Start + Duration) opened with `Enter`, reusing the existing `RenamePlanEntry` and `MovePlanEntry` RPCs (composed in one command; no proto/server change).
2. **Right-pane forms** — render the Planning picker and all add/edit forms in the right pane beside the grid (mirroring the Tasks tab's `viewWithForm`/`viewWithMove`), replacing the current full-width modal path.
3. **`t` adds a task** — rebind `PlanAddTask` from `a` to `t`; rebind `PlanToday` (jump-to-today) from `t` to `.`.
4. **Remove `clear`** — delete the `PlanClear` binding, `planClear` mode, and its form/submit/command path.
5. **Cell-only selection highlight** — scope the grid's selection styling to the entry's text cell using the Tasks tab's `highlightStyle` (bold + blue background + white foreground), leaving the hour gutter, rails, and box border unstyled.
6. **Enter edits on the Tasks tab** — rebind `Edit` from `e` to `enter` in the Tasks list handler.

The work reuses existing primitives (`paneBox`, `splitLines`/`padRightAnsi`, `highlightStyle`, the planning-aware `FullHelp`, the existing rename/move RPCs and commands) rather than introducing new abstractions. The only cross-package contract touched is `cli.GridOptions`, extended additively so the grid can style the selected entry's content cell without changing plain `todo plan` CLI output.

## Technical Context

**Language/Version**: Go 1.25.0

**Primary Dependencies**: Bubble Tea (`charmbracelet/bubbletea`), Lip Gloss (`charmbracelet/lipgloss`), Bubbles (`charmbracelet/bubbles` — `help`, `textinput`, `key`); ConnectRPC client for the Plan service (already wired). No new dependencies.

**Storage**: N/A — client-side TUI only. No DB, proto, or server changes. Existing `RenamePlanEntry`/`MovePlanEntry`/`AddPlanTask`/`AddPlanEvent`/`RemovePlanEntry` RPCs are reused unchanged; `ClearPlan` remains on the server/CLI but is no longer reachable from the TUI.

**Testing**: `go test ./...` from `services/todo`. TUI is covered by string-view tests (`view_test.go`, `plan_view_test.go`) and reducer tests (`update_test.go`, `plan_update_test.go`, `move_test.go`, `edit_test.go`) using `export_test.go` shims and a fixed clock / stub clients. Grid selection rendering is covered in `internal/cli` (`plan_grid` tests).

**Target Platform**: Terminal (TTY) on Linux/macOS; styling gated by `cli.WantStyled(os.Stdout)` (the `m.styled` flag) with a plain-text fallback.

**Project Type**: Single Go module, CLI + TUI client. This feature is UI-only.

**Performance Goals**: Interactive redraw on each key/tick; no new polling beyond the existing 1-second now-marker tick.

**Constraints**: Must degrade gracefully on narrow terminals (existing `splitLines`/`padRightAnsi` path) and on non-ANSI terminals (the `!m.styled` branch must remain correct and escape-free). The merged Edit form must not introduce a combined-update RPC — it composes the two existing RPCs.

**Scale/Scope**: ~7 focused changes within `internal/tui`; one additive field on `cli.GridOptions` plus a reworked `applySelection`. No new packages.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Principle I — Simplicity / YAGNI**: PASS. Every item reuses an existing primitive and removes more surface than it adds:
- Merged Edit → one `planEdit` form mode (Name + Start + Duration) whose submit composes the existing `renamePlanCmd` + `movePlanCmd`; deletes the two separate `planRename`/`planMove` modes and their init/submit pairs. No new RPC.
- Right-pane forms → reuse the Tasks tab's existing two-pane join (`paneBox` + `lipgloss.JoinHorizontal`, and the non-styled `splitLines`/`padRightAnsi` row-join); the Planning right pane already exists from feature 021.
- `t`/`.` rebind and `clear` removal → keymap edits plus deletion of `planClear` mode, `initClearForm`, `submitClearForm`, and `clearPlanCmd`'s TUI call site. Net deletion.
- Cell-only highlight → rework `applySelection` to style only the content cell with the existing `highlightStyle`, threaded through one additive `GridOptions` field. No new rendering pipeline.
- Enter-to-edit on Tasks → change the `Edit` binding's keys from `e` to `enter`; the existing `handleListKey` case is untouched.

No new abstraction is introduced; the Complexity Tracking table is empty.

**Principle II — API-First Design**: PASS (with note). This feature adds no network API/proto contracts and changes no `.proto` files. The one cross-package contract it touches is the shared `cli.GridOptions` struct (the boundary between the TUI and the shared grid renderer). That change is additive and backward-compatible, and is documented in `contracts/grid-selection.md` before implementation, satisfying the "contract before implementation" rule. The Planning tab's keybinding + view/UX contract is documented in `contracts/planning-ui.md`. The reuse of the existing `RenamePlanEntry`/`MovePlanEntry` RPCs by the merged Edit form is documented there as well; those proto contracts are unchanged.

## Project Structure

### Documentation (this feature)

```text
specs/022-planning-tab-refinements/
├── plan.md              # This file
├── research.md          # Phase 0 output — design decisions
├── data-model.md        # Phase 1 output — view-state model deltas
├── quickstart.md        # Phase 1 output — manual verification walkthrough
├── contracts/
│   ├── grid-selection.md # Additive cli.GridOptions selection-styling contract
│   └── planning-ui.md    # Planning + Tasks key & view contract (post-change)
└── checklists/
    └── requirements.md   # Spec quality checklist (from /speckit-specify)
```

### Source Code (repository root)

All changes live under the single Go module at `services/todo/`:

```text
services/todo/
├── internal/tui/
│   ├── keymap.go        # Edit→"enter"; PlanAddTask→"t"; PlanToday→"."; add PlanEdit;
│   │                    #   drop PlanRename/PlanMove/PlanClear; update Short/FullHelp
│   ├── model.go         # replace planRename/planMove/planClear modes with planEdit
│   ├── update.go        # handleListKey: Edit case now Enter; handlePlanningKey:
│   │                    #   Enter→initEditForm, t→add task, .→today; drop rename/move/clear cases
│   ├── plan_update.go   # initEditForm (Name+Start+Duration); submitEditForm composes
│   │                    #   rename+move; delete initRename/initMove/initClear & submits
│   ├── view.go          # viewPlanning: route plan.mode≠planList to a right-pane join
│   │                    #   (mirror viewWithForm) instead of the full-width modal path
│   ├── plan_view.go     # renderPlanFormView/renderPlanPickerView sized for the right pane;
│   │                    #   planFieldLabel for planEdit
│   └── *_test.go        # view + reducer tests for each change
└── internal/cli/
    └── plan_grid.go     # GridOptions: cell-only selection styling reusing the TUI
                         #   highlight; applySelection styles content, not gutter/border
```

**Structure Decision**: Single-module layout (the project default). No new directories or packages; the feature edits existing `internal/tui` files and makes one additive change to `internal/cli/plan_grid.go`. This keeps the change surface minimal and aligned with Principle I.

## Complexity Tracking

> No Constitution Check violations. No complexity to justify.
