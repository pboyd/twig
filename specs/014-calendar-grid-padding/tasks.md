---

description: "Task list for Calendar Grid Padding & Gutter Refinement"
---

# Tasks: Calendar Grid Padding & Gutter Refinement

**Input**: Design documents from `/specs/014-calendar-grid-padding/`

**Prerequisites**: plan.md, spec.md (with clarifications), research.md, quickstart.md

**Tests**: The plan's "Test plan (executable contract)" makes tests part of the deliverable, so test tasks are included.

**Organization**: Tasks are grouped by user story (US1, US2, US3) so each story can be implemented and tested independently. Because all three stories edit the same two files (`plan_grid.go`, `plan_grid_test.go`), parallelism across stories is limited; the `[P]` marker is used only for tasks that genuinely touch different concerns within a story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files / disjoint concerns, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

Single Go module at `services/todo/`. All implementation lives in:

- `services/todo/internal/cli/plan_grid.go`
- `services/todo/internal/cli/plan_grid_test.go`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish baseline before touching production code.

- [ ] T001 Read `services/todo/internal/cli/plan_grid.go` end-to-end and note the current per-row construction (gutter, rails, box edges, label rendering) so subsequent edits can be made surgically rather than rewriting.
- [ ] T002 Run `cd services/todo && go test ./internal/cli/...` and capture the baseline pass output for `plan_grid_test.go` so regressions during refactor are easy to spot.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Layout primitives that all three stories build on. Must complete before any user-story task.

- [ ] T003 In `services/todo/internal/cli/plan_grid.go`, introduce internal constants/locals for the new layout invariants used by all stories: left-padding-column index, right-padding-column index, gutter width (including the new extra space between hour label and now-marker), and the calendar interior width = `calendar_width − 2` (rails) `− 2` (padding cols). Update existing width calculations so the box interior width derives from this single source.
- [ ] T004 In `services/todo/internal/cli/plan_grid.go`, refactor the per-row composition path so each row is assembled as: gutter + left rail + left padding cell + box region + right padding cell + right rail. Today's renderer likely assembles rail-and-box together; this task separates the rail (always drawn) from the padding cell (light dash or space depending on row context) so US1/US2/US3 can each plug in their rule without re-tangling the path.

---

## Phase 3: User Story 1 — Hour grid always visible behind entries (P1)

**Goal**: Light hour line and rails continue across rows occupied by an entry; the heavy box is inset one column from each rail.

**Independent test**: Render a day with one 2-hour entry. Verify (a) the intermediate hour rows show a light `─` on both sides of the heavy `┃`, (b) the heavy box's left/right edges sit one column inside the outer rails on every row of the entry, (c) on the top/bottom-edge rows there is a single light `─` between each rail and the heavy corner (no heavy/light junction glyphs).

- [ ] T005 [P] [US1] In `services/todo/internal/cli/plan_grid_test.go`, add `TestPlanGrid_MultiHourEntry_HourLinesFlankBox` covering a single 08:00–10:00 entry: assert the 09:00 row contains `├─┃` on the left and `┃─┤` on the right, and that the 08:00/10:00 rows contain `├─┏…┓─┤` / `├─┗…┛─┤` with the corner characters one column inside the rails. Use the example block in `spec.md` as the golden reference.
- [ ] T006 [US1] In `services/todo/internal/cli/plan_grid.go`, update the box-edge composition for top edge, bottom edge, and interior rows so that the heavy character (`┏`/`┗`/`┃`/`┣`/`┫`) is written into the inset column, never into the rail column. The rail column always receives the rail character (`├`/`┤`/`│`).
- [ ] T007 [US1] In `services/todo/internal/cli/plan_grid.go`, populate the padding cell on hour rows that intersect an entry with a single light `─` (extending the hour line up to one column before the heavy box). On non-hour rows that contain a heavy box, populate the padding cell with a space (` `).
- [ ] T008 [US1] In `services/todo/internal/cli/plan_grid.go`, update existing goldens-touching tests within `plan_grid_test.go` whose expected strings were valid under 013 but no longer match (heavy chars previously in the rail column, missing padding column, etc.). Replace expected strings to match the new layout. After this task, `go test ./internal/cli/...` should pass.
- [ ] T009 [US1] In `services/todo/internal/cli/plan_grid_test.go`, add `TestPlanGrid_OffHourEntry_PaddingPreserved` covering a 09:15–09:45 entry: assert the heavy top and bottom rows each have a single space between rail and box on both sides (not a heavy char, not a light dash, since these rows are non-hour rows).
- [ ] T010 [US1] In `services/todo/internal/cli/plan_grid_test.go`, add `TestPlanGrid_SharedBorder_PaddingPreserved` covering two entries 13:00–13:15 and 13:15–13:30: assert exactly one `┣━…━┫` line at the shared `:15` row, with a single space between each rail and the heavy junction (the shared row is a non-hour row, so per the rule in `plan.md` the padding cell is a space rather than `─`).

---

## Phase 4: User Story 2 — Hour grid spans empty rows too (P1)

**Goal**: Even hours with no entries render the full light grid (hour lines and rails) edge to edge.

**Independent test**: Render an 08:00–11:00 window with a single 09:00–10:00 entry. The 08:00 and 11:00 hour rows render as full `├─…─┤`; the 08:15/:30/:45 and 10:15/:30/:45 rows render as `│ … │` with empty interior.

- [ ] T011 [P] [US2] In `services/todo/internal/cli/plan_grid_test.go`, add `TestPlanGrid_EmptyHours_FullGrid` covering a day with one 09:00–10:00 entry in an 08:00–11:00 window. Assert the 08:00 and 11:00 hour rows are continuous `├─…─┤` from left rail to right rail (including the padding columns rendered as `─`), and the empty :15/:30/:45 rows are `│` + spaces + `│`.
- [ ] T012 [US2] In `services/todo/internal/cli/plan_grid.go`, ensure the "no heavy box at this row" branch emits the hour line edge-to-edge on hour rows (filling both padding columns with `─`) and the rail-only pair on non-hour rows (filling both padding columns with a space). After this task, the test from T011 passes.
- [ ] T013 [US2] In `services/todo/internal/cli/plan_grid_test.go`, add `TestPlanGrid_FirstAndLastHourRows_FullGrid` covering the first and last hour rows of the visible window with no entry crossing them: assert each is a continuous `├─…─┤`.

---

## Phase 5: User Story 3 — "Now" indicator separated from hour label (P2)

**Goal**: On today's plan the `▶` marker has one blank column between it and the hour-label text; the marker column is reserved on every row (rendered blank when no marker is present) so rail position is identical across all rows.

**Independent test**: Render today's plan at a known wall-clock time. The marker row shows `HH:MM` + space + `▶` + rail. A row without the marker shows `HH:MM` + space + ` ` + rail (or all blanks before the rail on non-hour rows). A non-today render shows blanks in the marker column on every row, with rail position unchanged.

- [ ] T014 [P] [US3] In `services/todo/internal/cli/plan_grid_test.go`, add `TestPlanGrid_NowMarker_GutterSpacing` rendering today's plan with a fixed injected "now" time of 08:00 inside an 08:00–11:00 window: assert the 08:00 row's gutter is exactly `08:00 ▶` followed immediately by the left rail (i.e., `08:00` + 1 space + `▶` + rail), and the 09:00 row's gutter is `09:00  ` followed by the left rail (`09:00` + 2 spaces + rail).
- [ ] T015 [US3] In `services/todo/internal/cli/plan_grid.go`, modify the gutter composition so the hour label is followed by a single mandatory space, then the marker column (`▶` only on the current 15-minute row of today's plan; otherwise a space), then the left rail. Confirm the rail's absolute column index is identical on every row in the rendering.
- [ ] T016 [US3] In `services/todo/internal/cli/plan_grid_test.go`, add `TestPlanGrid_NonTodayRender_MarkerColumnBlank` rendering a non-today date and asserting the marker column is blank on every row while the rail column index matches the today-render from T014.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T017 Run `cd services/todo && go test ./internal/cli/...` and confirm all existing and new tests pass.
- [ ] T018 Run `cd services/todo && go build ./...` to ensure the binary still builds.
- [ ] T019 Follow the manual recipe in `specs/014-calendar-grid-padding/quickstart.md` against a running `make dev` stack and confirm the rendered output matches the example block in `spec.md` (hour lines flank boxes, boxes inset, marker separated from hour text).
- [ ] T020 Visually diff a small representative `plan` rendering against the spec example to catch any character-level drift not caught by goldens (e.g. wrong corner glyph, missed padding column on an unusual row).

---

## Dependencies

- Phase 1 (Setup) → Phase 2 (Foundational) → Phases 3/4/5 (user stories) → Phase 6 (Polish).
- Within Phase 3 (US1), T006 must complete before T007; T008 depends on T006+T007 (existing tests will fail until the layout is updated). T005, T009, T010 are test-only and can be authored at any time within US1.
- Phase 4 (US2) and Phase 5 (US3) both depend on the gutter-and-padding refactor from Phase 2 and on the box-edge changes from US1 (specifically T006+T007), because their assertions rely on the rail position and padding cells US1 establishes.
- Phase 6 depends on all preceding phases.

## Parallel Opportunities

- T005 (US1 test), T011 (US2 test), and T014 (US3 test) can all be authored in parallel — they're disjoint test files-of-concern within `plan_grid_test.go` and have no production-code dependency on each other.
- T009 and T010 (additional US1 tests) can be authored in parallel with each other once T005 is in place.
- T013 and T016 (additional US2/US3 tests) can be authored in parallel with their respective story implementation tasks.
- Implementation tasks across stories (T007, T012, T015) are NOT parallelizable — they all edit overlapping regions of `plan_grid.go`.

## Implementation Strategy

**MVP scope**: US1 + US2 (both P1). With these, the always-on grid and the box inset are delivered; the calendar reads as the user intended in the example. US3 (gutter spacing for the now marker) is P2 and can be deferred to a follow-up commit if needed, though it is small enough that there's little reason to defer.

**Incremental delivery**:

1. Land Phase 1 + Phase 2 as a no-behavior-change refactor (rails and padding cells separated, goldens unchanged).
2. Land Phase 3 (US1) — heavy box inset + always-on grid on entry rows. Update goldens.
3. Land Phase 4 (US2) — always-on grid on empty rows. Add new tests.
4. Land Phase 5 (US3) — gutter spacing for the now marker.
5. Land Phase 6 — manual verification + cleanup.

Each step keeps the test suite green and produces a meaningfully testable increment.

## Format Validation

All tasks above use the strict format `- [ ] TID [P?] [Story?] Description with file path`:

- Setup (T001–T002) and Foundational (T003–T004): no `[Story]` label.
- US1 (T005–T010): `[US1]` label on every task.
- US2 (T011–T013): `[US2]` label on every task.
- US3 (T014–T016): `[US3]` label on every task.
- Polish (T017–T020): no `[Story]` label.
- `[P]` is applied only to tasks that touch distinct, non-conflicting concerns (chiefly test-authoring tasks that can run before their corresponding implementation lands).
