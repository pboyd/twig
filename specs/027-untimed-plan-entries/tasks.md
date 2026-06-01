---
description: "Task list for Untimed Plan Entries"
---

# Tasks: Untimed Plan Entries

**Input**: Design documents from `/specs/027-untimed-plan-entries/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/plan-service.md, quickstart.md

**Tests**: Test tasks are included — this codebase has established handler/CLI/TUI test conventions (`export_test.go` shims, table tests) and `quickstart.md` gates on `go test ./...`.

**Organization**: All paths are relative to `services/twig/` unless noted. The entire contract + storage + handler change is **Foundational** (API-first): untimed entries must be representable and listable before any surface can use them. User-story phases are the CLI/TUI surfaces.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1–US4 (user-story phases only)

## Path Conventions

Single Go module at `services/twig/`. Generated code (`gen/`, `internal/db/`) is produced by `make proto` / `sqlc generate` — never hand-edited. The React SPA (`services/twig-web/`) is untouched (it does not consume `plan.v1`).

---

## Phase 1: Setup

**Purpose**: Confirm the toolchain needed for code generation and a green baseline.

- [X] T001 Confirm `buf` and `sqlc` CLIs are available and capture a green baseline: `cd services/twig && go test ./...`

---

## Phase 2: Foundational (Blocking Prerequisites — API-First)

**Purpose**: Make an untimed plan entry representable, persistable, addable, schedulable, and listable at the contract/storage/handler layer. **No user story can begin until this phase is complete.**

**⚠️ CRITICAL**: This is the shared contract + backend. It touches `proto/`, `db/`, and `internal/handler/plan.go` — files every story builds on.

- [X] T002 Update `proto/plan/v1/plan.proto`: change `start_minute` to `optional int32` on `PlanEntry` (field 5), `AddPlanTaskRequest` (field 3), and `MovePlanEntryRequest` (field 3); leave `AddPlanEventRequest.start_minute` required — per `contracts/plan-service.md`
- [X] T003 Regenerate ConnectRPC stubs: `cd services/twig && make proto` (updates `gen/plan/v1/`) — depends on T002
- [X] T004 [P] Add migration `db/migrations/000007_plan_untimed.up.sql` (`ALTER COLUMN start_minute DROP NOT NULL`; add `CHECK (task_id IS NOT NULL OR start_minute IS NOT NULL)`) and `000007_plan_untimed.down.sql` (drop constraint; `SET NOT NULL`) — per `data-model.md`
- [X] T005 [P] Edit `db/queries/plan.sql`: make `start_minute` nullable in `InsertPlanEntry` and `UpdatePlanEntryTime`; change `ListPlanEntriesForDay` to `ORDER BY plan_entries.start_minute ASC NULLS FIRST, plan_entries.id ASC`
- [X] T006 Regenerate DB layer: `cd services/twig && sqlc generate` (updates `internal/db/`) — depends on T005
- [X] T007 In `internal/handler/plan.go` `AddPlanTask`: when `start_minute` is absent, create an untimed entry — skip start-range, midnight, and overlap checks; insert `start_minute` as NULL (`pgtype.Int2{Valid:false}`); keep the existing default-duration logic — depends on T003, T006
- [X] T008 In `internal/handler/plan.go` `MovePlanEntry`: when `start_minute` is absent, unschedule — set stored `start_minute` to NULL via `UpdatePlanEntryTime`, preserve duration (`duration_minute==0` keeps it), skip the overlap check; present-start path unchanged — depends on T003, T006
- [X] T009 In `internal/handler/plan.go` `ListPlanEntries` and `dbPlanEntryToProto`: map a NULL stored `start_minute` to an absent proto `start_minute` (set the optional field only when valid); duration/name/completed unchanged — depends on T003, T006
- [X] T010 In `internal/handler/plan.go` `AddPlanEvent`: keep events always-timed — confirm a missing/invalid start is rejected with a friendly `InvalidArgument` (DB `event_timed` CHECK is the backstop) — depends on T003, T006
- [X] T011 [P] Handler tests in `internal/handler/plan_test.go`: untimed `AddPlanTask` (NULL start, default duration), `MovePlanEntry` schedule then unschedule round-trip (duration preserved), `ListPlanEntries` returns untimed with absent start ordered first, and `AddPlanEvent` still rejects a missing start — depends on T007–T010

**Checkpoint**: `cd services/twig && go test ./internal/handler/... && go build ./cmd/server` green — the API supports untimed entries end-to-end.

---

## Phase 3: User Story 1 — Capture a must-do-today task without a time (Priority: P1) 🎯 MVP

**Goal**: From the CLI, add a task to a day's plan with no start time (or `null` + duration) and see it listed distinctly above the grid.

**Independent Test**: `twig plan task <id>` then `twig plan` — the untimed entry appears in an untimed section above the grid, with a duration and no time; `twig plan task <id> null 45m` honors the duration.

- [X] T012 [P] [US1] Add `ParseStartOrNull(s) (minute int, timed bool, err error)` (case-insensitive `null`) in `internal/cli/timeparse/` with table tests
- [X] T013 [P] [US1] Add shared `RenderUntimed(entries, width, isTTY, opts)` in `internal/cli/plan_grid.go` — dark box per entry, one line per 15 minutes of duration, reusing the grid's style constants and `SelectedID` highlight; with tests in `internal/cli/plan_grid_test.go`
- [X] T014 [US1] In `internal/cli/plan.go` `runPlanTask`: make the start arg optional and accept `null` via `ParseStartOrNull`; when untimed, send `AddPlanTaskRequest` with start absent and the trailing arg parsed as a duration — depends on T012
- [X] T015 [US1] In `internal/cli/plan.go` `runPlanShow`: render untimed entries (via `RenderUntimed`) in a section above the grid and pass only timed entries to `RenderGrid` — depends on T013
- [X] T016 [US1] In `internal/cli/plan.go` `printPlanUsage`: update the `task` line to `task <id> [start|null] [dur|end]` and note untimed behavior, in the warm house tone (Principle IV)
- [X] T017 [US1] CLI tests in `internal/cli/plan_test.go`: untimed add path and `runPlanShow` untimed-section output (timed vs untimed placement)

**Checkpoint**: MVP — untimed capture and listing work entirely from the CLI.

---

## Phase 4: User Story 2 — See and manage untimed entries in the TUI (Priority: P2)

**Goal**: A top-left untimed pane (hidden when empty) renders untimed entries like grid boxes; they join the unified Up/Down selection and are editable; the add-task form's start is optional.

**Independent Test**: With an untimed entry on the day, open Planning — the pane shows it (one line per 15 min); ↑/↓ flows pane→grid in one cycle; editing name/duration affects only it; adding a task with a blank Start creates an untimed entry; an empty day hides the pane.

- [X] T018 [P] [US2] Add a helper to split `m.plan.entries` into untimed (absent start) and timed slices, preserving order, in `internal/tui/plan_view.go` (exported for test via `export_test.go`)
- [X] T019 [US2] In `internal/tui/plan_view.go`: render the untimed pane above the grid via `cli.RenderUntimed`, hide it entirely when there are no untimed entries, and pass only timed entries to the grid renderer — depends on T013, T018
- [X] T020 [US2] In `internal/tui/plan_view.go` / `plan_update.go`: pass `SelectedID = m.plan.entries[cursor].Id` to both the untimed pane and the grid so the owning pane highlights, keeping the single Up/Down cycle over `m.plan.entries` (untimed-first) — depends on T019
- [X] T021 [US2] In `internal/tui/plan_update.go`: make the add-task form's Start optional — `initTaskTimeForm` placeholder/label "Start (optional)", `submitTaskTimeForm` treats a blank Start as untimed, and extend `addPlanTaskCmd` to carry an optional start (`*int` / `timed bool`) — depends on Phase 2
- [X] T022 [US2] In `internal/tui/plan_view.go` `planFieldLabel`: reflect the optional Start label for the task-time and edit forms
- [X] T023 [US2] TUI tests in `internal/tui/plan_view_test.go` / `plan_update_test.go`: untimed pane renders boxes, pane hidden when empty, unified ↑/↓ crosses pane↔grid, blank-Start form add creates an untimed entry — depends on T018–T022

**Checkpoint**: Untimed entries are fully visible and manageable in the TUI.

---

## Phase 5: User Story 3 — Schedule an untimed entry later (and unschedule) (Priority: P2)

**Goal**: Give an untimed entry a start (CLI `mv`, TUI edit) to move it onto the grid; clear the start to send it back, preserving duration.

**Independent Test**: `twig plan mv <id> 10:30` schedules; `twig plan mv <id> null` (and bare `twig plan mv <id>`) unschedules, duration intact. In the TUI edit form, typing a time schedules; typing `null` unschedules.

- [X] T024 [P] [US3] In `internal/cli/plan.go` `runPlanMv`: make the start arg optional and accept `null` via `ParseStartOrNull`; absent/`null` → send `MovePlanEntryRequest` with start absent (unschedule, preserve duration); allow the bare `mv <id>` form — depends on T012, Phase 2
- [X] T025 [US3] In `internal/cli/plan.go` `printPlanUsage`: update the `mv` line to `mv <n> [start|null] [dur|end]` with untimed note, playful tone (Principle IV)
- [X] T026 [US3] In `internal/tui/plan_update.go` `submitEditForm`: parse the Start field as blank=keep / time=schedule / `null`=unschedule, and call `movePlanCmd` with an optional start (extend `movePlanCmd` to carry `*int` / `timed bool`) — depends on Phase 2
- [X] T027 [US3] In `internal/tui/plan_view.go`: update the edit-form Start label/help to document blank=keep, time=schedule, `null`=unschedule
- [X] T028 [US3] Tests: CLI `mv` schedule + unschedule (preserves duration) in `internal/cli/plan_test.go`; TUI edit schedule↔unschedule round-trip in `internal/tui/plan_update_test.go` — depends on T024, T026

**Checkpoint**: Entries convert freely between timed and untimed from both surfaces.

---

## Phase 6: User Story 4 — Send tasks from the Tasks tab (Priority: P3)

**Goal**: On the Tasks tab, `p` sends the highlighted task to today's plan as an untimed entry; `ctrl+p` opens a `YYYY-MM-DD` prompt (defaulting to tomorrow) and sends it to that day.

**Independent Test**: Highlight a task, press `p` → untimed entry on today; press `ctrl+p`, accept the pre-filled tomorrow → untimed entry on tomorrow; invalid date or Esc → nothing added with a friendly message.

- [X] T029 [P] [US4] In `internal/tui/keymap.go`: add `PlanSendToday` (`p`) and `PlanSendPickDay` (`ctrl+p`) bindings with help text; surface them in the Tasks-tab ShortHelp/FullHelp
- [X] T030 [US4] In `internal/tui/model.go`: add a Tasks-tab date-prompt view mode and state (a `textinput` pre-filled with tomorrow's `YYYY-MM-DD`)
- [X] T031 [US4] In `internal/tui/update.go`: handle `p` on the Tasks tab → `addPlanTaskCmd` for **today** with an absent start (untimed) for the highlighted task, plus a playful confirmation — depends on T021 (optional-start `addPlanTaskCmd`), T029
- [X] T032 [US4] In `internal/tui/update.go`: handle `ctrl+p` → open the date prompt; on submit validate `YYYY-MM-DD` and `addPlanTaskCmd` untimed for that day; Esc or invalid date → no-op with a friendly message — depends on T029, T030
- [X] T033 [US4] In `internal/tui/view.go` (and `update.go`): render the date prompt and show `p`/`ctrl+p` in the Tasks-tab help — depends on T030
- [X] T034 [US4] TUI tests in `internal/tui/update_test.go`: `p` adds an untimed entry to today; `ctrl+p` accept-tomorrow adds to tomorrow; invalid date and Esc add nothing — depends on T031, T032

**Checkpoint**: All four user stories independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T035 [P] Run the full suite: `cd services/twig && go test ./...`
- [X] T036 [P] Tone review (Principle IV) of all new user-facing copy: CLI help, send confirmations, invalid-date / empty-selection messages
- [X] T037 [P] Consistency review (Principle III): untimed pane styling matches grid boxes; `p`/`ctrl+p`/`null` follow existing conventions
- [X] T038 Verify the server builds and the migration applies cleanly (`make dev`; or `make migrate-up` with `DATABASE_URL`)
- [ ] T039 Execute `quickstart.md` end-to-end (CLI + TUI manual verification)

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (P1)** → **Foundational (P2)** → **User Stories (P3–P6)** → **Polish (P7)**.
- Foundational is the contract + storage + handler; it **blocks all stories**.

### Cross-story dependencies (note: several stories touch shared TUI files, so they are not freely parallel)

- **US2 (T019)** depends on **US1 (T013)** for the shared `RenderUntimed` renderer.
- **US4 (T031/T032)** depends on **US2 (T021)** for the optional-start `addPlanTaskCmd`.
- **US3 (T026)** and **US4 (T031/T032)** both edit `internal/tui/plan_update.go` / `update.go` — sequence them (priority order) rather than running in parallel.
- **US1, US3 CLI parts** (`plan.go`) depend only on Foundational and on `ParseStartOrNull` (T012).

### Within each story

- Helpers/renderers before the views that use them; commands/handlers before the tests that exercise them.

---

## Parallel Opportunities

- **Foundational**: T004 (migration) and T005 (queries) are `[P]`; T011 (handler tests) `[P]` after T007–T010.
- **US1**: T012 (timeparse) and T013 (renderer) are `[P]` — different files.
- **US2**: T018 (split helper) is `[P]`.
- **US3**: T024 (CLI `mv`) is `[P]` relative to the TUI tasks.
- **US4**: T029 (keymap) is `[P]`.
- **Polish**: T035–T037 are `[P]`.

```bash
# Foundational parallel batch (after T003):
Task: "Add migration 000007 up/down per data-model.md"          # T004
Task: "Edit db/queries/plan.sql nullable start + NULLS FIRST"   # T005

# US1 parallel batch:
Task: "Add ParseStartOrNull in internal/cli/timeparse/"          # T012
Task: "Add RenderUntimed in internal/cli/plan_grid.go"           # T013
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 Setup → Phase 2 Foundational (the whole API/storage/handler change).
2. Phase 3 US1 → **STOP and validate**: untimed capture + listing from the CLI. This is a shippable MVP.

### Incremental Delivery

1. Foundational ready → US1 (CLI) MVP.
2. US2 → untimed entries visible/manageable in the TUI.
3. US3 → schedule/unschedule from CLI and TUI.
4. US4 → one-key send from the Tasks tab.
5. Polish → full test run, tone/consistency review, quickstart.

### Notes

- `[P]` = different files, no incomplete dependency.
- Regenerated code (`gen/`, `internal/db/`) changes only via T003 (`make proto`) and T006 (`sqlc generate`).
- Commit after each task or logical group; stop at any checkpoint to validate a story independently.
