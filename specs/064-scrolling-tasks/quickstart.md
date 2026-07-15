# Quickstart: Scrolling Task List

**Feature**: 064-scrolling-tasks

## Build & run

```bash
# From repo root
go build -o twig ./cmd/twig
./twig            # launches the TUI on a TTY, opening on the Tasks tab
```

Requires `TWIG_API_KEY` and (optionally) `TWIG_ADDR`, or a `~/.config/twig/config.toml`. See CLAUDE.md for provisioning against a local `make dev` server.

## Manual verification

Use a **short terminal** (e.g. resize to ~15 rows) and an account with many tasks. Press `c` to show completed tasks so the list is comfortably taller than the screen.

1. **Cursor stays visible going down** (US1)
   - Put the cursor on the top task, hold `↓` (or `j`).
   - ✅ The list scrolls as the cursor reaches the bottom; the selected row is always on screen.

2. **Cursor stays visible going up** (US1)
   - From deep in the list, hold `↑` (or `k`) back to the top.
   - ✅ The list scrolls up; the selected row never disappears; it stops at the first task.

3. **Reach the last task** (US1)
   - Press `end`.
   - ✅ The last task is selected and visible; no blank space below it.

4. **Page down / up** (US2)
   - From the top, press `PgDn` repeatedly.
   - ✅ The view advances ~one screen each press and stops at the last task.
   - Press `PgUp` repeatedly.
   - ✅ The view moves back ~one screen each press and stops at the first task.

5. **Short list unchanged** (US1 AC3 / FR-005)
   - Collapse/filter so the list fits on screen (`/` to filter, or collapse subtrees with `←`).
   - ✅ No scrolling occurs; navigation looks exactly as before.

6. **Resize** (edge case)
   - Scroll partway down, then make the terminal shorter and taller.
   - ✅ The selected task stays visible; no blank space appears below the last task.

7. **List shrinks** (edge case)
   - Scroll to the bottom of a long completed list, then press `c` to hide completed.
   - ✅ The view re-clamps to a valid position (no stranded blank view).

## Automated tests

```bash
go test ./internal/tui/...
```

Expected coverage (added by this feature):
- `windowOffset` unit table: cursor above/below window, clamps at both ends, `n <= h` → 0.
- `renderList` renders only one viewport of rows and always includes the cursor row for representative offsets.
- `handleListKey` `PageUp`/`PageDown` move the cursor by ~one page and clamp at the ends.
- Resize (`WindowSizeMsg`) and list-shrink paths leave `listScroll` in range.
