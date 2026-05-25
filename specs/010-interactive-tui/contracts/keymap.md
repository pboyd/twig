# Contract: TUI Keymap

This is the authoritative keybinding contract. Every keybinding implemented in `internal/tui/keymap.go` MUST appear here, and every entry here MUST be discoverable via the in-app help screen (`?`).

## Modes

| Mode | When active |
|---|---|
| `modeList` | Default — task tree focused. |
| `modeEdit` / `modeNewSubtask` / `modeNewRoot` | An edit form is focused. |
| `modeHelp` | Help overlay is shown. |
| `modePomodoro` | An in-TUI pomodoro is running. |

## Keys — modeList

### Navigation

| Key | Action |
|---|---|
| `↑` / `K` | Highlight previous visible task |
| `↓` / `J` | Highlight next visible task |
| `←` / `H` | Collapse highlighted task's subtree |
| `→` / `L` | Expand highlighted task's subtree |

### Mutations on highlighted task

| Key | Action |
|---|---|
| `E` | Open edit form for highlighted task |
| `N` | Open edit form for a new subtask of highlighted task |
| `Ctrl-N` | Open edit form for a new root task |
| `Ctrl-D` | Delete highlighted task immediately (no confirm) |
| `Space` | Toggle completion of highlighted task |
| `0`–`9` | Set pomodoro estimate of highlighted task to the digit value |

### Pomodoro

| Key | Action |
|---|---|
| `S` | Start pomodoro on highlighted task; return to list on completion |
| `R` | Resume backgrounded pomodoro |

### Global

| Key | Action |
|---|---|
| `C` | Toggle completed-task visibility (preserve highlight by task id; fall back to first visible) |
| `Ctrl-R` | Full reload of the task tree from the server (preserve highlight by task id) |
| `?` | Open help overlay |
| `Q` | Quit |

## Keys — modeEdit / modeNewSubtask / modeNewRoot

| Key | Action |
|---|---|
| `Tab` / `Shift-Tab` | Cycle focus through fields and Save/Cancel buttons |
| `Enter` (on Save button) | Save and return to list |
| `Enter` (on Cancel button) | Cancel and return to list |
| `Ctrl-S` | Save from any field |
| `Esc` | Cancel from any field |

## Keys — modeHelp

| Key | Action |
|---|---|
| `?` | Dismiss help (toggle) |
| `Esc` | Dismiss help |

## Keys — modePomodoro

Pomodoro mode delegates to the existing `internal/cli` countdown component; its keymap is unchanged from the current `todo pom start` behavior. When the pomodoro ends (naturally or via the user's existing keys), the TUI returns to `modeList`.

## Highlight rules after mutations / mode transitions

These rules are part of the contract — implement them exactly:

| Action | Highlight after |
|---|---|
| `E` Save | Same task (unchanged id) |
| `E` Cancel | Same task |
| `N` Save | The newly created subtask; parent expanded if needed |
| `N` Cancel | The original task |
| `Ctrl-N` Save | The new root task |
| `Ctrl-N` Cancel | The original task |
| `Ctrl-D` | Next visible task; if none, previous visible task; if list empty, no highlight |
| `Space` (filter hides completed, task just completed) | The just-completed task stays highlighted with completed styling; on next `↑`/`↓` it disappears from list |
| `Space` (otherwise) | Same task |
| `S` end | Same task |
| `0`–`9` | Same task |
| `C` toggle | Same task if still visible; else first visible task |
| `Ctrl-R` | Same task if still present after reload; else first visible task |
| `?` dismiss | Same task that was highlighted before `?` |

## Conflict resolution log

- The source spec listed `H` for both "collapse subtree" and "show help". Resolved by binding help to `?` (and `Esc` for symmetric dismiss); `H` is collapse only.
- The source spec used `H` (single char) for "show help". Bound to `?` per clarification Q3.
