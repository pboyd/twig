---
description: "Task list for Esc Cancels TUI Forms"
---

# Tasks: Esc Cancels TUI Forms

**Input**: Design documents from `/specs/059-esc-cancel-form/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/keyboard-interaction.md, quickstart.md

**Tests**: Test tasks ARE included — the spec defines explicit per-story Independent Tests and the TUI module has an established unit-test convention (`*_test.go` + `export_test.go`). Tests are written alongside each story.

**Organization**: Tasks are grouped by user story. All work is in the root CLI/TUI module under `internal/tui/`. No server/API/proto/db changes.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1, US2, US3 — maps to the user stories in spec.md

## Path Conventions

Single Go module at repo root; all paths below are repo-relative under `internal/tui/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm a green baseline before changes.

- [ ] T001 Verify baseline builds and tests pass: run `go build -o twig ./cmd/twig` and `go test ./internal/tui/...` from repo root; note current pass state.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The dirty-detection capability on the edit form that both US1 (clean → cancel) and US2 (dirty → confirm) depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T002 Add an opened-state snapshot to `editFormModel` in `internal/tui/edit.go`: capture `name`, `description`, `due`, `pomodoroEstimate`, `snooze`, and (when `showGoalField`) `goalIdx` at construction time in `newBlankForm`, `NewEditForm`, `NewRootForm`, and `NewSubtaskForm`.
- [ ] T003 Add method `isDirty() bool` to `editFormModel` in `internal/tui/edit.go`: return true if any current field value differs from its snapshot after `strings.TrimSpace`, or if `goalIdx != origGoalIdx` when `showGoalField`. Whitespace-only differences are NOT dirty.
- [ ] T004 [P] Expose `isDirty` (and snapshot accessors as needed) to tests via `internal/tui/export_test.go`.
- [ ] T005 [P] Add unit tests for the snapshot + `isDirty` in `internal/tui/edit_test.go`: blank form is clean; pre-filled edit form unchanged is clean; changing any field is dirty; reverting a changed field to its original value is clean again; whitespace-only change is clean; changing the goal selector is dirty.

**Checkpoint**: `editFormModel` can reliably report whether it has unsaved input.

---

## Phase 3: User Story 1 - Quickly dismiss a form I didn't change (Priority: P1) 🎯 MVP

**Goal**: Pressing `Esc` on a form with no unsaved input closes it immediately, saves nothing, and restores the prior cursor.

**Independent Test**: Open any covered form without changing a field, press `Esc`, confirm it closes with nothing saved and the previous selection preserved (contract C1).

### Implementation for User Story 1

- [ ] T006 [US1] In `editFormModel.Update` (`internal/tui/edit.go`), when the calendar is closed, add an `Esc` (`tea.KeyEscape`) case that, if `!isDirty()`, returns a command emitting `editCancelledMsg{originalCursor}`. Leave the dirty case to fall through unchanged for now (handled in US2). Do not alter the existing calendar-open `Esc` branch.
- [ ] T007 [P] [US1] Add unit tests in `internal/tui/edit_test.go`: `Esc` on a clean blank new-task form emits `editCancelledMsg`; `Esc` on a clean pre-filled edit form emits `editCancelledMsg`; `Esc` while the calendar is open closes the calendar and does NOT emit `editCancelledMsg` (layering regression, contract C6).
- [ ] T008 [P] [US1] Add a model-level test in `internal/tui/update_test.go` verifying that handling `editCancelledMsg` from a clean Tasks-tab edit returns to `modeList` with `cursor` restored to `originalCursor` and no create/update command issued.

**Checkpoint**: `Esc` cleanly backs out of an unchanged edit/new form (Tasks & Goals tabs, since both use `editFormModel`).

---

## Phase 4: User Story 2 - Protect work I've typed from accidental loss (Priority: P1)

**Goal**: Pressing `Esc` on a form with unsaved input shows a discard-confirmation prompt; confirming discards and closes, declining keeps the form with input intact.

**Independent Test**: Type into a form, press `Esc`, see the prompt; confirm → form closes discarding input; repeat and decline → input retained (contracts C2–C5, C7).

### Implementation for User Story 2

- [ ] T009 [US2] Add `confirmingDiscard bool` to `Model` in `internal/tui/model.go`.
- [ ] T010 [US2] Define `editDiscardRequestedMsg{originalCursor int}` in `internal/tui/edit.go`; in `editFormModel.Update` (calendar closed), change the `Esc` branch so a dirty form emits `editDiscardRequestedMsg` instead of falling through (clean still emits `editCancelledMsg`).
- [ ] T011 [US2] Handle `editDiscardRequestedMsg` in `Model.Update` (`internal/tui/update.go`): set `m.confirmingDiscard = true` (leave the form state untouched).
- [ ] T012 [US2] Intercept the discard prompt at the top of `handleKey` in `internal/tui/update.go` (before tab routing): when `m.confirmingDiscard`, `y` performs the cancel (for an edit form: emit `editCancelledMsg{originalCursor}`) and clears the flag; `n` or `esc` clears the flag only and returns to the form. Use a playful, measured prompt string per `contracts/keyboard-interaction.md`.
- [ ] T013 [US2] Render the discard-confirmation overlay in `internal/tui/view.go`, mirroring the existing `confirmingQuit` rendering and `[y]es [n]o` styling; show it over the active form when `m.confirmingDiscard` is true.
- [ ] T014 [P] [US2] Expose `confirmingDiscard` to tests via `internal/tui/export_test.go` if not already accessible.
- [ ] T015 [P] [US2] Add tests in `internal/tui/update_test.go`: dirty `Esc` on a Tasks edit sets `confirmingDiscard`; `y` cancels (returns to `modeList`, cursor restored, no RPC, input gone); `n` and `esc` each clear the flag and leave the form with input intact; verify the same flow for a Goals-tab edit (`goalEdit`/`goalNew`).
- [ ] T016 [P] [US2] Add tests in `internal/tui/edit_test.go`: dirty `Esc` emits `editDiscardRequestedMsg`; revert-to-original then `Esc` emits `editCancelledMsg` (no prompt); whitespace-only change then `Esc` emits `editCancelledMsg`.

**Checkpoint**: Edit/new forms (Tasks & Goals) guard unsaved input on `Esc`; US1 + US2 together deliver the full requested behavior for `editFormModel`.

---

## Phase 5: User Story 3 - Consistent cancel across every form (Priority: P2)

**Goal**: The same `Esc` rules apply to the Planning-tab forms, and nested-overlay layering is uniform across all covered forms.

**Independent Test**: Visit each form type; `Esc` cancels per US1/US2; for a form with the calendar open, the first `Esc` closes the calendar and the next applies cancel rules (contracts C1–C6 across all surfaces).

### Implementation for User Story 3

- [ ] T017 [US3] Add an opened-state snapshot (`origValues []string` of each `fields[i].Value()`) to `planFormState` in `internal/tui/model.go`, captured wherever the plan form is initialized (`initTaskTimeForm`, `initAddEventForm`, `initEditForm`) in `internal/tui/update.go` / `internal/tui/plan_update.go`.
- [ ] T018 [US3] Add a `planFormDirty(form planFormState) bool` helper in `internal/tui/plan_update.go` comparing trimmed current field values to `origValues`.
- [ ] T019 [US3] In `handlePlanFormKey` (`internal/tui/update.go`), add an `Esc`/`keys.Cancel` case: if not dirty, set `m.plan.mode = planList` (immediate cancel, clear `plan.err`); if dirty, set `m.confirmingDiscard = true`.
- [ ] T020 [US3] Extend the `handleKey` discard-confirm routing (from T012) so that when `m.activeTab == tabPlanning` and `m.plan.mode` is a form mode (`planTaskTime`/`planEventForm`/`planEdit`), `y` resets `m.plan.mode = planList` instead of emitting `editCancelledMsg`; `n`/`esc` unchanged.
- [ ] T021 [P] [US3] Expose plan-form dirty state to tests via `internal/tui/export_test.go` as needed.
- [ ] T022 [P] [US3] Add tests in `internal/tui/plan_update_test.go`: clean `Esc` on each plan form mode returns to `planList`; dirty `Esc` sets `confirmingDiscard`; `y` resets to `planList`; `n`/`esc` retains the form and its field values.
- [ ] T023 [P] [US3] Add a test in `internal/tui/edit_test.go` (or `update_test.go`) covering calendar-layering for a covered form: first `Esc` closes the calendar, a subsequent `Esc` applies clean/dirty cancel rules.

**Checkpoint**: All covered forms — Tasks edit/new/subtask, Goals new/edit/new-task, Planning timed-task/event/edit — share identical `Esc` behavior with consistent overlay layering.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Discoverability, tone, and full verification.

- [ ] T024 [P] Add an `Esc: cancel` hint to the form footers: edit form footer in `internal/tui/edit.go` (`View`, currently "Ctrl+S: save  Tab: next field") and the plan form footer in `internal/tui/plan_view.go`.
- [ ] T025 [P] Review the discard-prompt copy for tone (Constitution Principle IV) against `contracts/keyboard-interaction.md`; ensure it is accurate, actionable, and warm-but-measured.
- [ ] T026 Run full suite: `go test ./...` and `go build -o twig ./cmd/twig` from repo root; fix any regressions.
- [ ] T027 Walk through `specs/059-esc-cancel-form/quickstart.md` manual steps 1–8 against `./twig` and confirm each contract (C1–C7) holds.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup. BLOCKS US1, US2, US3 (all need `isDirty`).
- **US1 (Phase 3)**: Depends on Foundational.
- **US2 (Phase 4)**: Depends on Foundational; builds on the `Esc` branch introduced in US1 (T006 → T010 modify the same branch).
- **US3 (Phase 5)**: Depends on Foundational; T020 extends the discard routing added in US2 (T012). Plan-form snapshot/dirty (T017–T019) are otherwise independent of US1/US2.
- **Polish (Phase 6)**: Depends on all desired stories being complete.

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational. Delivers clean-cancel MVP.
- **US2 (P1)**: Shares the edit-form `Esc` branch with US1; implement after US1 (sequential on `edit.go`).
- **US3 (P2)**: Plan-form work is independent; the confirm-routing extension (T020) depends on US2's T012.

### Within Each User Story

- Models/snapshot before behavior; behavior before its tests where the test asserts behavior.
- Tests marked [P] touch separate test files or independent cases and can run in parallel.

### Parallel Opportunities

- T004 and T005 (foundational tests/shims) can run together.
- US1: T007 and T008 in parallel (different test files).
- US2: T014, T015, T016 in parallel after T009–T013 land.
- US3: T021, T022, T023 in parallel after T017–T020 land.
- Polish: T024 and T025 in parallel.

---

## Parallel Example: User Story 2

```bash
# After T009–T013 (model field, message, handler, key interception, overlay render):
Task: "T015 update_test.go — discard overlay confirm/decline for Tasks & Goals edits"
Task: "T016 edit_test.go — dirty/clean/revert/whitespace Esc emissions"
```

---

## Implementation Strategy

### MVP First (US1)

1. Phase 1 Setup → Phase 2 Foundational → Phase 3 US1.
2. **STOP and validate**: clean `Esc` backs out of unchanged edit/new forms with nothing saved.

### Incremental Delivery

1. Foundational ready (dirty detection).
2. US1 → clean-cancel works (MVP).
3. US2 → unsaved-input confirmation (completes the P1 requirement).
4. US3 → plan forms + uniform layering.
5. Polish → hints, tone, full verification.

---

## Notes

- Entirely `internal/tui`; no server/API/proto/db changes (per plan.md).
- Status-update compose (external `$EDITOR`), the date prompt, and selection-only pickers are intentionally out of scope — they keep their current immediate-`Esc` behavior (see `contracts/keyboard-interaction.md`).
- US1 and US2 both modify the `Esc` branch in `editFormModel.Update`; sequence them to avoid conflicts.
- Commit after each task or logical group; verify tests fail before fixing where applicable.
