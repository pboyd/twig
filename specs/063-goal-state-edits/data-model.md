# Phase 1 Data Model: Goal State Edits

This feature adds one value to an existing enum. No new entities, tables, or relationships.

## Entity: Goal (modified)

The `Goal` entity is unchanged except for its `state` value set.

| Field | Type | Notes |
|---|---|---|
| id | int64 | unchanged |
| name | string | unchanged |
| description | string | unchanged |
| due | timestamp? | unchanged |
| **state** | enum | **gains a 5th value: `Hold`** |
| position | int64 | order within `(user, state)` group; re-assigned on state change |
| latest_status_update | StatusUpdate? | unchanged |

### State value set (was 4, now 5)

| Proto enum | DB token | Default-visible in TUI? | Group order |
|---|---|---|---|
| `GOAL_STATE_INCUBATING` | `incubating` | yes | 2 |
| `GOAL_STATE_COMMITTED` | `committed` | yes | 1 |
| **`GOAL_STATE_HOLD` = 5** | **`hold`** | **no (shown only with "show all")** | **3** |
| `GOAL_STATE_COMPLETED` | `completed` | no (shown only with "show all") | 4 |
| `GOAL_STATE_ARCHIVED` | `archived` | no (shown only with "show all") | 5 |

- `GOAL_STATE_UNSPECIFIED = 0` remains invalid on writes.
- The enum value `5` is appended (does not reorder existing values), preserving wire compatibility.
- Display label for `GOAL_STATE_HOLD` is **"Hold"**.

## Persistence changes

**Migration `000012_goal_hold_state`**:
- **up**: widen the constraint to `CHECK (state IN ('incubating','committed','completed','archived','hold'))`.
- **down**: restore the original 4-value constraint. Down is only safe when no rows use `'hold'`; the down migration should be written to fail loudly (or is documented as unsafe) if any `hold` rows exist, rather than silently corrupting data.

**Query `ListGoals` (`goal.sql`)**: extend the `ORDER BY CASE state` mapping so `'hold'` sorts as `3`, shifting `completed`→`4`, `archived`→`5`. Regenerate `internal/db/goal.sql.go` with `sqlc generate`.

No change to `SetGoalState`, `ListGoalStateGroup`, `ReorderGoal`, `CreateGoal`, `UpdateGoal`, or the `goals_user_state_position_idx` index.

## State transitions

State is set explicitly by the user; **all transitions are allowed** (the `SetGoalState` RPC imposes no restrictions). Two entry points:

1. **Edit-form State selector** — may set any of the five states.
2. **Space key (complete toggle)** — a constrained subset:

| Current state | Space → new state |
|---|---|
| Incubating | Completed |
| Committed | Completed |
| Hold | Completed |
| Completed | **Committed** (un-complete; per spec Clarification) |
| Archived | Completed |

Every state change re-ranks the goal to the bottom of the destination group's position order (existing `SetGoalState` behavior). Setting the current state again is an idempotent no-op that does not change position.

## Validation rules

- `state` must never be `GOAL_STATE_UNSPECIFIED` on a write (`goalStateToString` returns `InvalidArgument`).
- DB `CHECK` constraint is the backstop: only the five known tokens may be stored.
- No migration of existing rows — every current goal keeps its stored state; nothing is auto-moved to `hold` (spec FR-011).

## Ownership & scope

Unchanged: goals (and therefore their state) are scoped to the owning user by the auth middleware. Hold introduces no new sharing or visibility rules beyond the default-hidden behavior described above.
