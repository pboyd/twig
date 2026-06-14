# Implementation Plan: Markdown Rendering in the Web App

**Branch**: `055-markdown-web-rendering` | **Date**: 2026-06-14 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/055-markdown-web-rendering/spec.md`

## Summary

Render GitHub-flavored markdown in the React web app's user-authored task fields,
bringing the TUI's markdown support (`050-tui-markdown-rendering`) to the browser
where it can be reproduced faithfully (real headings, clickable links, ruled
tables). Long-text content (task **description**) gets full block rendering;
single-line **names** get inline-only rendering (bold/italic/strikethrough/inline
code/links) everywhere they appear (tree rows, detail header, day planner).
Rendering is **display-only** — editing keeps showing raw source.

**Technical approach**: Add one small React component
`src/components/Markdown.tsx` with two modes (`block` / `inline`), built on
[`react-markdown`](https://github.com/remarkjs/react-markdown) + `remark-gfm`.
`react-markdown` renders React elements (no `dangerouslySetInnerHTML`) and, with
`rehype-raw` deliberately omitted, treats embedded HTML as inert text and strips
dangerous URL schemes — so the markdown-XSS vector is closed by construction
(FR-009/SC-003) with no extra sanitizer. Element styling uses the existing theme
tokens / Tailwind dark-mode conventions (Principle III). Four display sites are
switched to the new component; the `TaskForm` editor is untouched (round-trip
safety). **No server, proto, sqlc, or API-contract changes.**

See [research.md](./research.md) for the library survey (why `react-markdown`
over `marked`+DOMPurify, and why no `@tailwindcss/typography`) and
[contracts/markdown-component.md](./contracts/markdown-component.md) for the
component's public API and behavioral contract.

## Technical Context

**Language/Version**: TypeScript 5.8, React 19 (SPA at `services/twig-web/`).

**Primary Dependencies**: New direct deps `react-markdown` (~v9) and `remark-gfm`
(~v4). Existing: Vite 6, Tailwind CSS v4, ConnectRPC/connect-query, react-router 7.
**No** `rehype-raw`, **no** DOMPurify (safety comes from `react-markdown`'s
React-element rendering + default URL transform).

**Storage**: N/A — display-only; markdown is stored unchanged as the existing
`task.v1.Task.name` / `.description` strings. No DB or proto changes.

**Testing**: Vitest 3 + Testing Library + jsdom (existing `*.test.tsx` /
`*.test.ts` conventions). New `Markdown.test.tsx` interaction tests; existing
page/component tests updated where text assertions change.

**Target Platform**: Modern browsers (authenticated SPA served same-origin behind
the server; dev via Vite proxy to `:8080`).

**Project Type**: Web frontend (React SPA) — sibling to the Go server module;
backend untouched.

**Performance Goals**: No perceptible render delay vs. the current plain-text view
(SC-006). Achieved by memoizing `Markdown` on `(children, mode, className)` so
unrelated re-renders do not re-parse.

**Constraints**: Untrusted content must never execute on view (FR-009/SC-003);
display-only with byte-for-byte round-trip (FR-007/SC-004); wide/long content
contained within its region, never breaking layout (FR-011); plain text visually
unchanged incl. line breaks (FR-012/SC-005); all styling from existing tokens
(Principle III).

**Scale/Scope**: One new component + its test; four render-site edits; two new npm
deps. Web app only — TUI/CLI and backend are out of scope.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One focused component reusing a mature library — no custom parser, no DOMPurify, no `@tailwindcss/typography`. Two render modes share one pipeline. No live-preview/editor changes (deferred per spec). No speculative config surface. |
| II. API-First Design | ✅ | No network contract change. The "contract" is the component's public API, fixed before implementation in [contracts/markdown-component.md](./contracts/markdown-component.md). Existing `task.v1.TaskService` consumed unchanged. |
| III. UI/UX Consistency | ✅ | Markdown elements map onto existing theme tokens (`src/theme/tokens.ts`) and the app's Tailwind dark-mode conventions; links match the existing in-app link style. No ad-hoc colors. Construct set kept consistent with the TUI (FR-013). |
| IV. Playful User Messages | ✅ | Essentially no new user-facing copy. Any placeholder text (e.g. image alt fallback) reuses the warm tone in `src/theme/messages.ts`. |

No unjustified violations. Gate **passes** (re-confirmed post-design — no new
abstractions introduced in Phase 1).

## Project Structure

### Documentation (this feature)

```text
specs/055-markdown-web-rendering/
├── plan.md              # This file
├── research.md          # Phase 0 — library & approach decisions
├── data-model.md        # Phase 1 — view-model (no persisted entities)
├── quickstart.md        # Phase 1 — setup, run, verify
├── contracts/
│   └── markdown-component.md   # Phase 1 — Markdown component API + behavior
├── checklists/
│   └── requirements.md  # Spec quality checklist (from /speckit-specify)
└── tasks.md             # Phase 2 — created by /speckit-tasks (NOT here)
```

### Source Code (repository root)

```text
services/twig-web/
├── package.json                       # + react-markdown, remark-gfm
├── src/
│   ├── components/
│   │   ├── Markdown.tsx                # NEW — block/inline GFM renderer
│   │   ├── Markdown.test.tsx           # NEW — interaction tests
│   │   ├── TreeRow.tsx                 # EDIT — name → <Markdown mode="inline">
│   │   └── PlanEntryRow.tsx            # EDIT — displayName → inline
│   ├── pages/
│   │   ├── TaskDetailPage.tsx          # EDIT — description → block; name → inline
│   │   ├── TaskDetailPage.test.tsx     # EDIT if text assertions change
│   │   └── PlanPage.test.tsx           # EDIT if text assertions change
│   └── theme/
│       └── tokens.ts                   # reused (no change expected)
```

**Structure Decision**: Single existing project — the React SPA at
`services/twig-web/`. All changes are confined to `src/components/` and
`src/pages/` plus the two new npm dependencies in `package.json`. No new
top-level directories; the Go server, `api/` proto module, and CLI/TUI modules
are untouched.

## Complexity Tracking

> No Constitution Check violations — this table is intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
