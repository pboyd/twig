# TUI Interaction Contract: Scrolling Task List

**Feature**: 064-scrolling-tasks | **Surface**: Tasks tab (interactive TUI)

This feature has no network/API/proto contract. Its user-facing contract is the keyboard interaction and viewport behavior of the Tasks list. This document is the source of truth that the implementation and tests must conform to.

## Key bindings (Tasks tab, `modeList`)

| Key(s) | Binding | Behavior | Status |
|--------|---------|----------|--------|
| `↑` / `k` | `Up` | Move selection up one row; scroll if it would leave the viewport | existing (scroll behavior new) |
| `↓` / `j` | `Down` | Move selection down one row; scroll if it would leave the viewport | existing (scroll behavior new) |
| `home` | `First` | Jump selection to the first task; viewport scrolls to top | existing (scroll behavior new) |
| `end` | `Last` | Jump selection to the last task; viewport scrolls so it is visible | existing (scroll behavior new) |
| `PgUp` | `PageUp` | Move selection up by ~one viewport height (clamped to first task) | **new** |
| `PgDn` | `PageDown` | Move selection down by ~one viewport height (clamped to last task) | **new** |

- New bindings use bubbletea key strings `"pgup"` and `"pgdown"` — identical to the goal-status reader, for cross-surface consistency.
- All other existing Tasks-tab bindings are unchanged.
- `PageUp`/`PageDown` MUST appear in the Tasks-tab help (`ShortHelp` and/or `FullHelp`) so they are discoverable.

## Viewport behavior contract

Let `n = len(visible)`, `h = listViewportHeight()`, `cursor` = selected index, `off` = top visible index.

1. **Cursor always visible**: after any key handled by the Tasks tab, `off <= cursor < off + h`.
2. **No overscroll top**: `off >= 0`.
3. **No overscroll bottom / no blank space**: `off <= max(0, n - h)`; the row after the last task is never shown as blank scrollable space.
4. **Fits-on-screen no-op**: if `n <= h`, then `off == 0` and the rendered output is byte-identical to the pre-feature output.
5. **Paging step**: `PageDown` moves `cursor` toward the end by `h` (clamped to `n-1`); `PageUp` moves toward the start by `h` (clamped to `0`). Reaching an already-first/last cursor is a no-op.
6. **Resize stability**: on terminal resize, rule 1 is re-established against the new `h`; the view never shows blank space below the last task.
7. **Length change stability**: when `visible` shrinks (filter, collapse, hide-completed), `off` is re-clamped so rules 2–3 still hold.

## Non-goals

- No horizontal scrolling.
- No independent scroll cursor separate from the selection.
- No change to the Report tab, Plan tab, Goals tab, or web frontend.
- No mouse-wheel handling (out of scope; keyboard-only).
