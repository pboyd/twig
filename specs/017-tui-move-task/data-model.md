# Data Model: Change Task Parent in TUI

No new persisted entities. This feature reuses the existing `Task` row and the existing `UpdateTask` contract; the data model below captures only the *client-side* (Bubble Tea) state added for the move dialog.

## Persisted entity touched (existing)

### Task (existing — see `services/todo/proto/task/v1/task.proto` and `services/todo/db/`)

Fields relevant to this feature:

- `id` (int64) — identifier of the task being moved.
- `parent_id` (optional int64) — the field being written. Unset ⇒ top-level task.
- `completed_at` (timestamp, nullable) — used to filter the candidate-parent list to incomplete tasks only.

No schema migration required.

## Client-side state (new — added to `Model` in `services/todo/internal/tui/model.go`)

### `moveState` (new struct, lives in `move.go`)

| Field | Type | Purpose |
|---|---|---|
| `taskID` | `int64` | The task being moved. |
| `candidates` | `[]moveCandidate` | Ordered, indented list of candidate parents to render. Index 0 is always the "no parent" sentinel. |
| `cursor` | `int` | Index into `candidates` of the currently highlighted row. Initialized to the row matching the task's current `parent_id` (or 0 for "no parent"). |
| `errMsg` | `string` | Last error from `UpdateTask` to render in the dialog footer. Empty when no error. |

### `moveCandidate` (new struct)

| Field | Type | Purpose |
|---|---|---|
| `taskID` | `int64` | The candidate parent's id. `0` means the "no parent" sentinel row. |
| `label` | `string` | Display text (task name, or "(no parent)" for the sentinel). |
| `depth` | `int` | Indent level for tree rendering (0 for the sentinel and for root tasks). |

### Transitions

```
modeList --(press 'm' with a task selected)--> modeMove
modeMove --(navigation keys)--> modeMove (cursor moves)
modeMove --(Enter)--> RPC UpdateTask
    success --> modeList (list refreshes; cursor returns to moved task)
    failure --> modeMove (errMsg populated, cursor preserved)
modeMove --(Esc)--> modeList (no RPC call, no state change to task)
```

### Construction rules for `candidates`

1. Start with the "no parent" sentinel at index 0.
2. Walk the tree of incomplete tasks in the same order the main TUI list uses.
3. Skip the task being moved AND every descendant of it (prevents selecting a cycle-creating parent client-side).
4. Append each remaining task with its tree depth.

### Validation rules

- Pre-condition for entering `modeMove`: a task must be selected on the list view; otherwise `m` is a no-op.
- Server-side cycle and completed-parent rejection is still authoritative; the client-side filter in rule 3 reduces (but does not solely guarantee) correctness, in line with defense-in-depth.
