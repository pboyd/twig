---
description: "Tasks for the change-task-parent TUI feature"
---

# Tasks: Change Task Parent in TUI

**Input**: Design documents in `/specs/017-tui-move-task/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/update-task.md`, `quickstart.md`

**Tests**: Included. The existing TUI in `services/todo/internal/tui/` is exercised via Go unit tests (see `update_test.go`, `view_test.go`, `tree_test.go`), and this feature follows the same convention.

**Organization**: Tasks are grouped by user story so each story is independently testable.

## Format

`- [ ] [TaskID] [P?] [Story?] Description`

- **[P]**: parallelizable (different file, no dependency on incomplete tasks)
- **[US1]/[US2]**: user-story tag (see `spec.md`)
- Setup, Foundational, and Polish tasks have no story tag

## Path Conventions

All code paths are under `services/todo/internal/tui/`. No proto or server changes.

---

## Phase 1: Setup (Shared Infrastructure)

No new dependencies, build steps, or scaffolding are required. Skipping.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the view-mode constant, key binding, and model state slot that both user stories depend on. Completing this phase makes either story unblockable.

- [x] T001 Add `Move` key binding (key `m`, help text "move (change parent)") to `KeyMap` and `DefaultKeyMap` in `services/todo/internal/tui/keymap.go`
- [x] T002 Add `modeMove` to the `viewMode` enum and add a `move *moveState` field on `Model` in `services/todo/internal/tui/model.go`
- [x] T003 Create `services/todo/internal/tui/move.go` with the `moveState` and `moveCandidate` struct definitions (fields per `data-model.md`) and an empty constructor `newMoveState(model *Model, taskID int64) *moveState` returning nil for now — a placeholder so US1 can wire it up

**Checkpoint**: Compile passes; no behavior change yet.

---

## Phase 3: User Story 1 — Move a task under a new parent (Priority: P1) 🎯 MVP

**Goal**: With a task selected in the list, pressing `m` opens a tree of incomplete tasks (current parent highlighted), the user navigates to a different incomplete task, presses Enter, and the task is reparented and re-rendered.

**Independent test**: Quickstart Scenario 1 — seed two top-level tasks each with a subtask, move a subtask under the other top-level task, verify the new tree and that it persists across TUI restart.

- [x] T004 [US1] Implement `buildCandidates` in `services/todo/internal/tui/move.go` — produce an ordered `[]moveCandidate` from `Model.tasks`: start with the `(no parent)` sentinel at index 0, then walk incomplete tasks in the same order/indentation as the main TUI tree, skipping the task being moved and its descendants. Reuse helpers from `tree.go` (extract a small unexported helper there if needed; do not duplicate ordering logic)
- [x] T005 [US1] Implement `newMoveState` in `services/todo/internal/tui/move.go` to call `buildCandidates` and initialize `cursor` to the index whose `taskID` matches the moving task's current `parent_id` (or 0 if top-level)
- [x] T006 [US1] Implement `(*moveState).Update(msg tea.Msg, m *Model) (tea.Cmd, bool)` in `services/todo/internal/tui/move.go` handling Up/Down (cursor), Enter (return a tea.Cmd that calls `UpdateTask` with the task's current name/description/due and the new `parent_id`), Esc (return a cancel sentinel), with bounds-checked navigation. Return `(cmd, true)` when the dialog should remain open, `(cmd, false)` when it should close
- [x] T007 [US1] Implement `(*moveState).View(width, height int) string` in `services/todo/internal/tui/move.go` — render the candidate tree with indentation, highlight the cursor row, and render the `errMsg` footer (empty when no error). Use the existing Lipgloss styles consistent with `edit.go`'s modal styling
- [x] T008 [US1] In `services/todo/internal/tui/update.go`, add a `case key.Matches(msg, m.keys.Move)` branch in the `modeList` handler that (a) returns immediately if no task is selected, (b) sets `m.move = newMoveState(m, selectedTaskID)`, (c) switches `m.mode = modeMove`
- [x] T009 [US1] In `services/todo/internal/tui/update.go`, add a `case modeMove:` branch that delegates to `m.move.Update(msg, m)` and, when it returns `keepOpen == false`, resets `m.mode = modeList` and `m.move = nil`
- [x] T010 [US1] In `services/todo/internal/tui/update.go`, add an `updateTaskResultMsg` (or extend the existing edit-saved message family) handler that, on success, refreshes the task list via the same path `editSavedMsg` uses today and on failure assigns the error string to `m.move.errMsg` and keeps `m.mode == modeMove`
- [x] T011 [US1] In `services/todo/internal/tui/view.go`, when `m.mode == modeMove`, overlay `m.move.View(width, height)` on top of the task list view (same pattern as `modeEdit`/`modeHelp`)
- [x] T012 [P] [US1] Add `services/todo/internal/tui/move_test.go` with tests: `buildCandidates` excludes the moving task and its descendants; sentinel is at index 0; `newMoveState` pre-selects the current parent (and the sentinel when current parent is unset); Up/Down navigation is bounds-checked; Enter returns a command targeting the chosen parent id (or unset); Esc returns the close signal without a command; an error response populates `errMsg` and keeps the dialog open
- [x] T013 [P] [US1] Extend `services/todo/internal/tui/update_test.go` with: pressing `m` in `modeList` with a selection enters `modeMove` and seeds `move` with the selected task; pressing `m` with no selection is a no-op; a successful `UpdateTask` response returns to `modeList` and triggers a refresh

**Checkpoint**: A user can reparent any incomplete task under any other incomplete task and cancel without effect.

---

## Phase 4: User Story 2 — Promote a task to top-level (Priority: P2)

**Goal**: From the move dialog, selecting the `(no parent)` sentinel and pressing Enter clears the task's parent so it becomes a top-level task. When the dialog opens for a task that is already top-level, the sentinel is pre-selected.

**Independent test**: Quickstart Scenario 2 — open the dialog on a child task, pick `(no parent)`, confirm, verify the task is now at the root; reopen the dialog on a root task and verify the sentinel row is pre-selected.

(Most code paths needed for US2 are shared with US1; the work below is the narrow additional coverage.)

- [x] T014 [US2] Confirm `(*moveState).Update`'s Enter handler builds an `UpdateTaskRequest` with `parent_id` UNSET (not zero) when `candidates[cursor].taskID == 0`. Adjust the implementation in `move.go` if needed
- [x] T015 [P] [US2] In `services/todo/internal/tui/move_test.go`, add tests: selecting the sentinel and pressing Enter sends an `UpdateTaskRequest` with `ParentId == nil`; opening the dialog on a task whose `parent_id` is unset places the cursor at index 0 (the sentinel)

**Checkpoint**: A user can promote any task to top-level from the dialog.

---

## Phase 5: Polish & Cross-Cutting Concerns

- [x] T016 Update `services/todo/internal/tui/help.go` to list the new `m` binding so it appears on the `?` help screen
- [x] T017 Run `cd services/todo && go test ./...` and confirm all tests pass
- [x] T018 Run `cd services/todo && go build -o todo ./cmd/todo` and execute Scenarios 1–7 in `specs/017-tui-move-task/quickstart.md` against `make dev`

---

## Dependencies

```
Phase 2 (T001-T003)
   │
   ├── Phase 3 (US1: T004 → T005 → T006 → T007 → T008 → T009 → T010 → T011, with T012, T013 in parallel)
   │      │
   │      └── Phase 4 (US2: T014 → T015)
   │
   └── Phase 5 (Polish, after both stories): T016 → T017 → T018
```

US2 depends on US1 because it reuses the dialog plumbing introduced in US1; it adds only the sentinel-specific behavior and its dedicated tests.

## Parallel Opportunities

- T012 and T013 can run in parallel with each other once T011 lands (different test files).
- T015 can run alongside any remaining US1 polish once T014 is in.

## MVP Scope

User Story 1 (Phase 2 + Phase 3) is the MVP — it delivers the headline capability ("change a task's parent in the TUI") and is independently shippable.

## Task Count Summary

- Phase 2 (Foundational): 3 tasks
- Phase 3 (US1, P1): 10 tasks
- Phase 4 (US2, P2): 2 tasks
- Phase 5 (Polish): 3 tasks
- **Total: 18 tasks**
