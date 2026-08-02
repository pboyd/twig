# Contract: Web Task Actions — RPC Usage

This feature adds **no** new API surface. This contract fixes how the web SPA uses three existing `task.v1.TaskService` RPCs (authoritative definitions: `api/proto/task/v1/task.proto`; goal rules: `specs/072-move-task-goal-rules/contracts/`). Implementation tasks reference this file.

## 1. Delete a task — `DeleteTask`

Request: `{ id: <task id> }` → Response: `{}` (empty).

- Server cascades to **all descendants** and removes plan entries; the client performs no subtree bookkeeping.
- The UI MUST obtain explicit confirmation first, and the confirmation copy MUST mention subtask cascade when the task has children.
- Error mapping:
  - `NotFound` → "already gone" message; still invalidate `listTasks` and navigate away (the end state the user wanted is reality).
  - other codes → generic connectivity message; stay on the page.
- Post-success: invalidate `listTasks`, navigate to `/tasks`, success toast.

## 2. Edit due date / snooze date — `UpdateTask`

`UpdateTask` is **full-replace on every editable field**. The payload MUST be built by `buildUpdatePayload` from a **freshly refetched** task (existing stale-parent guard in `TaskDetailPage.handleSaveEdit` stays), carrying:

| Field | Source |
|---|---|
| `id` | task |
| `name`, `description` | form (existing) |
| `due` | form — `Timestamp` at midnight UTC of the chosen day; **omitted** when cleared/empty |
| `snoozeUntil` | form — `Timestamp` at midnight UTC of the chosen day; **omitted** when cleared/empty |
| `parentId` | fresh task, unmodified — omitting it is interpreted as promotion-to-root and rewrites the goal link (specs/072) |

- `goal_id` is ignored on writes and MUST NOT be sent.
- Date semantics: `<input type="date">` value `YYYY-MM-DD` ⇄ `Timestamp{seconds: UTC midnight}` via `src/lib/dateFields.ts`; conversion is lossless both ways. Past dates are legal.
- Error mapping: `NotFound` → task-gone message + list refresh; others → generic message, form stays populated.
- Post-success: invalidate `getTask(id)` + `listTasks`.

## 3. Link / unlink goal — `SetTaskGoal`

Request: `{ taskId, goalId? }` — present links (moves the association to that goal), absent clears it. Response: `{ task }`.

- Applies to the task's whole subtree implicitly; client sends nothing about descendants.
- UI preconditions (mirror server rules, do not replace them):
  - Task with an **inherited** goal (an ancestor has `goalId`): render read-only inherited display, no link/unlink controls.
  - Picker offers only goals in selectable states (In Progress / Incubating / Hold) from `ListGoals`.
- Error mapping (`FailedPrecondition` carries both server causes; the client cannot distinguish them from the code alone, so choose message by context):
  - attempting to **set** → message explaining a parent's or subtask's existing goal is in the way (covers both causes in plain language).
  - `NotFound` → task-or-goal-gone message + refresh.
  - other codes → generic connectivity message.
- Post-success: invalidate `getTask(id)` + `listTasks` (goal objects themselves are unchanged; `listGoals` not invalidated).

## Cross-cutting

- All user-facing copy for these flows lives in `src/theme/messages.ts` (Principle IV).
- Every mutation reflects success or a human-readable error, and affected views update via query invalidation without a manual reload (FR-010).
