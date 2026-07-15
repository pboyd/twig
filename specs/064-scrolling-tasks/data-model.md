# Data Model: Scrolling Task List

**Feature**: 064-scrolling-tasks | **Date**: 2026-07-15

This feature introduces no persistent entities and no wire/DB schema. The only "data" is in-memory TUI view state.

## View-state entity

### Model.listScroll (new field)

| Attribute | Value |
|-----------|-------|
| Type | `int` |
| Location | `Model` struct, `internal/tui/model.go` |
| Meaning | Index into `m.visible` of the **top** row currently shown in the Tasks list viewport |
| Default | `0` (top of list) |
| Persisted? | No — transient; not written to the tree-state file |
| Invariant | Always kept in range `[0, max(0, len(m.visible) - listViewportHeight())]` and such that `m.cursor ∈ [listScroll, listScroll + listViewportHeight())` |

### Related existing state (unchanged)

| Field | Role |
|-------|------|
| `Model.visible []*visibleRow` | Flattened, filtered list of rows currently eligible for display |
| `Model.cursor int` | Index into `m.visible` of the selected task |
| `Model.height`, `Model.width`, `Model.styled` | Terminal dimensions / style mode — inputs to `listViewportHeight()` |

## Derived quantities (not stored)

### listViewportHeight() int

Number of task rows the list pane can display.

```
h = m.height - (styled ? 2 : 1) - statusHeight() - tabBarHeight
h = max(h, 1)
```

### windowOffset(off, cursor, height, n) int  (pure function)

Reconciles a stored offset to keep the cursor visible with minimal scrolling.

```
if cursor < off:            off = cursor
if cursor >= off + height:  off = cursor - height + 1
maxOff = max(0, n - height)
off = clamp(off, 0, maxOff)
return off
```

## State transitions

`listScroll` is recomputed via `windowOffset` (never set directly) whenever any input changes:

| Trigger | Effect on cursor | Effect on listScroll |
|---------|------------------|----------------------|
| `Up` / `Down` | ±1 (clamped) | re-derived to keep cursor visible |
| `First` (home) | → 0 | → 0 |
| `Last` (end) | → n−1 | → max offset |
| `PageUp` | −height (clamped) | re-derived (scrolls ~one screen up) |
| `PageDown` | +height (clamped) | re-derived (scrolls ~one screen down) |
| Expand / Collapse | may move to child/parent | re-derived |
| Complete / rank / refresh / toggle-all / filter apply (rebuild `m.visible`) | re-clamped via `findCursor`/`clampCursor` | re-derived; re-clamped if list shrank |
| `WindowSizeMsg` (resize) | unchanged | re-derived against new height |
| Empty list (`n == 0`) | n/a | 0 |

## Validation rules

- `listScroll` MUST never be negative and MUST never exceed `n - height` (no blank space below the last row) — enforced by `windowOffset` clamping.
- When `n <= height`, `listScroll` MUST be `0` (whole list fits; behavior identical to pre-feature).
