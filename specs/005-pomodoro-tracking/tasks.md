---

description: "Task list for feature 005 — Pomodoro Tracking"
---

# Tasks: Pomodoro Tracking

**Input**: Design documents from `/specs/005-pomodoro-tracking/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/pomodoro.md, quickstart.md

**Tests**: Included. plan.md's Testing section enumerates required handler and CLI tests; this feature follows the project's existing pattern (test files committed alongside code under each package).

**Organization**: Tasks are grouped by user story. Story IDs map to spec.md: **US1** = Run a focused work session (P1), **US2** = Cancel/quit/resume + external-cancel detection (P2), **US3** = Estimate pomodoros (P2), **US4** = Status (P3).

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Different file, no dependencies on incomplete tasks — safe to run in parallel.
- **[Story]**: User story label (US1, US2, US3, US4). Setup, Foundational, and Polish phases carry no story label.
- Paths are repository-relative.

## Path Conventions

Web service + CLI in one Go module under `services/todo/`. All source paths are relative to the repo root.

---

## Phase 1: Setup

**Purpose**: Pull in the one new transitive dependency we need; everything else is already in place from features 001–004.

- [ ] T001 Add `golang.org/x/term` to `services/todo/go.mod` via `go get golang.org/x/term@latest` and commit the resulting `go.mod` / `go.sum` (run from `services/todo/`).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema, generated code, shared package, and the foundational read endpoint and CLI scaffolding that every user story builds on.

**⚠️ CRITICAL**: All four user stories depend on this phase. Do not start any user-story phase until every task here is checked.

### Database

- [ ] T002 Create `services/todo/db/migrations/000005_pomodoros.up.sql` with: `ALTER TABLE tasks ADD COLUMN estimate SMALLINT NOT NULL DEFAULT 0 CHECK (estimate BETWEEN 0 AND 10);` then `CREATE TABLE pomodoros (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE, start_at TIMESTAMPTZ NOT NULL, end_at TIMESTAMPTZ, complete BOOLEAN NOT NULL DEFAULT FALSE, CHECK (end_at IS NULL OR end_at >= start_at), CHECK (NOT complete OR end_at IS NOT NULL));` plus `CREATE UNIQUE INDEX pomodoros_one_active_per_user ON pomodoros (user_id) WHERE end_at IS NULL;` and `CREATE INDEX pomodoros_task_id_idx ON pomodoros (task_id);` (see data-model.md).
- [ ] T003 [P] Create `services/todo/db/migrations/000005_pomodoros.down.sql` that drops `pomodoros`, drops the unique/btree indexes, and drops `tasks.estimate`.
- [ ] T004 Run the migration locally and verify in psql that the column, table, and partial unique index exist (`\d tasks`, `\d pomodoros`).

### sqlc queries

- [ ] T005 Add `SetTaskEstimate` to `services/todo/db/queries/task.sql` (`UPDATE tasks SET estimate = $2 WHERE id = $1 AND user_id = $3 RETURNING *;`). Verify no other query in this file needs changes — existing `RETURNING *` clauses automatically pick up `estimate` after sqlc regen.
- [ ] T006 Create `services/todo/db/queries/pomodoro.sql` with the six queries listed in plan.md / data-model.md: `StartPomodoro` (`INSERT ... RETURNING *`), `CancelActivePomodoro` (`UPDATE pomodoros SET end_at = NOW() WHERE user_id = $1 AND end_at IS NULL RETURNING *`), `CompleteActivePomodoro` (`UPDATE pomodoros SET end_at = start_at + INTERVAL '25 minutes', complete = TRUE WHERE user_id = $1 AND end_at IS NULL RETURNING *`), `GetActivePomodoro` (`SELECT * FROM pomodoros WHERE user_id = $1 AND end_at IS NULL`), `ListPomodorosForTask` (`SELECT * FROM pomodoros WHERE task_id = $1 AND user_id = $2 ORDER BY start_at`), `CountCompletedPomodorosForTask` (`SELECT count(*)::bigint AS count FROM pomodoros WHERE task_id = $1 AND user_id = $2 AND complete`).
- [ ] T007 Run `sqlc generate` from `services/todo/`; verify `internal/db/models.go` gains `Pomodoro` and that `Task` gains `Estimate`; verify `internal/db/pomodoro.sql.go` is generated with the six query functions.

### Proto contract & generated bindings

- [ ] T008 Edit `services/todo/proto/task/v1/task.proto` to match `specs/005-pomodoro-tracking/contracts/pomodoro.md` verbatim: add `int32 estimate = 7;` to `Task`; add `Pomodoro` message; add `SetEstimate`, `StartPomodoro`, `CancelPomodoro`, `CompletePomodoro`, `GetActivePomodoro` RPCs to `service TaskService`; add `completed_pomodoro_count` and `repeated Pomodoro pomodoros` to `GetTaskResponse`; add the five new request/response messages.
- [ ] T009 Run `buf generate` from `services/todo/`; verify `gen/task/v1/task.pb.go` and `gen/task/v1/taskv1connect/...` are regenerated and compile.

### Shared package

- [ ] T010 [P] Create `services/todo/internal/pomodoro/pomodoro.go` exporting `const Length = 25 * time.Minute` and `func Remaining(start, now time.Time) time.Duration { d := start.Add(Length).Sub(now); if d < 0 { return 0 }; return d }`. Add a small `pomodoro_test.go` covering both the constant value (sanity check) and `Remaining` (start in the past, start in the future, exactly at expiry).

### Handler scaffolding

- [ ] T011 Update `dbTaskToProto` in `services/todo/internal/handler/task.go` to set `pt.Estimate = int32(t.Estimate)`. (No other field changes.)
- [ ] T012 Extend `GetTask` in `services/todo/internal/handler/task.go` to, after fetching the row, also call `Queries.CountCompletedPomodorosForTask` and `Queries.ListPomodorosForTask` for the same `(task_id, user_id)`, and populate `completed_pomodoro_count` and `pomodoros` on the `GetTaskResponse` (a new `dbPomodoroToProto` helper lives in `pomodoro.go` from T013 — import it).
- [ ] T013 Create `services/todo/internal/handler/pomodoro.go` with: (a) `func dbPomodoroToProto(p db.Pomodoro) *taskv1.Pomodoro` helper; (b) `GetActivePomodoro` method on `*Task` that runs `Queries.GetActivePomodoro`, returns an empty `GetActivePomodoroResponse` on `pgx.ErrNoRows`, and otherwise wraps the row via `dbPomodoroToProto`; (c) a `pgErrIsUniqueViolation(err error, constraint string) bool` helper that pattern-matches `*pgconn.PgError` with code `23505` and the named constraint. Wire `GetActivePomodoro` into the connect handler registration alongside the other RPCs (see `cmd/server/main.go` for the existing pattern).
- [ ] T014 [P] Extend `services/todo/internal/handler/task_test.go` with a test for `GetTask` asserting that `completed_pomodoro_count` and `pomodoros` are populated correctly: insert a task, insert one completed and one canceled pomodoro for it directly via `Queries`, call `GetTask`, assert count = 1 and len(pomodoros) = 2 in `start_at` order.
- [ ] T015 [P] Create `services/todo/internal/handler/pomodoro_test.go` with a test for `GetActivePomodoro`: (a) returns empty response when no active row exists; (b) returns the active row when one exists; (c) does NOT return another user's active row.

### CLI scaffolding

- [ ] T016 In `services/todo/internal/cli/cli.go`, register a new `case "pom":` branch in `runTask`'s switch that dispatches to `runPom(client, args[1:])`. Update `printTaskUsage` to include the pom subcommands (`pom estimate <task_id> <n>`, `pom start <task_id> [--exec cmd]`, `pom resume [--exec cmd]`, `pom cancel`, `pom status`).
- [ ] T017 Create `services/todo/internal/cli/pom.go` containing a `runPom(client, args)` dispatcher that switches on `args[0]` over `estimate|start|resume|cancel|status` and prints a `pom`-specific usage message for unknown subcommands or empty args. Each subcommand is a stub (`return 1` with "not implemented") for now; later phases fill them in.

**Checkpoint**: Schema migrated, generated code rebuilt, `internal/pomodoro` package exists, `GetTask` returns pomodoro count + list, `GetActivePomodoro` works, CLI dispatches to `pom` stubs. User-story phases can now begin.

---

## Phase 3: User Story 1 — Run a focused work session (Priority: P1) 🎯 MVP

**Goal**: A user can run `task pom start <task_id>`, see a live countdown showing task name, estimate, and previously-completed count, let the 25-minute timer run to expiry, have the session recorded as a completed pomodoro, and optionally have a `--exec` command run on completion.

**Independent Test**: With a known task ID, run `todo task pom start <id>` (with a test-only shortened pomodoro length, see T018), observe the displayed task name / estimate / completed-count, let the timer reach zero, verify the server has a `pomodoros` row with `complete = true` and `end_at = start_at + 25 min`, and (with `--exec`) verify the command ran.

### Implementation for User Story 1

- [ ] T018 [US1] Add `StartPomodoro` method to `services/todo/internal/handler/pomodoro.go`: validate the requested task exists for the user (call `Queries.GetTask`; `NotFound` if missing); call `Queries.StartPomodoro`; on unique-violation (`pomodoros_one_active_per_user`) return `AlreadyExists` with a Connect error detail containing the conflicting `task_id` (look it up via `GetActivePomodoro`); on success return the new row via `dbPomodoroToProto`. Wire into the Connect handler registration.
- [ ] T019 [US1] Add `CompletePomodoro` method to `services/todo/internal/handler/pomodoro.go`: call `Queries.CompleteActivePomodoro`; on `pgx.ErrNoRows` return `FailedPrecondition` "no active pomodoro"; otherwise return the updated row. Wire into the Connect handler registration.
- [ ] T020 [P] [US1] Extend `services/todo/internal/handler/pomodoro_test.go` with `StartPomodoro` tests: happy path; `NotFound` for unknown task; `NotFound` when the task belongs to another user; `AlreadyExists` when the user already has an active pomodoro; verifies `start_at` is set and `end_at`/`complete` are null/false.
- [ ] T021 [P] [US1] Extend `services/todo/internal/handler/pomodoro_test.go` with `CompletePomodoro` tests: happy path (asserts `end_at == start_at + 25 min` exactly and `complete = true`); `FailedPrecondition` when no active pomodoro exists; verifies a completed pomodoro is no longer considered active by `GetActivePomodoro`.
- [ ] T022 [US1] Create `services/todo/internal/cli/pom_countdown.go` with: (a) a `countdownDeps` struct holding interfaces for `Now() time.Time`, `Sleep(time.Duration)`, `ReadKey() (byte, error)` returning 0 on no-input, `Render(state countdownState)`, plus the task-service client; (b) `runCountdown(ctx, deps, activePomodoro, task)` that loops on a 1-second tick rendering remaining time computed via `pomodoro.Remaining(activePomodoro.StartAt.AsTime(), deps.Now())` until remaining hits 0; (c) the real implementations of those deps using `time.Now`, `time.Sleep`, and a stdio renderer. Keystroke and external-cancel polling are added in US2 — for now `ReadKey` always returns 0 and the loop exits cleanly on remaining=0 returning a "completed" outcome.
- [ ] T023 [US1] Implement `runStart` in `services/todo/internal/cli/pom.go`: parse `<task_id>` (positional) and `--exec <cmd>` (optional); call `GetTask` (used for the displayed name + estimate + completed_pomodoro_count); call `StartPomodoro`; pass the returned active pomodoro plus the task into `runCountdown`; on the "completed" outcome call `CompletePomodoro`; if `--exec` was set, invoke it (via T024); return exit 0. Errors from RPCs map to printable messages + exit 1. Restart/resume and other-task-active prompts are added in US2 — for now, surface the raw `AlreadyExists` error to the user and exit 1.
- [ ] T024 [P] [US1] Add an `execHook(cmd string) error` function in `services/todo/internal/cli/pom.go` that runs `sh -c <cmd>` with `os.Stdin`, `os.Stdout`, `os.Stderr` inherited. The function returns an error wrapping the exit code when the command fails. The caller (`runStart`) prints `warning: --exec command exited with status N: <cmd>` to stderr when this returns an error but still returns 0 from `runStart`.
- [ ] T025 [P] [US1] Create `services/todo/internal/cli/pom_countdown_test.go`: use a fake `countdownDeps` with a stepped clock and a no-op renderer; assert the loop runs the expected number of iterations for a given start time and exits with "completed" outcome at expiry. Cover both the "exactly at expiry" boundary and a stale start_at (already-expired) starting state.
- [ ] T026 [P] [US1] Create `services/todo/internal/cli/pom_test.go` and add: a `runStart` test that uses a fake task-service client (returning a known task and a successful `StartPomodoro`/`CompletePomodoro`) plus the fake `countdownDeps` — asserts that `CompletePomodoro` was called and that exit is 0. Add a separate `execHook` test running `true` (expect nil error) and `false` (expect non-nil with non-zero exit code). Add a `runStart --exec` test asserting the warning is printed when the exec'd command fails but the function still returns 0.

**Checkpoint**: `todo task pom start <id>` works end-to-end — server records a completed pomodoro after the timer expires; `--exec` runs synchronously with inherited stdio. The MVP is shippable here, even though cancel/resume/status/estimate are not yet implemented.

---

## Phase 4: User Story 2 — Cancel, quit, resume, and external-cancel detection (Priority: P2)

**Goal**: While a pomodoro is running the user can press `c` to cancel or `q` to quit (without stopping the timer). They can later `task pom resume` to re-attach the countdown to the still-active pomodoro. If another terminal cancels the pomodoro the running countdown notices within ~1 s and exits cleanly. Trying to `start` while another pomodoro is active triggers the restart/resume (same task) or cancel-and-start (different task) prompts.

**Independent Test**: With a known task ID, run start; press `c` and verify the row is canceled (end_at set, complete false). Run start again; press `q`, then run resume in another terminal and verify the countdown re-attaches with the correct remaining time. Run start; in another terminal run `task pom cancel`; verify the first terminal's countdown exits within 5 seconds. Run start for task A; in another terminal try `task pom start B` and confirm the cancel-and-start prompt; decline and confirm no record changes.

### Implementation for User Story 2

- [ ] T027 [US2] Add `CancelPomodoro` method to `services/todo/internal/handler/pomodoro.go`: call `Queries.CancelActivePomodoro`; on `pgx.ErrNoRows` return `FailedPrecondition` "no active pomodoro"; otherwise return the updated row. Wire into the Connect handler registration.
- [ ] T028 [P] [US2] Extend `services/todo/internal/handler/pomodoro_test.go` with `CancelPomodoro` tests: happy path (asserts `end_at == NOW()` ± skew, `complete = false`); `FailedPrecondition` when no active pomodoro; verifies post-cancel that the row no longer appears in `GetActivePomodoro` but does appear in `GetTask`'s pomodoros list with `complete = false` and end_at set.
- [ ] T029 [P] [US2] Add a cascade test to `services/todo/internal/handler/pomodoro_test.go`: create a task, start a pomodoro on it, call `DeleteTask`, assert that no `pomodoros` rows remain for that task (query the database directly via `Queries`).
- [ ] T030 [P] [US2] Add a concurrency test to `services/todo/internal/handler/pomodoro_test.go`: spawn two goroutines that each call `StartPomodoro` for the same user (different tasks); use a `sync.WaitGroup`; assert exactly one returns nil and the other returns Connect `AlreadyExists`; assert the database contains exactly one active row.
- [ ] T031 [US2] Extend `services/todo/internal/cli/pom_countdown.go`: implement real keystroke reading using `golang.org/x/term` raw mode (guarded by `term.IsTerminal` — when stdin is not a TTY, `ReadKey` always returns 0 and the loop runs non-interactively); recognize `c`/`C` (returns "canceled" outcome) and `q`/`Q` (returns "quit" outcome); ensure raw-mode state is restored via `defer` even on panic/signal.
- [ ] T032 [US2] Extend `runCountdown` in `services/todo/internal/cli/pom_countdown.go` to also call `GetActivePomodoro` once per tick; if the response is empty or its `id` differs from the attached active pomodoro's id, return an "external" outcome with a flag for "canceled" vs "completed externally" (inferred from whether the new row exists with the same id but `complete` flipped; if no row found at all, treat as canceled-or-deleted). Update the caller chain to print an appropriate message and exit 0 on external outcomes.
- [ ] T033 [US2] Implement keystroke outcomes in `runStart` (services/todo/internal/cli/pom.go): on "canceled" outcome call `CancelPomodoro` and exit 0; on "quit" outcome exit 0 without any API call. Update tests in `pom_test.go` accordingly.
- [ ] T034 [US2] Implement `runResume` in `services/todo/internal/cli/pom.go`: parse `--exec <cmd>` (no positional args per FR-037); call `GetActivePomodoro`; if empty return error "no active pomodoro to resume" and exit 1; if `pomodoro.Remaining(start_at, now) == 0` call `CompletePomodoro`, run `--exec` if set, and exit 0 without rendering a countdown; otherwise call `GetTask` for the active pomodoro's `task_id` and hand off to `runCountdown` exactly as `runStart` does, with the same completion/cancel/quit/external handling.
- [ ] T035 [US2] Implement `runCancel` in `services/todo/internal/cli/pom.go`: takes no positional args (per FR-040); calls `CancelPomodoro`; on success prints a confirmation including the canceled task's id and exits 0; on `FailedPrecondition` prints "no active pomodoro" and exits 1.
- [ ] T036 [US2] Implement the restart/resume prompt path in `runStart`: when `StartPomodoro` returns `AlreadyExists`, parse the conflicting `task_id` from the error detail; if it equals the requested `task_id`, prompt "[r]estart or [s]resume" (FR-035); if it differs, prompt "[c]ancel current pomodoro on task N and start, or [a]bort" (FR-036). Restart = call `CancelPomodoro` then `StartPomodoro` again. Resume = call into `runResume`'s flow. Cancel-and-start = `CancelPomodoro` then `StartPomodoro` again. Abort = exit 1 with the "you have an active pomodoro on task N" message (FR-036 clarification).
- [ ] T037 [P] [US2] Extend `services/todo/internal/cli/pom_countdown_test.go` with: a keystroke test driving the fake `ReadKey` to return `c` and asserting "canceled" outcome; a `q` test asserting "quit" outcome; an external-cancel test where the fake client returns "no active pomodoro" partway through the run and asserts the "external" outcome within one tick.
- [ ] T038 [P] [US2] Extend `services/todo/internal/cli/pom_test.go` with: a `runResume` test for the happy path (delegates to the countdown); a `runResume` stale-active test (start_at is 30 min ago — asserts `CompletePomodoro` is called and no countdown is rendered); a `runResume` no-active test (asserts exit 1, error message); a `runCancel` happy path; a `runCancel` no-active test (exit 1); restart-prompt and cancel-and-start-prompt tests that drive a fake stdin reader through both "yes" and "no" branches.

**Checkpoint**: All of US2's acceptance scenarios pass. The single-active invariant has been stress-tested. Cascade-delete from tasks to pomodoros is verified. The countdown UI is fully interactive and self-detects external state changes.

---

## Phase 5: User Story 3 — Estimate pomodoros for a task (Priority: P2)

**Goal**: A user can set or overwrite a task's estimate via `task pom estimate <task_id> <n>`; values outside 0–10 are rejected with the "break it down further" message.

**Independent Test**: Pick an existing task ID; run `task pom estimate <id> 3`; call `GetTask` (or run start which displays it) and verify estimate is 3. Run `task pom estimate <id> 11` and verify it is rejected with exit 1 and the required message. Run `task pom estimate <id> 5` and verify it overwrites the previous value.

### Implementation for User Story 3

- [ ] T039 [US3] Add `SetEstimate` method to `services/todo/internal/handler/task.go`: validate `0 <= estimate <= 10` and otherwise return `InvalidArgument` with the exact message from contracts/pomodoro.md (`estimate must be between 0 and 10; tasks larger than 10 pomodoros must be broken down further`); call `Queries.SetTaskEstimate`; on `pgx.ErrNoRows` return `NotFound`; otherwise return the updated task via `dbTaskToProto`. Wire into the Connect handler registration.
- [ ] T040 [P] [US3] Extend `services/todo/internal/handler/task_test.go` with `SetEstimate` tests: happy path 0; happy path 10; rejects -1, 11, 100 with the precise error message and `InvalidArgument` code; `NotFound` for an unknown task; `NotFound` when the task belongs to another user; overwrite test (set 3, then set 7, assert final value is 7).
- [ ] T041 [US3] Implement `runEstimate` in `services/todo/internal/cli/pom.go`: parses `<task_id>` and `<n>` (both positional, both integers; print pom usage and exit 1 on parse failure); calls `SetEstimate`; on success prints `set estimate for task <id> to <n>` and exits 0; on `InvalidArgument` prints the server's message (which already contains the "break it down further" guidance) and exits 1; on `NotFound` prints "task not found" and exits 1.
- [ ] T042 [P] [US3] Extend `services/todo/internal/cli/pom_test.go` with `runEstimate` tests: happy path; argument-parse failure (non-integer `n`); server `InvalidArgument` is surfaced verbatim; server `NotFound` produces the friendly message; missing arguments print usage.

**Checkpoint**: `task pom estimate` is fully functional and its output is visible in `task pom start`'s countdown header (from US1) and `task pom status`'s output (from US4).

---

## Phase 6: User Story 4 — Status (Priority: P3)

**Goal**: A user can run `task pom status` to learn whether they have an active pomodoro and, if so, which task and how much time remains.

**Independent Test**: With no active pomodoro, `task pom status` prints "No active pomodoro." and exits 0. After `task pom start <id>`, `task pom status` (run from another terminal) prints the task id, name, start time, and remaining time, and exits 0.

### Implementation for User Story 4

- [ ] T043 [US4] Implement `runStatus` in `services/todo/internal/cli/pom.go`: takes no arguments; calls `GetActivePomodoro`; if empty prints `No active pomodoro.` and exits 0; otherwise calls `GetTask` for the active pomodoro's `task_id`, computes remaining via `pomodoro.Remaining`, and prints `Active pomodoro: task <id> "<name>"\nstarted: <start_at RFC3339>\nremaining: <m:ss>`; exits 0.
- [ ] T044 [P] [US4] Extend `services/todo/internal/cli/pom_test.go` with `runStatus` tests: no-active path (expected exact output); active path (asserts task id and name appear, remaining is in `m:ss` form, exit 0); stale-active path (start_at 30 min ago) — `remaining: 0:00`, still exit 0 (status does not auto-complete; only `resume` does).

**Checkpoint**: All four user stories are independently functional. The full quickstart.md script runs end-to-end.

---

## Phase 7: Polish & Cross-Cutting

**Purpose**: Final cleanup, full quickstart pass, lint/vet.

- [ ] T045 [P] Run `go vet ./...` and `gofmt -l` from `services/todo/`; resolve any findings.
- [ ] T046 [P] Run `go test ./...` from `services/todo/`; the full handler + CLI suite (including the new tests from T014/T015/T020/T021/T025/T026/T028/T029/T030/T037/T038/T040/T042/T044) must pass against a fresh test database with migration 000005 applied.
- [ ] T047 Run through `specs/005-pomodoro-tracking/quickstart.md` against a freshly-migrated local instance; check off each section. Capture any deviations as follow-up issues, do not edit the spec/plan.
- [ ] T048 [P] Verify SC-005 manually: start a pomodoro in terminal A; from terminal B run `task pom cancel`; confirm terminal A exits within 5 seconds with a clear message. Note the observed latency in the PR description.

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 (Setup)**: no dependencies.
- **Phase 2 (Foundational)**: depends on Phase 1. Blocks all user-story phases. Internal ordering: T002 → T004 (apply); T005 → T007 (sqlc regen); T008 → T009 (buf regen); T010 independent; T011/T012/T013 depend on T007 and T009; T014/T015 depend on T013; T016/T017 depend on T009.
- **Phase 3 (US1)**: depends on Phase 2.
- **Phase 4 (US2)**: depends on Phase 2. May overlap with Phase 3 *only* if developers are willing to integrate the countdown extension (T031–T033) on top of T022 — otherwise treat US1 as finishing first. The handler tasks in US2 (T027–T030) are independent of US1's handler tasks and can be parallelized.
- **Phase 5 (US3)**: depends on Phase 2 only. Independent of US1, US2, US4.
- **Phase 6 (US4)**: depends on Phase 2 (`GetActivePomodoro` from T013 and `pomodoro.Remaining` from T010). Independent of US1, US2, US3.
- **Phase 7 (Polish)**: depends on all user stories you intend to ship.

### Within each story

- Server handlers and their unit tests can be developed together; tests are written alongside, not strictly before, per the project's established convention.
- CLI subcommands depend on their corresponding server handlers being wired into the Connect registration.
- Countdown UI extensions (US2) build on the US1 countdown skeleton — share the file, share the test file.

### Parallel opportunities

- T002 and T003 are different files: parallelizable.
- T010, T014, T015 are different files: parallelizable.
- T020/T021/T028/T029/T030 are all additions to the same `pomodoro_test.go`. They are listed as `[P]` because they touch disjoint test functions and a sequential merge is trivial; if your team avoids same-file simultaneous edits, sequence them in numeric order instead.
- T024, T025, T026 are different files: fully parallelizable.
- T037, T038, T040, T042, T044 are different files (or disjoint test functions): parallelizable.
- US3 (T039–T042) is completely independent of US1/US2/US4 and can be done by a separate developer in parallel once Phase 2 is done.
- US4 (T043, T044) is similarly independent.

---

## Parallel Example: Phase 2 Foundational

```bash
# Once T007 (sqlc generate) and T009 (buf generate) have landed:
Task: "T010 — create services/todo/internal/pomodoro/pomodoro.go and pomodoro_test.go"
Task: "T014 — extend services/todo/internal/handler/task_test.go with GetTask pomodoro-fields test"
Task: "T015 — create services/todo/internal/handler/pomodoro_test.go with GetActivePomodoro tests"
```

## Parallel Example: User Story 1

```bash
# After T018, T019, T022, T023 land, the test-only files can be added in parallel:
Task: "T020 — extend pomodoro_test.go with StartPomodoro tests"
Task: "T021 — extend pomodoro_test.go with CompletePomodoro tests"
Task: "T024 — add execHook to pom.go"
Task: "T025 — create pom_countdown_test.go"
Task: "T026 — create pom_test.go with runStart and execHook tests"
```

---

## Implementation Strategy

### MVP first (User Story 1 only)

1. Phase 1 (Setup) + Phase 2 (Foundational).
2. Phase 3 (User Story 1) — `task pom start <id>` runs to completion, records the pomodoro, `--exec` runs.
3. **Validate**: run the quickstart "Run a pomodoro to completion" section. Demo.

### Incremental delivery

After MVP:

- Add **US2** (Cancel/Quit/Resume/External) for the full interactive UX. This is the next biggest user-value increment.
- Add **US3** (Estimate) for planning fidelity. Independent — can be done in parallel with US2.
- Add **US4** (Status) for ambient awareness. Smallest delta; can be done last or in parallel with US2/US3.

### Parallel team strategy

After Phase 2 completes:

- Developer A: US1 (Phase 3) — MVP.
- Developer B: US3 (Phase 5) — independent.
- Developer C: US4 (Phase 6) — independent.
- After US1 lands, Developer A picks up US2 (Phase 4) — the only story that extends US1's countdown file.

---

## Notes

- [P] tasks = different files or disjoint test functions, no incomplete-dependency on each other.
- [US?] label maps a task to a specific user story for traceability.
- Every user story above is independently completable and independently testable — even US2 (which extends US1's countdown) leaves the MVP working if you stop after Phase 3.
- Commit after each task or each logical group; the existing project workflow uses small commits.
- Avoid: vague tasks, same-file conflicts that block parallelism, cross-story dependencies that break independence.
