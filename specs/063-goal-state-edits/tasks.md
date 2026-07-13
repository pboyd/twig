# Tasks: Goal State Edits

**Input**: Design documents from `/specs/063-goal-state-edits/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/goal-service.md

**Tests**: Included — this repo tests goal behavior via table-driven unit tests (`goal_view_test.go`, `edit_test.go`, `update_test.go`, handler `goal_test.go`), and the plan enumerates test files. Test tasks are scoped to each story.

**Organization**: Tasks are grouped by user story so each story is an independently testable increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1, US2, US3 (setup/foundational/polish have no story label)
- Exact file paths are included in each task.

## Path Conventions

Three-module Go workspace: `api/` (shared proto), `services/twig/` (server), repo-root `internal/tui/` (TUI). Paths below are repo-relative.

---

## Phase 1: Setup (Shared Contract)

**Purpose**: Introduce the one contract change everything else builds on.

- [x] T001 Add `GOAL_STATE_HOLD = 5;` to the `GoalState` enum in `api/proto/goal/v1/goal.proto` (append after `GOAL_STATE_ARCHIVED`; add a comment: "still-pursued, not currently worked on")
- [x] T002 Regenerate ConnectRPC/protobuf stubs by running `make proto` (updates `api/gen/goal/v1/…`); confirm `GoalState_GOAL_STATE_HOLD` exists in generated Go

**Checkpoint**: `GOAL_STATE_HOLD` is available to both server and TUI modules.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Persist and map the new Hold state on the server. Blocks US1 and US2 (both rely on Hold being storable/round-trippable). Does not block US3.

- [x] T003 Create migration `services/twig/db/migrations/000012_goal_hold_state.up.sql` widening the constraint to `CHECK (state IN ('incubating','committed','completed','archived','hold'))`, and `000012_goal_hold_state.down.sql` restoring the 4-value constraint (down must fail loudly / be documented unsafe if any row has `state = 'hold'`)
- [x] T004 Extend the `ListGoals` `ORDER BY CASE state` in `services/twig/db/queries/goal.sql` so `'hold'` = 3, `'completed'` = 4, `'archived'` = 5; then run `cd services/twig && sqlc generate` to regenerate `internal/db/goal.sql.go`
- [x] T005 Add `case goalv1.GoalState_GOAL_STATE_HOLD → "hold"` to `goalStateToString` and `case "hold" → GOAL_STATE_HOLD` to `goalStateFromString` in `services/twig/internal/handler/goal.go` (keep the `UNSPECIFIED` rejection)
- [x] T006 [P] Add tests in `services/twig/internal/handler/goal_test.go` for the `hold`↔`GOAL_STATE_HOLD` round-trip and a `SetGoalState(hold)` call (mapping + destination-group re-rank), asserting `UNSPECIFIED` still errors

**Checkpoint**: Server can store, read back, and transition goals to/from Hold; server tests pass (`cd services/twig && go test ./...`).

---

## Phase 3: User Story 1 — Change a goal's state from the edit form (Priority: P1) 🎯 MVP

**Goal**: Replace the scattered per-state hotkeys with a single **State** selector in the goal edit form, covering all five states including Hold.

**Independent Test**: Open a goal's edit form, change the **State** selector to another value, save, and confirm the goal shows that state; confirm the former state keys (`i`/`o`/`v`) no longer change state.

- [x] T007 [US1] Add `state goalv1.GoalState` and `origState goalv1.GoalState` fields to `editFormModel`, plus a `cycleState(delta)` helper that walks `Incubating · Committed · Hold · Completed · Archived`, in `internal/tui/edit.go`
- [x] T008 [US1] Include the State field in the goal-only focus cycle (`cycleFocus`) and in `isDirty` (compare `state` vs `origState`) when `isGoal` is true, in `internal/tui/edit.go`
- [x] T009 [US1] Render a "State: <name>" selector row (goal forms only) in `editFormModel.View`, matching the existing label/value row style, in `internal/tui/edit.go`
- [x] T010 [US1] Propagate state on save: add `goalState goalv1.GoalState` and `stateChanged bool` to `editSavedMsg` and set them in `buildSaveMsg` (only when `isGoal`), in `internal/tui/edit.go`
- [x] T011 [P] [US1] Add a `"Hold"` case to `goalStateName` in `internal/tui/goal_view.go` (used by the selector and group headers)
- [x] T012 [US1] Seed `m.edit.state`/`m.edit.origState` from the selected goal in the goal edit-and-new form path (around the `goalToFakeTask` / `NewEditForm` / `NewRootForm` calls) in `internal/tui/update.go`
- [x] T013 [US1] In `handleGoalEditSaved`, when `msg.stateChanged`, batch `setGoalStateCmd(...)` with the existing `updateGoalCmd`/`createGoalCmd` (via `tea.Batch`), in `internal/tui/update.go`
- [x] T014 [P] [US1] Remove the `GoalSetIncubate`, `GoalSetCommit`, and `GoalSetArchive` bindings and drop them from the `GoalsHelp` rows in `internal/tui/keymap.go`
- [x] T015 [US1] Remove the `GoalSetIncubate` / `GoalSetCommit` / `GoalSetArchive` cases from `handleGoalsKey` in `internal/tui/update.go`
- [x] T016 [US1] Add tests in `internal/tui/edit_test.go` for `cycleState` order/wrap, dirty detection on state change, and `buildSaveMsg` setting `goalState`/`stateChanged`; add/extend a test asserting the removed keys no longer produce a state change

**Checkpoint**: Goal state is fully controllable from the edit form (all five states); old state hotkeys are inert. US1 is independently demoable.

---

## Phase 4: User Story 2 — Put a goal on hold so it's out of sight but not gone (Priority: P2)

**Goal**: Hold goals are hidden from the default Goals view and revealed by the existing "show all" toggle, grouped after Incubating and before Completed.

**Independent Test**: Set a goal to Hold, confirm it vanishes from the default view, toggle "show all" on to confirm it reappears under a Hold group in the right position, toggle off to confirm it hides again.

- [x] T017 [US2] Treat `GOAL_STATE_HOLD` like Completed/Archived in `visibleGoals` (hidden unless `showAll`) in `internal/tui/goal_view.go`
- [x] T018 [US2] Insert `GOAL_STATE_HOLD` into the `goalGroupHeaders` `showAll` ordering after Incubating and before Completed, in `internal/tui/goal_view.go`
- [x] T019 [P] [US2] Add tests in `internal/tui/goal_view_test.go`: Hold hidden by default, revealed with `showAll`, and grouped in the correct order/label

**Checkpoint**: Hold goals hide/reveal correctly and sit in the right group. US2 works on top of US1 (which supplies a way to set Hold) but tests independently with a Hold-state fixture.

---

## Phase 5: User Story 3 — Complete a goal with the same key used for tasks (Priority: P3)

**Goal**: Rebind goal completion to **Space** as a toggle — active goal → Completed, Completed → Committed — matching the Tasks tab.

**Independent Test**: Select an active goal, press Space → Completed (with a warm notice); select the Completed goal (show-all on), press Space → Committed; confirm hints show Space and not `d`.

- [x] T020 [US3] Remove the `GoalSetComplete` binding (`d`) from `keymap.go`; ensure the shared `Complete` (space) binding is surfaced in the `GoalsHelp` rows in `internal/tui/keymap.go`
- [x] T021 [US3] Replace the `GoalSetComplete` case in `handleGoalsKey` with a case matching `m.keys.Complete` that toggles state (current `COMPLETED` → `COMMITTED`; otherwise → `COMPLETED`) via `setGoalStateCmd`, with warm notices for both directions, in `internal/tui/update.go`
- [x] T022 [P] [US3] Add tests in `internal/tui/update_test.go`: Space on an active goal sets Completed; Space on a Completed goal sets Committed

**Checkpoint**: Space toggles goal completion exactly like a task; no goal-specific completion key remains.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T023 [P] Verify the full Goals help/keybinding hints read cleanly after removals (only Space for complete; no `i`/`o`/`v`/`d`), including any help-overlay layout in `internal/tui/keymap.go`
- [ ] T024 Run the manual verification walkthrough in `specs/063-goal-state-edits/quickstart.md` against `make dev` + `./twig` (Hold hide/reveal, edit-form State, Space toggle, existing goals untouched)
- [x] T025 Run both test suites and `go vet`: `go test ./...` (root) and `cd services/twig && go test ./...`; fix any regressions

---

## Dependencies & Execution Order

- **Phase 1 (Setup)** → blocks everything (the enum value).
- **Phase 2 (Foundational)** → blocks US1 and US2 (Hold must persist/round-trip). Does **not** block US3.
- **US1 (Phase 3)** → depends on Phase 2. Delivers state control incl. Hold.
- **US2 (Phase 4)** → depends on Phase 2; conceptually needs US1 to set Hold in practice, but tests with a Hold fixture independently. Touches `goal_view.go` (shares `goalStateName` added in T011).
- **US3 (Phase 5)** → depends only on Phase 1 (uses existing `COMPLETED`/`COMMITTED`). Can proceed in parallel with Phase 2/US1/US2, but shares `keymap.go` and `update.go` `handleGoalsKey` with US1 — coordinate edits (see below).
- **Polish (Phase 6)** → after all stories.

### File-contention notes (limits parallelism)

- `internal/tui/edit.go` — T007→T008→T009→T010 are sequential (same file).
- `internal/tui/update.go` (`handleGoalsKey` / goal-edit save) — T012, T013, T015 (US1) and T021 (US3) all edit this file; do them sequentially, not `[P]`.
- `internal/tui/keymap.go` — T014 (US1) and T020 (US3) both edit bindings/help; sequence them.
- `internal/tui/goal_view.go` — T011 (US1), T017 & T018 (US2) share this file; sequence.

## Parallel Opportunities

- **T006** (server test file) can run alongside T003–T005 once T005's mapping exists.
- **T011** (`goal_view.go` name) and **T014** (`keymap.go`) are `[P]` against the `edit.go` chain in US1.
- **T019** (`goal_view_test.go`) and **T022** (`update_test.go`) test files are `[P]` against implementation once their targets compile.
- Genuinely independent modules — server (`services/twig/**`) vs TUI (`internal/tui/**`) — can be worked by different developers after Phase 1.

## Implementation Strategy

- **MVP = Phase 1 + Phase 2 + Phase 3 (US1)**: the edit-form State selector with all five states and the hotkeys removed — the core "fewer keys, one place" value. Shippable on its own.
- **Increment 2 = US2**: Hold hides by default (the headline new capability), building directly on US1.
- **Increment 3 = US3**: Space-to-complete polish, aligning goals with tasks.
- Keep changes minimal per the Simplicity principle: reuse `SetGoalState` and the shared `Complete` binding; add exactly one enum value.
