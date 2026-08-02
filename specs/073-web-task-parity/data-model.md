# Data Model: Task Parity for the Web App

No new entities, fields, or storage. The feature exposes existing server-side fields and relationships in the web UI. Authoritative definitions live in `api/proto/task/v1/task.proto` and `api/proto/goal/v1/goal.proto`.

## Task (existing — fields this feature touches)

| Field | Type | Notes for this feature |
|---|---|---|
| `id` | int64 | Route param on `/tasks/:id`; delete/update/goal-link target. |
| `due` | Timestamp (optional) | Newly editable from the web. Set/cleared via `UpdateTask` (full-replace). Rendered date-only via `formatDueDate`. Stored as midnight UTC of the chosen day (client convention, see research R3). |
| `snooze_until` | Timestamp (optional) | Newly editable from the web. Pinned to midnight UTC of the chosen calendar day (proto contract). A task is "snoozed" while this day is strictly after the client's local current day. |
| `parent_id` | int64 (optional) | **Not editable here** (out of scope), but must be carried through every `UpdateTask` payload — omitting it reads as promotion-to-root and rewrites the goal link (specs/072). |
| `goal_id` | int64 (optional) | Read-only on `UpdateTask` writes. Set only on the association root; descendants inherit client-side by nearest-ancestor walk. Written only via `SetTaskGoal`. |
| `name`, `description` | string | Already editable; must continue to round-trip unchanged when only dates are edited (FR-009). |

### Derived (client-side only)

- **Effective goal** (`effectiveGoal.ts`): for task T, walk T's ancestor chain in the `ListTasks` result; the nearest task (including T) with `goalId` set determines the goal, and whether it is *direct* (T itself) or *inherited* (an ancestor). Not persisted.
- **Is snoozed** (existing `TreeRow` logic): `snooze_until` UTC calendar day strictly after local today.

### State transitions

- Delete: Task (and entire subtree, server cascade) → gone. Also removed from any plan by the server; no client bookkeeping.
- Goal link: none ↔ goal A ↔ goal B, always via `SetTaskGoal`; blocked when an ancestor has a goal or (when setting) a descendant has its own goal (`FailedPrecondition`).
- Due / snooze: unset ↔ set(any calendar day, past allowed).

## Goal (existing — read-only in this feature)

| Field | Type | Notes |
|---|---|---|
| `id` | int64 | `SetTaskGoalRequest.goal_id`; matching tasks filter on `GoalDetailPage`. |
| `name` | string | Shown in the task's goal section and picker. |
| `state` | GoalState | Picker lists non-hidden groups (In Progress, Incubating, Hold) in `goalGroups` order; Completed/Archived are excluded — `SetTaskGoal` would refuse them. |

Deleting a task never modifies the goal itself; only the association disappears (edge case in spec).

## Validation rules (client)

- Task name: unchanged existing rule (non-empty trimmed) — still the only client-side validation on the edit form.
- Due / snooze date inputs: any valid calendar date or empty (empty = clear). No range validation.
- Goal picker: only offers goals returned by `ListGoals` in selectable states; empty list renders the friendly empty message, not a control.
