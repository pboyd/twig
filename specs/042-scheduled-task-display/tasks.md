---
description: "Task list for Show Scheduled Days in Task Details"
---

# Tasks: Show Scheduled Days in Task Details

**Input**: Design documents from `/specs/042-scheduled-task-display/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/plan-service.md

**Tests**: Included — the project is test-heavy and research.md/quickstart.md specify a concrete test approach (handler tests gated on `DATABASE_URL`; TUI render/wiring tests with fakes).

**Organization**: Foundational plumbing (the reverse query + TUI fetch) blocks all stories; each user story phase then adds its user-visible behavior and its distinct acceptance tests.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1 / US2 / US3 (maps to spec.md user stories)
- All paths are repo-relative from `/home/user/dev/twig/`

## Path Conventions

Multi-module repo: shared proto in `api/`, server in `services/twig/`, CLI/TUI in `internal/tui/`. Generated code (`api/gen/`, `services/twig/internal/db/`) is never hand-edited — it's regenerated via `make proto` / `sqlc generate`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the codegen toolchain is available before touching contracts.

- [ ] T001 Verify `buf` (for `make proto`) and `sqlc` CLIs are installed and that `make dev` brings up Postgres + server; confirm `DATABASE_URL=postgres://twig:twig@localhost:5432/twig?sslmode=disable` works via `psql`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the reverse-lookup contract, query, handler, and TUI fetch that every user story depends on. No `Scheduled for:` line is rendered yet.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T002 Add `ListScheduledDays` RPC plus `ListScheduledDaysRequest`, `ListScheduledDaysResponse`, and `ScheduledDay` messages to `api/proto/plan/v1/plan.proto`, exactly per `specs/042-scheduled-task-display/contracts/plan-service.md`.
- [ ] T003 Run `make proto` from repo root to regenerate `api/gen/plan/v1/...` (including the `planv1connect` client method). Depends on T002. Do not hand-edit generated files.
- [ ] T004 [P] Add the `ListScheduledDaysForTasks` query to `services/twig/db/queries/plan.sql` per `data-model.md` (`SELECT DISTINCT task_id, day FROM plan_entries WHERE user_id = $1 AND task_id IS NOT NULL AND day >= $2 ORDER BY task_id, day`).
- [ ] T005 Run `sqlc generate` from `services/twig/` to regenerate `services/twig/internal/db/`. Depends on T004. Do not hand-edit generated files.
- [ ] T006 Implement `(*Plan).ListScheduledDays` in `services/twig/internal/handler/plan.go`: read the user id from context (as other Plan RPCs do), validate `from_day` with the existing `parseDay`, call `ListScheduledDaysForTasks`, and map each row to a `planv1.ScheduledDay{TaskId, Day}` (format `day` with `2006-01-02`). Return an empty list (not an error) when there are no rows. Depends on T003, T005.
- [ ] T007 Add a `scheduledDays map[int64][]string` field to `Model` in `internal/tui/model.go` (task id → ascending `YYYY-MM-DD` days) and initialize it (empty map) in `newModel`. Depends on T003.
- [ ] T008 Add `listScheduledDaysCmd(planClient, fromDay string) tea.Cmd` and a `scheduledDaysResultMsg` in `internal/tui/update.go` that calls `PlanService.ListScheduledDays` with `fromDay` = the **local** current date (`time.Now().Format("2006-01-02")`) and groups the response into `map[int64][]string` (days arrive already ascending/deduped). Handle the message by storing the map on the Model. Depends on T003, T007.
- [ ] T009 Trigger `listScheduledDaysCmd` at the points that keep the map fresh, in `internal/tui/update.go`: on `Init`, on the `Ctrl-R` refresh path, on switching to the Tasks tab, and after a successful `ctrl+p` send-to-plan (`addPlanTaskCmd` completion). Depends on T008.

**Checkpoint**: Server returns scheduled days; TUI holds a fresh map. Rendering follows in the story phases.

---

## Phase 3: User Story 1 — See that a task is scheduled (Priority: P1) 🎯 MVP

**Goal**: A selected task that is scheduled for today or a future day shows a `Scheduled for: <day>` line in the details pane; an unscheduled task shows no such line.

**Independent test**: Select a task scheduled for a single future day → details pane shows `Scheduled for: <YYYY-MM-DD>`. Select a never-scheduled task → no `Scheduled for:` line.

- [ ] T010 [US1] In `internal/tui/details.go`, emit a `Scheduled for:` line in **both** the plain (`!styled`) and styled paths of `renderDetails`, reading `m`-supplied `scheduledDays[task.Id]`. Use the existing `labelStyle` for the label in the styled path (mirroring `Due:`/`Snooze:`); join the day slice with `", "`; omit the line entirely when the slice is empty/absent. Pass the day slice into `renderDetails` (extend its signature or add a small helper) without coupling it to the network. Depends on T009.
- [ ] T011 [P] [US1] Add render unit tests in `internal/tui/details_test.go`: (a) one future day → line present with that date, (b) unscheduled task → no `Scheduled for:` substring, for both `styled=true` and `styled=false`. Depends on T010.
- [ ] T012 [P] [US1] Add a handler test in `services/twig/internal/handler/plan_test.go` (using `newTestPlanPool`, gated on `DATABASE_URL`): one task scheduled on a single future day is returned by `ListScheduledDays`; a second user's entries are not returned; a user with no scheduled tasks gets an empty list. Depends on T006.
- [ ] T013 [P] [US1] Add a TUI wiring test (e.g. `internal/tui/update_test.go`) using the existing `fakePlanClient` pattern asserting `listScheduledDaysCmd` is issued on init and that the request's `from_day` equals the local current date. Depends on T009.

**Checkpoint**: MVP — scheduled vs. unscheduled is visible from the Tasks tab.

---

## Phase 4: User Story 3 — Past scheduling is not shown (Priority: P1)

**Goal**: Days strictly before the user's local today are never shown; today is shown; a task scheduled only in the past shows no line. (Filtering lives in the foundational query + local `from_day`; this phase proves it.)

**Independent test**: Schedule a task only for yesterday → no line. Schedule for yesterday + tomorrow → only tomorrow shown. Schedule for yesterday + today → only today shown.

- [ ] T014 [US3] Confirm/adjust that `from_day` in `listScheduledDaysCmd` (`internal/tui/update.go`) is the **local** current date so the server's `day >= from_day` cutoff is local, per the spec clarification (and intentionally unlike the UTC `Snooze:` line). Depends on T009.
- [ ] T015 [P] [US3] Add handler tests in `services/twig/internal/handler/plan_test.go`: with entries on yesterday, today, and tomorrow for one task and `from_day = today`, the result contains today and tomorrow only (yesterday excluded; today included). Depends on T006.
- [ ] T016 [P] [US3] Add a render/wiring test (`internal/tui/details_test.go`) proving a task whose `scheduledDays` slice is empty (its only scheduling was filtered out as past) shows no `Scheduled for:` line. Depends on T010.

**Checkpoint**: Past days are provably hidden; today is provably shown.

---

## Phase 5: User Story 2 — See every day a task is scheduled for (Priority: P2)

**Goal**: A task scheduled across multiple current/future days lists all of them, ascending, comma-space separated, with each day appearing once.

**Independent test**: Schedule one task on two future days (and twice on one of them) → details pane shows both days once, ascending, joined by `, `.

- [ ] T017 [US2] Confirm the `renderDetails` join from T010 produces `d1, d2, …` in ascending order directly from the (already-ordered) map slice, with no client-side sort/dedup needed; adjust the join in `internal/tui/details.go` only if a gap is found. Depends on T010.
- [ ] T018 [P] [US2] Add a render unit test in `internal/tui/details_test.go`: a multi-day slice renders as `"2026-06-07, 2026-06-08"` (ascending, comma-space), for both styled and plain paths. Depends on T010.
- [ ] T019 [P] [US2] Add a handler test in `services/twig/internal/handler/plan_test.go`: a task with two entries on the same future day plus an entry on a later day yields that day once and both days in ascending order (verifies `DISTINCT` + `ORDER BY`); include an untimed (`start_minute` NULL) entry to confirm it still counts. Depends on T006.

**Checkpoint**: Full multi-day display verified end to end.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T020 [P] Run `go test ./...` from repo root and `cd services/twig && DATABASE_URL=postgres://twig:twig@localhost:5432/twig?sslmode=disable go test ./...`; fix any failures.
- [ ] T021 [P] Constitution review of `internal/tui/details.go`: confirm the new line uses only shared theme tokens/label style (Principle III) and introduces no new tone-bearing copy — `Scheduled for:` is a neutral field label consistent with `Due:`/`Completed:`/`Snooze:` (Principle IV).
- [ ] T022 Manual verification per `specs/042-scheduled-task-display/quickstart.md`: create a task, schedule it for yesterday/today/tomorrow via `ctrl+p`, launch the TUI, and confirm the pane shows today + tomorrow (not yesterday); confirm a past-only task shows no line.

---

## Dependencies

- **Setup (T001)** → everything.
- **Foundational (T002–T009)** blocks all user stories. Internal order: T002 → T003; T004 → T005; (T003 ∥ T005) → T006; T003 → T007 → T008 → T009.
- **US1 (T010–T013)**: T010 needs T009; T011 needs T010; T012 needs T006; T013 needs T009.
- **US3 (T014–T016)**: T014 needs T009; T015 needs T006; T016 needs T010.
- **US2 (T017–T019)**: T017/T018 need T010; T019 needs T006.
- **Polish (T020–T022)** after all stories.

User stories share the foundational query + render line, so once T010 lands, US1/US2/US3 differ mainly by their test coverage and can be verified in any order.

## Parallel Execution Examples

- **Foundational kickoff**: T004 (SQL query) runs in parallel with T002 (proto edit) — different modules/files.
- **US1 tests**: T011 (`details_test.go`), T012 (`plan_test.go`), T013 (`update_test.go`) are all different files → run in parallel after their impl deps land.
- **Across stories (post-T010)**: the handler tests T015 and T019 both touch `plan_test.go`, so do them sequentially; render tests T016 and T018 both touch `details_test.go`, also sequential — but a handler test and a render test in different files can run in parallel.
- **Polish**: T020 and T021 are independent (test run vs. code review) → parallel.

## Implementation Strategy

**MVP = Phase 1 + Phase 2 + Phase 3 (US1).** That delivers the whole user-visible value: you can tell from the Tasks tab whether a task is scheduled and on which upcoming day. US3 and US2 then harden the past-exclusion and multi-day formatting with targeted tests (the query already implements both). Ship incrementally: Foundational → US1 (demo) → US3 → US2 → Polish.
