# Quickstart: Snooze a Task

End-to-end manual verification once implemented. Assumes the dev stack is up (`make dev`) and a provisioned user with `TWIG_API_KEY` / `TWIG_ADDR` set.

## 0. Regenerate + build after contract/schema edits

```bash
make proto                       # regenerate Go stubs from task.proto
(cd services/twig && sqlc generate)   # regenerate db layer after task.sql edit
(cd services/twig-web && npm run gen) # regenerate web TS from proto
go build -o twig ./cmd/twig
make dev                         # rebuild server; runs migration 000009 on startup
```

## 1. TUI — snooze a task (US1)

```bash
./twig
```

1. Press the add-task key, fill **Name**, and in the new **Snooze** field enter a near-future day, e.g. tomorrow's date `YYYY-MM-DD`. Save.
2. Confirm the task is **not** shown in the default (pending) list.
3. Press `c` → the view becomes **"Show all"**; the snoozed task appears with a 💤 indicator. Press `c` again → back to **"Show only pending"**, task hidden.
4. Edit the snoozed task, clear the Snooze field, save → it returns to the pending list immediately (US1 #4 / FR-011).
5. Edit a task and set Snooze to **today or a past date** → it stays visible (FR-004).
6. Set Snooze on a **parent** task → the whole subtree disappears from pending; reveal with `c` and confirm parent + children show together (FR-012).

## 2. Auto-wake (US1 #2)

- With a task snoozed until tomorrow, re-launch / refresh the list **on or after** that day (or temporarily set the snooze to today) and confirm it reappears in the pending view with no edit.

## 3. CLI honors the snooze (US3)

```bash
./twig task list          # default: snoozed task absent
./twig task list --all    # snoozed task present
```

- The snoozed task is absent from the default `task list` and present under `--all` (FR-009). The CLI offers **no** way to set a snooze (FR-009).

## 4. Web honors the snooze (US3)

```bash
cd services/twig-web && npm run dev   # http://localhost:5173
```

- Log in; the snoozed task is **absent** from the default tree. (Snooze cannot be set from the web.)
- Once its day has passed, it appears as a normal task.

## 5. Empty-state safety (FR-013)

- Snooze/complete every visible task so the pending view is empty; confirm the UI says hidden tasks exist and how to reveal them (it is not a bare "no tasks" dead-end), in the established warm tone (Principle IV).

## Automated tests to add/extend

- **Go handler** (`services/twig/internal/handler/task_test.go`): `dbTaskToProto` maps `snooze_until` (set/unset); `CreateTask`/`UpdateTask` round-trip and clear-on-omit.
- **TUI** (`internal/tui/*_test.go` via `export_test.go`): `buildVisible` with an injected "today" hides future-snoozed nodes and their subtrees in pending view, shows them under "show all", and surfaces the 💤 indicator.
- **CLI** (`internal/cli/*_test.go`): default list prunes future-snoozed; `--all` keeps them; date-boundary cases (today vs tomorrow vs yesterday).
- **Web** (`services/twig-web/src/lib/*.test.ts`): `filterTree` hides future-snoozed (with injected reference date) and keeps subtrees together; `TreeRow` shows 💤.
