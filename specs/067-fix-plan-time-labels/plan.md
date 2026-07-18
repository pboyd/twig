# Implementation Plan: Show Actual Times in Plan Grid Labels

**Branch**: `067-fix-plan-time-labels` | **Date**: 2026-07-18 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/067-fix-plan-time-labels/spec.md`

## Summary

Two defects share one root cause: values computed for **box geometry** (start snapped down and end snapped up to 15-minute boundaries) are reused for two purposes that require the entry's **exact** interval.

The fix is to stop reusing them, in exactly two places:

1. `internal/cli/plan_grid.go:121-127` — build the entry label from `e.GetStartMinute()` and `start + duration` instead of the local `sn`/`se`. The `sn`/`se` variables keep driving row placement, untouched.
2. `internal/tui/plan_preview.go:110-144` — compare exact intervals to decide whether a conflict exists, then mark the slots covering the *overlap region* rather than every slot the preview touches.

No new files, no new abstractions, no API change, no data change. Net effect is roughly ten changed lines plus tests.

## Technical Context

**Language/Version**: Go 1.24 (per `go.mod`)

**Primary Dependencies**: `charm.land/lipgloss/v2` (TUI styling), `planv1` generated protobuf types. No new dependencies.

**Storage**: N/A — entries already store exact `start_minute` and `duration_minute`; nothing about persistence changes.

**Testing**: `go test ./...` from repo root. Table-driven unit tests in `internal/cli/plan_grid_test.go` and `internal/tui/plan_preview_test.go`.

**Target Platform**: Terminal (TUI) and CLI on Linux/macOS.

**Project Type**: CLI/TUI client within a three-module Go workspace. Only the root module is touched.

**Performance Goals**: N/A — the change is arithmetic-for-arithmetic in an already per-entry render loop. Conflict detection stays O(entries × slots).

**Constraints**: Box geometry output must not change for any input (SC-005). The renderer's `PreviewConflictSlots` contract — keys are slot-start minutes, multiples of 15 — must be preserved, because `internal/cli/plan_grid.go:193` and `:215` index that map by slot minute.

**Scale/Scope**: 2 source files, ~10 lines changed, 1 contract document, and new/updated tests.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | No new types, helpers, or indirection. The fix removes a coincidental reuse of two variables. A tempting "exact vs. snapped interval" abstraction was rejected — two inline expressions are simpler than a type. |
| II. API-First Design | ✅ | No wire contract changes. The rendering contract that *is* affected (`PreviewConflictSlots` semantics and label format) is documented in `contracts/grid-label.md`, committed before implementation, amending `specs/039-plan-entry-preview/contracts/grid-preview.md`. |
| III. UI/UX Consistency | ✅ | No new surface, style, color, or key binding. Label format string, ID prefix, truncation, and all styling hooks are unchanged — only the numeric values differ. The hour gutter stays snapped, since it labels the grid, not an entry. |
| IV. Playful User Messages | ✅ | No new user-facing prose. The changed text is clock times, which are data; playfulness does not apply and accuracy is the whole point. No existing copy is touched. |

**Post-Phase 1 re-check**: ✅ All four still pass. Phase 1 added one contract document and no code structure, so nothing in the design shifts any gate.

## Project Structure

### Documentation (this feature)

```text
specs/067-fix-plan-time-labels/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── grid-label.md    # Phase 1 output — label format + conflict-slot semantics
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
internal/cli/
├── plan_grid.go         # MODIFIED — label built from exact times (lines 121-127)
└── plan_grid_test.go    # MODIFIED — add non-aligned label cases

internal/tui/
├── plan_preview.go      # MODIFIED — exact-interval overlap, overlap-region slots
└── plan_preview_test.go # MODIFIED — add non-aligned + sub-slot conflict cases
```

**Structure Decision**: This is a surgical fix inside the existing root CLI/TUI module. The rendering layer lives in `internal/cli` (shared by the CLI's `plan` command and the TUI, which calls `cli.RenderGrid`), and the preview/conflict layer lives in `internal/tui` because it depends on live form state. That split is unchanged; each defect is fixed on the side of the split where it already lives. The `api/` and `services/twig/` modules are not touched.

## Design Detail

### Defect 1 — label text (`internal/cli/plan_grid.go`)

Inside the `layouts` loop, `sn` and `se` continue to compute `tl`/`bl` for row placement. The label gains two new locals derived from the entry rather than the grid:

- `es := int(e.GetStartMinute())` — exact start
- `ee := es + int(e.DurationMinute)` — exact end

The two `fmt.Sprintf` calls at lines 123 and 125 swap `sn`/`se` for `es`/`ee`. The format string, the ID prefix branch, and the subsequent `wrapLabel` call are untouched, so truncation and wrapping behavior carry over unchanged (FR-007).

Note the label width can now differ from before by at most zero characters — both forms render as exactly `HH:MM-HH:MM`, so no wrapping or truncation boundary shifts. This is what makes SC-005 hold trivially for the label path.

### Defect 2 — conflict detection (`internal/tui/plan_preview.go`)

`planPreviewConflicts` currently iterates the preview's slots and asks whether each *slot* overlaps another entry. That inflates the preview to slot granularity before comparing, which is what manufactures the false positive: a preview ending at 13:50 is treated as occupying all of 13:45–14:00, so it collides with an entry starting at 13:50.

The corrected shape, per other entry:

1. Compute the exact overlap region: `ovStart = max(pStart, oStart)`, `ovEnd = min(pEnd, oEnd)`.
2. If `ovStart >= ovEnd`, there is no overlap — skip. This is what makes touching boundaries safe, since half-open intervals give `ovStart == ovEnd` exactly at a shared boundary.
3. Otherwise mark every slot from `snapDown15(ovStart)` while `t < ovEnd`, stepping 15.

The artificial "empty duration counts as 15 minutes" clamps on both `pEnd` and `oEnd` are dropped — see research.md for that decision. `snapDown15` stays; it is still needed to map the overlap region onto slot keys.

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.
