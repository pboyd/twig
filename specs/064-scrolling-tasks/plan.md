# Implementation Plan: Scrolling Task List

**Branch**: `064-scrolling-tasks` | **Date**: 2026-07-15 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/064-scrolling-tasks/spec.md`

## Summary

The Tasks-tab list currently renders **every** visible row and lets the surrounding pane clip the overflow from the top, so on long lists the selection cursor scrolls off-screen and lower tasks become unreachable. This feature adds a viewport window to the Tasks list: a scroll offset kept in the model, a pure "keep-cursor-visible" derivation applied at render time, and `PageUp`/`PageDown` bindings that move the selection by roughly one screen — mirroring the paging already present in the Report tab and goal-status reader.

## Technical Context

**Language/Version**: Go 1.23 (root CLI/TUI module `github.com/pboyd/twig`)

**Primary Dependencies**: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2/key` (already in use)

**Storage**: N/A — purely in-memory TUI view state; no persistence, no server or DB changes

**Testing**: `go test ./...` (root module); table-driven unit tests in `internal/tui/` using `export_test.go` shims

**Target Platform**: Terminal (TTY) on Linux/macOS

**Project Type**: Single-module CLI/TUI (no web, no backend changes)

**Performance Goals**: Instant redraw; windowing must render at most one screen of rows regardless of list length

**Constraints**: TUI-only change; no proto/API/DB/server modifications; must not alter behavior for lists that already fit on screen

**Scale/Scope**: Small, contained change in `internal/tui/` — one new model field, one shared derivation helper, `renderList` windowing, two new key bindings, and their tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One `listScroll int` field + one pure `windowOffset` helper reused by render and the persist step. No new abstraction layer; mirrors the existing `reportState.scroll` pattern. |
| II. API-First Design | ✅ | No request/response, proto, or shared-type changes. The only "contract" is the TUI interaction surface (keys + viewport behavior), documented in `contracts/tui-interaction.md` before implementation. |
| III. UI/UX Consistency | ✅ | Reuses existing `Up`/`Down`/`First`/`Last`; adds `PageUp`/`PageDown` consistent with the goal-status reader's paging. Help text (`ShortHelp`/`FullHelp`) updated so the new keys are discoverable. No new colors/styles. |
| IV. Playful User Messages | ✅ | No new user-facing prose required. Existing empty-state copy ("(no tasks)", "Nothing matches that filter — even the twigs came up bare.") is preserved unchanged. |

No violations — Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/064-scrolling-tasks/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output (manual verification)
├── contracts/
│   └── tui-interaction.md   # Phase 1 output (keybindings + viewport behavior contract)
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

All changes are confined to the TUI package:

```text
internal/tui/
├── model.go        # + listScroll int field on Model
├── view.go         # renderList: window rows via viewport height + windowOffset
├── keymap.go       # + PageUp / PageDown bindings; ShortHelp/FullHelp include them
├── update.go       # handleListKey: PgUp/PgDn cases move cursor by a page;
│                   #   persist reconciled listScroll after list-key dispatch,
│                   #   on WindowSizeMsg, and after async visible-rebuild handlers
├── view_test.go    # windowing / cursor-visibility render tests
├── update_test.go  # PgUp/PgDn + scroll-reconciliation tests
└── export_test.go  # expose helpers/field if needed for tests
```

**Structure Decision**: Single-module TUI change. No new files strictly required; a new pure helper (`windowOffset`) and a viewport-height helper (`listViewportHeight`) live in `view.go` alongside the existing `splitLines`/`padRightAnsi` helpers. Tests extend the existing `*_test.go` files.

## Design Overview

### Core mechanism — cursor-driven viewport

1. **State**: add `listScroll int` to `Model` (the top row index of the Tasks list viewport). Mirrors `reportState.scroll`.

2. **Viewport height**: add `func (m Model) listViewportHeight() int` returning the number of task rows the list pane can show, using the same arithmetic the view functions already use:
   - styled: `m.height - 2 - m.statusHeight() - tabBarHeight`
   - plain: `m.height - 1 - m.statusHeight() - tabBarHeight`
   - clamped to a minimum of 1.

3. **Pure derivation**: add `func windowOffset(off, cursor, height, n int) int` implementing minimal-scroll "keep cursor visible":
   - if `cursor < off` → `off = cursor`
   - if `cursor >= off + height` → `off = cursor - height + 1`
   - clamp to `[0, max(0, n-height)]`

   This guarantees the cursor is always in view **and** never scrolls past the end (no blank space below the last task — satisfies FR-002, FR-004).

4. **Render** (`renderList`): compute `h := m.listViewportHeight()`, `off := windowOffset(m.listScroll, m.cursor, h, len(m.visible))`, then iterate only rows `[off, off+h)` (absolute indices, so the `i == m.cursor` cursor styling still matches). Because `renderList` derives the window itself, the cursor is **always** visible even if a persist site is missed — the stored `listScroll` is only a stability hint. When the list fits (`n <= h`), `off` is 0 and output is identical to today (FR-005).

5. **Persist for stability/paging**: recompute and store `m.listScroll = windowOffset(...)` in the Update path so paging and scroll stay stable across events:
   - after `handleListKey` returns (wrap the `modeList` dispatch in `handleKey`),
   - in the `tea.WindowSizeMsg` handler (resize — FR-006),
   - in the async handlers that rebuild `m.visible` (refresh/complete/toggle-all/filter-apply) so the offset re-clamps when the list length changes (FR-006).

### Paging — PgUp / PgDn (FR-003, User Story 2)

- Add `PageUp` (`key: "pgup"`) and `PageDown` (`key: "pgdown"`) bindings to `KeyMap`; include them in Tasks-mode `ShortHelp`/`FullHelp`.
- In `handleListKey`, add cases that move `m.cursor` by `listViewportHeight()` rows (down for PgDn, up for PgUp), clamped via `clampCursor` to `[0, len(visible)-1]`. The viewport derivation then advances by ~one screen and stops cleanly at the top/bottom (FR-004 edge cases). Moving the **cursor** (rather than an independent scroll offset) keeps the Tasks tab's single-cursor model coherent and guarantees the selected task stays on screen.

### Edge cases covered

- **Empty / cursor-out-of-range**: `renderList` already early-returns on `len(m.visible) == 0`; `windowOffset` clamps with `n=0` → offset 0; paging is a no-op.
- **List exactly fills screen**: `n == h` → max offset 0 → no scroll, no blank space.
- **Resize**: WindowSize handler re-derives offset; a shorter terminal re-clamps so the cursor stays visible.
- **List shrinks (filter/collapse/hide-completed)**: rebuild handlers re-clamp offset into range; the render derivation is the backstop.

## Complexity Tracking

No constitution violations — table intentionally omitted.
