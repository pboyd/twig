# Phase 1 Data Model: Goals in the Web App

No new persistent entities — the web app consumes existing server types from `api/proto/goal/v1/goal.proto` (generated into `src/gen/goal/v1/goal_pb.ts`) and `task/v1/task.proto`. This document records the shapes as the frontend sees them and the client-side view models / validation derived from the spec.

## Server types consumed (read-only definitions)

### Goal (from `goal_pb.ts`)

| Field | Type | Notes for web |
|-------|------|---------------|
| `id` | `bigint` | Stable id; used in route `/goals/:id` (string ↔ `BigInt`). |
| `name` | `string` | Required, ≤255 chars. Rendered inline markdown. |
| `description` | `string` | May be empty. Rendered block markdown when present. |
| `due` | `Timestamp \| undefined` | Optional. Displayed date-only when set. |
| `state` | `GoalState` enum | Drives grouping. Never `UNSPECIFIED` on reads. |
| `position` | `bigint` | Order within `(user, state)` group; ascending. Read-only. |
| `latestStatusUpdate` | `StatusUpdate \| undefined` | Present on reads. **Not used for rendering** (see research Decision 4) — kept available but the detail page drives status off `ListGoalStatusUpdates`. |

`GoalState` values: `GOAL_STATE_INCUBATING`, `GOAL_STATE_COMMITTED`, `GOAL_STATE_COMPLETED`, `GOAL_STATE_ARCHIVED` (and `UNSPECIFIED=0`, never seen on reads).

### StatusUpdate (from `goal_pb.ts`)

| Field | Type | Notes for web |
|-------|------|---------------|
| `id` | `bigint` | Identifies an update for edit/delete. |
| `goalId` | `bigint` | Owning goal. |
| `body` | `string` | Required, non-empty (trimmed), markdown. Rendered block markdown. |
| `createdAt` | `Timestamp` | Recorded time (UTC). Unchanged by edits. Shown human-readable. |

### Task (relevant subset, from `task_pb.ts`)

| Field | Type | Notes for web |
|-------|------|---------------|
| `id` | `bigint` | Link target `/tasks/:id`. |
| `name` | `string` | Inline markdown. |
| `goalId` | `bigint \| undefined` | Set on the association root only. Filter `goalId === goal.id` to list a goal's associated tasks. |
| `completedAt` | `Timestamp \| undefined` | Used for the existing checkmark styling if shown. |

## Client view models (derived, not persisted)

### GoalGroup (`lib/goalGroups.ts`)

```text
GoalGroup = { state: GoalState; label: string; goals: Goal[] }
```

- Input: `Goal[]` from `ListGoals` + `showHidden: boolean`.
- Output: ordered `GoalGroup[]`:
  1. `COMMITTED` (label "Committed")
  2. `INCUBATING` (label "Incubating")
  3. `COMPLETED` (label "Completed") — only when `showHidden`
  4. `ARCHIVED` (label "Archived") — only when `showHidden`
- Within each group, goals preserve server order (ascending `position`).
- Empty groups are omitted from the output.

### Show-hidden preference (`lib/showHiddenGoalsPref.ts`)

- `readShowHiddenGoals(): boolean` / `writeShowHiddenGoals(v: boolean): void`
- Backed by `localStorage["twig-show-hidden-goals"]`; defaults to `false`; guarded against `localStorage` exceptions (mirrors `showCompletedPref.ts`).

### Status update editor state (component-local, `StatusUpdateForm`)

```text
{ body: string; mode: "add" | "edit"; targetId?: bigint }
```

## Validation rules (client-side, from spec)

| Rule | Source | Behavior |
|------|--------|----------|
| Status body required (non-empty after trim) | FR-014 | Block submit on add **and** edit; show playful validation message; no RPC sent. Server also enforces (defense in depth). |
| Goals are read-only on web | FR-019 | No create/edit/delete/state/reorder UI is rendered or wired. |
| Tasks list is read-only on goal detail | FR-008 | Render links only; no link/unlink controls. |
| Full text readable | FR-012 | Status bodies wrap and the history scrolls; no truncation/line-clamp on the rendered body. |

## State & ordering semantics (display only)

- Goal lifecycle states are displayed, never mutated from the web (transitions remain in TUI/CLI).
- Status updates are presented newest-first exactly as returned by `ListGoalStatusUpdates`; "latest status" = first element.
- Editing a status update preserves its `createdAt` (server guarantees); the UI shows no separate "edited at".

## Freshness / invalidation map

| Mutation | Invalidate query keys |
|----------|-----------------------|
| `AddGoalStatusUpdate` | `listGoalStatusUpdates({ goalId })`, `getGoal({ id: goalId })`, `listGoals({})` |
| `UpdateGoalStatusUpdate` | `listGoalStatusUpdates({ goalId })`, `getGoal({ id: goalId })`, `listGoals({})` |
| `DeleteGoalStatusUpdate` | `listGoalStatusUpdates({ goalId })`, `getGoal({ id: goalId })`, `listGoals({})` |

(`listGoals` invalidation matters only if the list view ever surfaces a latest-status preview; included for correctness and cheap at this scale.)
