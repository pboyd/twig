# Research: Scrolling Task List

**Feature**: 064-scrolling-tasks | **Date**: 2026-07-15

The spec had no `[NEEDS CLARIFICATION]` markers. This document records the design decisions taken against the existing codebase so implementation can proceed without ambiguity.

## Decision 1 — Why the list doesn't scroll today

- **Finding**: `renderList` (`internal/tui/view.go`) joins **all** rows in `m.visible`. The caller view functions (`viewList`, `viewWithForm`, `viewWithMove`, `viewWithDatePrompt`, `viewWithFilter`) then either hand the result to `paneBox` (styled — lipgloss `Height` truncates overflow) or to `splitLines(list, maxLines)` (plain — slices to the top `maxLines`). Both clip from the **top**, so once the cursor moves below the visible area it disappears and rows past the clip are unreachable.
- **Decision**: Introduce a viewport window inside `renderList` driven by a scroll offset and the cursor position.
- **Rationale**: The clipping already happens at the pane boundary; we only need `renderList` to emit the correct one-screen slice instead of the whole list.
- **Alternatives considered**: Swapping in `bubbles/viewport` — rejected (YAGNI; the list has custom per-row cursor styling, markdown inline rendering, and tree prefixes that a generic viewport would complicate; the offset math is a few lines).

## Decision 2 — Cursor-driven scroll vs. independent scroll offset

- **Decision**: Scroll is a function of the selection cursor: keep the cursor visible with minimal movement. `PageUp`/`PageDown` move the **cursor** by one viewport height; the offset follows.
- **Rationale**: The Tasks tab is fundamentally a single-selection list (`m.cursor`). An independent scroll offset (like the Report tab, which has no selection) would let the cursor and viewport diverge, which is confusing in a selection list and complicates every existing action that reads `m.visible[m.cursor]`. Cursor-driven scrolling guarantees SC-002 (selected task always visible) by construction.
- **Alternatives considered**:
  - Report-tab style independent offset — rejected: Report has no cursor; Tasks does.
  - Centering the cursor on every move — rejected: jarring; minimal-scroll is the conventional list behavior users expect.

## Decision 3 — Where the offset lives and how it's kept correct

- **Decision**: Store `listScroll int` on `Model`. Derive the effective window inside `renderList` via a pure `windowOffset(off, cursor, height, n)` helper, and also persist that reconciled value in the Update path (after `handleListKey`, on `WindowSizeMsg`, and after async `m.visible` rebuilds).
- **Rationale**: Deriving the window at render time makes "cursor always visible" robust to any missed persist site (correctness backstop). Persisting keeps paging/scroll **stable** across events and gives tests a field to assert on. This mirrors the established `reportState.scroll` pattern (stored offset, reconciled in handlers).
- **Alternatives considered**: Pure render-time derivation with no stored state — rejected: without a stored anchor, scroll stability after jumps is worse and paging is harder to reason about/test.

## Decision 4 — Key bindings for paging

- **Decision**: Add `PageUp` (`"pgup"`) and `PageDown` (`"pgdown"`) to `KeyMap`, surfaced in Tasks-tab `ShortHelp`/`FullHelp`.
- **Rationale**: The user explicitly requested `PgUp`/`PgDn`. `"pgup"`/`"pgdown"` are the exact key strings already handled by the goal-status reader (`internal/tui/update.go`), so behavior is consistent across surfaces (Constitution III). `home`/`end` (`First`/`Last`) already jump to the first/last task and will now scroll correctly for free.
- **Alternatives considered**: `ctrl+f`/`ctrl+b` — rejected: not requested, and `pgup`/`pgdown` match the existing reader convention.

## Decision 5 — Viewport height source

- **Decision**: A `listViewportHeight()` method on `Model` reproduces the pane-height arithmetic the view functions already compute (`m.height - {1|2} - statusHeight() - tabBarHeight`, min 1), branching on `m.styled` for the 1-line border difference.
- **Rationale**: Keeps the render window and the paging step size in agreement with the actual drawn area. The formula is already duplicated across view functions; extracting a single helper is a minor consolidation, not new complexity.
- **Alternatives considered**: Threading an explicit height parameter through `renderList` from each caller — rejected: more call-site churn than a single model helper, and every caller uses the same formula anyway.

## No open questions

All Technical Context fields are resolved; no `NEEDS CLARIFICATION` remain.
