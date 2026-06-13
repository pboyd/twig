# Tasks: Auto-Assign Enhancements

**Input**: Design documents from `/specs/052-auto-assign-enhancements/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/auto-assign-behavior.md, quickstart.md

**Tests**: Included — this codebase follows TDD (handler + helper tests via `export_test.go` shims) and the research phase specifies exact test changes.

**Organization**: Tasks are grouped by user story. US1 (rounding) and US2 (bump) both touch `internal/cli/plan_grid.go`, `internal/cli/plan_grid_test.go`, `internal/tui/update.go`, and `internal/tui/plan_update_test.go`, so they are largely **sequential** at those shared files — `[P]` is only marked where files genuinely differ.

## Path Conventions

Single Go project (root module `github.com/pboyd/twig`). All edits are under `internal/cli/` and `internal/tui/`. No `api/` or `services/twig/` changes.

---

## Phase 1: Setup

**Purpose**: Confirm a green starting point before changing behavior.

- [X] T001 Run `go test ./internal/cli/ ./internal/tui/` from repo root and confirm all auto-schedule tests pass, establishing the pre-change baseline.

---

## Phase 2: Foundational

**Purpose**: None. This feature introduces no shared prerequisite beyond each story's own helpers (rounding lives in US1, `NextGapFloor` in US2). Proceed directly to user stories.

---

## Phase 3: User Story 1 - Placements land on tidy quarter-hour times (Priority: P1) 🎯 MVP

**Goal**: Every auto-assigned start lands on a 15-minute boundary — the day floor rounds down (9:03 → 9:00, even into the past, never before 8 AM); a slot after an existing entry rounds up to avoid overlap.

**Independent Test**: On today's plan at 9:03 with an open morning, press `a` on a task → it lands at 9:00; with an entry ending at 9:07, a following task lands at 9:15.

### Tests for User Story 1 ⚠️ (write first, ensure they fail)

- [X] T002 [P] [US1] In `internal/cli/plan_grid_test.go`, add `AutoScheduleSlot` cases asserting boundary-aligned starts: an obstacle ending off-boundary (e.g. 480–547 / 08:00–09:07) yields a 30-min task at 555 (09:15, rounded up); confirm existing boundary-aligned cases (480/540/600) still pass unchanged.
- [X] T003 [P] [US1] In `internal/tui/plan_update_test.go`, replace `TestAutoSchedule_TodayFloor_UsesCurrentMinute` with `TestAutoSchedule_RoundsFloorDownTo15`: with `nowFunc` at 9:03 on the plan day and an open morning, assert the task is placed at 540 (09:00), not 543; add a case at 8:07 asserting the floor does not drop below 480 (8:00).

### Implementation for User Story 1

- [X] T004 [US1] In `internal/cli/plan_grid.go`, add a `ceil15(m int) int` helper and round every candidate gap start up to the next 15-minute boundary inside `AutoScheduleSlot` (apply to both the merged-gap branch and the end-of-day tail) before the fit test, so returned `startMin` is always `% 15 == 0` and never overlaps an obstacle.
- [X] T005 [US1] In `internal/tui/update.go` (`PlanAutoSchedule` handler), round the computed `floorMin` **down** to the nearest 15-minute boundary (`floorMin = (floorMin/15)*15`) after the today/now adjustment, keeping the existing `max(480, nowMin)` so it never goes below 8 AM.

**Checkpoint**: Pressing `a` produces only quarter-hour starts; T002/T003 pass; baseline regression tests still green.

---

## Phase 4: User Story 2 - Pressing `a` on an already-placed task moves it on (Priority: P2)

**Goal**: When the highlighted task is already in its earliest-fitting slot, `a` bumps it past the entry closing its current stretch into the next free gap, instead of doing nothing — or shows the "no room" notice when no later gap exists.

**Independent Test**: Place a 30-min task at 8:00 with an entry 9:00–10:00 and free time after; press `a` → task jumps to 10:00. A task in the last free stretch of a full day → stays put + "Day's packed…" notice.

**Depends on**: US1 (T004 rounding is reused by the bump's re-run of `AutoScheduleSlot`; T005 handler edit precedes the handler change in T009).

### Tests for User Story 2 ⚠️ (write first, ensure they fail)

- [X] T006 [P] [US2] In `internal/cli/plan_grid_test.go`, add `NextGapFloor` unit tests: entry 540–600 with `fromMin=480` returns (600, true); two entries (540–600, 660–720) returns the first end (600, true); only an entry before `fromMin`, or no entries, returns ok=false; `excludeID` is honored.
- [X] T007 [US2] In `internal/tui/plan_update_test.go`, replace `TestAutoSchedule_AlreadyInPlace_NoOp` with `TestAutoSchedule_AlreadyInPlace_BumpsToNextGap` (task at 480 its earliest slot, entry 540–600, free after → moved to 600) and add `TestAutoSchedule_Bump_NoLaterGap_SetsNotice` (task in the last/only free stretch → unchanged + "Day's packed…" notice).

### Implementation for User Story 2

- [X] T008 [US2] In `internal/cli/plan_grid.go`, implement `NextGapFloor(timed []*planv1.PlanEntry, fromMin int, excludeID int32) (floorMin int, ok bool)` returning the `end` of the first merged obstacle whose `start >= fromMin`; `ok=false` when none — reuse the same merged-interval construction as `AutoScheduleSlot`.
- [X] T009 [US2] In `internal/tui/update.go` (`PlanAutoSchedule` handler), replace the current no-op branch (`*entry.StartMinute == startMin → return m, nil`) with the bump: call `cli.NextGapFloor(timed, *entry.StartMinute, entry.Id)`; on ok, `cli.AutoScheduleSlot(timed, dur, bumpFloor, entry.Id)` then `movePlanCmd(...)`; otherwise set the existing "Day's packed…" notice and don't move. Leave the unscheduled / `cur != start` paths (place / re-home-earlier) untouched.

**Checkpoint**: `a` on an already-earliest task advances it (never a silent no-op); T006/T007 pass; US1 behavior intact.

---

## Phase 5: User Story 3 - Existing guarantees still hold (Priority: P3)

**Goal**: Confirm events stay untouched, packed days notify, the highlight follows the move, and re-home-to-an-earlier-slot still works — all unchanged by US1/US2.

**Independent Test**: Re-run the existing auto-schedule scenarios and confirm identical outcomes.

- [X] T010 [US3] In `internal/tui/plan_update_test.go`, confirm `TestAutoSchedule_EventHighlighted_SetsNoticeNoMove`, `TestAutoSchedule_NoFit_SetsNotice`, `TestAutoSchedule_EmptyPlan_NoError`, and `TestAutoSchedule_ReHome_MovesEarlier` still pass; if `ReHome_MovesEarlier` used an off-boundary time, adjust its fixture to a boundary so it asserts the re-home-earlier path (not a bump) post-rounding.

**Checkpoint**: All feature-043 guarantees verified green alongside the new behavior.

---

## Phase 6: Polish & Cross-Cutting

- [X] T011 Run `gofmt -w` and `go vet ./...` on the changed packages.
- [X] T012 Run the full suite `go test ./...` and confirm green.
- [X] T013 Build and smoke-test per `quickstart.md`: `go build -o twig ./cmd/twig`, launch the TUI Plan tab, and verify the four manual scenarios (today-floor 9:03→9:00, after-entry 9:07→9:15, bump to next gap, no-room bump notice).

---

## Dependencies & Execution Order

- **Setup (T001)** → before everything.
- **US1 (T002–T005)** is the MVP and comes first. Tests T002/T003 are `[P]` (different files); implement T004 (cli) then T005 (tui).
- **US2 (T006–T009)** depends on US1: T009 edits the same handler block as T005, and the bump re-runs the now-rounding `AutoScheduleSlot` (T004). Within US2, T006 is `[P]`; T008 (cli) before T009 (tui).
- **US3 (T010)** after US1+US2 to validate no regressions.
- **Polish (T011–T013)** last.

Shared-file note: `plan_grid.go`, `plan_grid_test.go`, `update.go`, and `plan_update_test.go` are each touched by both US1 and US2 — do those stories sequentially, not in parallel.

## Parallel Execution Examples

- Within US1: run T002 (`plan_grid_test.go`) and T003 (`plan_update_test.go`) in parallel — different files.
- Within US2: T006 (`plan_grid_test.go`) can be written while reviewing T007's plan, but both precede their implementations.
- Cross-story parallelism is **not** available here due to shared files.

## Implementation Strategy

**MVP = User Story 1** (quarter-hour rounding). It is independently shippable: it improves every placement and requires no bump logic. Ship US1, verify, then layer US2 (bump) and US3 (regression sign-off).
