# Contract: TaskService (web app subset)

**Source of truth**: `services/todo/proto/task/v1/task.proto` (package `task.v1`). This document records which RPCs the web app consumes and the behavior the client depends on. **No proto changes** are part of this feature; this contract is a usage agreement against the existing service.

**Transport**: ConnectRPC over HTTP from the browser via `@connectrpc/connect-web`. Same origin as the page (Caddy / Vite proxy). The `todo_session` cookie authenticates every call (`credentials: "include"`).

## RPCs used

### ListTasks — load the tree (US2, FR-005)
- Request: `ListTasksRequest{}` (empty).
- Response: `ListTasksResponse{ tasks: Task[] }`, ordered by `id` ascending.
- Client builds the nested tree from `parent_id` (see `data-model.md`).
- Auth: returns `Unauthenticated` if the session is missing/expired → client redirects to `/login?next=…`.

### GetTask — task detail (US2, FR-007)
- Request: `GetTaskRequest{ id }`.
- Response: `GetTaskResponse{ task, completed_pomodoro_count, pomodoros }`. The web app uses only `task`; pomodoro fields are ignored.
- Errors: `NotFound` if the task doesn't exist for this user → "That task seems to have wandered off." (FR: deleted-elsewhere edge case).

### CreateTask — add task or sub-task (US1, FR-008, FR-009)
- Request: `CreateTaskRequest{ name, description, due, parent_id? }`.
  - Top-level task: omit `parent_id`.
  - Sub-task: set `parent_id` to the parent's `id`.
  - `due` omitted in this round (no due-date editing).
- Response: `CreateTaskResponse{ task }`.
- Client MUST block empty/whitespace `name` before calling (FR-010); the server also rejects it.
- On success: invalidate the `ListTasks` query so the tree refreshes.

### UpdateTask — edit (US3, FR-011) — FULL REPLACE
- Request: `UpdateTaskRequest{ id, name, description, due, parent_id? }`.
- **Semantics**: full replace — omitted optional fields are cleared. The client MUST resend the task's current `due` and `parent_id` (loaded via `GetTask`) alongside the edited `name`/`description`. See research R6.
- Response: `UpdateTaskResponse{ task }`.
- On success: invalidate `GetTask(id)` and `ListTasks`.

### CompleteTask — mark complete (US3, FR-012)
- Request: `CompleteTaskRequest{ id }`. Response: `CompleteTaskResponse{ task }`.
- Idempotent (re-completing returns the row unchanged).
- Errors: `FailedPrecondition` when the task has ≥1 incomplete descendant (FR-017) → playful blocking message; do not change UI state.

### UncompleteTask — reopen (US3, FR-012)
- Request: `UncompleteTaskRequest{ id }`. Response: `UncompleteTaskResponse{ task }`.
- Idempotent on already-incomplete tasks.
- Errors: `FailedPrecondition` when the task's parent is complete → playful message advising to reopen the parent first.

## RPCs intentionally NOT used (out of scope this round)

- `DeleteTask` — deletion is out of scope (spec Assumptions).
- `SetEstimate`, `StartPomodoro`, `CancelPomodoro`, `CompletePomodoro`, `GetActivePomodoro` — pomodoro is out of scope.

## Error-to-UX mapping (Principle IV)

| Connect error code | When | User-facing copy (tone-reviewed) |
|--------------------|------|----------------------------------|
| `Unauthenticated` | session missing/expired | redirect to `/login?next=…`; "Your session clocked out. Let's sign back in." |
| `NotFound` | task removed elsewhere | "That task seems to have wandered off." |
| `FailedPrecondition` (complete) | incomplete sub-tasks | "Hold on — finish its sub-tasks first." |
| `FailedPrecondition` (uncomplete) | parent complete | "Reopen its parent first to reopen this one." |
| `InvalidArgument` | empty/oversized name (defense in depth) | "A task needs a name to live by." |
| network / unknown | connectivity drop | "Couldn't reach the server. Want to try again?" + retry (FR-015) |
