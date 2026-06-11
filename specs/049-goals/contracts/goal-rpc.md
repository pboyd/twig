# API Contract: GoalService + task association

**Feature**: 049-goals | **Status**: contract — commit before implementation

Two pieces: a new proto package `goal/v1` for the goal object, and small
additions to `task.v1` for the task-side association. Regenerate stubs with
`make proto`.

## New file: `api/proto/goal/v1/goal.proto`

```proto
syntax = "proto3";

package goal.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/pboyd/twig/api/gen/goal/v1;goalv1";

// GoalService manages long-term goals. Goals never nest. Every RPC is scoped
// to the calling user by the auth middleware.
service GoalService {
  // CreateGoal stores a new goal in the incubating state, ranked at the
  // bottom of the caller's incubating group.
  rpc CreateGoal(CreateGoalRequest) returns (CreateGoalResponse);

  // GetGoal returns a single goal by id.
  rpc GetGoal(GetGoalRequest) returns (GetGoalResponse);

  // ListGoals returns all of the caller's goals, ordered by state then
  // position ascending. Clients filter hidden states themselves.
  rpc ListGoals(ListGoalsRequest) returns (ListGoalsResponse);

  // UpdateGoal replaces the editable fields (name, description, due) of an
  // existing goal. Full-replace semantics: omitted optional fields are
  // cleared. State and position are not editable here.
  rpc UpdateGoal(UpdateGoalRequest) returns (UpdateGoalResponse);

  // SetGoalState moves a goal to the given state (any transition allowed)
  // and appends it to the bottom of the destination state group's rank order.
  // Idempotent: setting the current state is a successful no-op that does
  // not change position.
  rpc SetGoalState(SetGoalStateRequest) returns (SetGoalStateResponse);

  // ReorderGoal repositions a goal within its current state group by placing
  // it immediately before or after an anchor goal in the same group. The
  // whole group is renumbered to a contiguous order.
  //
  // Errors:
  //   NotFound        — goal or anchor missing (for this user)
  //   InvalidArgument — anchor is in a different state group, or anchor == goal
  rpc ReorderGoal(ReorderGoalRequest) returns (ReorderGoalResponse);

  // DeleteGoal removes a goal. Associated tasks are NOT deleted or modified;
  // their association is cleared (DB-level ON DELETE SET NULL).
  rpc DeleteGoal(DeleteGoalRequest) returns (DeleteGoalResponse);
}

enum GoalState {
  GOAL_STATE_UNSPECIFIED = 0;
  GOAL_STATE_INCUBATING = 1;
  GOAL_STATE_COMMITTED = 2;
  GOAL_STATE_COMPLETED = 3;
  GOAL_STATE_ARCHIVED = 4;
}

message Goal {
  // Server-assigned, positive, increasing, never reused.
  int64 id = 1;
  // Required, non-empty (trimmed), at most 255 characters.
  string name = 2;
  // Optional longer text; empty string when absent.
  string description = 3;
  // Optional due timestamp (UTC); unset when absent.
  google.protobuf.Timestamp due = 4;
  // Lifecycle state. Never GOAL_STATE_UNSPECIFIED on reads.
  GoalState state = 5;
  // Order within the (user, state) group. Lower sorts first. Read-only:
  // populated on reads, ignored on writes.
  int64 position = 6;
}

message CreateGoalRequest {
  string name = 1;
  string description = 2;
  google.protobuf.Timestamp due = 3;
}
message CreateGoalResponse { Goal goal = 1; }

message GetGoalRequest { int64 id = 1; }
message GetGoalResponse { Goal goal = 1; }

message ListGoalsRequest {}
message ListGoalsResponse { repeated Goal goals = 1; }

message UpdateGoalRequest {
  int64 id = 1;
  // Full-replace: these become the goal's new editable state.
  string name = 2;
  string description = 3;
  google.protobuf.Timestamp due = 4;
}
message UpdateGoalResponse { Goal goal = 1; }

message SetGoalStateRequest {
  int64 id = 1;
  // Must not be GOAL_STATE_UNSPECIFIED.
  GoalState state = 2;
}
message SetGoalStateResponse { Goal goal = 1; }

message ReorderGoalRequest {
  int64 goal_id = 1;
  // Exactly one must be set; must identify a goal in the same state group.
  oneof anchor {
    int64 before_goal_id = 2;
    int64 after_goal_id = 3;
  }
}
message ReorderGoalResponse {
  // The moved goal's state group (including it), in the new order,
  // each with its updated position.
  repeated Goal goals = 1;
}

message DeleteGoalRequest { int64 id = 1; }
message DeleteGoalResponse {}
```

## Additions to `api/proto/task/v1/task.proto`

```proto
message Task {
  // …existing fields 1–10 unchanged…

  // Effective only on the association root: the goal this task's subtree
  // belongs to. Unset when the task is not directly associated with a goal
  // (descendants inherit the nearest ancestor's goal client-side).
  // Read-only: populated on reads, ignored on CreateTask/UpdateTask writes.
  optional int64 goal_id = 11;
}

service TaskService {
  // …existing RPCs unchanged…

  // SetTaskGoal associates a task (and implicitly its whole subtree) with a
  // goal, or clears the association when goal_id is unset.
  //
  // Errors:
  //   NotFound           — task or goal missing (for this user)
  //   FailedPrecondition — an ancestor of the task already has a goal, or
  //                        (when setting) a descendant has its own goal
  rpc SetTaskGoal(SetTaskGoalRequest) returns (SetTaskGoalResponse);
}

message SetTaskGoalRequest {
  int64 task_id = 1;
  // Present: associate with this goal. Absent: clear the association.
  optional int64 goal_id = 2;
}
message SetTaskGoalResponse { Task task = 1; }
```

## Semantics

| Aspect | Behavior |
|---|---|
| Scope | All RPCs operate on the calling user's rows only (auth middleware), like every existing service |
| Create state | Always `incubating`, position = bottom of incubating group |
| State transitions | Any → any; no task side effects; re-rank to bottom of destination group; idempotent no-op for same-state |
| Name validation | Required, trimmed-non-empty, ≤ 255 chars → `InvalidArgument` otherwise (mirrors CreateTask) |
| UpdateGoal | Full-replace for name/description/due; absent due clears it; cannot change state or position |
| Delete | Goal row removed; `tasks.goal_id` cleared by FK action; tasks untouched |
| SetTaskGoal nesting rules | Enforced at write time; see data-model.md validation table. UpdateTask re-parenting that would create nested goal associations is rejected with `FailedPrecondition` |
| ListTasks | Now populates `goal_id` on association-root tasks; otherwise unchanged |
| Registration | `goalv1connect.NewGoalServiceHandler(&handler.Goal{Queries, Pool})` in `cmd/server/main.go`, inside the auth-middleware mux |
| Web/Vite | No web app changes; if the dev proxy ever needs it, `/goal.v1` would join `vite.config.ts` — explicitly out of scope now |
```
