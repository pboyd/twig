# Tasks: Activity Report

**Input**: Design documents from `/specs/048-activity-report/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/report-rpc.md, contracts/report-cli.md

**Tests**: Included — the repo has established test conventions (plan §Testing, research D8) and every prior feature ships tests alongside implementation.

**Organization**: Tasks are grouped by user story. US1 (TUI) and US2 (CLI) render the day-grouped layout; US3 adds the accomplishment-grouped layout for periods > 14 days to both surfaces. Until US3 lands, long periods render day-grouped — each story remains an independently testable increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 (TUI recent activity), US2 (CLI report), US3 (long-range review)

## Phase 1: Setup (API contract)

**Purpose**: Commit the API contract change and regenerate stubs — required by Principle II before any implementation.

- [X] T001 Add `CountCompletedPomodoros` RPC and `CountCompletedPomodorosRequest`/`CountCompletedPomodorosResponse` messages to `api/proto/task/v1/task.proto` exactly per `contracts/report-rpc.md`, then run `make proto` to regenerate `api/gen/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Server-side count RPC and the shared `internal/report` domain logic that every story consumes.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 [P] Add `CountCompletedPomodorosInRange` query to `services/twig/db/queries/pomodoro.sql` per `contracts/report-rpc.md`, then run `sqlc generate` in `services/twig/` to regenerate `services/twig/internal/db/`
- [X] T003 Implement the `CountCompletedPomodoros` handler in `services/twig/internal/handler/pomodoro.go` — user scoping from auth context, `InvalidArgument` for missing timestamps or `end <= start`, count via the T002 query (depends on T001, T002)
- [X] T004 Add `DATABASE_URL`-gated integration test for `CountCompletedPomodoros` in `services/twig/internal/handler/pomodoro_test.go` — counts only `complete` pomodoros with `end_at` in `[start, end)`, ignores other users, `InvalidArgument` cases (depends on T003)
- [X] T005 [P] Create `internal/report/period.go` — `Period` type (From/To local days, Label), preset resolution (`recent`, `today`, `yesterday`, `week`, `last-week`, `month`, `quarter`, `year`; Monday-start weeks), explicit `YYYY-MM-DD` range parsing with validation, `StartUTC`/`EndUTC` half-open range math, `Days` span, and the 14-day layout threshold per `data-model.md`
- [X] T006 Add `internal/report/period_test.go` — preset boundaries (ISO week, month/quarter/year starts), near-midnight and DST-transition day math, inverted/unparseable range validation errors (depends on T005)
- [X] T007 Create `internal/report/group.go` — build `Entry` values from `[]*taskv1.Task` (filter `completed_at` in `[StartUTC, EndUTC)`, resolve `ParentName`, `TopLevelID`, `Depth` by walking `parent_id`), day grouping (most recent local day first, entries by `CompletedAt` desc), and `Totals` per `data-model.md` (depends on T005)
- [X] T008 Add `internal/report/group_test.go` — in/out-of-range filtering, uncompleted tasks excluded, subtask parent context, local-day attribution for near-midnight completions, day ordering (depends on T007)

**Checkpoint**: Foundation ready — `go test ./...` and `cd services/twig && go test ./...` pass; user stories can proceed (in parallel if desired).

---

## Phase 3: User Story 1 - Review recent activity in the TUI (Priority: P1) 🎯 MVP

**Goal**: A Report tab in the TUI showing completed work for the selected preset, defaulting to `recent` (yesterday + today), with single-keystroke preset switching.

**Independent Test**: Complete tasks on known dates, open the TUI, press `tab` to reach the Report tab, and verify day-grouped entries, preset cycling with `←/→`, the playful empty state, and totals — with no CLI command present.

### Implementation for User Story 1

- [X] T009 [US1] Add `tabReport` to the `tab` constants and a `reportState` struct (preset index, period, day groups, totals, loaded, err) to the `Model` in `internal/tui/model.go`
- [X] T010 [US1] Add report fetch command and tab key handling in `internal/tui/update.go` — on entering the tab (default preset `recent`) and on `←/→`/`h`/`l` preset cycling, fetch `ListTasks` + `CountCompletedPomodoros` and build groups via `internal/report`; `r` refreshes; reuse existing error plumbing (depends on T009)
- [X] T011 [P] [US1] Add Report-tab bindings/help entries (preset cycle ←/→, refresh, scroll) to `internal/tui/keymap.go` following the existing `PlanningMode`-style contextual help pattern
- [X] T012 [US1] Create `internal/tui/report_view.go` — themed rendering per `contracts/report-cli.md` TUI section: header with period label + totals, day-grouped body with parent context on subtasks, playful empty state, scroll handling for overflow; all styles from `internal/tui/theme.go` (depends on T009, T010)
- [X] T013 [US1] Wire the Report tab into the tab bar and view dispatch in `internal/tui/view.go` (depends on T012)
- [X] T014 [US1] Add `internal/tui/report_view_test.go` — default preset on entry, preset cycling updates period label, day-grouped ordering, empty state copy, totals line; use the existing `nowFunc`-style time injection (depends on T013)

**Checkpoint**: User Story 1 fully functional — TUI answers "what did I do yesterday?" end-to-end.

---

## Phase 4: User Story 2 - Generate a report from the CLI (Priority: P2)

**Goal**: `twig report [<preset>] [--from <date> --to <date>]` prints a copy-paste-friendly report; plain text when piped.

**Independent Test**: Run `twig report`, `twig report yesterday`, `twig report --from … --to …`, and `twig report | cat`; verify contents, styling gates, validation errors (exit 1), and the empty state (exit 0) without touching the TUI.

### Implementation for User Story 2

- [X] T015 [US2] Create `internal/cli/report.go` — argument parsing per `contracts/report-cli.md` (preset xor `--from`/`--to`, both-or-neither, playful validation errors naming accepted formats, exit 1), fetch `ListTasks` + `CountCompletedPomodoros`, render the day-grouped layout with TTY-gated styling via existing `internal/cli/render.go` conventions, playful empty state (exit 0)
- [X] T016 [US2] Register the `report` command in `internal/cli/cli.go` — `Run` dispatch, root usage listing, `runHelp` case, and a `printReportUsage` function consistent with existing usage text (depends on T015)
- [X] T017 [US2] Add `internal/cli/report_test.go` — preset and explicit-range selection, day-grouped output ordering, parent context, styled vs piped output, validation errors and exit codes, empty state (depends on T016)

**Checkpoint**: User Stories 1 and 2 both work independently; `twig help report` documents the command.

---

## Phase 5: User Story 3 - Long-range accomplishment review (Priority: P3)

**Goal**: Periods > 14 days switch to the accomplishment-grouped layout — completed top-level tasks as headlines with their completed subtasks beneath, plus a "progress on ongoing work" section — on both surfaces.

**Independent Test**: Seed multi-month completion history (quickstart.md shows how), run `twig report quarter` and open the TUI Report tab on `quarter`; verify Finished/ongoing sections, headline ordering, totals, and that every in-period completion appears exactly once.

### Implementation for User Story 3

- [X] T018 [US3] Add accomplishment grouping to `internal/report/group.go` — `AccomplishmentGroup` per `data-model.md`: group entries by `TopLevelID`, split into Finished (top-level completed in period, newest first) and ongoing sections, entries ordered by `Depth` then `CompletedAt`
- [X] T019 [US3] Extend `internal/report/group_test.go` — every entry in exactly one group/section (SC-004 invariant), Finished vs ongoing classification (incl. top-level completed *outside* the period), deep-nesting depth/order (depends on T018)
- [X] T020 [P] [US3] Render the accomplishment layout in `internal/cli/report.go` for periods > 14 days per `contracts/report-cli.md` — Finished / "Progress on ongoing work" sections, indentation, `(done <date>)` annotations; extend `internal/cli/report_test.go` with layout-switch and quarter-output cases (depends on T018)
- [X] T021 [P] [US3] Render the accomplishment layout in `internal/tui/report_view.go` for periods > 14 days using shared theme styles; extend `internal/tui/report_view_test.go` with layout-switch cases for `quarter`/`year` presets (depends on T018)

**Checkpoint**: All user stories independently functional; the 14-day rule selects the layout automatically on both surfaces.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T022 [P] Update `CLAUDE.md` — add `report` to the CLI commands list (Binaries section) and any command examples that enumerate `task`, `pom`, `plan`
- [X] T023 [P] Tone review (Principle IV) of every new user-facing string in `internal/cli/report.go`, `internal/cli/cli.go` usage text, and `internal/tui/report_view.go` — playful but accurate; errors actionable
- [X] T024 Run the full `specs/048-activity-report/quickstart.md` walkthrough against `make dev` (CLI checks, TUI checks, piped output, validation errors, uncomplete-removes-entry)
- [X] T025 Verify all suites green: `go test ./...` at repo root and `cd services/twig && go test ./...` (plus a `DATABASE_URL` run for the handler test); `go build -o twig ./cmd/twig`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — T001 first (contract before implementation, Principle II)
- **Foundational (Phase 2)**: T002 can start immediately (parallel with T001); T003 needs T001+T002; T005–T008 are independent of the server work and can run in parallel with it
- **US1 (Phase 3), US2 (Phase 4)**: Each depends only on Phase 2 — they touch disjoint packages (`internal/tui` vs `internal/cli`) and can proceed in parallel
- **US3 (Phase 5)**: Depends on Phase 2; T020 also touches files from US2 and T021 from US1, so US3 follows whichever surfaces exist (full FR-004 coverage requires US1+US2 done)
- **Polish (Phase 6)**: After all desired stories

### Story Dependency Notes

- **US1 (P1)**: Independent — MVP on its own
- **US2 (P2)**: Independent of US1 (separate packages)
- **US3 (P3)**: Core logic (T018–T019) is independent; rendering tasks extend US1/US2 files

### Parallel Opportunities

- T001 ∥ T002 (different modules)
- T005–T008 chain ∥ T002–T004 chain (root module vs server module)
- After Phase 2: all of Phase 3 ∥ all of Phase 4 (different packages/developers)
- Within US3: T020 ∥ T021 (CLI vs TUI files)
- Polish: T022 ∥ T023

## Parallel Example: After Foundational

```bash
# Developer A — User Story 1 (TUI):
Task: "Add tabReport + reportState to internal/tui/model.go"        # T009 →T010→…→T014

# Developer B — User Story 2 (CLI):
Task: "Create internal/cli/report.go command"                       # T015 →T016→T017
```

## Implementation Strategy

**MVP first**: T001–T008 (contract + foundation), then Phase 3 (US1). Stop, run the TUI checks from quickstart.md, demo. That alone answers the primary use-case ("what did I do yesterday?").

**Incremental delivery**: add US2 (`twig report`) for copy-pasteable status updates, then US3 for quarter/year reviews. Each checkpoint leaves the app shippable; long periods simply render day-grouped until US3 lands.

## Notes

- Tasks reference the contracts they implement: T001–T004 → `contracts/report-rpc.md`; T015–T017, T020 → `contracts/report-cli.md` (required by the constitution's Development Workflow)
- Never hand-edit `api/gen/` or `services/twig/internal/db/` — regenerate (`make proto`, `sqlc generate`)
- Commit after each task or logical group; stop at any checkpoint to validate a story independently
