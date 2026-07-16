---
description: "Task list for 065-goal-task-filter"
---

# Tasks: Goal Task Filter Shortcut

**Input**: Design documents from `/specs/065-goal-task-filter/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/keybinding.md](./contracts/keybinding.md),
[quickstart.md](./quickstart.md)

**Tests**: Included. `plan.md` Change 3 specifies a per-requirement test table, and the repo's
`internal/tui/*_test.go` convention expects `Model.Update` coverage for every keybinding.

**Organization**: Grouped by user story. US1 is the MVP and is independently shippable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 / US2, mapping to the user stories in spec.md
- Exact file paths are given in every task

## Path Conventions

Root CLI/TUI module (`github.com/pboyd/twig`), all paths relative to the repo root. This feature touches
`internal/tui/` **only** — `api/`, `services/twig/`, and `services/twig-web/` are untouched
(`plan.md` Structure Decision).

## Scope note

This feature composes an expression the user could have typed and pushes it through the existing filter
path. Production code is ~25 lines across two files. Most of the work below is test coverage and the
foundational test scaffolding that does not yet exist. **US2 is expected to require no production code
beyond US1** — see Phase 4.

---

## Phase 1: Setup

**Purpose**: Establish a known-good baseline before touching anything.

- [ ] T001 Confirm a clean baseline by running `go test ./...` from the repo root; record that it passes before any change, so later failures are unambiguously attributable to this feature

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Test scaffolding and the key binding that every story below depends on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete. T002 in particular is not
optional plumbing — without it, the first test that exercises this shortcut nil-panics.

- [ ] T002 Add a `FilterTasks` method to `fakeTaskClient` in `internal/tui/update_test.go` (~line 769). The fake embeds `taskv1connect.TaskServiceClient` as a nil interface and does **not** implement `FilterTasks`, so any test that triggers a filter panics today. Capture the request in a `lastFilterReq *taskv1.FilterTasksRequest` field and return a configurable `filterIDs []int64` / `filterErr error`, matching the existing fake's field conventions.
- [ ] T003 [P] Add export shims to `internal/tui/export_test.go`: `ExportFilterInputValue(m Model) string` returning `m.filterInput.Value()` (required to assert post-condition P2 in `contracts/keybinding.md` — `ExportFilterExpr` already exists but reads a *different* field and is not a substitute); `ExportSetGoalMode(m *Model, mode int)`; mode constants `ExportGoalModeEdit` and `ExportGoalModePickLink` (`ExportGoalModeList` and `ExportGoalModeStatusHistory` already exist); and `ExportSetShowAll(m *Model, v bool)` for the FR-009 passthrough test
- [ ] T004 Declare the `GoalGoToTasks key.Binding` field in the Goals-tab group of `KeyMap` in `internal/tui/keymap.go`, and bind it in `DefaultKeyMap()` as `key.WithKeys("ctrl+t")` / `key.WithHelp("ctrl+t", "show goal's tasks")`. Do **not** modify the existing `PlanGoToTask` binding, which independently declares `ctrl+t` for the Plan tab (FR-008; `contracts/keybinding.md` §1)

**Checkpoint**: The binding exists, tests can fake a filter round-trip and inspect the filter input.

---

## Phase 3: User Story 1 - Jump from a goal to its tasks (Priority: P1) 🎯 MVP

**Goal**: Pressing `ctrl+t` on a goal lands the user on the Tasks tab, scoped to that goal's whole tree,
without them ever knowing the goal's ID.

**Independent Test**: Create a goal with a nested subtask plus a second unrelated goal, put the cursor on
the first, press `ctrl+t`, and confirm the Tasks tab shows only the first goal's tree — parent and nested
subtask — with `^goal_id=<id>` in the filter bar.

### Tests for User Story 1 ⚠️

> Write these first and confirm they FAIL before T007.

- [ ] T005 [US1] Create `internal/tui/goal_filter_test.go` with table-driven `Model.Update` tests over synthetic `tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl}`, built via the existing `ExportNewGoalModel(taskClient, goals)` helper, covering: **(a)** the jump sets `activeTab == tabTasks` and `ExportFilterInputValue(m) == "^goal_id=<id>"` and dispatches `FilterTasks` with that exact expression (FR-001, FR-002); **(b)** an empty goals list is a no-op — `activeTab` unchanged, nil command returned (FR-006); **(c)** the shortcut is inert in `goalEdit`, `goalPickLink`, and `goalStatusHistory` sub-modes (FR-007); **(d)** `showAll` reaches `FilterTasks` unmodified in both true and false states (FR-009). Assert the expression is the bare relational condition — no `completed=false` appended (`contracts/keybinding.md` §2)
- [ ] T006 [P] [US1] Add the Plan-tab regression test in `internal/tui/plan_update_test.go`: `ctrl+t` on a plan entry still jumps to the Tasks tab with the cursor on that entry's task and **no** filter applied. Different file from T005, so parallelizable. This is the guard for the shared key (FR-008) and is non-negotiable per `plan.md` Change 3

### Implementation for User Story 1

- [ ] T007 [US1] Implement the `key.Matches(msg, m.keys.GoalGoToTasks)` case in `handleGoalsKey` in `internal/tui/update.go`, placed in the main action switch near the `GoalAddTask` case (~line 1439) so the existing sub-mode early returns provide FR-007 gating with no new checks (research R4). Follow the ordering in `plan.md` Change 2 exactly: guard the cursor → resolve the goal from the in-scope `visible` slice → build `expr` via `fmt.Sprintf("^goal_id=%d", g.Id)` → set `activeTab`/`keys.GoalMode`/errors → **`m.filterInput.SetValue(expr)`** → blur → `mode = modeList` → `filterInvalid = false` → `filterGen++` → `return m, filterCmd(m.client, expr, m.showAll, m.nowOrDefault(), m.filterGen)`. **`SetValue` MUST precede the dispatch**: `handleFilterResult` (`update.go:199`) records the active filter as `m.filterExpr = m.filterInput.Value()`, reading the input widget rather than the dispatched expression (research R3). Do not assign `m.filterExpr` here, and do not position the cursor — `handleFilterResult` owns both
- [ ] T008 [US1] Run `go test ./internal/tui/` and confirm T005 and T006 now pass

**Checkpoint**: US1 is fully functional and shippable on its own. The shortcut works; a user can reach any
goal's tasks with one keystroke.

---

## Phase 4: User Story 2 - Adjust or clear the goal filter after arriving (Priority: P2)

**Goal**: The filter the shortcut applied behaves exactly like a hand-typed one — editable, clearable,
and survives task mutations.

**Independent Test**: After a `ctrl+t` jump, press `/` and confirm the input is pre-filled with the goal
expression; press `esc` in the list and confirm the full task list returns.

**Expected production code: none.** This story is delivered by T007's `SetValue`-first ordering. These
tests exist to *prove* that and to lock it against regression — if any fail, the fix belongs in T007's
ordering, not in new code. Adding a US2-specific code path would create the second filter behavior that
`contracts/keybinding.md` §2 forbids.

### Tests for User Story 2 ⚠️

- [ ] T009 [US2] Extend `internal/tui/goal_filter_test.go` with post-arrival tests: after dispatching a successful `filterResultMsg`, `ExportFilterExpr(m) == "^goal_id=<id>"` (FR-003, post-condition P4); the filter input is blurred and `mode == modeList` (P6); and reopening with `/` presents the expression pre-filled for editing (US2 scenario 1)
- [ ] T010 [US2] Add the replacement test to `internal/tui/goal_filter_test.go`: seed a *different* active filter via the existing `ExportSetFilterState(&m, "completed=false", ids)` plus a matching `filterInput` value, then jump from a goal and assert both `filterInput` and — after the result — `filterExpr` hold only `^goal_id=<id>`, with no trace of the prior expression and no `AND`-merge (FR-004, post-condition P5)
- [ ] T011 [US2] Add the clear-path test to `internal/tui/goal_filter_test.go`: after a jump, `esc` in the list routes through the existing `clearFilter()` and restores the unfiltered task list (FR-003, US2 scenario 2)
- [ ] T012 [US2] Run `go test ./internal/tui/` and confirm T009–T011 pass with no production change beyond T007. If any fail, correct the ordering in T007 rather than adding a branch

**Checkpoint**: Both stories work. The shortcut's filter is provably indistinguishable from a typed one.

---

## Phase 5: Polish & Cross-Cutting Concerns

- [ ] T013 Add `GoalGoToTasks` to the `GoalMode` branch of `FullHelp` in `internal/tui/keymap.go` (FR-010). Same file as T004, so sequential. Do **not** add it to the `GoalMode` `ShortHelp` row, which is already near capacity (`contracts/keybinding.md` §1)
- [ ] T014 [P] Add a help-listing test in `internal/tui/goal_filter_test.go` asserting the Goals `FullHelp` contains the `ctrl+t` binding (FR-010)
- [ ] T015 [P] Verify Principle IV compliance: confirm the feature introduces no new user-facing strings beyond the help label, and that an empty goal (FR-004) routes to the existing "no tasks match" state. If implementation surfaced a need for goal-specific empty copy, it must be warm rather than terse per `contracts/keybinding.md` §5 and match the register of `goal_view.go:268`
- [ ] T016 Run `gofmt -l internal/tui/` and `go vet ./internal/tui/`; resolve any output
- [ ] T017 Run the full suite `go test ./...` from the repo root and confirm green, including the untouched `services/twig` filter tests
- [ ] T018 Walk the nine manual scenarios in [quickstart.md](./quickstart.md) against a live TUI (`make dev`, then `go build -o twig ./cmd/twig && ./twig`), with fixture goals A (nested subtask + a completed task), B (unrelated open task), and C (no tasks). Scenario 8 — Plan-tab `ctrl+t` unchanged — is a release blocker

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on T001 — **blocks both user stories**
- **US1 (Phase 3)**: Depends on Phase 2. Independently shippable
- **US2 (Phase 4)**: Depends on **US1's T007** — it verifies that implementation's ordering, so unlike a typical story it is not parallelizable with US1
- **Polish (Phase 5)**: Depends on US1; T013/T014 need T004

### Task-level dependencies

```text
T001 (baseline)
  └─> T002 (FilterTasks fake) ─┐
      T003 (export shims) ─────┼─> T005 (US1 tests) ─┐
      T004 (binding) ──────────┘   T006 (plan guard) ┼─> T007 (impl) ─> T008 (verify)
                                                      │                    │
                                   T013 (help) ───────┘                    ├─> T009,T010,T011 (US2 tests)
                                     └─> T014 (help test)                  │      └─> T012 (US2 verify)
                                                                           └─> T016,T017,T018 (polish)
```

### Story independence

US1 stands alone and is the MVP. US2 is a verification layer over US1's implementation rather than a
separate slice of function — an honest reading of a feature this small. It is still independently
*testable* (its scenarios exercise distinct behavior), just not independently *implementable*.

### Parallel Opportunities

- **T002 ∥ T003 ∥ T004** — three different files (`update_test.go`, `export_test.go`, `keymap.go`), no shared state. The largest parallel win here.
- **T005 ∥ T006** — `goal_filter_test.go` vs `plan_update_test.go`.
- **T014 ∥ T015** — test vs. copy review.
- T009–T011 all edit `goal_filter_test.go` and are therefore **sequential**, despite being one story.

---

## Parallel Example: Phase 2

```bash
# Launch the three foundational tasks together — different files, no shared state:
Task: "Add FilterTasks to fakeTaskClient in internal/tui/update_test.go"
Task: "Add ExportFilterInputValue + goal-mode shims to internal/tui/export_test.go"
Task: "Declare GoalGoToTasks binding in internal/tui/keymap.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 baseline → Phase 2 foundational (T002 first; nothing runs without the fake)
2. Phase 3: write T005/T006, watch them fail, implement T007, watch them pass
3. **STOP and VALIDATE**: quickstart scenarios 1, 5, 6, 7, 8 — especially 8
4. Shippable: the feature's entire user value is delivered here

### Incremental Delivery

1. Setup + Foundational → binding declared, tests can run
2. US1 → the jump works → validate → ship
3. US2 → prove the filter is ordinary → validate → ship
4. Polish → help discoverability, formatting, full quickstart

### Notes

- `[P]` = different files, no dependencies
- The riskiest line in the feature is the `SetValue`-before-dispatch ordering in T007; T009/T010 are what catch it if it regresses
- Commit after each task or logical group
- Per the constitution's workflow, artifacts and implementation land on `065-goal-task-filter`
