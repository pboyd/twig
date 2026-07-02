# Phase 0 Research: Unscheduled Tasks as a List

The Technical Context has no `NEEDS CLARIFICATION` markers (the three open UX decisions were
resolved in `/speckit-clarify`, recorded in spec.md → Clarifications). Research here documents the
codebase facts and design decisions that shape the implementation.

## Decision 1 — Rewrite `RenderUntimed` rather than add a parallel renderer

**Decision**: Replace the box-drawing body of `internal/cli/plan_grid.go::RenderUntimed` with
single-row, checkbox-prefixed output. Keep the function signature
`RenderUntimed(entries, width, isTTY, opts GridOptions) string` unchanged.

**Rationale**: All three TUI render paths (`renderPlanGrid`, `renderPlanGridContent`, and the styled
two-pane path in `plan_view.go`) plus the `twig plan` CLI (`plan.go`) already call this one function
and already pass the `GridOptions` needed for selection highlighting. Rewriting in place means zero
changes to callers and preserves the height accounting (`strings.Count(untimedStr, "\n")`). Adding a
second renderer would duplicate call-site wiring and risk the two drifting (violates Principle I).

**Alternatives considered**:
- *New `RenderUntimedList` + branch at each call site*: more code, four call sites to touch, and a
  dead code path (the old box renderer) to delete or maintain. Rejected.
- *Render untimed tasks through the Tasks-tab `renderList` in `tui/view.go`*: that method is bound to
  `Model` and the task-tree (`m.visible`, tree prefixes, chevrons, snooze), not plan entries. Reusing
  it would require adapting plan entries into `visibleRow`s — more coupling than value. We instead
  mirror its *row format* (checkbox + name) in `RenderUntimed`. Rejected.

## Decision 2 — Each untimed task is exactly one row (drop duration→rows geometry)

**Decision**: Emit one line per untimed entry. Ignore `DurationMinute` for untimed entries (no
multi-row boxes, no shared `┣┫` borders).

**Rationale**: A list row communicates "no reserved block," which is the entire point of the feature.
Duration is meaningless for an item you do "in any 5-minute gap." One row per task also *frees*
vertical space compared to today's box rendering, which helps the "list grows, grid shrinks" model.

**Alternatives considered**: Preserve duration as a trailing hint (e.g. `☐ task (15m)`) — rejected as
scope creep; the spec says presentation of unscheduled tasks as quick, block-less items.

## Decision 3 — Row format mirrors the Tasks tab (`tui/view.go::renderList`)

**Decision**: Styled (TTY) row = `"  " + checkbox + " " + name`, where `checkbox` is `☑` when
`entry.Completed` else `☐`. Completed rows get the existing strike/dim treatment
(`applyCompletion` / `cli.Strike`). The selected row is highlighted via the existing
`opts.SelectedID` + `opts.SelectionStyle` mechanism (falling back to `applySelection` when
`SelectionStyle` is nil), matching how the box renderer highlights today. No `[id]` prefix is shown
(clarified: "Checkbox + name"). Long names are truncated to the available width, consistent with
`renderList`.

**Rationale**: Principle III (UI/UX Consistency) — the untimed list should read like the Tasks tab.
The Tasks tab uses `☐`/`☑` from the shared theme; reusing those glyphs avoids introducing new visual
language. Selection/completion helpers already exist in `plan_grid.go`, so no new styling code.

**Non-TTY / plain mode**: mirror `renderList`'s plain branch — no checkbox glyph, just the name
(completion still conveyed via `cli.Strike` which no-ops when unstyled). Keeps `twig plan | cat`
output clean and stable.

**Alternatives considered**: Keep `[id]` prefix for CLI scriptability — rejected per clarification
(Checkbox + name, no visible ID); plan-entry IDs are still available via the details pane and
`twig plan` structured paths if needed elsewhere.

## Decision 4 — The shared renderer also updates `twig plan` CLI output (intentional)

**Decision**: Let the `twig plan` CLI command inherit the new list rendering (it calls the same
`RenderUntimed`). Do not special-case the CLI back to boxes.

**Rationale**: The spec's motivation (unscheduled tasks are quick, block-less items) applies equally
to the CLI surface, and Principle III favors keeping the TUI Plan tab and `twig plan` visually
coherent. FR-009 protects only the *web app*; it is silent on the CLI. Special-casing the CLI would
add branching complexity (Principle I) for no user benefit.

**Alternatives considered**: Gate the list style behind `isTTY` so only the TUI changes — rejected;
it splits one surface's semantics on a rendering flag and leaves `twig plan` inconsistent with the
TUI it mirrors.

## Decision 5 — Separator and height model are already correct; leave them

**Decision**: No change to `RenderUntimedSeparator` (already emits a divider-only line ⇒ satisfies
FR-003 "divider line only") or to the height allocation in `plan_view.go` (already computes
`availableRows = height - untimedLines` and trims the combined output to the pane height ⇒ satisfies
FR-008 "list grows, grid shrinks").

**Rationale**: The two clarified behaviors (divider-only separation, list-grows/grid-shrinks) match
what the code already does. Verifying and *not* changing them is the simplest path and de-risks the
"no-unscheduled day is byte-identical" requirement (FR-006), since those paths are untouched.

**Verification note**: Confirm empty-`untimed` still returns `""` from `RenderUntimed` (early return
at top) and empty-`untimedIDs` still returns `""` from the separator, so FR-006 holds by construction.
