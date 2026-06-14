# Contract: `Markdown` Component (web app)

**Feature**: `055-markdown-web-rendering` | **Date**: 2026-06-14

This feature changes **no network contract** — there are no proto, endpoint, or
payload changes (Principle II is satisfied: the existing `task.v1.TaskService`
contract is consumed unchanged). The "contract" for this frontend-only feature
is the **public API of the new React component** that all render sites depend on.
It is defined here before implementation so render sites and the component can be
built against a stable interface.

## Location

`services/twig-web/src/components/Markdown.tsx`

## Public API

```ts
export interface MarkdownProps {
  /** Raw markdown source. Rendered display-only; never mutated. */
  children: string;

  /**
   * Rendering mode:
   * - "block"  → full GFM construct set (headings, lists, tables, code, etc.)
   * - "inline" → emphasis-level inline constructs only, single-line, no block
   *              elements emitted.
   * @default "block"
   */
  mode?: "block" | "inline";

  /** Optional extra class names applied to the wrapper element. */
  className?: string;
}

/**
 * Renders GitHub-flavored markdown as safe React elements using the app's
 * theme tokens. Display-only. Memoized on (children, mode, className).
 */
export function Markdown(props: MarkdownProps): JSX.Element;
```

## Behavioral contract

| ID | Requirement | Maps to |
|----|-------------|---------|
| C-01 | Renders via React elements only — MUST NOT use `dangerouslySetInnerHTML`, and MUST NOT enable `rehype-raw`. Embedded raw HTML renders as inert text. | FR-009, SC-003 |
| C-02 | `mode="block"` supports: bold, italic, bold-italic, strikethrough, inline code, links, headings (h1–h6), ordered/unordered/nested lists, task-list checkboxes, blockquotes, fenced & indented code blocks, horizontal rules, GFM tables. | FR-001, FR-002, FR-003 |
| C-03 | `mode="inline"` emits only inline elements (`strong`, `em`, `del`, `code`, `a`); no block element (`p`, `h*`, `ul`, `ol`, `blockquote`, `table`, `pre`, `hr`) is produced. Block syntax degrades to legible inline text on a single line. | FR-006 |
| C-04 | Source markers for supported constructs are consumed (not shown) in output. | FR-004 |
| C-05 | Links render as `<a>` with safe `href` (dangerous schemes neutralized); external links use `target="_blank"` + `rel="noopener noreferrer"`. | FR-005 |
| C-06 | Single newlines render as line breaks (parity with prior `whitespace-pre-wrap`). | FR-012, SC-005 |
| C-07 | Empty/whitespace-only `children` renders nothing (no wrapper artifact, no error). | edge case |
| C-08 | Malformed/incomplete markdown renders best-effort readable text without throwing. | FR-010 |
| C-09 | Wide tables, long code lines, and long URLs are contained (scroll/wrap within region); surrounding layout is never broken. | FR-011 |
| C-10 | All styling derives from existing theme tokens / Tailwind conventions, with dark-mode (`dark:`) parity. No ad-hoc colors. | Principle III |
| C-11 | Component is referentially stable for identical `(children, mode, className)` (memoized) so unrelated re-renders do not re-parse. | SC-006 |
| C-12 | Rendering is display-only; it never emits callbacks or side effects that mutate the source text. | FR-007 |

## Consumers (render sites updated to use this component)

| Site | File | Mode | Replaces |
|------|------|------|----------|
| Task detail description | `src/pages/TaskDetailPage.tsx` | `block` | `<p className="… whitespace-pre-wrap">{task.description}</p>` |
| Task detail header name | `src/pages/TaskDetailPage.tsx` | `inline` | `{task.name}` in `<h1>` |
| Tree row name | `src/components/TreeRow.tsx` | `inline` | `{task.name}` |
| Plan entry name | `src/components/PlanEntryRow.tsx` | `inline` | `{entry.displayName}` |

Editing surfaces (`TaskForm`) are **not** consumers — they continue to present
raw markdown source in a `<textarea>` (FR-008).

## Test contract

Interaction tests (Vitest + Testing Library, per existing `*.test.tsx`
conventions) MUST cover, at minimum:
- block mode renders each construct to its expected element (heading→`<h*>`,
  list→`<li>`, table→`<table>`, link→`<a href>`, code fence→`<code>` in `<pre>`,
  strikethrough→`<del>`, task checkbox→checkbox input);
- inline mode renders emphasis but emits no block element and stays single-line;
- a `javascript:` link is neutralized (no executable `href`);
- raw `<script>`/`<img onerror>` in source does not produce an executable
  element;
- malformed markdown renders without throwing;
- empty string renders nothing.
