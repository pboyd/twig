---

description: "Task list for 066-add-task-to-plan"
---

# Tasks: Add New Tasks to a Plan

**Input**: Design documents from `/specs/066-add-task-to-plan/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: INCLUDED. `contracts/ui-contract.md` §5 enumerates CT-01..CT-14 as tests that "must exist
and pass", and Constitution Quality Gate 1 requires the committed contracts to match the
implementation. Every CT is claimed by exactly one task below.

**Organization**: Grouped by user story so each is independently implementable and testable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on incomplete work)
- **[Story]**: US1, US2, US3 — maps to the user stories in spec.md
- Every task names its exact file path

## Path Conventions

Two client surfaces, per plan.md § Project Structure:

- **TUI** (root Go module): `internal/tui/`
- **Web** (React SPA): `services/twig-web/src/`

**No server work.** `api/proto/**`, `api/gen/**`, and `services/twig/**` are not touched by any task
here. If a task seems to need a proto or handler change, stop — that contradicts `research.md` R1
and needs a design conversation, not a workaround.

**Surface tags**: `[TUI]` / `[WEB]` in descriptions mark which surface a task lands on. TUI and web
tasks never share a file, so they are almost always parallelizable with each other.

---

## Phase 1: Setup

**Purpose**: Establish a known-good baseline. This feature adds no dependency, no directory, and no
tooling, so there is genuinely nothing to initialize — don't invent work here.

- [X] T001 Verify the baseline is green before touching anything: `go test ./...` from the repo root and `npm test` in `services/twig-web/`. Record any pre-existing failure so it is not later mistaken for a regression from this feature.
- [X] T002 [P] Bring up the stack per `quickstart.md` §1 (`make dev`, provision a user, export `TWIG_API_KEY`/`TWIG_ADDR`) so the manual verification steps in later phases are actually runnable.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The form plumbing both US1 and US2 build on — focus wiring and state on the TUI, the
component interface on the web. No user-visible behavior lands in this phase.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T003 [TUI] Add `focusPlan = 6` to the focus constant block in `internal/tui/edit.go`, renumbering `focusGoal`, `focusSave`, `focusCancel`, and `focusCount` accordingly. Placement is between Snooze and Goal, per `contracts/ui-contract.md` §2. Then grep `internal/tui/` for bare numeric literals compared against these constants — the plan flags this renumbering as the top risk.
- [X] T004 [TUI] Add the plan choice constants (`planChoiceNone`/`planChoiceToday`/`planChoiceTomorrow`/`planChoiceDate`) and the `planIdx`, `planDate`, `showPlanField`, `origPlanIdx` fields to `editFormModel` in `internal/tui/edit.go`, per `data-model.md` § TUI. Depends on T003.
- [X] T005 [TUI] Set `showPlanField` in the form constructors in `internal/tui/edit.go`: `true` in `newBlankForm`/`NewChildForm` (creating a task), `false` in `NewEditForm` and every goal form. This is the FR-005/FR-010/FR-011 gate — the control must exist *only* when creating a task. Depends on T004.
- [X] T006 [P] [TUI] Add `export_test.go` shims exposing the plan form state (`planIdx`, `planDate`, `showPlanField`) for tests in `internal/tui/export_test.go`, following the existing shim conventions in that file.
- [X] T007 [P] [WEB] Add the `showPlanControl?: boolean` prop (default `false`) and widen `onSubmit` to `(name: string, description: string, planDay?: string) => Promise<void>` in `services/twig-web/src/components/TaskForm.tsx`. Add no UI yet. Verify `pages/TaskDetailPage.tsx` still typechecks untouched — its 2-arg handler must satisfy the widened type (plan.md § Risks).
- [X] T008 [P] [WEB] Add the partial-failure message to `services/twig-web/src/theme/messages.ts` per the copy contract in `contracts/ui-contract.md` §4: it must state both halves (task saved, plan missed) and name the recovery path. Suggested: `"Task saved, but it didn't make it onto the plan. Want to add it from the list?"`

**Checkpoint**: Form plumbing exists and is inert. Both suites still green. User stories can begin.

---

## Phase 3: User Story 1 - Plan a new task for today as you create it (Priority: P1) 🎯 MVP

**Goal**: Choosing "Today" on the create form puts the new task on today's plan in the same save.

**Independent Test**: Create a task with **Today** selected; confirm it appears in the task tree
*and* as an untimed entry on today's plan. Then create one without touching the control and confirm
it lands on no plan at all.

**Scope note**: the left/right cycle implemented here mechanically includes **Tomorrow**, because
`None → Today → Tomorrow → None` is one list — splitting it across stories would be artificial.
US2's real work is the *arbitrary-date* path. Tomorrow's verification lives in US2.

### Tests for User Story 1 ⚠️

> Write these first and watch them fail. They encode the contract, not the implementation.

- [X] T009 [P] [US1] [TUI] Form-state tests in `internal/tui/edit_test.go`: **CT-01** (a new create form has `planIdx == planChoiceNone`), **CT-02** (cycling right from none yields Today, Tomorrow, then wraps to none), **CT-03** (save with Today ⇒ `editSavedMsg.planDay` is today's ISO day), **CT-10** (touching the control makes `isDirty()` true), **CT-13** (changing the plan choice leaves due/snooze/estimate/goal untouched in `editSavedMsg`). Table-driven, per the existing conventions in this file.
- [X] T010 [P] [US1] [TUI] Save-path tests in a new `internal/tui/plan_from_create_test.go` using fake clients: **CT-04** (`AddPlanTask` is called with `start_minute` absent and `duration_minute: 0`), **CT-05** (untouched control ⇒ `AddPlanTask` never called), **CT-08** (`CreateTask` fails ⇒ `AddPlanTask` never called), **CT-09** (`AddPlanTask` fails ⇒ the task stays created and the partial-failure message is reported).
- [X] T011 [P] [US1] [WEB] Component tests in `services/twig-web/src/components/TaskForm.test.tsx`: **CT-01** (a fresh form renders "No plan" selected), **CT-02** (all four choices render, in contract order), **CT-03** (submitting with Today ⇒ `onSubmit`'s third arg is today's ISO day), **CT-05** (submitting untouched ⇒ third arg `undefined`), **CT-14** (the control is keyboard-operable and the segments expose radio semantics).
- [X] T012 [P] [US1] [WEB] Page tests in `services/twig-web/src/pages/TaskTreePage.test.tsx`, mocking mutations as `AddToPlanControl.test.tsx` already does: **CT-08** (create fails ⇒ no plan call), **CT-09** (plan call fails ⇒ task stays created, partial-failure toast, form input not discarded), **CT-11** (success invalidates both `listTasks` and that day's `listPlanEntries`).

### Implementation for User Story 1 — TUI

- [X] T013 [US1] [TUI] Handle `←`/`→` on `focusPlan` in `internal/tui/edit.go`, cycling `None → Today → Tomorrow → None`. Copy the shape of the existing `cycleGoal` modular arithmetic; `planChoiceDate` is a destination, not a cycle stop, and cycling away from it clears `planDate` (`data-model.md` § State transitions).
- [X] T014 [US1] [TUI] Render the Plan field in `internal/tui/edit.go` between Snooze and Goal, using the existing `fieldLabel` helper — no ad-hoc styling (Principle III). Render `‹ No plan ›` / `‹ Today ›` / `‹ Tomorrow ›` / `‹ 2026-07-24 ›` per `contracts/ui-contract.md` §2, with the focused-field hint line `←/→ cycle · ctrl+g: summon the calendar`. Depends on T013.
- [X] T015 [US1] [TUI] Include the plan choice in `isDirty()` in `internal/tui/edit.go` by comparing `planIdx`/`planDate` against `origPlanIdx`, so the discard-confirmation keeps working (UC-06). Depends on T004.
- [X] T016 [US1] [TUI] Add `planDay string` to `editSavedMsg` and resolve `planIdx` → an absolute ISO day in `buildSaveMsg`, in `internal/tui/edit.go`. Use the form's existing `nowFunc` hook, not `time.Now()` directly — this is what makes FR-005 and CT-06 testable. Empty string means no plan. Depends on T013.
- [X] T017 [US1] [TUI] Add a `planv1connect.PlanServiceClient` parameter to `createTaskCmd` in `internal/tui/update.go` and chain `AddPlanTask` after `CreateTask` when `msg.planDay != ""`, per the call sequence in `contracts/rpc-usage.md`. Order is `CreateTask` → `SetTaskGoal` → `AddPlanTask`, so a plan failure cannot cost the user the task or the goal. Pass `start_minute` absent and `duration_minute: 0`. Depends on T016.
- [X] T018 [US1] [TUI] Handle the plan outcome in `internal/tui/update.go`: on success keep the existing `fetchAfterMutation` refresh; on `FAILED_PRECONDITION` report the already-on-plan case as a soft notice, not a hard error; on any other error report the partial-failure message while keeping the created task (FR-007). Update the `createTaskCmd` call site in `handleEditSaved` to pass `m.planClient` (already on `Model`). Depends on T017.

### Implementation for User Story 1 — Web

- [X] T019 [P] [US1] [WEB] Build the segmented control in `services/twig-web/src/components/TaskForm.tsx`, shown only when `showPlanControl` is true: a `radiogroup` labelled "Add to plan" with **No plan** / **Today** / **Tomorrow** segments (the 📅 affordance is US2). Use `todayIso()`/`tomorrowIso()` from `lib/planDays.ts` and existing design tokens / the `Button` component — no one-off styles (Principle III). Exactly one segment carries `aria-checked`; selection is conveyed by more than color; 44px min touch targets; light and dark themes. Resolve the choice into `onSubmit`'s `planDay` arg. Depends on T007.
- [X] T020 [US1] [WEB] Update `handleAddTask` in `services/twig-web/src/pages/TaskTreePage.tsx` to accept `planDay`, call `addPlanTask` after `createTask` succeeds, and invalidate both `listTasks` and that day's `listPlanEntries` (copy the key shape, including `cardinality: "finite"`, from `AddToPlanControl.tsx:30`). Show `messages.addedToToday`/`addedToDay` on success, `messages.alreadyOnPlan` on `FAILED_PRECONDITION`, and the new partial-failure message otherwise — via the existing `ToastProvider`. Pass `showPlanControl` to its `TaskForm`. Depends on T019, T008.
- [X] T021 [US1] [WEB] Apply the same treatment to `handleAddSubTask` and the `TaskForm` at `services/twig-web/src/components/TreeRow.tsx:194`, so sub-tasks are plannable identically (spec edge case: the parent's plan state is irrelevant to the child's). Depends on T019, T008.

**Checkpoint**: US1 is fully functional on both surfaces. This is the MVP — it delivers the entire
"save the user a step" value on its own. Stop here and validate before starting US2.

---

## Phase 4: User Story 2 - Plan a new task for tomorrow or another day (Priority: P2)

**Goal**: The same control reaches Tomorrow directly and any other date without leaving the form.

**Independent Test**: Create a task with **Tomorrow**; confirm it lands on tomorrow's plan and that
today's is unchanged. Then use the date affordance with a day several weeks out and confirm it lands
there.

**Depends on**: US1 (the cycle and the save path it built). This is the one story pair that is not
independent — US2 is the arbitrary-date affordance layered onto US1's control.

### Tests for User Story 2 ⚠️

- [X] T022 [P] [US2] [TUI] Calendar tests in `internal/tui/edit_test.go`: **CT-12** (a calendar pick sets `planDate`, moves the selector to `planChoiceDate`, and renders the ISO date; cycling away clears it), **CT-06** (with `nowFunc` pinned across a midnight boundary, Today resolves to the day at *save* time, not at form-open time). Also assert the Tomorrow path end to end: save with Tomorrow ⇒ `planDay` is tomorrow's ISO day.
- [X] T023 [P] [US2] [WEB] Date-input tests in `services/twig-web/src/components/TaskForm.test.tsx`: the 📅 affordance reveals a labelled date input; picking a date and submitting passes that ISO day as `planDay`; a `"date"` choice with no date entered passes `undefined` rather than a garbage day (spec edge case: unparseable date leaves the control alone).

### Implementation for User Story 2

- [X] T024 [US2] [TUI] Extend the `keys.Calendar` guard in `internal/tui/edit.go` from `(focusDue || focusSnooze)` to also match `focusPlan`. Reuse the existing `ctrl+g` binding — **do not add a key binding** (Principle III, `contracts/ui-contract.md` §2).
- [X] T025 [US2] [TUI] Give the Plan field its own calendar render path in `internal/tui/edit.go` and seed `openCalendar` from `planDate` (falling back to today) rather than from a focused text input. This is the one place existing plumbing does not transfer — `writeDateField` pairs the calendar with a `textinput.Model` the Plan field does not have (`research.md` R3). Do **not** add a hidden `textinput` to reuse the helper; that was considered and rejected as indirection. Depends on T024.
- [X] T026 [US2] [TUI] Handle the calendar pick in `internal/tui/edit.go`: store the ISO day in `planDate` and move `planIdx` to `planChoiceDate` so the field renders the date. Depends on T025.
- [X] T027 [P] [US2] [WEB] Add the 📅 affordance and the revealed `<input type="date">` to the segmented control in `services/twig-web/src/components/TaskForm.tsx`, with an accessible label, matching `AddToPlanControl`'s existing date-input usage. Selecting a date makes it the active choice; clearing it returns the control to its previous value. Depends on T019.

**Checkpoint**: Any day is reachable from the create form on both surfaces.

---

## Phase 5: User Story 3 - Consistent plan control across TUI and web (Priority: P3)

**Goal**: Both surfaces present the same choices, in the same order, with the same default and the
same result — so the control is learnable once.

**Independent Test**: Create a task planned for today in the TUI and another in the web app; confirm
both produce an equivalent untimed entry on the same day, and that both forms open on "No plan".

### Tests for User Story 3 ⚠️

- [X] T028 [P] [US3] [TUI] **CT-07** in `internal/tui/edit_test.go`: `showPlanField` is false for the task edit form and for every goal form (FR-010, FR-011) — assert the Plan field is absent from the rendered view and skipped by `cycleFocus`, so `tab` traversal on those forms is unchanged.
- [X] T029 [P] [US3] [WEB] **CT-07** in `services/twig-web/src/pages/TaskDetailPage.test.tsx`: the edit form renders no plan control (FR-011). This is the regression guard on the deliberate scope boundary — existing tasks are planned from the task list, not the edit form.

### Implementation for User Story 3

- [X] T030 [US3] Reconcile the two surfaces against `contracts/ui-contract.md` §1 (UC-01..UC-08): same four choices, same order, same "No plan" default, same untimed result, same local-time resolution. Fix whichever surface drifted. Any divergence found that is *not* fixable as a bug is a spec question, not a judgment call — surface it.
- [X] T031 [US3] Verify the copy on both surfaces against `contracts/ui-contract.md` §4 (Constitution Principle IV): messages carry the house tone; control labels stay literal ("No plan"/"Today"/"Tomorrow"), because playfulness in a control label costs the clarity the principle explicitly protects.

**Checkpoint**: All three stories are functional and consistent.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T032 [P] Run the full `quickstart.md` walkthrough by hand on both surfaces, including §5's two failure paths (stop the server and create a task with Today selected; confirm no stray plan entry and an honest message).
- [X] T033 [P] Confirm every contract test CT-01..CT-14 in `contracts/ui-contract.md` §5 exists and passes. A missing CT is an incomplete contract (Constitution Quality Gate 1), not a nice-to-have.
- [X] T034 Verify `git diff --stat` touches **no** file under `api/`, `services/twig/`, or any migration — the client-only claim in plan.md is falsifiable, so falsify it.
- [X] T035 Re-check the Constitution gates in `plan.md` § Constitution Check against the code as built: no new abstraction (I), contracts match implementation (II), no new key binding and no one-off styles (III), new copy reviewed for tone (IV).
- [X] T036 Run both suites clean: `go test ./...` from the repo root and `npm test` in `services/twig-web/`, plus `npm run build` to catch type errors the test run misses.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies.
- **Foundational (Phase 2)**: after Setup. **Blocks all user stories.**
- **US1 (Phase 3)**: after Foundational. Independent of US2/US3.
- **US2 (Phase 4)**: after **US1** — it layers the date path onto US1's control. (This is the one
  cross-story dependency; it is inherent to the design, not an artifact of task ordering.)
- **US3 (Phase 5)**: after US1 and US2 — parity can only be checked once both paths exist.
- **Polish (Phase 6)**: after all desired stories.

### Within US1

```
T009–T012 (tests, all parallel)
   │
   ├─ TUI:  T013 → T014 ─┐
   │        T015 ────────┤ (all edit.go — sequential)
   │        T016 ────────┘
   │           └─► T017 → T018   (update.go)
   │
   └─ WEB:  T019 → T020
                 └─► T021
```

TUI (T013–T018) and web (T019–T021) are fully parallel — they share no file and no module.

### Parallel Opportunities

- **T001 ∥ T002** — setup.
- **T006 ∥ T007 ∥ T008** — different files across both surfaces. T003→T004→T005 are all `edit.go`, so they are strictly sequential.
- **T009 ∥ T010 ∥ T011 ∥ T012** — all four test tasks, four different files.
- **The entire TUI track ∥ the entire web track**, within every story. This is the big one: two people can take US1 end to end on each surface and never touch the same file.
- **T022 ∥ T023**, **T028 ∥ T029**, **T032 ∥ T033**.

### File-conflict warnings

These files are touched by multiple tasks and must **not** be worked in parallel:

| File | Tasks |
|---|---|
| `internal/tui/edit.go` | T003, T004, T005, T013, T014, T015, T016, T024, T025, T026 |
| `internal/tui/update.go` | T017, T018 |
| `internal/tui/edit_test.go` | T009, T022, T028 |
| `services/twig-web/src/components/TaskForm.tsx` | T007, T019, T027 |
| `services/twig-web/src/components/TaskForm.test.tsx` | T011, T023 |

---

## Parallel Example: User Story 1

```bash
# All four US1 test tasks at once — four separate files:
Task: "T009 TUI form-state tests (CT-01,02,03,10,13) in internal/tui/edit_test.go"
Task: "T010 TUI save-path tests (CT-04,05,08,09) in internal/tui/plan_from_create_test.go"
Task: "T011 Web component tests (CT-01,02,03,05,14) in services/twig-web/src/components/TaskForm.test.tsx"
Task: "T012 Web page tests (CT-08,09,11) in services/twig-web/src/pages/TaskTreePage.test.tsx"

# Then the two implementation tracks in parallel — no shared files:
Task: "T013–T018 TUI: cycle, render, dirty, planDay, createTaskCmd chain"
Task: "T019–T021 Web: segmented control, TaskTreePage, TreeRow"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1: Setup — know your baseline.
2. Phase 2: Foundational — inert plumbing. **Blocks everything.**
3. Phase 3: US1 — Today works end to end on both surfaces.
4. **STOP and VALIDATE**: create a task with Today and see it on today's plan; create one without
   touching the control and see it on no plan. That second check is SC-003/SC-005 and matters more
   than the first — the feature is worthless if it changes what happens to people who ignore it.
5. Shippable. US1 alone delivers the step this feature set out to save.

### Incremental Delivery

1. Setup + Foundational → plumbing ready.
2. + US1 → **MVP**: today's plan in one save.
3. + US2 → any day reachable from the form.
4. + US3 → verified parity across surfaces.
5. + Polish → contracts and constitution gates confirmed against the built code.

### Parallel Team Strategy

The natural split is **by surface, not by story**. After Phase 2:

- Developer A: the TUI track (T013–T018, then T024–T026)
- Developer B: the web track (T019–T021, then T027)

They share no file. Both converge on US3 (T030–T031), which is inherently a two-surface task and
should be done by one person looking at both.

---

## Notes

- **[P] means different files.** The tables above list every multi-task file; check them before
  parallelizing anything.
- **Tests first within each story** — write them, watch them fail, then implement. Each CT names the
  requirement it guards, so a red test tells you which contract you broke.
- **Never roll back a created task** because the plan write failed (FR-007). If you find yourself
  writing a compensating delete, re-read `data-model.md` § "Write sequence and failure modes" — that
  path was rejected deliberately. A failed convenience must not destroy real work.
- **`start_minute` absent, `duration_minute: 0`** on every `AddPlanTask` call. Sending a start minute
  invents a time the user never chose.
- Commit after each task or logical group. Stop at any checkpoint to validate.
