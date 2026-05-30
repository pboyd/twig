---

description: "Task list for Planning Tab Refinements"
---

# Tasks: Planning Tab Refinements

**Input**: Design documents from `/specs/022-planning-tab-refinements/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Test tasks ARE included. The project's TUI is covered by string-view and reducer tests (`internal/tui/*_test.go`, `internal/cli/plan_grid_test.go`). Each behavior/render change gets a test, including a `!styled` assertion to protect the non-ANSI fallback, and the grid change keeps `SelectedID==0` output byte-for-byte identical.

**Organization**: Tasks are grouped by user story (from spec.md) in priority order. All work is in the single Go module at `services/todo/`; run tests with `cd services/todo && go test ./...`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1–US6 maps to the spec's user stories
- Exact file paths are given relative to `services/todo/`

---

## Phase 1: Setup

**Purpose**: Establish a green baseline before changing behavior.

- [X] T001 Confirm the starting point is green: run `cd services/todo && go build ./... && go test ./...` and note the passing baseline (especially `internal/cli` grid tests and existing `internal/tui` tests).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Cross-cutting prerequisites for all stories.

**Note**: This feature has **no blocking foundational work** — it edits existing `internal/tui` files and makes one additive change to `internal/cli/plan_grid.go`, all scoped within their stories. User stories may begin immediately after Setup. (Phase intentionally minimal per the plan's Constitution Check, Principle I.)

> **Shared-file coordination** (read before parallelizing): `keymap.go` is edited by US1, US2, US5, US6; `update.go` by US1 and US6; `model.go`, `plan_update.go`, `plan_view.go` by US1 and US6. These edits are independent in intent but touch the same files and the same `ShortHelp`/`FullHelp` groups, so do **not** mark them `[P]` across stories. Apply keymap edits in priority order (US1 → US2 → US5 → US6) so the help groups reach a consistent final state.

**Checkpoint**: Baseline green → user stories can begin.

---

## Phase 3: User Story 1 - Edit a scheduled entry with one Enter-driven form (Priority: P1) 🎯 MVP

**Goal**: A single `planEdit` form (Name + Start + Duration) opens with `Enter` on the selected entry and, on save, composes the existing `RenamePlanEntry` + `MovePlanEntry` RPCs; the separate `rename` (`r`) and `move` (`m`) prompts are gone.

**Independent Test**: Select an entry, press `Enter` → a form with Name (prefilled) + Start + Duration appears; change name and time, save → grid reflects both; save with no changes → no-op; `Enter` on an empty grid → nothing. (Rendering location is finalized in US3; US1 is testable with the form rendered by the existing form renderer.)

### Tests for User Story 1 ⚠️

- [X] T002 [P] [US1] Reducer test: `Enter` on Planning with an entry selected enters `planEdit` with fields `[Name(prefilled), Start, Duration]` and `entryID` set; `Enter` with no entries is inert — in `services/todo/internal/tui/plan_update_test.go`.
- [X] T003 [P] [US1] Reducer test: `submitEditForm` composes correctly — a changed Name issues `RenamePlanEntry`; a provided Start/Duration issues `MovePlanEntry`; both-unchanged is a no-op that closes the form and issues no RPC; empty Name sets `plan.err` and keeps the form open — in `services/todo/internal/tui/plan_update_test.go`.

### Implementation for User Story 1

- [X] T004 [US1] In `services/todo/internal/tui/model.go`: replace the `planRename` and `planMove` `planMode` values with a single `planEdit` value (update the enum + comments).
- [X] T005 [US1] In `services/todo/internal/tui/keymap.go`: add a `PlanEdit` binding (`enter`, help "enter / edit entry"); remove the `PlanRename` and `PlanMove` bindings; update the Planning `ShortHelp()`/`FullHelp()` groups to drop rename/move and include `PlanEdit`. (Shared file — see coordination note; sequence with US2/US5/US6.)
- [X] T006 [US1] In `services/todo/internal/tui/plan_update.go`: add `initEditForm()` (prefill Name from the selected entry; Start/Duration blank with placeholders, blank = keep; set `entryID`); add `submitEditForm()` that runs a single command calling `renamePlanCmd` only when the Name changed and `movePlanCmd` only when a Start/Duration was entered, then reloads with the entry highlighted; delete `initRenameForm`/`initMoveForm` and `submitRenameForm`/`submitMoveForm`.
- [X] T007 [US1] In `services/todo/internal/tui/update.go`: in `handlePlanningKey` add an `Enter`→`initEditForm()` case guarded on a non-empty selection and remove the `PlanRename`/`PlanMove` cases; in `handlePlanModalKey` add `planEdit` to the form-mode routing (and drop `planRename`/`planMove`).
- [X] T008 [US1] In `services/todo/internal/tui/plan_view.go`: add the `planEdit` case to `planFieldLabel` (`[Name, Start, Duration]`) and to `renderPlanFormView` (title "Edit entry"); remove the `planRename`/`planMove` cases.

**Checkpoint**: `Enter` opens the merged Edit form; save composes rename+move; rename/move keys gone; `go test ./...` green.

---

## Phase 4: User Story 2 - Edit a task with Enter on the Tasks tab (Priority: P1)

**Goal**: On the Tasks tab, `Enter` opens the task edit form and `e` no longer does.

**Independent Test**: On Tasks, select a task, press `Enter` → edit form opens; press `e` → nothing; help advertises `enter` for edit.

### Tests for User Story 2 ⚠️

- [X] T009 [P] [US2] Reducer test: on the Tasks list, `Enter` enters `modeEdit` with the edit form for the selected task; `e` does not enter `modeEdit`; `Enter` on an empty list is inert — in `services/todo/internal/tui/update_test.go`.
- [X] T010 [P] [US2] View/help test: the Tasks-tab help/short-help advertises `enter` (not `e`) for "edit task" — in `services/todo/internal/tui/view_test.go`.

### Implementation for User Story 2

- [X] T011 [US2] In `services/todo/internal/tui/keymap.go`: change the `Edit` binding keys from `e` to `enter` and its help to `key.WithHelp("enter", "edit task")`. (Shared file — sequence after US1's keymap edit.) No `handleListKey` logic change is needed (it already matches `m.keys.Edit`).

**Checkpoint**: `Enter` edits on Tasks; `e` is inert for edit; help updated; tests green.

---

## Phase 5: User Story 3 - Add/edit entries without a full-screen takeover (Priority: P2)

**Goal**: The Planning picker and all add/edit forms render in the right pane beside the grid, mirroring the Tasks tab; the full-width modal path is removed.

**Independent Test**: Trigger add-task (`t`), add-event (`e`), and edit (`Enter`) → each renders in the right pane with the grid still visible on the left; narrow terminals degrade gracefully; on save/cancel the right pane returns to the Details view.

### Tests for User Story 3 ⚠️

- [X] T012 [P] [US3] View test: with `plan.mode` set to `planPickTask`, `planTaskTime`, `planEventForm`, and `planEdit`, `viewPlanning` renders a left grid pane + right picker/form pane (assert the grid lines are still present alongside the form), in both styled and `!styled` modes — in `services/todo/internal/tui/plan_view_test.go`.

### Implementation for User Story 3

- [X] T013 [US3] In `services/todo/internal/tui/view.go` `viewPlanning`: replace the full-width modal branch (`m.plan.mode != planList`) with a two-pane layout — grid left (unfocused), active picker/form right (focused/accent border) — using `paneBox` + `lipgloss.JoinHorizontal` (styled) and the `splitLines`/`padRightAnsi` row-join (non-styled), reusing `viewWithForm`'s height arithmetic so the status line stays bottom-pinned.
- [X] T014 [US3] In `services/todo/internal/tui/plan_view.go`: size `renderPlanFormView` and `renderPlanPickerView` to the passed right-pane width (wrap/trim long task names and field rows to fit), so the picker/forms render correctly in the narrower pane.

**Checkpoint**: Picker and all forms render in the right pane; grid stays visible; degrades on narrow terminals; tests green.

---

## Phase 6: User Story 4 - Planning selection highlight matches the Tasks tab (Priority: P2)

**Goal**: The selected entry's text cell uses the Tasks tab's `highlightStyle` (bold + blue background + white foreground); the hour gutter, rails, and box border are not restyled.

**Independent Test**: Move the selection on the grid → the selected entry's text block is bold white on blue, identical to a selected Tasks row; the hour marker (`07:00`) and the entry's border are unchanged. Compare directly against the Tasks tab.

### Tests for User Story 4 ⚠️

- [X] T015 [P] [US4] CLI test: with a `SelectionStyle` styler set and `SelectedID` matching, a selected single-row entry styles only the content cell (the `┣`/`┫` and the gutter are not wrapped); a selected multi-row entry styles only the interior content row(s), not the `┏━┓`/`┗━┛`/`┣━┫` border rows; `SelectedID==0` output is byte-for-byte unchanged (golden); row widths are identical with and without selection — in `services/todo/internal/cli/plan_grid_test.go`.

### Implementation for User Story 4

- [X] T016 [US4] In `services/todo/internal/cli/plan_grid.go`: add the additive `SelectionStyle func(string) string` field to `GridOptions` and rework `applySelection`/`accentOpen` and the top/bottom/shared-border cases so the styler is applied to the entry's content cell only — never the gutter, rails, or box-drawing border — keeping `SelectionStyle==nil` and `SelectedID==0` behavior identical to today (per `contracts/grid-selection.md`).
- [X] T017 [US4] In `services/todo/internal/tui/plan_grid.go` `planGridOptions()`: set `SelectionStyle: highlightStyle.Render` so the TUI selection reuses the exact Tasks-tab highlight; leave it unset for the plain `todo plan` CLI.

**Checkpoint**: Selected cell matches the Tasks highlight; gutter/border untouched; CLI output unchanged; tests green.

---

## Phase 7: User Story 5 - Add a task with `t`, jump-to-today on `.` (Priority: P3)

**Goal**: `t` adds (schedules) a task on the Planning tab; the jump-to-today action moves from `t` to `.`.

**Independent Test**: On Planning, `t` opens the add-task picker; `.` jumps the grid to today; `t` does not jump to today; help advertises `t add task` and `. today`.

### Tests for User Story 5 ⚠️

- [X] T018 [P] [US5] Reducer test: `t` on Planning opens the add-task picker (enters `planPickTask` / issues the task-list command); `.` sets the day to today and reloads; pressing `t` does not change the day — in `services/todo/internal/tui/plan_update_test.go`.
- [X] T019 [P] [US5] View/help test: the Planning help/short-help advertises `t` for "add task" and `.` for "today" — in `services/todo/internal/tui/view_test.go`.

### Implementation for User Story 5

- [X] T020 [US5] In `services/todo/internal/tui/keymap.go`: change `PlanAddTask` keys `a`→`t` (help "t / add task") and `PlanToday` keys `t`→`.` (help ". / today"); update the Planning `ShortHelp()`/`FullHelp()` strings accordingly. No `handlePlanningKey` change needed (it matches via `m.keys.PlanAddTask`/`m.keys.PlanToday`). (Shared file — sequence after US1/US2.)

**Checkpoint**: `t` adds a task, `.` jumps to today, no key collisions; tests green.

---

## Phase 8: User Story 6 - Remove the clear action from the TUI (Priority: P3)

**Goal**: The `clear` action is gone from the Planning tab — no binding, no mode, no form, no help/status mention. Per-entry delete still works.

**Independent Test**: On Planning, `c` is inert (no clear form); help/status do not mention clear; `ctrl+d` still removes a single entry.

### Tests for User Story 6 ⚠️

- [X] T021 [P] [US6] Reducer/view test: `c` on Planning does not enter a clear mode (is inert), and the Planning help/`FullHelp` does not list a clear action — in `services/todo/internal/tui/plan_update_test.go` and `services/todo/internal/tui/view_test.go`.

### Implementation for User Story 6

- [X] T022 [US6] In `services/todo/internal/tui/keymap.go`: remove the `PlanClear` binding and drop it from the Planning `FullHelp()` group. (Shared file — sequence after US1/US2/US5.)
- [X] T023 [US6] In `services/todo/internal/tui/model.go`: remove the `planClear` `planMode` value.
- [X] T024 [US6] In `services/todo/internal/tui/update.go`: remove the `PlanClear` case from `handlePlanningKey` and remove `planClear` from the `handlePlanModalKey` form-mode list.
- [X] T025 [US6] In `services/todo/internal/tui/plan_update.go`: delete `initClearForm`, `submitClearForm`, the `planClear` branch of `submitPlanForm`, and the now-unused `clearPlanCmd` (the `todo plan clear` CLI in `internal/cli/plan.go` is unaffected).
- [X] T026 [US6] In `services/todo/internal/tui/plan_view.go`: remove the `planClear` cases from `renderPlanFormView` and `planFieldLabel`.

**Checkpoint**: Clear is absent from the TUI; per-entry delete intact; tests green.

---

## Phase 9: Polish & Cross-Cutting Concerns

- [X] T027 Run `cd services/todo && gofmt -l . && go vet ./... && go test ./...`; fix any formatting/vet/test issues.
- [ ] T028 [P] Execute the `quickstart.md` manual walkthrough end-to-end (all 7 sections) against `make dev` + a built `todo` binary; confirm the right-pane forms, cell-only highlight, key rebinds, and the non-ANSI fallback.
- [X] T029 [P] Verify FR-011 (no two advertised Planning/Tasks actions share a key after the rebinds) and SC-004 (every advertised key maps to a working action) by auditing the final `ShortHelp`/`FullHelp` groups against the handlers.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none — start immediately.
- **Foundational (Phase 2)**: none blocking.
- **User Stories (Phases 3–8)**: each depends only on Setup. Deliver in priority order (US1 → US2 → US3 → US4 → US5 → US6) or in parallel by file ownership, respecting the shared-file coordination note.
- **Polish (Phase 9)**: after the desired stories are complete.

### User Story Dependencies & soft ordering

- **US1 (P1)**: independent. Introduces `planEdit` + `Enter`-edit and removes rename/move. The Edit form initially renders via the existing form renderer; US3 later moves it (and all forms) to the right pane.
- **US2 (P1)**: independent (keymap-only).
- **US3 (P2)**: independent of US1's logic but renders whatever form modes exist; landing after US1 means its right-pane test also covers `planEdit`. Best sequenced after US1 so the picker/forms (incl. edit) all move together.
- **US4 (P2)**: fully isolated to `internal/cli` + `planGridOptions` — parallelizable with any other story.
- **US5 (P3)** and **US6 (P3)**: keymap/handler edits; sequence their `keymap.go`/help edits after US1/US2 so the help groups end consistent.

### Cross-story file notes (avoid same-file conflicts)

- `keymap.go`: US1 (T005), US2 (T011), US5 (T020), US6 (T022) — apply in that order; the `ShortHelp`/`FullHelp` planning group is only consistent once all four land. Not `[P]` together.
- `update.go`: US1 (T007) and US6 (T024) edit different handlers/cases of the same file — sequence, not `[P]`.
- `model.go`: US1 (T004) and US6 (T023) — sequence.
- `plan_update.go`: US1 (T006) and US6 (T025) — sequence.
- `plan_view.go`: US1 (T008), US3 (T014), US6 (T026) — independent functions; sequence within the file.
- `view.go` `viewPlanning`: rewritten once by US3 (T013).

### Within Each User Story

- Tests first (write, watch them fail), then implementation, then re-run to green.

### Parallel Opportunities

- All `[P]` test tasks within a story run in parallel (distinct test files/cases).
- **US4 (T015–T017)** is isolated to `internal/cli` + `plan_grid.go` and can proceed fully in parallel with the `internal/tui` interaction stories.
- Across stories, the keymap/handler-sharing stories (US1/US2/US5/US6) should be sequenced; US3 (rendering) and US4 (grid styling) can be parallelized against them by a second contributor.

---

## Parallel Example: User Story 1

```bash
# Write both reducer tests first (different cases, parallelizable):
Task: "T002 Reducer test: Enter opens the merged planEdit form (plan_update_test.go)"
Task: "T003 Reducer test: submitEditForm composes rename+move (plan_update_test.go)"
# Then implement in dependency order:
Task: "T004 Replace planRename/planMove with planEdit (model.go)"
Task: "T005 PlanEdit binding + help groups (keymap.go)"
Task: "T006 initEditForm/submitEditForm; delete rename/move forms (plan_update.go)"
Task: "T007 Enter→initEditForm; route planEdit (update.go)"
Task: "T008 planEdit field labels + form title (plan_view.go)"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

The two P1 stories are the headline interaction change and the lowest-risk slice:

1. Phase 1 Setup (baseline green).
2. US1 (merged Enter-driven Edit form) → validate.
3. US2 (Enter edits on Tasks) → validate.
4. **STOP and demo**: one consistent Enter-to-edit gesture across both tabs.

### Incremental Delivery

1. Setup → US1 → US2 (Enter-edit MVP).
2. US3 (right-pane forms) → ship.
3. US4 (cell-only highlight) → ship (parallelizable throughout).
4. US5 (`t` add-task, `.` today) → ship.
5. US6 (remove clear) → ship.
6. Polish (Phase 9).

### Notes

- `[P]` = different files, no dependencies.
- Keep a `!styled` assertion for every rendering change to protect the non-ANSI fallback.
- The `GridOptions.SelectionStyle` change MUST keep `SelectedID==0` output byte-for-byte identical (Constitution Principle II; verified by T015).
- Apply the shared-file (`keymap.go`/`update.go`/`model.go`) edits in priority order; commit after each task or logical group.
