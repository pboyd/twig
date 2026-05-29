# Contract: Planning Tab UI (keys + view)

The Planning tab's user-facing contract after this feature. "Same as Tasks" means the binding/behavior is shared with the Tasks tab and must be visually and functionally identical.

## Keybindings (planList mode)

| Key | Action | Status after this feature |
|---|---|---|
| `↑/k`, `↓/j` | Move selection between entries | unchanged |
| `[` / `]` | Previous / next day | unchanged |
| `t` | Jump to today | unchanged |
| `a` | Add task (opens picker) | picker now shows **sub-tasks** |
| `e` | Add event | unchanged |
| `r` | Rename selected entry | unchanged |
| `m` | Move selected entry | unchanged |
| `ctrl+d` | Remove selected entry | unchanged |
| `c` | Clear from time | unchanged |
| `ctrl+r` | Refresh | unchanged |
| `x` | **Cancel running pomodoro** | **now works** (was inert) |
| `?` | **Help** | **now opens the Planning help screen** (was inert) |
| `q` | Quit (confirm if pomodoro running) | quit-confirm `y`/`n` **now handled** on Planning |
| `tab` / `shift+tab` | Switch tab (blocked while a modal is open) | unchanged |

## Help screen contract

- `?` on the Planning tab opens a full-screen help view rendered by the shared `help.Model` (`viewHelp()`), styled identically to the Tasks help screen.
- Content is `KeyMap.FullHelp()` with `PlanningMode == true` (already implemented): navigation, day nav, add/rename/move/remove/clear, today, refresh, tab, help, quit. It MUST also surface pomodoro cancel where a pomodoro control is advertised.
- Dismiss (`?` or the help dismiss key) returns to the grid with the prior selection intact.

## Pomodoro / status-bar contract

- The shared status bar renders identically on both tabs (running timer with `[x] cancel`, completion banner, quit-confirm prompt).
- `x` cancels a running pomodoro from the Planning tab with the same effect as from Tasks; inert when nothing is running.
- The status line is pinned to the **bottom row** of the screen at all supported heights.

## Layout contract

- Two panes: **left** = day header + calendar grid; **right** = read-only details pane for the selected entry.
- Styled mode uses `paneBox` (rounded borders, accent for the focused/selected styling) + `JoinHorizontal`; non-styled mode uses the `splitLines`/`padRightAnsi` row join. Narrow terminals degrade gracefully (no layout corruption).
- The grid uses the shared **blue accent** theme for the selected entry (and box accents), not monotone bold.
- **No redundant pane title**: the left pane shows no `"Tasks"`/`"Planning"` title (the tab bar names the view). The right pane keeps its `"Details"` title; the Planning date header is retained.

## Details pane contract (read-only)

For the selected entry, show: name, `HH:MM–HH:MM` window, duration; for a task-linked entry also the linked task and completion state; events omit task-only fields. Empty day → placeholder, no error. Editing remains via the existing rename/move prompts (the pane is not an inline editor).

## Non-regression contract

All previously specified Planning behaviors (grid rendering, now-marker auto-advance, add/edit/remove/clear, day navigation, refresh, completion strike-through, modal tab-switch blocking) continue to function unchanged.
