# Phase 1 Data Model: Interactive TUI

The TUI does not introduce new persistent entities. The only "data model" here is the in-memory session state that the Bubble Tea `Model` holds between messages. All persistent entities (Task, PomodoroSession) belong to the existing server schema and are not changed by this feature.

## In-memory session state

### `Model` (the root Bubble Tea model)

| Field | Type | Description |
|---|---|---|
| `client` | `taskv1connect.TaskServiceClient` | Existing ConnectRPC client. |
| `pomClient` | `pomodorov1connect.PomodoroServiceClient` | Existing ConnectRPC client. |
| `tree` | `[]*treeNode` | Root nodes of the task tree, in `todo task list` order (built via `internal/cli.BuildTree`). |
| `visible` | `[]*visibleRow` | Flattened, depth-aware list of currently-visible rows; recomputed whenever expansion state, tree, or filter changes. |
| `cursor` | `int` | Index into `visible`. The highlighted task is `visible[cursor]`. |
| `expanded` | `map[int64]bool` | Task ID → whether that task's subtree is expanded. Default is `false` (collapsed). |
| `showCompleted` | `bool` | Filter toggle: `false` (default) hides completed tasks; `true` shows all. |
| `pendingComplete` | `*int64` | Task ID that was just toggled to "completed" while `showCompleted=false`. Held visible (with completed styling) until the cursor moves away, then dropped. |
| `mode` | `viewMode` | `modeList`, `modeEdit`, `modeNewSubtask`, `modeNewRoot`, `modeHelp`, `modePomodoro`. |
| `edit` | `editFormModel` | The edit form state (only meaningful in edit-related modes). |
| `help` | `help.Model` | Bubbles help component instance. |
| `err` | `error` | Last user-facing error (server failure, validation), shown in a status line. |
| `width`, `height` | `int` | Terminal dimensions, updated on `tea.WindowSizeMsg`. |

### `treeNode`

Reused from `internal/cli` (existing structure populated by `buildTree`). Adds no fields.

| Field | Type | Description |
|---|---|---|
| `task` | `*taskv1.Task` | The protobuf task. |
| `children` | `[]*treeNode` | Sorted child nodes. |

### `visibleRow`

| Field | Type | Description |
|---|---|---|
| `node` | `*treeNode` | The task displayed on this row. |
| `depth` | `int` | Indent level (0 = root). |
| `treePrefix` | `string` | The `├── ` / `└── ` / indentation string for this row. |
| `marker` | `string` | `[+]` or `[-]` for this row. |

### `editFormModel`

| Field | Type | Description |
|---|---|---|
| `taskID` | `*int64` | `nil` for new tasks; set when editing an existing task. |
| `parentID` | `*int64` | `nil` for root tasks; set for subtasks (existing or new). |
| `name` | `textinput.Model` | Single-line. |
| `description` | `textarea.Model` | Multi-line. |
| `due` | `textinput.Model` | Free-text date, parsed with `internal/cli.parseDue`. |
| `pomodoroEstimate` | `textinput.Model` | Free-text integer, numeric validation on save. |
| `focusIndex` | `int` | 0..N-1 selects the focused field; N selects "Save"; N+1 selects "Cancel". |
| `originalCursor` | `int` | Cursor index to restore on cancel for "new" forms. |

## Derived computations

These are not state — they are pure functions of state:

- **`buildVisible(tree, expanded, showCompleted, pendingComplete) []*visibleRow`** — depth-first walk of `tree`, skipping subtrees of collapsed nodes, skipping completed tasks unless `showCompleted` or `node.task.id == *pendingComplete`. Emits `[+]` when a task has children and `expanded[id]==false`; emits `[-]` otherwise. Tree-line prefix is computed during the walk using the same `├──`/`└──`/`│  `/`   ` conventions as `internal/cli.renderTree`.
- **`renderDetails(task) string`** — formats the right pane: id, name (with strikethrough+dim if completed), due (via `formatDue`), description, pomodoro estimate. Reuses existing formatters.

## State transitions

```
modeList ──E──> modeEdit         (Save→modeList, same task highlighted)
                                 (Cancel→modeList, same task highlighted)

modeList ──N──> modeNewSubtask   (Save→modeList, new subtask highlighted, parent expanded)
                                 (Cancel→modeList, original task highlighted)

modeList ──Ctrl-N──> modeNewRoot (Save→modeList, new root task highlighted)
                                 (Cancel→modeList, original task highlighted)

modeList ──S──> modePomodoro     (on completion→modeList, same task highlighted)

modeList ──?──> modeHelp         (?/Esc→modeList, original cursor)

modeList ──Space──> modeList     (mutation only; pendingComplete may be set)

modeList ──Ctrl-D──> modeList    (mutation only; cursor moves to next or previous visible row)

modeList ──Ctrl-R──> modeList    (full reload; cursor preserved by task id when possible)

modeList ──C──> modeList         (filter toggle; cursor preserved by task id when possible,
                                  else falls back to first visible)

modeList ──0..9──> modeList      (mutation only; sets pomodoroEstimate on highlighted task)

modeList ──Q──> exit             (only in modeList; ignored in other modes)
```

## Validation rules

Validation is delegated to existing helpers and server behavior:

- **Due date**: parsed via `internal/cli.parseDue` on form save; on parse failure, the form stays open and `err` is set.
- **Name**: required and non-empty (server-side validation; surface error in `err` line on failed save).
- **Pomodoro estimate (digit keys & form field)**: must be a non-negative integer in the range the existing API accepts (0–9 from keys; form field allows free input, validated on save).
- **Completion toggle**: relies on the server's existing toggle endpoint; no client-side validation.

## Persistence boundary

All state outside the `Model` struct is owned by the server. The TUI:
- Reads via `TaskService.ListTasks` at launch, after every mutation, and on `Ctrl-R`.
- Writes via `TaskService.CreateTask`, `UpdateTask`, `DeleteTask`, `CompleteTask` (or whatever the existing handlers are named), and `PomodoroService` methods.

No client-side caching beyond the current `Model.tree` snapshot.
