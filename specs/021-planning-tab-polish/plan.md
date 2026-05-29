# Implementation Plan: Planning Tab Polish

**Branch**: `021-planning-tab-polish` | **Date**: 2026-05-29 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/021-planning-tab-polish/spec.md`

## Summary

Bring the TUI Planning tab to parity with the Tasks tab in both behavior and visual feel. Three behavior gaps (pomodoro-cancel inert on Planning, no Planning help screen, add-task picker hides sub-tasks) are fixed by routing to controls that already exist on the Tasks side. The visual gap (monotone vs. blue, single-pane vs. two-pane, floating status line, redundant panel titles) is closed by reusing the Tasks tab's existing rendering primitives — `paneBox`, the shared theme palette, the `help` model's already-planning-aware `FullHelp`, and the list flattener `buildVisible` — rather than building new abstractions. Work is confined to the `internal/tui` package plus a small, backward-compatible extension to the shared `cli.GridOptions` so the grid can be themed in the TUI without changing the plain `todo plan` CLI output.

## Technical Context

**Language/Version**: Go 1.25.0

**Primary Dependencies**: Bubble Tea (`charmbracelet/bubbletea`), Lip Gloss (`charmbracelet/lipgloss`), Bubbles (`charmbracelet/bubbles` — `help`, `textinput`, `key`); ConnectRPC clients for Task/Plan/Pomodoro services (already wired).

**Storage**: N/A — client-side TUI only. No DB, proto, or server changes. Existing Plan/Task/Pomodoro RPCs are reused unchanged.

**Testing**: `go test ./...` from `services/todo`. TUI is covered by string-view tests (`view_test.go`, `plan_view_test.go`) and reducer tests (`update_test.go`, `plan_update_test.go`) using `export_test.go` shims and a fixed clock / stub clients.

**Target Platform**: Terminal (TTY) on Linux/macOS; styling gated by `cli.WantStyled(os.Stdout)` (the `m.styled` flag) with a plain-text fallback.

**Project Type**: Single Go module, CLI + TUI client. This feature is UI-only.

**Performance Goals**: Interactive redraw on each key/tick; no new polling beyond the existing 1-second now-marker tick.

**Constraints**: Must degrade gracefully on narrow terminals (existing `splitLines`/`padRightAnsi` path) and on non-ANSI terminals (the `!m.styled` branch must remain correct and escape-free).

**Scale/Scope**: ~6 focused changes within `internal/tui`; one additive field on `cli.GridOptions`. No new packages.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Principle I — Simplicity / YAGNI**: PASS. Every item reuses an existing primitive:
- Pomodoro cancel → add one `key.Matches(msg, m.keys.PomCancel)` case in `handlePlanningKey`, calling the existing `cancelPomCmd`.
- Help → reuse the existing `modeHelp`/`viewHelp()` path; `KeyMap.FullHelp()` already branches on `PlanningMode`, so no new help content is authored.
- Sub-tasks in picker → reuse `buildVisible` with a fully-expanded map (the same flattener the Tasks list uses); no new tree walker.
- Two-pane + bottom status line + titles → reuse `paneBox`, `splitLines`, `padRightAnsi`, and the shared theme palette already used by `viewList`.
- Grid theming → one additive, optional field on the existing `GridOptions` struct; no new rendering pipeline.

No new abstraction is introduced; therefore the Complexity Tracking table is empty.

**Principle II — API-First Design**: PASS (with note). This feature adds no network API/proto contracts. The one cross-package contract it touches is the shared `cli.GridOptions` struct (the boundary between the TUI and the shared grid renderer). That contract change is additive and backward-compatible, and is documented in `contracts/grid-options.md` before implementation, satisfying the "contract before implementation" rule. The UI/keybinding contract for the Planning tab is documented in `contracts/planning-ui.md`.

## Project Structure

### Documentation (this feature)

```text
specs/021-planning-tab-polish/
├── plan.md              # This file
├── research.md          # Phase 0 output — design decisions
├── data-model.md        # Phase 1 output — view-state model deltas
├── quickstart.md        # Phase 1 output — manual verification walkthrough
├── contracts/
│   ├── grid-options.md  # Additive cli.GridOptions contract
│   └── planning-ui.md   # Planning tab key + view contract
└── checklists/
    └── requirements.md  # Spec quality checklist (from /speckit-specify)
```

### Source Code (repository root)

All changes live under the single Go module at `services/todo/`:

```text
services/todo/
├── internal/tui/
│   ├── update.go        # + PomCancel & quit-confirm handling in handlePlanningKey;
│   │                    #   + Help routing on Planning; picker built fully-expanded
│   ├── view.go          # viewPlanning → two-pane (paneBox) w/ bottom-pinned status;
│   │                    #   drop redundant "Tasks"/"Planning" pane titles
│   ├── plan_view.go     # + renderPlanDetail (right pane); themed selection/header
│   ├── plan_grid.go     # planGridOptions sets the new accent styling option
│   ├── view.go / theme  # reuse highlightStyle/cursorBg/accent for grid selection
│   ├── keymap.go        # ensure PomCancel surfaces in Planning ShortHelp/FullHelp
│   └── *_test.go        # view + reducer tests for each change
└── internal/cli/
    └── plan_grid.go     # GridOptions: additive optional accent/selection styling
```

**Structure Decision**: Single-module layout (the project default). No new directories or packages; the feature edits existing `internal/tui` files and makes one additive change to `internal/cli/plan_grid.go`. This keeps the change surface minimal and aligned with Principle I.

## Complexity Tracking

> No Constitution Check violations. No complexity to justify.
