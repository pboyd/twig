---
description: "Task list for 013-plan-calendar-grid"
---

# Tasks: Plan Calendar Grid View

**Input**: Design documents from `specs/013-plan-calendar-grid/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/plan.proto.patch.md`, `quickstart.md`

**Tests**: Included. The existing `internal/cli/plan_grid_test.go` is golden-string based, and the quickstart already prescribes a fresh golden suite. New tests are added incrementally inside each user story phase so each story remains independently verifiable.

**Organization**: Tasks are grouped by user story from `spec.md`. US1 and US2 are both P1; all four user stories collaborate inside the same `plan_grid.go` file, so most user-story phases mutate that one file. Each phase still concludes with a working, demonstrable increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on incomplete tasks)
- **[Story]**: Maps to a user story from `spec.md` (US1, US2, US3, US4)
- File paths are project-relative

## Path Conventions

All implementation paths are under `services/todo/`. Generated directories (`gen/`, `internal/db/`) are never hand-edited — they regenerate from `proto/` and `db/queries/`.

---

## Phase 1: Setup

No new setup is required — this feature lives entirely inside an existing Go module with an established build (`make proto`, `sqlc generate`, `go test ./...`). Skip this phase.

---

## Phase 2: Foundational — wire the `completed` field end-to-end

**Purpose**: Get `PlanEntry.completed` flowing from the database to the CLI before any rendering work depends on it. This is the only contract change in the feature and blocks the completed-task styling required by FR-013.

**Constitution gate (II. API-First)**: The proto edit (T001) must land in the same commit as, or before, any code that consumes the new field.

- [X] T001 Add `bool completed = 7;` to the `PlanEntry` message in `services/todo/proto/plan/v1/plan.proto`, per `specs/013-plan-calendar-grid/contracts/plan.proto.patch.md`
- [X] T002 Run `make proto` from the repo root to regenerate `services/todo/gen/plan/v1/`; verify the new `Completed` field appears on the Go struct
- [X] T003 Update the `ListPlanEntries` SELECT in `services/todo/db/queries/plan.sql` to LEFT JOIN `tasks` on `task_id` and project `(plan_entries.task_id <> 0 AND tasks.completed_at IS NOT NULL) AS completed`
- [X] T004 Run `cd services/todo && sqlc generate` to regenerate `services/todo/internal/db/`; verify the new column appears on the returned row struct
- [X] T005 Update the entry-mapping path in `services/todo/internal/handler/plan.go` so `ListPlanEntries` copies the new row column onto `PlanEntry.Completed`
- [X] T006 [P] Add handler unit tests in `services/todo/internal/handler/plan_test.go` covering three cases: event entry → `Completed == false`; task entry with `completed_at == NULL` → `false`; task entry with `completed_at` set → `true`

**Checkpoint**: A CLI built against the updated server can read `entry.Completed` from any `ListPlanEntries` response.

---

## Phase 3: User Story 1 — empty-day calendar grid (Priority: P1) 🎯 MVP

**Goal**: Render the bare calendar scaffolding for any day: hour rows from 08:00–17:00 (extended outward when entries push past), three quarter-hour text rows between each pair of hour labels, terminal-adaptive width.

**Independent Test**: Call `RenderGrid([], "2026-05-27", now, 80, false)` and confirm the output is an 08:00–17:00 grid with no boxes, three blank text rows between hour labels, and a right-edge border at column 80.

- [X] T007 [US1] Rewrite the signature in `services/todo/internal/cli/plan_grid.go` to `func RenderGrid(entries []*planv1.PlanEntry, day string, now time.Time, width int, isTTY bool) string`; gut the existing body
- [X] T008 [US1] In `services/todo/internal/cli/plan_grid.go`, implement the window computation: default `[8*60, 17*60)`, extended outward (rounded to hour) to contain every entry's snapped span
- [X] T009 [US1] In `services/todo/internal/cli/plan_grid.go`, implement the row-by-row scaffold rendering: for each 15-minute row, emit the hour label (or 6-space gutter), the left edge (`├` on hour rows, `│` between), the horizontal field (light `─` on hour rows, spaces between), and the right edge
- [X] T010 [US1] Update the call site in `services/todo/internal/cli/plan.go` to pass `time.Now()`, a terminal width obtained from `golang.org/x/term.GetSize(int(os.Stdout.Fd()))` (floor 60, default 80 on non-TTY), and the existing TTY check
- [X] T011 [P] [US1] In `services/todo/internal/cli/plan_grid_test.go`, replace the existing goldens with: (a) empty-day default window, (b) empty-day with an entry that forces window extension to 07:00 or 18:00

**Checkpoint**: `todo plan` against an empty day shows an 08:00–17:00 calendar that fills the terminal width.

---

## Phase 4: User Story 2 — entries as time-proportional boxes (Priority: P1)

**Goal**: Draw each plan entry as a heavy box positioned by its snapped 15-minute start/end; label includes ordinal, time range, and name; wrap then truncate with ellipsis.

**Independent Test**: Render a single entry from 08:00–10:00 with a short label and a single entry from 11:15–12:00 with a long label; visually compare against the spec's mock for those cases.

- [X] T012 [US2] In `services/todo/internal/cli/plan_grid.go`, add a per-entry layout step: snap `start_minute` and `start_minute + duration_minute` to the nearest 15 minutes, clamp `endRow >= startRow + 1`
- [X] T013 [US2] In `services/todo/internal/cli/plan_grid.go`, integrate entry rendering into the row loop: top edge uses `┏━┓` joined to the surrounding `├─┤`; bottom edge uses `┗━┛`; interior uses `┃ … ┃`
- [X] T014 [US2] In `services/todo/internal/cli/plan_grid.go`, implement the label formatter: first text row gets `[id] HH:MM-HH:MM name`; wrap the name across subsequent interior rows; if the wrapped name would exceed the box's interior rows, truncate the last visible row with `...`
- [X] T015 [US2] In `services/todo/internal/cli/plan_grid.go`, handle the single-row (15-minute) entry case: emit the box as one row with `┣` left, heavy `━` field, `┫` right, and the label inline on the same row
- [X] T016 [P] [US2] In `services/todo/internal/cli/plan_grid_test.go`, add goldens for: 08:00–10:00 short label, 10:00–10:30 with truncation, 11:15–12:00 with wrapping, 13:00–13:15 single-row entry

**Checkpoint**: `todo plan` renders a believable calendar for any single-entry or multi-entry day, matching the spec mock for those cases.

---

## Phase 5: User Story 3 — adjacent entries share a border (Priority: P2)

**Goal**: When entry A's snapped end row equals entry B's snapped start row, the shared row uses a single heavy horizontal line (with `┣━━━━┫` joining both boxes), not two stacked lines.

**Independent Test**: Render entries 10:00–10:30 immediately after an 08:00–10:00 entry; the 10:00 row must be a single heavy line that is also the bottom of the first box and the top of the second.

- [X] T017 [US3] In `services/todo/internal/cli/plan_grid.go`, in the per-row drawing branch, detect when the current row is simultaneously some entry's bottom and another entry's top and emit a single shared `┣━━━━┫` row instead of stacking two borders
- [X] T018 [P] [US3] In `services/todo/internal/cli/plan_grid_test.go`, add goldens for: 08:00–10:00 + 10:00–10:30 sharing a border at 10:00; 13:00–13:15 + 13:15–13:30 sharing a border at 13:15

**Checkpoint**: Two adjacent entries consume exactly the sum of their durations in 15-minute units — no extra row.

---

## Phase 6: User Story 4 — current-time gutter marker (Priority: P2)

**Goal**: When the rendered day equals today (in the local timezone), place a `▶` in the left gutter on the 15-minute row containing the current local time. No marker on non-today renders.

**Independent Test**: Render with `day == "2026-05-27"` and `now == 10:37` and verify the row representing 10:30–10:45 has `▶` in the gutter; render with `day == "2026-05-26"` and confirm no marker appears anywhere.

- [X] T019 [US4] In `services/todo/internal/cli/plan_grid.go`, compute `nowRow` from the injected `now` and the day under render; if the day is not today (in `now.Location()`), set `nowRow = -1`
- [X] T020 [US4] In `services/todo/internal/cli/plan_grid.go`, while emitting each row, write `▶` into the leftmost gutter column when the row index equals `nowRow` (taking priority over the blank gutter; the hour-label gutter still wins on hour rows — render the marker as a small overlay, e.g. between the label and the left edge, to avoid clobbering the hour text)
- [X] T021 [P] [US4] In `services/todo/internal/cli/plan_grid_test.go`, add goldens for: today + now=10:37 (marker on the 10:30 row); not-today (no marker anywhere)

**Checkpoint**: The marker is helpful on today's plan and invisible on past/future days.

---

## Phase 7: Polish — completed-task strikethrough, cross-cutting (FR-013)

**Goal**: Apply strikethrough + dim styling to the label text of any entry whose `Completed` is true, but only when `isTTY` is true. Box border characters are never restyled.

- [X] T022 In `services/todo/internal/cli/plan_grid.go`, when emitting label text inside an entry box, wrap the label substring with `\x1b[2;9m` and reset with `\x1b[0m` if `entry.Completed && isTTY`; otherwise emit the plain string. Reuse the existing TTY/style helpers in `services/todo/internal/cli/render.go` rather than introducing new constants
- [X] T023 [P] In `services/todo/internal/cli/plan_grid_test.go`, add two goldens for completion: `isTTY=false` (no escape codes; label text is plain), `isTTY=true` (label is wrapped in the SGR sequence). Keep the TTY-false golden as the canonical comparison; the TTY-true golden documents the escape-code contract
- [X] T024 Run `cd services/todo && go test ./...` and confirm the whole suite passes
- [X] T025 Build and run the dev stack (`make dev`), provision a user, schedule the entries from `quickstart.md` step 6, complete one task, run `todo plan`, and visually verify the rendering matches the spec mock (hour grid, boxes, shared borders, now-marker, strikethrough on the completed task)

---

## Dependencies & ordering

```
Phase 2 (foundational, T001–T006)
   │
   ▼
Phase 3 US1 (T007–T011)
   │
   ▼
Phase 4 US2 (T012–T016)
   │
   ▼
Phase 5 US3 (T017–T018)
   │
   ▼
Phase 6 US4 (T019–T021)
   │
   ▼
Phase 7 polish (T022–T025)
```

User stories US1–US4 are listed in the order they layer onto the same `plan_grid.go`. Phase 2 must precede Phase 7 (which is the only consumer of `entry.Completed`). Phases 3→4→5 must run in order because each builds on the previous file state. Phase 6 (US4) and Phase 7 (strikethrough) only depend on Phase 3 being complete and can be re-ordered against each other if convenient.

### Parallel opportunities

- **T002 vs T003**: code-gen tasks against different files — can run in parallel by two developers, though the same `make proto` invocation also touches Go modules so a single-developer linearization is simpler
- **T006**, **T011**, **T016**, **T018**, **T021**, **T023**: every test-writing task is marked `[P]` against its implementation tasks; tests share the same `plan_grid_test.go` file so they cannot all run truly in parallel, but they can be authored independently then merged

### MVP scope

US1 + US2 alone (Phases 2 → 3 → 4) deliver a complete, demoable calendar view that replaces the current renderer. US3, US4, and Phase 7 are quality upgrades on top of that MVP.

## Format validation

Every task above starts with `- [ ]`, has a sequential `T###` ID, includes the user-story label for user-story phases (and no story label for foundational/polish phases), and names a concrete file path. The two purely-procedural tasks (`T002` `make proto`, `T004` `sqlc generate`, `T024` `go test`, `T025` `make dev`) name the command instead of a file, which is the correct unit of work for those steps.
