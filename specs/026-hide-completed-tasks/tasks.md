---
description: "Task list for Hide Completed Tasks"
---

# Tasks: Hide Completed Tasks

**Input**: Design documents from `/specs/026-hide-completed-tasks/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ui-contract.md ✅, quickstart.md ✅

**Tests**: INCLUDED. The plan's Definition of Done and the existing codebase convention
(`src/lib/tree.test.ts`, `src/lib/updatePayload.test.ts`) require unit tests for pure
`src/lib/` helpers plus a page interaction test. Pure-logic tests are written before their
implementation; the page interaction test accompanies the page wiring.

**Organization**: Grouped by user story. This feature is intentionally layered — US2 builds
on US1, and US3 builds on US2 — so the stories ship as incremental MVP slices rather than
fully parallel tracks (see Dependencies).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 / US2 / US3 (maps to spec.md user stories); omitted for Setup/Foundational/Polish
- All paths are relative to repo root; this feature touches only `services/twig-web/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm a clean starting baseline in the existing web app (no new dependencies needed).

- [X] T001 Establish clean baseline: from `services/twig-web/`, run `npm test` and `npm run build` and confirm both pass before making changes.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The pure tree-pruning helper that every user story depends on.

**⚠️ CRITICAL**: No user story work can begin until `filterTree` exists and is tested.

- [X] T002 Write failing unit tests for `filterTree` in `services/twig-web/src/lib/filterTree.test.ts`, covering every row of the contract in `contracts/ui-contract.md`: identity passthrough when `showCompleted` is true; drop completed leaf; keep incomplete leaf; keep incomplete parent while removing its completed children (order preserved); prune fully-completed subtree; keep a completed ancestor that still has an incomplete descendant; purity (no mutation, `depth` preserved).
- [X] T003 Implement `filterTree(nodes: TaskNode[], showCompleted: boolean): TaskNode[]` in `services/twig-web/src/lib/filterTree.ts` using the bottom-up "keep if incomplete OR has a kept descendant" rule (research Decision 2), making T002 pass. Import `TaskNode` from `./tree`.

**Checkpoint**: `npm test -- filterTree` is green. User stories can now begin.

---

## Phase 3: User Story 1 - Completed tasks hidden by default (Priority: P1) 🎯 MVP

**Goal**: On opening the task list, completed tasks are not shown; the view focuses on outstanding work, and an "all done" list is never mistaken for an empty one.

**Independent Test**: With a mix of completed and incomplete tasks, load the page → only incomplete tasks render. Complete a visible task → it disappears. Complete the last task → the playful "all done, completed hidden" message shows (not the generic empty state).

### Implementation for User Story 1

- [X] T004 [US1] In `services/twig-web/src/pages/TaskTreePage.tsx`, derive a filtered tree by calling `filterTree(tree, false)` (completed hidden by default) and render the filtered tree in the existing `<ul>` instead of the raw `tree`. Completion mutations already invalidate the `listTasks` query, so the filtered view updates automatically (FR-001, FR-003, FR-004, FR-006).
- [X] T005 [P] [US1] Add a playful "all tasks done — completed hidden" message to `services/twig-web/src/theme/messages.ts` (warm/playful tone per Constitution Principle IV), e.g. `allCompletedHidden`.
- [X] T006 [US1] In `services/twig-web/src/pages/TaskTreePage.tsx`, distinguish the all-done state from a truly empty list: when `tasks.length > 0` but the filtered tree is empty, render the new `allCompletedHidden` message plus a way to reveal completed tasks, rather than `EmptyState` (FR-008). (Depends on T004, T005.)
- [X] T007 [US1] Add an interaction test in `services/twig-web/src/pages/TaskTreePage.test.tsx`: given completed + incomplete tasks, only incomplete render by default; given all tasks completed, the `allCompletedHidden` message renders and `EmptyState` does not.

**Checkpoint**: Completed tasks are hidden by default and the all-done state is unambiguous — shippable MVP.

---

## Phase 4: User Story 2 - Reveal completed tasks on demand (Priority: P2)

**Goal**: A single control toggles the list between hiding and showing completed tasks; shown completed tasks are clearly marked done.

**Independent Test**: With completed tasks hidden, activate the control → completed tasks appear (marked done); deactivate → they hide again. Reopening a completed task while shown keeps it visible as incomplete.

### Implementation for User Story 2

- [X] T008 [US2] In `services/twig-web/src/pages/TaskTreePage.tsx`, introduce a `showCompleted` boolean state (initial value `false`) and pass it to `filterTree(tree, showCompleted)`, replacing the hardcoded `false` from T004.
- [X] T009 [US2] In the existing header control row of `services/twig-web/src/pages/TaskTreePage.tsx` (alongside the "Tasks" title / "+ Add task" button), add a Show/Hide completed toggle using the shared `Button` component (`variant="secondary"`) and theme tokens; clicking it flips `showCompleted`. The toggle is in the always-rendered header so it stays reachable when no tasks are visible (FR-002, FR-005, FR-009). Wire the all-done state's "reveal" action (T006) to this same toggle.
- [X] T010 [US2] Extend `services/twig-web/src/pages/TaskTreePage.test.tsx`: activating the toggle reveals completed tasks (rendered with the done styling); deactivating hides them again.

**Checkpoint**: Users can reveal/hide completed tasks on demand; US1 behavior still holds.

---

## Phase 5: User Story 3 - Preference is remembered (Priority: P3)

**Goal**: The user's last show/hide choice survives page reloads and return visits in the same browser; first-time users default to hidden.

**Independent Test**: Activate show-completed, remount/reload → still shown. Clear the stored preference → default is hidden.

### Implementation for User Story 3

- [X] T011 [P] [US3] Create `services/twig-web/src/lib/showCompletedPref.ts` exporting `readShowCompleted(): boolean` (reads `localStorage` key `twig-show-completed`, default `false`, never throws) and `writeShowCompleted(value: boolean): void` (best-effort, swallows storage errors), mirroring the try/catch style of `readExpandedIds`/`writeExpandedIds`.
- [X] T012 [P] [US3] Write unit tests in `services/twig-web/src/lib/showCompletedPref.test.ts`: missing/unreadable key → `false`; round-trip `writeShowCompleted(true)` then `readShowCompleted()` → `true`; any non-`"true"` stored value → `false`; storage exceptions are caught.
- [X] T013 [US3] In `services/twig-web/src/pages/TaskTreePage.tsx`, initialize `showCompleted` from `readShowCompleted()` and call `writeShowCompleted(next)` whenever the toggle changes (FR-007). (Depends on T008, T009, T011.)
- [X] T014 [US3] Extend `services/twig-web/src/pages/TaskTreePage.test.tsx`: with a stored preference of shown, the page renders completed tasks after (re)mount; with no stored preference, completed tasks are hidden.

**Checkpoint**: All three user stories are functional; the preference persists across reloads.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verification and consistency review across the feature.

- [ ] T015 Run the `quickstart.md` manual smoke test (steps 1–5) against `make dev` + `npm run dev`, confirming FR-001, FR-002, FR-005, FR-007, FR-008, FR-009 by observation.
- [X] T016 From `services/twig-web/`, run `npm test` and `npm run build`; ensure all suites pass and there are no TypeScript/type-check errors.
- [X] T017 [P] Review the new `allCompletedHidden` copy for playful tone (Constitution Principle IV) and the toggle's styling for theme/interaction consistency with existing controls (Constitution Principle III).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup — **BLOCKS all user stories** (everything needs `filterTree`).
- **User Stories (Phases 3–5)**: Each depends on Foundational. Because this feature is layered, the stories are also sequential:
  - **US1 (P1)**: after Foundational.
  - **US2 (P2)**: after US1 (replaces US1's hardcoded `false` with toggle state; reuses US1's reveal action).
  - **US3 (P3)**: after US2 (persists the toggle introduced in US2).
- **Polish (Phase 6)**: after all desired stories are complete.

### Within Each Story

- Pure-logic tests (T002) before their implementation (T003).
- US1: T004 → T006 (same file, T006 also needs T005); T005 is parallel to T004; T007 verifies.
- US2: T008 → T009 (same file) → T010 verifies.
- US3: T011 & T012 in parallel; T013 needs T008/T009/T011; T014 verifies.

### Parallel Opportunities

- **T002** (filterTree tests) can be written while reviewing the contract; **T003** follows.
- **T005** (messages.ts) runs in parallel with **T004** (different files).
- **T011** (showCompletedPref.ts) and **T012** (its test) run in parallel with each other.
- **T017** (tone/consistency review) can run alongside other polish checks.
- Tasks touching `TaskTreePage.tsx` (T004, T006, T008, T009, T013) are **not** parallel — same file, ordered.

---

## Parallel Example: User Story 3

```bash
# T011 and T012 touch different files and can be developed together:
Task: "Create showCompletedPref.ts (read/write localStorage, default false)"
Task: "Write showCompletedPref.test.ts (default, round-trip, robustness)"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 (Setup) → Phase 2 (Foundational: `filterTree` + tests).
2. Phase 3 (US1): filter applied with completed hidden by default + all-done empty state.
3. **STOP and VALIDATE**: completed tasks hidden, all-done state clear. This alone delivers the core value and is demoable.

### Incremental Delivery

1. Setup + Foundational → `filterTree` ready and tested.
2. US1 → hidden-by-default MVP → demo.
3. US2 → toggle to reveal/hide → demo.
4. US3 → preference persists across reloads → demo.

Each increment is independently testable and adds value without breaking the previous one.

---

## Notes

- Entire footprint is in `services/twig-web/`; no backend, proto, DB, or migration changes (contracts/ui-contract.md records the API as unchanged).
- [P] = different files, no dependency on an incomplete task.
- Verify pure-logic tests fail before implementing them.
- Commit after each task or logical group.
- Constitution gates to satisfy before merge: Principle I (no new abstractions — only `filterTree` + `showCompletedPref`), Principle II (no API change, recorded), Principle III (toggle reuses `Button`/theme), Principle IV (new `allCompletedHidden` copy is playful).
