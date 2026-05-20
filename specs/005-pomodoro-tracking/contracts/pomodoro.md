# Contract: Pomodoro Tracking (005)

This document is the source of truth for the proto delta and RPC contracts added by feature 005. The implementation phase edits `services/todo/proto/task/v1/task.proto` to match this document verbatim; `buf generate` regenerates the Go bindings.

## Proto delta

All additions are within `package task.v1`, `service TaskService`, and existing message scope. No existing field numbers change.

### Modified: `Task`

```proto
message Task {
  // ... existing fields 1–6 unchanged ...

  // User's estimated number of pomodoros to complete this task.
  // 0 means "no estimate" (default). Server enforces 0 <= estimate <= 10.
  int32 estimate = 7;
}
```

### New: `Pomodoro`

```proto
message Pomodoro {
  int64 id = 1;
  int64 task_id = 2;

  // Moment the pomodoro started (UTC).
  google.protobuf.Timestamp start_at = 3;

  // Unset while the pomodoro is active. Set to start_at + 25 minutes
  // for a completed pomodoro, or to the moment of cancellation for a
  // canceled one.
  google.protobuf.Timestamp end_at = 4;

  // True only for completed pomodoros (timer ran to expiry).
  // False for active and canceled pomodoros.
  bool complete = 5;
}
```

### Modified: `GetTaskResponse`

```proto
message GetTaskResponse {
  Task task = 1;

  // Number of pomodoros that have run to completion for this task.
  int64 completed_pomodoro_count = 2;

  // All pomodoros (active, completed, canceled) for this task,
  // ordered by start_at ascending.
  repeated Pomodoro pomodoros = 3;
}
```

`GetTask`'s request shape is unchanged. `ListTasks`, `CreateTaskResponse`, `UpdateTaskResponse`, and `CompleteTaskResponse` are unchanged in shape (they continue to return only `Task` — which now carries `estimate`).

### New RPCs on `TaskService`

```proto
service TaskService {
  // ... existing RPCs unchanged ...

  // SetEstimate overwrites a task's estimate.
  rpc SetEstimate(SetEstimateRequest) returns (SetEstimateResponse);

  // StartPomodoro creates a new active pomodoro for the calling user
  // against the named task.
  rpc StartPomodoro(StartPomodoroRequest) returns (StartPomodoroResponse);

  // CancelPomodoro cancels the calling user's active pomodoro.
  rpc CancelPomodoro(CancelPomodoroRequest) returns (CancelPomodoroResponse);

  // CompletePomodoro marks the calling user's active pomodoro complete.
  rpc CompletePomodoro(CompletePomodoroRequest) returns (CompletePomodoroResponse);

  // GetActivePomodoro returns the calling user's active pomodoro, if any.
  rpc GetActivePomodoro(GetActivePomodoroRequest) returns (GetActivePomodoroResponse);
}

message SetEstimateRequest {
  int64 task_id = 1;
  int32 estimate = 2;
}
message SetEstimateResponse {
  Task task = 1;
}

message StartPomodoroRequest {
  int64 task_id = 1;
}
message StartPomodoroResponse {
  Pomodoro pomodoro = 1;
}

message CancelPomodoroRequest {}
message CancelPomodoroResponse {
  Pomodoro pomodoro = 1;
}

message CompletePomodoroRequest {}
message CompletePomodoroResponse {
  Pomodoro pomodoro = 1;
}

message GetActivePomodoroRequest {}
message GetActivePomodoroResponse {
  // Unset when the user has no active pomodoro.
  Pomodoro pomodoro = 1;
}
```

## RPC contracts

All requests are authenticated via the existing bearer-token interceptor; "the user" below means the authenticated user. Unless otherwise stated, the failure modes also include `Unauthenticated` (missing/invalid token) and `Internal` (unexpected DB error).

### `SetEstimate`

- **Request**: `task_id` (existing task owned by the user); `estimate` (int32).
- **Behavior**: Sets `tasks.estimate` for the task. Overwrites any prior value.
- **Response**: The full `Task` with the new `estimate`.
- **Errors**:
  - `InvalidArgument` — `estimate < 0` or `estimate > 10`. Error message: `"estimate must be between 0 and 10; tasks larger than 10 pomodoros must be broken down further"`.
  - `NotFound` — no task with that id is owned by the user.

### `StartPomodoro`

- **Request**: `task_id` (existing task owned by the user).
- **Behavior**: Inserts a new pomodoro row with `start_at = NOW()`, `end_at = NULL`, `complete = false`.
- **Response**: The created `Pomodoro`.
- **Errors**:
  - `NotFound` — no task with that id is owned by the user.
  - `AlreadyExists` — the user already has an active pomodoro (any task). The error detail includes the active pomodoro's `task_id` so clients can present "you have an active pomodoro on task N" without an extra round-trip.

### `CancelPomodoro`

- **Request**: empty.
- **Behavior**: Sets `end_at = NOW()` (`complete` remains false) on the user's active pomodoro.
- **Response**: The updated `Pomodoro`.
- **Errors**:
  - `FailedPrecondition` — the user has no active pomodoro.

### `CompletePomodoro`

- **Request**: empty.
- **Behavior**: Sets `end_at = start_at + 25 minutes` and `complete = true` on the user's active pomodoro. The server computes `end_at` from its own `start_at` plus the constant; the client never supplies a timestamp.
- **Response**: The updated `Pomodoro`.
- **Errors**:
  - `FailedPrecondition` — the user has no active pomodoro.

### `GetActivePomodoro`

- **Request**: empty.
- **Behavior**: Returns the user's active pomodoro, if any.
- **Response**: `pomodoro` is unset when none is active; otherwise it is the active row. This is **not** an error — clients use the unset/set distinction directly.
- **Errors**: none beyond the universal ones.

### `GetTask` (extended response)

- **Request**: unchanged.
- **Response now also includes**:
  - `completed_pomodoro_count` — number of completed pomodoros for this task.
  - `pomodoros` — full list (active, completed, canceled) for this task, ordered by `start_at` ascending.
- **Backward compatibility**: New fields use new field numbers; existing field 1 (`task`) is unchanged. Old clients that ignore unknown fields continue to work.

## Cascade semantics

`DeleteTask` is unchanged on the wire, but its database-level effect now also cascades to `pomodoros`: deleting a task removes every `pomodoros` row that references it (active, completed, or canceled). This is implemented by the FK declaration `task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE`.

## Error-code summary

| RPC | Possible non-universal errors |
|---|---|
| `SetEstimate` | `InvalidArgument` (out of range), `NotFound` (task) |
| `StartPomodoro` | `NotFound` (task), `AlreadyExists` (single-active invariant) |
| `CancelPomodoro` | `FailedPrecondition` (no active pomodoro) |
| `CompletePomodoro` | `FailedPrecondition` (no active pomodoro) |
| `GetActivePomodoro` | (none — empty response when no active pomodoro) |
| `GetTask` (extended) | unchanged from feature 001 |
