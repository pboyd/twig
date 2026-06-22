---
description: "Task list for feature: Goals in the Web App"
---

# Tasks: Goals in the Web App

**Input**: Design documents from `/specs/058-add-web-goals/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/web-goals.md, quickstart.md

**Tests**: Included. The web module has co-located tests for every component/lib, and plan.md (research Decision 11) + quickstart require `npm test` green. Tests are written alongside each task's implementation.

**Organization**: Tasks are grouped by user story. All work is under `services/twig-web/`. The backend is NOT modified; generated stubs in `src/gen/goal/v1/` are reused as-is.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1–US4 maps to the spec's user stories
- All paths are relative to `services/twig-web/` unless noted.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Make the goal service reachable in dev.

- [ ] T001 Add `"/goal.v1": "http://localhost:8080"` to the `server.proxy` map in `services/twig-web/vite.config.ts` (keeps goal RPCs same-origin for the SameSite=Strict cookie).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared helpers and copy used by multiple stories.

**⚠️ CRITICAL**: Complete before user-story phases.

- [ ] T002 [P] Create `src/lib/formatTimestamp.ts` (human-readable absolute date/time from a protobuf `Timestamp`, plus a date-only formatter for goal due dates, via `Intl.DateTimeFormat`) and `src/lib/formatTimestamp.test.ts` covering fixed inputs, undefined/empty, and date-only output.
- [ ] T003 [P] Add goal + status-update user-facing copy to `src/theme/messages.ts` in the warm/playful tone (empty goal list, all-hidden hint, no-status-updates-yet, empty-status-validation, status saved/edited/deleted toasts, delete-update confirm, goal-not-found, generic goal/status load error).

**Checkpoint**: Helpers and copy ready — user-story work can begin.

---

## Phase 3: User Story 1 - Browse goals in the web app (Priority: P1) 🎯 MVP

**Goal**: A signed-in user can reach a Goals area, see goals grouped by state with Completed/Archived hidden by default (toggle to reveal), and open a goal to read its name, description (markdown), due date, state, and read-only associated tasks.

**Independent Test**: With existing goals, open `/goals`: groups show Committed then Incubating, hidden toggle reveals/hides Completed+Archived and persists across reload; opening a goal shows its fields and linked tasks. No status-update interaction needed.

### Implementation for User Story 1

- [ ] T004 [P] [US1] Create `src/lib/goalGroups.ts` — pure `goalGroups(goals, showHidden)` returning ordered, non-empty `GoalGroup[]` (Committed, Incubating, then Completed, Archived only when `showHidden`; preserve server position order) — and `src/lib/goalGroups.test.ts`.
- [ ] T005 [P] [US1] Create `src/lib/showHiddenGoalsPref.ts` (`readShowHiddenGoals`/`writeShowHiddenGoals` backed by `localStorage["twig-show-hidden-goals"]`, exception-guarded, default false) and `src/lib/showHiddenGoalsPref.test.ts`, mirroring `showCompletedPref.ts`.
- [ ] T006 [P] [US1] Create `src/components/GoalListItem.tsx` (props `{ goal, onOpen }`; name via `<Markdown mode="inline">`, optional due via `formatTimestamp`, no mutation controls) and `src/components/GoalListItem.test.tsx`.
- [ ] T007 [US1] Create `src/pages/GoalsPage.tsx` — `useQuery(listGoals, {})`, group via `goalGroups`, render group headings + `GoalListItem` rows, show-hidden toggle button (persist via T005), loading `Spinner`, `ErrorBanner` on failure, empty state + "Show all" affordance when only hidden goals exist — and `src/pages/GoalsPage.test.tsx`. (depends on T004, T005, T006)
- [ ] T008 [US1] Create `src/pages/GoalDetailPage.tsx` skeleton — `useParams` id, `useQuery(getGoal, { id })`, read-only header (name inline-md, description block-md, due via `formatTimestamp`, state label), back button to `/goals`, loading `Spinner`, `NotFound` → friendly not-found with link back, generic `ErrorBanner` — and `src/pages/GoalDetailPage.test.tsx`. (depends on T002, T003)
- [ ] T009 [US1] In `src/pages/GoalDetailPage.tsx`, add a read-only "Associated tasks" section: `useQuery(listTasks, {})` filtered to `goalId === id`, render names (inline markdown) as links to `/tasks/:id`, hide section when none; extend `GoalDetailPage.test.tsx`. (depends on T008)
- [ ] T010 [US1] Register routes `"/goals"` → `GoalsPage` and `"/goals/:id"` → `GoalDetailPage` in `src/App.tsx`. (depends on T007, T008)
- [ ] T011 [P] [US1] Add a "Goals" `NavLink` (to `/goals`, using existing `navLinkClass`) to `src/components/AppHeader.tsx` and update `src/components/AppHeader.test.tsx`.
- [ ] T012 [P] [US1] Add a "Goals" entry to the mobile menu in `src/components/HeaderMenu.tsx`.

**Checkpoint**: Goals are browsable end-to-end (nav → grouped list → detail with fields + linked tasks). MVP shippable.

---

## Phase 4: User Story 2 - Read a goal's status updates (Priority: P1)

**Goal**: On a goal's detail, the user sees the latest status update (timestamp + rendered markdown) and can browse the full newest-first history; long entries are fully readable; goals with none show a friendly empty state.

**Independent Test**: Open a goal with one or more updates: latest shows with timestamp + rendered markdown; history lists all newest-first; a long update wraps/scrolls without truncation; a goal with no updates shows the empty message.

### Implementation for User Story 2

- [ ] T013 [P] [US2] Create `src/components/StatusUpdateList.tsx` — props `{ updates, onEdit?, onDelete? }`; render newest-first, each item with `formatTimestamp(createdAt)` + body via `<Markdown mode="block">`, wrapping/scrolling with no truncation (edit/delete affordances rendered only when handlers passed) — and `src/components/StatusUpdateList.test.tsx` (read-only rendering + ordering).
- [ ] T014 [US2] In `src/pages/GoalDetailPage.tsx`, add `useQuery(listGoalStatusUpdates, { goalId: id })`; show `updates[0]` as "Latest status" (timestamp + markdown), render history via `StatusUpdateList`, and the "no status updates yet" empty state when the list is empty; extend `GoalDetailPage.test.tsx`. (depends on T008, T013)

**Checkpoint**: Goals + reading status updates both work independently.

---

## Phase 5: User Story 3 - Record a new status update (Priority: P2)

**Goal**: From a goal's detail, the user writes and saves a new status update; it is auto-timestamped, becomes the latest status, and empty/whitespace text is rejected.

**Independent Test**: Add an update → appears as latest with current timestamp at top of history; submitting empty/whitespace is rejected with a clear message and no record created.

### Implementation for User Story 3

- [ ] T015 [P] [US3] Create `src/components/StatusUpdateForm.tsx` — props `{ initialBody?, submitLabel, loading?, onSubmit, onCancel }`; single textarea + Save/Cancel; trims and blocks empty/whitespace submit with the validation message from `messages.ts` (no RPC sent) — and `src/components/StatusUpdateForm.test.tsx`.
- [ ] T016 [US3] In `src/pages/GoalDetailPage.tsx`, add an "Add status update" affordance using `StatusUpdateForm`; wire `useMutation(addGoalStatusUpdate)`, invalidate `listGoalStatusUpdates({ goalId })` + `getGoal({ id })` + `listGoals({})` via `createConnectQueryKey`, show success toast, handle error via `ErrorBanner`; extend `GoalDetailPage.test.tsx`. (depends on T014, T015)

**Checkpoint**: Users can read and record updates.

---

## Phase 6: User Story 4 - Edit or delete an existing status update (Priority: P3)

**Goal**: From the history, the user edits an update's text (timestamp preserved) or deletes it (with confirmation); empty edits are rejected; deleting the only update returns to the empty state.

**Independent Test**: Edit an update → new text renders, timestamp unchanged; empty edit rejected with original retained; delete (confirm) removes only that update; deleting the last one shows "no status updates yet".

### Implementation for User Story 4

- [ ] T017 [US4] Wire edit in `src/pages/GoalDetailPage.tsx`: pass an `onEdit` handler to `StatusUpdateList` that opens `StatusUpdateForm` seeded with the update's body; `useMutation(updateGoalStatusUpdate)` + same invalidation set; success toast; extend `GoalDetailPage.test.tsx` (incl. empty-edit rejection). (depends on T016)
- [ ] T018 [US4] Wire delete in `src/pages/GoalDetailPage.tsx`: pass an `onDelete` handler to `StatusUpdateList` that confirms (warm/measured copy), `useMutation(deleteGoalStatusUpdate)` + same invalidation set, success toast, and falls back to the empty state when the last update is removed; extend `GoalDetailPage.test.tsx`. (depends on T014; integrates with T016)

**Checkpoint**: Full status-update management (read/write/edit/delete) complete.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T019 [P] Review all new copy added in T003 (and any inline strings) for Principle IV tone compliance; confirm no raw/dry system text leaked into pages or components.
- [ ] T020 Run `cd services/twig-web && npm test` (all green) and `npm run build` (tsc + vite type-check clean).
- [ ] T021 Execute `specs/058-add-web-goals/quickstart.md` end-to-end against `make dev` + `npm run dev`, including the cross-interface consistency check (web ↔ TUI) and error/not-found paths.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: After Setup. Blocks user stories (formatTimestamp + messages are shared).
- **US1 (Phase 3)**: After Foundational. The gateway MVP — creates the pages/routes/nav the other stories build on.
- **US2 (Phase 4)**: After US1 (extends `GoalDetailPage`; uses `StatusUpdateList`).
- **US3 (Phase 5)**: After US2 (adds the form + add-mutation into the detail page that renders status).
- **US4 (Phase 6)**: After US3 (edit reuses the add flow's form/mutation pattern; delete extends the list rendered in US2).
- **Polish (Phase 7)**: After all desired stories.

### User Story Dependencies

- US1 (P1): independent once Foundational is done.
- US2 (P1): builds on US1's `GoalDetailPage`.
- US3 (P2): builds on US2's status rendering.
- US4 (P3): builds on US3's form/mutation and US2's list.

> Note: Because all status-update UI lives inside the single `GoalDetailPage`, US2→US3→US4 are intentionally sequential (same file) rather than parallel. They remain independently *testable* increments.

### Within Each Story

- `[P]` tasks (different files) first/parallel; page-integration tasks depend on their components.

### Parallel Opportunities

- **Phase 2**: T002, T003 in parallel.
- **US1**: T004, T005, T006 in parallel; T011, T012 in parallel (and parallel with T007–T009 since different files), then T010 after T007+T008.
- **US2**: T013 in parallel with finishing US1 component work; T014 after.
- **US3**: T015 can start in parallel; T016 after T014+T015.
- **Polish**: T019 in parallel with code freeze; T020/T021 last.

---

## Parallel Example: User Story 1

```bash
# Pure libs + presentational component together (different files):
Task: "Create src/lib/goalGroups.ts + test"
Task: "Create src/lib/showHiddenGoalsPref.ts + test"
Task: "Create src/components/GoalListItem.tsx + test"

# Nav edits together (different files), independent of the pages:
Task: "Add Goals NavLink to src/components/AppHeader.tsx + test"
Task: "Add Goals entry to src/components/HeaderMenu.tsx"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 Setup (proxy) → Phase 2 Foundational (formatTimestamp, messages).
2. Phase 3 US1 → **STOP & VALIDATE**: browse goals end-to-end (nav, grouped list, hidden toggle, detail fields, linked tasks).
3. Shippable MVP — viewing goals on the web.

### Incremental Delivery

1. US1 (browse) → demo.
2. US2 (read status) → demo.
3. US3 (write status) → demo.
4. US4 (edit/delete) → demo.
5. Polish (tone review, build/test, quickstart).

---

## Notes

- All RPCs use existing generated stubs in `src/gen/goal/v1/`; no `npm run gen`, no proto/server changes.
- Mutation invalidation set is identical across add/edit/delete (see data-model.md freshness map): `listGoalStatusUpdates({ goalId })`, `getGoal({ id })`, `listGoals({})`.
- Status display is driven by `listGoalStatusUpdates` only (latest = `updates[0]`); `goal.latestStatusUpdate` is not used for rendering.
- Goals are read-only on web (FR-019): do not add create/edit/delete/state/reorder controls. Associated tasks are display-only (FR-008).
- Commit after each task or logical group.
