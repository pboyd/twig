---
description: "Task list for Pomodoro Progress Display"
---

# Tasks: Pomodoro Progress Display

**Input**: Design documents from `/specs/031-pomodoro-progress-display/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included. This codebase is test-heavy (table-driven render tests + DB-backed handler tests that skip without `DATABASE_URL`); the rendering contract defines an explicit test matrix, so tests are written first (TDD) within each story.

**Organization**: Tasks are grouped by user story. All paths are relative to the Go module at `services/twig/` unless noted; `make proto` runs from the repo root.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 (task-tree row), US2 (planning-tab row), US3 (active glyph)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Shared palette tokens used by the renderer (Foundational) and the active glyph (US3)

- [X] T001 [P] Add adaptive palette tokens `pomodoroDone` (red, e.g. Light `#CC0000` / Dark `#FF5555`) and `pomodoroOver` (yellow, e.g. Light `#B58900` / Dark `#FFD75F`) to `internal/tui/theme.go`, following the existing `AdaptiveColor` style (no inline colors — Principle III)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Completed-pomodoro-count data plumbing + the shared glyph renderer. Required by BOTH US1 and US2.

**⚠️ CRITICAL**: US1 and US2 cannot be completed until this phase is done. (US3 depends only on T001.)

### Backend — completed count on ListTasks (contract: `contracts/listtasks-completed-count.md`)

- [X] T002 Add `int32 completed_pomodoro_count = 8;` to the `Task` message in `proto/task/v1/task.proto` with a comment marking it read-only/server-computed
- [X] T003 Regenerate protobuf + ConnectRPC stubs by running `make proto` from the repo root (updates `gen/` — do not hand-edit) (depends on T002)
- [X] T004 [P] Add the `CountCompletedPomodorosByTask :many` query (per-user, `GROUP BY task_id`, completed only) to `db/queries/pomodoro.sql` per the contract
- [X] T005 Regenerate sqlc code by running `sqlc generate` in `services/twig/` (updates `internal/db/` — do not hand-edit) (depends on T004)
- [X] T006 Populate `completed_pomodoro_count` per task in the `ListTasks` handler in `internal/handler/task.go`: fetch counts via `CountCompletedPomodorosByTask`, build a `map[int64]int64`, and set the field (default 0) when mapping each db row to a proto `Task` (depends on T003, T005)
- [X] T007 [P] Add a DB-backed handler test in `internal/handler/task_test.go` asserting `ListTasks` returns the correct `completed_pomodoro_count` (e.g. a task with 2 completed pomodoros → 2; a task with none → 0), following the existing `newTestHandler` pattern that skips without `DATABASE_URL` (depends on T006)

### Shared renderer (contract: `contracts/pomodoro-row-rendering.md`)

- [X] T008 [P] Write the table-driven test `renderPomodoroRow` in `internal/tui/pomodoro_row_test.go` covering the contract test matrix (empty-state, 5/0, 5/2, 5/5, 5/6, 3/1, 0/2, and a plain-mode case), asserting glyph counts and per-segment styling — write FIRST, expect failure
- [X] T009 Implement `renderPomodoroRow(estimate, completed int, styled bool) string` in `internal/tui/pomodoro_row.go`: compute `n=max(est,comp)`, `done=min(comp,est)`, `remain=est-done`, `over=max(0,comp-est)`; emit `done`×bold-`pomodoroDone`, `remain`×`dim`, `over`×bold-`pomodoroOver`; return `""` when est==0 && comp==0; plain mode emits `n` unstyled 🍅 (makes T008 pass; uses T001 tokens)
- [X] T010 [P] If the tests need it, expose `renderPomodoroRow` to the `tui_test` package via `internal/tui/export_test.go` (match the existing export shim pattern)

**Checkpoint**: Tasks carry completed counts and the shared glyph renderer is green. User stories can now proceed.

---

## Phase 3: User Story 1 - Pomodoro progress on a task (Priority: P1) 🎯 MVP

**Goal**: The task-tree details pane shows the glyph row (replacing the numeric `Est:` line), and it stays accurate after pomodoros complete.

**Independent Test**: Select a task with estimate 5; see five dimmed 🍅. Complete two pomodoros → first two bold red. Complete through six → six glyphs, five red + one yellow. A task with no estimate and no completions shows no row.

- [X] T011 [P] [US1] Update `internal/tui/details_test.go`: assert `renderDetails` emits the glyph row for representative (estimate, completed) combos and omits it (no `Est:` line) when both are 0; adjust any existing assertions that expected `Est: N pomodoros`
- [X] T012 [US1] In `internal/tui/details.go`, replace the numeric `Est:` line (both styled and plain branches) with `renderPomodoroRow(int(task.GetEstimate()), int(task.GetCompletedPomodoroCount()), styled)`, rendered only when the row is non-empty (depends on T009, T003)
- [X] T013 [US1] In `internal/tui/update.go`, reload the tree after a pomodoro completes so counts refresh: batch `listTasksCmd(m.client)` into the `pomCompletedMsg` handler (alongside the existing hook/banner commands)

**Checkpoint**: US1 fully functional and independently testable (the spec's worked example passes — SC-005).

---

## Phase 4: User Story 2 - Pomodoro progress for a linked task on the planning tab (Priority: P2)

**Goal**: The planning-tab details pane shows the same glyph row for a plan entry linked to a task; event entries show no row.

**Independent Test**: Select a plan entry linked to a task with estimate 3 and 1 completed → three glyphs, first bold red, rest dim. An event entry shows no row. Coloring matches the task-tree pane.

- [X] T014 [P] [US2] Update `internal/tui/plan_view_test.go`: assert `renderPlanDetail` includes the glyph row for a linked task (with estimate/completed) and omits it for an event entry (TaskId == 0) and for a linked task with no pomodoros
- [X] T015 [US2] Add a `findTask(tree []*cli.TreeNode, taskID int64) *taskv1.Task` lookup helper in `internal/tui/pomodoro.go` next to `findTaskName` (returns nil when absent)
- [X] T016 [US2] Thread the linked task into the planning detail: in `internal/tui/view.go` (the `renderPlanDetail` call site) look up the entry's task via `findTask(m.tree, entry.TaskId)` and pass its estimate/completed to `renderPlanDetail`; in `internal/tui/plan_view.go` render `renderPomodoroRow(...)` for a linked task when non-empty (depends on T009, T015)

**Checkpoint**: US1 and US2 both work; both panes render identical glyph rows for the same task (SC-003).

---

## Phase 5: User Story 3 - Active pomodoro glyph red and bold (Priority: P3)

**Goal**: The leading 🍅 on the running-pomodoro status line is red and bold.

**Independent Test**: Start a pomodoro; the status-bar 🍅 renders red + bold. Plain mode keeps the existing `Pom …` fallback.

- [X] T017 [P] [US3] Add a test in `internal/tui/view_test.go` asserting the active-pomodoro status line's 🍅 carries the `pomodoroDone` bold styling when `styled`, and the plain fallback is unchanged
- [X] T018 [US3] In `internal/tui/view.go` `renderStatus`, style the leading 🍅 on the active-pomodoro line(s) (the running line and the confirm-quit line) with `pomodoroDone` + bold; leave the timer text and plain-mode fallback as-is (depends on T001)

**Checkpoint**: All three user stories independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T019 [P] Run `go test ./...` in `services/twig/` and resolve any fallout (including snapshot/assertion updates from the `Est:` → glyph-row change)
- [X] T020 Run `go build ./cmd/twig ./cmd/server` in `services/twig/` to confirm both binaries build
- [ ] T021 Execute `specs/031-pomodoro-progress-display/quickstart.md` manual verification (worked example 5→2→6, planning-tab parity, active glyph, plain mode via `| cat`)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (T001)**: No dependencies — start immediately.
- **Foundational (T002–T010)**: Backend chain T002→T003, T004→T005, then T006 (needs T003+T005), T007 (needs T006). Renderer T008→T009 (needs T001), T010 optional. **Blocks US1 and US2.**
- **US1 (P1)**: Needs T009 + T003. The MVP.
- **US2 (P2)**: Needs T009 + T016. Independent of US1.
- **US3 (P3)**: Needs only T001. Independent of the backend and of US1/US2.
- **Polish (T019–T021)**: After the desired stories are complete.

### Within Each User Story

- Write the story's `[P]` test task first (TDD), then implement.

### Parallel Opportunities

- **Foundational**: the backend chain (T002–T007) and the renderer (T008–T010) are independent tracks — run them in parallel. T004 is `[P]` relative to T002/T003.
- **After Foundational**: US1, US2, US3 can proceed in parallel (different files); US3 can even start right after T001.
- Test tasks T011, T015, T018 are `[P]` (different files).

---

## Parallel Example: Foundational phase

```bash
# Track A (backend): T002 → T003, and T004 → T005, then T006 → T007
# Track B (renderer): T008 (test) → T009 (impl), T010 if needed
# Tracks A and B touch disjoint files and can run concurrently.
```

## Parallel Example: after Foundational

```bash
# Different developers / different files:
Task: "US1 — glyph row in internal/tui/details.go (+ details_test.go, update.go)"
Task: "US2 — glyph row in internal/tui/plan_view.go (+ plan_view_test.go, view.go, pomodoro.go)"
Task: "US3 — active glyph in internal/tui/view.go (+ view_test.go)"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 Setup (T001).
2. Phase 2 Foundational (T002–T010) — backend count + shared renderer.
3. Phase 3 US1 (T011–T013).
4. **STOP and VALIDATE**: the spec's worked example (estimate 5 → 2 done → 6 done) renders exactly (SC-005).

### Incremental Delivery

1. Setup + Foundational → foundation ready.
2. US1 → task-tree glyph row (MVP, demo).
3. US2 → planning-tab parity (demo).
4. US3 → active glyph polish (demo).

---

## Notes

- `gen/` and `internal/db/` are generated — never hand-edit; regenerate via `make proto` / `sqlc generate`.
- The new proto field is additive; the web client is unaffected and need not regenerate.
- [P] = different files, no incomplete dependencies. Commit after each task or logical group.
- The numeric `Est:` line is intentionally replaced (not duplicated) by the glyph row — update existing details assertions accordingly.
