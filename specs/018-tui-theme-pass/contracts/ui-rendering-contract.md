# UI Rendering Contract: TUI Theme Pass

This is the contract the `internal/tui` renderer MUST satisfy. There are **no
API/proto/RPC contract changes** in this feature (no message, service, endpoint,
or DB query is touched). The "contract" here is the visible rendering behavior,
expressed as testable invariants so the renderer and its tests stay aligned (per
constitution Quality Gate 1, adapted for a UI-only change).

## 1. Styled vs. unstyled gate

| Condition | Required output |
|-----------|-----------------|
| `styled == true` (TTY) | Borders, titles, chevrons, checkboxes, palette colors, softened cursor, footer tint. |
| `styled == false` (piped/redirected) | Plain text only: **no** ANSI escape codes, **no** decorative unicode glyphs (`▾ ▸ ☐ ☑ ╭ ─ ╮ ╰ ╯ │ ▎`), preserving today's plain output. |

**Test hooks**: assert absence of `\x1b[` and of the glyph runes when
`styled == false`; assert presence when `styled == true`.

## 2. Row layout (styled path)

Each list row renders as:

```
{treePrefix}{chevron-or-space} {checkbox} {name}
```

| Element | Rule |
|---------|------|
| `treePrefix` | Existing connectors (`├── `, `└── `, `│   `), unchanged. |
| chevron | `▾` if `expandable && expanded`; `▸` if `expandable && !expanded`; single space `" "` if `!expandable`. |
| checkbox | `☑` if task completed; `☐` otherwise — present on **every** row. |
| name | Truncated to inner width; completed names get strikethrough + dim-green. |

**Invariants**:
- C2.1 chevron present **iff** the row is `expandable`.
- C2.2 checkbox present on every styled row, glyph matches completion.
- C2.3 completed row name carries strikethrough/dim-green; incomplete does not.

## 3. Panes, borders, titles, focus

- C3.1 List pane and second pane (Details / form / move) are each a
  `RoundedBorder()` box.
- C3.2 Each box has a title identifying it (`Tasks`, `Details`).
- C3.3 The **focused** pane's border uses `accent`; the inactive pane uses
  `border` (dim gray).
- C3.4 Applies to all three list-mode layouts: `viewList`, `viewWithForm`,
  `viewWithMove`.

## 4. Sizing & width fidelity

- C4.1 `lipgloss.Width(pane)` equals the pane's allotted outer width.
- C4.2 Name truncation and description wrapping use **inner** width
  (outer − 2 borders − padding).
- C4.3 Inner height = `height − 1 (footer) − 2 (border)`.
- C4.4 Degenerate sizes: inner width clamps to ≥ 0, inner height to ≥ 1; render
  never panics.

## 5. Detail pane

- C5.1 Task name rendered as a bold `accent` header at the top (styled path).
- C5.2 Labels `ID / Due / Est / Completed` are `dim` and column-aligned.
- C5.3 Description wraps to inner width.
- C5.4 Completed timestamp in `completed` (dim green).
- C5.5 Unstyled path: plain text, no codes (existing
  `TestRenderDetails_CompletedTaskNoStrikethrough` stays green).

## 6. Cursor

- C6.1 Cursor row shows a left accent bar `▎` plus a subtle background tint
  (`cursorBg`) and bold text — not a full-width solid fill.
- C6.2 The cursor style does not suppress the completed-name strikethrough on the
  cursor row.

## 7. Footer

- C7.1 The help/shortcut line renders as a full-width footer with a faint
  background tint, styled via the bubbles `help` model styles.
- C7.2 Error messages use the `error` color and remain clearly distinguished.

## 8. Palette (semantic names — `theme.go`)

All as `lipgloss.AdaptiveColor` (light + dark variants). Renderer MUST reference
these names, not inline color literals.

| Name | Role |
|------|------|
| `accent` | active border, headers, cursor bar |
| `border` | inactive pane border |
| `borderActive` | focused pane border (= accent) |
| `dim` | labels, tree connectors, footer text |
| `completed` | done tasks (dim green) |
| `error` | error messages |
| `cursorBar` | left accent bar `▎` on the cursor row |
| `cursorBg` | subtle cursor row background tint |

- C8.1 Existing `highlightStyle` and `errorStyle` are defined here and reference
  the palette.
- C8.2 Colors are adaptive so styled output is legible on light and dark
  terminals (SC-005).

## 9. Behavior preservation

- C9.1 No keybinding, navigation, selection, edit, move, completion-toggle, or
  fold behavior changes. This contract governs rendering only.
