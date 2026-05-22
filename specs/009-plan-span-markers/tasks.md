---

description: "Task list for plan-span-markers feature"
---

# Tasks: Plan Span Markers

**Input**: Design documents from `/specs/009-plan-span-markers/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: This feature already has table-driven tests for `RenderGrid` in `services/todo/internal/cli/plan_grid_test.go`. Tests for the new behavior are required (they encode the visual contract from the spec) and are listed as ordinary tasks under the single user story — no separate TDD phase.

**Organization**: This feature has exactly one user story (US1). All implementation tasks live under the US1 phase. There is no shared setup or foundational phase because no new project infrastructure is introduced.

## Format: `[ID] [P?] [Story] Description`

---

## Phase 1: User Story 1 — Read the plan at a glance without name repetition (Priority: P1) 🎯 MVP

**Goal**: Replace per-slot repetition of the task name in `todo plan` output with start/middle/end/single-slot span markers, with the task name printed exactly once per contiguous block.

**Independent Test**: Run `go test ./internal/cli/... -run RenderGrid` and verify all updated and new table-driven cases pass. Then follow `specs/009-plan-span-markers/quickstart.md` to render against a live plan and confirm the four marker glyphs render in the four expected positions.

- [ ] T001 [US1] Update existing snapshot expectations in `services/todo/internal/cli/plan_grid_test.go` so all current multi-slot test cases reflect the new visual contract (`┌` on the start row carrying number + name; `│` on intermediate rows with no name; `└` on the final row with no name).
- [ ] T002 [P] [US1] Add a table-driven test case in `services/todo/internal/cli/plan_grid_test.go` for a single-slot entry — assert the rendered row is `─ <id> <name>` (no `┌`/`└`).
- [ ] T003 [P] [US1] Add a table-driven test case in `services/todo/internal/cli/plan_grid_test.go` for a two-slot entry — assert exactly two rows: a `┌`-row with name and a `└`-row without name, no `│` row between them.
- [ ] T004 [P] [US1] Add a table-driven test case in `services/todo/internal/cli/plan_grid_test.go` for two adjacent entries (entry A ends in slot N, entry B starts in slot N+1) — assert entry A's last row is `└` and entry B's next row is `┌ <id> <name>`, with no blank row between them.
- [ ] T005 [US1] Rewrite the per-slot cell logic inside `RenderGrid` in `services/todo/internal/cli/plan_grid.go`: for each 15-minute slot, determine `startsHere = (t <= e.StartMinute < t+15)` and `endsHere = (t < e.StartMinute + e.DurationMinute <= t+15)` for the owning entry, then select the cell content as: `startsHere && endsHere → "─ <id> <name>"`; `startsHere && !endsHere → "┌ <id> <name>"`; `!startsHere && endsHere → "└"`; `!startsHere && !endsHere → "│"`. Preserve the existing `~HH:MM` mid-slot start-label behavior and the leading `HH:MM │ ` row framing.
- [ ] T006 [US1] Run `cd services/todo && go test ./internal/cli/...` and confirm all tests (including T001–T004) pass.
- [ ] T007 [US1] Execute the live-render section of `specs/009-plan-span-markers/quickstart.md` against `make dev` and tick all four scenario checkboxes in that document.

**Checkpoint**: User Story 1 delivers the entire feature. Plan view shows span markers with the name printed once per block.

---

## Phase 2: Polish & Cross-Cutting Concerns

- [ ] T008 [P] Run `cd services/todo && go vet ./...` and `gofmt -l services/todo/internal/cli/` and resolve any findings.

---

## Dependencies

- T001 must be done before T005 (the existing snapshots will fail loudly under the new code; updating them first keeps the test failures during T005 limited to genuinely new behavior).
- T002, T003, T004 are independent of each other and of T001 — they only add new table rows. They can be authored in parallel ([P]).
- T005 depends on T001–T004 (it must satisfy them).
- T006 depends on T005.
- T007 depends on T006.
- T008 depends on T005 (formats the new code).

## Parallel Opportunities

- T002, T003, T004 can be authored together in a single editing pass since they each add an independent row to the same test table — coordinate on insertion location to avoid merge conflicts within the same file.

## MVP Scope

The whole feature is the MVP. Completing User Story 1 (T001–T007) delivers SC-001 through SC-003 from the spec.
