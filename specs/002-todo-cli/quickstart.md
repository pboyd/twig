# Quickstart: Todo CLI Tool

**Feature**: 002-todo-cli | **Date**: 2026-05-18

How to build, configure, and exercise the `todo` CLI once it is implemented. Run
commands from the repository root unless noted.

## 1. Build the binary

The CLI is the `cmd/todo` binary in the `services/todo` module:

```bash
cd services/todo && go build -o todo ./cmd/todo
```

This produces a `todo` executable. (A `make cli` target may be added to wrap
this.)

## 2. Start a backend

The CLI needs a running `TaskService` (feature 001):

```bash
make dev        # podman-compose up --build: postgres + server on :8080
```

## 3. Point the CLI at the backend

The backend address comes from `TODO_ADDR`, defaulting to
`http://localhost:8080`:

```bash
export TODO_ADDR=http://localhost:8080   # optional; this is the default
```

## 4. Exercise the commands

### Add tasks

```bash
./todo task add "Write the report"
# => created task 1

./todo task add --parent 1 --due 2026-06-01T17:00:00Z "Draft section 1"
# => created task 2

./todo task add --parent 2 --due 2026-05-25 "Gather sources"
# => created task 3   (bare date stored as 2026-05-25T00:00:00Z)
```

### List the tree

```bash
./todo task list
# [1] Write the report
# └── [2] Draft section 1 (due 2026-06-01T17:00:00Z)
#     └── [3] Gather sources (due 2026-05-25T00:00:00Z)
```

With no tasks recorded:

```bash
./todo task list
# no tasks
```

### Modify a task

```bash
./todo task mod 3 "Gather primary sources"
# => updated task 3

./todo task mod --due 2026-05-30T09:00:00Z 3 "Gather primary sources"
# => updated task 3   (name must be re-supplied; due changes, parent preserved)
```

### Remove a task

```bash
./todo task rm 1
# => deleted task 1   (cascades: tasks 2 and 3 are removed too)
```

## 5. Expected error behavior

| Situation | Message destination | Exit code |
|---|---|---|
| Missing name / bad subcommand / no args | stderr (usage help) | 1 |
| Non-integer `<id>` or `--parent` | stderr | 1 |
| Unparseable `--due` | stderr (names accepted formats) | 1 |
| Unknown task id (`rm`, `mod`) | stderr (backend not-found) | 1 |
| Unknown `--parent` id | stderr (backend error) | 1 |
| `--parent` would create a cycle (`mod`) | stderr (backend cycle error) | 1 |
| Backend unreachable | stderr (connection error) | 1 |
| Any success | stdout | 0 |

Verify the single-code contract:

```bash
./todo task rm 999999 ; echo "exit=$?"
# => exit=1
```

## 6. Verify acceptance scenarios

Map the spec's user stories to manual checks:

- **US1 (add)** — add with name only, with `--due`, with `--parent`; each prints
  a new id and the task then appears in `list`. Adding without a name, or with
  an unknown `--parent`, fails with a clear message and exit 1.
- **US2 (list)** — with tasks nested several levels deep, `list` shows every
  task once, connector-drawn under its parent; siblings in id order; a task with
  no due date still lists; empty store prints `no tasks`.
- **US3 (mod)** — rename, change `--due`, and re-parent in separate invocations;
  the id is unchanged and unflagged fields stay put; unknown id fails not-found;
  a self-ancestor `--parent` fails with the cycle error.
- **US4 (rm)** — remove a leaf; remove a parent and confirm its subtree is gone
  from `list`; remove an unknown id and get a not-found error.

## 7. Run tests

```bash
cd services/todo && go test ./internal/cli/...
```

The CLI tests use an in-memory fake `TaskService` over `httptest` — no
PostgreSQL and no running server are required.
