---

description: "Task list for Task CRUD API implementation"
---

# Tasks: Task CRUD API

**Input**: Design documents from `/specs/001-task-crud-api/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/task.proto

**Tests**: Test tasks ARE included. The plan (Technical Context) and research.md R8 commit to a testing approach — table-driven unit tests for handler validation and integration tests against a real PostgreSQL. Within each story, write the test task first and confirm it fails before implementing.

**Organization**: Tasks are grouped by user story. Stories map to spec.md: US1 (P1) capture & review, US2 (P2) update, US3 (P2) hierarchy, US4 (P3) delete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: User story the task belongs to (US1–US4)
- All paths are repo-relative; the service lives under `services/todo/`

## Notes on shared files

Three files are touched by multiple stories. This is expected for a single-service, single-table feature and constrains parallelism:

- `services/todo/internal/handler/task.go` — every story adds or refines RPC methods here.
- `services/todo/db/queries/task.sql` — each story appends its queries here.
- `services/todo/internal/handler/task_test.go` — each story appends its tests here.

Connect-Go generates a `TaskServiceHandler` interface requiring all five RPC methods, so Phase 2 creates the handler with all five methods as `CodeUnimplemented` stubs; each story then replaces the relevant stub(s) with real logic. The server compiles and runs after Phase 2.

---

## Phase 1: Setup

**Purpose**: Confirm tooling before code generation

- [ ] T001 Verify the `buf`, `sqlc`, and `migrate` CLIs are available and confirm no new Go module dependencies are required for `services/todo` — the `google.protobuf.Timestamp` well-known type is provided by the existing `google.golang.org/protobuf` dependency and Buf's bundled WKT imports.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: API contract, schema, and a runnable handler skeleton — all four user stories depend on this phase

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T002 Add the TaskService proto contract at `services/todo/proto/task/v1/task.proto` by copying `specs/001-task-crud-api/contracts/task.proto` (defines all five RPCs and message types).
- [ ] T003 Generate Connect/proto code by running `make proto`; confirm `services/todo/gen/task/v1/` (`task.pb.go` and `taskv1connect/task.connect.go`) is created and committed.
- [ ] T004 [P] Create the schema migration `services/todo/db/migrations/000002_tasks.up.sql` (the `tasks` table, `tasks_parent_id_idx` index) and `services/todo/db/migrations/000002_tasks.down.sql` (`DROP TABLE`), per data-model.md.
- [ ] T005 Create the handler skeleton `services/todo/internal/handler/task.go` — a `Task` struct holding `*db.Queries`, with all five RPC methods (`CreateTask`, `GetTask`, `ListTasks`, `UpdateTask`, `DeleteTask`) implemented as stubs returning `connect.NewError(connect.CodeUnimplemented, ...)`.
- [ ] T006 Register the `TaskService` handler in `services/todo/cmd/server/main.go` using `taskv1connect.NewTaskServiceHandler(&handler.Task{Queries: queries})` and mount it on the existing mux.

**Checkpoint**: `make dev` runs; migration `000002` applies; all five Task RPCs are reachable and return `Unimplemented`.

---

## Phase 3: User Story 1 - Capture and review tasks (Priority: P1) 🎯 MVP

**Goal**: Create tasks (name required; description, due, parent optional), retrieve one by id, and list all tasks ordered by id.

**Independent Test**: Create several tasks (some name-only, some with all fields); confirm each is returned unchanged by `ListTasks` and `GetTask`; confirm a name-less create and an unknown-id get are rejected.

### Tests for User Story 1

- [ ] T007 [P] [US1] Write tests in `services/todo/internal/handler/task_test.go`: unit tests for name validation (empty, whitespace-only, >255 chars) and integration tests for CreateTask / GetTask / ListTasks against a real PostgreSQL (covers spec scenarios US1-1..US1-6). Confirm they fail before implementation.

### Implementation for User Story 1

- [ ] T008 [P] [US1] Add `CreateTask`, `GetTask`, and `ListTasks` queries to `services/todo/db/queries/task.sql` per data-model.md.
- [ ] T009 [US1] Regenerate the database layer: run `sqlc generate` in `services/todo` — produces `services/todo/internal/db/task.sql.go` and the `Task` struct in `models.go` (depends on T004, T008).
- [ ] T010 [US1] Add a `db.Task` → `taskv1.Task` conversion helper in `services/todo/internal/handler/task.go` (maps nullable `due`/`parent_id` to proto presence).
- [ ] T011 [US1] Implement `CreateTask` in `services/todo/internal/handler/task.go` — trim and validate the name (blank → `InvalidArgument`, >255 → `InvalidArgument`), insert via `CreateTask` query, return the created task (depends on T009, T010).
- [ ] T012 [US1] Implement `GetTask` in `services/todo/internal/handler/task.go` — `pgx.ErrNoRows` → `CodeNotFound` (depends on T010).
- [ ] T013 [US1] Implement `ListTasks` in `services/todo/internal/handler/task.go` — return all tasks ordered by id, empty list when none (depends on T010).

**Checkpoint**: User Story 1 is fully functional and independently testable — the MVP.

---

## Phase 4: User Story 2 - Update an existing task (Priority: P2)

**Goal**: Replace the editable fields (name, description, due, parent) of an existing task by id, preserving the identifier; clearing optional fields is supported.

**Independent Test**: Create a task, update each field, confirm the retrieved task reflects the new values with an unchanged id; confirm clearing the due date works; confirm a blank-name update and an unknown-id update are rejected.

### Tests for User Story 2

- [ ] T014 [P] [US2] Write integration tests in `services/todo/internal/handler/task_test.go` for UpdateTask: field updates, clearing optional fields, blank name → `InvalidArgument`, unknown id → `NotFound` (covers spec scenarios US2-1..US2-4). Confirm they fail before implementation.

### Implementation for User Story 2

- [ ] T015 [P] [US2] Add the `UpdateTask` query to `services/todo/db/queries/task.sql` (`UPDATE ... RETURNING *`).
- [ ] T016 [US2] Regenerate the database layer: run `sqlc generate` in `services/todo` (depends on T015).
- [ ] T017 [US2] Implement `UpdateTask` in `services/todo/internal/handler/task.go` — full-replace of name/description/due/parent_id, validate the name, `RETURNING` 0 rows → `CodeNotFound` (depends on T016).

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Organize tasks into a hierarchy (Priority: P2)

**Goal**: Reject unknown parent references with a clean error and prevent cycles, so tasks can be nested to arbitrary depth and re-parented safely.

**Independent Test**: Build a multi-level hierarchy; re-parent a task; confirm setting a task's parent to itself or a descendant is rejected as a cycle; confirm an unknown parent id is rejected.

**⚠️ Dependency**: This story refines `CreateTask` (T011) and `UpdateTask` (T017), so it requires User Story 1 and User Story 2 to be complete.

### Tests for User Story 3

- [ ] T018 [P] [US3] Write integration tests in `services/todo/internal/handler/task_test.go`: build a 3-level hierarchy, re-parent a task, reject self-parent and descendant-parent as cycles, reject unknown parent id (covers spec scenarios US3-1..US3-7). Confirm they fail before implementation.

### Implementation for User Story 3

- [ ] T019 [P] [US3] Add the `TaskExists` and `ParentChainContains` (recursive CTE) queries to `services/todo/db/queries/task.sql` per data-model.md.
- [ ] T020 [US3] Regenerate the database layer: run `sqlc generate` in `services/todo` (depends on T019).
- [ ] T021 [US3] Add a parent-existence pre-check to `CreateTask` in `services/todo/internal/handler/task.go` — when `parent_id` is set, `TaskExists` false → `CodeInvalidArgument` (depends on T020).
- [ ] T022 [US3] Add parent-existence and cycle checks to `UpdateTask` in `services/todo/internal/handler/task.go` — when `parent_id` is set: `TaskExists` false → `InvalidArgument`; `ParentChainContains(new parent, this id)` true → `InvalidArgument` (depends on T020).

**Checkpoint**: Hierarchy is enforced — unknown parents and cycles are rejected on both create and update.

---

## Phase 6: User Story 4 - Delete a task (Priority: P3)

**Goal**: Delete a task by id; deleting a task also deletes its entire subtree (cascade).

**Independent Test**: Delete a leaf task and confirm it is gone; delete a parent and confirm its whole subtree is absent from `ListTasks`; confirm deleting an unknown id returns not-found.

### Tests for User Story 4

- [ ] T023 [P] [US4] Write integration tests in `services/todo/internal/handler/task_test.go` for DeleteTask: delete a leaf, delete a parent and assert the whole subtree is removed (cascade), unknown id → `NotFound` (covers spec scenarios US4-1..US4-4). Confirm they fail before implementation.

### Implementation for User Story 4

- [ ] T024 [P] [US4] Add the `DeleteTask` query (`DELETE ... WHERE id = $1 RETURNING id`) to `services/todo/db/queries/task.sql`.
- [ ] T025 [US4] Regenerate the database layer: run `sqlc generate` in `services/todo` (depends on T024).
- [ ] T026 [US4] Implement `DeleteTask` in `services/todo/internal/handler/task.go` — `RETURNING` 0 rows → `CodeNotFound`; descendants are removed automatically by the `ON DELETE CASCADE` foreign key (depends on T025).

**Checkpoint**: All four user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Verification across all stories

- [ ] T027 Run `go build ./...` and `go vet ./...` in `services/todo`; fix any issues.
- [ ] T028 Run the full test suite `go test ./...` in `services/todo` against a running PostgreSQL (`DATABASE_URL` from `compose.yaml`); all tests green.
- [ ] T029 Execute the `specs/001-task-crud-api/quickstart.md` validation — every RPC plus each error case (`invalid_argument`, `not_found`) against `make dev`.
- [ ] T030 [P] Confirm `gofmt` formatting on `services/todo/internal/handler/task.go` and `task_test.go`; verify `services/todo/gen/task/` and the `000002` migration are committed.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories.
- **User Stories (Phases 3–6)**: All depend on Foundational.
- **Polish (Phase 7)**: Depends on all targeted user stories being complete.

### User Story Dependencies

- **US1 (P1)**: After Foundational. Independent — this is the MVP.
- **US2 (P2)**: After Foundational. Independent of US1 (separate RPC method).
- **US3 (P2)**: After Foundational **and US1 and US2** — it refines the `CreateTask` and `UpdateTask` handlers, so it is not independent of them.
- **US4 (P3)**: After Foundational. Independent of US1/US2/US3.

### Within Each User Story

- Write the test task first; confirm it fails before implementation.
- `task.sql` query change → `sqlc generate` → handler method (queries before code).
- Tasks editing the same file (`task.go`, `task.sql`, `task_test.go`) run sequentially.

### Parallel Opportunities

- Phase 2: T004 (migration) runs in parallel with T002/T003 (proto) — different files.
- Each story's test task (T007, T014, T018, T023) can be written in parallel with that story's query task — different files.
- Stories US1, US2, US4 can be developed by different people after Foundational, but all three edit `task.go`, `task.sql`, and `task_test.go` — coordinate to avoid merge conflicts. US3 must wait for US1 + US2.

## Parallel Example: Phase 2 Foundational

```bash
# After T002 + T003 (proto authored and generated), the migration is independent:
Task: "T004 Create migration 000002_tasks.up.sql / .down.sql"
# can proceed alongside finalizing the generated proto code.
```

## Parallel Example: User Story 1

```bash
# T007 (test file) and T008 (query file) touch different files — start together:
Task: "T007 Write tests in services/todo/internal/handler/task_test.go"
Task: "T008 Add CreateTask/GetTask/ListTasks queries to services/todo/db/queries/task.sql"
```

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1: Setup.
2. Phase 2: Foundational — server runs with stubbed RPCs.
3. Phase 3: User Story 1 — create, get, list.
4. **STOP and VALIDATE**: exercise US1 via quickstart.md.
5. Deploy/demo — this is a usable task tracker.

### Incremental Delivery

1. Setup + Foundational → server runnable.
2. US1 → capture & review → MVP.
3. US2 → update.
4. US3 → hierarchy enforcement (requires US1 + US2).
5. US4 → delete with cascade (can be slotted in any time after Foundational).
6. Phase 7 → verify the whole feature.

## Notes

- [P] = different files, no dependency on an incomplete task.
- [Story] label maps each task to a spec.md user story for traceability.
- `sqlc generate` is idempotent — it regenerates the whole `internal/db/` package from all current queries each time.
- Commit after each task or logical group; `after_plan`/`after_tasks` auto-commit is configurable in `.specify/extensions/git/git-config.yml`.
