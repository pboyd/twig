# Contract: UpdateTask (reused — no changes)

This feature does not introduce a new contract. It consumes the existing `task.v1.TaskService.UpdateTask` RPC defined in `services/todo/proto/task/v1/task.proto`.

## RPC

```
rpc UpdateTask(UpdateTaskRequest) returns (UpdateTaskResponse);
```

## Request fields used by the move dialog

| Field | Type | How the TUI fills it |
|---|---|---|
| `id` | `int64` | The id of the task being moved. |
| `name` | `string` | The task's current name (unchanged). |
| `description` | `string` | The task's current description (unchanged). |
| `due` | `google.protobuf.Timestamp` | The task's current due (unchanged). |
| `parent_id` | `optional int64` | The id of the chosen parent. **Unset** when the user picks the "no parent" entry. |

Per the existing semantics in `task.proto`, an unset `parent_id` clears the parent and makes the task top-level.

## Response

`UpdateTaskResponse.task` — the updated task. The TUI uses this only to confirm success; it then refreshes the task list from `ListTasks`.

## Server-enforced invariants (already implemented)

- Parent must exist and belong to the same user (`handler/task.go:222-229`).
- Reparenting that would create a cycle is rejected with `InvalidArgument: "parent_id would create a cycle"` (`handler/task.go:231-237`).
- Parent must be incomplete; otherwise rejected with `InvalidArgument: "cannot move task under task N: parent is complete"` (`handler/task.go:238-245`).

## Error handling in the dialog

Any RPC error (cycle, completed-parent, parent-not-found, transport) is rendered as `errMsg` in the dialog footer; the dialog stays open and the cursor is preserved so the user can pick a different parent or press Esc to cancel. See FR-007 in `spec.md`.

## What is NOT in this feature

- No new fields on `UpdateTaskRequest`.
- No new RPC method.
- No new proto messages.
- No `make proto` regeneration step.
