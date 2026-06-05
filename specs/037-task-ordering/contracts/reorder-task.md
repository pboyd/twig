# Contract: Task ordering (position field + ReorderTask RPC)

Source of truth for the server and both write clients (TUI, web). Defined before
implementation per Constitution Principle II. To be added to
`api/proto/task/v1/task.proto` and regenerated with `make proto` (Go) and
`npm run gen` (web).

## 1. `Task.position` field

Add to the existing `Task` message:

```proto
message Task {
  // ... existing fields 1..8 ...

  // Order of this task within its sibling group (tasks sharing the same
  // parent_id, or the root group for parentless tasks). Lower sorts first.
  // Read-only: populated on reads, ignored on CreateTask/UpdateTask writes.
  int64 position = 9;
}
```

- Populated on every `Task` returned by `GetTask`, `ListTasks`, `CreateTask`,
  `UpdateTask`, `CompleteTask`, `UncompleteTask`, `SetEstimate`, and
  `ReorderTask`.
- Ignored if a client sets it on a write request.

## 2. `ReorderTask` RPC

Add to `service TaskService`:

```proto
// ReorderTask repositions a task within its current sibling group by placing it
// immediately before or after a sibling anchor. It never changes the task's
// parent. The whole sibling group is renumbered to a contiguous order.
rpc ReorderTask(ReorderTaskRequest) returns (ReorderTaskResponse);

message ReorderTaskRequest {
  // The task to move.
  int64 task_id = 1;

  // Where to place it, relative to a sibling anchor. Exactly one must be set.
  oneof anchor {
    // Place task_id immediately before this sibling.
    int64 before_task_id = 2;
    // Place task_id immediately after this sibling.
    int64 after_task_id = 3;
  }
}

message ReorderTaskResponse {
  // The moved task's sibling group (including the moved task), in the new
  // order, each with its updated position. Lets clients refresh without a
  // separate ListTasks round-trip if desired.
  repeated Task siblings = 1;
}
```

### Semantics

- The task is removed from its current slot and reinserted immediately
  before/after the anchor, then the entire `(user_id, parent_id)` group is
  renumbered to `0..n-1` (`position ASC, id ASC`).
- `parent_id` is **never** modified (FR-004). Re-parenting remains the job of
  `UpdateTask`.
- The operation runs in a single transaction over the sibling group so
  concurrent reorders converge with no task lost or duplicated (FR-015).
- Idempotent in effect when the requested placement already holds (the renumber
  produces the same order).

### Validation & errors (ConnectRPC codes)

| Condition | Code |
|---|---|
| `task_id` does not exist for the caller | `NotFound` |
| anchor task does not exist for the caller | `NotFound` |
| no anchor set, or both set | `InvalidArgument` |
| anchor equals `task_id` | `InvalidArgument` |
| anchor is not a sibling (different `parent_id`) | `InvalidArgument` |

### Authorization

Like all task RPCs, scoped to the authenticated user via `auth.Middleware`; a
caller can only reorder and anchor against their own tasks.

## 3. Client mapping (informative)

| Action | Request |
|---|---|
| TUI `{` (rank higher) | `ReorderTask(task, before_task_id = previous VISIBLE sibling)` — no-op if already first visible |
| TUI `}` (rank lower) | `ReorderTask(task, after_task_id = next VISIBLE sibling)` — no-op if already last visible |
| Web drag, drop above sibling Y | `ReorderTask(task, before_task_id = Y)` |
| Web drag, drop below sibling Y | `ReorderTask(task, after_task_id = Y)` |

The web mapping from a dnd-kit drop to a before/after anchor is implemented as a
pure, unit-tested helper (`src/lib/reorderAnchor.ts`).

## 4. Existing-RPC behavior changes

- `CreateTask`: assign `position = end of sibling group` (FR-012). No request
  shape change.
- `UpdateTask`: when the request changes `parent_id`, assign the moved task the
  `end` position of the destination group (FR-013). No request shape change.
- `ListTasks` / `GetTask`: unchanged request shape; responses now carry
  `position`.
