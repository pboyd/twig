---
description: "Task list for Auto-Schedule a Task"
---

# Tasks: Auto-Schedule a Task

**Input**: Design documents from `/specs/043-auto-schedule-task/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/auto-schedule.md, quickstart.md

**Tests**: Included. This codebase relies on table-tested pure helpers (`internal/cli`) and handler tests (`internal/tui`, via the `export_test.go` shim pattern), and the plan's contract mapping calls out specific tests. Test tasks are therefore part of each phase and written before their implementation.

**Organization**: Tasks are grouped by user story so each can be implemented and verified independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1 / US2 / US3 (Setup, Foundational, and Polish carry no story label)
- All paths are repository-root-relative (root CLI/TUI module)

## Path Conventions

This feature touches only the root module:

- Pure algorithm: `internal/cli/plan_grid.go` (+ `internal/cli/plan_grid_test.go`)
- TUI keymap: `internal/tui/keymap.go`
- TUI handler: `internal/tui/update.go` (+ `internal/tui/update_test.go`, `internal/tui/export_test.go`)

No `services/twig`, `api/`, or `services/twig-web` changes.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Register the key binding that exposes the feature; shared by all three stories.

- [X] T001 [P] Add a `PlanAutoSchedule` `key.Binding` bound to `"a"` (help: `"a"`, `"auto-schedule"`) to the `KeyMap` struct and its initializer in `internal/tui/keymap.go`, and include it in the planning-mode `ShortHelp`/`FullHelp` listings alongside `PlanAddTask`/`PlanAddEvent`/`PlanEdit`.

**Checkpoint**: `a` is a recognized planning-tab binding and shows in help (handler is wired in later phases).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The pure earliest-fit slot-finder that every story's placement relies on.

**⚠️ CRITICAL**: User Story 1 and User Story 2 cannot function until `AutoScheduleSlot` exists.

- [X] T002 [P] Write table-driven unit tests for `AutoScheduleSlot` in `internal/cli/plan_grid_test.go` covering: empty day → 480; gap immediately after a blocking entry (08:00–09:00 → 30-min task at 540); too-small interior gap skipped (08:00–09:00 & 09:15–10:00 → 30-min task at 600); `floorMin` respected (no start < floor); zero-duration task fitted as a 30-minute block; block must end by 1440 (no overflow past midnight); `excludeID` removes the named entry from obstacles; overlapping obstacles treated as a union; no-fit returns `ok == false`. Ensure tests fail before T003.
- [X] T003 Implement `AutoScheduleSlot(timed []*planv1.PlanEntry, durationMin, floorMin int, excludeID int32) (startMin int, ok bool)` in `internal/cli/plan_grid.go` per `contracts/auto-schedule.md` §2: union timed obstacles (skipping `excludeID`) clipped to `[floorMin, 1440)`, scan free intervals ascending, return the earliest whose length ≥ `max(durationMin, 30 when durationMin<=0)`; pure, no input mutation. Make T002 pass.

**Checkpoint**: The placement engine is correct and unit-tested in isolation.

---

## Phase 3: User Story 1 - Drop a task into the next open slot (Priority: P1) 🎯 MVP

**Goal**: Pressing `a` on a highlighted (untimed) task places it at the earliest free slot at/after the scheduling floor where its duration fits, keeping it highlighted.

**Independent Test**: In the planning tab, highlight a task with a duration on a day with gaps and press `a`; confirm it becomes a timed entry at the earliest fitting start ≥ floor and stays selected. On a full day, confirm a playful "no room" notice and no change.

- [X] T004 [US1] Ensure handler tests can drive planning key presses: confirm an entry point for `handlePlanKey` exists for tests; if not, add a thin exported shim in `internal/tui/export_test.go` (mirroring existing shims).
- [X] T005 [US1] Write handler tests in `internal/tui/update_test.go` for pressing `a` on a task: (a) empty day places at 08:00 via a `MovePlanEntry`/`movePlanCmd` with `start_minute=480`, `timed=true`, duration unchanged; (b) gap-after-block places at end of the blocking entry; (c) too-small gap is skipped; (d) **today after 08:00** uses a floor of the current local minute, never 08:00 (FR-003) — inject/control "now" so the test is deterministic; (e) no fitting slot → `m.notice` set (playful), no command dispatched, entries unchanged. Tests fail before T006.
- [X] T006 [US1] Implement the `case key.Matches(msg, m.keys.PlanAutoSchedule)` branch in `handlePlanKey` in `internal/tui/update.go`: guard on `len(m.plan.entries) > 0` and highlighted `entry.TaskId != 0`; compute `floorMin = 480`, raised to the current local minute-of-day when `m.plan.day == time.Now().Local().Format("2006-01-02")` and that minute > 480; split timed entries (reuse `splitPlanEntries`); call `cli.AutoScheduleSlot(timed, int(entry.DurationMinute), floorMin, entry.Id)`; on `ok`, dispatch `movePlanCmd(m.planClient, m.plan.day, entry.Id, startMin, int(entry.DurationMinute), true)`; on `!ok`, set a playful `m.notice` and return without a command. Make T005 pass.

**Checkpoint**: US1 is a usable MVP — tasks auto-place into the next open slot (today-aware) with a graceful full-day message; highlight follows via the reused `movePlanCmd`.

---

## Phase 4: User Story 2 - Re-home a task that no longer fits where it is (Priority: P2)

**Goal**: Pressing `a` on an already-timed task lifts it from its current slot and places it at the earliest fitting slot (which may be earlier), and is a no-op when it's already at that earliest slot.

**Independent Test**: Place a task at 14:00 with 08:00 free; press `a`; confirm it moves to 08:00. With a task already at the earliest fitting slot, press `a`; confirm no change and no spurious write.

- [X] T007 [US2] Write handler tests in `internal/tui/update_test.go`: (a) a task timed at 14:00 with an open 08:00 slot moves to 08:00 (the task does not block itself — relies on `excludeID`); (b) a task already at the earliest fitting start produces **no** dispatched command and no change (FR-011 / US2.3). Tests fail before T008.
- [X] T008 [US2] In the `handlePlanKey` auto-schedule branch (`internal/tui/update.go`), add the no-op guard: when the highlighted task is already timed (`entry.StartMinute != nil`) and `int(*entry.StartMinute) == startMin`, return without dispatching `movePlanCmd` (and without a notice). The re-home/exclude-self path already works through `entry.Id` as `excludeID` from T006. Make T007 pass.

**Checkpoint**: Re-homing and already-in-place no-op both verified; US1 behavior remains green.

---

## Phase 5: User Story 3 - Auto-schedule does not apply to events (Priority: P3)

**Goal**: Pressing `a` on an event (or with nothing highlighted) changes nothing; an event highlight yields a playful "tasks only" notice.

**Independent Test**: Highlight an event, press `a`; confirm no entry moves and a playful notice explains auto-schedule is task-only. Press `a` on an empty day; confirm nothing happens and no error.

- [X] T009 [US3] Write handler tests in `internal/tui/update_test.go`: (a) highlighted event (`TaskId == 0`) → no command dispatched, plan unchanged, `m.notice` set to a playful task-only message; (b) empty `m.plan.entries` → no command, no error, no panic. Tests fail before T010.
- [X] T010 [US3] Extend the auto-schedule branch in `internal/tui/update.go` with the event/empty guards (mirroring the existing Complete/PomStart event guards): when `entry.TaskId == 0`, set a playful `m.notice` and return; when no entries/no highlight, return silently. Make T009 pass.

**Checkpoint**: All three stories complete; guards prevent any unwanted mutation.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Tone, consistency, and verification across the whole feature.

- [X] T011 [P] Review all new `m.notice` strings (no-fit, event-only) for Principle IV tone and Principle III consistency with surrounding planning messages in `internal/tui/update.go`; refine wording (warm, accurate, actionable).
- [X] T012 [P] Confirm the `a` binding renders correctly in the planning help overlay and does not collide with any existing planning binding (`internal/tui/keymap.go` / help view).
- [X] T013 Run `gofmt`/`go vet ./...` and the full suite `go test ./...` from repo root; ensure `internal/cli` and `internal/tui` tests pass.
- [ ] T014 Walk through `specs/043-auto-schedule-task/quickstart.md` against a running TUI (`go build -o twig ./cmd/twig && ./twig`) to manually verify the acceptance scenarios (empty day, gap, too-small gap, re-home, already-in-place, event guard, today-afternoon, full day).

---

## Dependencies & Execution Order

- **Setup (Phase 1)**: T001 — independent; can be done first or in parallel with Phase 2.
- **Foundational (Phase 2)**: T002 → T003. **Blocks US1 and US2** (they call `AutoScheduleSlot`).
- **US1 (Phase 3)**: T004 → T005 → T006. Depends on T003 (engine) and T001 (binding).
- **US2 (Phase 4)**: T007 → T008. Builds on the US1 handler branch (T006) and the engine's `excludeID` (T003).
- **US3 (Phase 5)**: T009 → T010. Extends the US1 handler branch (T006); independent of US2.
- **Polish (Phase 6)**: T011–T014 after the stories they cover are implemented; T013/T014 last.

Story independence: US1 is a standalone MVP. US2 and US3 each extend the same handler branch but are independently testable and add no cross-dependency on each other.

## Parallel Opportunities

- T001 (keymap) ∥ T002 (cli tests) — different files/areas.
- Within a story, the test task and impl task are sequential (TDD); across stories US2 and US3 implementation could be interleaved once US1's T006 lands, but both edit `internal/tui/update.go`, so coordinate edits to avoid conflicts (not marked `[P]`).
- Polish T011 ∥ T012 — different concerns/files.

## Implementation Strategy

1. **MVP = Phase 1 + Phase 2 + Phase 3 (US1)**: delivers the core "press `a` to place a task in the next open slot," today-aware, with a graceful full-day message and highlight-follow.
2. **Increment 2 = US2**: re-homing already-timed tasks + already-in-place no-op.
3. **Increment 3 = US3**: event/empty guards with playful messaging.
4. **Polish**: tone/consistency review, full test run, manual quickstart verification.

## Task Summary

- **Total**: 14 tasks
- **Setup**: 1 (T001) · **Foundational**: 2 (T002–T003) · **US1**: 3 (T004–T006) · **US2**: 2 (T007–T008) · **US3**: 2 (T009–T010) · **Polish**: 4 (T011–T014)
- **Tests**: T002 (unit), T005/T007/T009 (handler) — written before their implementations
