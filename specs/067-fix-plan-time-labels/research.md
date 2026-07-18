# Phase 0 Research: Show Actual Times in Plan Grid Labels

The Technical Context contained no NEEDS CLARIFICATION markers — this is a defect fix inside an existing, well-understood rendering path, with no new technology to evaluate. Research therefore focused on pinning down the exact defect sites and resolving the four judgment calls the fix requires.

## Defect localization

**Finding**: Only two sites in the codebase reuse snapped geometry values for non-geometry purposes.

A sweep for `snapDown15`/`snapUp15` across all non-test Go files returns nine call sites. Seven are legitimately geometric and must not change:

| Site | Purpose | Verdict |
|---|---|---|
| `internal/cli/plan_grid.go:64-65` | window expansion | keep |
| `internal/cli/plan_grid.go:101-102` | row placement (`tl`/`bl`) | keep |
| `internal/cli/plan_grid.go:156` | "now" marker block | keep |
| `internal/cli/plan_grid.go:244` | hour-gutter labels | keep (FR-008 — labels the grid, not an entry) |
| `internal/cli/plan_grid.go:702-703` | `baseWindowFor` window sizing | keep |
| `internal/cli/plan_grid.go:741` | `GridWindow` anchor block | keep |
| `internal/cli/plan_grid.go:123, 125` | **entry label text** | **fix** |
| `internal/tui/plan_preview.go:133` | **conflict slot iteration** | **fix** |

**Rationale**: This confirms the spec's framing that box geometry is correct and only two consumers are wrong. It also bounds the blast radius — no other surface reads snapped values for display.

**Alternatives considered**: Grepping for time formatting independently found only `%02d:%02d-%02d:%02d` at lines 123 and 125 plus the gutter's `%02d:%02d` at 244, cross-confirming there is no third label path (e.g. in `RenderUntimed`, which handles entries with no start time at all).

## Decision 1 — Where to derive the exact end time

**Decision**: Compute the label's end as `startMinute + durationMinute` inline at the label site.

**Rationale**: The entry already carries both fields; the sum is the definition of its end. Computing it inline keeps the change to two lines and matches how the surrounding code already derives `se`.

**Alternatives considered**: Adding an `exactSpan(e) (start, end int)` helper, or an `interval` struct pairing exact and snapped forms. Both were rejected under Principle I — a helper used twice in one loop body is indirection without payoff, and the struct would ripple through `entryLayout` for no behavioral gain.

## Decision 2 — Zero or negative duration in labels

**Decision**: Render exactly what the data says. An entry with duration 0 starting at 13:05 labels as `13:05-13:05`, while its box still occupies one full slot via the existing `if se < sn+15` clamp.

**Rationale**: The clamp at `plan_grid.go:103-105` exists so a short entry stays visible — a geometry concern. Applying it to the label would reintroduce the exact bug being fixed, just at a different threshold. Honest output is the point of the feature.

**Alternatives considered**: Suppressing the range and showing only a start time for zero-duration entries. Rejected: it adds a branch and a second label format for a state the server's 30-minute default makes rare, violating both Principle I and Principle III's single-format consistency.

## Decision 3 — The artificial 15-minute minimum in conflict math

**Decision**: Drop the `if pEnd <= pStart { pEnd = pStart + 15 }` and matching `oEnd` clamps. Conflict detection uses exact intervals with no minimum.

**Rationale**: SC-003 and SC-004 define a conflict as an overlap of at least one minute. A zero-length interval overlaps nothing by that definition, so inflating it to 15 minutes can only manufacture the same class of false positive this feature exists to remove. The clamps were a defensive holdover from slot-based reasoning, and slot-based reasoning is exactly what is being replaced.

**Consequence accepted**: A zero-duration preview draws a one-slot box but reports no conflict. The geometry minimum and the conflict semantics diverge — which is correct, since one is about visibility and the other about truth.

**Alternatives considered**: Retaining the clamps for safety. Rejected: "safety" here means preferring a false warning to none, and a conflict indicator that cries wolf is worse than one that stays quiet, per User Story 2's rationale.

## Decision 4 — Which slots to mark for a genuine conflict

**Decision**: Mark the slots spanned by the *overlap region* — `snapDown15(max(pStart, oStart))` up to `min(pEnd, oEnd)` — rather than all slots the preview occupies.

**Rationale**: This is the most informative marking available at slot resolution: it shows *where* the collision is, not merely that the preview collides somewhere. It also preserves the renderer's contract, since `internal/cli/plan_grid.go:193` and `:215` index `PreviewConflictSlots` by slot-start minute and are unchanged.

The marked slots are always a subset of the preview's own snapped box: the overlap region lies inside the preview's interval, so snapping its start down cannot escape the box's top row. This matters because the renderer only applies `ConflictStyle` to preview rows — a slot key outside the preview's box would be silently dropped.

**Alternatives considered**:
- *Marking every preview slot when any conflict exists.* Rejected — it loses the location of the collision and re-inflates the preview, the same mistake in a new form.
- *Sub-slot markers (e.g. partial-row shading).* Rejected — a grid row is 15 minutes and is the finest visual unit available; this was recorded as an accepted assumption in the spec rather than a limitation to engineer around.

## Regression surface

**Finding**: The two existing conflict tests (`TestPlanPreviewConflicts_Overlap`, `TestPlanPreviewConflicts_Touching`) use 15-minute-aligned times exclusively, so both remain valid and passing under the new logic — the overlap test's expected slots 630 and 645 are exactly the overlap region 10:30–11:00.

**Implication**: The existing suite cannot detect either defect, which is precisely why they shipped. New tests must use deliberately non-aligned times (the user's 13:50 and 14:10) and a sub-slot case. This is recorded as the primary test-design constraint for Phase 2.
