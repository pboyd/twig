# Research: Markdown Rendering in the Web App

**Feature**: `055-markdown-web-rendering` | **Date**: 2026-06-14

This document resolves the technical unknowns for rendering GitHub-flavored
markdown in the React web app (`services/twig-web/`). The companion TUI feature
(`050-tui-markdown-rendering`) established the supported construct set and the
display-only / round-trip-safe philosophy; this feature brings the same content
to the browser, where rendering can be more faithful (real headings, clickable
links, ruled tables) but introduces a sanitization concern absent in the
terminal.

## Decision 1: Markdown rendering library — `react-markdown` + `remark-gfm`

**Decision**: Render markdown with [`react-markdown`](https://github.com/remarkjs/react-markdown) configured with the [`remark-gfm`](https://github.com/remarkjs/remark-gfm) plugin.

**Rationale**:
- `react-markdown` parses markdown to an AST and renders **React elements directly** — it does **not** use `dangerouslySetInnerHTML`. This is the single most important property for FR-009/SC-003: there is no HTML string to inject, so the classic markdown-XSS vector is closed by construction.
- It does **not** render raw/embedded HTML unless the `rehype-raw` plugin is explicitly added. We will **not** add `rehype-raw`, so any raw HTML in source is treated as literal text — satisfying the "raw HTML is inert" edge case without a separate sanitizer dependency.
- It ships a default URL transform (`defaultUrlTransform`) that strips dangerous schemes such as `javascript:` from `href`/`src`, directly satisfying FR-005.
- `remark-gfm` adds exactly the GFM constructs the spec requires that core markdown lacks: tables, strikethrough, task-list checkboxes, autolinks. This aligns the web construct set with the TUI's goldmark-GFM set (FR-013/SC-002).
- It exposes a `components` prop to map each element to our own React components, letting us apply existing Tailwind theme tokens (Principle III) rather than inheriting opinionated default styles.
- Mature, widely used, actively maintained, TypeScript-native — low risk, no custom parser needed (Principle I: we do not hand-roll a markdown parser).

**Alternatives considered**:
- **`marked` / `markdown-it` + `DOMPurify`**: Produces an HTML string that must be injected via `dangerouslySetInnerHTML` and then sanitized with DOMPurify. This is the higher-risk path (a sanitizer misconfiguration = stored XSS) and pulls in two dependencies plus a manual sanitize step. Rejected: more moving parts and more risk for no fidelity gain.
- **`@tailwindcss/typography` (`prose`) for styling**: Considered for the description block. Rejected as the *primary* mechanism because its opinionated defaults (margins, colors, dark-mode via `prose-invert`) would need overriding to match the existing token palette, and it does not address inline-only rendering for names. A small custom `components` map is simpler and keeps full token control (see Decision 3). `prose` may optionally be used later but is not required.
- **Custom AST walker (mirroring the TUI's `internal/markdown`)**: Unnecessary in the browser — the DOM already renders headings/links/tables natively, so a hand-written renderer would be reinventing what `react-markdown` provides safely. Rejected per Principle I.

**Bundle impact**: `react-markdown` + `remark-gfm` add roughly ~40–45 KB gzipped. Acceptable for an authenticated SPA; loaded on the task-detail route where the description renders. Inline-name rendering reuses the same already-loaded modules.

## Decision 2: Inline-only rendering for short fields (names)

**Decision**: Provide a thin `Markdown` wrapper with two modes — **block** (full construct set, used for task description) and **inline** (emphasis-level constructs only, used for task names) — both backed by the same `react-markdown` + `remark-gfm` pipeline.

**Rationale**:
- Names appear in single-line contexts (tree rows, detail header, planner) and must not emit block elements (`<p>`, `<h1>`, `<ul>`) that would break layout (FR-006, US2 scenario 2).
- Inline mode restricts output to inline elements (`em`, `strong`, `del`, `code`, `a`) using `react-markdown`'s `allowedElements` + `unwrapDisallowed`, and maps the wrapping `p` to a fragment/`span`. Block syntax like a leading `# ` degrades to legible inline text rather than producing a heading.
- Sharing one pipeline keeps behavior consistent between names and descriptions (Principle III) and avoids a second dependency.

**Alternatives considered**:
- **Regex-based inline emphasis for names**: Rejected — fragile, diverges from the GFM parser used elsewhere, and risks inconsistent rendering between a name and the same syntax in a description.

## Decision 3: Styling via a token-aligned `components` map

**Decision**: Map markdown elements to React components/`className`s built from the existing theme (`src/theme/tokens.ts` palette and the Tailwind utility conventions already used across pages), rather than adopting a generic prose stylesheet.

**Rationale**:
- Principle III (UI/UX Consistency): headings, code, blockquotes, tables, and links must look like the rest of the app, including dark mode (`dark:` variants are used throughout, e.g. `TaskDetailPage`). A component map lets each construct reuse the same color/spacing tokens already in use.
- Links must be styled like existing in-app links (e.g. the blue `text-blue-700 dark:text-blue-400 hover:underline` used in `PlanEntryRow`) and open external destinations safely (`target="_blank"` + `rel="noopener noreferrer"`).
- Code blocks need contained overflow (`overflow-x-auto`) so wide content scrolls within the block and never breaks page layout (FR-011).
- Tables need a horizontally scrollable wrapper for the same reason.

**Alternatives considered**:
- **`@tailwindcss/typography`**: see Decision 1. Viable but heavier to align with tokens; deferred.

## Decision 4: Display-only rendering preserves stored content

**Decision**: Markdown is rendered only in **display** contexts. Editing continues to use the existing raw `<textarea>` in `TaskForm`; no preview pane is added in this iteration.

**Rationale**:
- FR-007/FR-008/SC-004 require byte-for-byte round-trip safety. `TaskForm` already edits raw text and `lib/updatePayload.ts` sends a full-replace payload, so the stored string is exactly what the user typed. Rendering happens elsewhere (detail view, tree, planner) and never feeds back into the stored value.
- No backend, proto, or `updatePayload` changes are required (Principle II: no contract change).
- A live preview is explicitly out of scope (spec Assumptions) to keep the iteration small (Principle I).

## Decision 5: Robustness, performance, and accessibility

**Decisions**:
- **Malformed input**: `react-markdown` does not throw on malformed markdown — it renders best-effort text. Components are pure render functions, so there is no crash path (FR-010). An interaction test will assert an unterminated table / stray markup renders without error.
- **Empty/plain content**: empty/whitespace renders as nothing; plain text renders as a single paragraph (block mode) or inline text (inline mode), visually equivalent to today's `whitespace-pre-wrap` presentation. Line breaks: enable soft line breaks so single newlines render as `<br>` to match the current `whitespace-pre-wrap` behavior (FR-012, SC-005).
- **Performance**: parsing happens on render; memoize the `Markdown` component with `React.memo` keyed on `(text, mode)` so unchanged content does not re-parse during unrelated re-renders (SC-006). For the field sizes in this app this is comfortably imperceptible.
- **Accessibility**: rendered semantic elements (headings, lists, tables) improve screen-reader structure over the current flat text. External links carry `rel="noopener noreferrer"`.

## Summary of resolved unknowns

| Unknown | Resolution |
|---------|-----------|
| Which library? | `react-markdown` + `remark-gfm` (no `rehype-raw`) |
| XSS / sanitization (FR-009) | Inherent: React-element rendering, no raw HTML, built-in URL transform |
| Inline vs block for names (FR-006) | One wrapper, two modes (`allowedElements` + `unwrapDisallowed`) |
| Styling / consistency (Principle III) | Token-aligned `components` map, dark-mode aware |
| Round-trip safety (FR-007/008) | Display-only; existing raw `TaskForm` textarea unchanged |
| Backend changes | None |
| Performance (SC-006) | `React.memo` on `(text, mode)` |
| Construct parity with TUI (FR-013) | GFM via `remark-gfm`, matching goldmark-GFM set |
