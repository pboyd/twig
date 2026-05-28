# TUI Theme Pass — Design

Date: 2026-05-28

## Goal

Refresh the look of the interactive TUI (`internal/tui`) so the panels are
clearly defined and the whole app shares one consistent visual language. Today
the two panes (task list, details) are split by a single space with no framing,
the fold/completion state is conveyed by flat `[-]`/`[+]` markers and
strikethrough, and color is ad hoc (one hardcoded highlight, one error style).

## Scope

Full theme pass: borders, glyphs, detail-pane polish, footer, a centralized
palette, and a softened cursor. Behavior is unchanged — this is purely
presentation. The existing `styled` flag (false when stdout is not a TTY)
continues to gate all styling; unstyled output stays plain.

## Visual direction

Refined rounded: rounded borders, a single blue accent reused for the active
pane border and the cursor, dimmed secondary text. The accent does the work
rather than heavy background fills.

```
╭─ Tasks ─────────────────────────╮ ╭─ Details ───────────╮
│ ▾ ☐ build an enchanting room    │ │ mine deep for       │
│ ├─▾ ☐ add bookcases             │ │ diamonds            │
│ │ ├─ ☐ collect leather          │ │                     │
│ │ └─ ☑ collect paper            │ │ ID    7             │
│ └─▾ ☐ add enchanting table      │ │ Est   4 pomodoros   │
│▎    └─ ☐ mine deep for diamonds │ │                     │
│ ☐ Bar                           │ │                     │
│ ☐ Baz                           │ │                     │
╰─────────────────────────────────╯ ╰─────────────────────╯
 ↑/k ↓/j move · e edit · space done · ? help · q quit
```

## Decisions

### Glyphs: fold and completion are separate states

The current `[-]`/`[+]` marker is a **fold** indicator (`-` expanded/leaf, `+`
collapsed with hidden children), not a checkbox. Completion is shown only by
strikethrough today. These are two independent states and the redesign shows
both:

- **Fold:** chevron `▾` (expanded) / `▸` (collapsed), rendered only on rows that
  have expandable children (the same condition that produces `[+]` in
  `tree.go`). Leaves and fully-expanded rows show no chevron.
- **Completion:** a checkbox `☐` (open) / `☑` (done) on every row. `space`
  already toggles completion, so the checkbox is the honest affordance for it.
  Completed rows additionally render dim-green + strikethrough on the name.

Row layout: `{treePrefix}{chevron-or-space} {checkbox} {name}`.

### Borders and layout

Replace the hand-rolled column stitching (`splitLines` + `padRightAnsi` loops in
`view.go`) with lipgloss bordered boxes joined by `lipgloss.JoinHorizontal`.
Each pane is a `RoundedBorder()` box with a title (`Tasks` / `Details`). The
focused pane's border uses the accent color; the inactive pane uses dim gray.

Sizing: the border consumes one column on each side and one row top and bottom.
Inner content width per pane is the outer pane width minus 2 (borders) minus any
horizontal padding; inner content height is `height − 1 (status) − 2 (border)`.
Width/height are routed through the box style so name truncation and description
wrapping use the inner width, not the outer.

This bordered treatment applies to all three list-mode layouts so they stay
consistent: the default list view, the edit/new-task form view
(`viewWithForm`), and the move-task view (`viewWithMove`).

### Detail pane

- Task name promoted to a bold accent header at the top of the pane.
- `ID / Due / Est / Completed` labels dimmed and column-aligned.
- Description wrapped to the inner width below the labels.
- Completed timestamp in dim green.

### Footer

The help line becomes a subtle full-width footer with a faint background tint,
anchored at the bottom, styled through the bubbles `help` model's style fields
(`help.Styles`). Error messages continue to use the error color.

### Centralized palette (`theme.go`)

One file defines semantic colors as `lipgloss.AdaptiveColor` so light terminals
stay legible:

| Name           | Role                                   |
|----------------|----------------------------------------|
| `accent`       | active border, headers, cursor bar     |
| `border`       | inactive pane border                   |
| `borderActive` | focused pane border (= accent)         |
| `dim`          | labels, tree connectors, footer text   |
| `completed`    | done tasks (dim green)                 |
| `error`        | error messages                         |
| `cursorBar`    | left accent bar `▎` on the cursor row  |
| `cursorBg`     | subtle cursor row background tint       |

All existing styles (`highlightStyle`, `errorStyle`) move here and reference the
palette. The cursor highlight softens from a full-width solid blue bar to a left
accent bar `▎` plus a gentle background tint and bold text.

## Architecture

No new packages. Changes are confined to `internal/tui`:

- `theme.go` (new) — palette + shared styles.
- `view.go` — bordered boxes via lipgloss, `JoinHorizontal`, new row rendering
  (chevron + checkbox), softened cursor, footer styling. Removes the manual
  `splitLines`/`padRightAnsi` stitching where lipgloss now handles layout.
- `details.go` — styled header + dimmed aligned labels.
- `tree.go` — expose fold state to the renderer (chevron vs. the old `[-]`/`[+]`
  string). The expandable-children computation is unchanged; only its glyph
  output changes.

Data flow is unchanged: `Model.View()` → per-mode view fn → pane renderers.

## Error handling

Presentation-only; no new failure modes. Degenerate terminal sizes (very small
width/height) clamp inner dimensions to ≥ 0 / ≥ 1 as the current code already
does, so boxes never panic on negative sizes.

## Testing

- Keep existing unstyled-path assertions green (e.g.
  `TestRenderList_NoStrikethroughWhenUnstyled`) — plain output stays plain.
- Add tests for the glyph mapping: chevron present only on rows with expandable
  children; checkbox reflects completion state.
- Add a test that bordered output preserves the expected visible width
  (`lipgloss.Width`) per pane.
- Detail-pane test for header + label alignment in the unstyled path.
