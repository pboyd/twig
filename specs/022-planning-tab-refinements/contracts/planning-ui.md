# Contract: Planning + Tasks keybindings & view (post-change)

UI/interaction contract for the TUI after this feature. No network/proto contracts change; the merged Edit form reuses the existing **`RenamePlanEntry`** and **`MovePlanEntry`** RPCs unchanged.

## Keybindings

### Tasks tab

| Key | Action | Change |
|---|---|---|
| `enter` | Edit selected task | **was `e`** |
| `e` | (unbound for edit) | removed from edit |
| `n` / `ctrl+n` | New subtask / new root | unchanged |
| `ctrl+d` | Delete task | unchanged |
| `space` | Toggle complete | unchanged |
| `s` / `x` | Start / cancel pomodoro | unchanged |
| `m` | Move (change parent) | unchanged |
| `c` | Toggle completed filter | unchanged |
| `?` / `q` / `tab` | Help / quit / switch tab | unchanged |

### Planning tab

| Key | Action | Change |
|---|---|---|
| `t` | Add task (open picker) | **was `a`** |
| `e` | Add event | unchanged |
| `enter` | **Edit selected entry** (merged Name+Start+Duration) | **new** (replaces `r` rename + `m` move) |
| `r` | — | **removed** (folded into Edit) |
| `m` | — | **removed** (folded into Edit) |
| `c` | — | **removed** (clear deleted from TUI) |
| `ctrl+d` | Remove selected entry | unchanged |
| `[` / `]` | Prev / next day | unchanged |
| `.` | Jump to today | **was `t`** |
| `ctrl+r` | Refresh | unchanged |
| `x` | Cancel pomodoro | unchanged |
| `?` / `q` / `tab` | Help / quit / switch tab | unchanged |

**Invariant (FR-011)**: every advertised key maps to exactly one action per tab; no two advertised actions share a key.

## Edit entry flow (Planning)

```
[grid, entry selected] --enter--> [right pane: Edit form: Name / Start / Duration]
  Name      = entry.Name (prefilled)
  Start     = blank (placeholder shows current); blank = keep
  Duration  = blank = keep
--ctrl+s/enter--> validate:
  Name empty            → error "name cannot be empty", stay open
  Start unpar  (if set) → error, stay open
  Duration unparse(set) → error, stay open
  valid → run editPlanCmd:
     if Name changed     → RenamePlanEntry(day, id, name)
     if Start/Dur given  → MovePlanEntry(day, id, start, dur)
     then ListPlanEntries(day) highlight id
  → planList; right pane returns to details
--esc--> planList, no change, right pane returns to details
```

- Pressing `enter` with **no entry selected** (empty grid) opens nothing and shows no error.
- A save where neither name nor time changed is a no-op that just closes the form (no RPC).

## Add flows (Planning) — right pane

- **Add task** (`t`): `planPickTask` picker → on select, `planTaskTime` (Start/Duration) form. Both render in the **right pane** beside the grid.
- **Add event** (`e`): `planEventForm` (Name/Start/Duration) in the **right pane**.
- Save → apply + reload; cancel (`esc`) → return to details. Grid stays visible throughout.

## View contract

- Planning tab, `plan.mode == planList`: two panes — grid (left, focused/accent border) + details (right). *(unchanged from 021)*
- Planning tab, `plan.mode != planList`: two panes — grid (left, unfocused) + active picker/form (right, focused/accent border). **Replaces the prior full-width modal.**
- Styled path: `paneBox` + `lipgloss.JoinHorizontal`, tab bar on top, status line pinned bottom (mirrors `viewWithForm`). Non-styled path: `splitLines` + `padRightAnsi` row-join (mirrors the Tasks fallback).
- Narrow terminals degrade via the same split logic as the Tasks tab (no grid/form corruption).

## Selection highlight contract

- The selected entry's **content cell** is rendered with the Tasks tab's `highlightStyle` (bold + blue accent background + white foreground) — see `contracts/grid-selection.md`.
- The hour gutter, slot rails, and the entry's box border are **not** restyled by selection.
- Non-color terminals degrade to plain text with no stray escape codes.

## Help / status

- Planning and Tasks help screens and the status line derive from `KeyMap.ShortHelp()`/`FullHelp()`; updating the bindings updates all advertised text.
- Help/status MUST NOT mention `clear`, `rename`, or `move (entry)` on the Planning tab after this change; they MUST advertise `t` add-task, `enter` edit, and `.` today.
