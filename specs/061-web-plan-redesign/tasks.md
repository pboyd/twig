# Tasks: Web Plan Tab Day-Planner Redesign

**Input**: Design documents from `/specs/061-web-plan-redesign/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ui-contract.md, quickstart.md

**Tests**: Included — SC-003 requires the existing and updated test suite to pass, and the repo convention is colocated Vitest/Testing-Library tests for every component and lib.

**Organization**: Tasks are grouped by user story. All paths are relative to the repo root; every source change lives under `services/twig-web/`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Phase 1: Setup

**Purpose**: Confirm a green baseline before touching anything — this is a redesign of a working page.

- [X] T001 Verify baseline: run `npm install`, `npm test -- --run`, and `npm run build` in `services/twig-web/` and confirm all pass before changes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Pure slot/window math used by the timeline (US1) and the now-indicator (US3).

**⚠️ CRITICAL**: US1 and US3 cannot start until this phase is complete.

- [X] T002 Add timeline geometry helpers to `services/twig-web/src/lib/planView.ts`: `SLOT_MINUTES` (15), `SLOT_PX` (44), `snapDown15`, `snapUp15`, `computeWindow` (auto-fit to snapped entry spans, hour-aligned, unioned with 480–1020 default), `slotIndex`, `slotCount`, `hourLabels` (reusing `formatMinute`), per data-model.md
- [X] T003 [P] Unit-test the new helpers in `services/twig-web/src/lib/planView.test.ts`: default window on empty day, hour alignment, entries outside the default extending the window, snapping of odd start/duration values, slot indices, hour labels

**Checkpoint**: `npm test -- --run` green — timeline stories can begin.

---

## Phase 3: User Story 1 - See the day as a planner, not a list (Priority: P1) 🎯 MVP

**Goal**: Timed entries render as blocks on an hour-ruled timeline (position = start time, height ∝ duration, gaps visible); untimed entries move to a checklist section above the timeline.

**Independent Test**: Plan a day with 8:00–8:30 and 11:00–12:00 entries (quickstart.md seeding); the Plan tab shows an hour-labeled grid where the 60-minute block spans twice the rows of the 30-minute block with empty ruled rows between, and the untimed section sits above the timeline.

### Implementation for User Story 1

- [X] T004 [P] [US1] Create `services/twig-web/src/components/PlanTimeline.tsx` per contracts/ui-contract.md: CSS grid with one 44px row per 15-minute slot, hour-labeled left gutter, ruled hour boundaries, entry blocks spanning their snapped slots (task names link to `/tasks/:id`, events italic plain text), completed names struck through, existing check-button complete action and trash remove action carried over for now (US2 replaces the control), per-entry inline error text, dark-mode variants for all styles
- [X] T005 [P] [US1] Component tests in `services/twig-web/src/components/PlanTimeline.test.tsx`: block row-span proportional to duration, gap rows empty, hour labels present, event vs task rendering, remove/complete callbacks fire, per-entry error shown
- [X] T006 [US1] Rework `services/twig-web/src/pages/PlanPage.tsx`: render untimed section (heading "Untimed") above `PlanTimeline`; pass timed entries, handlers, pending ids, and error map; keep day navigation, `Spinner`, `ErrorBanner`, and `messages.planEmpty` states unchanged (depends on T004)
- [X] T007 [US1] Simplify `services/twig-web/src/components/PlanEntryRow.tsx` into the untimed-row renderer: drop the `timeLabel` branch (timed rendering now lives in `PlanTimeline`), keep name link, complete/remove actions, and error display
- [X] T008 [US1] Update `services/twig-web/src/pages/PlanPage.test.tsx`: untimed section precedes timeline in the DOM, timed entries render inside the timeline, empty/loading/error states and day navigation unchanged

**Checkpoint**: Plan tab shows a real day-planner. Deliverable MVP.

---

## Phase 4: User Story 2 - Consistent task controls across tabs (Priority: P2)

**Goal**: One shared circle done-control across Tasks and Plan tabs, toggling both ways (complete ⇄ re-open), with the Plan tab's separate completed badge removed.

**Independent Test**: Place an incomplete task on today's plan; its done control is pixel-identical to the Tasks tab's circle. Tapping completes it (filled circle + strikethrough); tapping again re-opens it. Blocked complete/re-open shows the existing playful messages.

### Implementation for User Story 2

- [X] T009 [P] [US2] Extract the circle toggle markup from `TreeRow.tsx` into `services/twig-web/src/components/CompletionToggle.tsx` with props `{ completed, disabled?, onToggle }` and state-based `aria-label`s, per contracts/ui-contract.md
- [X] T010 [P] [US2] Component tests in `services/twig-web/src/components/CompletionToggle.test.tsx`: empty circle when incomplete, filled + check when complete, `onToggle` fires, disabled state blocks interaction
- [X] T011 [US2] Refactor `services/twig-web/src/components/TreeRow.tsx` to render `CompletionToggle` (zero visual change; existing `TaskTreePage`/`TreeRow` tests must stay green) (depends on T009)
- [X] T012 [US2] Wire two-way toggling in `services/twig-web/src/pages/PlanPage.tsx`: add `uncompleteTask` mutation; toggle handler dispatches complete/uncomplete on `entry.completed`; map `FailedPrecondition` to `messages.completeBlockedBySubtasks` / `messages.reopenBlockedByParent`; invalidate plan + tasks queries (depends on T009)
- [X] T013 [US2] Replace the check button and green completed badge in `services/twig-web/src/components/PlanTimeline.tsx` with `CompletionToggle` on task-entry blocks (events keep no toggle) (depends on T009, T012)
- [X] T014 [US2] Replace the check button in `services/twig-web/src/components/PlanEntryRow.tsx` (untimed rows) with `CompletionToggle` (depends on T009, T012)
- [X] T015 [US2] Update tests: `services/twig-web/src/pages/PlanPage.test.tsx` (toggle completes and re-opens, blocked-reopen message, badge gone) and `services/twig-web/src/components/PlanTimeline.test.tsx` (block renders `CompletionToggle` states)

**Checkpoint**: Controls indistinguishable across tabs; both tabs' suites green.

---

## Phase 5: User Story 3 - Orient within today (Priority: P3)

**Goal**: An accent-colored current-time line on today's timeline only.

**Independent Test**: With fake timers set inside the window on today's date, the timeline shows `data-testid="now-indicator"` at the correct offset; on another day (or now outside the window) it is absent.

### Implementation for User Story 3

- [X] T016 [US3] Add the now-indicator to `services/twig-web/src/components/PlanTimeline.tsx`: absolutely positioned accent line at `(nowMinute − windowStart) / 15 × 44px`, rendered only when `day` is today and now is within the window; refresh via 60-second interval effect (cleaned up on unmount) (depends on US1 T004)
- [X] T017 [US3] Indicator tests in `services/twig-web/src/components/PlanTimeline.test.tsx` with `vi.useFakeTimers`: correct offset for a known time, absent on non-today days, absent when now is outside the window, advances after the interval ticks

**Checkpoint**: All three stories functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T018 Manual verification per `specs/061-web-plan-redesign/quickstart.md`: seed a day, check light + dark themes at a 390px viewport, confirm every entry action is tappable (44px targets) and no new copy bypassed `src/theme/messages.ts`
- [X] T019 Full verification: `npm test -- --run` and `npm run build` in `services/twig-web/` both clean
- [X] T020 [P] Update the frontend key-paths list in `AGENTS.md` (symlinked as `CLAUDE.md`): add `CompletionToggle` and `PlanTimeline` to the `src/components/` line

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none — start immediately
- **Foundational (Phase 2)**: after Setup — blocks US1 and US3 (US2's `CompletionToggle` work does not need it, but runs after US1 by priority)
- **User Stories (Phase 3–5)**: US1 after Phase 2; US2 independent of Phase 2 but touches `PlanTimeline`/`PlanPage`, so run after US1 (T013 depends on T004); US3 after US1 (extends `PlanTimeline`)
- **Polish (Phase 6)**: after all desired stories

### User Story Dependencies

- **US1 (P1)**: depends only on Foundational — the MVP
- **US2 (P2)**: T009–T011 (toggle extraction + TreeRow refactor) are independent of US1 and could start any time; T012–T015 modify US1's files
- **US3 (P3)**: extends `PlanTimeline` from US1

### Parallel Opportunities

- T002 ∥ nothing (single file), then T003 alongside T004/T005
- T004 ∥ T005 (component vs test file), and both ∥ T009/T010 (US2's toggle extraction)
- T009 ∥ T010
- T020 ∥ T018/T019

## Parallel Example: after Phase 2

```bash
# Three independent files at once:
Task: "Create PlanTimeline component in services/twig-web/src/components/PlanTimeline.tsx"
Task: "Write PlanTimeline tests in services/twig-web/src/components/PlanTimeline.test.tsx"
Task: "Extract CompletionToggle in services/twig-web/src/components/CompletionToggle.tsx"
```

## Implementation Strategy

**MVP first**: Phases 1–3 deliver the day-planner timeline (the core complaint) with existing controls intact — ship/demo there if desired. US2 then unifies the controls, US3 adds the now line. Each checkpoint leaves the suite green and the page fully functional, so you can stop after any phase.
