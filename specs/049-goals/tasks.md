# Tasks: Goals

**Input**: Design documents from `/specs/049-goals/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/goal-rpc.md, contracts/goal-cli.md

**Tests**: Included — the repo has established test conventions (plan §Testing, research D10) and every prior feature ships tests alongside implementation.

**Organization**: Tasks are grouped by user story. All server-side work is foundational: ConnectRPC handlers must implement the full generated service interface to compile, so the `GoalService` and `SetTaskGoal` implementations land in Phase 2 and every story phase is purely client-side. US1 (TUI goal lifecycle) is the MVP; US2 (task association), US3 (`twig goal` CLI), and US4 (ranking) are independent increments on top of the same foundation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 (TUI goal lifecycle), US2 (task association), US3 (CLI), US4 (ranking)

## Phase 1: Setup (API contract)

**Purpose**: Commit the API contract change and regenerate stubs — required by Principle II before any implementation.

- [X] T001 Create `api/proto/goal/v1/goal.proto` (GoalService, GoalState enum, Goal + request/response messages) and add `optional int64 goal_id = 11` to `Task` plus the `SetTaskGoal` RPC and messages in `api/proto/task/v1/task.proto`, exactly per `contracts/goal-rpc.md`; run `make proto` to regenerate `api/gen/` (server code will not compile until T004/T006 implement the new methods — that is expected mid-phase state)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema, queries, the complete server-side GoalService + task-association implementation, and the shared `internal/goal` domain logic every story consumes.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 [P] Create migration `services/twig/db/migrations/000010_goals.up.sql` / `.down.sql` per `data-model.md` — `goals` table (identity PK, `user_id` FK, non-blank `name` CHECK, `description` default `''`, nullable `due`, `state` TEXT CHECK in the four values default `'incubating'`, `position` INTEGER) with `goals_user_state_position_idx`, and `ALTER TABLE tasks ADD COLUMN goal_id BIGINT REFERENCES goals(id) ON DELETE SET NULL`
- [X] T003 Add goal queries to new `services/twig/db/queries/goal.sql` (create-at-bottom-of-incubating, get, list ordered by state+position, update name/description/due, set-state-append-to-group, list-state-group for reorder renumbering, delete) and task queries to `services/twig/db/queries/task.sql` (set/clear `goal_id`, ancestor-has-goal and descendant-has-goal recursive checks, include `goal_id` in task selects); run `sqlc generate` in `services/twig/` (depends on T002)
- [X] T004 Implement `services/twig/internal/handler/goal.go` — full `GoalService` per `contracts/goal-rpc.md`: CreateGoal (validate name like CreateTask, always incubating, bottom of group), GetGoal, ListGoals, UpdateGoal (full-replace name/description/due), SetGoalState (any transition, same-state no-op, append to destination group), ReorderGoal (before/after anchor, same-group validation, contiguous renumber — mirror the ReorderTask handler), DeleteGoal; user scoping from auth context throughout (depends on T001, T003)
- [X] T005 Implement `SetTaskGoal` in `services/twig/internal/handler/task.go` — `NotFound` for missing task/goal, `FailedPrecondition` when an ancestor has a goal or (on set) a descendant has one, clearing always allowed; populate `goal_id` in ListTasks/GetTask responses; add the re-parent nesting guard to UpdateTask (`FailedPrecondition` when a move would nest goal associations) per `data-model.md` (depends on T001, T003)
- [X] T006 Register the GoalService handler in `services/twig/cmd/server/main.go` (`goalv1connect.NewGoalServiceHandler(&handler.Goal{Queries: queries, Pool: pool})` inside the auth-middleware mux) (depends on T004)
- [X] T007 [P] Add `DATABASE_URL`-gated integration tests in `services/twig/internal/handler/goal_test.go` — create defaults (incubating, bottom position), name validation, all-direction state transitions with bottom-of-group placement and same-state no-op, reorder within group + cross-group anchor rejection, delete clears `tasks.goal_id` without touching tasks, cross-user isolation (depends on T004, T006)
- [X] T008 [P] Add association cases to `services/twig/internal/handler/task_test.go` — set/clear `goal_id`, subtree nesting rejections (ancestor and descendant), re-parent guard, `goal_id` present in list/get responses (depends on T005)
- [X] T009 [P] Create `internal/goal/goal.go` (root module) — state display order (Committed, Incubating, Completed, Archived), default-visible vs hidden states, state-name parsing/formatting for CLI args and rendering, and effective-goal computation (nearest self-or-ancestor `goal_id` over `[]*taskv1.Task`) plus association-root subtree selection for a goal, per `data-model.md`
- [X] T010 [P] Add `internal/goal/goal_test.go` — table-driven: group ordering, hidden-state filtering, state parse round-trips, effective goal for roots/children/grandchildren, subtree selection with multiple roots (depends on T009)

**Checkpoint**: Foundation ready — `go test ./...` and `cd services/twig && go test ./...` pass; quickstart §1–§2 server behavior verifiable with `grpcurl`/tests; user stories can proceed (in parallel if desired).

---

## Phase 3: User Story 1 - Capture and track long-term goals (Priority: P1) 🎯 MVP

**Goal**: A Goals tab — first in the TUI tab bar (startup stays on Tasks) — with a two-pane layout where the user creates, edits, deletes goals, moves them through all four states, sees them grouped by state, and toggles visibility of Completed/Archived.

**Independent Test**: With no task association or CLI involvement: open the TUI, `shift+tab` to Goals, create goals, edit fields, cycle all four states, verify grouping (Committed before Incubating), default hiding of Completed/Archived, the `c` reveal toggle, and the playful empty state.

### Implementation for User Story 1

- [X] T011 [US1] Add `tabGoals` as the first `tab` constant in `internal/tui/model.go` with a `goalState` struct (goals, cursor, showAll, loaded, err, mode/form fields), a `goalClient goalv1connect.GoalServiceClient` on `Model`, and an explicit `activeTab: tabTasks` in `newModel` so startup stays on Tasks (clarification 2026-06-11)
- [X] T012 [US1] Construct and wire the goal client through the TUI entrypoint in `internal/tui/tui.go` / `internal/tui/client.go`, mirroring the existing `planClient` wiring (depends on T011)
- [X] T013 [US1] Add goal commands and key handling in `internal/tui/update.go` — fetch `ListGoals` on tab entry; cursor nav across state groups; `n` new / `e`+`enter` edit (reuse the existing edit-form component for name/description/due, calling CreateGoal/UpdateGoal); `ctrl+d` delete with the surviving-tasks confirm copy; `i`/`o`/`d`/`v` SetGoalState with refresh and the playful complete/archive notices; `c` show-all toggle — all per `contracts/goal-cli.md` (depends on T012)
- [X] T014 [P] [US1] Add Goals-tab bindings and contextual help entries (new/edit/delete, state keys, toggle, nav) to `internal/tui/keymap.go` following the existing per-tab help pattern
- [X] T015 [US1] Create `internal/tui/goal_view.go` — two-pane themed rendering per `contracts/goal-cli.md`: left pane state-group headers + goals in position order with due dates, right pane selected goal's name/state/due/description plus associated-task subtrees via `internal/goal` (empty in this story), playful empty state; all styles from `internal/tui/theme.go` (depends on T013)
- [X] T016 [US1] Wire Goals into the tab bar (`Goals · Tasks · Plan · Report`) and view dispatch in `internal/tui/view.go`; include Goals in the tab-cycle order (depends on T015)
- [X] T017 [US1] Add `internal/tui/goal_view_test.go` — startup tab is Tasks, tab order, group ordering and headers, create-appends-to-bottom-of-Incubating, state change moves goal between groups, completed/archived hidden until `c`, empty-state copy, delete confirm copy (depends on T016)

**Checkpoint**: User Story 1 fully functional — goals live their whole lifecycle in the TUI without touching tasks.

---

## Phase 4: User Story 2 - Associate tasks with goals (Priority: P2)

**Goal**: Tasks link to goals from both sides — goal pane (create-attached, link, unlink) and task surfaces (TUI edit form, CLI `--goal` flags) — with the goal visible in task detail views only.

**Independent Test**: Associate and dissociate tasks via the goal pane, the TUI task edit form, and `twig task add/mod --goal`; verify the goal detail lists subtrees, inherited subtasks show the goal in their detail pane, tree rows are unchanged, nesting conflicts surface the playful error, and unlinked tasks behave exactly as today.

### Implementation for User Story 2

- [X] T018 [US2] Add goal-pane association actions in `internal/tui/update.go` — `a` opens the task form and creates the task then calls `SetTaskGoal` to attach it; `L` opens the existing task picker (`pickerState`, as in the Plan add-task flow) and links the chosen task's subtree; `U` picks among the goal's association roots and clears via `SetTaskGoal`; surface `FailedPrecondition` with the playful nesting message, per `contracts/goal-cli.md`
- [X] T019 [P] [US2] Add the **Goal** field to the task edit form in `internal/tui/edit.go` — cycles `none` → each visible (committed/incubating) goal by name; on save, a changed value calls `SetTaskGoal` (never UpdateTask) per research D2
- [X] T020 [P] [US2] Show `Goal: <name>` in the TUI task detail pane in `internal/tui/details.go` using effective-goal computation from `internal/goal` (inherited goals included); tree rows unchanged
- [X] T021 [P] [US2] Add `--goal <id>` to `twig task add` and `--goal <id>|none` to `twig task mod` in `internal/cli/task.go` (mod path calls `SetTaskGoal`), plus a `Goal: <name>` line in CLI task detail output and the playful nesting-conflict error, per `contracts/goal-cli.md`
- [X] T022 [US2] Add association tests — picker link/unlink flows, edit-form goal cycling, detail-pane `Goal:` line in `internal/tui/goal_view_test.go` + `internal/tui/details_test.go`; `--goal` flag parsing, `none` clearing, nesting-error rendering in `internal/cli/task_test.go` (depends on T018, T019, T020, T021)

**Checkpoint**: User Stories 1 and 2 work independently — goals can carry task subtrees from either side.

---

## Phase 5: User Story 3 - Manage goals from the CLI (Priority: P3)

**Goal**: `twig goal` covers list/add/show/mod/state/rm with `twig task`-style conventions, playful copy, and cross-surface visibility with the TUI.

**Independent Test**: From a shell only: run every subcommand from quickstart §1, verify grouping and `--all`, state-change errors listing valid states, playful not-found message with exit 1, and that CLI changes appear in the TUI (and vice versa).

### Implementation for User Story 3

- [ ] T023 [US3] Create `internal/cli/goal.go` — `twig goal [--all]` listing grouped by state in position order, `add [--due <date>] [--desc <text>] <name>`, `show <id>` (fields + associated task subtrees via ListTasks + `internal/goal`, with the no-tasks line), `mod <id> [<name>] [--due] [--desc]`, `state <id> <state>`, `rm <id>`; TTY-gated styling via `internal/cli/render.go`, all copy per `contracts/goal-cli.md`
- [ ] T024 [US3] Register `goal` in `internal/cli/cli.go` — `Run` dispatch, `runHelp` case, `printGoalUsage`, and root usage listing `goal` before `task` (depends on T023)
- [ ] T025 [US3] Add `internal/cli/goal_test.go` — listing groups/order, `--all`, add/mod/state/rm round-trips against a stub client, invalid state arg lists the four states, not-found copy + exit codes, empty-state copy, styled vs piped output (depends on T024)

**Checkpoint**: All CRUD + state operations available on both surfaces (SC-003).

---

## Phase 6: User Story 4 - Rank goals (Priority: P4)

**Goal**: `{`/`}` reorder the selected goal within its state group in the TUI; the order persists and applies to every listing.

**Independent Test**: Create three goals in one state, reorder with `{`/`}`, confirm no-op at group edges, restart the TUI and run `twig goal` — order identical in both.

### Implementation for User Story 4

- [ ] T026 [US4] Handle `{`/`}` on the Goals tab in `internal/tui/update.go` — call `ReorderGoal` with the adjacent same-group goal as before/after anchor, no-op at group edges or single-goal groups, refresh group order from the response; reuse the existing RankUp/RankDown bindings and add Goals-tab help entries in `internal/tui/keymap.go`
- [ ] T027 [US4] Add ranking tests in `internal/tui/goal_view_test.go` — reorder within group, edge no-ops, order preserved across simulated reload; assert CLI listing order follows `position` in `internal/cli/goal_test.go` (depends on T026)

**Checkpoint**: All four user stories independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, tone review, and end-to-end validation.

- [ ] T028 [P] Update `CLAUDE.md` — add `goal` to the CLI command list (cmd/twig commands line) and mention the Goals tab where the TUI is described
- [ ] T029 [P] Tone + consistency review (Principles III & IV quality gates): audit every new user-facing string in `internal/cli/goal.go`, `internal/cli/task.go`, `internal/tui/goal_view.go`, `internal/tui/update.go` against `contracts/goal-cli.md` copy; verify key bindings and styles match existing tabs
- [ ] T030 Run the full `specs/049-goals/quickstart.md` walkthrough against `make dev` (sections 1–5), plus `go test ./...`, `cd services/twig && go test ./...` (with `DATABASE_URL` for gated tests), and `go vet ./...`; verify SC-006 by confirming pre-existing task tests pass unmodified

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on T001 — BLOCKS all user stories (the server must compile and pass tests)
- **User Stories (Phases 3–6)**: All depend only on Phase 2 — independent of each other and parallelizable, with two soft seams: US2's T022 touches `goal_view_test.go` (created in US1's T017) and US4's T026/T027 touch US1 files, so when run sequentially the order US1 → US2 → US3 → US4 avoids merge friction
- **Polish (Phase 7)**: Depends on all implemented stories

### User Story Dependencies

- **US1 (P1)**: Foundational only — no other story
- **US2 (P2)**: Foundational only; integrates with US1's goal pane but is testable via task surfaces alone
- **US3 (P3)**: Foundational only — pure CLI, no dependence on US1/US2 code
- **US4 (P4)**: Foundational only; extends US1's tab (shares files with US1)

### Parallel Opportunities

- Phase 2: T002 ∥ T009; after T003: T004 ∥ T005; after those: T007 ∥ T008 ∥ T010
- Phase 3: T014 in parallel with T011–T013
- Phase 4: T019 ∥ T020 ∥ T021 (different files) once T018 is underway
- Phase 7: T028 ∥ T029
- With multiple developers after Phase 2: US1+US4 (developer A), US2 (developer B), US3 (developer C)

## Parallel Example: Phase 2

```bash
# After T001, launch together:
Task: "Migration 000010_goals in services/twig/db/migrations/"          # T002
Task: "internal/goal domain package in internal/goal/goal.go"           # T009

# After T003, launch together:
Task: "GoalService handler in services/twig/internal/handler/goal.go"   # T004
Task: "SetTaskGoal + guards in services/twig/internal/handler/task.go"  # T005
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: commit the contract, regenerate stubs
2. Phase 2: schema + full server implementation + `internal/goal` (CRITICAL — server must compile with the new service interfaces)
3. Phase 3: TUI Goals tab
4. **STOP and VALIDATE**: quickstart §3 lifecycle checks — goals are usable end-to-end with zero task involvement
5. Ship/demo if ready

### Incremental Delivery

1. Setup + Foundational → server fully capable, no UI yet
2. US1 → TUI lifecycle (MVP!) → validate independently
3. US2 → associations from both sides → validate via task surfaces
4. US3 → CLI parity (SC-003 met) → validate from a shell alone
5. US4 → ranking polish → validate persistence across restart
6. Phase 7 → docs, tone review, full quickstart + SC spot checks

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- Server work is deliberately all-foundational: ConnectRPC handler interfaces are all-or-nothing per service
- Each story phase ends at a checkpoint matching its spec acceptance scenarios
- Commit after each task or logical group; `tasks.md` checkboxes track progress
