# Quickstart: TUI Theme Pass

How to build, run, and verify the TUI theme pass during and after implementation.

## Build & run

```bash
cd services/todo
go build -o todo ./cmd/todo

# Launch the interactive TUI (requires a TTY + a running server + auth env).
# See CLAUDE.md for `make dev` and TODO_API_KEY / TODO_ADDR setup.
./todo
```

Visual checklist when running on a TTY:
- Task list and details appear as two **rounded-bordered** boxes with `Tasks` /
  `Details` titles.
- The focused pane's border is the blue accent; the other is dim gray.
- Rows show a chevron (`▾`/`▸`) **only** on rows with collapsible children, and a
  checkbox (`☐`/`☑`) on every row.
- Completed tasks: dim-green + strikethrough name.
- Cursor row: left accent bar `▎` + subtle tint, not a solid blue fill.
- Bottom: a full-width footer help line.
- Resize the terminal very small — it must not panic.

Verify the **unstyled** path stays plain:

```bash
# Piping (non-TTY) must produce plain output: no ANSI, no box/glyph runes.
./todo | cat | head
```

## Test

```bash
cd services/todo
go test ./internal/tui/...        # package under change
go test ./...                     # full suite before opening a PR
```

Key tests to keep green / add (see contract for the authoritative list):
- `TestRenderList_NoStrikethroughWhenUnstyled` — plain output unaffected.
- `TestRenderDetails_CompletedTaskNoStrikethrough` — detail header has no
  strike/dim; `Completed:` line present.
- NEW: chevron present **iff** row is expandable.
- NEW: checkbox glyph matches completion state on every styled row.
- NEW: `lipgloss.Width` of each pane equals its allotted width.
- NEW: detail header + label alignment (unstyled path).
- NEW: unstyled path emits no decorative glyph runes.

## Files you'll touch

| File | What |
|------|------|
| `internal/tui/theme.go` (new) | Semantic AdaptiveColor palette + shared styles. |
| `internal/tui/tree.go` | Expose `expandable` / `expanded` on `visibleRow`. |
| `internal/tui/view.go` | Bordered boxes via lipgloss + `JoinHorizontal`; row glyphs; cursor; footer. |
| `internal/tui/details.go` | Styled header + dimmed aligned labels; `styled` param. |
| `internal/tui/export_test.go` | Update `ExportRenderDetails` signature for the new `styled` param. |

## Definition of done

- All acceptance scenarios in `spec.md` demonstrable in a running TTY session.
- Success criteria SC-001…SC-006 hold (focus obvious, glyphs honest, plain
  output unchanged, width fidelity, light/dark legibility, no panic).
- `go test ./...` green; no placeholder tokens in spec/plan/contracts.
