# Contract: Pomodoro row rendering (visual)

The TUI visual contract for the shared renderer. This is the testable spec for
`renderPomodoroRow(estimate, completed int, styled bool) string` in
`internal/tui/pomodoro_row.go`.

## Glyph

The tomato emoji `🍅` (the existing pomodoro glyph). No other characters are
emitted by the row itself (callers add surrounding labels/newlines).

## Counts

```
n      = max(estimate, completed)
done   = min(completed, estimate)   // styled: bold red (pomodoroDone)
remain = estimate - done            // styled: dim (existing dim token)
over   = max(0, completed - estimate) // styled: bold yellow (pomodoroOver)
```

The row is exactly `n` glyphs, ordered: `done` red, then `remain` dim, then
`over` yellow.

## Empty state

`estimate == 0 && completed == 0` → return the empty string `""` (caller omits
the line entirely). Callers MUST NOT print a label when the row is empty.

## Styled vs. plain

| Mode | Behavior |
|------|----------|
| `styled == true` | Each segment wrapped in its Lipgloss style: red+bold, dim, yellow+bold. |
| `styled == false` | Emit `n` plain `🍅` glyphs, no ANSI. Count is still correct (FR-010). |

## Color tokens

Defined in `internal/tui/theme.go` (no inline colors):
- `pomodoroDone` — red, used **bold** for completed-within-estimate glyphs and the active-pomodoro status glyph.
- `pomodoroOver` — yellow, used **bold** for over-estimate glyphs.
- `dim` — existing token, for estimated-not-done glyphs.

## Active-pomodoro glyph (US3)

In `renderStatus` (`view.go`), the leading `🍅` on the running-pomodoro line(s)
is styled with `pomodoroDone` + bold. Only the glyph is restyled; the timer text
is unchanged. Plain mode keeps the existing non-styled `Pom …` fallback.

## Test matrix (table-driven)

| estimate | completed | styled | expected glyph count | expected segments (red/dim/yellow) |
|---------:|----------:|:------:|---------------------:|------------------------------------|
| 0 | 0 | any | 0 (empty string) | — |
| 5 | 0 | true | 5 | 0 / 5 / 0 |
| 5 | 2 | true | 5 | 2 / 3 / 0 |
| 5 | 5 | true | 5 | 5 / 0 / 0 |
| 5 | 6 | true | 6 | 5 / 0 / 1 |
| 3 | 1 | true | 3 | 1 / 2 / 0 |
| 0 | 2 | true | 2 | 0 / 0 / 2 |
| 5 | 2 | false | 5 | plain (no styling), 5 glyphs |

Styled-segment assertions can count occurrences of the ANSI sequences produced
by each style, following the existing `details_test.go` / `view_test.go`
approach (or assert on a styled/plain split as those tests do).
