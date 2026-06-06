---

description: "Task list for Web Day Planner (View)"
---

# Tasks: Web Day Planner (View)

**Input**: Design documents from `/specs/041-web-day-planner/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/plan-service.md, quickstart.md

**Tests**: Included. This frontend follows a test-alongside convention (every page/lib has a `.test.tsx`/`.test.ts`), and plan.md/quickstart.md specify unit + interaction tests. Test tasks are written first within each phase and must fail before implementation.

**Organization**: Tasks are grouped by user story. This is a **frontend-only** feature — all paths are under `services/twig-web/`. No backend, proto, or codegen changes (the `plan.v1.PlanService` RPC and its TS stubs already exist).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 / US2 / US3 (maps to spec.md user stories)

## Path Conventions

All files live under `services/twig-web/`. Run all `npm` commands from that directory.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Wiring needed before the view can talk to the server or render copy.

- [ ] T001 Add `"/plan.v1": "http://localhost:8080"` to the `server.proxy` map in `services/twig-web/vite.config.ts` (mirrors existing `/task.v1`, `/health.v1` entries) so the same-origin session cookie reaches `PlanService` in dev.
- [ ] T002 [P] Add `planEmpty` (no plan for the viewed day) and `planError` (plan failed to load) copy in `services/twig-web/src/theme/messages.ts`, matching the existing warm/playful tone (Principle IV).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The pure, framework-free view helpers consumed by every user story.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T003 [P] Write unit tests in `services/twig-web/src/lib/planView.test.ts` for: `todayString()`, `addDays(day, delta)` (incl. month/year rollover), `isToday(day)`, `formatDayLabel(day)`, `formatMinute(min)` (boundaries: 0→"12:00 am", 720→"12:00 pm", 1439→"11:59 pm"), `formatTimeRange(start, duration)`, `resolveEntries(entries, taskNameById)` (name override → task name → "Untitled entry" fallback; sets `kind`, `completed`, `timed`, time fields), `groupPlan(resolved)` (timed/untimed split, `isEmpty`). Write FIRST; ensure they fail.
- [ ] T004 Implement the pure helpers in `services/twig-web/src/lib/planView.ts` (types `ResolvedEntry`, `GroupedPlan` per data-model.md) so T003 passes. No React, no network, no `Date`-based timezone conversion (wall-clock from `start_minute`, FR-020).

**Checkpoint**: `npm test` green for `planView.test.ts`; helpers ready for all stories.

---

## Phase 3: User Story 1 - Check today's plan on the go (Priority: P1) 🎯 MVP

**Goal**: A signed-in user can reach a day planner from the app nav and see today's plan — timed entries in chronological order with wall-clock time labels and names, task entries distinct from events, plus loading/error/empty states.

**Independent Test**: With a plan created for today (timed task entries + an event), open the app on a mobile-sized viewport, navigate to Plan, and confirm today's entries render in time order with start–end labels, task vs event are distinguishable, and an empty day shows the empty state.

### Tests for User Story 1

- [ ] T005 [US1] Write interaction tests in `services/twig-web/src/pages/PlanPage.test.tsx` (US1 slice): mocks `listPlanEntries`/`listTasks`; asserts timed entries render in order with time labels; task-linked name shows (incl. fallback to task name and to generic label); event is visually distinguishable; loading spinner, error banner + retry, and empty state (`messages.planEmpty`) each render. Write FIRST; ensure they fail.

### Implementation for User Story 1

- [ ] T006 [P] [US1] Create `services/twig-web/src/components/PlanEntryRow.tsx` — renders one `ResolvedEntry` (display-only for now): time-range label for timed entries, display name, and a visual distinction between task entries and events (FR-005, FR-006, FR-007, FR-019). Uses Tailwind tokens/spacing consistent with existing components (Principle III).
- [ ] T007 [US1] Create `services/twig-web/src/pages/PlanPage.tsx` — defaults the viewed day to `planView.todayString()`; `useQuery(listPlanEntries, { day })` + `useQuery(listTasks, {})`; builds `taskNameById`, runs `resolveEntries` + `groupPlan`; renders the timed list via `PlanEntryRow`; uses `Spinner` (loading), `ErrorBanner` + `refetch` (FR-017), inline empty state with `messages.planEmpty` (FR-012); wraps in `AppHeader`. (Depends on T004, T006.)
- [ ] T008 [US1] Add `<Route path="/plan" element={<PlanPage />} />` in `services/twig-web/src/App.tsx`, keeping `/tasks` as the default landing route. (Depends on T007.)
- [ ] T009 [P] [US1] Add Tasks ↔ Plan navigation links in `services/twig-web/src/components/AppHeader.tsx` (reachable from primary nav, FR-002) using `react-router` navigation, consistent across pages.

**Checkpoint**: Navigate to `/plan`, see today's plan render with correct times/names/states. US1 is independently demoable (MVP).

---

## Phase 4: User Story 2 - Understand each entry at a glance (Priority: P2)

**Goal**: Each entry shows enough to act on it — duration/time span, completed marker for finished task entries, untimed entries grouped separately, and task entries link through to their detail page (events stay inert).

**Independent Test**: With a plan containing a timed task entry, a completed task entry, an untimed entry, and an event, confirm the completed task is marked done, untimed items appear in a separate section, tapping a task entry opens `/tasks/:taskId`, and tapping an event does nothing.

### Tests for User Story 2

- [ ] T010 [US2] Extend `services/twig-web/src/pages/PlanPage.test.tsx` (US2 slice): untimed entries render in a separate section from timed (FR-009); a completed task entry is marked done (FR-008); a task entry navigates to `/tasks/:taskId` on activation (FR-011); an event entry is non-interactive (FR-021); a gap between timed entries is apparent (FR-010). Write FIRST; ensure they fail.

### Implementation for User Story 2

- [ ] T011 [US2] Enhance `services/twig-web/src/components/PlanEntryRow.tsx`: render a completed marker when `completed` (FR-008); render task entries as an activatable link/button to `/tasks/:taskId` and keep event entries display-only/non-interactive (FR-011, FR-021).
- [ ] T012 [P] [US2] Update `services/twig-web/src/pages/PlanPage.tsx` to render the `groupPlan` untimed list in a clearly separated section below the timed schedule (FR-009), with timed entries showing start–end so gaps are visible (FR-010).

**Checkpoint**: US1 + US2 both work; entries are fully informative and task entries are navigable.

---

## Phase 5: User Story 3 - Look at another day's plan (Priority: P3)

**Goal**: The user can step to adjacent days and return to today; a day with no plan shows the empty state for that date.

**Independent Test**: With plans on two dates, step forward/back a day and confirm the displayed plan and date update; "Today" returns to today; a day with no plan shows the empty state naming that date.

### Tests for User Story 3

- [ ] T013 [US3] Extend `services/twig-web/src/pages/PlanPage.test.tsx` (US3 slice): prev/next controls change the day passed to `listPlanEntries` and update the displayed date label; "Today" control restores today; a navigated day with no entries shows the empty state for that date (FR-018, FR-012). Write FIRST; ensure they fail.

### Implementation for User Story 3

- [ ] T014 [US3] Add day navigation to `services/twig-web/src/pages/PlanPage.tsx`: hold the viewed day in state (default today), add previous/next-day and "Today" controls using `planView.addDays`/`isToday`/`formatDayLabel`, and show the current date label. The plan query re-runs when the day changes; no arbitrary date picker (FR-018).

**Checkpoint**: All three user stories independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verification and consistency across the feature.

- [ ] T015 [P] Run `npm test` in `services/twig-web` and resolve any failures across `planView.test.ts` and `PlanPage.test.tsx`.
- [ ] T016 [P] Run `npm run build` in `services/twig-web` (type-check + production build) and fix any type errors.
- [ ] T017 Execute the `quickstart.md` manual verification (all 10 checks) against `make dev` + `npm run dev` on a mobile-sized viewport.
- [ ] T018 Read-only & tone review: confirm the plan view calls **only** `ListPlanEntries` (no `PlanService` mutations) and exposes no add/edit/move/delete affordances (FR-013, SC-006), and that new user-facing copy is consistent with Principle IV.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup. BLOCKS all user stories (every story uses `planView.ts`).
- **User Stories (Phase 3–5)**: All depend on Foundational. US2 and US3 build on the `PlanPage`/`PlanEntryRow` files created in US1, so in practice run in priority order (P1 → P2 → P3).
- **Polish (Phase 6)**: Depends on the desired user stories being complete.

### User Story Dependencies

- **US1 (P1)**: Depends only on Foundational. Delivers the MVP.
- **US2 (P2)**: Extends US1's `PlanPage`/`PlanEntryRow`. Independently testable but shares those files (not parallel with US1).
- **US3 (P3)**: Extends US1's `PlanPage`. Independently testable; shares that file.

### Within Each User Story

- Test task written first and failing → implementation.
- Pure helpers (Foundational) before components; component (`PlanEntryRow`) before/with page (`PlanPage`); page before route wiring.

### Parallel Opportunities

- T002 (messages) can run parallel to T001 (proxy).
- T003 (helper tests) can be written in parallel with Setup.
- T006 (`PlanEntryRow`) and T009 (`AppHeader`) are different files → parallelizable; both independent of each other.
- Across stories: because US2/US3 edit the same `PlanPage.tsx`/`PlanEntryRow.tsx`, they are **not** safely parallel; sequence them.
- T015 and T016 (test + build) can run in parallel.

---

## Parallel Example: User Story 1

```bash
# After T007 (PlanPage) exists, these touch different files:
Task: "T006 Create PlanEntryRow component in services/twig-web/src/components/PlanEntryRow.tsx"
Task: "T009 Add Tasks/Plan nav in services/twig-web/src/components/AppHeader.tsx"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup (proxy + messages).
2. Phase 2: Foundational (`planView.ts` + tests).
3. Phase 3: User Story 1 (PlanPage + PlanEntryRow + route + nav).
4. **STOP and VALIDATE**: today's plan is viewable from the nav.
5. Demo (MVP).

### Incremental Delivery

1. Setup + Foundational → ready.
2. US1 → today's plan view (MVP) → demo.
3. US2 → richer entries + task navigation → demo.
4. US3 → day stepping → demo.

---

## Notes

- [P] = different files, no dependencies on incomplete tasks.
- No backend/proto/codegen work — `plan.v1` stubs already exist in `src/gen/plan/v1/`; do **not** run `npm run gen`.
- Verify each story's tests fail before implementing.
- Commit after each task or logical group.
