# UI Render Contract: Untimed Task List

**Surface**: TUI Plan tab + `twig plan` CLI (both call `internal/cli/plan_grid.go::RenderUntimed`).

**Scope**: Defines the exact output of `RenderUntimed` for untimed plan entries after this feature.
The web app is out of scope (FR-009). This is the source-of-truth contract that
`internal/cli/plan_grid_test.go` and `internal/tui/plan_view_test.go` assert against.

## Function signature (unchanged)

```go
func RenderUntimed(entries []*planv1.PlanEntry, width int, isTTY bool, opts GridOptions) string
```

## C1 — Empty input

- `len(entries) == 0` ⇒ returns `""` (empty string), no trailing newline. *(Unchanged; guarantees
  FR-006: a no-unscheduled day renders byte-identical to today.)*

## C2 — One line per entry

- Exactly one `"\n"`-terminated line is emitted per entry, in the order given.
- `DurationMinute` MUST NOT affect the output (no multi-row boxes, no shared borders).

## C3 — Styled (TTY) row format

For each entry, the base row is:

```
"  " + checkbox + " " + name
```

- `checkbox` = `"☑"` when `entry.Completed` is true, else `"☐"`.
- `name` = `entry.Name` rendered inline and truncated so the full line fits within `width`
  (reserve the width of the `"  " + checkbox + " "` prefix). Truncation matches the Tasks tab
  (`tui/view.go::renderList`) behavior.
- When `entry.Completed`, `name` is rendered with the completed treatment (dim + strikethrough),
  consistent with the box renderer's `applyCompletion` and the Tasks tab.

### C3.1 — Selection highlight

- An entry is *selected* when `isTTY && opts.SelectedID != 0 && entry.Id == opts.SelectedID`.
- When selected and `opts.SelectionStyle != nil`, the row's content is wrapped with
  `opts.SelectionStyle(...)` (same mechanism the box renderer uses).
- When selected and `opts.SelectionStyle == nil`, the plain fallback `applySelection` is applied.
- Non-selected rows receive no selection styling.

## C4 — Plain (non-TTY) row format

- No checkbox glyph is emitted (mirrors `renderList`'s plain branch): row is the name with leading
  indentation only.
- Completion is still conveyed via `applyCompletion` (which calls `cli.DimStrike`), a no-op when
  unstyled — so plain output stays ASCII-clean for piping (`twig plan | cat`).
- No selection styling in plain mode.

## C5 — No visible task ID

- The `[id]` prefix used by the old box renderer MUST NOT appear (clarified: "Checkbox + name").
- `opts.HideID` becomes irrelevant to untimed rendering; the ID is never shown regardless.

## C6 — Interaction invariance (FR-005)

- `RenderUntimed` is pure rendering. It MUST NOT change selection, completion, scheduling,
  reordering, or details behavior. Those are driven by entry `Id`, which is unchanged.

## Illustrative output (styled, width sufficient)

```
  ☐ schedule doctor appointment
  ☑ call dentist
  ☐ renew library card
```

Followed (by the caller) by the existing divider from `RenderUntimedSeparator`, then the grid:

```
  ☐ schedule doctor appointment
  ☑ call dentist
 ├──────────────────────────────┤
 08:00 ┃ ...grid...
```
