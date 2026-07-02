---

description: "Task list for Unscheduled Tasks as a List"
---

# Tasks: Unscheduled Tasks as a List

**Input**: Design documents from `/specs/060-unscheduled-task-list/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/untimed-list-rendering.md, quickstart.md

**Tests**: Included. This is a rendering change; the string/golden render tests are the primary way the
contract is verified, and the codebase is already test-driven for these functions.

**Organization**: Tasks are grouped by the two P1 user stories from spec.md. US1 delivers the visual
change (the MVP); US2 verifies interactions are unchanged on top of it.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 or US2 (from spec.md)
- Exact file paths are included in each task.

## Path Conventions

Single Go module at repo root. Affected code lives in `internal/cli/` and `internal/tui/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish a known-green baseline before touching the renderer.

- [X] T001 Confirm a clean baseline: run `go build -o twig ./cmd/twig` and `go test ./...` from repo root; all pass before changes.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Lock in the invariant that a day with no unscheduled tasks must render byte-identical after the change (FR-006), before refactoring the renderer.

**⚠️ CRITICAL**: Complete before user story work so the refactor can be proven non-regressive.

- [X] T002 Add a characterization test in `internal/cli/plan_grid_test.go` asserting `RenderUntimed(nil, …)` and `RenderUntimed([]*planv1.PlanEntry{}, …)` return `""` (empty), plus a `RenderUntimedSeparator(nil, …)` returns `""` — pinning the empty-unscheduled path (FR-006, contract C1). Ensure it passes against current code.

**Checkpoint**: Empty-day invariant is pinned by a test; renderer refactor can proceed safely.

---

## Phase 3: User Story 1 - See unscheduled tasks as a simple list (Priority: P1) 🎯 MVP

**Goal**: Render untimed plan entries as `☐ / ☑ name` list rows above the day grid, separated by the existing divider, instead of time-block boxes.

**Independent Test**: Open the Plan tab (or run `twig plan`) for a day with ≥1 unscheduled task and ≥1 scheduled block; unscheduled tasks appear as checkbox+name rows above the divider, scheduled items remain grid blocks.

### Tests for User Story 1 (write first; they should FAIL before T007) ⚠️

- [X] T003 [P] [US1] In `internal/cli/plan_grid_test.go`, add tests for the new styled `RenderUntimed` row format per contract C2/C3/C5: one line per entry, `"  ☐ " + name` for incomplete and `"  ☑ " + name` for completed (`entry.Completed`), completed name struck, and no `[id]` prefix.
- [X] T004 [P] [US1] In `internal/cli/plan_grid_test.go`, add tests for selection highlight (C3.1): with `GridOptions{SelectedID: id, SelectionStyle: fn}` the matching row is wrapped by `SelectionStyle`; with `SelectionStyle: nil` the plain `applySelection` fallback is applied; non-selected rows are unstyled.
- [X] T005 [P] [US1] In `internal/cli/plan_grid_test.go`, add a plain-mode (`isTTY=false`) test (C4): rows are name-only with no checkbox glyph and no selection styling; completed names pass through `cli.Strike` (no-op unstyled).
- [X] T006 [P] [US1] In `internal/tui/plan_view_test.go`, add a test that the Plan tab composes untimed list rows, then the divider (`RenderUntimedSeparator`), then the grid, for a day mixing untimed and timed entries.

### Implementation for User Story 1

- [X] T007 [US1] Rewrite `RenderUntimed` in `internal/cli/plan_grid.go`: emit exactly one line per entry as `"  " + checkbox + " " + name` (styled) / name-only (plain), with `checkbox = ☑ if entry.Completed else ☐`; apply completed strike/dim via the existing `applyCompletion`/`cli.Strike`; truncate `name` to the available width; drop all `DurationMinute`→rows box geometry and the `[id]`/`HideID` prefix. Preserve selection via existing `opts.SelectedID`/`opts.SelectionStyle`/`applySelection` and the empty-input early return. (Satisfies FR-001, FR-004, FR-008 line-count; contract C1–C5.)
- [X] T008 [US1] Update existing assertions in `internal/cli/plan_grid_test.go` that expected the old box output (`┣┫`, `┏┓`, multi-row) for untimed entries to the new list format; delete now-obsolete box-specific untimed cases.
- [X] T009 [P] [US1] Update `internal/cli/plan_test.go` assertions on `twig plan` untimed output to the new list rows (shared renderer; research Decision 4) — styled and piped/plain paths.
- [X] T010 [P] [US1] Update any untimed-output assertions in `internal/tui/plan_view_test.go` that referenced box glyphs to the new checkbox/name rows; confirm the no-unscheduled case is unchanged.
- [X] T011 [US1] Run `go test ./internal/cli/ ./internal/tui/` and `go build ./cmd/twig`; confirm US1 tests (T003–T006) and updated suites pass.

**Checkpoint**: Unscheduled tasks render as a list above the divider/grid in both the TUI and `twig plan`; empty-day output unchanged.

---

## Phase 4: User Story 2 - Interact with unscheduled tasks unchanged (Priority: P1)

**Goal**: Confirm every existing interaction on unscheduled tasks (select, complete, edit, schedule, reorder, details) behaves identically after the presentation change. No production code changes expected — this phase is regression verification.

**Independent Test**: On the Plan tab, select an unscheduled row and exercise each action; each produces the same result as before the change.

### Tests for User Story 2 ⚠️

- [X] T012 [P] [US2] In `internal/tui/plan_view_test.go` (or `internal/tui/plan_update_test.go`), assert selection navigation still moves onto/through untimed rows and between the list and the grid using entry IDs (FR-005) after the render change.
- [X] T013 [P] [US2] Verify/extend `internal/tui/plan_us2_test.go` (and `internal/tui/plan_update_test.go`) for completing an untimed task (checkbox flips ☐→☑, completed styling) and scheduling an untimed task (moves to grid, leaves the list) — FR-005, FR-007.
- [X] T014 [P] [US2] Verify reorder-within-untimed (`reorderPlanEntryCmd`) and opening details for an untimed entry still behave as before, in `internal/tui/plan_update_test.go` / relevant details test (FR-005).
- [X] T015 [P] [US2] Confirm the completed-untimed visibility rules are unchanged by re-running `internal/tui/display_filter_test.go` and `internal/tui/pending_complete_test.go` (FR-010); adjust only assertions tied to the old box rendering, not behavior.

**Checkpoint**: All unscheduled-task interactions verified unchanged; US1 + US2 both hold.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Full-suite validation, formatting, and manual smoke test.

- [X] T016 Run `gofmt -l` (expect no diffs) and `go vet ./...` on the repo root module.
- [X] T017 Run the full `go test ./...` from repo root; confirm the entire suite is green.
- [X] T018 Execute the manual verification checklist in `specs/060-unscheduled-task-list/quickstart.md` (TUI + `twig plan` CLI), ticking each FR-mapped item.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup. Pins FR-006 before the refactor. Blocks US1.
- **User Story 1 (Phase 3)**: Depends on Foundational. Delivers the MVP.
- **User Story 2 (Phase 4)**: Depends on US1 (can only verify "unchanged behavior" against the new rendering once T007 lands).
- **Polish (Phase 5)**: Depends on US1 + US2 complete.

### Within User Story 1

- Write tests T003–T006 first (they fail against current box renderer) → implement T007 → update legacy assertions T008–T010 → validate T011.
- T008 and T007 touch the same file (`plan_grid.go` / `plan_grid_test.go`) so are sequential; T009 and T010 are different files and parallel with each other.

### Parallel Opportunities

- T003, T004, T005, T006 are new tests in two files — the three in `plan_grid_test.go` share a file (sequence or coordinate), T006 (`plan_view_test.go`) is fully parallel.
- T009 (`plan_test.go`) and T010 (`plan_view_test.go`) run in parallel after T007.
- All US2 verification tasks T012–T015 are independent checks and can run in parallel once US1 is done.

---

## Parallel Example: User Story 1

```bash
# After T007 (the rewrite) lands, update legacy assertions in parallel:
Task: "T009 Update twig plan untimed assertions in internal/cli/plan_test.go"
Task: "T010 Update untimed assertions in internal/tui/plan_view_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 Setup → green baseline.
2. Phase 2 Foundational → pin the empty-day invariant (FR-006).
3. Phase 3 US1 → rewrite `RenderUntimed`, update tests.
4. **STOP and VALIDATE**: unscheduled tasks now render as a list above the grid (TUI + CLI).

### Incremental Delivery

1. Setup + Foundational → safe-to-refactor baseline.
2. US1 → the visual change ships and is independently demoable (MVP).
3. US2 → regression verification confirms no behavior drift.
4. Polish → full suite + manual quickstart.

---

## Notes

- No proto/server/DB/web changes. The web Plan view is explicitly out of scope (FR-009) — do not modify `services/twig-web`.
- The change is one function (`RenderUntimed`); most tasks are test updates around it.
- `internal/tui/plan_view.go` needs no logic change — its height math and `GridOptions` threading already support the list (research Decisions 1 & 5); T010 only touches its test file.
- Commit after each task or logical group; verify tests fail before implementing (T003–T006 before T007).
