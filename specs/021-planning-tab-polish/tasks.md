---

description: "Task list for Planning Tab Polish"
---

# Tasks: Planning Tab Polish

**Input**: Design documents from `/specs/021-planning-tab-polish/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Test tasks ARE included. The project's TUI is covered by string-view and reducer tests (`internal/tui/*_test.go`, `internal/cli/plan_grid_test.go`), and research.md §8 defines the test approach. Each behavior/render change gets a test, including a `!styled` assertion to protect the non-ANSI fallback.

**Organization**: Tasks are grouped by user story (from spec.md) in priority order. All work is in the single Go module at `services/todo/`; run tests with `cd services/todo && go test ./...`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1–US5 maps to the spec's user stories
- Exact file paths are given relative to `services/todo/`

---

## Phase 1: Setup

**Purpose**: Establish a green baseline before changing behavior.

- [ ] T001 Confirm the starting point is green: run `cd services/todo && go build ./... && go test ./...` and note the passing baseline (especially `internal/cli` grid tests and existing `internal/tui` tests).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Cross-cutting prerequisites for all stories.

**Note**: This feature has **no blocking foundational work** — it edits existing `internal/tui` files and makes one additive change to `internal/cli/plan_grid.go`, all scoped within their stories. User stories may begin immediately after Setup. (Phase intentionally minimal per the plan's Constitution Check, Principle I.)

**Checkpoint**: Baseline green → user stories can begin.

---

## Phase 3: User Story 1 - Cancel a running pomodoro from the Planning tab (Priority: P1) 🎯 MVP

**Goal**: The advertised `[x] cancel` (and the running-pomodoro quit confirmation) work on the Planning tab, identically to the Tasks tab.

**Independent Test**: Start a pomodoro on Tasks, switch to Planning, press `x` → pomodoro stops and the status bar reverts; with nothing running, `x` is inert; pressing `q` with a pomodoro running shows the `y/n` quit-confirm and both answers behave.

### Tests for User Story 1 ⚠️

- [ ] T002 [P] [US1] Reducer test: `x` (PomCancel) while Planning is active and a pomodoro is running issues `cancelPomCmd`; inert when none running — in `services/todo/internal/tui/plan_update_test.go` (or `update_test.go`), using the stub client + `export_test.go` shims.
- [ ] T003 [P] [US1] Reducer test: with `confirmingQuit` set on the Planning tab, `n`/`esc` cancels the quit and `y` quits — in `services/todo/internal/tui/update_test.go`.

### Implementation for User Story 1

- [ ] T004 [US1] In `services/todo/internal/tui/update.go` `handlePlanningKey`: (a) handle the running quit-confirmation (`y`/`n`/`esc`) at the top, mirroring `handleListKey`; (b) add a `key.Matches(msg, m.keys.PomCancel)` case in the planList branch that runs `cancelPomCmd(m.client)` when `m.pom != nil && !m.pom.completed`.

**Checkpoint**: Pomodoro cancel and quit-confirm work on Planning; `go test ./...` green.

---

## Phase 4: User Story 2 - Get help on the Planning tab (Priority: P1)

**Goal**: `?` opens a Planning help screen styled identically to the Tasks help, listing Planning keys (including pomodoro cancel), and dismiss returns to the grid with selection intact.

**Independent Test**: On Planning, press `?` → full-screen help with Planning bindings; press `?` again → grid returns, same entry selected.

### Tests for User Story 2 ⚠️

- [ ] T005 [P] [US2] Reducer test: `?` on Planning enters help; the dismiss key returns to planList with the prior `m.plan.cursor` intact — in `services/todo/internal/tui/update_test.go`.
- [ ] T006 [P] [US2] View test: help rendered while on the Planning tab shows Planning bindings (PlanningMode `FullHelp`), including a pomodoro-cancel entry — in `services/todo/internal/tui/view_test.go`.

### Implementation for User Story 2

- [ ] T007 [US2] In `services/todo/internal/tui/update.go` `handlePlanningKey`: route `m.keys.Help` to enter help (reuse `modeHelp`, or add a `planHelp` `planMode` value) and handle dismissal back to planList without losing `m.plan.cursor`.
- [ ] T008 [US2] In `services/todo/internal/tui/plan_view.go` (and/or `view.go` `viewPlanning`): when the help flag is set on Planning, render `m.viewHelp()` so the shared `help.Model` full-help view is shown (consistent style/dismissal with Tasks).
- [ ] T009 [US2] In `services/todo/internal/tui/keymap.go`: ensure the Planning branches of `ShortHelp()`/`FullHelp()` surface `PomCancel` (and any other status-bar-advertised control) so the help screen matches the advertised keys.

**Checkpoint**: `?` opens/closes Planning help; bindings match the status bar; tests green.

---

## Phase 5: User Story 3 - See sub-tasks when scheduling a task (Priority: P2)

**Goal**: The add-task picker shows the full incomplete-task tree with nested sub-tasks (reusing the Tasks-list flattener), and a sub-task can be selected and scheduled.

**Independent Test**: With a task that has sub-tasks, press `a` on Planning → picker shows sub-tasks nested under their parent; select a sub-task, set a time → entry appears on the grid.

### Tests for User Story 3 ⚠️

- [ ] T010 [P] [US3] Reducer test: after `planTasksMsg` with a parent+child tree, `m.plan.picker.visible` includes the sub-task rows (with tree connectors), and selecting a sub-task opens the task-time form with that task's id — in `services/todo/internal/tui/plan_update_test.go`.
- [ ] T011 [P] [US3] Unit test: `allTaskIDs` returns every id (parents + descendants) for a nested tree — in `services/todo/internal/tui/plan_update_test.go`.

### Implementation for User Story 3

- [ ] T012 [US3] Add `allTaskIDs(tree []*cli.TreeNode) map[int64]bool` helper in `services/todo/internal/tui/plan_update.go` (or `tree.go`).
- [ ] T013 [US3] In `services/todo/internal/tui/update.go` `planTasksMsg` handler: build `m.plan.picker.visible` via `buildVisible(msg.tree, allTaskIDs(msg.tree), false, nil)` and set `picker.expanded` to that map, so sub-tasks are visible (replacing the empty-map call at the current `buildVisible(msg.tree, make(map[int64]bool), false, nil)`).

**Checkpoint**: Picker shows sub-tasks consistent with the move dialog; sub-tasks schedulable; tests green.

---

## Phase 6: User Story 4 - Planning tab matches the Tasks tab's look and feel (Priority: P2)

**Goal**: Blue accent theme on the Planning grid, a polished tab bar, no redundant in-pane titles (both tabs), and the status line pinned to the bottom row.

**Independent Test**: Side-by-side, the Planning grid uses the accent highlight (not monotone bold), the tab bar clearly marks the active tab, neither tab shows a redundant `Tasks`/`Planning` pane title, and the status line sits on the bottom row at tall and short heights.

### Tests for User Story 4 ⚠️

- [ ] T014 [P] [US4] CLI regression + opt-in test: `RenderGrid` with `GridOptions{Styled:false}` is byte-for-byte unchanged from current output; with `Styled:true` (and TTY) the selected entry uses the accent highlight — in `services/todo/internal/cli/plan_grid_test.go`.
- [ ] T015 [P] [US4] View test: Tasks-tab views render no redundant left-pane `Tasks` title while keeping the `Details` title; the tab bar marks the active tab — in `services/todo/internal/tui/view_test.go`.
- [ ] T016 [P] [US4] View test: the Planning view fills content height so `renderStatus()` is on the bottom row at small and large heights (assert in both styled and `!styled` modes) — in `services/todo/internal/tui/plan_view_test.go`.

### Implementation for User Story 4

- [ ] T017 [US4] Extend `GridOptions` with the additive `Styled` field and apply the accent highlight for the selected entry (and accent box borders) in `applySelection`/border rendering in `services/todo/internal/cli/plan_grid.go`, keeping the `Styled:false` path identical to today (per `contracts/grid-options.md`).
- [ ] T018 [US4] In `services/todo/internal/tui/plan_grid.go` `planGridOptions()`: set `Styled: m.styled` so the TUI grid is themed while non-ANSI terminals stay plain.
- [ ] T019 [US4] In `services/todo/internal/tui/view.go`: drop the redundant `"Tasks"` left-pane title in `viewList`, `viewWithForm`, and `viewWithMove` (pass `""`); keep the right pane's `"Details"` title.
- [ ] T020 [US4] In `services/todo/internal/tui/plan_view.go` `renderTabBar`: polish the tab bar using the theme palette (accent for the active tab, dim separator) consistent with the Tasks styling.
- [ ] T021 [US4] In `services/todo/internal/tui/view.go` `viewPlanning`: pad the content region to the full computed content height so `renderStatus()` is pinned to the bottom row (interim single-pane fix; preserved by US5's two-pane layout). Theme the day header/selection consistent with the accent palette in `plan_view.go`.

**Checkpoint**: Planning matches the Tasks look; status pinned to bottom; CLI output unchanged; tests green.

---

## Phase 7: User Story 5 - Review entry details beside the plan (Priority: P3)

**Goal**: Two-pane Planning layout — calendar grid left, read-only details pane right — that updates as the selection moves, mirroring the Tasks tab split.

**Independent Test**: On Planning, the grid is on the left and a details pane on the right; moving the selection updates the details (name, `HH:MM–HH:MM` window, duration; task entries also show linked task + completion; events omit task-only fields); an empty day shows a placeholder; narrow terminals degrade gracefully.

### Tests for User Story 5 ⚠️

- [ ] T022 [P] [US5] View test: `viewPlanning` renders a left grid pane + right details pane; details reflect the selected entry and differ for a task entry vs. an event entry; empty day shows a placeholder; assert both styled and `!styled` paths — in `services/todo/internal/tui/plan_view_test.go`.
- [ ] T023 [P] [US5] Unit test: `renderPlanDetail` projects a `PlanEntry` to the expected read-only fields (and omits task-only fields for events) — in `services/todo/internal/tui/plan_view_test.go`.

### Implementation for User Story 5

- [ ] T024 [US5] Add `renderPlanDetail(entry *planv1.PlanEntry, width int, styled bool) string` in `services/todo/internal/tui/plan_view.go` (read-only; name as accent header mirroring `renderDetails`; window/duration; linked task + completion for task entries; placeholder when nil).
- [ ] T025 [US5] Restructure `viewPlanning` in `services/todo/internal/tui/view.go` to a two-pane layout: left = day header + grid, right = `renderPlanDetail`, composed with `paneBox` + `lipgloss.JoinHorizontal` (styled) and the `splitLines`/`padRightAnsi` row-join fallback (non-styled), reusing `viewList`'s height arithmetic so the status line stays bottom-pinned. Left pane title `""`, right pane title `"Details"`.
- [ ] T026 [US5] Wire the details pane to `selectedPlanEntry(m.plan.entries, m.plan.cursor)` so it updates on `↑`/`↓`, and confirm narrow-terminal degradation matches the Tasks tab.

**Checkpoint**: Two-pane Planning works; details track selection; tests green.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T027 Run `cd services/todo && gofmt -l . && go vet ./... && go test ./...`; fix any formatting/vet/test issues.
- [ ] T028 [P] Execute the `quickstart.md` manual walkthrough end-to-end (all 7 sections) against `make dev` + a built `todo` binary; confirm visual parity and the non-ANSI fallback.
- [ ] T029 [P] Verify no regressions to prior Planning behaviors (now-marker auto-advance, add/edit/remove/clear, day nav, refresh, completion strike-through, modal tab-switch blocking) — spot-check via existing tests and quickstart (SC-007 / FR-015).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none — start immediately.
- **Foundational (Phase 2)**: none blocking.
- **User Stories (Phases 3–7)**: each depends only on Setup. They can be delivered in priority order (US1 → US2 → US3 → US4 → US5) or in parallel by file ownership.
- **Polish (Phase 8)**: after the desired stories are complete.

### Cross-story file notes (avoid same-file conflicts)

- `update.go` is edited by US1 (T004), US2 (T007), and US3 (T013). These edits are independent (different handlers/cases) but touch one file — sequence them or coordinate to avoid merge conflicts; do **not** mark them `[P]` together.
- `view.go` `viewPlanning` is edited by US4 (T021, interim status fix) and US5 (T025, two-pane). **US5's T025 supersedes T021's single-pane padding** while preserving the bottom-pinned status. If US5 is in the same delivery, T021 can be folded into T025; if US4 ships alone, T021 stands.
- `plan_view.go` is edited by US2 (T008), US4 (T020/T021), and US5 (T024). Independent functions; sequence within the file.

### Within Each User Story

- Tests first (write, watch them fail), then implementation, then re-run to green.

### Parallel Opportunities

- All `[P]` test tasks within a story run in parallel (distinct test files/cases).
- T014 (CLI test) and T017 (CLI impl) are isolated to `internal/cli` and can proceed independently of the `internal/tui` stories.
- Across stories, US3 (picker) is fully independent of US4/US5 (rendering) and can be done in parallel by a second contributor.

---

## Parallel Example: User Story 1

```bash
# Write both reducer tests first (different cases, parallelizable):
Task: "T002 Reducer test: x cancels pomodoro on Planning (plan_update_test.go)"
Task: "T003 Reducer test: quit-confirm y/n on Planning (update_test.go)"
# Then implement:
Task: "T004 Add PomCancel + quit-confirm handling in handlePlanningKey (update.go)"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

The two P1 stories fix advertised-but-broken controls and are the highest-value, lowest-risk slice:

1. Phase 1 Setup (baseline green).
2. US1 (pomodoro cancel + quit-confirm) → validate.
3. US2 (Planning help) → validate.
4. **STOP and demo**: all controls advertised in the Planning status bar now work.

### Incremental Delivery

1. Setup → US1 → US2 (parity MVP).
2. US3 (sub-tasks in picker) → independent, ship.
3. US4 (visual consistency) → ship.
4. US5 (two-pane details) → ship (folds in US4's interim status fix).
5. Polish (Phase 8).

### Notes

- `[P]` = different files, no dependencies.
- Keep a `!styled` assertion for every rendering change to protect the non-ANSI fallback.
- The `GridOptions.Styled` change MUST keep `Styled:false` output byte-for-byte identical (Constitution Principle II; verified by T014).
- Commit after each task or logical group.
