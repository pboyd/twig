# API Contract: Task Completion

This document is the source of truth for the proto changes introduced by
feature 004. The proto file at `services/todo/proto/task/v1/task.proto` will
be edited to match this contract before the handler or CLI code is changed.

## Proto delta

```proto
// services/todo/proto/task/v1/task.proto
syntax = "proto3";

package task.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/pboyd/todo/services/todo/gen/task/v1;taskv1";

service TaskService {
  rpc CreateTask(CreateTaskRequest) returns (CreateTaskResponse);
  rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
  rpc UpdateTask(UpdateTaskRequest) returns (UpdateTaskResponse);
  rpc DeleteTask(DeleteTaskRequest) returns (DeleteTaskResponse);

  // CompleteTask marks the identified task complete and records the completion
  // moment. It is idempotent: re-completing a task is a successful no-op that
  // returns the row unchanged with its original completed_at.
  //
  // Errors:
  //   NotFound          — no task exists with the given id (for this user)
  //   FailedPrecondition — the task has at least one incomplete descendant
  rpc CompleteTask(CompleteTaskRequest) returns (CompleteTaskResponse);
}

message Task {
  int64  id           = 1;
  string name         = 2;
  string description  = 3;
  google.protobuf.Timestamp due = 4;
  optional int64 parent_id      = 5;
  // NEW. Unset when the task is incomplete; set to the moment of completion
  // (UTC) when complete. Once set, it never changes.
  google.protobuf.Timestamp completed_at = 6;
}

message CompleteTaskRequest {
  // Identifier of the task to mark complete.
  int64 id = 1;
}

message CompleteTaskResponse {
  // The task in its post-completion state. For a task that was already
  // complete, this is the unchanged row.
  Task task = 1;
}
```

Field numbers `1`–`5` on `Task` are preserved exactly — no existing client
breaks from this change. The new `completed_at` field takes number `6`.

## Request / response semantics

### `CompleteTask`

| Concern | Behavior |
|---------|----------|
| Idempotency | If `task.completed_at` is already set, the RPC returns successfully with the row unchanged. |
| Subtask check | Before mutating, the server verifies no descendant of the task (at any depth) has `completed_at` unset. If any descendant is incomplete, the RPC fails with `FailedPrecondition` and the error message identifies at least one blocking descendant by id. |
| Timestamp source | The server records `NOW()` at the moment of the SQL update; the client does not supply a timestamp. |
| Authorization | Same as every other task RPC — the request must carry a valid credential resolving to a `user_id`, and only that user's tasks are visible or mutable. |
| Not found | A task id that does not exist for the calling user returns `NotFound`. The same code is returned whether the row genuinely does not exist or it belongs to another user. |

### Existing RPCs — behavioral additions

`CreateTask` and `UpdateTask` get one new failure mode each. Their wire
contracts (request/response shapes) are unchanged.

| RPC | New failure |
|-----|-------------|
| `CreateTask` | If `parent_id` is set and the parent's `completed_at` is non-null, the RPC fails with `FailedPrecondition` and the error message states that the parent task is complete. No row is created. |
| `UpdateTask` | If `parent_id` is set in the request and the proposed parent's `completed_at` is non-null, the RPC fails with `FailedPrecondition` with the same message. No row is updated. |

`ListTasks` is wholly unchanged on the wire. The server returns every task
for the calling user, each carrying its `completed_at` field; the CLI applies
the user-visible filter modes (default, `--completed`, `--all`) and the
default-mode subtree pruning during render.

## CLI contract delta

The CLI is not a wire contract, but it is the externally visible interface of
this feature and is captured here for completeness.

```text
# NEW subcommand
todo task complete <id>

# MODIFIED subcommand — accepts two mutually-exclusive flags
todo task list                 # default: incomplete tasks only,
                               #   pruning any fully-complete subtree
todo task list --completed     # completed tasks only
todo task list --all           # every task, with completion status visible
```

Render contract for every list mode:

- Each line is prefixed with a checkbox marker: `[ ]` for incomplete, `[x]`
  for complete.
- A completed task's line ends with ` (completed <RFC3339-UTC>)`.
- Within each parent, kept children appear in id order.
- Tree connectors (`├──`, `└──`, `│   `) follow the existing rendering rules
  from feature 002.

Confirmation lines from mutations follow the existing pattern:

- `completed task <id>` on a fresh completion.
- `task <id> already complete (at <RFC3339-UTC>)` on an idempotent re-complete.

Exit codes follow the existing CLI convention: `0` on success (including
idempotent re-complete), `1` on every error (not found, blocked by
incomplete descendants, malformed id, conflicting flags, transport error).
