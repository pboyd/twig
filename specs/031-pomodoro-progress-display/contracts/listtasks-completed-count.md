# Contract: ListTasks completed-pomodoro count

**Status**: Defined before implementation (Principle II).

## Proto change (additive, backward-compatible)

Add a field to the existing `Task` message in `proto/task/v1/task.proto`:

```proto
message Task {
  int64 id = 1;
  string name = 2;
  string description = 3;
  google.protobuf.Timestamp due = 4;
  optional int64 parent_id = 5;
  google.protobuf.Timestamp completed_at = 6;
  int32 estimate = 7;
  // Number of completed pomodoros for this task by the calling user.
  // Read-only; computed server-side. 0 when none. Ignored on writes.
  int32 completed_pomodoro_count = 8;   // NEW
}
```

- **Field number 8** is unused today — safe to add.
- Additive field: existing clients (web SPA, current CLI) ignore it. No breaking change.
- Regenerate stubs with `make proto` (Go) and `npm run gen` is **not required** for backend; the web client is unaffected and need not regenerate unless it wants the field.

## Population semantics

| RPC | Behavior |
|-----|----------|
| `ListTasks` | Each returned `Task` has `completed_pomodoro_count` set to the count of that task's completed pomodoros for the calling user. Tasks with no completed pomodoros report `0`. |
| `GetTask` | Unchanged. `GetTaskResponse.completed_pomodoro_count` (existing top-level field) remains the source for single-task reads; setting it on the nested `Task` is optional and out of scope. |
| Write RPCs (`Create/Update/SetEstimate/...`) | `completed_pomodoro_count` is ignored on input and need not be populated on the returned `Task`. |

## New DB query

Add to `db/queries/pomodoro.sql`, then run `sqlc generate`:

```sql
-- name: CountCompletedPomodorosByTask :many
SELECT task_id, count(*)::bigint AS count
FROM pomodoros
WHERE user_id = $1 AND complete
GROUP BY task_id;
```

Returns one row per task that has ≥1 completed pomodoro. The `ListTasks` handler builds a `map[int64]int64` from these rows and assigns `completed_pomodoro_count` per task (default 0 when absent).

## Handler change (`internal/handler/task.go`, `ListTasks`)

1. Load tasks (existing `t.Queries.ListTasks(ctx, userID)`).
2. Load counts (`t.Queries.CountCompletedPomodorosByTask(ctx, userID)`); build `map[int64]int64`.
3. When mapping each db row → proto `Task`, set `CompletedPomodoroCount: int32(counts[task.ID])`.

Counts are bounded (estimate ≤ 10; completions are small integers), so `int32` is safe.

## Acceptance

- `ListTasks` for a user with a task that has 2 completed pomodoros returns that `Task` with `completed_pomodoro_count == 2`.
- A task with no completed pomodoros returns `completed_pomodoro_count == 0`.
- Existing `ListTasks` consumers continue to function unchanged.
