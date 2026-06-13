# Data Model: Goal Status Updates

**Feature**: 051-goal-status-updates | **Date**: 2026-06-12

## Entities

### Status Update (new)

| Field | Type | Constraints |
|---|---|---|
| `id` | BIGINT identity PK | server-assigned, positive, never reused |
| `goal_id` | BIGINT NOT NULL, FK → `goals(id)` `ON DELETE CASCADE` | the owning goal; deleting the goal deletes its updates |
| `body` | TEXT NOT NULL | required, non-blank after trim (CHECK `btrim(body) <> ''`) |
| `created_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | recorded moment; unchanged by edits |

Index: `goal_status_updates_goal_created_idx ON goal_status_updates (goal_id, created_at DESC, id DESC)`
— supports newest-first listing per goal and selecting the latest.

**Ownership / scoping**: no `user_id` column. Every query joins to `goals` and
filters on `goals.user_id`, so a user can only see or modify updates on their own
goals (FR-011). Referencing another user's goal/update yields no row → `NotFound`.

**Latest update (derived)**: the single row with the greatest
`(created_at, id)` for a goal. Populated into `Goal.latest_status_update` on
`ListGoals`/`GetGoal`; absent when the goal has no updates.

### Goal (existing, extended)

| New field | Type | Constraints |
|---|---|---|
| `latest_status_update` | proto `StatusUpdate` (read-only) | populated on reads; ignored on writes (like `position`) |

No column change to the `goals` table — the latest is computed at read time.

## Validation rules

- `body` MUST be non-empty after trimming whitespace, enforced by:
  - DB `CHECK (btrim(body) <> '')`, and
  - handler validation returning `InvalidArgument` (playful message) before the
    DB call, and
  - the TUI discarding an empty `$EDITOR` result with a notice (no RPC sent).
- `created_at` is server-assigned on insert and never modified by
  `UpdateGoalStatusUpdate`.

## Lifecycle

- **Create** → row inserted with `now()`; becomes the new latest.
- **Edit** → `body` replaced; `created_at` and ordering unchanged.
- **Delete** → row removed; if it was the only update, the goal returns to the
  "no status yet" state and `latest_status_update` becomes absent.
- **Goal deleted** → all its updates removed by DB cascade.

## Proto mapping

`goal.v1.StatusUpdate`:
- `int64 id = 1`
- `int64 goal_id = 2`
- `string body = 3`
- `google.protobuf.Timestamp created_at = 4`

`goal.v1.Goal` gains `StatusUpdate latest_status_update = 7` (read-only:
populated on reads, ignored on `CreateGoal`/`UpdateGoal`). Field number 7 is the
next free tag after `position = 6`.
