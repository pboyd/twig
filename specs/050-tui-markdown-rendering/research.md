# Research: Markdown Rendering in TUI Text Fields

**Feature**: 050-tui-markdown-rendering
**Date**: 2026-06-12

This document resolves the open technical questions deferred from the spec (markdown tooling choice, terminal-capability constraints) and records a frank assessment of which spec requirements are realistically achievable in a terminal.

## Decision 1: Parser — `github.com/yuin/goldmark` (custom renderer)

**Decision**: Use `goldmark` with its built-in GFM extension (`goldmark/extension.GFM`) for parsing, and write a small custom AST walker that emits lipgloss-styled, width-aware terminal output. Render into a new root-module package `internal/markdown`.

**Rationale**:

- goldmark is the de-facto, actively maintained (v1.7.13) CommonMark + GFM parser for Go. Its `extension.GFM` bundle gives tables, strikethrough, task lists, and autolinks — exactly the GFM constructs FR-003 enumerates. It is already present in the module cache as a transitive dependency, so adding it as a direct dependency of the CLI module is low-risk and pulls in no CGo or heavy sub-deps.
- goldmark cleanly separates parsing from rendering. We can walk the AST ourselves and emit ANSI via the existing `charm.land/lipgloss/v2` palette, which is the only way to satisfy **Principle III (UI/UX consistency)**: markdown styling must come from the app's shared adaptive theme, not a bundled stylesheet.
- A custom walker is also the only way to satisfy **FR-008 (inline-only rendering for short, single-line fields)**: we walk only inline AST nodes (emphasis, strong, code, strikethrough, link, text) and ignore block structure, collapsing to one line. No off-the-shelf terminal renderer exposes this.

**Alternatives considered**:

| Alternative | Why rejected |
|---|---|
| **charmbracelet/glamour** | The obvious off-the-shelf option and actively maintained, but: (1) it is document-oriented (designed to render whole READMEs with its own word-wrap, margins, and JSON style themes) and exposes no API to render an inline fragment for a single-line task name (fails FR-008); (2) it bundles `chroma` syntax highlighting and its own ANSI/styling stack built on **lipgloss v1** (`github.com/charmbracelet/lipgloss`), while this project is on **`charm.land/lipgloss/v2`** — mixing the two is awkward and bloats the lean CLI module; (3) its opinionated styling does not map onto our adaptive `theme.go` palette, conflicting with Principle III. Usable for long fields only, but adopting it *and* hand-rolling inline rendering means two rendering engines — worse than one custom walker. |
| **MichaelMure/go-term-markdown** | Renders markdown to a terminal and even handles images, but is effectively unmaintained (no meaningful activity in years) — matches the user's finding that nothing off-the-shelf both fits and is maintained. Also lipgloss-v1-era and not inline-capable. |
| **gomarkdown/markdown + custom** | Viable parser, but goldmark has first-class GFM support, better-maintained, and is already in our dependency graph. No advantage. |
| **Hand-rolled regex/parser** | Rejected per Principle I in the other direction — GFM is too irregular (tables, nested emphasis, code spans, escapes) to parse correctly by hand. A real parser is the *simpler* correct path. |

## Decision 2: Inline emphasis styling via lipgloss

**Decision**: Map markdown inline constructs to `lipgloss.Style` attributes drawn from the existing palette:

- **Bold** → `Bold(true)`
- **Italic** → `Italic(true)` (see caveat below)
- **Bold+italic** → both
- **Strikethrough** → `Strikethrough(true)`
- **Inline code** → foreground accent on a subtle background (reuse `accent`/`dim` tokens; no new colors)
- **Links** → underline + accent (see Decision 4)

lipgloss/v2 supports `Bold`, `Italic`, `Underline`, and `Strikethrough`, so no escape-sequence hand-assembly is needed for emphasis.

**Caveat (reality check)**: Terminal **italic** (SGR 3) is inconsistently supported — some terminals ignore it, some render it as reverse video. We will apply it, but acceptance for italic is "italic styling is applied," not "every terminal displays slanted glyphs." This is an inherent terminal limitation, not a defect.

## Decision 3: Block rendering — lists, tables, quotes, code, headings, rules

**Decision**:

- **Headings** → bold accent text, with the level conveyed by a leading marker (e.g. a styled `#`/`##` prefix or weight). True heading semantics do not exist in a terminal (acknowledged in FR-006); this is an approximation.
- **Unordered lists** → Unicode bullets (`•`, `◦`, `▪` by depth); **ordered lists** → `N.`; **task lists** → `☐`/`☑`. Indentation by nesting depth.
- **Blockquotes** → a styled left bar (`│ ` in `dim`) per line.
- **Fenced/indented code blocks** → preserved verbatim (no reflow, no syntax highlighting — Principle I) on a subtle background; long lines are the one place content may exceed width (see Decision 5).
- **Horizontal rules** → a full-width line of `─`.
- **Tables** → rendered through lipgloss/v2's `table` package, which is width-aware: it sizes columns to the available width and wraps cell content. This makes GFM tables far more tractable than a hand-rolled grid.

**Rationale**: All glyphs are standard Unicode and all styling reuses palette tokens. Reusing the lipgloss `table` package (already a dependency) avoids reinventing column layout and gives us reflow "for free."

## Decision 4: Links and the OSC 8 detection problem

**Decision**: When rendering in **styled** mode, emit the link text wrapped in an OSC 8 hyperlink escape (`ansi.SetHyperlink`, available in the already-present `charmbracelet/x/ansi`) and style it (underline + accent). In **plain/unstyled** mode, render `text (url)` so the destination is visible.

**Reality check / pushback on the spec**: FR-007 (as clarified) says emit OSC 8 "on terminals that support hyperlink escape sequences" and fall back "on terminals without that support." **There is no reliable runtime way to detect per-terminal OSC 8 support** — there is no query/response for it. So the achievable rule keys off the boundary we *can* detect (styled vs. plain), not off true OSC 8 support:

- Terminals that understand OSC 8 → clickable link text.
- Terminals that don't → they silently ignore the OSC wrapper and still show the (styled) link text; the raw URL is **not** shown inline in styled mode.

This means in styled mode on a non-OSC-8 terminal, the URL itself is not visible inline. **Recommended spec adjustment**: reword FR-007 so the inline `text (url)` fallback is tied to *plain/unstyled* output rather than to undetectable "OSC 8 support." This is captured as an open item for the user in the plan's pushback section.

## Decision 5: Width, wrapping, and the table-reflow ambition

**Decision**: All wrapping is **display-width aware** (using `rivo/uniseg` / `mattn/go-runewidth`, both already dependencies) — not byte-length based. The existing `wrapDescription` helper wraps on `len()` (bytes), which is wrong for non-ASCII; the markdown renderer must not repeat that and should supersede `wrapDescription` for fields it renders.

**Reality check / pushback on the clarified overflow guarantee**: The clarification states overflow content soft-wraps so "all content stays visible," with "table columns reflow/shrink to fit." This is **mostly** achievable but has two honest limits:

1. **Very wide tables** (many columns, or long unbreakable cell tokens) at a narrow pane width get squeezed to a few characters per column via the lipgloss table sizer. Content technically wraps and stays "visible," but readability degrades — there is a practical minimum width below which a wide table is cramped. We will not guarantee pretty results for pathological tables; we guarantee no layout corruption and no panics.
2. **Code blocks and long URLs** are intentionally **not** wrapped (wrapping code changes its meaning; wrapping a URL breaks it). These may extend to the pane edge and are **clipped** by the surrounding lipgloss pane rather than wrapped. This is a deliberate, narrow exception to "all content stays visible."

**Recommended spec adjustment**: soften FR-011 / the overflow clarification to "best-effort soft-wrap; code blocks and unbreakable tokens may be clipped at the pane boundary; extremely wide tables degrade in readability but never corrupt surrounding UI." Captured for the user in the plan.

## Decision 6: Performance — memoized rendering

**Decision**: Wrap the renderer in a `Renderer` value that **caches** rendered output keyed by `(text, width, styled, inline)`. The TUI re-runs `View()` on every Bubble Tea message; re-parsing markdown for every visible tree row on every keystroke would be wasteful. The cache is invalidated implicitly by key (a changed width or text produces a new key); a bounded map (or simple size cap with reset) prevents unbounded growth.

**Rationale**: Satisfies SC-006 ("no perceptible lag"). Inline rendering of short names is cheap, but multiplied by rows × frames it adds up; long-field block rendering is more expensive and benefits most from caching. Caching keeps parsing off the hot path. This is the one piece of deliberate machinery beyond a naive renderer, justified by the re-render model.

## Decision 7: Degradation and safety

- **No color/ANSI** (`styled == false`, already detected via `cli.WantStyled(os.Stdout)`): emit plain text with structural glyphs but **zero escape sequences** (FR-013, SC-005). The renderer takes `styled` and branches, mirroring the existing codebase pattern.
- **Malformed markdown**: goldmark does not error on malformed input — it produces a best-effort AST — so FR-012 ("never crash") is satisfied by construction. The walker must still guard against unexpected node types (default: render the node's raw text).
- **Images / raw HTML** (FR-014): images render as their alt text (or `[image: alt]` placeholder); raw HTML is passed through as literal text rather than interpreted. Any user-facing placeholder copy follows Principle IV (warm tone).
- **Literal markup characters** (e.g. a bare `*` not forming emphasis): goldmark already treats these as text, so they are preserved (edge case satisfied).

## Summary of recommended spec adjustments (for user decision)

1. **FR-007 (links)**: tie the inline `text (url)` fallback to *plain/unstyled* output, not to undetectable OSC 8 support.
2. **FR-011 / overflow clarification (tables & code)**: soften "all content stays visible" to best-effort wrap, with code blocks / long URLs / pathological wide tables allowed to clip at the pane boundary without corrupting layout.

Both are presented to the user in the plan; neither blocks starting implementation, since the core rendering work is identical either way.
