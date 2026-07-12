# Contract: FilterTasks RPC

**Feature**: 062-task-filters | **Service**: `task.v1.TaskService` | **File to change**: `api/proto/task/v1/task.proto` (regenerate with `make proto`)

This contract is the source of truth (Principle II). Implementation must conform to it.

## Proto additions

```proto
service TaskService {
  // ... existing RPCs unchanged ...

  // FilterTasks evaluates a filter expression (see the filter grammar
  // contract) against the calling user's tasks and returns the ids of the
  // tasks that match. It never returns ancestor-context rows — deciding how
  // to display matches is the client's job.
  //
  // Errors:
  //   InvalidArgument — the expression does not parse or fails validation
  //                     (unknown field, bad operator, malformed date, ...).
  //                     The message is human-readable and safe to display.
  rpc FilterTasks(FilterTasksRequest) returns (FilterTasksResponse);
}

message FilterTasksRequest {
  // The filter expression. Must be non-empty; clients treat an empty input
  // as "no filter" and simply don't call FilterTasks.
  string expression = 1;

  // The client's "show all" state. When false and the expression contains
  // no completed (resp. snoozed) condition, an implicit completed=false
  // (resp. snoozed=false) condition is applied. Explicit conditions in the
  // expression replace only their own attribute's implicit default.
  bool show_all = 2;

  // The client's current local calendar day, formatted YYYY-MM-DD.
  // Required. Used to evaluate snoozed: a task is snoozed when its
  // snooze_until day is strictly after this day (same rule clients already
  // use for default views).
  string today = 3;
}

message FilterTasksResponse {
  // Ids of the calling user's tasks that satisfy the expression, ascending.
  // Empty when nothing matches (including when an id in the expression
  // references a task or goal that doesn't exist — that is not an error).
  repeated int64 task_ids = 1;
}
```

## Behavioral guarantees

1. **Purity**: same expression + same task state + same `show_all`/`today` ⇒ same `task_ids`, regardless of client (SC-005).
2. **User scoping**: only the authenticated caller's tasks are consulted or returned; ids belonging to other users behave exactly like nonexistent ids.
3. **Nonexistent references**: `parent_id=99999` (no such task) yields an empty result, not an error (FR-012).
4. **Invalid expressions**: any parse/validation failure returns ConnectRPC `InvalidArgument` with an actionable, tone-compliant message (Principle IV). No other error code is used for expression problems.
5. **No pagination**: result is the complete matched-id set (task lists are small; consistent with `ListTasks`).
6. **`ListTasks` unchanged**: existing RPCs keep their exact contracts.

## Client obligations (TUI now, web later)

- Render matched tasks **plus their ancestor chains** (from the client-held tree), preserving structure; hide non-matching descendants of matches (FR-003).
- Treat `InvalidArgument` during progressive typing as a soft state: keep the last valid result set, show an invalid indicator (FR-009).
- Re-fire the current expression after any task mutation (FR-013).
- Send the user's current local day as `today` on every call.
