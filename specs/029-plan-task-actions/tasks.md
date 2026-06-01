---

description: "Task list for Plan Task Actions"
---

# Tasks: Plan Task Actions

**Input**: Design documents from `/specs/029-plan-task-actions/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included. The TUI package follows a table-driven `tea.KeyMsg` dispatch testing convention (`internal/tui/*_test.go`, `export_test.go` shims, `fakeTaskClient`/`fakePlanClient`), and the plan + quickstart call for tests of the two new planning-tab actions. Tests are written first within each story.

**Organization**: Tasks are grouped by user story (US1 = complete, US2 = pomodoro) so each can be implemented and verified independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 (complete) or US2 (start pomodoro)

## Path Conventions

Single Go module at `services/twig/`. All paths below are relative to `services/twig/`. This is a TUI-only change — no `proto/`, `gen/`, `internal/db/`, `internal/handler/`, or `twig-web/` work.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm a clean baseline before changing the planning-tab key handler.

- [X] T001 Establish baseline: from `services/twig/`, run `go build ./cmd/twig` and `go test ./internal/tui/...` and confirm both are green before any change.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: None required.

No foundational/shared code is needed. Both stories independently add a `case` to the existing `handlePlanKey` (`planList` mode) switch in `internal/tui/update.go`; pomodoro reuses the existing `startPomCmd`, and complete adds its own small helper within US1. Per Principle I (YAGNI), no shared helper or new abstraction is introduced.

**Checkpoint**: Proceed directly to User Story 1.

---

## Phase 3: User Story 1 - Complete a linked task from the plan (Priority: P1) 🎯 MVP

**Goal**: From the planning tab, `space` toggles the completion of the task linked to the selected entry; the entry redraws as completed in place after a day reload, and every entry linked to the same task reflects the change.

**Independent Test**: On the planning tab with a task-linked entry selected, press `space` → linked task becomes complete (visible here and on the Tasks tab); press `space` again → it un-completes. On an event entry or empty plan, `space` is a no-op (with a playful notice for events).

### Tests for User Story 1 ⚠️ (write first, ensure they FAIL)

- [X] T002 [US1] Add a table-driven test in `internal/tui/update_test.go` for the planning-tab complete action, building a model with `buildPlanTestModel(fc)` plus a `&fakeTaskClient{}` assigned to `m.client`, `m.activeTab = tabPlanning`, `m.plan.mode = planList`, and seeded `m.plan.entries`. Cover: (a) incomplete task-linked entry selected + `space` → returns a non-nil cmd and the cmd ultimately drives the `CompleteTask` path; (b) completed task-linked entry + `space` → drives the `UncompleteTask` path; (c) event entry (`TaskId == 0`) + `space` → no mutation, `m.notice` set, no cmd that mutates; (d) empty `m.plan.entries` + `space` → no-op. Confirm these FAIL before T003/T004. (If `fakeTaskClient` lacks `CompleteTask`/`UncompleteTask` methods, extend it in `internal/tui/update_test.go` to record the last call.)

### Implementation for User Story 1

- [X] T003 [US1] Add `completePlanTaskCmd(taskClient taskv1connect.TaskServiceClient, day string, taskID int64, complete bool, entryID int32, notice string) tea.Cmd` in `internal/tui/update.go`: when `complete`, call `CompleteTask`; else `UncompleteTask`; on error return `planMutatedMsg{err: err}`; on success return `planMutatedMsg{notice: notice, highlightID: entryID}`. (Reuses the existing `planMutatedMsg` → `listPlanHighlightCmd` reload + notice + error-routing path.)
- [X] T004 [US1] In `handlePlanKey` (`planList` mode) in `internal/tui/update.go`, add `case key.Matches(msg, m.keys.Complete)`: guard `len(m.plan.entries) > 0`; let `entry := m.plan.entries[m.plan.cursor]`; if `entry.TaskId == 0` set the playful event notice (see `research.md`) and return; else clear `m.err`, compute the success notice and `return m, completePlanTaskCmd(m.client, m.plan.day, entry.TaskId, !entry.Completed, entry.Id, notice)`.
- [X] T005 [US1] Author/finalize the US1 user-facing copy (event-not-a-task notice + completion confirmation) in the warm/playful house tone per Principle IV and `research.md`, wired in T004.
- [X] T006 [US1] Run `go test ./internal/tui/...` and confirm the T002 tests now pass and no existing planning-tab tests regress.

**Checkpoint**: Completing/uncompleting a linked task from the plan works end-to-end and is independently testable — this is the MVP.

---

## Phase 4: User Story 2 - Start a pomodoro for a linked task from the plan (Priority: P2)

**Goal**: From the planning tab, `s` starts a pomodoro for the task linked to the selected entry, identical to the Tasks tab; inert on events and empty plans.

**Independent Test**: On the planning tab with a task-linked entry selected, press `s` → a pomodoro starts for that task and the running indicator shows it (also visible after switching to the Tasks tab). On an event entry or empty plan, `s` is a no-op (playful notice for events).

> Depends on Phase 3 only because both stories edit the same `handlePlanKey` switch and `update_test.go` (avoid merge conflict); behavior is independent.

### Tests for User Story 2 ⚠️ (write first, ensure they FAIL)

- [X] T007 [US2] Add a table-driven test in `internal/tui/update_test.go` for the planning-tab pomodoro action (model with `tabPlanning` + `planList` + seeded entries, task client capable of the pomodoro-start call — reuse `fakePomClient`/`fakeTaskClient` as the existing pomodoro tests do). Cover: (a) task-linked entry selected + `s` → returns the pomodoro-start cmd carrying `entry.TaskId`/`entry.Name`; (b) event entry (`TaskId == 0`) + `s` → no start, `m.notice` set; (c) empty `m.plan.entries` + `s` → no-op. Confirm these FAIL before T008.

### Implementation for User Story 2

- [X] T008 [US2] In `handlePlanKey` (`planList` mode) in `internal/tui/update.go`, add `case key.Matches(msg, m.keys.PomStart)`: guard `len(m.plan.entries) > 0`; let `entry := m.plan.entries[m.plan.cursor]`; if `entry.TaskId == 0` set the playful event notice and return; else clear `m.err` and `return m, startPomCmd(m.client, entry.TaskId, entry.Name)`.
- [X] T009 [US2] Author/finalize the US2 event-not-actionable pomodoro notice in the house tone per Principle IV and `research.md`, wired in T008.
- [X] T010 [US2] Run `go test ./internal/tui/...` and confirm the T007 tests pass with no regression.

**Checkpoint**: Both complete and start-pomodoro work from the planning tab, independently testable.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Help-text surfacing and final verification across both stories.

- [X] T011 In `internal/tui/keymap.go`, surface the new planning-tab actions in `PlanningMode` help: add `k.Complete` and `k.PomStart` to the `PlanningMode` branch of `ShortHelp` and/or `FullHelp` so `?` on the planning tab documents `space` (toggle complete) and `s` (start pomodoro).
- [X] T012 Update any `PlanningMode` help-rendering test in `internal/tui/help.go`/`*_test.go` if it asserts the exact set of planning-tab bindings, to include the two new actions.
- [X] T013 Run the full module test suite from `services/twig/`: `go test ./...` — confirm all green.
- [ ] T014 Execute `specs/029-plan-task-actions/quickstart.md` manually against `make dev` + a freshly built `./twig`: complete/uncomplete from plan, cross-tab consistency, FR-004 (same task on two entries), pomodoro from plan + cross-tab, and event/empty no-op cases.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: None.
- **User Story 1 (Phase 3)**: After Setup. Delivers the MVP.
- **User Story 2 (Phase 4)**: After Setup; sequenced after US1 only to avoid editing the same `handlePlanKey` switch / `update_test.go` concurrently. Behaviorally independent of US1.
- **Polish (Phase 5)**: After both stories are in.

### Within Each User Story

- Test task first (T002 / T007) and must FAIL before implementation.
- US1: helper (T003) before the key case (T004) before copy finalization (T005) before verify (T006).
- US2: key case (T008) before copy finalization (T009) before verify (T010).

### Parallel Opportunities

- This feature is small and concentrated in two files (`update.go`, `update_test.go`) plus `keymap.go`. There is **no safe within- or cross-story parallelism** on `update.go`/`update_test.go`; no tasks are marked `[P]`.
- If staffed by two people, the only clean split is: one finishes US1 (Phase 3) fully, then US2 (Phase 4) proceeds; the keymap help task T011 (`keymap.go`) can be done independently once either story's key is bound.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: baseline green.
2. Phase 3: complete/uncomplete from the plan, test-first.
3. **STOP and VALIDATE**: exercise US1 from the quickstart; demo if ready.

### Incremental Delivery

1. Setup → baseline.
2. US1 (complete) → test independently → MVP.
3. US2 (pomodoro) → test independently.
4. Polish: help text + full suite + manual quickstart.

---

## Notes

- No proto/DB/handler/generated-code/`twig-web` changes (see plan.md and contracts/plan-task-actions.md — no API delta).
- The `m.notice` status channel and `planMutatedMsg` (reload + notice + `tabAgnosticErr`) already exist from feature 028 and are reused as-is.
- Keys `space` (`Complete`) and `s` (`PomStart`) are currently unbound in `planList` mode — no collision with existing planning controls.
- New user-facing copy must pass Principle IV tone review (proposed strings live in research.md).
- Commit after each story checkpoint.
