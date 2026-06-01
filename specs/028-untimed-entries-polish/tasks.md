---
description: "Task list for Untimed Plan Entries — Polish & Bugfixes"
---

# Tasks: Untimed Plan Entries — Polish & Bugfixes

**Input**: Design documents from `/specs/028-untimed-entries-polish/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/plan-service.md, quickstart.md

**Tests**: Included. This codebase has established test patterns for all three change sites (`internal/handler/plan_test.go`, `internal/cli/plan_grid_test.go`, `internal/tui/*_test.go`), and these are bugfix/regression-sensitive changes, so each story gets focused tests written before implementation.

**Organization**: Tasks are grouped by user story. Phases are ordered by priority: the two P1 stories (US2, US3) first, then US1 (P2), then US4 (P3). User-story IDs (US1–US4) match the spec for traceability.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1–US4 per spec.md
- All paths are under `services/twig/` (the Go module)

## Path Conventions

Single Go module at `services/twig/`. No proto/`gen/`/`internal/db/` changes this feature (no `make proto` / `sqlc generate`).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish a known-green baseline before changes.

- [X] T001 Confirm baseline is green: from `services/twig/` run `go test ./...` and `go build ./cmd/twig ./cmd/server`; record any pre-existing failures before starting.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Cross-story prerequisites.

**None.** The three change sites — handler (US2), CLI renderer (US3, US4), and TUI (US1) — are independent and require no shared scaffolding. Each user story below can be implemented and tested on its own. (One soft sequencing note: US4 edits `internal/cli/plan_grid.go`, which US3 also restructures, so do US4 after US3 to avoid churn — see Dependencies.)

**Checkpoint**: Proceed directly to user stories.

---

## Phase 3: User Story 2 - Prevent duplicate untimed entries (Priority: P1) 🎯 MVP

**Goal**: Enforce "at most one untimed entry per (day, task)" server-side so every client path inherits it.

**Independent Test**: Via CLI — `twig plan task <day> T` twice (2nd rejected), `twig plan task <day> T 09:00` (allowed), `twig plan mv <timed-id> null` onto an existing untimed duplicate (rejected, keeps time).

### Tests for User Story 2 ⚠️ (write first, ensure they fail)

- [X] T002 [US2] Add handler tests in `services/twig/internal/handler/plan_test.go` covering: (a) `AddPlanTask` untimed rejects a 2nd untimed entry for the same (day, task_id) with `FailedPrecondition` and creates nothing; (b) `AddPlanTask` timed still succeeds alongside an existing untimed entry for the same task; (c) untimed add succeeds when only a *timed* entry exists for the task; (d) `MovePlanEntry` clearing start is rejected when a *different* untimed entry exists for the task and the entry keeps its time; (e) `MovePlanEntry` clearing start succeeds when no other untimed entry exists.

### Implementation for User Story 2

- [X] T003 [US2] In `AddPlanTask` (untimed branch, `isTimed == false`) in `services/twig/internal/handler/plan.go`: lock the day via `LockPlanEntriesForDay` inside the existing serializable tx, scan for an entry with the same `task_id` and unset `start_minute`, and reject with `connect.CodeFailedPrecondition` if found (no insert).
- [X] T004 [US2] In `MovePlanEntry` (clear branch, `isTimed == false`) in `services/twig/internal/handler/plan.go`: lock the day, scan for a *different* entry (id ≠ request id) with the same `task_id` (from `existing.TaskID`) and unset `start_minute`, and reject with `FailedPrecondition` if found (entry unchanged).
- [X] T005 [US2] Author the shared playful duplicate-rejection message (Principle IV — accurate + actionable) used by T003 and T004 in `services/twig/internal/handler/plan.go`.

**Checkpoint**: At most one untimed entry per (day, task) across CLI add, CLI mv-clear (and, transitively, TUI). Run `go test ./internal/handler/...`.

---

## Phase 4: User Story 3 - Untimed entries render like gridded entries (Priority: P1)

**Goal**: Untimed entries render with the grid's box geometry — title inside the box, joined borders, consistent per-line colors.

**Independent Test**: `twig plan show <day>` with two adjacent multi-line untimed entries → titles inside `┏┓` boxes, boxes share a `┣┫` boundary, last-line colors match the other lines and a gridded entry of the same duration/state.

### Tests for User Story 3 ⚠️ (write first, ensure they fail)

- [X] T006 [US3] Add rendering tests in `services/twig/internal/cli/plan_grid_test.go` asserting that `RenderUntimed` output: has a `┏…┓` top border above each title (title not on the border line), shares a `┣…┫` boundary between two adjacent untimed entries, and applies consistent selection/completion styling across every line including the last; assert byte/style parity with the equivalent gridded entry box.

### Implementation for User Story 3

- [X] T007 [US3] Extract the per-entry box-line drawing (top `┏┓`, interior `┃content┃`, shared `┣┫`, bottom `┗┛`, with `applySelection`/`applyCompletion`) from `RenderGrid` into a shared helper in `services/twig/internal/cli/plan_grid.go`.
- [X] T008 [US3] Rewrite `RenderUntimed` in `services/twig/internal/cli/plan_grid.go` to stack entries back-to-back through the shared helper (shared boundaries between adjacent untimed entries), removing the bespoke `┣label┫`-as-first-row geometry; keep the existing gutter/`boxWidth`/`contentWidth` constants so output aligns with the grid.

**Checkpoint**: Untimed and gridded entries are visually indistinguishable. Run `go test ./internal/cli/...`.

---

## Phase 5: User Story 1 - Status-bar feedback on send (Priority: P2)

**Goal**: Pressing `p`/`ctrl+p` on the Tasks tab shows a confirmation in the status bar (and a rejection message when blocked as a duplicate).

**Independent Test**: In the TUI Tasks tab, `p` → confirmation naming the task + "today"; `ctrl+p` → confirmation naming the task + chosen date; `p` on an already-untimed task → rejection message visible on the Tasks tab, no second entry.

### Implementation for User Story 1

- [X] T009 [P] [US1] Add a `notice string` field to `Model` in `services/twig/internal/tui/model.go` (transient info channel, distinct from `err`).
- [X] T010 [US1] In `renderStatus` in `services/twig/internal/tui/view.go`, render `m.notice` when it is non-empty and no active error is present (errors still take precedence); falls back to the help line when both are empty.
- [X] T011 [US1] In `services/twig/internal/tui/plan_update.go`, carry a success notice from the send (e.g. add a `notice` field to `planMutatedMsg` set by `addPlanTaskCmd`, or set it when handling the result) and set `m.notice` on success; clear the notice on the next user action.
- [X] T012 [US1] In the `PlanSendToday` (`p`) and date-prompt-confirm (`ctrl+p`) handlers in `services/twig/internal/tui/update.go`, supply the notice context (task name + target day) and route the send's *error* result to the tab-agnostic `m.err` (not the planning-only `m.plan.err`) so a duplicate rejection (US2) is visible on the Tasks tab.
- [X] T013 [US1] Author playful send-confirmation copy (Principle IV) for `p` (today) and `ctrl+p` (chosen date) wired into T011/T012.

### Tests for User Story 1

- [X] T014 [US1] Add TUI tests: `p` sets a success notice naming the task/today; `ctrl+p` confirm sets a notice naming the chosen date; a duplicate send surfaces the rejection on the Tasks tab — in `services/twig/internal/tui/update_test.go` (and assert `renderStatus` shows the notice in `services/twig/internal/tui/view_test.go`).

**Checkpoint**: Sends give visible feedback; rejections are visible on the Tasks tab. Run `go test ./internal/tui/...`.

---

## Phase 6: User Story 4 - Visual separation between untimed pane and day planner (Priority: P3)

**Goal**: A clear divider between the untimed pane and the grid, present only when untimed entries exist.

**Independent Test**: `twig plan show <day-with-untimed>` shows a separator between the pane and grid; `twig plan show <day-without-untimed>` shows no pane/separator and the grid uses the full area (unchanged).

### Tests for User Story 4 ⚠️ (write first, ensure they fail)

- [X] T015 [US4] Add tests asserting a separator line is emitted between the untimed pane and grid when untimed entries exist, and absent (full-area grid, byte-identical to no-untimed baseline) when none exist — in `services/twig/internal/cli/plan_grid_test.go` and `services/twig/internal/tui/plan_view_test.go`.

### Implementation for User Story 4

- [X] T016 [US4] Emit a theme-consistent separator between the untimed pane output and the grid only when untimed entries exist, wired where they are concatenated (`RenderUntimed`/grid join in `services/twig/internal/cli/plan_grid.go` and `services/twig/internal/tui/plan_view.go`); ensure the no-untimed path is unchanged.

**Checkpoint**: Boundary is clear when present, invisible when absent. Run `go test ./internal/cli/... ./internal/tui/...`.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T017 [P] Run the full suite: from `services/twig/` `go test ./...` (all green).
- [X] T018 [P] Build both binaries: from `services/twig/` `go build ./cmd/twig ./cmd/server`.
- [X] T019 Review all new user-facing copy (duplicate rejection, send confirmations) against Constitution Principle IV (warm, accurate, actionable).
- [ ] T020 Execute `specs/028-untimed-entries-polish/quickstart.md` manual verification (CLI + TUI) end to end.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none.
- **Foundational (Phase 2)**: none — no blocking prerequisites.
- **User Stories**: all depend only on Setup. They are independent and individually testable, with one soft ordering: **US4 (Phase 6) after US3 (Phase 4)** because both edit `internal/cli/plan_grid.go` (US3 restructures it; doing US4 after avoids rework).
- **Polish (Phase 7)**: after the stories you intend to ship.

### User Story Dependencies

- **US2 (P1)**: independent (handler only).
- **US3 (P1)**: independent (CLI renderer only).
- **US1 (P2)**: independent (TUI). Its duplicate-rejection-visibility test (T012/T014) is most meaningful once US2 exists, but US1 itself stands alone.
- **US4 (P3)**: independent; sequence after US3 (shared file).

### Within Each Story

- Tests first (T002, T006, T015), then implementation.
- US2: T003 and T004 are in the same file (`plan.go`) → sequential; T005 (shared message) supports both.
- US3: T007 (extract helper) before T008 (use it) — same file, sequential.
- US1: T009 (model field) first; T010/T011/T012 follow (T010 needs the field).

### Parallel Opportunities

- The two P1 stories can be worked in parallel by different people: **US2** (`internal/handler/`) and **US3** (`internal/cli/`) touch disjoint files.
- **US1** (`internal/tui/`) is also disjoint from US2/US3 and can run alongside them.
- T009 is `[P]` (isolated file). T017/T018 are `[P]` (independent commands).

---

## Parallel Example: the two P1 stories

```bash
# Developer A — US2 (handler):
#   T002 tests → T003 → T004 → T005   (services/twig/internal/handler/plan*.go)
# Developer B — US3 (CLI renderer):
#   T006 tests → T007 → T008          (services/twig/internal/cli/plan_grid*.go)
# Developer C — US1 (TUI):
#   T009 → T010/T011/T012 → T013 → T014   (services/twig/internal/tui/*.go)
```

---

## Implementation Strategy

### MVP First

The two P1 stories are the MVP: **US2** (correctness — no duplicate untimed entries) and **US3** (the most visible rendering defect). Ship after Phase 3 + Phase 4, validate via `go test ./...` and `twig plan show`.

### Incremental Delivery

1. Setup (Phase 1) → baseline green.
2. US2 → correctness fix, test via CLI.
3. US3 → rendering parity, test via `plan show`.
4. US1 → send feedback in the TUI.
5. US4 → visual separation.
6. Polish → full suite, tone review, quickstart.

Each story adds value without breaking the others.

---

## Notes

- No proto/DB/codegen changes — do not run `make proto` or `sqlc generate`.
- `[P]` = different files, no incomplete dependency.
- The duplicate rule is server-side (one place), so the CLI gets it for free and the TUI only needs to *surface* the resulting error (US1).
- Commit after each task or logical group; stop at any checkpoint to validate a story independently.
