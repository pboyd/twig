# Contract: `cli.GridOptions` selection styling (additive)

**Boundary**: between the TUI (`internal/tui`) and the shared calendar-grid renderer (`internal/cli/plan_grid.go`, function `RenderGrid`). This is the only cross-package contract this feature changes. It is **additive and backward-compatible**: the plain `todo plan` CLI is unaffected.

## Before

```go
type GridOptions struct {
    HideID     bool  // omit the "[id] " prefix from entry labels
    SelectedID int32 // highlight this entry's rows (0 = none); applied only when isTTY
    Styled     bool  // when true (and isTTY): accent-color highlight instead of plain bold
}
```

Selection behavior (current): when `SelectedID` matches an entry, **every rendered line of that entry** — the hour gutter (`07:00`), the `│`/`├┤` rails, the box border (`┏┓┗┛┣┫┃`), and the label — is wrapped in a single ANSI run: bright-blue **foreground** (`\x1b[94m`) when `Styled`, else bold (`\x1b[1m`).

## After

```go
type GridOptions struct {
    HideID         bool
    SelectedID     int32
    Styled         bool
    SelectionStyle func(string) string // NEW: styles the selected entry's content cell.
                                        // nil = fall back to prior plain behavior.
}
```

### Semantics

1. **Scope = content cell only.** When an entry is selected and `isTTY`, the selection styling is applied **only to the entry's text content**, never to:
   - the hour gutter / now-marker column (`07:00`, `▶`),
   - the slot rails (`│`, `├`, `┤`),
   - the entry's own box-drawing border (`┏ ┓ ┗ ┛ ┣ ┫ ┃` and the heavy `━` corner/edge fills that form the border line).
2. **Multi-row entries**: only the **interior label rows** (`┃<content>┃`) are styled — the `<content>` between the walls. The top (`┏━┓`), bottom (`┗━┛`), and shared (`┣━┫`) border rows of a selected entry receive **no** selection styling.
3. **Single-row entries** (`┣<label━━━>┫`): the inner field (label plus its heavy-fill padding to the field width) is styled as the cell; the enclosing `┣`/`┫` are not.
4. **Styler source**:
   - When `SelectionStyle != nil`, the cell content is rendered with `SelectionStyle(content)`. The TUI passes `highlightStyle.Render` (Lip Gloss: `Bold(true).Background(accent).Foreground("15")`), making the selected cell **bold white text on the blue accent background**, identical to a selected Tasks-tab row.
   - When `SelectionStyle == nil`, behavior falls back to the prior plain styling (plain bold / `\x1b[94m`) **but still scoped to the content cell** — though in practice the only nil-styler caller (`todo plan`) passes `SelectedID == 0`, so no selection is drawn.
5. **Width/layout invariant**: applying the styler MUST NOT change the printed width of any row. Background/bold/fg codes wrap the already-padded content; the cell occupies the same column span as before so the grid stays aligned. (The styler is applied to the padded content string, not used to re-pad.)
6. **Non-TTY / non-styled**: when `!isTTY`, no selection styling is emitted (unchanged). When `isTTY && !Styled`, the cell is styled via the fallback (plain bold) on the content only.

### Backward compatibility

- Existing callers that do not set `SelectionStyle` compile unchanged (zero value `nil`).
- `todo plan` output (no selection) is byte-for-byte unchanged.
- `Styled` is retained; `SelectionStyle` supersedes the old whole-line accent path for the TUI.

### Test obligations (`internal/cli`)

- Selected single-row entry: content cell carries the styler's codes; the `┣`/`┫` and gutter do not.
- Selected multi-row entry: interior content row is styled; `┏━┓`/`┗━┛` border rows and the gutter are not.
- `SelectedID == 0`: output identical to pre-change (golden).
- Row widths identical with and without selection (alignment preserved).
