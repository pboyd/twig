---
description: "Task list for Hide Completed Unscheduled Plan Entries"
---

# Tasks: Hide Completed Unscheduled Plan Entries

**Input**: Design documents from `/specs/044-hide-completed-unscheduled/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included. The project unit-tests every TUI/web change, and quickstart.md enumerates the coverage to add.

**Organization**: Tasks are grouped by user story. This is a brownfield, client-only change — no proto/server/DB work (the server already populates `PlanEntry.completed`).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1, US2, US3 from spec.md
- File paths are repo-root relative

## Path Conventions

- TUI/CLI (root module): `internal/tui/`, `internal/cli/`
- Web SPA: `services/twig-web/src/`
- No changes to `api/proto/`, `api/gen/`, or `services/twig/` (server)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish a clean baseline before any change.

- [ ] T001 Confirm a green baseline: run `go test ./...` at repo root and `npm test` in `services/twig-web/`, noting current pass state before edits.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared TUI mechanism used by both US1 (TUI load-time filtering) and US2 (deferred hide). The web surface (US1) does not depend on this phase.

**⚠️ CRITICAL**: T003–T004 block the TUI portions of US1 and all of US2.

- [ ] T002 Add `pendingComplete *int32` field (entry id of a just-completed untimed entry to retain while highlighted) to the `planState` struct in `internal/tui/model.go`.
- [ ] T003 [P] Write a failing unit test for a `displayedPlanEntries(entries []*planv1.PlanEntry, pendingComplete *int32) []*planv1.PlanEntry` helper in `internal/tui/plan_update_test.go`: excludes untimed entries with `Completed==true`, keeps the one whose id equals `*pendingComplete`, never drops timed entries (even completed) or events, keeps incomplete untimed entries, and is order-preserving and idempotent.
- [ ] T004 Implement the `displayedPlanEntries` helper in `internal/tui/plan_update.go` to satisfy T003 (predicate: drop entry iff `StartMinute == nil && Completed && (pendingComplete == nil || id != *pendingComplete)`).

**Checkpoint**: Tested pure filter helper + state field exist; user-story wiring can begin.

---

## Phase 3: User Story 1 - Completing an unscheduled task clears it from the plan (Priority: P1) 🎯 MVP

**Goal**: A completed unscheduled (untimed) entry does not appear in the plan view, in both the TUI (on load/render) and the web planner. Timed entries and events are unaffected.

**Independent Test**: Load a day whose plan contains a completed untimed task entry and confirm it is absent from the view, while a completed timed entry on the same day remains visible — verifiable separately in the TUI and the web app.

### Tests for User Story 1

- [ ] T005 [P] [US1] Write failing tests in `services/twig-web/src/lib/planView.test.ts`: `groupPlan` excludes completed entries from `untimed`, keeps completed entries in `timed`, and reports `isEmpty === true` when the only untimed entries are completed and there are no timed entries.
- [ ] T006 [P] [US1] Write a failing test in `internal/tui/plan_update_test.go`: `handlePlanEntriesMsg` stores a displayed list that omits completed untimed entries (with `pendingComplete == nil`) while retaining timed/completed and event entries.

### Implementation for User Story 1

- [ ] T007 [P] [US1] In `services/twig-web/src/lib/planView.ts`, exclude completed entries from the untimed group (`entries.filter((e) => !e.timed && !e.completed)`); leave the timed grouping unchanged so `isEmpty` reflects the filtered untimed list.
- [ ] T008 [US1] In `internal/tui/plan_update.go` `handlePlanEntriesMsg`, set `m.plan.entries = displayedPlanEntries(msg.entries, m.plan.pendingComplete)` before clamping the cursor / applying `highlightID`, so the loaded plan omits completed untimed entries.

**Checkpoint**: Completed untimed entries are gone from both surfaces on load. (In the TUI, completing from within the planner hides immediately until US2 adds deferral.) MVP is functional.

---

## Phase 4: User Story 2 - Completing an unscheduled task in the TUI keeps it in place until you move on (Priority: P1)

**Goal**: In the TUI, completing the highlighted untimed entry leaves it visible (crossed out) and highlighted; it is removed only on the next render where it is no longer the active highlight (cursor move, day change, tab switch, go-to-task) — matching the tasks-tab interaction.

**Independent Test**: Highlight an untimed entry, complete it, confirm it stays crossed-out and highlighted; then move the highlight (or switch tab/day) and confirm it disappears.

### Tests for User Story 2

- [ ] T009 [US2] Write failing tests in `internal/tui/update_test.go` (and/or `plan_update_test.go`): (a) completing the highlighted untimed entry sets `m.plan.pendingComplete` to its id and the entry remains in the displayed/rendered plan after reload (struck via existing `applyCompletion`); (b) cursor Up/Down, prev/next/today day change, tab switch, and go-to-task each clear `pendingComplete` and drop the entry from the displayed plan; (c) completing a *timed* entry never sets `pendingComplete`.

### Implementation for User Story 2

- [ ] T010 [US2] In `internal/tui/update.go` Complete handler (the `m.keys.Complete` case in the planning branch, ~line 667), when `complete == true` and the highlighted entry is untimed (`entry.StartMinute == nil`) and task-linked, set `m.plan.pendingComplete = &entry.Id` before issuing `completePlanTaskCmd`; do not set it for timed entries.
- [ ] T011 [US2] In `internal/tui/update.go`, clear `m.plan.pendingComplete = nil` in the navigation handlers — cursor `Up`/`Down`, `PlanPrevDay`/`PlanNextDay`/`PlanToday`, the tab-switch handler, and `PlanGoToTask` — and re-derive the displayed list with `displayedPlanEntries(m.plan.entries, nil)` (clamping the cursor) so the retained entry is removed once the highlight leaves it.

**Checkpoint**: TUI completion matches the tasks-tab deferred-hide UX. US1 + US2 both pass.

---

## Phase 5: User Story 3 - Completed-then-reopened task returns to the plan (Priority: P2)

**Goal**: Reopening (uncompleting) a task restores its untimed entry to the plan view; reopening a still-highlighted pending entry shows it normally again. Web behavior is emergent from the US1 filter.

**Independent Test**: Complete an untimed entry so it leaves the plan, reopen the task, and confirm the entry reappears (TUI and web). In the TUI, reopen a just-completed pending entry and confirm it is no longer struck.

### Tests for User Story 3

- [ ] T012 [P] [US3] Write failing tests in `internal/tui/update_test.go`: after completing then uncompleting an untimed entry, `pendingComplete` is cleared and the entry renders normally (not struck) and remains present.
- [ ] T013 [P] [US3] Add a test in `services/twig-web/src/lib/planView.test.ts` asserting `groupPlan` includes incomplete (reopened) untimed entries in the `untimed` group.

### Implementation for User Story 3

- [ ] T014 [US3] In `internal/tui/update.go` Complete handler, when `complete == false` (uncomplete) set `m.plan.pendingComplete = nil` so the reloaded, now-incomplete entry displays normally (its `Completed == false` already lets it pass `displayedPlanEntries`).

**Checkpoint**: All three user stories pass independently.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verify the increment end-to-end and confirm scope boundaries.

- [ ] T015 [P] Run the full suites: `go test ./...` at repo root and `npm test` + `npm run build` in `services/twig-web/`; confirm all green.
- [ ] T016 [P] Walk through `specs/044-hide-completed-unscheduled/quickstart.md` manual steps for both the TUI and the web planner.
- [ ] T017 Confirm scope: `git diff --name-only` touches only `internal/tui/` and `services/twig-web/src/lib/` (plus this spec dir) — no changes under `api/`, `services/twig/`, or `internal/cli/` (Constitution II: contract unchanged).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none.
- **Foundational (Phase 2)**: after Setup. Blocks the TUI parts of US1 (T008) and all of US2. Does **not** block the web parts of US1 (T005, T007).
- **US1 (Phase 3)**: web tasks need only Setup; TUI task T008 needs Phase 2.
- **US2 (Phase 4)**: needs Phase 2 and US1's T008 (shares `handlePlanEntriesMsg`/displayed-list behavior).
- **US3 (Phase 5)**: needs US2 (shares the `pendingComplete` lifecycle in the Complete handler).
- **Polish (Phase 6)**: after all desired stories.

### User Story Dependencies

- **US1 (P1)**: foundation for the others; web half fully independent.
- **US2 (P1)**: builds on US1's TUI load filter; same files (`update.go`, `plan_update.go`).
- **US3 (P2)**: builds on US2's Complete-handler changes.

### Within Each Story

- Write the failing test before the implementation task it covers.
- TUI tasks editing `update.go`/`plan_update.go` are sequential (same files); web tasks run in parallel with TUI.

### Parallel Opportunities

- T003 (TUI helper test) ∥ T002 (model field) — different files.
- T005/T007 (web) ∥ T006/T008 (TUI) — different modules.
- T012 (TUI test) ∥ T013 (web test).
- T015 ∥ T016 in Phase 6.
- Within the TUI module, `update.go` and `plan_update.go` edits must not be parallelized with each other.

---

## Parallel Example: User Story 1

```bash
# Web and TUI halves of US1 proceed in parallel (different modules):
Task: "T005 [US1] failing groupPlan tests in services/twig-web/src/lib/planView.test.ts"
Task: "T007 [US1] filter completed untimed in services/twig-web/src/lib/planView.ts"
# alongside:
Task: "T006 [US1] failing handlePlanEntriesMsg filter test in internal/tui/plan_update_test.go"
Task: "T008 [US1] apply displayedPlanEntries in internal/tui/plan_update.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 Setup → Phase 2 Foundational → Phase 3 US1.
2. **STOP and VALIDATE**: completed untimed entries are gone from both surfaces; timed/events unaffected.
3. Ship — the core value is delivered.

### Incremental Delivery

1. US1 (MVP) → US2 (TUI deferred-hide polish) → US3 (reopen safety net).
2. Each story is independently testable and adds value without breaking the prior one.

---

## Notes

- `PlanEntry.completed` is already on the wire — no API/proto/server/DB tasks exist by design.
- The crossed-out look reuses existing `applyCompletion`/`DimStrike` in `internal/cli/plan_grid.go` — no new styling task.
- No new user-facing copy (Constitution IV): existing completion notices are unchanged.
- Commit after each task or logical group; verify tests fail before implementing.
