---

description: "Task list for feature 006 — Daily Planning"
---

# Tasks: Daily Planning

**Input**: Design documents from `/specs/006-daily-planning/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/plan.md, quickstart.md

**Tests**: Included. plan.md's Testing section enumerates required handler and CLI tests; this feature follows the project's existing pattern (test files committed alongside code under each package).

**Organization**: Tasks are grouped by user story. Story IDs map to spec.md: **US1** = Build a daily plan from tasks and events (P1), **US2** = View the daily plan (P1), **US3** = Modify entries (P2), **US4** = Clear the rest of the day (P3).

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Different file, no dependencies on incomplete tasks — safe to run in parallel.
- **[Story]**: User story label (US1, US2, US3, US4). Setup, Foundational, and Polish phases carry no story label.
- Paths are repository-relative.

## Path Conventions

Web service + CLI in one Go module under `services/todo/`. All source paths are relative to the repo root.

---

## Phase 1: Setup

**Purpose**: No new dependencies are required for this feature; this phase is intentionally tiny.

- [X] T001 Confirm the working tree is on branch `006-daily-planning` and that `services/todo/` builds cleanly (`go build ./...`) before any changes.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema, generated code, shared package, time-parse helper, and the CLI/handler scaffolding that every user story builds on.

**⚠️ CRITICAL**: All four user stories depend on this phase. Do not start any user-story phase until every task here is checked.

### Database

- [X] T002 Create `services/todo/db/migrations/000006_plan_entries.up.sql` exactly as written in `specs/006-daily-planning/data-model.md` (composite PK `(user_id, day, id)`; CHECKs for `start_minute` range, `duration_minute` positive, midnight-crossing, and name-or-task; `task_id` FK with `ON DELETE CASCADE`; `plan_entries_task_id_idx`).
- [X] T003 [P] Create `services/todo/db/migrations/000006_plan_entries.down.sql` that runs `DROP TABLE IF EXISTS plan_entries;`.
- [X] T004 Apply the migration locally and verify in psql (`\d plan_entries`) that the composite PK and all three CHECK constraints are present.

### sqlc queries

- [X] T005 Create `services/todo/db/queries/plan.sql` with the following named queries (see plan.md Storage section):
    - `ListPlanEntriesForDay :many` — `SELECT * FROM plan_entries WHERE user_id = $1 AND day = $2 ORDER BY start_minute`
    - `GetPlanEntry :one` — `SELECT * FROM plan_entries WHERE user_id = $1 AND day = $2 AND id = $3`
    - `LockPlanEntriesForDay :many` — `SELECT * FROM plan_entries WHERE user_id = $1 AND day = $2 FOR UPDATE`
    - `NextPlanEntryId :one` — `SELECT COALESCE(MAX(id), 0) + 1 AS next_id FROM plan_entries WHERE user_id = $1 AND day = $2`
    - `InsertPlanEntry :one` — `INSERT INTO plan_entries (user_id, day, id, task_id, name, start_minute, duration_minute) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING *`
    - `UpdatePlanEntryName :one` — `UPDATE plan_entries SET name = $4 WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING *`
    - `UpdatePlanEntryTime :one` — `UPDATE plan_entries SET start_minute = $4, duration_minute = $5 WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING *`
    - `DeletePlanEntry :one` — `DELETE FROM plan_entries WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING id`
    - `DeletePlanEntriesFromMinute :execrows` — `DELETE FROM plan_entries WHERE user_id = $1 AND day = $2 AND start_minute >= $3`
    - `TrimPlanEntryDuration :one` — `UPDATE plan_entries SET duration_minute = $4 WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING *`
- [X] T006 Run `sqlc generate` from `services/todo/`; verify `internal/db/models.go` gains a `PlanEntry` struct and that `internal/db/plan.sql.go` is generated with all ten query functions.

### Proto contract & generated bindings

- [X] T007 Create `services/todo/proto/plan/v1/plan.proto` exactly as written in `specs/006-daily-planning/contracts/plan.md` (the `PlanEntry` message, the `PlanService` with seven RPCs, and the seven request/response message pairs). Verify `option go_package = "github.com/pboyd/todo/services/todo/gen/plan/v1;planv1";`.
- [X] T008 Update `services/todo/buf.gen.yaml` (or equivalent) if needed so the new `plan/v1` proto is picked up, then run `buf generate` from `services/todo/`; verify `gen/plan/v1/plan.pb.go` and `gen/plan/v1/planv1connect/...` are generated and compile.

### Shared package

- [X] T009 [P] Create `services/todo/internal/plan/plan.go` exporting `func Overlap(aStart, aDur, bStart, bDur int) bool { return aStart < bStart+bDur && bStart < aStart+aDur }` (half-open `[start, start+dur)` semantics — touching is NOT an overlap). Add `services/todo/internal/plan/plan_test.go` with table-driven cases covering: full overlap, partial overlap (front and back), exact touching at the boundary (both directions — must return false), and fully disjoint intervals.

### CLI time-parse helper

- [X] T010 [P] Create `services/todo/internal/cli/timeparse/timeparse.go` exporting `ParseStart(s string) (int, error)` (returns minutes since midnight) and `ParseDuration(s string) (int, error)` (returns minutes). `ParseStart` must accept `"13:15"` (24-hour), `"1:15pm"` / `"1:15PM"` / `"1:15 pm"` / `"01:15 PM"` (12-hour with optional space, case-insensitive meridiem). `ParseDuration` must accept `"90m"`, `"1h"`, `"2h"`, `"1h30m"`, `"0h45m"`. Reject anything outside these forms with a clear error.
- [X] T011 [P] Create `services/todo/internal/cli/timeparse/timeparse_test.go` with table-driven tests asserting that the documented variants of `ParseStart` all map to the same minute (e.g. all three of `"13:15"`, `"1:15pm"`, `"01:15 PM"` produce `795`) and that the documented variants of `ParseDuration` map correctly (`"90m"`=`"1h30m"`=`90`; `"2h"`=`120`). Include error cases: empty string, junk, hour > 23, minute > 59, missing unit.

### Handler scaffolding

- [X] T012 Create `services/todo/internal/handler/plan.go` with the new `Plan` handler struct (or `Plan` methods on the existing service holder — match the pattern used by `Task` in `internal/handler/task.go`). Add: (a) `func dbPlanEntryToProto(e db.PlanEntry) *planv1.PlanEntry` helper (formats `day` as `YYYY-MM-DD`); (b) `parseDay(s string) (time.Time, error)` helper that validates the `YYYY-MM-DD` format and returns a `time.Time` at local midnight; (c) `ListPlanEntries` method that calls `Queries.ListPlanEntriesForDay` and returns the rows via the helper. Wire the new service into the Connect mux in `cmd/server/main.go` alongside `TaskService`.
- [X] T013 [P] Create `services/todo/internal/handler/plan_test.go` with a `ListPlanEntries` test: insert two entries directly via `Queries.InsertPlanEntry` (one with task_id, one with name only); call `ListPlanEntries` for that day; assert both rows are returned in `start_minute` ascending order with correct `day`/`id`/`task_id`/`name`/`start_minute`/`duration_minute`. Add an isolation test that another user's entries on the same day are NOT returned.

### CLI scaffolding

- [X] T014 In `services/todo/internal/cli/cli.go`, register a new `case "plan":` branch in the top-level dispatcher that calls `runPlan(client, args[1:])` (mirroring how `task` is dispatched). Update the top-level usage string to include `plan`.
- [X] T015 Create `services/todo/internal/cli/plan.go` with: (a) a `runPlan(client, args)` dispatcher that first consumes the optional `--date YYYY-MM-DD` flag (and falls back to today's local date when absent), then switches on the next positional arg over `task|event|rm|rename|mv|clear`, defaulting (no subcommand) to a `runPlanShow` stub; (b) a `pomPlanUsage()` style helper printing the full `plan` usage; (c) every subcommand handler is a stub (`return 1` with "not implemented") for now; later phases fill them in. Add a `planv1` client field to the existing CLI client holder (or add a parallel `planClient` parameter) matching the existing pattern from `task` for Connect client wiring.

**Checkpoint**: Migration applied; generated code rebuilt; `internal/plan.Overlap` and `internal/cli/timeparse` packages exist and tested; `ListPlanEntries` works; CLI dispatches `todo plan ...` to stubs. User-story phases can now begin.

---

## Phase 3: User Story 1 — Build a daily plan from tasks and events (Priority: P1) 🎯 MVP

**Goal**: A user can run `todo plan task <id> <start> [dur]` to schedule work on a task and `todo plan event <name> <start> [dur]` to block off other time. Each call assigns a sequential per-day id, rejects overlap (with touching allowed), and applies the documented default durations.

**Independent Test**: With a known task ID, run `todo plan task 5 9:00am` (expect `→ entry 1`), then `todo plan event "Lunch" 12:00pm 45m` (expect `→ entry 2`). Connect to the database and verify both `plan_entries` rows exist for today's date with correct minutes and that overlapping a third entry across either is rejected.

### Implementation for User Story 1

- [X] T016 [US1] Add `AddPlanEvent` method to `services/todo/internal/handler/plan.go`: validate `day` parses; validate `start_minute` in `[0, 1440)`; validate `duration_minute >= 0` and apply the default-30 when 0; validate trimmed `name` is non-empty and ≤255 chars; in one `pgx.BeginTx` (with `pgx.TxOptions{IsoLevel: pgx.Serializable}`) run `LockPlanEntriesForDay`, iterate the locked rows calling `plan.Overlap` against the proposed interval, and return Connect `FailedPrecondition` (message names the conflicting `id`) on any overlap; otherwise call `NextPlanEntryId` then `InsertPlanEntry` (with `task_id = NULL`) and commit; return the inserted row via `dbPlanEntryToProto`. Wire into the Connect handler registration.
- [X] T017 [US1] Add `AddPlanTask` method to `services/todo/internal/handler/plan.go`: validate `day` and `start_minute` as above; if the client sends `duration_minute = 0`, look up the task (using `Queries.GetTask` + `Queries.CountCompletedPomodorosForTask` — both exist from feature 005) and compute `(estimate - completed) * 30` when `estimate > completed`, else default to `30`; verify the task exists for this user before computing default (else `NotFound`); then run the same serializable-tx-locked overlap check as `AddPlanEvent` and `InsertPlanEntry` (with `name = NULL`, `task_id = $taskID`); return the row.
- [X] T018 [P] [US1] Extend `services/todo/internal/handler/plan_test.go` with `AddPlanEvent` tests: happy path with explicit `duration_minute`; default-30 when `duration_minute = 0`; rejection of empty `name`; rejection of out-of-range `start_minute`; rejection of `start + duration > 1440` (midnight crossing); `FailedPrecondition` on overlap (assert error message contains the conflicting `id`); touching-boundary allowed (entry A 10:00–11:00 and B 11:00–12:00 both succeed); id allocation produces `1`, then `2`, then after `DeletePlanEntry` of id 2 the next insert gets `3` (FR-003: no renumbering).
- [X] T019 [P] [US1] Extend `services/todo/internal/handler/plan_test.go` with `AddPlanTask` tests: happy path with explicit duration; default duration from `(estimate - completed) * 30` when both are present and remaining > 0 (set `tasks.estimate = 3`, insert one completed pomodoro, assert resulting `duration_minute = 60`); default 30 when task has `estimate = 0`; default 30 when remaining ≤ 0 (estimate already met); `NotFound` for unknown task; `NotFound` when the task belongs to another user; same overlap/touching/midnight-crossing assertions as US1's event tests.
- [X] T020 [P] [US1] Add a concurrency test to `services/todo/internal/handler/plan_test.go`: two goroutines call `AddPlanEvent` simultaneously for the same `(user_id, day)` with identical start_minute/duration; assert exactly one returns nil and the other returns `FailedPrecondition`; assert the database contains exactly one row for that day. Verifies the serializable-tx invariant.
- [X] T021 [US1] Implement `runPlanTask` in `services/todo/internal/cli/plan.go`: parse `<task_id>` (int), `<start>` (via `timeparse.ParseStart`), optional `[duration]` (via `timeparse.ParseDuration`, default 0 meaning "let the server decide"); call `AddPlanTask`; on success print `→ entry <id>` and exit 0; on `NotFound` print `task not found` and exit 1; on `FailedPrecondition` print the server's message verbatim (it already names the conflicting id) and exit 1; on `InvalidArgument` likewise.
- [X] T022 [US1] Implement `runPlanEvent` in `services/todo/internal/cli/plan.go`: parse `<name>` (single positional, may contain spaces if quoted by the shell), `<start>`, optional `[duration]` (default 0 meaning "use server's 30-minute default"); call `AddPlanEvent`; on success print `→ entry <id>` and exit 0; errors handled as in `runPlanTask`.
- [X] T023 [P] [US1] Create `services/todo/internal/cli/plan_test.go` and add: a `runPlanTask` happy-path test with a fake `planv1` client; a `runPlanEvent` happy-path test; argument-parse failures (non-integer `task_id`, malformed start, malformed duration) print usage and exit 1; server `FailedPrecondition` and `NotFound` are surfaced with non-zero exit; `--date YYYY-MM-DD` overrides today and is forwarded as the `day` field on the RPC.

**Checkpoint**: `todo plan task` and `todo plan event` work end-to-end against the live server. The grid view is not yet implemented (the default `plan` invocation is still a stub) — that lands in US2 and completes the MVP.

---

## Phase 4: User Story 2 — View the daily plan (Priority: P1) 🎯 MVP

**Goal**: A user can run `todo plan` (today) or `todo plan --date YYYY-MM-DD` and see an ASCII day-planner grid bracketed by the first/last entries, with intermediate gaps rendered as empty rows. Entries with a linked task and no explicit `name` display the task's name.

**Independent Test**: With two known entries on today's plan (one task-linked with no explicit name, one event), run `todo plan` and verify the grid starts at the earlier entry's start time, ends at the later entry's end time, omits hours outside that span, shows a gap row for the empty interval between them, and the task-linked entry displays the task's name.

### Implementation for User Story 2

- [X] T024 [US2] Extend `services/todo/internal/handler/plan.go` so `ListPlanEntries` also populates `name` from the task when the row's `name IS NULL`: after fetching the day's rows, collect the distinct non-zero `task_id`s, call `Queries.GetTask` (or a batched variant if convenient) for each to retrieve the task name, and substitute it into the returned `PlanEntry.name` field. This keeps the substitution logic on the server so all clients see the same display value.
- [X] T025 [P] [US2] Extend `services/todo/internal/handler/plan_test.go` with a `ListPlanEntries` test asserting name fallback: insert one row with `name = NULL` and a `task_id`, insert one row with `name = 'Lunch'` and no `task_id`; call `ListPlanEntries`; assert the first row's returned `name` equals the task's name and the second row's `name` is `'Lunch'`.
- [X] T026 [US2] Create `services/todo/internal/cli/plan_grid.go` exporting `RenderGrid(entries []*planv1.PlanEntry) string`: if `entries` is empty, return `"Plan is empty.\n"`; otherwise compute `floor = min(start_minute)` rounded down to the nearest 15-minute mark, `ceil = max(start_minute + duration_minute)` rounded up to the nearest 15-minute mark; for each 15-minute row from `floor` to `ceil`, emit `HH:MM │ <cell>`, where `<cell>` is the entry id and name for rows that fall inside an entry's `[start, start+duration)` window (with a `░` continuation marker on rows after the entry's first), and empty for gap rows. Use a `~` prefix on the time label when an entry begins between two 15-minute boundaries (e.g. `~10:05`) per research.md §7.
- [X] T027 [P] [US2] Create `services/todo/internal/cli/plan_grid_test.go` with golden-output tests: empty-plan message; single entry exactly on a quarter-hour; two entries with a gap (verifies bracketing and gap rendering); off-quarter-hour entry (verifies the `~` prefix); task-linked entry with no explicit name (verifies the server-substituted name shows up — this is a unit test of the renderer over a hand-constructed `[]*planv1.PlanEntry`).
- [X] T028 [US2] Implement `runPlanShow` in `services/todo/internal/cli/plan.go` (the no-subcommand default): call `ListPlanEntries(day)` against the resolved `day`; pass the response's `entries` into `RenderGrid`; write the rendered string to stdout; exit 0. Errors from the RPC print a clear message and exit 1.
- [X] T029 [P] [US2] Extend `services/todo/internal/cli/plan_test.go` with: a `runPlanShow` happy-path test using a fake client that returns a fixed list of entries (asserts stdout starts with the expected first-entry time row); an empty-plan test (asserts the empty-plan message is printed); a `--date` test (asserts the fake client's `ListPlanEntries` was invoked with the requested day).

**Checkpoint**: The MVP is shippable here. `todo plan task`, `todo plan event`, and `todo plan` collectively deliver Acceptance Scenarios 1–6 of US1 and 1–4 of US2. The full quickstart.md steps 1–3 succeed.

---

## Phase 5: User Story 3 — Modify entries on the plan (Priority: P2)

**Goal**: After building a plan, the user can `rm`, `rename`, or `mv` individual entries by id and see the changes reflected on the next `todo plan` render.

**Independent Test**: Build a three-entry plan, then `rm 2`, `rename 1 "Deep work"`, and `mv 3 14:00 45m`. Run `todo plan` and verify: entry 2 is absent (no renumbering), entry 1's name is "Deep work", entry 3 now begins at 14:00 with a 45-minute duration. Attempt `mv 1 15:00` where it would overlap entry 3 — confirm the move is rejected and entry 1 is unchanged.

### Implementation for User Story 3

- [X] T030 [US3] Add `RemovePlanEntry` method to `services/todo/internal/handler/plan.go`: call `Queries.DeletePlanEntry`; if no row was affected (pgx returns `ErrNoRows` from the `RETURNING id`), return Connect `NotFound`; otherwise return an empty response. No transaction needed (single statement).
- [X] T031 [US3] Add `RenamePlanEntry` method to `services/todo/internal/handler/plan.go`: validate trimmed `name` is non-empty and ≤255 chars (else `InvalidArgument`); call `Queries.UpdatePlanEntryName`; map `ErrNoRows` to `NotFound`; return the updated row.
- [X] T032 [US3] Add `MovePlanEntry` method to `services/todo/internal/handler/plan.go`: validate `start_minute` in `[0, 1440)`; in one serializable tx, load the existing row via `Queries.GetPlanEntry` (NotFound mapping) — record its current `duration_minute`; if `duration_minute` from the request is 0, keep the existing value, otherwise use the request value; validate `start + dur <= 1440` (else `InvalidArgument`); load `LockPlanEntriesForDay`, filter out the row being moved (by id), check `plan.Overlap` against each remaining row, return `FailedPrecondition` (with conflicting id) if any overlap; call `Queries.UpdatePlanEntryTime` with the chosen values; commit; return the updated row.
- [X] T033 [P] [US3] Extend `services/todo/internal/handler/plan_test.go` with `RemovePlanEntry` tests: happy path; `NotFound` for unknown id; isolation (another user's entry on same `(day, id)` is not removed); FR-003 check (after removing id 2 from `[1, 2, 3]`, a subsequent `AddPlanEvent` gets id 4, not 2).
- [X] T034 [P] [US3] Extend `services/todo/internal/handler/plan_test.go` with `RenamePlanEntry` tests: happy path; `InvalidArgument` for empty/whitespace name; `InvalidArgument` for name > 255 chars; `NotFound` for unknown id; rename of a previously task-linked entry (verifies the explicit name overrides the task's name on subsequent `ListPlanEntries`).
- [X] T035 [P] [US3] Extend `services/todo/internal/handler/plan_test.go` with `MovePlanEntry` tests: happy path (start and duration both updated); duration-omitted (`duration_minute = 0` in request) preserves prior duration; rejects move that would overlap another entry (with the moved entry's old slot now vacated — verifies the "filter out self" logic); accepts a move that touches another entry at the boundary; rejects move past midnight; `NotFound` for unknown id; isolation (another user's entry is not movable).
- [X] T036 [US3] Implement `runPlanRm` in `services/todo/internal/cli/plan.go`: parse `<n>` (int); call `RemovePlanEntry`; success exits 0 with `removed entry <n>` (or similar); `NotFound` prints `entry not found` and exits 1.
- [X] T037 [US3] Implement `runPlanRename` in `services/todo/internal/cli/plan.go`: parse `<n>` and `<name>` (single positional name); call `RenamePlanEntry`; success prints `renamed entry <n>` and exits 0; map error codes as in `runPlanRm`.
- [X] T038 [US3] Implement `runPlanMv` in `services/todo/internal/cli/plan.go`: parse `<n>`, `<start>` (via `timeparse.ParseStart`), optional `[duration]` (via `timeparse.ParseDuration`; absent → send `0` meaning "keep current"); call `MovePlanEntry`; success prints `moved entry <n>` and exits 0; `FailedPrecondition` prints the server's message verbatim and exits 1.
- [X] T039 [P] [US3] Extend `services/todo/internal/cli/plan_test.go` with `runPlanRm` / `runPlanRename` / `runPlanMv` tests using a fake `planv1` client: happy paths, NotFound/invalid-argument surfacing, the `mv` duration-omitted case forwards `duration_minute = 0` on the RPC.

**Checkpoint**: All three modify subcommands work end-to-end. Quickstart steps 3 and 4 succeed. Acceptance Scenarios 1–6 of US3 pass.

---

## Phase 6: User Story 4 — Clear the rest of the day (Priority: P3)

**Goal**: A user can run `todo plan clear` (defaults to current local time) or `todo plan clear <start>` to remove every entry starting at or after `start`, while shortening any entry that straddles `start` so it ends at `start`.

**Independent Test**: With entries at 09:00–10:00, 11:00–12:00, and 13:00–14:00 and current time around 11:30, run `todo plan clear`: the 13:00 entry is removed, the 11:00 entry is shortened to 11:00–11:30 (`trimmed_straddling_entry = true, deleted_count = 1`), the 09:00 entry is untouched. Then run `todo plan clear 09:00` and verify all entries are removed (deleted_count = 2, trimmed = false because the 09:00–09:30 entry starts exactly at the cutoff so it is fully deleted, not trimmed).

### Implementation for User Story 4

- [X] T040 [US4] Add `ClearPlan` method to `services/todo/internal/handler/plan.go`: validate `day` and `start_minute`; in one serializable tx, call `LockPlanEntriesForDay`; identify the (at most one) row that straddles — i.e. `start < start_minute < start + duration_minute`; if such a row exists, call `Queries.TrimPlanEntryDuration` with `start_minute - row.start_minute` as the new duration; then call `Queries.DeletePlanEntriesFromMinute` and capture its returned rowcount; commit; return `{ deleted_count: rowcount, trimmed_straddling_entry: <bool> }`.
- [X] T041 [P] [US4] Extend `services/todo/internal/handler/plan_test.go` with `ClearPlan` tests: empty-day no-op (returns `{0, false}`); cutoff before all entries removes everything (`{n, false}`); cutoff after all entries removes nothing; cutoff exactly at an entry's start removes that entry without trimming (FR-016 says "starting at or after" — exact-start is a deletion, not a trim); cutoff in the middle of an entry trims and removes later entries (`{laterCount, true}`); concurrent modify (a second goroutine adding an entry while `ClearPlan` runs) does not lose or duplicate rows — verifies the serializable tx.
- [X] T042 [US4] Implement `runPlanClear` in `services/todo/internal/cli/plan.go`: optional positional `<start>` (via `timeparse.ParseStart`; when absent compute "now" as the local clock's `hour*60 + minute` on the target `day`); call `ClearPlan`; print `Cleared <deleted_count> entries; trimmed <0|1> straddling entries.` and exit 0; on `InvalidArgument` print the server message and exit 1.
- [X] T043 [P] [US4] Extend `services/todo/internal/cli/plan_test.go` with `runPlanClear` tests: no-arg path (asserts the call uses the current local minute — inject a fake clock via the existing CLI clock seam, or stub the relevant helper); explicit-start path; output format check (the exact "Cleared … trimmed …" line); error surfacing on `InvalidArgument`.

**Checkpoint**: All four user stories are independently functional. The full quickstart.md script runs end-to-end.

---

## Phase 7: Polish & Cross-Cutting

**Purpose**: Final cleanup, full quickstart pass, lint/vet.

- [X] T044 [P] Run `go vet ./...` and `gofmt -l` from `services/todo/`; resolve any findings.
- [X] T045 [P] Run `go test ./...` from `services/todo/`; the full handler + CLI suite (including the new tests from T009/T011/T013/T018–T020/T023/T025/T027/T029/T033–T035/T039/T041/T043) must pass against a fresh test database with migration `000006` applied.
- [X] T046 Run through `specs/006-daily-planning/quickstart.md` against a freshly-migrated local instance; check off each section. Capture any deviations as follow-up issues, do not edit the spec/plan.
- [X] T047 [P] Verify SC-002 manually: attempt three different overlap variations (full overlap, half-overlap front, half-overlap back) from the CLI and confirm each one prints an error that names the conflicting entry id. Verify the touching boundary case (back-to-back entries) is accepted.
- [X] T048 [P] Verify SC-003 manually: from the CLI run `todo plan event "x" 13:15 90m` followed by `todo plan event "y" 1:15pm 1h30m` on a fresh day — the second call must fail with overlap referencing the first (proves both start formats parsed to the same minute; both duration formats parsed to 90).

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 (Setup)**: no dependencies.
- **Phase 2 (Foundational)**: depends on Phase 1. Blocks all user-story phases. Internal ordering: T002 → T004 (apply); T005 → T006 (sqlc regen); T007 → T008 (buf regen); T009/T010/T011 independent of the above; T012/T013 depend on T006 and T008; T014/T015 depend on T008 and T010.
- **Phase 3 (US1)**: depends on Phase 2.
- **Phase 4 (US2)**: depends on Phase 2; server-side test in T025 conceptually depends on US1's seed data, but it inserts rows directly via `Queries`, so US2 handler/CLI work can proceed in parallel with US1. Note: the **MVP** requires both US1 and US2 to land (both P1).
- **Phase 5 (US3)**: depends on Phase 2; independent of US1/US2/US4. Tests use direct `Queries` inserts.
- **Phase 6 (US4)**: depends on Phase 2; independent of US1/US2/US3.
- **Phase 7 (Polish)**: depends on all user stories you intend to ship.

### Within each story

- Server handler and its unit tests can be developed together, per the project's established convention.
- CLI subcommands depend on their corresponding server handlers being wired into the Connect registration.
- `plan_grid.go` (US2) is independent of any RPC and can be developed before or during US2's handler work.

### Parallel opportunities

- T002 and T003 are different files: parallelizable.
- T009, T010, T011, T013 are different files: parallelizable.
- T018/T019/T020 are all additions to `plan_test.go`. They are listed `[P]` because they touch disjoint test functions; sequence them in numeric order if your team avoids same-file simultaneous edits.
- T025, T027, T029, T033, T034, T035, T039, T041, T043 — same caveat; same-file but disjoint functions.
- US3 (T030–T039) and US4 (T040–T043) are independent of US1/US2 once Phase 2 is done; a separate developer can take either.

---

## Parallel Example: Phase 2 Foundational

```bash
# Once T006 (sqlc generate) and T008 (buf generate) have landed:
Task: "T009 — create services/todo/internal/plan/plan.go (+ test) for the Overlap helper"
Task: "T010 — create services/todo/internal/cli/timeparse/timeparse.go"
Task: "T011 — create services/todo/internal/cli/timeparse/timeparse_test.go"
Task: "T013 — create services/todo/internal/handler/plan_test.go with ListPlanEntries tests"
```

## Parallel Example: User Story 1

```bash
# After T016, T017, T021, T022 land, the test files can be filled in parallel:
Task: "T018 — extend plan_test.go with AddPlanEvent tests"
Task: "T019 — extend plan_test.go with AddPlanTask tests"
Task: "T020 — extend plan_test.go with concurrent-add race test"
Task: "T023 — create plan_test.go (CLI) with runPlanTask/runPlanEvent tests"
```

---

## Implementation Strategy

### MVP first (User Stories 1 + 2)

Both US1 and US2 are P1; neither is useful alone. The MVP is therefore:

1. Phase 1 (Setup) + Phase 2 (Foundational).
2. Phase 3 (US1) and Phase 4 (US2) — can be developed in parallel after Phase 2 once the proto is generated; both must land for an end-to-end demo.
3. **Validate**: run quickstart.md sections 1–3 (schedule a task, schedule an event, render the plan).

### Incremental delivery

After MVP:

- Add **US3** (modify) for in-place plan refinement.
- Add **US4** (clear) for end-of-day reset. Independent of US3; can be done in parallel.

### Parallel team strategy

After Phase 2 completes:

- Developer A: US1 (Phase 3) — server adds.
- Developer B: US2 (Phase 4) — list + grid renderer.
- Developer C: US3 (Phase 5) — modify subcommands (independent of A and B).
- US4 is small enough to be a Phase 7-adjacent pickup by any developer.

---

## Notes

- [P] tasks = different files or disjoint test functions, no incomplete-dependency on each other.
- [US?] label maps a task to a specific user story for traceability.
- The MVP is US1+US2 (both P1). US3 and US4 are independent follow-ups.
- Commit after each task or each logical group; the existing project workflow uses small commits.
- Avoid: vague tasks, same-file conflicts that block parallelism, cross-story dependencies that break independence.
