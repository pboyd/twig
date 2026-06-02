# Tasks: CLI Task Uncomplete

**Input**: Design documents from `specs/032-cli-task-uncomplete/`

**Prerequisites**: plan.md ✅, spec.md ✅, contracts/cli-uncomplete.md ✅

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to

---

## Phase 1: Foundational (Blocking Prerequisites)

**Purpose**: Extend the `fakeTaskService` test double so it satisfies the `TaskServiceClient` interface once `runUncomplete` is added. This must land before any test can compile.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T001 Add `UncompleteTask` stub to `fakeTaskService` in `services/twig/internal/cli/task_test.go` — clears `CompletedAt` on the matching task or returns `CodeNotFound`

**Checkpoint**: `go test ./internal/cli/...` compiles (tests may fail; that is expected)

---

## Phase 2: User Story 1 — Reopen a completed task from the command line (Priority: P1) 🎯 MVP

**Goal**: A user can run `twig task uncomplete <id>` to mark a completed task incomplete again.

**Independent Test**: Run `twig task complete 1` then `twig task uncomplete 1`; confirm task reappears in the default listing and output reads `task 1 is back on your list`.

### Implementation for User Story 1

- [x] T002 [US1] Add `runUncomplete` function to `services/twig/internal/cli/task.go` — mirrors `runComplete`; calls `client.UncompleteTask`; prints `"task %d is back on your list\n"` on success, `"task %d was already incomplete\n"` when `CompletedAt` is already nil
- [x] T003 [P] [US1] Add `case "uncomplete": return runUncomplete(client, addr, args[1:])` to the task dispatch switch in `services/twig/internal/cli/cli.go`
- [x] T004 [P] [US1] Add `uncomplete <id>` usage line to `printTaskUsage` in `services/twig/internal/cli/cli.go` — place it immediately after the `complete <id>` line
- [x] T005 [US1] Add four tests for `runUncomplete` in `services/twig/internal/cli/task_test.go` (depends on T001, T002):
  - `TestUncompleteLeaf` — complete a task, then uncomplete it; expect exit 0 and "back on your list" in stdout
  - `TestUncompleteIdempotent` — uncomplete a task that is already incomplete; expect exit 0 and "already incomplete" in stdout
  - `TestUncompleteMalformedID` — pass `"abc"`; expect exit 1 and "integer" in stderr
  - `TestUncompleteNotFound` — pass `999`; expect exit 1 and non-empty stderr

**Checkpoint**: `go test ./internal/cli/...` passes; all four new tests green

---

## Phase 3: Polish & Cross-Cutting Concerns

- [x] T006 Run `cd services/twig && go test ./...` and confirm no regressions across all packages
- [x] T007 [P] Run `cd services/twig && go build -o /dev/null ./cmd/twig` to confirm the binary compiles cleanly

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies — start immediately
- **User Story (Phase 2)**: Depends on T001 completion — `fakeTaskService` must compile with `UncompleteTask` before tests can be written
- **Polish (Phase 3)**: Depends on Phase 2 completion

### Within User Story 1

- T002 must complete before T005 (tests reference `runUncomplete`)
- T003 and T004 are independent of each other and of T002 (different edit sites in `cli.go`)
- T001 must complete before T002 and T005

### Parallel Opportunities

- T003 and T004 can be written in parallel (both edit `cli.go` but at different locations — serialize within one edit session)
- T006 and T007 can run in parallel

---

## Parallel Example: User Story 1

```
# After T001 lands:
T002 — implement runUncomplete in task.go
T003 — add dispatch case in cli.go
T004 — add usage line in cli.go

# After T002, T003, T004:
T005 — write four tests
```

---

## Implementation Strategy

### MVP (this feature is a single story — all tasks are MVP)

1. T001 — extend fake service (foundational)
2. T002 → T003 → T004 → T005 — implement and test
3. T006 → T007 — verify no regressions

Total: 7 tasks across 3 files.
