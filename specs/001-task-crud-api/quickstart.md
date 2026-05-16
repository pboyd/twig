# Quickstart: Task CRUD API

**Feature**: 001-task-crud-api | **Date**: 2026-05-16

How to build, run, and exercise the `TaskService` once it is implemented. Run all commands from the repository root.

## 1. Generate code from the proto

After `services/todo/proto/task/v1/task.proto` is in place:

```bash
make proto      # runs `buf generate` in services/todo
```

This regenerates `services/todo/gen/task/v1/` (the `task.pb.go` and `taskv1connect/` packages).

Regenerate the database layer after `db/queries/task.sql` and the `000002` migration exist:

```bash
cd services/todo && sqlc generate
```

This refreshes `internal/db/` — `models.go` gains a `Task` struct and `task.sql.go` is added.

## 2. Start the stack

```bash
make dev        # podman-compose up --build: postgres + server
```

The server applies pending migrations on startup (including `000002_tasks`) and listens on `:8080`. To run migrations manually against a running database:

```bash
DATABASE_URL='postgres://todo:todo@localhost:5432/todo?sslmode=disable' make migrate-up
```

## 3. Exercise the RPCs

The service speaks Connect over HTTP/2 (h2c). Each RPC is a POST to `/task.v1.TaskService/<Method>` with a JSON body. Examples use `curl`.

### Create a task

```bash
curl -s http://localhost:8080/task.v1.TaskService/CreateTask \
  -H 'Content-Type: application/json' \
  -d '{"name": "Write the report"}'
# => {"task":{"id":"1","name":"Write the report","description":""}}
```

Create a child task:

```bash
curl -s http://localhost:8080/task.v1.TaskService/CreateTask \
  -H 'Content-Type: application/json' \
  -d '{"name": "Draft section 1", "parentId": "1", "due": "2026-06-01T17:00:00Z"}'
# => {"task":{"id":"2","name":"Draft section 1","due":"2026-06-01T17:00:00Z","parentId":"2"... }}
```

### Get a task

```bash
curl -s http://localhost:8080/task.v1.TaskService/GetTask \
  -H 'Content-Type: application/json' -d '{"id": "1"}'
```

### List tasks

```bash
curl -s http://localhost:8080/task.v1.TaskService/ListTasks \
  -H 'Content-Type: application/json' -d '{}'
# => {"tasks":[{"id":"1",...},{"id":"2",...}]}  (ordered by id ascending)
```

### Update a task (full replace)

```bash
curl -s http://localhost:8080/task.v1.TaskService/UpdateTask \
  -H 'Content-Type: application/json' \
  -d '{"id": "1", "name": "Write the final report"}'
# Omitting description/due/parentId clears them.
```

### Delete a task (cascades to descendants)

```bash
curl -s http://localhost:8080/task.v1.TaskService/DeleteTask \
  -H 'Content-Type: application/json' -d '{"id": "1"}'
# Deletes task 1 and its child task 2.
```

## 4. Expected error behavior

| Situation | Connect code | HTTP status |
|---|---|---|
| Missing / blank name, name > 255 chars | `invalid_argument` | 400 |
| `parentId` references a non-existent task | `invalid_argument` | 400 |
| Update that would create a cycle (task as its own ancestor) | `invalid_argument` | 400 |
| Get / Update / Delete with an unknown `id` | `not_found` | 404 |
| Unexpected database failure | `internal` | 500 |

## 5. Verify acceptance scenarios

Map the spec's user stories to manual checks:

- **US1** — create tasks with/without optional fields; `ListTasks` and `GetTask` return them unchanged.
- **US2** — `UpdateTask` changes name/description/due; omitting an optional field clears it; unknown `id` ⇒ `not_found`.
- **US3** — assign `parentId` to build a multi-level tree; re-parent a task; attempt to set a task's parent to itself or a descendant ⇒ `invalid_argument`; unknown `parentId` ⇒ `invalid_argument`.
- **US4** — delete a leaf task; delete a parent and confirm its whole subtree is gone from `ListTasks`; delete an unknown `id` ⇒ `not_found`.

## 6. Run tests

```bash
cd services/todo && go test ./...
```

Integration tests require a reachable PostgreSQL (use the `postgres` service from `compose.yaml`); set `DATABASE_URL` accordingly.
