# Phase 1 Data Model: Task Web App

The web app introduces **no new persisted entities**. It consumes the existing `task.v1` proto types. This document maps the spec's entities to the existing proto and records the client-side view models the SPA builds in memory.

## Source of truth

All entities are defined in `services/todo/proto/task/v1/task.proto`. Generated TypeScript lands in `services/todo-web/src/gen/`. The web app uses a subset of the proto fields.

## Entity: Task (existing proto `task.v1.Task`)

| Proto field | Type | Web app usage |
|-------------|------|---------------|
| `id` | int64 | Identity; route param `/tasks/:id`; tree key. |
| `name` | string | **Displayed as "title."** Required, trimmed, ≤255 chars. The spec's "title" = proto `name`. |
| `description` | string | Optional longer text; empty string when absent. Editable. |
| `due` | Timestamp | **Not edited in this round.** Read and **re-sent unchanged** on `UpdateTask` (full-replace — see below). May be shown read-only. |
| `parent_id` | optional int64 | Defines the tree. Set when creating a sub-task. **Re-sent unchanged** on `UpdateTask`. |
| `completed_at` | Timestamp | Unset = incomplete, set = complete. Drives the completion indicator in tree + detail. Toggled via Complete/Uncomplete RPCs, never written directly. |
| `estimate` | int32 | Out of scope (pomodoro). Ignored by the web app. |

### Spec-to-proto terminology map

| Spec term | Proto term |
|-----------|-----------|
| title | `name` |
| sub-task | a `Task` whose `parent_id` = the parent's `id` |
| completion status / "complete" | `completed_at` set/unset |
| mark complete / reopen | `CompleteTask` / `UncompleteTask` |

### Validation rules (enforced by backend; mirrored in UI for fast feedback)

- `name` MUST be non-empty after trimming and ≤255 chars (FR-010). The form blocks submit and shows a playful message before calling the API.
- A parent task cannot be completed while any descendant is incomplete (FR-017) — enforced server-side, surfaced as a friendly error on `FailedPrecondition`.
- A task cannot be reopened while its parent is complete — server-side, surfaced likewise.

### State transitions (completion)

```
incomplete --CompleteTask (all descendants complete)--> complete
complete   --UncompleteTask (parent not complete)-----> incomplete
```

- `CompleteTask` blocked by `FailedPrecondition` if any descendant incomplete.
- `UncompleteTask` blocked by `FailedPrecondition` if parent complete.
- `CompleteTask` is idempotent (safe re-tap).

## Client-side view model: TaskNode (in-memory only)

`ListTasks` returns a flat `Task[]` ordered by `id` ascending. `lib/tree.ts` transforms it into a nested structure for rendering. Not persisted.

```ts
interface TaskNode {
  task: Task;            // the proto message
  children: TaskNode[];  // ordered by id ascending (server order preserved)
  depth: number;         // 0 for top-level; used for mobile indentation
}
```

Build rules:
- Roots = tasks with no `parent_id`.
- A child attaches under the node whose `id === child.parent_id`.
- Sibling order = input order (id ascending).
- Orphans (parent missing from the user's set — shouldn't happen, but defensive) are treated as roots.

## Client-side transient UI state (not persisted, not server state)

- **Expanded/collapsed** branches: `Set<bigint>` of expanded task ids, local `useState` (FR-006).
- **Form inputs**: controlled fields in add/edit forms.
- **`next` redirect target**: query param on `/login` after a `401`.

## Edit payload rule (critical — full-replace)

`UpdateTask` replaces all editable fields. To edit only name/description without data loss, the client builds the request from the **currently loaded task**:

```ts
// from the task loaded via GetTask
UpdateTaskRequest {
  id,                       // target
  name: editedName,         // user-edited
  description: editedDesc,  // user-edited
  due: currentTask.due,         // preserved unchanged
  parentId: currentTask.parentId // preserved unchanged
}
```

Omitting `due`/`parentId` would clear the due date / re-parent the task to top level. See `contracts/task-service.md` and research R6.
