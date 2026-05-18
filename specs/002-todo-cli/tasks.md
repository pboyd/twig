---
description: "Task list for Todo CLI Tool implementation"
---

# Tasks: Todo CLI Tool

**Input**: Design documents from `/specs/002-todo-cli/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

**Tests**: Test tasks ARE included — research.md Decision 8 and plan.md mandate a
two-layer `go test` strategy (unit tests for pure helpers, command tests against
an in-memory fake `TaskService`). Test tasks precede their implementation tasks.

**Organization**: Tasks are grouped by user story (priority order P1 → P4) so
each story is an independently testable increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story the task belongs to (US1–US4)
- File paths are repository-root-relative; the Go module is
  `github.com/pboyd/todo/services/todo`

## Path Conventions

CLI is added to the existing `services/todo` Go module as a second binary:

- Entrypoint: `services/todo/cmd/todo/main.go`
- Command logic: `services/todo/internal/cli/{cli,task,render}.go`
- Tests: `services/todo/internal/cli/{render_test,task_test}.go`
- Reused generated client: `services/todo/gen/task/v1` (`taskv1`) and
  `services/todo/gen/task/v1/taskv1connect`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project structure for the new binary and package

- [ ] T001 Create the directory structure `services/todo/cmd/todo/` and `services/todo/internal/cli/` per plan.md
- [ ] T002 [P] Add the built `todo` binary to `services/todo/.gitignore`
- [ ] T003 Create the thin CLI entrypoint in `services/todo/cmd/todo/main.go` (`os.Exit(cli.Run(os.Args[1:]))`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Dispatch, transport, error mapping, and the test harness that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Implement root dispatch, `TODO_ADDR` resolution (default `http://localhost:8080`), Connect client construction over a stock `http.Client`, the `task`-group router with an unknown-subcommand→usage default, top-level/group usage help, and exit-code translation in `services/todo/internal/cli/cli.go` — exposes `Run([]string) int`
- [ ] T005 [P] Implement RPC and transport error mapping (`connect.CodeOf`: NotFound, InvalidArgument verbatim, dial/transport failures → "cannot reach backend") in `services/todo/internal/cli/render.go`
- [ ] T006 [P] Create the in-memory fake `TaskService` (implements `taskv1connect.TaskServiceHandler`, map-backed) and the `httptest`-based command-test harness in `services/todo/internal/cli/task_test.go`

**Checkpoint**: `Run` compiles, prints usage and exits `1` for any unknown subcommand; user story implementation can now begin

---

## Phase 3: User Story 1 - Add a task from the command line (Priority: P1) 🎯 MVP

**Goal**: `todo task add [--parent <id>] [--due <timestamp>] <name>` creates a task and reports its new id

**Independent Test**: Run `add` with a name only, with `--due`, and with `--parent`; each reports success and a new id, and the task is then visible via the backend. Missing name or malformed flag fails with a usage error and exit `1`.

### Tests for User Story 1

- [ ] T007 [P] [US1] Table-driven unit tests for `--due` parsing (RFC 3339, bare `YYYY-MM-DD`→`00:00:00Z`, rejection of malformed input) in `services/todo/internal/cli/render_test.go`
- [ ] T008 [P] [US1] Command tests for `add` (name only, with `--due`, with `--parent`, missing name, unknown parent, malformed `--due`/`--parent`) in `services/todo/internal/cli/task_test.go`

### Implementation for User Story 1

- [ ] T009 [US1] Implement the `parseDue` helper (try `time.RFC3339`, then `2006-01-02`; convert to `*timestamppb.Timestamp`; usage error naming both forms on failure) in `services/todo/internal/cli/render.go`
- [ ] T010 [US1] Implement the `add` subcommand (`flag.FlagSet` for `--parent`/`--due`, required positional `<name>`, integer validation, `CreateTask` RPC, success line naming the new id) and wire its route into the `task`-group dispatch in `services/todo/internal/cli/task.go` and `services/todo/internal/cli/cli.go` (depends on T009)

**Checkpoint**: `add` is fully functional and independently testable — MVP

---

## Phase 4: User Story 2 - View all tasks as a tree (Priority: P2)

**Goal**: `todo task list` prints every task as a `tree`-style hierarchy showing id, name, and due date

**Independent Test**: With tasks nested several levels deep, `list` shows every task once, connector-drawn under its parent, siblings in id order; a task with no due date still lists; an empty store prints `no tasks` and exits `0`.

### Tests for User Story 2

- [ ] T011 [P] [US2] Table-driven unit tests for render-tree construction and `tree`-style rendering (root ordering, nesting depth, `├──`/`└──`/`│` glyphs, blank due field) in `services/todo/internal/cli/render_test.go`
- [ ] T012 [P] [US2] Command tests for `list` (multi-level tree, id-ascending siblings, task with no due date, empty `no tasks` result) in `services/todo/internal/cli/task_test.go`

### Implementation for User Story 2

- [ ] T013 [US2] Implement render-tree construction from the flat `ListTasks` response (index by id, attach to parents, id-ascending order), `tree`-style depth-first rendering, and due-date display formatting (`UTC().Format(time.RFC3339)`) in `services/todo/internal/cli/render.go`
- [ ] T014 [US2] Implement the `list` subcommand (`ListTasks` RPC, `no tasks` message on empty, print the rendered tree) and wire its dispatch route in `services/todo/internal/cli/task.go` and `services/todo/internal/cli/cli.go` (depends on T013)

**Checkpoint**: `add` and `list` both work independently

---

## Phase 5: User Story 3 - Modify an existing task (Priority: P3)

**Goal**: `todo task mod [--parent <id>] [--due <timestamp>] <id> <name>` updates a task, preserving unflagged fields

**Independent Test**: Record a task, then `mod` to change name, `--due`, and `--parent` in separate invocations; the id is unchanged and unflagged fields (including description) stay put; an unknown id fails not-found with no `UpdateTask`; a self-ancestor `--parent` surfaces the backend cycle error.

### Tests for User Story 3

- [ ] T015 [US3] Command tests for `mod` (rename, change `--due`, re-parent, unflagged fields preserved, unknown id not-found, self-ancestor cycle error) in `services/todo/internal/cli/task_test.go`

### Implementation for User Story 3

- [ ] T016 [US3] Implement the `mod` subcommand using fetch-then-update — required positional `<id>`+`<name>`, `--parent`/`--due` flags, `GetTask` then `UpdateTask` carrying full state (description from fetch, `due`/`parent_id` from flag-or-fetch), success line naming the id — and wire its dispatch route in `services/todo/internal/cli/task.go` and `services/todo/internal/cli/cli.go` (reuses `parseDue` from T009)

**Checkpoint**: `add`, `list`, and `mod` all work independently

---

## Phase 6: User Story 4 - Remove a task (Priority: P4)

**Goal**: `todo task rm <id>` deletes a task (the backend cascades to descendants)

**Independent Test**: Record a task, `rm` its id, confirm it is gone from `list`; remove a parent and confirm its subtree is gone; remove an unknown id and get a not-found error with exit `1`.

### Tests for User Story 4

- [ ] T017 [US4] Command tests for `rm` (delete a leaf, delete a parent with descendants, unknown id not-found, malformed id usage error) in `services/todo/internal/cli/task_test.go`

### Implementation for User Story 4

- [ ] T018 [US4] Implement the `rm` subcommand (required positional integer `<id>`, integer validation, `DeleteTask` RPC, success line naming the deleted id) and wire its dispatch route in `services/todo/internal/cli/task.go` and `services/todo/internal/cli/cli.go`

**Checkpoint**: All four subcommands are independently functional

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Build ergonomics and end-to-end validation

- [ ] T019 [P] Add a `make cli` target that builds `cmd/todo` (per quickstart.md §1) to the repository `Makefile`
- [ ] T020 Run `gofmt -l` and `go vet ./...` in `services/todo` and fix any findings
- [ ] T021 Run `go test ./internal/cli/...` from `services/todo` and confirm all unit and command tests pass
- [ ] T022 Execute the quickstart.md manual validation (build the binary, start a backend, exercise each command and the error cases)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **User Stories (Phase 3–6)**: All depend on Foundational completion
  - Implemented in priority order P1 → P2 → P3 → P4
- **Polish (Phase 7)**: Depends on all targeted user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends only on Foundational — the MVP
- **US2 (P2)**: Depends only on Foundational — independently testable
- **US3 (P3)**: Depends on Foundational; reuses `parseDue` (T009, US1)
- **US4 (P4)**: Depends only on Foundational — independently testable

Stories are independently testable, but several share files: `task.go`,
`task_test.go`, `render.go`, and `render_test.go` are each touched by more than
one phase. Tasks on a shared file MUST run sequentially — they are already
ordered by phase/priority, so following the numeric order satisfies this.

### Within Each User Story

- Test tasks come before their implementation tasks
- Pure helpers (`render.go`) before the subcommand that uses them
- Story complete before moving to the next priority

### Parallel Opportunities

- T002 runs parallel to T001/T003 in Setup
- T004, T005, T006 (Foundational) are different files — all [P]
- T007 + T008 (US1 tests) are different files — [P]
- T011 + T012 (US2 tests) are different files — [P]
- T019 (Polish) is parallel to T020–T022 prep

---

## Parallel Example: Foundational Phase

```bash
# T004, T005, T006 touch different files and can run together:
Task: "Implement root dispatch and client in services/todo/internal/cli/cli.go"
Task: "Implement error mapping in services/todo/internal/cli/render.go"
Task: "Create fake TaskService test harness in services/todo/internal/cli/task_test.go"
```

## Parallel Example: User Story 1 Tests

```bash
# T007 and T008 touch different files and can run together:
Task: "--due parsing unit tests in services/todo/internal/cli/render_test.go"
Task: "add command tests in services/todo/internal/cli/task_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1 (`add`)
4. **STOP and VALIDATE**: test `add` independently against the fake backend
5. This is a usable MVP — tasks can be captured from the command line

### Incremental Delivery

1. Setup + Foundational → foundation ready
2. Add US1 (`add`) → test → MVP
3. Add US2 (`list`) → test → capture-and-review tool
4. Add US3 (`mod`) → test → editing supported
5. Add US4 (`rm`) → test → full CRUD CLI
6. Polish → `make cli` target, fmt/vet clean, all tests green, quickstart verified

---

## Notes

- [P] tasks = different files, no dependency on an incomplete task
- The CLI adds no new Go module and no new third-party dependency (Principle I)
- `mod` is two RPCs (`GetTask` + `UpdateTask`); fetch-then-update preserves the
  description this tool never exposes (FR-012)
- All failures exit `1`; messages go to stderr, normal output to stdout
- Commit after each task or logical group
