# Data Model: Markdown Rendering in TUI Text Fields

**Feature**: 050-tui-markdown-rendering
**Date**: 2026-06-12

This feature persists **no new data**. Stored task/goal/plan text is unchanged (FR-010, SC-003); rendering is a display-only transform. The "model" here is the in-memory render model, not a database schema.

## Render-time types

### `Theme`

Palette + style configuration passed *into* the renderer by `internal/tui` (which owns `theme.go`). Keeps the renderer free of any `internal/tui` import (no cycle) while honoring Principle III (single shared palette).

| Field | Type | Meaning |
|---|---|---|
| `Accent` | `color.Color` | Emphasis/link/heading foreground (maps to palette `accent`). |
| `Dim` | `color.Color` | Blockquote bar, code background tint, secondary structure (maps to `dim`). |
| `CodeBg` | `color.Color` | Inline/code-block background tint (derived from palette; no new ad-hoc color). |
| `Text` | `color.Color` (optional) | Default foreground; zero value = terminal default. |

Constructed once at startup from the resolved palette and stored on the `Renderer`.

### `Options`

Per-call rendering inputs.

| Field | Type | Meaning |
|---|---|---|
| `Width` | `int` | Available display width (columns). `<= 0` means "no wrapping." |
| `Styled` | `bool` | Mirrors `Model.styled`. `false` → plain output, zero escape sequences (SC-005). |

`Theme` is held on the `Renderer`, not passed per call.

### `Renderer`

The stateful façade held on `Model`. Owns the configured goldmark `Markdown` parser, the `Theme`, and a memoization cache.

| Field | Type | Meaning |
|---|---|---|
| `md` | `goldmark.Markdown` | Parser configured with `extension.GFM`. |
| `theme` | `Theme` | Style configuration. |
| `cache` | `map[cacheKey]string` | Memoized rendered output; bounded (cap with reset, or small LRU). |

### `cacheKey`

Identity of a render request, so identical re-renders (every Bubble Tea frame) are served from cache.

| Field | Type |
|---|---|
| `text` | `string` |
| `width` | `int` |
| `styled` | `bool` |
| `inline` | `bool` (block vs. inline-only render) |

## Field classification (drives render depth)

The spec's "length character" maps to which entry point a call site uses — there is no stored flag:

| Field | Render mode | Call |
|---|---|---|
| Task name, Goal name (single-line) | Inline-only | `RenderInline` |
| Task description, Goal description, plan-entry notes (multi-line) | Full block | `Render` |
| Report tab text (generated) | As-is | not routed through the renderer unless it carries user-authored markdown |

## AST construct → terminal representation

Reference mapping the walker implements (detail in [contracts/markdown-package.md](./contracts/markdown-package.md)):

| GFM construct | Styled output | Plain output |
|---|---|---|
| Bold / Strong | `Bold` | text only |
| Italic / Emphasis | `Italic` (terminal-dependent display) | text only |
| Bold+Italic | `Bold`+`Italic` | text only |
| Strikethrough | `Strikethrough` | text only |
| Inline code | accent fg on `CodeBg` | backtick-free text |
| Heading | bold accent, level-prefixed | level-prefixed text |
| Unordered list | Unicode bullet by depth (`•/◦/▪`) | `-` markers |
| Ordered list | `N.` | `N.` |
| Task list item | `☑`/`☐` | `[x]`/`[ ]` |
| Blockquote | `dim` `│ ` left bar | `> ` prefix |
| Code block | verbatim on `CodeBg`, not reflowed | verbatim |
| Horizontal rule | full-width `─` | `---` |
| Table | lipgloss/v2 `table`, width-aware | ASCII grid |
| Link | OSC 8 hyperlink, underline+accent | `text (url)` |
| Image | `[image: alt]` placeholder (warm copy) | same |
| Raw HTML | passed through as literal text | same |

## Lifecycle / state

- The `Renderer` is created once (in `NewModel`) and lives for the session; its cache persists across frames and is naturally invalidated by key (changed text or width → new key).
- No transitions, no persistence, no migrations.
