# Contract: `internal/markdown` package API

**Feature**: 050-tui-markdown-rendering
**Date**: 2026-06-12
**Stability**: internal package; consumed only by `internal/tui`.

Per Constitution Principle II, this is the source-of-truth interface, defined before implementation. The TUI integration depends on these signatures; implementation must conform to them.

## Public surface

```go
package markdown

import "image/color"

// Theme is the palette the renderer styles with. Built by internal/tui from
// theme.go so the renderer never imports internal/tui (no import cycle) and so
// all styling derives from the single shared palette (Principle III).
type Theme struct {
    Accent color.Color // emphasis, links, headings
    Dim    color.Color // blockquote bar, secondary structure
    CodeBg color.Color // inline code / code block background tint
    Text   color.Color // default fg; zero value = terminal default
}

// Options are per-call rendering inputs.
type Options struct {
    Width  int  // available columns; <= 0 disables wrapping
    Styled bool // false => plain output with ZERO escape sequences
}

// Renderer holds the configured parser, theme, and a memoization cache.
// Safe for use from the single TUI goroutine (not concurrency-safe; the TUI
// renders on one goroutine). Construct once and reuse for the session.
type Renderer struct { /* unexported: md, theme, cache */ }

// NewRenderer builds a Renderer with a GFM-configured goldmark parser and the
// given theme.
func NewRenderer(theme Theme) *Renderer

// Render renders multi-line / long-text markdown with full block + inline
// support (headings, lists, tables, blockquotes, code, rules, links).
// Result has no trailing newline. Display-only; never mutates input.
func (r *Renderer) Render(text string, opts Options) string

// RenderInline renders single-line / short-field markdown with INLINE
// constructs only (bold, italic, bold-italic, strikethrough, inline code,
// links). Block syntax is flattened to one line (e.g. a leading "# " or "- "
// is neutralized; FR-008). Output contains no newlines.
func (r *Renderer) RenderInline(text string, opts Options) string
```

## Behavioral contract

### `Render(text, opts)` — long fields (FR-001..FR-007)

1. Parses `text` as GFM. goldmark never errors on malformed input → no panics (FR-012). The walker handles unknown node types by emitting their literal text.
2. Emits each block per the construct→representation table in [data-model.md](../data-model.md).
3. **Wrapping is display-width aware** (uniseg/runewidth), not byte length. Prose/lists/quotes soft-wrap to `opts.Width`. Tables go through lipgloss/v2 `table` sized to `opts.Width`. Code blocks and unbreakable tokens (long URLs) are NOT wrapped and may be clipped by the caller's pane (documented limitation — see plan pushback #2).
4. `opts.Styled == false` → output contains **zero** ANSI escape sequences (SC-005, FR-013); structure uses plain markers (`-`, `>`, `[ ]`, `---`, ASCII table).
5. `opts.Styled == true` → styles come only from `Theme` (Principle III).
6. Links: styled → `ansi.SetHyperlink(url)` wrapping styled text; plain → `text (url)` (FR-007, with detection caveat in research.md).
7. Pure function of `(text, opts, theme)` — same inputs yield identical output (cache-safe). No I/O, no globals.

### `RenderInline(text, opts)` — short fields (FR-008, FR-009)

1. Parses `text`, walks **inline** descendants only; block containers contribute their inline children flattened onto one line.
2. Never emits `\n`. Width is advisory (callers pad/truncate as today via `padRightAnsi`).
3. Same styled/plain rules as `Render`.
4. Applied at every display site for a given field (FR-009): tree rows, list rows, detail headers, pickers.

### Caching

- Both methods consult an internal cache keyed by `(text, width, styled, inline)`. A hit returns the stored string; a miss renders and stores it.
- The cache is bounded (size cap with reset, or small LRU) so a long session cannot grow it without limit. Correctness never depends on the cache (it only affects speed).

## Consumer contract (`internal/tui`)

- `internal/tui` adds a `*markdown.Renderer` field to `Model`, constructed in `NewModel` via `markdown.NewRenderer(buildMarkdownTheme())`, where `buildMarkdownTheme()` (new helper in `theme.go`) maps palette vars → `markdown.Theme`.
- Call sites pass `Options{Width: <pane width>, Styled: m.styled}`.
- Replaces direct use of `wrapDescription` for rendered fields; `wrapDescription` may remain for any non-markdown text or be removed if fully superseded.

## Out of scope (explicit non-goals)

- Syntax highlighting of code blocks (no chroma) — Principle I.
- Rendering images as terminal graphics (sixel/kitty) — alt-text placeholder only.
- Any change to proto, server, DB, or the React web app.
- Concurrency safety — single-goroutine TUI use only.
