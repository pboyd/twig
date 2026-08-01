# Contract: Goal Behavior of `TaskService.UpdateTask`

**Service**: `task.v1.TaskService`
**RPC**: `UpdateTask(UpdateTaskRequest) → UpdateTaskResponse`
**Status**: Behavioral amendment. No message shape changes.

This contract governs what happens to `Task.goal_id` when `UpdateTask` changes a task's parent. It
is the source of truth for the server implementation and for what the TUI, CLI, and web clients may
assume.

## Message shape

Unchanged. For reference:

```proto
message UpdateTaskRequest {
  int64 id = 1;
  string name = 2;
  string description = 3;
  google.protobuf.Timestamp due = 4;
  // Unset clears the parent, making the task top-level.
  optional int64 parent_id = 5;
  google.protobuf.Timestamp snooze_until = 6;
}
```

`Task.goal_id` remains `optional int64`, read-only, ignored on writes. Clients cannot set it here;
the server derives it from the move.

## Full-replace semantics — required client behavior

`UpdateTask` is **full-replace on every editable field**, `parent_id` included. A client that omits
`parent_id` is asking to make the task top-level, not asking to leave the parent alone.

Clients MUST send the task's current `parent_id` on any update that is not intended as a move. This
is already what `buildUpdatePayload` (`services/twig-web/src/lib/updatePayload.ts`) and
`moveTaskCmd` (`internal/tui/update.go:539`) do.

## Definition: parent change

The server compares the requested `parent_id` against the value stored for the task.

- **No parent change** — requested equals stored (including both unset).
- **Descent** — requested is set, and differs from stored.
- **Promotion** — requested is unset, and stored is set.

Goal rules apply only on a descent or a promotion. This makes an ordinary rename safe.

## Guarantees

### G1 — A move is never refused for a goal reason

No combination of goal links on the moved task, its descendants, the destination, or the
destination's ancestors causes `UpdateTask` to fail. The `FAILED_PRECONDITION` error
`"moving this task would nest goal associations — clear the goal link first"` is **removed** and will
not be returned by any version implementing this contract.

### G2 — Descent clears goal links

On a descent, the moved task's `goal_id` becomes unset, and so does the `goal_id` of every
descendant of the moved task at any depth. This holds whether or not the destination has a goal, and
whether or not it is the same goal.

### G3 — Promotion preserves the displayed goal

On a promotion, if the moved task has no `goal_id` of its own, it is assigned the `goal_id` of its
nearest goal-bearing ancestor as measured **before** the move. If it had no goal-bearing ancestor, it
stays unset and this is not an error. If it already had its own `goal_id`, that value is kept.

### G4 — Closed goals are carried, not dropped

G3 applies even when the inherited goal is completed or archived. This is the one case where
`UpdateTask` produces a link that `SetTaskGoal` would refuse to create. Clients MUST render such a
link rather than treating it as invalid data.

### G5 — Non-move updates never touch goal links

When there is no parent change, `goal_id` is unchanged for the task and for every descendant,
regardless of what else the request modifies.

### G6 — Atomicity

The parent change, the position change, and all goal-link changes commit together. No client can
observe a state where the task has moved but goal links have not been reconciled. If any step fails,
nothing is applied.

### G7 — Position on any parent change

On a descent or a promotion, the task is placed at the end of its destination sibling group. This now
holds for promotions to the root group, which previously kept a stale position.

## Errors — unchanged

| Condition | Code | Behavior |
|---|---|---|
| Task not found | `NOT_FOUND` | No changes applied. |
| Destination task not found | `INVALID_ARGUMENT` | No changes applied. |
| Destination is the task or a descendant | `INVALID_ARGUMENT` | `"parent_id would create a cycle"`. No changes applied. |
| Destination is completed | `FAILED_PRECONDITION` | `"Task %d is already crossed off — nothing moves under a finished job."` No changes applied. |
| Name empty or over 255 chars | `INVALID_ARGUMENT` | No changes applied. |

## Unaffected: `SetTaskGoal`

`TaskService.SetTaskGoal` keeps its current rules in full. It still refuses to link a goal to a task
that has a parent, to a task with a goal-linked ancestor, to a task with a goal-linked descendant, or
to a completed or archived goal. This contract changes what *moves* do to goal links, not what
clients may request directly.

## Invariant clients may rely on

After any `UpdateTask` call, no task that has a parent holds its own `goal_id`.

Tasks that violated this before the change may still exist and are corrected when they or an ancestor
are next moved. Clients MUST continue to resolve a task's displayed goal by walking to the nearest
goal-bearing ancestor — the existing `internal/goal.EffectiveGoalID` and `src/lib/tree.ts` behavior
is correct and stays.
