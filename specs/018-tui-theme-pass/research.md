# Phase 0 Research: TUI Theme Pass

All Technical Context items are resolved (no `NEEDS CLARIFICATION` remain). The
feature is presentation-only against an existing, well-understood codebase, so
research focuses on the few design decisions the implementation must commit to.

## Decision 1 — Layout: lipgloss bordered boxes vs. hand-rolled stitching

**Decision**: Render each pane as a `lipgloss.NewStyle().Border(lipgloss.RoundedBorder())`
box with a title, then combine the two panes with `lipgloss.JoinHorizontal`.
Compute inner content dimensions from the outer pane size and set them via the
box style (`.Width()` / `.Height()`), then truncate names and wrap descriptions
to that **inner** width.

**Rationale**: The current `viewList`/`viewWithForm`/`viewWithMove` functions
each re-implement the same `splitLines` + `padRightAnsi` + `fmt.Sprintf("%s %s")`
column stitching. lipgloss's border + join primitives do this correctly
(including ANSI-aware width) and let us drop the manual stitching. This *reduces*
code (Constitution I). Borders also give us the framing + titles the spec
requires for free.

**Alternatives considered**:
- *Keep manual stitching, draw box-drawing chars by hand* — rejected: more code,
  error-prone width math, exactly what we're trying to remove.
- *bubbles `viewport`/full layout framework* — rejected: YAGNI; lipgloss boxes
  are sufficient and lighter.

**Sizing rules** (from design doc, made precise):
- Two panes side by side. List outer width `= width/2`; second pane outer width
  `= width - listWidth` (JoinHorizontal needs no manual 1-col gap, but a single
  space gap may be kept between boxes if desired — see Decision 6).
- Border consumes 1 column each side and 1 row top+bottom.
- Inner content width = outer pane width − 2 (borders) − horizontal padding.
- Inner content height = `height − 1 (status/footer) − 2 (border)`.
- Clamp inner width to ≥ 0 and inner height to ≥ 1 so degenerate sizes never
  panic (preserves current `maxLines < 1 → 1` guard intent).

## Decision 2 — Fold and completion are separate glyphs

**Decision**: Replace the single `marker` (`[-]`/`[+]`) with two independent
pieces rendered as `{treePrefix}{chevron-or-space} {checkbox} {name}`:
- **Chevron**: `▾` (expanded) / `▸` (collapsed), shown **only** when the row has
  expandable children — i.e. exactly the condition that currently yields `[+]`
  (`hasExpandableChildren`). Leaves and fully-expanded rows render a single space
  placeholder so columns stay aligned.
- **Checkbox**: `☐` (open) / `☑` (done) on **every** row, reflecting
  `Task.GetCompletedAt() != nil`.

**Rationale**: The current `[-]`/`[+]` is purely a fold indicator and completion
is conveyed only by strikethrough — two states crammed into one signal. The spec
(FR-004, FR-005) requires both shown honestly. `space` already toggles
completion, so a checkbox is the truthful affordance.

**Implementation note**: `tree.go`'s `emitNode` already computes `isExpanded` and
`hasExpandableChildren`. Expose these to the renderer rather than baking a glyph
string into the row. Cleanest: replace `visibleRow.marker string` with
structured fields the renderer maps to glyphs, e.g.:
- `expandable bool` (has hidden/collapsible children → chevron shown)
- `expanded   bool` (chevron direction)
The completion glyph derives from `node.Task` directly in the renderer. The
expandable-children computation itself is unchanged; only its **output
representation** changes (glyph decided at render time, gated by `styled`).

**Alternatives considered**:
- *Keep `marker string`, just swap the characters* — rejected: still conflates
  fold + completion and forces the unstyled/ASCII fallback to live in `tree.go`.
- *Single combined glyph* — rejected: violates the "two independent states"
  requirement.

## Decision 3 — Unstyled (non-TTY) fallback for new glyphs

**Decision**: The unicode glyphs (`▾ ▸ ☐ ☑ ╭─╮ ▎`) and all color/border styling
appear **only** when `styled == true`. When `styled == false`, output stays the
current plain ASCII form: no borders, no chevrons/checkboxes, no ANSI. The
existing `splitLines`/`padRightAnsi` plain layout (or an equivalently plain
join) is used for the unstyled path.

**Rationale**: FR-012 and SC-003 require non-terminal output to remain plain, and
existing tests (`TestRenderList_NoStrikethroughWhenUnstyled`) assert it. Decorative
unicode in piped output would break scripts and golden tests. The `styled` flag
already exists on `Model` and is the right gate.

**Open implementation choice (low-risk)**: whether the unstyled path keeps the
literal `[-]`/`[+]` marker or shows a plain completion token. The current tests
only assert *absence* of ANSI, not the marker text, so the safest default is to
**preserve today's unstyled output exactly** (keep `[-]`/`[+]`, name only,
strikethrough suppressed). This guarantees SC-003. Tasks will lock this with a
test.

## Decision 4 — Centralized palette (`theme.go`) with AdaptiveColor

**Decision**: One new file `theme.go` defines semantic styles using
`lipgloss.AdaptiveColor{Light:..., Dark:...}` for: `accent`, `border`,
`borderActive` (= accent), `dim`, `completed` (dim green), `errorColor`,
`cursorBar`, `cursorBg`. Existing `highlightStyle` and `errorStyle` (today in
`view.go`) move here and reference the palette. The `help.Model` styling
currently in `help.go` already uses `AdaptiveColor` and stays compatible (it may
optionally reference `dim`).

**Rationale**: FR-009/FR-010/SC-005 require legibility on light *and* dark
terminals and one coherent accent. `AdaptiveColor` is lipgloss's built-in
mechanism for this and is already used in `help.go`, so it's the established
project pattern. Centralizing avoids the current ad-hoc hardcoded `Color("4")` /
`Color("9")`.

**Alternatives considered**:
- *`CompleteColor`/`CompleteAdaptiveColor` (explicit ANSI/256/TrueColor)* —
  rejected: YAGNI; AdaptiveColor is enough and simpler.
- *Keep styles inline per file* — rejected: defeats the "consistent visual
  language" goal and duplicates color literals.

## Decision 5 — Softened cursor

**Decision**: Replace the full-width solid blue bar (`highlightStyle` =
`Background(Color("4")).Foreground(Color("15")).Bold(true)`) with a left accent
bar `▎` (in `cursorBar`/accent) on the cursor row plus a subtle background tint
(`cursorBg`) and bold text. Implemented as a style applied to the cursor row's
inner content within the list box.

**Rationale**: Spec FR-011 / design "softened cursor". The accent bar + gentle
tint reads as selection without the heavy fill dominating the pane.

**Risk**: existing tests assume a specific highlight. `view_test.go` strikethrough
tests don't assert the cursor background, but the cursor row is `m.cursor == 0`
by default in some tests — verify the strikethrough substring still appears on
the cursor row (it should, since the name styling is independent of the cursor
style). Tasks include a cursor-rendering test.

## Decision 6 — Pane join gap and titles

**Decision**: Use `lipgloss.JoinHorizontal(lipgloss.Top, listBox, otherBox)`.
Keep a one-space visual separation between the two boxes (either a literal space
column or right-margin on the left box) to match the design mockup's
`╮ ╭` spacing. Titles (`Tasks` / `Details`) are embedded in the top border via a
title style, or rendered as a styled first inner line if border-title embedding
proves fiddly — either satisfies "a title that identifies it" (FR-001). Prefer
border-embedded titles for fidelity to the mockup.

**Rationale**: Matches the design mockup; JoinHorizontal handles per-line height
padding that the manual loop did by hand.

## Decision 7 — `renderDetails` styled gate

**Decision**: Thread the `styled` bool into the detail renderer (change
`renderDetails(task, width)` → `renderDetails(task, width, styled)`, and update
`ExportRenderDetails` + callers). Header bold-accent and dimmed labels apply only
when `styled`.

**Rationale**: FR-007 wants a styled header/labels, but FR-012/SC-003 require
plain unstyled output. `renderDetails` currently has no styled awareness; the
existing `TestRenderDetails_CompletedTaskNoStrikethrough` calls it directly and
asserts no strike/dim — that test stays green because the header uses bold+accent
(not dim/strike) and, in the unstyled path, no codes at all. The signature change
is small and keeps the gate honest.

**Alternatives considered**:
- *Always style the header regardless of TTY* — rejected: breaks the plain-output
  contract and the spirit of existing tests.

## Summary of file-level impact

| File | Change |
|------|--------|
| `theme.go` (new) | Semantic AdaptiveColor palette + shared styles; absorbs `highlightStyle`, `errorStyle`. |
| `tree.go` | `visibleRow` exposes fold state (`expandable`/`expanded`) instead of pre-baked `marker`; logic unchanged. |
| `view.go` | Bordered boxes + `JoinHorizontal`; new row layout (chevron+checkbox); softened cursor; footer; drop manual stitching for styled path. |
| `details.go` | Styled header + dimmed aligned labels; add `styled` param. |
| `help.go` | Unchanged or minor palette reference; footer tint applied via help styles in `view.go`/`theme.go`. |
| tests | Add glyph-mapping, border-width, cursor, footer, and detail-alignment tests; keep unstyled assertions green. |
