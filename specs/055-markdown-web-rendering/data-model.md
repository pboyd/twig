# Data Model: Markdown Rendering in the Web App

**Feature**: `055-markdown-web-rendering` | **Date**: 2026-06-14

This feature is **display-only** and introduces **no new persisted data, no new
API messages, and no schema changes**. Markdown is stored exactly as today: as
the plain-text content of existing fields. What follows describes the *view-model
concepts* the rendering layer operates on — not database entities.

## Existing fields rendered (no change to storage)

| Field | Source type | Render mode | Render sites |
|-------|-------------|-------------|--------------|
| Task **name** | `task.v1.Task.name` (string) | Inline-only | `TreeRow` (tree), `TaskDetailPage` (header `<h1>`), `PlanEntryRow` (`entry.displayName`) |
| Task **description** | `task.v1.Task.description` (string) | Block (full) | `TaskDetailPage` (detail body) |

- Both fields are already part of the `task.v1.TaskService` contract; this
  feature consumes them unchanged.
- `PlanEntryRow` renders `entry.displayName`, a view-model string derived from
  the task name (for `kind === "task"` entries). Free-text plan entries reuse the
  same inline rendering for visual consistency.

## Render modes (view-model)

### Block mode
Full GitHub-flavored markdown construct set. Used for multi-line content.

Supported constructs (parity with the TUI feature, FR-013):
- **Inline**: bold, italic, bold-italic, strikethrough, inline code, links
- **Block**: headings (h1–h6), unordered lists, ordered lists, nested lists,
  task-list checkboxes, blockquotes, fenced & indented code blocks, horizontal
  rules, tables (with header row + column alignment)
- **Soft line breaks**: single newlines render as line breaks (parity with the
  current `whitespace-pre-wrap` behavior)

### Inline mode
Emphasis-level constructs only, rendered on a single line:
- bold, italic, bold-italic, strikethrough, inline code, links
- Block-level syntax (leading `#`, `-`, `>`, etc.) degrades to legible inline
  text; no block element is emitted (FR-006).

## Rendering invariants

These are the rules the rendering layer MUST uphold (derived from FRs):

1. **Display-only** — rendering never mutates the stored string (FR-007). The
   editor (`TaskForm`) shows raw source (FR-008).
2. **No raw HTML execution** — embedded HTML is inert/text; no
   `dangerouslySetInnerHTML` (FR-009).
3. **Safe link targets** — dangerous URL schemes (e.g. `javascript:`) are
   neutralized; external links open with `rel="noopener noreferrer"` (FR-005).
4. **No syntax leakage** — source markers for supported constructs are consumed,
   not displayed (FR-004).
5. **Contained overflow** — wide tables / long code lines / long URLs scroll or
   wrap within their region and never break page layout (FR-011).
6. **Fail-safe** — malformed/empty/plain content renders as readable text and
   never errors (FR-010, FR-012).

## State transitions

None. There is no stateful entity; rendering is a pure function of
`(text, mode)`.
