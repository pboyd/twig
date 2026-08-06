---

description: "Task list for 074-plan-objective-notes"
---

# Tasks: Plan Objectives and Notes

**Input**: Design documents from `/specs/074-plan-objective-notes/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: Included. Not because the spec demanded TDD, but because this repo tests every
tier — handler integration tests, CLI tests, and TUI view/update tests all exist for the
surfaces this feature touches, and `quickstart.md` names the target file for each. Skipping
them would leave the feature the only untested part of the planning tab.

**Organization**: Grouped by user story. The backend is genuinely shared by all four
stories, so it lives in Foundational rather than being split across them.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel — different file, no dependency on an incomplete task
- **[Story]**: US1–US4, mapping to the user stories in spec.md
- Every task names the exact file it changes

## Path Conventions

Three Go modules (see plan.md → Project Structure):

- `api/` — protobuf + generated ConnectRPC stubs
- `services/twig/` — server: migrations, sqlc queries, handlers
- repo root — CLI (`internal/cli/`) and TUI (`internal/tui/`)

Generated code (`api/gen/`, `services/twig/internal/db/`) is produced by tooling and never
hand-edited.

---

## Phase 1: Setup

**Purpose**: Make the environment able to prove the work

- [x] T001 Bring up the stack and export `DATABASE_URL` per the Environment section of `specs/074-plan-objective-notes/quickstart.md` — without it the handler tests skip silently and a green `go test ./...` proves nothing

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The `plan_days` table and the RPCs that read and write it. Every user story
depends on this; nothing user-facing can be built or tested until it lands.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Contract (Principle II — contract before implementation)

- [x] T002 Add the `PlanDay` message, the `day = 2` field on `ListPlanEntriesResponse`, and the `SetPlanObjective` / `SetPlanNotes` RPCs with their request/response messages to `api/proto/plan/v1/plan.proto`, exactly as specified in `specs/074-plan-objective-notes/contracts/plan-day.md`
- [x] T003 Run `make proto` from the repo root to regenerate `api/gen/plan/v1/` — commit the generated output, do not hand-edit it

### Storage

- [x] T004 [P] Create `services/twig/db/migrations/000015_plan_days.up.sql` creating the `plan_days` table per the schema in `specs/074-plan-objective-notes/data-model.md` — PK `(user_id, day)`, `objective VARCHAR(255) NOT NULL DEFAULT ''`, `notes TEXT NOT NULL DEFAULT ''`, `user_id` FK to `users(id)` `ON DELETE CASCADE`
- [x] T005 [P] Create `services/twig/db/migrations/000015_plan_days.down.sql` dropping the `plan_days` table
- [x] T006 Add `GetPlanDay`, `UpsertPlanDayObjective`, and `UpsertPlanDayNotes` to `services/twig/db/queries/plan.sql` per data-model.md — each upsert writes exactly one column so the two setters can never clobber each other
- [x] T007 Run `sqlc generate` from `services/twig/` to regenerate `services/twig/internal/db/` — commit the generated output, do not hand-edit it

### Handler

- [x] T008 Add a `dbPlanDayToProto` converter and a `loadPlanDay` helper (translating `pgx.ErrNoRows` into a zero-value `PlanDay` rather than an error) to `services/twig/internal/handler/plan.go`
- [x] T009 Populate `ListPlanEntriesResponse.day` in `ListPlanEntries` in `services/twig/internal/handler/plan.go` — always non-nil, echoing the requested day, with empty strings when no row exists
- [x] T010 Implement `SetPlanObjective` in `services/twig/internal/handler/plan.go`: trim surrounding whitespace, reject over 255 runes with a warm actionable `CodeInvalidArgument` message, upsert, return the full `PlanDay`
- [x] T011 Implement `SetPlanNotes` in `services/twig/internal/handler/plan.go`: trim, no length limit, upsert, return the full `PlanDay`
- [x] T012 Register nothing new in `services/twig/cmd/server/main.go` — verify the existing `PlanService` registration already exposes both new RPCs, and fix it if not

### Handler tests

These share one file, so they run in sequence rather than in parallel.

- [x] T013 Test `ListPlanEntries` day population in `services/twig/internal/handler/plan_test.go`: day with a stored row, day with no row, day with entries but no row, and day with a row but no entries
- [x] T014 Test `SetPlanObjective` in `services/twig/internal/handler/plan_test.go`: set, overwrite, clear with `""`, clear with whitespace-only, trimming, 255-rune boundary accepted, 256 rejected with nothing written, invalid day rejected, and `notes` left untouched
- [x] T015 Test `SetPlanNotes` in `services/twig/internal/handler/plan_test.go`: set, clear, trim, multi-line content preserved verbatim, `objective` left untouched, and two users' values for the same date staying independent

**Checkpoint**: The backend is complete and provable. `psql "$DATABASE_URL" -c '\d plan_days'` shows the table; the server suite passes with `DATABASE_URL` set. User story work can begin.

---

## Phase 3: User Story 1 — Set and see the day's objective (Priority: P1) 🎯 MVP

**Goal**: A full-width objective band above the planning tab's grid/details row, omitted
entirely when unset, with `o` opening a single-field editor in that same slot where Enter
saves and Esc cancels.

**Independent Test**: On a day with no objective, confirm no band and a full-height grid.
Press `o`, type, press Enter — the band appears with the text. Navigate away and back; it
persists and is per-day.

### State and keys

- [x] T016 [US1] Add `objective string` and `objectiveInput textinput.Model` to `planState` in `internal/tui/model.go`
- [x] T017 [US1] Add `planObjectiveEdit` to the `planMode` enum in `internal/tui/model.go`
- [x] T018 [P] [US1] Add the `PlanObjective` binding (`o`, help text "set objective") to `KeyMap` in `internal/tui/keymap.go` and surface it in `ShortHelp`/`FullHelp` under the existing `k.PlanningMode` gate

### Data flow

- [x] T019 [US1] Add `day *planv1.PlanDay` to `planEntriesMsg` in `internal/tui/plan_update.go` and populate it in `listPlanCmd` / `listPlanHighlightCmd` from `ListPlanEntriesResponse.day`
- [x] T020 [US1] Apply the loaded objective in `handlePlanEntriesMsg` in `internal/tui/plan_update.go` under the existing stale-day rules, and never over an open editor's draft when the response is a background (`bg`) load
- [x] T021 [US1] Add `setPlanObjectiveCmd` and its result message to `internal/tui/plan_update.go`, calling `SetPlanObjective` and carrying the returned `PlanDay` or the error

### Interaction

- [x] T022 [US1] Handle `o` in the planning tab's key switch in `internal/tui/update.go` — enter `planObjectiveEdit` pre-filled with `m.plan.objective`, only when `m.plan.mode == planList` so it cannot open over a picker or entry form
- [x] T023 [US1] Handle the objective editor's keys in `internal/tui/update.go`: Enter dispatches `setPlanObjectiveCmd` and returns to `planList`, Esc discards the draft and returns to `planList`
- [x] T024 [US1] Handle the objective save result in `internal/tui/update.go` — on success update `m.plan.objective` from the response and show a warm confirmation; on failure keep the editor open with its text and surface the error through the planning tab's existing error path

### Rendering

- [x] T025 [US1] Render the objective band and its editor variant in `internal/tui/plan_view.go`: content via `m.md.Render`, inner height capped at 3 lines, shown when the objective is non-empty **or** `mode == planObjectiveEdit`, zero height otherwise
- [x] T026 [US1] Subtract the band's height from `innerH` and prepend it in the **styled** branch of `viewPlanning` in `internal/tui/view.go`, before the grid and right panes are sized
- [x] T027 [US1] Do the same in the **unstyled** branch of `viewPlanning` in `internal/tui/view.go`, using its existing row-join instead of `paneBox`

### Tests

- [x] T028 [P] [US1] Add any needed accessors for the new `planState` fields and modes to `internal/tui/export_test.go`
- [x] T029 [US1] Test band rendering in `internal/tui/plan_view_test.go`: omitted when unset (output byte-identical to a no-objective baseline, per SC-003), shown when set, height capped at 3 lines, shown while editing on a day with no objective, and both the styled and unstyled branches
- [x] T030 [US1] Test the interaction in `internal/tui/plan_update_test.go`: `o` opens pre-filled, Enter saves, Esc discards, clearing to empty removes the band, `o` is inert while a picker or entry form is open, and each day shows its own value
- [x] T031 [P] [US1] Test in `internal/tui/autorefresh_apply_test.go` that a background load neither overwrites an open objective draft nor applies a response for a different day

**Checkpoint**: US1 is independently shippable — the objective works end to end in the TUI.

---

## Phase 4: User Story 2 — Write and read the day's notes (Priority: P1)

**Goal**: A Notes pane in the right column below Details, always present; `n` turns the
whole right column into a textarea that saves with `ctrl+s`, cancels with `esc`, and reaches
`$EDITOR` with `ctrl+g`.

**Independent Test**: Press `n`, type several lines (Enter inserts newlines, does not save),
press `ctrl+s`, and confirm the Notes pane shows the text and it persists across day
navigation.

### State and keys

- [x] T032 [US2] Add `notes string` and `notesInput textarea.Model` to `planState` in `internal/tui/model.go`
- [x] T033 [US2] Add `planNotesEdit` to the `planMode` enum in `internal/tui/model.go`
- [x] T034 [P] [US2] Add the `PlanNotes` binding (`n`, help text "edit notes") to `KeyMap` in `internal/tui/keymap.go` and surface it under the `k.PlanningMode` gate

### Data flow

- [x] T035 [US2] Apply the loaded notes in `handlePlanEntriesMsg` in `internal/tui/plan_update.go`, under the same stale-day and open-draft rules used for the objective
- [x] T036 [US2] Add `setPlanNotesCmd` and its result message to `internal/tui/plan_update.go`, calling `SetPlanNotes`

### Interaction

- [x] T037 [US2] Handle `n` in the planning tab's key switch in `internal/tui/update.go` — enter `planNotesEdit` pre-filled with `m.plan.notes`, only when `m.plan.mode == planList`
- [x] T038 [US2] Handle the notes editor's keys in `internal/tui/update.go`: `ctrl+s` saves and returns to `planList`, `esc` discards, Enter inserts a newline and must **not** save
- [x] T039 [US2] Route `ctrl+g` in `planNotesEdit` to the existing `openEditorCmd` in `internal/tui/update.go`, and add a `planNotesEdit` branch to the `editorFinishedMsg` handler (before the existing task-edit branch) that loads the returned text into `notesInput`, leaving it unsaved; on error keep the draft and show a warm actionable message
- [x] T040 [US2] Handle the notes save result in `internal/tui/update.go` — success updates `m.plan.notes` and confirms warmly; failure keeps the editor open with its text

### Rendering

- [x] T041 [US2] Render the Notes pane and the full-column notes editor in `internal/tui/plan_view.go`: content via `m.md.Render`, pane rendered even when notes are empty, editor replacing the whole right column while `mode == planNotesEdit`
- [x] T042 [US2] Split the right column in the **styled** branch of `viewPlanning` in `internal/tui/view.go`: `detailsInner = ceil(h/2)` floored at 3 inner lines, Notes taking the remainder, with the split suppressed while a picker or entry form owns the column
- [x] T043 [US2] Do the same in the **unstyled** branch of `viewPlanning` in `internal/tui/view.go`

### Tests

- [x] T044 [US2] Test pane rendering in `internal/tui/plan_view_test.go`: Notes pane present when empty, content shown when set, the even split with the details floor, the editor taking the whole column, no Notes pane while a picker or form is open, and both styled and unstyled branches
- [x] T045 [US2] Test the interaction in `internal/tui/plan_update_test.go`: `n` opens pre-filled, Enter inserts a newline without saving, `ctrl+s` saves, `esc` discards, `ctrl+g` round-trips through `editorFinishedMsg` leaving the draft unsaved, and `n` is inert while another form is open
- [x] T046 [P] [US2] Test in `internal/tui/autorefresh_apply_test.go` that a background load does not overwrite an open notes draft
- [x] T047 [P] [US2] Add a regression test in `internal/tui/update_test.go` that `n` on the Tasks tab still creates a subtask (FR-023)

**Checkpoint**: US1 and US2 both work independently. The planning tab is feature-complete.

---

## Phase 5: User Story 3 — Fields render as formatted markdown (Priority: P2)

**Goal**: Both fields render like task descriptions do — same renderer, same theme, same
plain-text fallback — while the editors hold raw source. US1 and US2 already call
`m.md.Render`; this phase proves the fidelity and hardens the overflow and narrow-terminal
behaviour those stories introduce.

**Independent Test**: Set an objective and notes containing a heading, a bullet list, bold
text, a link, and inline code. Confirm they render as on the Tasks tab at equal width, that
`o` and `n` show raw markdown, and that the unstyled path emits no ANSI.

- [x] T048 [US3] Verify and correct the `m.md.Render` calls in `internal/tui/plan_view.go` — correct inner width for each pane, `Styled: m.styled` threaded through, so both fields pick up the shared theme and unstyled fallback automatically
- [x] T049 [US3] Test markdown fidelity in `internal/tui/plan_view_test.go`: headings, lists, emphasis, links, and inline code render, output matches the task-description rendering at equal width, and the unstyled path produces plain wrapped text with no escape sequences
- [x] T050 [US3] Test that both editors present raw markdown source rather than rendered output in `internal/tui/plan_update_test.go`
- [x] T051 [US3] Test resilience in `internal/tui/plan_view_test.go`: content far longer and wider than its pane is clipped inside the pane, an unbreakable long URL does not exceed the width, malformed markdown renders as text without error, and a narrow/short terminal leaves the grid, borders, tab bar, and status line intact

**Checkpoint**: Both panes read correctly at any size, styled or not.

---

## Phase 6: User Story 4 — Read and set the objective from the command line (Priority: P2)

**Goal**: `twig plan [--date D] objective [text]` prints or sets the day's objective,
inheriting `--date` and the today-default from the plan command.

**Independent Test**: Set an objective from the CLI, read it back bare (`test "$(...)" = 'Ship it'`),
and confirm the TUI shows the same value for that day.

- [x] T052 [US4] Add the `objective` case to the subcommand switch in `runPlan` in `internal/cli/plan.go` — placed after the existing `--date` parsing so it inherits the flag and today-default with no flag handling of its own
- [x] T053 [US4] Implement `runPlanObjective` in `internal/cli/plan.go`: no argument reads via `ListPlanEntries` and prints `resp.Msg.Day.Objective` bare with a trailing newline (nothing at all when empty, exit 0); one argument calls `SetPlanObjective` and prints a warm confirmation; more than one argument is a usage error on stderr with exit 1
- [x] T054 [US4] Add the `objective` line to `printPlanUsage` in `internal/cli/plan.go`, matching the existing subcommand-description style
- [x] T055 [US4] Test the subcommand in `internal/cli/plan_test.go`: read on an empty day prints nothing and exits 0, write then read round-trips, clearing with `''` works, the read output carries no label or ANSI so it is pipeable, an over-long objective errors actionably, too many arguments is a usage error, and `--date` targets the right day while its absence targets today

**Checkpoint**: All four stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [x] T056 [P] Update the planning-tab description in `AGENTS.md` to mention the objective band, the Notes pane, and the `o` / `n` keys
- [x] T057 [P] Review every new user-facing string in `internal/tui/`, `internal/cli/plan.go`, and `services/twig/internal/handler/plan.go` against Principle IV — warm, accurate, actionable — noting the one documented exception: the CLI objective read prints bare text for scriptability
- [x] T058 Confirm `make proto` and `sqlc generate` leave no uncommitted diff, and that `make migrate-down` cleanly drops `plan_days`
- [x] T059 Run the full verification pass in `specs/074-plan-objective-notes/quickstart.md` — CLI checks, TUI checks, guards, and the cross-surface round trip — with `go test ./...` green at the repo root and in `services/twig/` with `DATABASE_URL` set

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (T001)**: no dependencies
- **Foundational (T002–T015)**: blocks every user story. Within it: T002 → T003 (proto before generation); T004/T005/T006 → T007 (schema and queries before `sqlc generate`); T003 + T007 → T008–T012 (both generators before handler code); T008–T012 → T013–T015 (implementation before its tests)
- **US1 (T016–T031)**, **US2 (T032–T047)**, **US3 (T048–T051)**, **US4 (T052–T055)**: all depend only on Foundational
- **Polish (T056–T059)**: depends on every story you intend to ship

### Story dependencies

All four stories are independent in **behaviour** — any one can ship alone on top of the
backend, and each has its own independent test.

They are not fully independent in **files**. US1 and US2 both edit `internal/tui/model.go`,
`view.go`, `plan_view.go`, `plan_update.go`, `update.go`, and `keymap.go`; US3 edits files
US1 and US2 create content in. Two people working US1 and US2 simultaneously will collide.
US4 touches only `internal/cli/plan.go` and is genuinely parallel to all TUI work.

Recommended order: **US1 → US2 → US3**, with **US4 in parallel** to any of them.

### Within each story

State and key bindings → data flow → interaction → rendering → tests. Rendering last,
because the layout tasks need the state they read to exist.

### Parallel opportunities

- T004 and T005 (up and down migrations, different files)
- T018, T028, T031 in US1; T034, T046, T047 in US2 — each touches a file no sibling task in
  its phase touches
- T056 and T057 in Polish
- All of US4 against all of the TUI work

Tasks sharing a file are never marked `[P]`, which is why the handler tests (T013–T015, one
file) and most TUI tasks run in sequence.

---

## Parallel Example: Foundational

```bash
# The two migration halves are independent files:
Task: "Create services/twig/db/migrations/000015_plan_days.up.sql"
Task: "Create services/twig/db/migrations/000015_plan_days.down.sql"
```

## Parallel Example: US4 alongside the TUI

```bash
# Different module, no shared files with US1/US2/US3:
Task: "Implement runPlanObjective in internal/cli/plan.go"
Task: "Render the objective band in internal/tui/plan_view.go"
```

---

## Implementation Strategy

### MVP first (Foundational + US1)

1. T001 — environment, with `DATABASE_URL` exported
2. T002–T015 — the `plan_days` table and both RPCs, contract first
3. T016–T031 — the objective band and its editor
4. **Stop and validate**: run the objective section of quickstart.md
5. Shippable: the objective works end to end in the TUI

### Incremental delivery

1. Foundational → backend proven by integration tests
2. + US1 → objective in the TUI (**MVP**)
3. + US2 → notes in the TUI; the planning tab is feature-complete
4. + US3 → markdown fidelity and layout resilience confirmed
5. + US4 → objective reachable from the CLI and from scripts
6. + Polish → docs, tone, and the full quickstart pass

Each step leaves the planning tab working; none breaks the step before it.

### Notes

- Generated code (`api/gen/`, `services/twig/internal/db/`) is committed but never hand-edited
- `go test ./...` passes vacuously without `DATABASE_URL` — the handler tests skip silently,
  so always run the server suite with it set
- Both branches of `viewPlanning` (styled and unstyled) need every layout change; forgetting
  the unstyled one is the easiest mistake in this feature
- Commit after each task or logical group
