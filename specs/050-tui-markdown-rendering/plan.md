# Implementation Plan: Markdown Rendering in TUI Text Fields

**Branch**: `050-tui-markdown-rendering` | **Date**: 2026-06-12 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/050-tui-markdown-rendering/spec.md`

## Summary

Render GitHub-flavored markdown in the interactive TUI's user-authored text fields: full block rendering (headings, lists, tables, blockquotes, code, rules, links) for long/multi-line fields, and inline-only rendering (bold, italic, strikethrough, inline code, links) for short single-line fields (names). Rendering is display-only; editing shows raw source.

**Technical approach**: A new root-module package `internal/markdown` parses with `github.com/yuin/goldmark` (+ its GFM extension) and walks the AST to emit display-width-aware, `lipgloss/v2`-styled terminal output using the app's existing adaptive theme palette. Tables reuse lipgloss/v2's width-aware `table` package; links use OSC 8 escapes via the already-present `charmbracelet/x/ansi`. A memoizing `Renderer` keeps parsing off Bubble Tea's per-frame hot path. The `internal/tui` render sites (task details, goal detail, tree rows, goal/plan name displays) call the new package instead of the byte-based `wrapDescription`. No server, proto, or web changes.

See [research.md](./research.md) for the tooling survey (why goldmark over glamour) and a frank assessment of what is *not* realistically achievable in a terminal.

## Technical Context

**Language/Version**: Go 1.26.0

**Primary Dependencies**: `github.com/yuin/goldmark` v1.7.13 (new direct dep — already in module cache as transitive); `charm.land/lipgloss/v2` v2.0.3 (existing; styling + `table` subpackage); `github.com/charmbracelet/x/ansi` v0.11.7 (existing indirect; `SetHyperlink` for OSC 8); `rivo/uniseg` / `mattn/go-runewidth` (existing; display-width-aware wrapping).

**Storage**: N/A — display-only; no persistence changes (FR-010, SC-003).

**Testing**: `go test ./...` (root module). Golden-style assertions on rendered strings, asserting both styled (ANSI-bearing) and plain (escape-free) output.

**Target Platform**: Linux/macOS terminal (interactive TUI on a TTY).

**Project Type**: Single project — CLI/TUI client (root module).

**Performance Goals**: No perceptible render lag (SC-006). Achieved via memoized rendering keyed by `(text, width, styled, inline)`; parsing happens on content/width change, not every frame.

**Constraints**: Output must be display-width correct for non-ASCII/emoji (supersede the byte-based `wrapDescription`); plain mode must emit zero escape sequences (SC-005); never panic on malformed input (FR-012); all colors/styles drawn from the existing `theme.go` palette only (Principle III).

**Scale/Scope**: ~6 TUI render sites; one new package (~parser config + AST walker + table/link helpers + cache). TUI only — plain CLI and the React web app are explicitly out of scope.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ (with note) | One new dependency (goldmark) and one focused package. A custom AST walker is the *simplest correct* path: GFM is too irregular to hand-parse, and it is the only way to get inline-only rendering + theme integration. Renderer scope is deliberately minimal — no syntax highlighting, no images-as-graphics, no config surface. See Complexity Tracking. |
| II. API-First Design | ✅ | No network contract changes. The "contract" is the new package's public Go API, defined before implementation in [contracts/markdown-package.md](./contracts/markdown-package.md). |
| III. UI/UX Consistency | ✅ | All markdown styles map onto existing palette tokens (`accent`, `dim`, `completed`, etc.); the renderer receives a `Theme` built by `internal/tui` from `theme.go`. No ad-hoc colors. Rendering respects the existing `styled` gate. |
| IV. Playful User Messages | ✅ | Minimal new copy. Only user-facing strings are placeholders for unrenderable content (e.g. image alt fallback), which will follow the warm-tone convention. |

No unjustified violations. Gate **passes**.

## Pushback: spec items that are over-ambitious (for user decision)

The user asked for pushback on anything unrealistic. Two items in the (otherwise sound) spec overreach what a terminal can guarantee. Neither blocks implementation — the rendering code is identical either way — but the spec wording should be softened so acceptance tests are not written against impossible guarantees:

1. **Link fallback detection (FR-007)** — The spec says emit OSC 8 "on terminals that support hyperlink escape sequences" and fall back otherwise. *There is no reliable runtime detection of OSC 8 support.* Realistic rule: styled mode → emit OSC 8 (clickable in supporting terminals; silently ignored elsewhere); plain mode → `text (url)`. **Recommendation**: reword FR-007 so the inline-URL fallback is tied to *plain/unstyled* output, not to undetectable terminal capability. Consequence accepted: in styled mode on a non-OSC-8 terminal, the URL is not shown inline (the styled link text still is).

2. **"All content stays visible" for tables/code (FR-011 + overflow clarification)** — Soft-wrap is achievable for prose, lists, quotes, and (via lipgloss `table`) most tables. But: code blocks and long URLs are intentionally *not* wrapped (wrapping changes their meaning), and pathological wide tables degrade to a few characters per column. **Recommendation**: soften to "best-effort soft-wrap; code blocks / unbreakable tokens may clip at the pane boundary; very wide tables degrade in readability but never corrupt surrounding UI or panic."

Everything else in the spec (inline emphasis, lists, blockquotes, task lists, rules, headings-as-styled-text, display-only round-trip, plain-mode degradation, malformed-input safety, non-ASCII width) is realistically achievable as designed.

## Project Structure

### Documentation (this feature)

```text
specs/050-tui-markdown-rendering/
├── plan.md              # This file
├── research.md          # Phase 0 — tooling survey + feasibility pushback
├── data-model.md        # Phase 1 — render model (no persisted entities)
├── quickstart.md        # Phase 1 — how to use & verify the renderer
├── contracts/
│   └── markdown-package.md   # Public Go API of internal/markdown
└── checklists/
    └── requirements.md  # Spec quality checklist (from /speckit-specify)
```

### Source Code (repository root)

```text
internal/markdown/                # NEW package (root module)
├── markdown.go        # Renderer type, Options/Theme, Render() + RenderInline(), cache
├── block.go           # block-level AST walker (headings, lists, quotes, code, rules)
├── table.go           # GFM table → lipgloss/v2 table
├── inline.go          # inline AST walker (emphasis, code, strikethrough, links)
├── link.go            # OSC 8 vs. text (url) link rendering
├── markdown_test.go   # styled + plain golden assertions, edge cases
├── inline_test.go
└── table_test.go

internal/tui/
├── theme.go           # ADD: helper to build markdown.Theme from the palette
├── details.go         # CHANGE: task name (inline) + description (block) via markdown
├── goal_view.go       # CHANGE: goal name (inline) + description (block) via markdown
├── view.go            # CHANGE: tree-row task names via inline render
├── plan_view.go       # CHANGE: plan entry / task name displays via inline render
├── report_view.go     # (verify) Report text rendered as-is unless user-authored
└── model.go           # ADD: *markdown.Renderer field on Model (holds cache)
```

**Structure Decision**: Single-project layout. The renderer lives in its own root-module package `internal/markdown` (not inside `internal/tui`) so it is independently unit-testable and free of Bubble Tea types. It depends only on goldmark + lipgloss/v2 + ansi. To avoid an import cycle and honor Principle III, `internal/tui` owns the palette (`theme.go`) and passes a `markdown.Theme` into the renderer; the renderer never imports `internal/tui`. The memoizing `Renderer` is held on the `Model` so its cache lives for the session.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| New `internal/markdown` package + goldmark dependency | FR-001–FR-009 require faithful GFM rendering, including inline-only rendering for single-line fields, which no maintained off-the-shelf terminal renderer provides. | (a) glamour — document-oriented, lipgloss-v1, no inline API, bundles chroma, clashes with the shared theme (Principle III). (b) Hand-rolled parser — GFM (tables, nested emphasis, code spans, escapes) is too irregular to parse correctly by hand; a real parser is the simpler *correct* path. |
| Memoizing `Renderer` cache | Bubble Tea re-renders `View()` every message; re-parsing markdown for every visible row per frame risks perceptible lag (SC-006). | A stateless render-every-frame function is simpler but reparses on the hot path; caching keyed by `(text,width,styled,inline)` is the minimal mechanism that meets the perf goal. |

## Phase Outputs

- **Phase 0** → [research.md](./research.md) ✅ (all NEEDS CLARIFICATION resolved; tooling chosen; feasibility limits documented)
- **Phase 1** → [data-model.md](./data-model.md), [contracts/markdown-package.md](./contracts/markdown-package.md), [quickstart.md](./quickstart.md), CLAUDE.md agent-context pointer updated ✅
- **Phase 2** → tasks.md (generated by `/speckit-tasks`, not this command)
