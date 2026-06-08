# Implementation Plan: Auto-Schedule a Task

**Branch**: `043-auto-schedule-task` | **Date**: 2026-06-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/043-auto-schedule-task/spec.md`

## Summary

Add an `a` key binding to the TUI planning tab that auto-schedules the highlighted **task** entry: it finds the earliest free slot in the current plan day — at or after a scheduling floor (8:00 AM, or the current time when the day is today and it is already past 8:00 AM) — that is long enough to hold the task's duration (or a default 30-minute block when the task has no duration), and moves the task there. Events and an empty highlight are no-ops with a playful explanation. The core slot-finding is a pure function added to `internal/cli` (next to `GridWindow`), and the move reuses the existing `MovePlanEntry` RPC via `movePlanCmd`, which already re-highlights the moved entry (satisfying "highlight follows the task").

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: Bubble Tea (TUI), ConnectRPC client (`planv1connect`), existing `internal/cli` plan grid helpers; no new dependencies

**Storage**: PostgreSQL via the server's existing `MovePlanEntry` RPC — no schema or query changes (CLI/TUI side only)

**Testing**: `go test ./...` (root module). New unit tests for the slot-finder in `internal/cli`; TUI handler test in `internal/tui` using the existing `export_test.go` shim pattern

**Target Platform**: Terminal (TTY) on Linux/macOS — the interactive TUI

**Project Type**: CLI/TUI client within a three-module Go workspace (this feature touches only the root CLI/TUI module)

**Performance Goals**: Instant from the user's perspective — single keypress; slot search is O(n log n) over a single day's entries (tiny n)

**Constraints**: No new API endpoint; reuse `MovePlanEntry`. Slot search confined to one day (floor → 1440). Start time chosen at exact minute (no snapping). All new user-facing text must be playful (Principle IV)

**Scale/Scope**: A day holds at most a few dozen plan entries; no scale concerns

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One pure slot-finder function + one key handler that reuses `movePlanCmd`. No new abstractions, no new RPC, no config. The 8 AM floor and 30-min default are hard-coded constants per spec (not made configurable). |
| II. API-First Design | ✅ | No new data contract — reuses the existing `plan.v1.PlanService/MovePlanEntry` RPC. The behavioral contract (key binding + reused RPC + slot-finder signature) is documented in `contracts/`. |
| III. UI/UX Consistency | ✅ | The `a` binding is registered in the shared `KeyMap` and surfaced in planning help alongside `t`/`e`/`enter`, matching existing planning-action conventions. No new colors/styles. |
| IV. Playful User Messages | ✅ | Event no-op, no-fit, and no-duration-handled messages are written in the warm/witty house tone, mirroring existing planning notices (e.g. the Complete/PomStart event guards). |

**Post-Phase 1 re-check**: No change — design introduces no new abstractions, surfaces, or contracts beyond the above. Gate remains ✅.

## Project Structure

### Documentation (this feature)

```text
specs/043-auto-schedule-task/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── auto-schedule.md # Keybinding + slot-finder + reused-RPC contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
internal/cli/
├── plan_grid.go         # ADD: AutoScheduleSlot(timed, durationMin, floorMin, excludeID) (start int, ok bool)
└── plan_grid_test.go    # ADD: unit tests for AutoScheduleSlot (gaps, floor, no-fit, exclude-self, default block)

internal/tui/
├── keymap.go            # ADD: PlanAutoSchedule key.Binding ("a"); include in planning ShortHelp/FullHelp
├── update.go            # ADD: case key.Matches(msg, m.keys.PlanAutoSchedule) in handlePlanKey — event/empty/no-fit guards, compute floor, call AutoScheduleSlot, dispatch movePlanCmd
├── update_test.go       # ADD: handler tests (task moved, event no-op + notice, no-fit notice, today-floor)
└── export_test.go       # ADD: shim(s) if the handler needs to expose an unexported helper to tests
```

**Structure Decision**: This is a TUI-only change in the root module. The slot-finding algorithm lives as a pure, table-tested function in `internal/cli` (the home of the existing pure plan-grid helpers such as `GridWindow` and `splitPlanEntries`), keeping rendering-free logic separate from the Bubble Tea handler. The handler in `internal/tui/update.go` wires the key to the algorithm and the existing `movePlanCmd`. No server (`services/twig`), proto (`api/`), or web (`services/twig-web`) changes.

## Complexity Tracking

> No Constitution Check violations — table intentionally empty.
