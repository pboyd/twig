# Contract: GoalService (Goal State Edits)

The **only** contract change in this feature is one new value on the `GoalState` enum. No RPC signatures, request messages, or response messages change. This document also records the existing RPCs the TUI relies on, so the client behavior is pinned.

Source of truth: `api/proto/goal/v1/goal.proto` (regenerate stubs with `make proto`).

## Change: `GoalState` enum — add `GOAL_STATE_HOLD`

```proto
enum GoalState {
  GOAL_STATE_UNSPECIFIED = 0;
  GOAL_STATE_INCUBATING  = 1;
  GOAL_STATE_COMMITTED   = 2;
  GOAL_STATE_COMPLETED   = 3;
  GOAL_STATE_ARCHIVED    = 4;
  GOAL_STATE_HOLD        = 5;  // NEW: still-pursued, not currently worked on
}
```

- Value `5` is **appended** — existing numbers are untouched, so the change is wire-compatible with existing stored/serialized data.
- `GOAL_STATE_UNSPECIFIED` remains invalid on any write.
- Server mapping: `hold` ⇆ `GOAL_STATE_HOLD` added to `goalStateToString` / `goalStateFromString` in `services/twig/internal/handler/goal.go`.
- DB backing: `goals.state` `CHECK` constraint widened to include `'hold'` (migration `000012`).

## Unchanged RPCs used by this feature

### `SetGoalState(SetGoalStateRequest) → SetGoalStateResponse`

Used by **both** the edit-form State selector and the Space complete/uncomplete toggle.

```proto
message SetGoalStateRequest {
  int64 id = 1;
  GoalState state = 2;  // must not be GOAL_STATE_UNSPECIFIED; may now be GOAL_STATE_HOLD
}
message SetGoalStateResponse { Goal goal = 1; }
```

Behavior (existing, relied upon):
- Any transition allowed, including to/from `HOLD`.
- Setting the goal's current state is an idempotent no-op that does **not** change position.
- On a real change, the goal is appended to the bottom of the destination state group's rank order.
- Errors: `InvalidArgument` if `state == UNSPECIFIED`; `NotFound` if the goal isn't the caller's.

### `UpdateGoal(UpdateGoalRequest) → UpdateGoalResponse` — unchanged

```proto
message UpdateGoalRequest {
  int64 id = 1;
  string name = 2;
  string description = 3;
  google.protobuf.Timestamp due = 4;
  // NOTE: no state field — full-replace of editable text fields only.
}
```

The edit form calls `UpdateGoal` for name/description/due and, separately, `SetGoalState` when the State selector changed. `UpdateGoal` does **not** gain a `state` field.

### `ListGoals(ListGoalsRequest) → ListGoalsResponse` — behavior note

Signature unchanged. The server-side `ORDER BY` is extended so `hold` sorts between `incubating` and `completed`. Clients still filter hidden states themselves (the TUI hides `hold`, `completed`, and `archived` unless "show all" is on).

## Client (TUI) contract expectations

- `setGoalStateCmd` may now be invoked with `GOAL_STATE_HOLD` (from the edit form) — no code path assumes a fixed set of four states.
- The complete toggle sends `GOAL_STATE_COMPLETED` for any non-completed goal and `GOAL_STATE_COMMITTED` for a completed goal.
- Goal grouping/visibility treats `HOLD` like `COMPLETED`/`ARCHIVED` for default hiding, but orders it ahead of them.
