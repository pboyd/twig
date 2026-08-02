---
description: "Task list for 073-web-task-parity"
---

# Tasks: Task Parity for the Web App

**Input**: Design documents from `/specs/073-web-task-parity/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/web-task-actions.md, quickstart.md

**Tests**: Test tasks ARE included. The plan's Source Code layout explicitly lists `dateFields.test.ts`, `effectiveGoal.test.ts` and extended `TaskForm.test.tsx` / `TaskDetailPage.test.tsx`, and the repo convention (AGENTS.md) requires non-trivial logic in `src/lib/` to ship with a unit test.

**Organization**: Grouped by user story so each can be implemented, tested, and shipped independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1–US4 from spec.md
- All paths are relative to the repo root; frontend work is confined to `services/twig-web/`

## Path Conventions

Frontend-only feature. All source paths live under `services/twig-web/src/`. All commands run from `services/twig-web/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the workspace is ready; no scaffolding needed (existing module, existing routes, existing proxy entries for `task.v1` and `goal.v1`).

- [X] T001 Verify the frontend toolchain is green before changes: run `npm install` and `npm test` in `services/twig-web/` and confirm the existing suite passes
- [X] T002 Confirm no regeneration is needed: verify `deleteTask`, `updateTask`, and `setTaskGoal` are exported from `services/twig-web/src/gen/task/v1/task-TaskService_connectquery.ts` and `listGoals` from `services/twig-web/src/gen/goal/v1/goal-GoalService_connectquery.ts` (no `npm run gen`, no proto change in this feature)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The shared copy surface every story writes into. Adding all new message keys in one pass avoids four stories editing the same file in conflicting ways.

**⚠️ CRITICAL**: T003 blocks the UI tasks of every user story (Principle IV — no hardcoded strings in components).

- [X] T003 Add all new user-facing copy to `services/twig-web/src/theme/messages.ts` in the existing warm/playful tone, as one grouped addition: delete flow (`deleteTaskConfirm`, `deleteTaskConfirmWithSubtasks`, `taskDeleted`, `taskAlreadyGone`, `taskDeleteError`), schedule fields (`dueLabel`, `snoozeLabel`, `noDueDate`, `snoozedUntil` as a `(date: string) => string`), and goal linking (`goalSectionHeading`, `noGoalLinked`, `goalInheritedFrom` as a `(goalName: string) => string`, `noGoalsToPick`, `goalLinked`, `goalUnlinked`, `goalLinkBlocked`, `goalLinkError`)

**Checkpoint**: Copy is in place — user stories can proceed in parallel.

---

## Phase 3: User Story 1 - Delete a task (Priority: P1) 🎯 MVP

**Goal**: A user can delete a task from `TaskDetailPage` after an explicit confirmation that names the subtask cascade, then lands back on `/tasks` with the task and its descendants gone.

**Independent Test**: Create a task with a subtask, open the parent's detail page, delete it, confirm — both disappear from the task list and the app navigates to `/tasks`.

### Tests for User Story 1 ⚠️

> Write these first and confirm they fail before implementing T006.

- [X] T004 [US1] Add delete-flow tests to `services/twig-web/src/pages/TaskDetailPage.test.tsx`: clicking Delete shows the inline confirmation without calling `DeleteTask`; the confirmation copy mentions subtasks only when the task has children; Cancel dismisses the panel and leaves the task view unchanged (spec scenarios 1–3)
- [X] T005 [US1] Add delete outcome tests to `services/twig-web/src/pages/TaskDetailPage.test.tsx`: confirming calls `DeleteTask` with the task id, then invalidates `listTasks` and navigates to `/tasks` with a success toast; a `Code.NotFound` failure shows the "already gone" message and still navigates/refreshes; any other error shows the generic connectivity message and stays on the page (spec scenario 4, contract §1)

### Implementation for User Story 1

- [X] T006 [US1] Implement the delete flow in `services/twig-web/src/pages/TaskDetailPage.tsx`: import `deleteTask` from the task connectquery module, add a `confirmingDelete` state, render a destructive "Delete" `Button` in the view-mode action row that swaps to an inline confirmation panel (warning copy from T003, chosen by whether `subTasks.length > 0`, with confirm/cancel `Button`s reusing the existing card styling), and on confirm call `deleteTask({ id: task.id })`, invalidate the `listTasks` query key via `createConnectQueryKey`, show the success `Toast`, and `navigate("/tasks")` — mapping `Code.NotFound` to the "already gone" message (still invalidating and navigating) and all other `ConnectError`s to `messages.connectivityError` via the existing `ErrorBanner`

**Checkpoint**: US1 is fully functional and independently testable — deletion works end to end from the web app.

---

## Phase 4: User Story 2 - Set or clear a task's due date (Priority: P2)

**Goal**: A user editing a task can set, change, or clear its due date; the date shows on the detail card and survives a reload, and no other field changes.

**Independent Test**: Edit a task, set a due date, save, reload and see the date; edit again, clear it, and see it gone — with name, description, snooze, and tree position untouched.

### Tests for User Story 2 ⚠️

- [X] T007 [P] [US2] Create `services/twig-web/src/lib/dateFields.test.ts` covering both directions of the ISO-day ⇄ `Timestamp` conversion: `"2026-08-02"` → `Timestamp` at midnight UTC (seconds exactly, `nanos` 0); a midnight-UTC `Timestamp` → `"2026-08-02"`; `""`/`undefined` round-trips to `undefined`/`""`; past dates convert without complaint; conversion is lossless in both directions
- [X] T008 [P] [US2] Add due-date cases to `services/twig-web/src/lib/updatePayload.test.ts`: an edited due value overrides `task.due`; an empty due value omits `due` from the payload (clearing it); `parentId` and `snoozeUntil` are still carried through unchanged; `goalId` is never present in the payload (contract §2)
- [X] T009 [US2] Add due-field tests to `services/twig-web/src/components/TaskForm.test.tsx`: the "Due" date input renders only when `showScheduleFields` is set (absent in the default create flow), is prefilled from `initialDue`, and its value reaches `onSubmit` as an ISO day string (`""` when cleared)

### Implementation for User Story 2

- [X] T010 [US2] Create `services/twig-web/src/lib/dateFields.ts` with two pure exported functions — `isoDayToTimestamp(day: string): Timestamp | undefined` (midnight UTC of the calendar day, `undefined` for empty input) and `timestampToIsoDay(ts: Timestamp | undefined): string` (UTC calendar fields, `""` for undefined) — documenting the midnight-UTC contract from research R3
- [X] T011 [US2] Extend `buildUpdatePayload` in `services/twig-web/src/lib/updatePayload.ts` to accept the edited due and snooze values (as `Timestamp | undefined`) and emit them in place of `task.due` / `task.snoozeUntil`, keeping the existing `parentId` pass-through comment and behaviour intact
- [X] T012 [US2] Add the "Due" date field to `services/twig-web/src/components/TaskForm.tsx`: new optional props `showScheduleFields`, `initialDue` (ISO day string) and a `due` state driving an `<input type="date">` (labelled, min 44px touch target, matching the existing plan-date input styling), gated on `showScheduleFields` so the create flow is unchanged, and widen the `onSubmit` signature to pass the ISO day string through
- [X] T013 [US2] Wire due-date editing in `services/twig-web/src/pages/TaskDetailPage.tsx`: pass `showScheduleFields` and `initialDue={timestampToIsoDay(task.due)}` to the edit-mode `TaskForm`, convert the submitted ISO day back via `isoDayToTimestamp`, and hand it to `buildUpdatePayload` inside the existing fresh-refetch guard in `handleSaveEdit` (contract §2)
- [X] T014 [US2] Display the due date on the view-mode card in `services/twig-web/src/pages/TaskDetailPage.tsx` using the existing `formatDueDate` from `src/lib/formatTimestamp.ts`, with the "no due date" copy from T003 when unset

**Checkpoint**: US1 and US2 both work independently.

---

## Phase 5: User Story 3 - Snooze a task (Priority: P2)

**Goal**: A user can set, change, or clear a task's snooze date; snoozed tasks read as snoozed on the detail card and keep the existing 💤 marker in the tree.

**Independent Test**: Snooze a task until tomorrow from the web app, verify the detail card says so and the tree row shows 💤; clear the snooze and verify both markers disappear.

**Note**: US3 shares `dateFields.ts` (T010), the `buildUpdatePayload` extension (T011), and the `showScheduleFields` gate (T012) with US2. If US3 is built first, those three tasks come with it; they are listed once, in US2.

### Tests for User Story 3 ⚠️

- [X] T015 [P] [US3] Add snooze cases to `services/twig-web/src/lib/updatePayload.test.ts`: an edited snooze value overrides `task.snoozeUntil`; an empty snooze value omits `snoozeUntil` (un-snoozing); a payload carrying both an edited due and an edited snooze applies both together (spec edge case); `parentId` still carried through
- [X] T016 [US3] Add snooze-field tests to `services/twig-web/src/components/TaskForm.test.tsx`: the "Snooze until" input renders only under `showScheduleFields`, is prefilled from `initialSnooze`, and reaches `onSubmit` as an ISO day string (`""` when cleared)
- [X] T017 [US3] Add snooze display tests to `services/twig-web/src/pages/TaskDetailPage.test.tsx`: a task snoozed to a future day shows the "Snoozed until …" line; a task with no snooze, or one whose snooze day is today or past, shows no snooze marker (spec scenarios 1–2)

### Implementation for User Story 3

- [X] T018 [US3] Add the "Snooze until" date field to `services/twig-web/src/components/TaskForm.tsx` alongside the due field (same `showScheduleFields` gate, new `initialSnooze` prop, ISO day string passed through `onSubmit`)
- [X] T019 [US3] Wire snooze editing in `services/twig-web/src/pages/TaskDetailPage.tsx`: pass `initialSnooze={timestampToIsoDay(task.snoozeUntil)}`, convert the submitted value with `isoDayToTimestamp`, and pass it to `buildUpdatePayload` in `handleSaveEdit`
- [X] T020 [US3] Show snooze state on the view-mode card in `services/twig-web/src/pages/TaskDetailPage.tsx`: render the "Snoozed until …" copy when the `snoozeUntil` UTC calendar day is strictly after the local current day, matching the existing `isSnoozed` comparison in `services/twig-web/src/components/TreeRow.tsx` (reuse the tree's 💤 marker as-is; no TreeRow change)

**Checkpoint**: US1–US3 all work independently; the whole date-editing surface is complete.

---

## Phase 6: User Story 4 - Link or unlink a task with a goal (Priority: P3)

**Goal**: From a task's detail page a user can link it to a goal, switch goals, or unlink — with inherited goals shown read-only and rule rejections explained in plain language.

**Independent Test**: Link a root task to a goal, see the goal on the task and the task on the goal's page; switch goals; unlink. On a subtask of a linked task, see the goal marked inherited with no controls.

### Tests for User Story 4 ⚠️

- [X] T021 [P] [US4] Create `services/twig-web/src/lib/effectiveGoal.test.ts`: a task with its own `goalId` resolves as `direct`; a task whose nearest goal-bearing ancestor has one resolves as `inherited` with that ancestor's goal id; the nearest ancestor wins when several ancestors carry goals; a task with no goal anywhere in its chain resolves to none; a root task with no goal resolves to none; a missing/broken `parentId` link terminates the walk without looping
- [X] T022 [US4] Add goal-section tests to `services/twig-web/src/pages/TaskDetailPage.test.tsx`: no goal → picker listing only non-hidden goals, and selecting one calls `SetTaskGoal` with `{ taskId, goalId }` then invalidates `getTask` + `listTasks` (scenario 1); a direct goal → goal name linked to its page plus switch/unlink controls, where unlink calls `SetTaskGoal` with `goalId` unset (scenarios 2–3); an inherited goal → read-only inherited explanation with no controls (scenario 4); an empty `listGoals` result → the friendly no-goals message instead of an empty `<select>` (scenario 6)
- [X] T023 [US4] Add goal error-mapping tests to `services/twig-web/src/pages/TaskDetailPage.test.tsx`: a `Code.FailedPrecondition` from `SetTaskGoal` renders the plain-language rule explanation and leaves the displayed goal unchanged (scenario 5, FR-008, SC-003); `Code.NotFound` renders the gone message and refreshes; other codes render the generic connectivity message

### Implementation for User Story 4

- [X] T024 [P] [US4] Create `services/twig-web/src/lib/effectiveGoal.ts` exporting a pure resolver that takes a task id and the flat `ListTasks` array and returns the nearest goal along the ancestor chain plus whether it is `direct` (the task itself) or `inherited`, with a visited-set guard so a cyclic or dangling `parentId` cannot loop (data-model "Derived")
- [X] T025 [US4] Add the Goal section to view mode in `services/twig-web/src/pages/TaskDetailPage.tsx`: query `listGoals`, resolve the effective goal with `effectiveGoal` over the already-loaded `allTasksData.tasks`, and render the three states — direct (goal name as a `Link` to `/goals/:id` plus switch/unlink controls), inherited (name plus the inherited explanation, no controls), none (a native `<select>` of goals ordered by `goalGroups(goals, false)` so Completed/Archived are excluded, or the no-goals message when that list is empty)
- [X] T026 [US4] Implement the link/unlink mutations in `services/twig-web/src/pages/TaskDetailPage.tsx`: call `setTaskGoal({ taskId, goalId })` to link or switch and `setTaskGoal({ taskId })` to unlink, then invalidate the `getTask` and `listTasks` query keys (not `listGoals`) and show the matching success toast (contract §3)
- [X] T027 [US4] Map `SetTaskGoal` failures in `services/twig-web/src/pages/TaskDetailPage.tsx`: `Code.FailedPrecondition` → the goal-rule explanation covering both server causes (an ancestor's or a descendant's existing goal), `Code.NotFound` → the task-or-goal-gone message plus a `listTasks` refresh, everything else → `messages.connectivityError`, all surfaced through the existing `ErrorBanner` with the task left unchanged

**Checkpoint**: All four user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T028 Verify FR-009 end to end with a test in `services/twig-web/src/pages/TaskDetailPage.test.tsx`: editing only the due date leaves name, description, snooze, `parentId`, and the goal link untouched in the `UpdateTask` payload (the fresh-refetch guard in `handleSaveEdit` still runs first)
- [X] T029 Confirm out-of-scope boundaries held (FR-011): no parent-move control, no Pomodoro UI, and no task filtering were added in `services/twig-web/src/pages/TaskDetailPage.tsx` or `services/twig-web/src/components/TaskForm.tsx`
- [X] T030 Run `npm test` and `npm run build` in `services/twig-web/` and confirm both pass with no TypeScript errors
- [X] T031 Walk through `specs/073-web-task-parity/quickstart.md` against a live stack (`make dev` + `npm run dev`), verifying all four manual scenarios
- [X] T032 Update the Frontend section of `AGENTS.md` to note that the SPA now covers task deletion, due/snooze editing, and goal linking (`CLAUDE.md` is a symlink — edit `AGENTS.md` only)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: T003 blocks every UI task in Phases 3–6
- **US1 (Phase 3)**: Depends only on Phase 2 — fully independent of US2–US4
- **US2 (Phase 4)**: Depends on Phase 2
- **US3 (Phase 5)**: Depends on Phase 2; shares T010/T011/T012 with US2 (whichever story is built first lands them)
- **US4 (Phase 6)**: Depends on Phase 2 — independent of US1–US3
- **Polish (Phase 7)**: Depends on all desired stories

### Within Each User Story

- Tests first, confirmed failing, then implementation
- Lib modules before the components that import them (T010 → T012/T013; T024 → T025)
- Component props before page wiring (T012 → T013, T018 → T019)
- Mutation wiring before error mapping (T026 → T027)

### Same-File Serialization (why some tasks are not [P])

- `TaskDetailPage.tsx` is touched by T006, T013, T014, T019, T020, T025, T026, T027 — never mark these [P] against each other
- `TaskForm.tsx` is touched by T012 and T018 — sequential
- `TaskDetailPage.test.tsx` is touched by T004, T005, T017, T022, T023, T028 — sequential
- `updatePayload.test.ts` is touched by T008 and T015 — sequential

### Parallel Opportunities

- T007, T008, T021 are in three separate new/existing test files and can be written in parallel
- T010 and T024 are independent new lib modules (though T010 lands with US2)
- With multiple developers: after T003, one takes US1 (T004–T006), one takes US2+US3 (T007–T020, sequential between them because of shared files), one takes US4 (T021–T027)

---

## Parallel Example: Independent Test Authoring

```bash
# After T003, three test files can be written concurrently:
Task: "Create src/lib/dateFields.test.ts covering ISO day ⇄ Timestamp conversion"
Task: "Add due-date cases to src/lib/updatePayload.test.ts"
Task: "Create src/lib/effectiveGoal.test.ts covering direct vs inherited resolution"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1: Setup (T001–T002)
2. Phase 2: Foundational (T003)
3. Phase 3: US1 — delete a task (T004–T006)
4. **STOP and VALIDATE**: quickstart step 1 — delete a parent with a subtask, confirm both vanish
5. Ship: the single most conspicuous gap is closed

### Incremental Delivery

1. Setup + Foundational → copy surface ready
2. + US1 → deletion works (MVP)
3. + US2 → due dates editable (brings `dateFields.ts` and the payload extension with it)
4. + US3 → snoozing works, riding on US2's plumbing (small increment)
5. + US4 → goal linking, the rule-heaviest story, last
6. Phase 7 → verify, build, document

---

## Notes

- No server, proto, or `vite.config.ts` changes — `task.v1` and `goal.v1` are already proxied
- `goal_id` must never appear in an `UpdateTask` payload; goal writes go through `SetTaskGoal` only
- Every new string goes in `src/theme/messages.ts` (T003), never inline in a component
- Commit after each task or logical group
