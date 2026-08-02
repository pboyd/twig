---
description: "Task list for feature implementation"
---

# Tasks: Goal Handling When Moving Tasks

**Input**: Design documents from `/specs/072-move-task-goal-rules/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/update-task-goal-behavior.md, quickstart.md

**Tests**: Test tasks ARE included. The spec did not request TDD explicitly, but this is a
behavioral bugfix against a handler with an established integration-test suite
(`services/twig/internal/handler/task_test.go`, 30+ tests covering every other `UpdateTask`
behavior). Shipping move/goal semantics without matching coverage would break repo convention, and
the FR-008 rename regression is invisible without a test. Tests are written before implementation
within each phase.

**Organization**: Grouped by user story. All three stories flow through the same rewritten
`UpdateTask` branch, so Phase 2 is unusually load-bearing — it is the shared plumbing, not scaffolding.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1, US2, US3 — maps to user stories in spec.md
- Exact file paths included in every task

## Path Conventions

Multi-module Go monorepo. Paths are repo-root-relative:

- Server module: `services/twig/`
- Shared API module: `api/`
- CLI/TUI module: repo root `internal/`

Per plan.md, all behavior change lands in `services/twig/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the dev stack and capture a green baseline before touching the move path

- [x] T001 Start the dev stack with `make dev` and confirm postgres on `localhost:5432` and the server on `:8080` are healthy
- [x] T002 Export `DATABASE_URL="postgres://twig:twig@localhost:5432/twig?sslmode=disable"` so the handler integration tests in `services/twig/internal/handler/` run instead of skipping (they call `t.Skip` when it is unset — a silent pass is the main hazard in this feature)
- [x] T003 Capture a green baseline: run `go test ./...` at repo root and `cd services/twig && go test ./...`, and record which `TestUpdateTask_*` / `TestReorderTask_*` tests currently pass

**Checkpoint**: Baseline green and integration tests confirmed running, not skipping

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The shared query surface, transaction wrapper, and parent-change comparison that all
three user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete. Every story's behavior
hangs off the parent-change comparison in T009.

- [x] T004 Add the `NearestAncestorGoal` recursive-CTE query to `services/twig/db/queries/task.sql` — walks `parent_id` upward from `$1` scoped to `user_id = $2`, tracks depth, returns the `goal_id` of the closest strict ancestor having one via `ORDER BY depth LIMIT 1`; model it on the existing `AncestorHasGoal` CTE in the same file but return the id rather than `EXISTS`
- [x] T005 Add the `ClearSubtreeGoals` recursive-CTE statement to `services/twig/db/queries/task.sql` — `UPDATE tasks SET goal_id = NULL` over all descendants of `$1` scoped to `user_id = $2`, excluding the row `$1` itself; model it on the existing `DescendantHasGoal` CTE (same file as T004, so run after it)
- [x] T006 Regenerate the query layer: `cd services/twig && sqlc generate`, producing `NearestAncestorGoal` and `ClearSubtreeGoals` in `services/twig/internal/db/task.sql.go` (generated — do not hand-edit)
- [x] T007 In `services/twig/internal/handler/task.go`, add a `t.Pool == nil` guard to `UpdateTask` **after** the `validateName` call and before any DB work, mirroring `ReorderTask` at line ~478 — `TestUpdateTask_NameValidation` constructs `&handler.Task{Queries: nil}` with no `Pool` and must keep returning `InvalidArgument`, not an internal error
- [x] T008 In `services/twig/internal/handler/task.go`, wrap the body of `UpdateTask` after the guard in `tx, err := t.Pool.Begin(ctx)` / `defer tx.Rollback(ctx)` / `q := t.Queries.WithTx(tx)` / `tx.Commit(ctx)`, converting every `t.Queries.X` call in the method to `q.X` (satisfies FR-011 / contract G6)
- [x] T009 In `services/twig/internal/handler/task.go`, load the stored task via `q.GetTask` at the top of the transaction and compute the parent-change classification — `noChange` / `descent` / `promotion` — by comparing stored `ParentID` against `req.Msg.ParentId`, per the table in data-model.md; **do not** branch on `req.Msg.ParentId != nil`, which means "field present" and is always true (research decision 1)
- [x] T010 In `services/twig/internal/handler/task.go`, change the repositioning block (currently guarded by `if req.Msg.ParentId != nil` at ~line 325) to run on any descent **or** promotion, using the T009 classification, so promotions to root stop keeping a stale position (research decision 6, contract G7)
- [x] T011 [P] Add a test helper to `services/twig/internal/handler/task_test.go` that creates a goal and links it to a task via the `Goal` handler from `newGoalTestHandlerWithPool`, plus a `t.Cleanup` that deletes rows from `goals` for the test user — the existing `newTestHandler` cleanup only removes `tasks` and `users`

**Checkpoint**: `UpdateTask` is transactional, classifies parent changes correctly, repositions on
both descent and promotion, and the goal queries exist. No goal-clearing behavior yet.

---

## Phase 3: User Story 1 - Move a goal-linked task under another task (Priority: P1) 🎯 MVP

**Goal**: Moving a goal-linked task under another task succeeds and clears the moved task's own goal
link, instead of failing with `failed_precondition`

**Independent Test**: Link a goal to two separate top-level tasks, move one under the other — the
move succeeds and the moved task no longer holds its own goal link

### Tests for User Story 1 ⚠️

> Write these first and confirm they FAIL against the Phase 2 state

- [x] T012 [P] [US1] Add `TestUpdateTask_MoveGoalLinkedUnderGoalLinked` to `services/twig/internal/handler/task_test.go` — two top-level tasks with different goals, move one under the other, assert no error and moved task's `GoalId` is nil (spec US1 scenario 1)
- [x] T013 [P] [US1] Add `TestUpdateTask_MoveUnderSameGoal` to `services/twig/internal/handler/task_test.go` — both tasks linked to the *same* goal, assert the move succeeds with no error (spec US1 scenario 2; this is the case the old check rejected most confusingly)
- [x] T014 [P] [US1] Add `TestUpdateTask_MoveGoalLinkedUnderGoalFree` to `services/twig/internal/handler/task_test.go` — goal-linked task moved under a goal-free task, assert moved task's `GoalId` is nil afterward (spec US1 scenario 3, FR-002)
- [x] T015 [P] [US1] Add `TestUpdateTask_RenameDoesNotClearGoal` to `services/twig/internal/handler/task_test.go` — update only the `Name` of a goal-linked top-level task while passing its current `parent_id`, assert `GoalId` is unchanged; then repeat for a nested task and assert no goal link in its subtree is touched (FR-008, contract G5 — the regression that full-replace semantics make easy to introduce)

### Implementation for User Story 1

- [x] T016 [US1] In `services/twig/internal/handler/task.go`, delete the nested-goal precondition block at ~lines 276-311 in its entirety, including the `GetTask`/`AncestorHasGoal`/`DescendantHasGoal` calls that feed it and the `"moving this task would nest goal associations — clear the goal link first"` error (FR-001, FR-012, contract G1)
- [x] T017 [US1] In `services/twig/internal/handler/task.go`, on a descent (per T009 classification) set the moved task's `goal_id` to NULL as part of the update, leaving it untouched on `noChange` (FR-002, contract G2)
- [x] T018 [US1] Confirm the retained validations still reject and change nothing — cycle, destination-not-found, and completed-destination paths must return before any goal write, and their errors must be unchanged (FR-009); verify `TestUpdateTask_CompleteParentRejected` in `services/twig/internal/handler/task_test.go` still passes

**Checkpoint**: The reported blocking error is gone. Moves under any destination succeed and leave
no goal link on the moved task. Renames are unaffected.

---

## Phase 4: User Story 2 - Promote a sub-task to the top level (Priority: P1)

**Goal**: A sub-task promoted to the top level keeps the goal it was displaying, instead of arriving
with no goal at all

**Independent Test**: Link a goal to a top-level task, add a sub-task, promote the sub-task to
"no parent" — it arrives at the top level linked to that same goal

### Tests for User Story 2 ⚠️

- [x] T019 [P] [US2] Add `TestUpdateTask_PromoteInheritsGoal` to `services/twig/internal/handler/task_test.go` — sub-task under a goal-linked parent, promoted by omitting `parent_id`, assert its `GoalId` equals the parent's goal (spec US2 scenario 1, FR-004)
- [x] T020 [P] [US2] Add `TestUpdateTask_PromoteFromDepthTwo` to `services/twig/internal/handler/task_test.go` — three-level chain `A(goal) → B → C`, promote `C`, assert it gets `A`'s goal, proving "nearest ancestor" resolves past a goal-free intermediate (spec US2 scenario 2)
- [x] T021 [P] [US2] Add `TestUpdateTask_PromoteWithNoGoalAnywhere` to `services/twig/internal/handler/task_test.go` — sub-task under a goal-free parent, promoted, assert `GoalId` is nil and no error is returned (spec US2 scenario 3, FR-005)
- [x] T022 [P] [US2] Add `TestUpdateTask_PromoteKeepsOwnGoal` to `services/twig/internal/handler/task_test.go` — a top-level goal-linked task "promoted" again (no-op move to root), assert its goal is unchanged (spec US2 scenario 5, FR-006)
- [x] T023 [P] [US2] Add `TestUpdateTask_PromoteCarriesClosedGoal` to `services/twig/internal/handler/task_test.go` — complete or archive the goal via the `Goal` handler, then promote a sub-task under it and assert the closed goal is still linked; also assert `SetTaskGoal` still rejects linking that goal directly with `"cannot link a task to a completed or archived goal"` (FR-007, FR-010, contract G4)

### Implementation for User Story 2

- [x] T024 [US2] In `services/twig/internal/handler/task.go`, on a promotion call `q.NearestAncestorGoal` for the moved task **before** the `UpdateTask` write rewrites `parent_id` — afterwards the task has no ancestors and the query always returns empty (data-model.md ordering note)
- [x] T025 [US2] In `services/twig/internal/handler/task.go`, on a promotion set the moved task's `goal_id` to the T024 result only when the task has no `goal_id` of its own; write it directly through the update rather than routing via `SetTaskGoal`, which would reject closed goals (FR-004, FR-006, FR-007, research decision 5)

**Checkpoint**: Both P1 stories done. Descent clears, promotion preserves. The two originally
reported user-visible failures are fixed.

---

## Phase 5: User Story 3 - The rule holds after every move (Priority: P2)

**Goal**: The invariant "a task with a parent never holds its own goal link" holds after any move,
including for legacy rows written before this change

**Independent Test**: Perform a mixed sequence of moves across depths, then query for rows with both
a non-NULL `parent_id` and a non-NULL `goal_id` — expect zero

### Tests for User Story 3 ⚠️

- [x] T026 [P] [US3] Add `TestUpdateTask_DescentClearsDescendantGoals` to `services/twig/internal/handler/task_test.go` — construct a legacy-shaped violation by writing `goal_id` onto a nested task directly through the pool, move its ancestor under another task, assert the nested task's `goal_id` is NULL (spec US3 scenarios 1-2, FR-003)
- [x] T027 [P] [US3] Add `TestUpdateTask_DescentClearsDeepDescendantGoals` to `services/twig/internal/handler/task_test.go` — same as T026 but with the violating row three levels down, proving the clear recurses rather than touching only direct children
- [x] T028 [P] [US3] Add `TestUpdateTask_InvariantAfterMoveSequence` to `services/twig/internal/handler/task_test.go` — run a mixed sequence of descents and promotions across goal-linked and goal-free tasks, then assert zero rows match `parent_id IS NOT NULL AND goal_id IS NOT NULL` for the test user (spec US3 scenario 3)

### Implementation for User Story 3

- [x] T029 [US3] In `services/twig/internal/handler/task.go`, on a descent call `q.ClearSubtreeGoals` for the moved task inside the same transaction, so every descendant at any depth loses its own goal link alongside the moved task's (FR-003, contract G2)
- [x] T030 [US3] Verify atomicity end to end: confirm every goal read and write in `UpdateTask` uses the transactional `q` rather than `t.Queries`, and that a failure after the parent write rolls back both the move and the goal changes (FR-011, contract G6)

**Checkpoint**: All three stories complete. The invariant holds across move sequences and legacy
rows are repaired opportunistically.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T031 [P] Update the `Task.goal_id` comment in `api/proto/task/v1/task.proto` (~line 135) to note that a move may change it — descent clears, promotion assigns the inherited goal — then run `make proto`; the generated `api/gen/` output should be otherwise unchanged (research decision 7)
- [x] T032 [P] Add a `parent_id` note to `services/twig-web/src/lib/updatePayload.ts` recording that it is full-replace and that omitting `parent_id` promotes the task to the top level, matching the "Full-replace semantics" section of `specs/072-move-task-goal-rules/contracts/update-task-goal-behavior.md`
- [x] T033 Run the full suite: `go test ./...` at repo root and `cd services/twig && go test ./...` with `DATABASE_URL` set, and confirm no previously passing test regressed against the T003 baseline
- [x] T034 Walk `specs/072-move-task-goal-rules/quickstart.md` scenarios 1-12 against a live stack, confirming each "Expect" and each "Before this change" contrast
- [x] T035 Verify via the web UI (`cd services/twig-web && npm run dev`) that renaming a goal-linked task inline does not drop its goal badge — the SPA sends a full-replace payload on every edit, so this is the real-world FR-008 check (quickstart scenario 7)
- [x] T036 [P] Review every user-facing string touched by this feature for Constitution Principle IV tone; the feature only deletes a message, so confirm no dry replacement text was introduced and the retained move errors in `services/twig/internal/handler/task.go` are unchanged
- [x] T037 Confirm no leftover dead code in `services/twig/internal/handler/task.go` — specifically that `AncestorHasGoal` and `DescendantHasGoal` are still referenced by `SetTaskGoal` (~lines 612, 621) and were not removed along with the deleted precondition block in T016

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup — **BLOCKS all user stories**
- **User Stories (Phases 3-5)**: All depend on Phase 2 completion
- **Polish (Phase 6)**: Depends on Phases 3-5

### User Story Dependencies

- **US1 (P1)**: Depends only on Phase 2. Independently testable and shippable — it alone removes the reported error.
- **US2 (P1)**: Depends only on Phase 2. Touches the promotion arm of the same branch; independent of US1's descent arm.
- **US3 (P2)**: Depends on Phase 2. Its implementation (T029) extends the descent arm US1 establishes, so **US3 should follow US1** even though its tests are independent.

### Within Each User Story

- Tests written and failing before implementation
- Queries before handler logic (Phase 2 already sequences this)
- T024 strictly before T025 — the ancestor lookup must precede the parent write

### Critical Sequencing Inside `task.go`

T007 → T008 → T009 → T010, then T016 → T017, then T024 → T025, then T029. All edit the same
`UpdateTask` method and **cannot** be parallelized despite belonging to different stories.

### Parallel Opportunities

- T004 and T005 both edit `task.sql` — sequential, not parallel
- All test-writing tasks within a phase are `[P]`: T012-T015, T019-T023, T026-T028
- T011 is `[P]` against Phase 2's handler work (different file)
- T031, T032, T036 are `[P]` in Polish (proto, web, review — all different files)

---

## Parallel Example: User Story 1

```bash
# Launch the four US1 test-writing tasks together — all append to task_test.go
# as independent test functions, so write them in one pass:
Task: "TestUpdateTask_MoveGoalLinkedUnderGoalLinked in services/twig/internal/handler/task_test.go"
Task: "TestUpdateTask_MoveUnderSameGoal in services/twig/internal/handler/task_test.go"
Task: "TestUpdateTask_MoveGoalLinkedUnderGoalFree in services/twig/internal/handler/task_test.go"
Task: "TestUpdateTask_RenameDoesNotClearGoal in services/twig/internal/handler/task_test.go"

# Then implement sequentially — all three edit the same method:
# T016 (delete precondition) → T017 (clear on descent) → T018 (verify rejections)
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1: Setup — baseline green, `DATABASE_URL` exported
2. Phase 2: Foundational — **critical**, blocks everything
3. Phase 3: User Story 1
4. **STOP and VALIDATE**: quickstart scenarios 1-3 and 7
5. This alone fixes the reported `failed_precondition` and the nested-goal-link bug

### Incremental Delivery

1. Setup + Foundational → transactional move path with correct change detection
2. Add US1 → descent clears → validate → the blocking error is gone
3. Add US2 → promotion preserves → validate → silent goal loss is gone
4. Add US3 → subtree clear → validate → invariant holds, legacy rows repair
5. Polish → proto comment, docs, full quickstart walk

### Parallel Team Strategy

Limited by design — US1, US2, and US3 all edit one method in one file. With two developers:

- Developer A: Phase 2 handler work (T007-T010), then all implementation tasks in sequence
- Developer B: T004-T006 (SQL + sqlc), T011 (test helper), then all test tasks (T012-T015, T019-T023, T026-T028) in parallel with A's implementation

---

## Notes

- `services/twig/internal/db/task.sql.go` is **generated** — produced by T006, never hand-edited
- `api/gen/` is **generated** — produced by `make proto` in T031, never hand-edited
- Handler integration tests `t.Skip` without `DATABASE_URL`; a skipped suite looks identical to a
  passing one in CI output. T002 exists specifically to prevent that false green.
- Out of scope, flagged in plan.md: `moveTaskCmd` at `internal/tui/update.go:539` omits
  `SnoozeUntil`, so moving a task in the TUI un-snoozes it. Same class as commit `1f84006`, which
  fixed the web side only. Not addressed by any task here.
- Commit after each task or logical group; stop at any checkpoint to validate a story independently
