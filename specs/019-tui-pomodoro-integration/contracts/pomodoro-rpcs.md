# Contract: Pomodoro RPCs (reused — no changes)

This feature introduces **no new contract**. It consumes existing, already-implemented RPCs on `task.v1.TaskService`, defined in `services/todo/proto/task/v1/task.proto`. No `make proto` regeneration is required.

## RPCs used by the TUI

```text
rpc StartPomodoro(StartPomodoroRequest) returns (StartPomodoroResponse);
rpc CancelPomodoro(CancelPomodoroRequest) returns (CancelPomodoroResponse);
rpc CompletePomodoro(CompletePomodoroRequest) returns (CompletePomodoroResponse);
rpc GetActivePomodoro(GetActivePomodoroRequest) returns (GetActivePomodoroResponse);
rpc GetTask(GetTaskRequest) returns (GetTaskResponse);   // resolve task name when auto-attaching
```

## Messages (existing)

```text
message Pomodoro {
  int64 id = 1;
  int64 task_id = 2;
  google.protobuf.Timestamp start_at = 3;   // authoritative; remaining derived locally
  google.protobuf.Timestamp end_at = 4;
  bool complete = 5;
}

message StartPomodoroRequest  { int64 task_id = 1; }
message StartPomodoroResponse { Pomodoro pomodoro = 1; }
message CancelPomodoroRequest  {}
message CancelPomodoroResponse { Pomodoro pomodoro = 1; }
message CompletePomodoroRequest  {}
message CompletePomodoroResponse { Pomodoro pomodoro = 1; }
message GetActivePomodoroRequest  {}
message GetActivePomodoroResponse { Pomodoro pomodoro = 1; }   // pomodoro is nil/absent when none active
```

## How the TUI uses each call

| Call | When | TUI behavior |
|---|---|---|
| `GetActivePomodoro` | TUI launch (`Init`) | If `pomodoro` present, seed `activePom` (derive remaining from `start_at`) and start ticking; else idle status bar. |
| `StartPomodoro` | user presses `s` | On success seed `activePom`, fire `on_start`, start ticking. See conflict handling below. |
| `CancelPomodoro` | user presses `x`, or as the first half of a different-task start | Clear `activePom`, fire `on_cancel`. |
| `CompletePomodoro` | tick observes remaining == 0 | Mark completed, fire `on_complete`, show banner. |
| `GetTask` | auto-attach when the task name is not already in the loaded tree | Resolve the task's display name for the status line. |

## Start conflict semantics (existing server behavior)

`StartPomodoro` returns `connect.CodeAlreadyExists` when a pomodoro is already active. The active task id is carried in the error detail as a `StartPomodoroRequest` (read via the existing `extractActiveTaskID` helper). The TUI then applies the non-interactive policy (R4 / FR-010):

- **same task** → attach to the running pomodoro (no restart).
- **different task** → `CancelPomodoro`, then `StartPomodoro` for the new task.

## Error handling in the TUI

Any RPC error (`StartPomodoro`, `CancelPomodoro`, `CompletePomodoro`, `GetActivePomodoro`, `GetTask`) is surfaced through the existing `Model.err` status-area path and rendered via `cli.UserMessage`; the TUI stays usable (FR-015). Errors are never written raw to the terminal.

## What is NOT in this feature

- No new RPC methods, messages, or fields.
- No proto changes and no `make proto` step.
- No server-side, handler, db, or migration changes.
- No change to the CLI `todo pom` subcommands' use of these RPCs.
