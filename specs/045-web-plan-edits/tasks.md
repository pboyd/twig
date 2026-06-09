---
description: "Task list for Basic Planning Edits on the Web App"
---

# Tasks: Basic Planning Edits on the Web App

**Input**: Design documents from `/specs/045-web-plan-edits/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/consumed-apis.md, quickstart.md

**Tests**: Included. This codebase keeps a colocated test for every page/helper (`PlanPage.test.tsx`, `TaskTreePage.test.tsx`, `lib/*.test.ts`); new/edited surfaces follow that convention.

**Scope reminder**: Frontend-only, all paths under `services/twig-web/`. No proto/server/DB changes and no `npm run gen` (consumed contracts already exist — see `contracts/consumed-apis.md`).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1–US4 maps to the user stories in spec.md

## Path Conventions

All paths are relative to `services/twig-web/`.

---

## Phase 1: Setup (Shared)

**Purpose**: Shared copy used by multiple stories, added once to avoid cross-story edits to the same file.

- [ ] T001 [P] Add new playful user-facing copy to `src/theme/messages.ts`: `addedToToday`, `addedToDay` (a `(label: string) => string` builder), `alreadyOnPlan` (duplicate confirmation), `entryRemoved`, and an `addFailed`/reuse of `connectivityError` for failures. Keep the warm/playful tone (Principle IV); leave existing `completeBlockedBySubtasks`/`connectivityError` in place for reuse.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The shared transient-toast primitive (FR-005), used by US1 and US3. App-level mount done once here.

**⚠️ CRITICAL**: US1 and US3 depend on this phase. US2 and US4 do not import the toast and may proceed in parallel with it.

- [ ] T002 Create `src/components/Toast.tsx` — presentational, auto-dismissing toast styled only from `src/theme/tokens.ts` (success/error tone), tap-to-dismiss, accessible (`role="status"`).
- [ ] T003 Create `src/context/ToastProvider.tsx` — `ToastProvider` (holds a small queue) + `useToast()` hook exposing `show(message, tone?)`; each toast auto-removes after ~3s; renders a fixed overlay of `Toast` items.
- [ ] T004 Wrap the router in `src/App.tsx` with `<ToastProvider>` so `useToast()` is available on every page.
- [ ] T005 [P] Create `src/context/ToastProvider.test.tsx` — `show()` renders the message and it auto-dismisses; multiple toasts stack.

**Checkpoint**: `useToast()` available app-wide; toast verified in isolation.

---

## Phase 3: User Story 1 - Add a task to today's plan (Priority: P1) 🎯 MVP

**Goal**: A one-tap "add to today" control on each task row that creates an untimed plan entry and confirms with a toast.

**Independent Test**: On the Tasks tab, tap a task's add-to-today control → toast confirms → the task shows in today's Untimed group on the Plan tab; a second tap reports it's already on today's plan with no duplicate.

- [ ] T006 [P] [US1] Create `src/lib/planDays.ts` with `todayIso()` (local `YYYY-MM-DD`, reusing/extending `planView.todayString`) and `dayPickerLabel(iso)` ("Today" / else `formatDayLabel`).
- [ ] T007 [P] [US1] Create `src/lib/planDays.test.ts` covering `todayIso()` format and `dayPickerLabel()` for today vs other days.
- [ ] T008 [US1] Create `src/components/AddToPlanControl.tsx` (today path only): a primary icon button that calls `addPlanTask({ day: todayIso(), taskId, durationMinute: 0 })` (no `startMinute`), shows `messages.addedToToday` via `useToast` on success, maps `ConnectError` `Code.FailedPrecondition` → `messages.alreadyOnPlan` toast, other errors → `messages.connectivityError` toast; on success `invalidateQueries` for `listPlanEntries` keyed to `todayIso()`. Accept `taskId: bigint` as a prop. (depends on T001, T003, T006)
- [ ] T009 [US1] Mount `<AddToPlanControl taskId={task.id} />` in the `TreeRow.tsx` action cluster (next to add-sub-task), with a 44px touch target consistent with sibling buttons. (depends on T008)
- [ ] T010 [US1] Update `src/pages/TaskTreePage.test.tsx` (and/or a new `src/components/AddToPlanControl.test.tsx`): add-to-today success shows toast + fires `addPlanTask` with `day=today` and no `startMinute`; `FailedPrecondition` shows "already on plan" copy; transport error shows failure copy and leaves the list unchanged.

**Checkpoint**: US1 fully functional — adding a task to today from the list works and is independently testable.

---

## Phase 4: User Story 2 - Complete a task from the planning tab (Priority: P2)

**Goal**: A per-entry complete control on task-linked planner entries that marks the task done without leaving the plan.

**Independent Test**: On the Plan tab, tap a task entry's complete control → untimed entry disappears (044) or timed entry shows done; the task reads complete on the Tasks tab; an entry whose task has incomplete sub-tasks shows an inline block message; events expose no complete control.

- [ ] T011 [P] [US2] Edit `src/components/PlanEntryRow.tsx` to render a **complete** control only when `entry.kind === "task" && entry.taskId !== undefined && !entry.completed`; add `onComplete(entry)`, `completing?: boolean`, and `actionError?: string` props; render `actionError` inline (amber style as in `TreeRow`). No control for events.
- [ ] T012 [US2] In `src/pages/PlanPage.tsx`, wire `useMutation(completeTask)` and a `handleComplete(entry)` that calls `completeTask({ id: entry.taskId })`, then `invalidateQueries` for `listPlanEntries` (current `day`) and `listTasks`; map `Code.FailedPrecondition` → `messages.completeBlockedBySubtasks`, else `messages.connectivityError`, surfaced via the row's `actionError`; pass handler + per-entry pending/error into `PlanEntryRow`. (depends on T011)
- [ ] T013 [US2] Update `src/pages/PlanPage.test.tsx`: completing a task fires `completeTask` and invalidates plan + tasks queries; blocked-by-subtasks shows the inline message and leaves the entry; event rows render no complete control.

**Checkpoint**: US1 and US2 both work independently.

---

## Phase 5: User Story 3 - Add a task to a future day's plan (Priority: P2)

**Goal**: Extend the add control with day selection — Today shortcut (from US1) plus a Tomorrow preset and an arbitrary-date picker.

**Independent Test**: From a task's other-day picker, choose Tomorrow or an arbitrary date, confirm Add (≤3 actions) → toast names the day → the task appears in that day's Untimed group; a past date still adds.

- [ ] T014 [P] [US3] Extend `src/lib/planDays.ts` with `tomorrowIso()` (`addDays(todayIso(),1)`) and extend `dayPickerLabel` to return "Tomorrow"; update `src/lib/planDays.test.ts` accordingly. (depends on T006)
- [ ] T015 [US3] Extend `src/components/AddToPlanControl.tsx` with a popover affordance: a **Tomorrow** quick preset and a native `<input type="date">` + an **Add** confirm (disabled until a date is chosen); each calls `addPlanTask({ day: <iso>, taskId, durationMinute: 0 })`, shows `messages.addedToDay(dayPickerLabel(iso))` on success and `invalidateQueries` for `listPlanEntries` keyed to that day; past dates allowed (no lower bound). Reuse `Field`/`Button` styling. (depends on T008, T014)
- [ ] T016 [US3] Update `src/pages/TaskTreePage.test.tsx`/`AddToPlanControl.test.tsx`: Tomorrow preset and an arbitrary picked date each fire `addPlanTask` with the right `day`; toast names the day; a past date is accepted (not blocked).

**Checkpoint**: US1–US3 independently functional.

---

## Phase 6: User Story 4 - Remove an entry from the plan (Priority: P3)

**Goal**: A per-entry remove control on the planner for any entry kind; the linked task is untouched.

**Independent Test**: On the Plan tab, remove an entry (≤2 actions) → it disappears from the day; the linked task still exists on the Tasks tab; removal works for untimed, timed, and event entries.

- [ ] T017 [US4] Edit `src/components/PlanEntryRow.tsx` to add a **remove** control for **all** entry kinds; add `onRemove(entry)`, `removing?: boolean` props and reuse the inline `actionError` slot from T011 for failures. (sequential after T011 — same file)
- [ ] T018 [US4] In `src/pages/PlanPage.tsx`, wire `useMutation(removePlanEntry)` and `handleRemove(entry)` calling `removePlanEntry({ day, id: entry.id })`, then `invalidateQueries` for `listPlanEntries` (current `day`); on error surface `messages.connectivityError` via the row's `actionError`; optionally show `messages.entryRemoved` toast. (sequential after T012 — same file)
- [ ] T019 [US4] Update `src/pages/PlanPage.test.tsx`: removing an untimed/timed/event entry fires `removePlanEntry` and invalidates the day's plan; a failed remove leaves the entry and shows the inline error.

**Checkpoint**: All four user stories independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T020 [P] Run `npm run build` (type-check) and `npm test` in `services/twig-web/`; fix any type or test fallout.
- [ ] T021 [P] Review all new copy in `src/theme/messages.ts` for Principle IV tone and verify the new controls + toast use only `theme/tokens.ts` styling for Principle III consistency.
- [ ] T022 Run the `quickstart.md` manual verification for US1–US4 against `make dev` + `npm run dev`.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: After Setup. Blocks US1 and US3 (toast). Does NOT block US2/US4.
- **US1 (Phase 3)**: After Foundational. MVP.
- **US2 (Phase 4)**: After Setup; independent of US1/US3. (Touches `PlanEntryRow.tsx` + `PlanPage.tsx`.)
- **US3 (Phase 5)**: After US1 (extends `AddToPlanControl` + `planDays`) and Foundational.
- **US4 (Phase 6)**: After US2 for file-ordering only — US4 edits the same `PlanEntryRow.tsx` (T017 after T011) and `PlanPage.tsx` (T018 after T012). Functionally independent of US1/US3.
- **Polish (Phase 7)**: After all desired stories.

### Within / across stories

- T001 (messages) before T008/T015/T018 that reference the new copy.
- T002 → T003 → T004 (toast build → provider → app mount); T005 after T003.
- T006 before T008 and T014; T008 before T009 and T015.
- **Shared-file ordering**: US2 and US4 both modify `PlanEntryRow.tsx` and `PlanPage.tsx`, so they are NOT parallel with each other — do US2 then US4 (T011→T017, T012→T018).

### Parallel Opportunities

- T001 and the Phase 2 toast work touch different files and can overlap (T001 [P]).
- Within US1: T006 and T007 [P] (helper + its test) before T008.
- US2 (Phase 4) can run in parallel with US1/US3 by a second developer (different files: `PlanEntryRow.tsx`/`PlanPage.tsx` vs `AddToPlanControl.tsx`/`TreeRow.tsx`).
- Polish T020 and T021 [P].

---

## Parallel Example: User Story 1

```bash
# Helper + its unit test together, before the control:
Task: "Create src/lib/planDays.ts (todayIso, dayPickerLabel)"
Task: "Create src/lib/planDays.test.ts"
```

```bash
# Cross-story parallelism after Setup + Foundational:
Developer A: US1 (AddToPlanControl + TreeRow)        # add-to-today
Developer B: US2 (PlanEntryRow complete + PlanPage)  # complete from planner
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 (Setup) → Phase 2 (Foundational toast) → Phase 3 (US1).
2. **STOP and VALIDATE**: add-to-today works end-to-end (success, duplicate, failure).
3. Demo: the web app can now feed today's plan from the task list.

### Incremental Delivery

1. Setup + Foundational → toast ready.
2. US1 → add-to-today (MVP) → validate/demo.
3. US2 → complete-from-planner → validate/demo.
4. US3 → add-to-future-day (extends US1) → validate/demo.
5. US4 → remove-entry → validate/demo.

### Notes

- [P] = different files, no incomplete-task dependency.
- No backend, proto, or generated-code changes — do not run `npm run gen`.
- Commit after each task or logical group.
- Keep new copy playful (Principle IV) and styling token-driven (Principle III).
