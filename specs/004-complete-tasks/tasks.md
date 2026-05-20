---

description: "Task list for feature 004-complete-tasks"
---

# Tasks: Complete Tasks

**Input**: Design documents in `/specs/004-complete-tasks/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/task-completion.md](./contracts/task-completion.md)

**Tests**: Included. The spec's acceptance scenarios and the plan's Testing section establish that this feature ships with handler and CLI tests. Test tasks appear in each user-story phase before the implementation tasks they cover.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no incomplete-task dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- File paths are exact; the layout is documented in [plan.md](./plan.md#project-structure)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Land the schema delta, the API contract delta, and the codegen, so every subsequent task can compile.

- [ ] T001 Create migration `services/todo/db/migrations/000004_complete.up.sql` adding `completed_at TIMESTAMPTZ` to `tasks`, per [data-model.md](./data-model.md#schema-delta).
- [ ] T002 Create migration `services/todo/db/migrations/000004_complete.down.sql` dropping `completed_at` from `tasks`.
- [ ] T003 Edit `services/todo/proto/task/v1/task.proto` to add `Task.completed_at` (field number 6) and the `CompleteTask` RPC with its `CompleteTaskRequest` / `CompleteTaskResponse` messages, matching [contracts/task-completion.md](./contracts/task-completion.md#proto-delta) verbatim.
- [ ] T004 Add the three new query definitions (`CompleteTask`, `HasIncompleteDescendants`, `GetParentCompletion`) to `services/todo/db/queries/task.sql`, copied verbatim from [data-model.md](./data-model.md#query-additions).
- [ ] T005 Run `buf generate` in `services/todo/` and commit the regenerated files under `services/todo/gen/task/v1/`.
- [ ] T006 Run `sqlc generate` in `services/todo/` and commit the regenerated `services/todo/internal/db/models.go` (Task gains `CompletedAt pgtype.Timestamptz`) and `services/todo/internal/db/task.sql.go` (bindings for the three new queries).
- [ ] T007 Apply migration `000004_complete` to the local development database with `migrate -path db/migrations -database "$TODO_DATABASE_URL" up` from `services/todo/`.

**Checkpoint**: Schema, proto, and generated code are all in lockstep with this feature's contract; downstream tasks compile.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: One shared edit every user story depends on: surfacing `completed_at` on the wire.

**⚠️ CRITICAL**: No user-story work can begin until this phase is complete.

- [ ] T008 Extend `dbTaskToProto` in `services/todo/internal/handler/task.go` so a non-zero `CompletedAt` on the DB row populates `taskv1.Task.completed_at` via `timestamppb.New(...)`. Leave the field unset when `CompletedAt.Valid` is false.
- [ ] T009 [P] Update the existing `TestDbTaskToProto`-style test in `services/todo/internal/handler/task_test.go` (or add a focused subtest if none exists) covering both branches of T008: a row with `CompletedAt.Valid = true` round-trips a non-nil `Task.completed_at`; a row with `CompletedAt.Valid = false` produces a nil `Task.completed_at`.

**Checkpoint**: Every task returned by every existing RPC now carries its completion state to the client; user-story phases can proceed in parallel.

---

## Phase 3: User Story 1 — Mark a task complete (Priority: P1) 🎯 MVP

**Goal**: A user can run `todo task complete <id>` against a leaf task and the task records the completion moment. Re-running the command on an already-complete task is a no-op that preserves the original timestamp. A non-existent id is rejected with exit `1`.

**Independent Test**: With the feature 003 user/API key set up, run `todo task add "thing"`, then `todo task complete <id>`, then `todo task list --all` (which exists as part of US3 — for US1-only verification, query the DB or use the existing `GetTask` to confirm `completed_at` is set). Re-run `todo task complete <id>` and confirm exit code `0` with no change in `completed_at`. Run against an unknown id and confirm exit code `1`.

### Tests for User Story 1

- [ ] T010 [P] [US1] In `services/todo/internal/handler/task_test.go`, add table-driven tests for `CompleteTask`: (a) marks an incomplete leaf task complete and returns a `Task` with `completed_at` set near `time.Now()`; (b) called twice in a row, the second call returns the same `completed_at` as the first; (c) returns `connect.CodeNotFound` for an id that does not belong to the calling user (cross-user isolation); (d) returns `connect.CodeNotFound` for an id that does not exist at all.
- [ ] T011 [P] [US1] In `services/todo/internal/cli/task_test.go`, add tests for `runComplete`: (a) calls `CompleteTask` with the parsed id and prints `completed task <id>` on success returning exit `0`; (b) on the idempotent re-complete success path prints `task <id> already complete (at <timestamp>)` and returns exit `0` — detected by the returned task's `completed_at` predating the call; (c) returns exit `1` and prints a `<id> must be an integer` usage error for `todo task complete abc`; (d) returns exit `1` and prints `task <id> not found` when the server returns `CodeNotFound`.

### Implementation for User Story 1

- [ ] T012 [US1] Implement `(t *Task) CompleteTask(ctx, req)` in `services/todo/internal/handler/task.go`: read `userID := auth.UserID(ctx)`, call `t.Queries.CompleteTask(ctx, db.CompleteTaskParams{ID: req.Msg.Id, UserID: userID})`, map `pgx.ErrNoRows` to `connect.CodeNotFound` with message `"task not found"`, map other errors to `connect.CodeInternal`, and on success return `connect.NewResponse(&taskv1.CompleteTaskResponse{Task: dbTaskToProto(row)})`. (The descendant check is added in US4 — for US1 alone, completion always succeeds on found rows.)
- [ ] T013 [US1] Wire the new RPC into the server in `services/todo/cmd/server/main.go` — the existing `taskv1connect.NewTaskServiceHandler(...)` will pick up the new method automatically once the generated code from T005 is present; verify by building `go build ./services/todo/cmd/server`.
- [ ] T014 [US1] Add `runComplete(client, args)` in `services/todo/internal/cli/task.go`: parse a single integer positional argument (mirroring `runRm`'s error message style), call `client.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: id}))`, map errors via `mapError`, distinguish the "freshly completed" vs "already complete" output based on whether the returned task's `completed_at` predates the request (compare against `time.Now()` minus a small slack or, simpler, compare the returned timestamp against the moment captured just before the call).
- [ ] T015 [US1] Register `"complete"` in the subcommand switch in `runTask` inside `services/todo/internal/cli/cli.go` and extend `printTaskUsage` to mention the new subcommand.

**Checkpoint**: A leaf task can be completed end-to-end via the CLI; idempotent re-complete and not-found behaviors match the spec.

---

## Phase 4: User Story 2 — Default to showing only incomplete tasks (Priority: P1)

**Goal**: `todo task list` (no flags) returns only incomplete tasks, with the tree-pruning rule from spec Q1: completed leaves are hidden, completed parents stay visible only when they have any incomplete descendant, fully-complete subtrees disappear entirely.

**Independent Test**: With several tasks created and a mix marked complete (using US1's CLI subcommand), run `todo task list`; confirm completed rows are absent, incomplete parents with completed children still appear (with the completed children pruned), and a subtree where every task is complete is gone.

### Tests for User Story 2

- [ ] T016 [P] [US2] In `services/todo/internal/cli/render_test.go`, add tests for a new `pruneIncomplete(roots []*treeNode) []*treeNode` helper: (a) a complete leaf is dropped; (b) an incomplete leaf is kept; (c) an incomplete parent with a mix of complete and incomplete children keeps itself and the incomplete children only; (d) a parent with all-complete descendants disappears entirely; (e) the original id-ordering of kept siblings is preserved.

### Implementation for User Story 2

- [ ] T017 [US2] In `services/todo/internal/cli/render.go`, add `pruneIncomplete(roots []*treeNode) []*treeNode` that returns a new slice containing only nodes for which the node itself is incomplete OR any descendant (recursively) is incomplete; recurse into kept nodes' children. A task is incomplete when `node.task.GetCompletedAt() == nil`.
- [ ] T018 [US2] In `services/todo/internal/cli/task.go`, modify `runList` to call `pruneIncomplete` between `buildTree` and `renderRoots` when the default filter applies (until US3 lands, that is always). Adjust the "no tasks" message so it triggers when the pruned list is empty, not just when the server returned zero tasks.

**Checkpoint**: Default `todo task list` reflects the spec's "what's left to do" view.

---

## Phase 5: User Story 3 — Filter the task list by completion status (Priority: P2)

**Goal**: `todo task list` accepts `--completed` and `--all` (mutually exclusive). Every list mode prefixes lines with `[ ]` / `[x]` and shows a completion timestamp for completed rows.

**Independent Test**: With a mix of complete and incomplete tasks, run each of `todo task list`, `todo task list --completed`, `todo task list --all` and confirm: (a) the prefix appears on every row; (b) completed rows show ` (completed <RFC3339-UTC>)`; (c) `--completed` shows only complete tasks; (d) `--all` shows everything; (e) `todo task list --completed --all` exits `1` with the conflict message.

### Tests for User Story 3

- [ ] T019 [P] [US3] In `services/todo/internal/cli/render_test.go`, extend `renderRoots` / `renderTree` tests to assert the `[ ]` / `[x]` prefix on every line and the trailing ` (completed YYYY-MM-DDTHH:MM:SSZ)` suffix for completed tasks. Cover both root nodes and nested children.
- [ ] T020 [P] [US3] In `services/todo/internal/cli/render_test.go`, add tests for a new `filterCompleted(roots []*treeNode) []*treeNode` helper: keeps any node whose task is complete, preserving tree position; incomplete intermediate parents that have no kept descendants are dropped, but if an incomplete parent has at least one completed descendant, the completed descendant is promoted to the nearest kept ancestor's level (or to a root when no kept ancestor exists). Id ordering preserved among siblings.
- [ ] T021 [P] [US3] In `services/todo/internal/cli/task_test.go`, add tests for `runList`'s flag parsing: (a) `--completed` invokes the completed-only render path; (b) `--all` invokes the all-tasks render path with no pruning; (c) both flags together → stderr message `--completed and --all are mutually exclusive`, exit `1`, no RPC call made (assert via a fake client that records calls).

### Implementation for User Story 3

- [ ] T022 [US3] In `services/todo/internal/cli/render.go`, modify `renderTree` and `renderRoots` to always emit a `[ ]` or `[x]` prefix before the id and, when `completed_at` is set, append ` (completed <RFC3339-UTC>)` after the existing `(due ...)` clause. Add a small helper `formatCompletedAt(ts *timestamppb.Timestamp) string` paralleling `formatDue`.
- [ ] T023 [US3] In `services/todo/internal/cli/render.go`, add `filterCompleted(roots []*treeNode) []*treeNode` implementing the promotion semantics described in T020.
- [ ] T024 [US3] In `services/todo/internal/cli/task.go`, rewrite `runList`'s body to use `flag.NewFlagSet` with `--completed` and `--all` boolean flags (mirroring `runAdd`'s pattern), validate they are not both set, call `ListTasks`, then dispatch: no flag → `pruneIncomplete` then render; `--completed` → `filterCompleted` then render; `--all` → render directly. The "no tasks" message must trigger whenever the rendered set is empty regardless of mode.

**Checkpoint**: All three list modes work; the checkbox marker and completion timestamp appear consistently; conflicting flags are rejected at the CLI layer.

---

## Phase 6: User Story 4 — Block completion when subtasks are incomplete (Priority: P2)

**Goal**: The completion invariant from [data-model.md](./data-model.md#invariants) is enforced server-side: `CompleteTask` is blocked when any descendant is incomplete (naming at least one blocking descendant); `CreateTask` and `UpdateTask` reject new or re-parented tasks under a complete parent.

**Independent Test**: Create a parent task with one incomplete child; attempt to complete the parent → exit `1` with a message naming the child. Complete the child, then the parent → success. Attempt `todo task add --parent <complete-id> ...` → exit `1`. Attempt `todo task mod --parent <complete-id> ...` → exit `1`.

### Tests for User Story 4

- [ ] T025 [P] [US4] In `services/todo/internal/handler/task_test.go`, extend the `CompleteTask` tests with: (a) a parent with one incomplete direct child → `connect.CodeFailedPrecondition`, error message mentions the blocking child's id, parent's `completed_at` unchanged in the DB; (b) a parent with one already-complete child → completion succeeds; (c) a three-level chain where the deepest grandchild is incomplete → completing the top-level parent fails with the grandchild's id surfaced; (d) once all descendants are complete, the top-level parent completes successfully.
- [ ] T026 [P] [US4] In `services/todo/internal/handler/task_test.go`, add tests for `CreateTask`'s new precondition: (a) creating with `parent_id` pointing at a complete task → `connect.CodeFailedPrecondition` with a "parent is complete" message; no row is inserted (assert via a follow-up `ListTasks`). (b) creating with an incomplete parent — existing behavior — still succeeds.
- [ ] T027 [P] [US4] In `services/todo/internal/handler/task_test.go`, add tests for `UpdateTask`'s new precondition: (a) update setting `parent_id` to a complete task → `connect.CodeFailedPrecondition` with the same message; no DB change. (b) update not changing `parent_id`, against a task that itself has a complete ancestor — does not regress (existing path stays valid).
- [ ] T028 [P] [US4] In `services/todo/internal/cli/task_test.go`, add tests asserting the user-visible error messages: `runComplete` surfaces the server's "incomplete descendants: [...]" message to stderr with exit `1`; `runAdd` and `runMod` surface "parent is complete" with exit `1`.

### Implementation for User Story 4

- [ ] T029 [US4] In `services/todo/internal/handler/task.go`'s `CompleteTask` (added in T012), before calling `t.Queries.CompleteTask`, first read the task itself with `GetTask` to detect not-found (so the not-found code path remains correct); then call `t.Queries.HasIncompleteDescendants(ctx, db.HasIncompleteDescendantsParams{ParentID: req.Msg.Id, UserID: userID})`. If true, look up at least one blocking descendant id (a small follow-up query — add `ListIncompleteDescendantIds` to `task.sql` if needed, or reuse the recursive CTE with a different projection) and return `connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("incomplete descendants: %v", ids))`. Skip the mutation when blocked. Already-complete tasks still pass through (idempotent re-complete remains a no-op).
- [ ] T030 [US4] In `services/todo/internal/handler/task.go`'s `CreateTask`, after the existing parent-existence check (which currently calls `TaskExists`), additionally call `t.Queries.GetParentCompletion(ctx, db.GetParentCompletionParams{ID: *req.Msg.ParentId, UserID: userID})` (or fold the check into a single fetch); if the returned `completed_at` is non-null, return `connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot add a subtask under task %d: parent is complete", *req.Msg.ParentId))` and do not insert.
- [ ] T031 [US4] In `services/todo/internal/handler/task.go`'s `UpdateTask`, mirror T030 inside the `if req.Msg.ParentId != nil` branch: after the existence + cycle checks, reject when the proposed new parent's `completed_at` is non-null with `connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot move task under task %d: parent is complete", newParentID))`.
- [ ] T032 [US4] If T029 needed a helper query, add `ListIncompleteDescendantIds` to `services/todo/db/queries/task.sql` (same recursive CTE as `HasIncompleteDescendants` but `SELECT id FROM descendants WHERE completed_at IS NULL ORDER BY id`), re-run `sqlc generate`, and update T029's implementation to use it. If T029 is implemented with a single combined query (returning either an empty list or the blocking ids), this task collapses into T029 and can be skipped.

**Checkpoint**: The invariant holds — no path through the API can produce a complete task with an incomplete descendant.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T033 [P] Walk the [quickstart.md](./quickstart.md) end-to-end against a freshly-migrated database and confirm every printed line matches the implemented output (CLI confirmation strings, exit codes, tree shapes). Adjust the quickstart text if any wording diverged from the implementation; the contract document is the source of truth and should not be changed.
- [ ] T034 [P] Run `go vet ./...` and `go test ./...` from `services/todo/` and fix any issues introduced by the new code.
- [ ] T035 Mark the spec checklist `specs/004-complete-tasks/checklists/requirements.md` as still passing after implementation (it should — no spec changes are expected during implementation).

---

## Dependencies

```text
Phase 1 (Setup, T001–T007) ──► Phase 2 (Foundational, T008–T009) ──► Phase 3 (US1) ──► Phase 4 (US2)
                                                                                  └──► Phase 5 (US3)
                                                                                  └──► Phase 6 (US4)
                                                                                            │
                                                                                            ▼
                                                                                       Phase 7 (Polish)
```

- **US1** is the only story with a code dependency on Phase 2's `dbTaskToProto` change (it returns a `Task` from `CompleteTask`).
- **US2, US3, US4** each depend on **US1** only because their tests need a way to mark a task complete from the CLI; they do not share files with each other and can be implemented in parallel after US1.
- **US3** and **US2** both touch `services/todo/internal/cli/render.go` and `services/todo/internal/cli/task.go`. They can run in parallel by separate developers but require a small merge resolution on `runList`'s body — recommended to land US2 first, then US3 extends it.
- **US4** is server-side only and shares no files with US2 or US3; it is fully parallel.

## Parallel-Execution Examples

After Phase 2 completes, three parallel work streams open:

- Stream A (CLI rendering): T016 → T017 → T018 → T019 → T020 → T022 → T023 → T024
- Stream B (Server invariants): T025 → T026 → T027 → T029 → T030 → T031 → T032
- Stream C (CLI error surfacing): T028 (depends on T029–T031 having landed for end-to-end runs, but its assertions can be drafted in parallel)

Inside Phase 3, T010 and T011 can be authored in parallel (separate files); T012–T015 must follow in the listed order because T013 and T014 import the generated method created by T012's signature, and T015 references `runComplete` introduced in T014.

## Implementation Strategy

- **MVP scope**: Phase 1 + Phase 2 + Phase 3 (User Story 1). This alone delivers the spec's primary requirement — "users need to be able to mark a task complete" — with idempotent semantics and per-user isolation. Listing behavior is unchanged at MVP, which is acceptable because every existing task is incomplete on the first migration.
- **Incremental delivery**:
  1. MVP — completion works; default listing unchanged (acceptable transitional state).
  2. + US2 — default listing now hides completed work; the feature is usable end-to-end for a single-user workflow.
  3. + US3 — completed and all-tasks views become available; visual indicators land.
  4. + US4 — the invariant is enforced; up to this point, FR-003 / FR-013 / FR-014 are guarded only by user discipline.
  5. + Polish — quickstart and lint pass.
- **Risk surface**: the recursive CTE (`HasIncompleteDescendants`) is the only piece of non-trivial SQL. Validate it with the multi-level test case in T025(c) before relying on it in production.

## Total task count

- Phase 1 (Setup): 7
- Phase 2 (Foundational): 2
- Phase 3 (US1): 6 (2 test, 4 impl)
- Phase 4 (US2): 3 (1 test, 2 impl)
- Phase 5 (US3): 6 (3 test, 3 impl)
- Phase 6 (US4): 8 (4 test, 4 impl)
- Phase 7 (Polish): 3
- **Total: 35 tasks**
