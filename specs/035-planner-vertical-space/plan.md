# Implementation Plan: Planner Vertical Space

**Branch**: `035-planner-vertical-space` | **Date**: 2026-06-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/035-planner-vertical-space/spec.md`

## Summary

Make the TUI Planning tab's calendar grid height-aware. Today the grid always
draws a fixed window (default 08:00–17:00, expanded only to *contain* entries)
and the TUI simply truncates the rendered output to fit. This wastes the lower
half of tall terminals and, on short terminals, fills scarce rows with already-
elapsed morning hours.

Technical approach: introduce a pure window-computation helper in the `cli`
package that, given the day's entries, `now`, the day string, and the number of
rows available for the timed grid, returns the `[startMin, endMin]` window. The
TUI computes available rows (after the untimed pane + separator) and passes the
resulting window to `RenderGrid` via two new optional `GridOptions` fields. When
those fields are unset (the CLI's `GridOptions{}`), `RenderGrid` behaves exactly
as today — guaranteeing byte-for-byte CLI parity.

Two render modes follow from the window helper:
- **Fill** (room to spare): keep the 08:00 start, extend the end later — toward
  end of day — to consume the available rows.
- **Anchor** (room too tight, viewing today): start the grid at the 15-minute
  block containing `now`; earlier blocks are omitted.

## Technical Context

**Language/Version**: Go 1.25.0

**Primary Dependencies**: charmbracelet/bubbletea + lipgloss (TUI); rendering is
plain string building in `internal/cli` (no new dependencies)

**Storage**: N/A — pure rendering change; no DB, proto, or API surface touched

**Testing**: `go test ./...`; table-driven unit tests in `internal/cli`
(`plan_grid_test.go`) and `internal/tui` (`plan_view_test.go`, `plan_us2_test.go`)

**Target Platform**: Terminal TUI (Linux/macOS) on a TTY

**Project Type**: Single Go module (`services/twig/`) with CLI/TUI + server binaries

**Performance Goals**: Window recomputed every frame; helper is O(entries) integer
math — negligible (<1ms) relative to a render

**Constraints**: CLI (non-TUI) day-planner output MUST stay byte-for-byte
identical (FR-008); behavior MUST degrade gracefully on tiny terminals (FR-010)

**Scale/Scope**: A single day's plan entries (typically <50); window spans at most
00:00–24:00

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Two optional struct fields + one pure helper + wiring at the existing 3 grid call sites. No new abstraction layer; reuses existing grid geometry. |
| II. API-First Design | ✅ | No request/response or proto change. The only contract is the internal rendering interface (GridOptions extension + window helper signature), documented in `contracts/`. |
| III. UI/UX Consistency | ✅ | Reuses the existing grid box geometry, gutter, now-marker, and theme. No new colors or styles. Same window vocabulary (08:00 start, 15-min blocks). |
| IV. Playful User Messages | ✅ | No new user-facing copy; this is a layout change. |

**Result**: PASS — no violations, Complexity Tracking not required.

## Project Structure

### Documentation (this feature)

```text
specs/035-planner-vertical-space/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (conceptual: the window value object)
├── quickstart.md        # Phase 1 output (manual verification steps)
├── contracts/
│   └── grid-window.md   # Rendering-interface contract (GridOptions + helper)
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # From /speckit-tasks (not created here)
```

### Source Code (repository root)

```text
services/twig/internal/cli/
├── plan_grid.go         # RenderGrid: add WindowStartMin/WindowEndMin override
│                        #   + new GridWindow() helper (pure window computation)
└── plan_grid_test.go    # Unit tests for GridWindow + override behavior;
                         #   existing default-window tests guard CLI parity

services/twig/internal/tui/
├── plan_view.go         # renderPlanGrid / renderPlanGridContent / renderPlanningView:
│                        #   measure untimed+sep rows, compute window, pass override
└── plan_view_test.go    # TUI tests: fill mode, anchor mode, non-today, tiny terminal
```

**Structure Decision**: Single Go module, existing packages. The windowing
decision lives in `internal/cli` (pure, unit-testable, alongside `RenderGrid`);
the TUI in `internal/tui` measures available height and calls it. No new files
are required — changes are confined to `plan_grid.go` and `plan_view.go` plus
their test files.

## Phase 0: Research

See [research.md](./research.md). All Technical Context items are resolved; the
spec's clarifications removed the open design questions. Research confirms:
- Adding fields to the `GridOptions` struct preserves CLI parity (zero value =
  current behavior).
- Computing the window externally and disabling `RenderGrid`'s auto-expansion on
  any overridden side cleanly supports both fill and anchor modes.
- Out-of-window entries already drop out of `RenderGrid`'s per-row lookup maps,
  giving the desired "show only from the current-time block down" clipping.

## Phase 1: Design & Contracts

- **data-model.md**: the conceptual *visible window* value object and the
  fill/anchor/top-truncate decision table.
- **contracts/grid-window.md**: the rendering interface — new `GridOptions`
  fields and the `GridWindow` helper signature/semantics, with the CLI-parity
  invariant stated explicitly.
- **quickstart.md**: manual TUI verification steps for fill, anchor, non-today,
  and tiny-terminal scenarios, plus the `go test` regression command.
- **Agent context**: update the `<!-- SPECKIT -->` block in `CLAUDE.md` to point
  at this plan.

## Phase 2: (deferred to /speckit-tasks)

Task breakdown is produced by `/speckit-tasks`. Expected shape: (1) `GridWindow`
helper + unit tests, (2) `RenderGrid` override fields + parity tests, (3) TUI
wiring at the three call sites + behavior tests, (4) full `go test ./...` regression.
