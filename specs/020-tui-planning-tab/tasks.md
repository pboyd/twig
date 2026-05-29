---
description: "Task list for TUI Planning Tab implementation"
---

# Tasks: TUI Planning Tab

**Input**: Design documents from `/specs/020-tui-planning-tab/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/plan-rpcs.md

**Tests**: Included — this codebase ships unit tests for every TUI feature (`update_test.go`, `view_test.go`, etc.) and the plan's Testing section specifies test files. Test tasks are written before the implementation they cover.

**Organization**: Tasks are grouped by user story. All work is client-side under `services/todo/internal/tui/`, plus one parametrization in `services/todo/internal/cli/`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1–US4 maps to the user stories in spec.md

## Path Conventions

Single Go module at `services/todo/`. All commands run from `services/todo/` (e.g. `go test ./...`).

---

## Phase 1: Setup

**Purpose**: Confirm a clean starting point before changes.

- [X] T001 Establish a green baseline: from `services/todo/` run `go test ./...` and confirm it passes before making changes.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared types, the plan-service client, and the parametrized grid renderer that every user story builds on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 [P] Parametrize the grid renderer: add a `GridOptions{HideID bool, SelectedID int32}` parameter to `RenderGrid` in `services/todo/internal/cli/plan_grid.go` (drop the `[%d] ` label prefix when `HideID`; apply a highlight style to the lines of the entry whose id == `SelectedID` when styled), and update the CLI caller in `services/todo/internal/cli/plan.go` to pass the zero-value options so existing CLI output is byte-identical.
- [X] T003 [P] Extend `services/todo/internal/cli/plan_grid_test.go` with cases proving: default options reproduce current output, `HideID` removes the id prefix, and `SelectedID` highlights only that entry's rows under styled output.
- [X] T004 Build a `planv1connect.PlanServiceClient` in `services/todo/internal/tui/client.go` (same HTTP client, base URL, gzip, and bearer interceptor as the task client) and thread it through `services/todo/internal/tui/tui.go` into `newModel`.
- [X] T005 Add planning state types in `services/todo/internal/tui/model.go`: `tab` enum (`tabTasks`, `tabPlanning`), `planMode` enum (`planList`, `planPickTask`, `planTaskTime`, `planEventForm`, `planRename`, `planMove`, `planClear`), and the `planState`, `pickerState`, `planFormState` structs; add `activeTab`, `planClient`, and `plan` fields to `Model`, initializing `plan.day` to today in `newModel`.
- [X] T006 Add planning test shims to `services/todo/internal/tui/export_test.go` (constructor/accessors to set `activeTab`, seed `planState.entries`/`cursor`/`day`/`mode`, and read the selected entry) so reducer/view tests can drive planning state.

**Checkpoint**: Shared types, plan client, and parametrized renderer compile; `go test ./...` still green.

---

## Phase 3: User Story 1 - View the day's plan inside the TUI (Priority: P1) 🎯 MVP

**Goal**: A tab bar lets the user switch to a Planning tab that renders today's plan as the CLI-style grid (no `[id]`), with an auto-advancing now-marker and completion strikethrough, while the Tasks tab and the pomodoro status bar are preserved.

**Independent Test**: Launch the TUI, press Tab → the Planning grid renders for today with the now-marker; switch back → Tasks tab unchanged; a running pomodoro keeps ticking across the switch.

### Tests for User Story 1

- [X] T007 [P] [US1] In `services/todo/internal/tui/view_test.go`, test that the tab bar renders with the active tab highlighted, and (new `plan_view_test.go` if cleaner) that the Planning view renders the grid via `cli.RenderGrid` with `HideID` set, shows the day header with a today indicator, draws the now-marker only when the day is today, and strikes through completed-task entries.
- [X] T008 [P] [US1] In `services/todo/internal/tui/update_test.go`, test that Tab/Shift-Tab toggles `activeTab`, that switching to Planning issues a list command and back preserves the Tasks cursor/expansion, that a `planEntriesMsg` populates and clamps the selection, and that the pomodoro tick continues while the Planning tab is active.

### Implementation for User Story 1

- [X] T009 [US1] Add `NextTab`/`PrevTab` key bindings (Tab / Shift+Tab) to `services/todo/internal/tui/keymap.go`.
- [X] T010 [US1] Add `listPlanCmd(day)` + `planEntriesMsg{entries, err}` and its reducer handling in new file `services/todo/internal/tui/plan_update.go` (store entries in `start_minute` order, set `loaded`, clamp `cursor`, surface `err`).
- [X] T011 [US1] Add `planTickCmd()` + `planTickMsg` in `services/todo/internal/tui/plan_update.go`: a 1-second `tea.Tick` that re-renders and re-arms only while the Planning tab is active and `plan.day` == today (drives the now-marker, FR-008a).
- [X] T012 [US1] Handle tab switching in `services/todo/internal/tui/update.go`: Tab/Shift+Tab toggles `activeTab`; on activating Planning, fire `listPlanCmd(plan.day)` and start `planTickCmd`; route key/tick messages to the planning reducer when `activeTab == tabPlanning`.
- [X] T013 [US1] Render the tab bar and Planning content: add the one-line `Tasks │ Planning` bar (active highlighted) and the planning view (day header + grid via `cli.RenderGrid(..., GridOptions{HideID:true, SelectedID:selectedID})`) in `services/todo/internal/tui/view.go` and new `services/todo/internal/tui/plan_view.go`; pass `now` so the marker shows only for today.
- [X] T014 [US1] Subtract the tab-bar line from inner-height math in every view function in `services/todo/internal/tui/view.go` (alongside the existing status-bar height) so both tabs lay out correctly; add a thin `services/todo/internal/tui/plan_grid.go` helper mapping the selection cursor ↔ entry id for `SelectedID`.

**Checkpoint**: Planning tab is viewable, the grid matches the CLI, the marker advances unattended, and the pomodoro spans tabs. MVP complete.

---

## Phase 4: User Story 2 - Schedule tasks and block out events (Priority: P2)

**Goal**: From the Planning tab the user adds task entries (via a task picker + time prompt) and event entries (name + time prompt), seeing each appear immediately and highlighted.

**Independent Test**: On Planning, press `a` → pick a task → enter start/duration → entry appears highlighted; press `e` → name/start/duration → event appears; invalid time shows an error and creates nothing; Esc cancels.

### Tests for User Story 2

- [X] T015 [P] [US2] In `services/todo/internal/tui/plan_update_test.go`, test the add-task flow (pick task → time form → `AddPlanTask` called with parsed minutes; blank duration → `0`; new entry highlighted on reload), the add-event flow (`AddPlanEvent`), invalid-time → error with form still open, Esc → no change, and that the tab-switch keys are ignored while a picker/form is open (FR-023).
- [X] T016 [P] [US2] In `services/todo/internal/tui/plan_view_test.go`, test that the task picker renders the tree and that the task/event prompt forms render their fields and focus.

### Implementation for User Story 2

- [X] T017 [US2] Add `AddTask` (`a`) and `AddEvent` (`e`) bindings and reuse Save/Cancel/Tab field-nav bindings for planning forms in `services/todo/internal/tui/keymap.go`.
- [X] T018 [US2] Add `listTasksForPickerCmd()` + `planTasksMsg{tree, err}` in `services/todo/internal/tui/plan_update.go` (fetch via `ListTasks`, build with `cli.BuildTree`, keep incomplete tasks) and enter `planPickTask`.
- [X] T019 [US2] Implement the task picker (tree flatten/render/navigation, reusing the Tasks-tab row helpers) in `services/todo/internal/tui/plan_view.go` + `plan_update.go`; selecting a node captures its task id and advances to `planTaskTime`; an empty picker is cancellable.
- [X] T020 [US2] Implement the `planTaskTime` (start, optional duration) and `planEventForm` (name, start, optional duration) forms with Bubbles `textinput` in `services/todo/internal/tui/plan_view.go` (field nav, prefill, submit/cancel).
- [X] T021 [US2] Add `addPlanTaskCmd` + `addPlanEventCmd` + `planMutatedMsg{highlightID, err}` handling in `services/todo/internal/tui/plan_update.go`: validate inputs with `internal/cli/timeparse`, send duration `0` when blank, on success reload `plan.day` and select the returned entry id, on error surface a readable message and leave entries unchanged.
- [X] T022 [US2] Enforce the modal tab-switch guard in `services/todo/internal/tui/update.go`: while `plan.mode != planList` (or a Tasks-tab form is open) the tab-switch keys are consumed/ignored (FR-023).

**Checkpoint**: Users can populate today's plan with tasks and events from the TUI.

---

## Phase 5: User Story 3 - Edit and remove plan entries in place (Priority: P3)

**Goal**: Move a highlight between entries and rename, move, remove, or clear them, with the grid updating immediately and the selection settling sensibly.

**Independent Test**: With ≥2 entries, up/down moves the highlight; `r` renames; `m` moves; `d` removes (selection clamps); `c` clears from a time.

### Tests for User Story 3

- [X] T023 [P] [US3] In `services/todo/internal/tui/plan_update_test.go`, test that up/down (and j/k) move `cursor` (updating `SelectedID`), that rename/move/remove/clear call the right RPCs with parsed values (move with blank duration sends `0` to keep existing), that selection follows the edited entry and clamps after delete, and that entry actions are no-ops on an empty day.

### Implementation for User Story 3

- [X] T024 [US3] Add selection navigation (up/`k`, down/`j`) for `planList` in `services/todo/internal/tui/plan_update.go`, moving `cursor` over entries in chronological order with clamping.
- [X] T025 [US3] Add `Rename` (`r`), `Move` (`m`), `Remove` (`ctrl+d`, matching the Tasks-tab delete), and `Clear` (`c`) bindings in `services/todo/internal/tui/keymap.go`.
- [X] T026 [US3] Implement the `planRename` (prefilled name), `planMove` (start + optional duration), and `planClear` (start prefilled with now) forms in `services/todo/internal/tui/plan_view.go`.
- [X] T027 [US3] Add `renamePlanCmd`, `movePlanCmd`, `removePlanCmd`, and `clearPlanCmd` + `planMutatedMsg` handling in `services/todo/internal/tui/plan_update.go`: validate with `timeparse`, reload after success, keep selection on the edited entry (or clamp after remove/clear), surface errors (FR-021).

**Checkpoint**: Full in-place editing parity with the CLI day-planner (except the intentionally-absent complete toggle).

---

## Phase 6: User Story 4 - Plan a different day (Priority: P3)

**Goal**: Navigate to any day (prev/next/today), edit it with all the same actions, and refresh on demand; no now-marker off today.

**Independent Test**: `]` advances to tomorrow with its own entries; add lands on that day; `t` returns to today; `[` reaches past days; no marker shown off today; `Ctrl+R` refreshes.

### Tests for User Story 4

- [X] T028 [P] [US4] In `services/todo/internal/tui/plan_update_test.go`, test that `[`/`]` change `plan.day` by one day and reload, that `t` resets to today, that the loaded entries are per-day, that `Ctrl+R` reloads the current day, and (via `plan_view_test.go`) that no now-marker renders when the day is not today.

### Implementation for User Story 4

- [X] T029 [US4] Add `PrevDay` (`[`), `NextDay` (`]`), `Today` (`t`), and `Refresh` (`Ctrl+R`) bindings for the Planning tab in `services/todo/internal/tui/keymap.go`.
- [X] T030 [US4] Implement day navigation and manual refresh in `services/todo/internal/tui/plan_update.go`: adjust `plan.day` ±1 day or to today and reload via `listPlanCmd`; `Ctrl+R` reloads the current day.
- [X] T031 [US4] Ensure the day header in `services/todo/internal/tui/plan_view.go` shows the in-view date with a today/not-today indicator and that the now-marker is suppressed when `plan.day != today` (and the planning tick does not re-arm off today).

**Checkpoint**: All four user stories independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T032 [P] Make help context-aware per active tab: update `ShortHelp`/`FullHelp` in `services/todo/internal/tui/keymap.go` and the help pane in `services/todo/internal/tui/help.go` to list the planning keys when the Planning tab is active.
- [X] T033 Run `gofmt`/`go vet ./...` and the full suite `go test ./...` from `services/todo/`; fix any failures.
- [X] T034 Walk through `specs/020-tui-planning-tab/quickstart.md` against a running server to confirm all scenarios (including the auto-advancing marker and cross-CLI consistency).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none.
- **Foundational (Phase 2)**: depends on Setup; **blocks all user stories**.
- **User Story 1 (Phase 3)**: depends on Foundational. This is the MVP and establishes the tab + grid + entry-loading that the other stories operate within.
- **User Stories 2–4 (Phases 4–6)**: depend on Foundational and on US1's tab/grid/load being in place; once US1 is done they are independently testable and may proceed in any order (they touch overlapping files — `plan_update.go`, `plan_view.go`, `keymap.go` — so coordinate if parallelized).
- **Polish (Phase 7)**: depends on the desired user stories being complete.

### Within Each User Story

- Write the listed test task(s) first and watch them fail, then implement.
- In `plan_update.go`/`plan_view.go`, commands/reducer before the keys that trigger them.
- Story complete and independently testable before moving to the next priority.

### Parallel Opportunities

- **Foundational**: T002 and T003 (cli package) run in parallel with each other only after the signature exists; both are parallel to nothing in `internal/tui`. T004 and T005 both touch `model.go`/`newModel`, so run them sequentially.
- **Within a story**: the test tasks marked [P] (different test files) can be written in parallel. Most implementation tasks within a story touch the same new files (`plan_update.go`, `plan_view.go`) and are therefore sequential.
- **Across stories**: US2/US3/US4 edit overlapping files; prefer sequential delivery in priority order unless coordinating carefully.

---

## Parallel Example: Foundational Phase

```bash
# T002/T003 operate in the cli package; T005's type work is in the tui package.
Task: "T002 Parametrize RenderGrid with GridOptions in internal/cli/plan_grid.go"
Task: "T005 Add planning state types in internal/tui/model.go"
# (T003 follows T002 in the same file; T004 precedes/follows T005 in model.go — keep sequential.)
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1: Setup → 2. Phase 2: Foundational → 3. Phase 3: US1.
4. **STOP and VALIDATE**: Planning tab renders today's plan, marker advances, pomodoro spans tabs, Tasks tab preserved.
5. Demo the MVP.

### Incremental Delivery

1. Foundation + US1 → viewable planning tab (MVP).
2. + US2 → can build the day (tasks + events).
3. + US3 → in-place editing (rename/move/remove/clear).
4. + US4 → plan any day.
5. Polish (help, vet/test, quickstart).

Each story adds value without breaking the previous ones.

---

## Notes

- [P] = different files, no dependency on an incomplete task.
- No proto/handler/db/migration work: the feature reuses the existing `PlanService` RPCs (see `contracts/plan-rpcs.md`).
- Keep CLI grid output unchanged when parametrizing `RenderGrid` (T002).
- Commit after each task or logical group; stop at any checkpoint to validate a story independently.
