# Phase 1 Data Model: Snooze a Task

## Entity delta: `Task`

One new optional attribute is added to the existing `Task` entity. Everything else is unchanged.

| Attribute | Type | Storage | Proto | Meaning |
|---|---|---|---|---|
| `snooze_until` | optional timestamp (pinned to midnight UTC of a calendar day) | `tasks.snooze_until TIMESTAMPTZ` (nullable) | `Task.snooze_until` (`google.protobuf.Timestamp`, field 10) | When set to a future day, the task is hidden from default views until that day. Unset = not snoozed. |

### Database

Migration `000009_task_snooze`:

```sql
-- up
ALTER TABLE tasks ADD COLUMN snooze_until TIMESTAMPTZ;

-- down
ALTER TABLE tasks DROP COLUMN snooze_until;
```

No index needed: filtering is client-side over the already-fetched list; `snooze_until` is never a query predicate.

### Query changes (`db/queries/task.sql`, then `sqlc generate`)

- `CreateTask` — add `snooze_until` to the column list and values (new positional param).
- `UpdateTask` — add `snooze_until = $N` to the `SET` clause (full-replace; NULL clears it).
- `ListTasks`, `GetTask`, etc. use `SELECT *`, so they pick up the column automatically once regenerated.

### Validation rules

- `snooze_until` is optional and free of cross-field constraints. It is **independent of `due`** and of `completed_at` (a task may be snoozed and/or completed).
- A value equal to or before "today" is permitted and inert (treated as not snoozed) — no server-side rejection.
- No special interaction with completion: completing/uncompleting does not touch `snooze_until`.

## Derived rule: client-side visibility predicate

Shared (in spirit) by TUI, CLI, and web. Let `today` = the client's **local** calendar date, and for a task with a set `snooze_until`, let `snoozeDate` = the **UTC** calendar date of `snooze_until`.

```
isSnoozed(task)  := task.snooze_until is set AND snoozeDate > today
isCompleted(task):= task.completed_at is set            (unchanged)

Default / "pending" view  → hide task if isCompleted(task) OR isSnoozed(task)
"Show all" view (TUI)     → show every task
```

### Subtree behavior (FR-012, clarified)

Because each client emits/keeps a node *before* descending into its children and skips the whole node when hidden, a hidden (snoozed or completed) parent hides its **entire subtree** automatically — no descendant is promoted. This already holds for completed tasks; adding the snooze term to the same skip condition extends it to snoozed parents with no extra logic.

- **TUI** (`internal/tui/tree.go` `emitNode`): the early `return` for a hidden node already prevents recursion; add `isSnoozed` to that condition and to the `hasVisibleChildren` check.
- **CLI** (`internal/cli/render.go` `pruneIncomplete`): a node is kept only if it (or a kept descendant) is *actionable*; extend "actionable" from "incomplete" to "incomplete AND not snoozed".
- **Web** (`services/twig-web/src/lib/filterTree.ts` `keep`): extend `isIncomplete` to `isActionable = !completed && !snoozed`.

## State / lifecycle

`snooze_until` has no lifecycle of its own beyond being set, changed, or cleared via the TUI add/edit forms. The task's effective "snoozed?" status is **time-derived** (it flips to false automatically when the local day reaches the snooze day) rather than stored — so no background job or write is needed for a task to "wake up".
