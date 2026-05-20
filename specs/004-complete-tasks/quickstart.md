# Quickstart: Complete Tasks

This walkthrough assumes the backend and CLI are already set up per feature
003 (a provisioned user, an API key exported as `TODO_API_KEY`, and the
backend reachable on `TODO_ADDR` or the default).

## 1. Apply the migration

```bash
cd services/todo
migrate -path db/migrations -database "$TODO_DATABASE_URL" up
```

This applies `000004_complete.up.sql`, adding the nullable `completed_at`
column to `tasks`. No backfill is needed: every existing row is incomplete
by definition (the column defaults to `NULL`).

## 2. Regenerate proto and sqlc bindings

```bash
cd services/todo
buf generate
sqlc generate
```

`buf generate` produces the updated `taskv1.Task` (with `completed_at`) and
the new `taskv1.CompleteTaskRequest` / `taskv1.CompleteTaskResponse` plus the
`CompleteTask` method on the client/server stubs. `sqlc generate` refreshes
`internal/db/models.go` (the `Task` struct gains `CompletedAt
pgtype.Timestamptz`) and `internal/db/task.sql.go` (bindings for
`CompleteTask`, `HasIncompleteDescendants`, `GetParentCompletion`).

## 3. Build the server and the CLI

```bash
go build ./services/todo/cmd/server
go build ./services/todo/cmd/todo
```

Restart the server; the CLI is a single static binary.

## 4. Complete a leaf task

```bash
todo task add "buy groceries"            # → created task 1
todo task complete 1                     # → completed task 1
todo task list                           # → (empty — the only task is done)
todo task list --completed
# [x] 1 buy groceries (completed 2026-05-20T18:42:11Z)
todo task list --all
# [x] 1 buy groceries (completed 2026-05-20T18:42:11Z)
```

## 5. The subtask invariant

```bash
todo task add "release v2"                            # → created task 2
todo task add --parent 2 "write changelog"            # → created task 3
todo task add --parent 2 "tag the commit"             # → created task 4

todo task complete 2
# error: cannot complete task 2: incomplete descendants: [3, 4]
# (exit 1)

todo task complete 3                                  # → completed task 3
todo task complete 2
# error: cannot complete task 2: incomplete descendants: [4]
# (exit 1)

todo task complete 4
todo task complete 2                                  # → completed task 2
```

After the parent is complete, no new subtasks may be added under it:

```bash
todo task add --parent 2 "post the release"
# error: cannot add a subtask under task 2: parent is complete
# (exit 1)
```

## 6. The default-view tree-pruning rule

Set up a small tree with mixed completion:

```bash
todo task add "house chores"                          # → 5
todo task add --parent 5 "vacuum"                     # → 6
todo task add --parent 5 "laundry"                    # → 7
todo task complete 6
```

```bash
todo task list
# [ ] 5 house chores
# ├── [x] 6 vacuum (completed 2026-05-20T18:50:02Z)
# └── [ ] 7 laundry
```

The incomplete parent stays visible even though one child is complete; the
completed child is shown (it is part of an incomplete subtree). Now complete
the laundry:

```bash
todo task complete 7
todo task complete 5
todo task list
# (empty — task 5 and all descendants are complete; the branch is pruned)

todo task list --all
# [x] 5 house chores (completed 2026-05-20T18:55:18Z)
# ├── [x] 6 vacuum (completed 2026-05-20T18:50:02Z)
# └── [x] 7 laundry (completed 2026-05-20T18:54:33Z)
```

## 7. Idempotent re-completion

```bash
todo task complete 1
# task 1 already complete (at 2026-05-20T18:42:11Z)
# (exit 0)
```

The original completion timestamp is preserved. Scripts that complete tasks
repeatedly can do so safely.

## 8. Errors at a glance

| Command | Exit | Stderr (excerpt) |
|---------|------|------------------|
| `todo task complete 999` (no such task) | 1 | `error: task 999 not found` |
| `todo task complete abc` (malformed id) | 1 | `<id> must be an integer, got "abc"` |
| `todo task complete 2` with incomplete descendants | 1 | `cannot complete task 2: incomplete descendants: [...]` |
| `todo task list --completed --all` | 1 | `--completed and --all are mutually exclusive` |
| `todo task add --parent <complete-id> ...` | 1 | `cannot add a subtask under task <id>: parent is complete` |
| `todo task mod --parent <complete-id> <id> <name>` | 1 | `cannot move task under task <id>: parent is complete` |
