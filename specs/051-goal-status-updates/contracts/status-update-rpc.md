# Contract: Goal Status Update RPCs

**Feature**: 051-goal-status-updates | **Date**: 2026-06-12
**Surface**: ConnectRPC, additions to the existing `goal.v1.GoalService`.

This contract is the source of truth for the proto changes and MUST be committed
before implementation (Principle II). After editing the proto, run `make proto`.

## Message additions (`api/proto/goal/v1/goal.proto`)

```proto
message StatusUpdate {
  // Server-assigned, positive, increasing, never reused.
  int64 id = 1;
  // The owning goal.
  int64 goal_id = 2;
  // Required, non-empty (trimmed). Markdown.
  string body = 3;
  // When the update was recorded (UTC). Unchanged by edits.
  google.protobuf.Timestamp created_at = 4;
}

message Goal {
  // ... existing fields id=1 .. position=6 ...

  // The goal's most recent status update; unset when the goal has none.
  // Read-only: populated on reads (ListGoals/GetGoal), ignored on writes.
  StatusUpdate latest_status_update = 7;
}
```

## RPC additions (`service GoalService`)

```proto
// ListGoalStatusUpdates returns a goal's status updates, newest first.
rpc ListGoalStatusUpdates(ListGoalStatusUpdatesRequest)
    returns (ListGoalStatusUpdatesResponse);

// AddGoalStatusUpdate records a new status update on a goal (created_at = now).
rpc AddGoalStatusUpdate(AddGoalStatusUpdateRequest)
    returns (AddGoalStatusUpdateResponse);

// UpdateGoalStatusUpdate replaces the body of an existing update.
// created_at is preserved.
rpc UpdateGoalStatusUpdate(UpdateGoalStatusUpdateRequest)
    returns (UpdateGoalStatusUpdateResponse);

// DeleteGoalStatusUpdate removes a single status update.
rpc DeleteGoalStatusUpdate(DeleteGoalStatusUpdateRequest)
    returns (DeleteGoalStatusUpdateResponse);

message ListGoalStatusUpdatesRequest { int64 goal_id = 1; }
message ListGoalStatusUpdatesResponse { repeated StatusUpdate updates = 1; }

message AddGoalStatusUpdateRequest {
  int64 goal_id = 1;
  string body = 2;       // required, non-empty after trim
}
message AddGoalStatusUpdateResponse { StatusUpdate update = 1; }

message UpdateGoalStatusUpdateRequest {
  int64 id = 1;
  string body = 2;       // required, non-empty after trim
}
message UpdateGoalStatusUpdateResponse { StatusUpdate update = 1; }

message DeleteGoalStatusUpdateRequest { int64 id = 1; }
message DeleteGoalStatusUpdateResponse {}
```

## Behavior & error semantics

All RPCs are scoped to the authenticated caller via the auth middleware and a
join to `goals` on `user_id`.

| RPC | Success | Errors |
|---|---|---|
| `ListGoalStatusUpdates` | updates for the goal, ordered `created_at DESC, id DESC` (empty list if none) | `NotFound` if the goal doesn't exist for this user |
| `AddGoalStatusUpdate` | inserts with `created_at = now()`; returns the new update | `InvalidArgument` if `body` is empty/whitespace; `NotFound` if the goal isn't the caller's |
| `UpdateGoalStatusUpdate` | replaces `body`; `created_at` unchanged; returns the update | `InvalidArgument` if `body` empty/whitespace; `NotFound` if the update isn't on one of the caller's goals |
| `DeleteGoalStatusUpdate` | removes the row | `NotFound` if the update isn't on one of the caller's goals |

- `Goal.latest_status_update` is populated by `ListGoals` and `GetGoal` and left
  unset for goals with no updates. `CreateGoal`/`UpdateGoal` ignore any value a
  client sends in it (consistent with `position`).
- Error messages use playful but actionable copy (Principle IV), e.g. an empty
  body → *"A status update needs a few words — mind jotting something down?"*

## SQL queries (`services/twig/db/queries/goal_status_update.sql`)

Scoping is enforced by the join to `goals` in every statement.

```sql
-- name: ListGoalStatusUpdates :many
SELECT su.* FROM goal_status_updates su
JOIN goals g ON g.id = su.goal_id
WHERE su.goal_id = $1 AND g.user_id = $2
ORDER BY su.created_at DESC, su.id DESC;

-- name: AddGoalStatusUpdate :one
INSERT INTO goal_status_updates (goal_id, body)
SELECT $1, $3 FROM goals WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateGoalStatusUpdate :one
UPDATE goal_status_updates su SET body = $3
FROM goals g
WHERE su.id = $1 AND su.goal_id = g.id AND g.user_id = $2
RETURNING su.*;

-- name: DeleteGoalStatusUpdate :one
DELETE FROM goal_status_updates su
USING goals g
WHERE su.id = $1 AND su.goal_id = g.id AND g.user_id = $2
RETURNING su.id;

-- name: GetLatestGoalStatusUpdate :one
SELECT * FROM goal_status_updates
WHERE goal_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;
```

(`AddGoalStatusUpdate` returns no row when the goal isn't the caller's → handler
maps the empty result to `NotFound`. Body trimming/validation also happens in the
handler before the call so the friendly `InvalidArgument` wins over the DB CHECK.)

For populating `Goal.latest_status_update` on `ListGoals`, the handler may fetch
latest updates for the listed goals (e.g. a batched `... WHERE goal_id = ANY($1)`
picking the top row per goal) to avoid N+1; the exact query is an implementation
detail left to the handler so long as it stays a single round trip.
