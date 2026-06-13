# Implementation Plan: Auto-Assign Enhancements

**Branch**: `052-auto-assign-enhancements` | **Date**: 2026-06-13 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/052-auto-assign-enhancements/spec.md`

## Summary

Extend the TUI planning tab's `a` auto-schedule action with two refinements, both implemented entirely in the CLI/TUI module (no API, proto, or database changes):

1. **15-minute rounding** — auto-assigned start times always land on a quarter-hour boundary. The day floor (8 AM, or the later of 8 AM and the current minute when planning today) is rounded **down** to a boundary by the TUI handler (so 9:03 → 9:00, even though that's in the past). Slot starts derived from a preceding entry's end are rounded **up** inside `AutoScheduleSlot` so the task never overlaps.
2. **Bump on already-placed** — when the highlighted task already sits in its earliest-fitting slot, pressing `a` no longer no-ops. Instead the handler computes a new floor at the end of the entry that closes the task's current free stretch and re-runs the slot search, landing the task in the next distinct free gap (or surfacing the existing "no room" notice when there is none).

The change reuses the existing `cli.AutoScheduleSlot` helper, the existing `movePlanCmd` → `PlanService.MovePlanEntry` RPC, and the existing notice strings. One small pure helper (`NextGapFloor`) is added beside `AutoScheduleSlot`.

## Technical Context

**Language/Version**: Go 1.x (repo root module `github.com/pboyd/twig`)

**Primary Dependencies**: Bubble Tea (TUI), existing ConnectRPC `PlanService` client — no new dependencies

**Storage**: N/A — no persistence changes; placement is sent via the existing `MovePlanEntry` RPC

**Testing**: `go test ./...` — table/unit tests in `internal/cli/plan_grid_test.go` and `internal/tui/plan_update_test.go`

**Target Platform**: Terminal (TTY) on Linux/macOS

**Project Type**: Single project — Go CLI/TUI client

**Performance Goals**: Instant (single keypress, in-memory slot scan over a day's entries)

**Constraints**: Pure client-side logic; must not regress existing auto-schedule behavior (feature 043)

**Scale/Scope**: One day's plan entries (tens of entries at most); two functions touched plus their tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses `AutoScheduleSlot` and `MovePlanEntry`. Adds one small pure helper (`NextGapFloor`) rather than a new abstraction layer. Rounding is integer arithmetic, no new types. |
| II. API-First Design | ✅ | No new API, proto, or DB contract. The consumed contract is the existing `PlanService.MovePlanEntry` RPC; the feature's behavioral contract (helper functions + keybinding) is documented in `contracts/auto-assign-behavior.md`. |
| III. UI/UX Consistency | ✅ | No new surface. Same `a` keybinding, same planning-tab highlight behavior, same notice styling as feature 043. |
| IV. Playful User Messages | ✅ | Reuses the existing playful "Day's packed — no room left to squeeze this one in." notice for the bump's no-room case. No new user-facing copy introduced. |

**Result**: PASS — no violations, Complexity Tracking not required.

## Project Structure

### Documentation (this feature)

```text
specs/052-auto-assign-enhancements/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── auto-assign-behavior.md   # Phase 1 output (behavioral contract)
└── checklists/
    └── requirements.md  # From /speckit-specify
```

### Source Code (repository root)

```text
internal/
├── cli/
│   ├── plan_grid.go         # AutoScheduleSlot: align candidate starts UP to 15-min boundary;
│   │                        #   add NextGapFloor helper for the bump
│   └── plan_grid_test.go    # Unit tests for rounding + NextGapFloor
└── tui/
    ├── update.go            # `a` handler: round day floor DOWN to boundary; bump-instead-of-no-op
    └── plan_update_test.go  # Handler tests: floor rounding, bump, regressions
```

**Structure Decision**: Single Go project. All changes live in the root module's `internal/cli` (pure slot-math helpers) and `internal/tui` (the `a` key handler). No changes to the `api/` or `services/twig/` modules.

## Phase 0: Research

See [research.md](./research.md). Key decisions:

- **Rounding split**: handler rounds the day floor **down** (it owns `now`, so it owns the "even if in the past" rule); `AutoScheduleSlot` rounds every candidate gap-start **up** to the next boundary (it owns the obstacle geometry). A down-rounded boundary floor is a no-op under the internal round-up, so the two compose cleanly.
- **Bump mechanism**: add `NextGapFloor(timed, fromMin, excludeID)` returning the end minute of the first entry that begins at/after `fromMin` (the entry closing the current stretch). The handler calls `AutoScheduleSlot` again with that floor. `ok=false` ⇒ no later gap ⇒ existing "no room" notice.
- **Bump trigger**: only when `entry.StartMinute != nil && *entry.StartMinute == startMin` (the former no-op condition). All other cases keep the existing earliest-fit placement, preserving the re-home-earlier behavior.

## Phase 1: Design & Contracts

- **Data model**: [data-model.md](./data-model.md) — no new entities; documents the in-memory plan-entry shape and the free-gap/boundary concepts the math operates on.
- **Contract**: [contracts/auto-assign-behavior.md](./contracts/auto-assign-behavior.md) — function signatures and behavioral guarantees for `AutoScheduleSlot` (revised) and `NextGapFloor` (new), plus the `a`-key handler decision table.
- **Quickstart**: [quickstart.md](./quickstart.md) — how to exercise the two enhancements by hand and which tests to run.
- **Agent context**: `CLAUDE.md` SPECKIT marker updated to point at this plan.

## Complexity Tracking

No Constitution violations — table intentionally empty.
