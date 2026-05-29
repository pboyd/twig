# Contract: `cli.GridOptions` (additive)

**Type**: `services/todo/internal/cli` → `GridOptions`
**Consumed by**: `cli.RenderGrid` (non-interactive `todo plan` output) and the TUI Planning grid (`internal/tui` via `planGridOptions`).
**Change class**: Additive, backward-compatible. Zero value preserves current behavior.

## Current

```go
type GridOptions struct {
    HideID     bool  // omit "[id] " prefix
    SelectedID int32 // highlight this entry (0 = none); applied only when isTTY
}
```

`applySelection` currently wraps the selected entry's lines in bold (`\x1b[1m … \x1b[0m`). This is the "monotone" appearance the Planning tab inherits.

## New (planned)

Add an optional styling toggle so the TUI can render the grid with the shared theme accent while the plain CLI output stays unchanged:

```go
type GridOptions struct {
    HideID     bool
    SelectedID int32
    Styled     bool // when true (and isTTY): accent-highlight the selected
                    // entry and tint box borders with the accent color
}
```

## Guarantees

1. **CLI unchanged**: When `Styled == false` (the zero value, used by the non-interactive CLI), `RenderGrid` returns exactly the same bytes as before this feature. Verified by the existing `internal/cli` grid tests, which must continue to pass without modification.
2. **TTY gate preserved**: All styling (existing bold selection and the new accent path) is applied only when `isTTY == true`. Non-TTY output remains plain.
3. **TUI opt-in**: The TUI sets `Styled: m.styled` in `planGridOptions()`. When `m.styled` is false (non-ANSI terminal), the grid degrades to plain text with no escape sequences.

## Notes

- The precise styling implementation (e.g. reusing `cursorBg` background for the selected entry to match the Tasks list cursor, and `accent` for box borders) is an implementation detail behind this flag; only the flag and its guarantees are contractual.
- If, during implementation, color/style hook fields prove cleaner than a single `bool`, they MUST remain additive with a zero value equal to today's behavior, and this contract MUST be updated before the implementation lands (Constitution Principle II).
