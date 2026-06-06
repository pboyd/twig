---
description: "Task list for Snooze a Task (040-snooze-task)"
---

# Tasks: Snooze a Task

**Input**: Design documents from `/specs/040-snooze-task/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/task-api.md, quickstart.md

**Tests**: Included — this repo tests pervasively (handler/CLI/TUI/web) and `quickstart.md` enumerates the tests to add. Test tasks live within each story's phase.

**Organization**: Grouped by user story so each can be implemented and tested independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, US3 (maps to spec.md user stories)

## Path Conventions

Multi-module repo (per plan.md): proto in `api/`, server in `services/twig/`, CLI/TUI in `internal/`, web in `services/twig-web/`. Generated code (`api/gen/`, `services/twig/internal/db/`, `services/twig-web/src/gen/`) is regenerated, never hand-edited.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare codegen toolchain and create empty files the foundational phase fills in.

- [ ] T001 Verify codegen toolchain is available (`buf`, `sqlc`) and create empty migration files `services/twig/db/migrations/000009_task_snooze.up.sql` and `services/twig/db/migrations/000009_task_snooze.down.sql`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Thread `snooze_until` through the shared contract, DB, and server so the field round-trips end-to-end. **All three user stories depend on this phase** (US1 writes it; US1/US2/US3 read it).

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T002 [P] Add `snooze_until` to `Task` (field 10), `CreateTaskRequest` (field 5), and `UpdateTaskRequest` (field 6) in `api/proto/task/v1/task.proto`, per `contracts/task-api.md`
- [ ] T003 [P] Write the column migration: `ALTER TABLE tasks ADD COLUMN snooze_until TIMESTAMPTZ;` in `services/twig/db/migrations/000009_task_snooze.up.sql` and `ALTER TABLE tasks DROP COLUMN snooze_until;` in the `.down.sql`
- [ ] T004 [P] Add `snooze_until` to the `CreateTask` (column list + values) and `UpdateTask` (`SET` clause) queries in `services/twig/db/queries/task.sql`
- [ ] T005 Run `make proto` from repo root to regenerate Go stubs in `api/gen/` (depends on T002)
- [ ] T006 Run `sqlc generate` in `services/twig/` to regenerate `services/twig/internal/db/` (depends on T003, T004)
- [ ] T007 Map the field in `services/twig/internal/handler/task.go`: in `dbTaskToProto` set `pt.SnoozeUntil` when `t.SnoozeUntil.Valid`; in `CreateTask` and `UpdateTask` set `params.SnoozeUntil` from `req.Msg.SnoozeUntil` (mirror the existing `due` handling) (depends on T005, T006)
- [ ] T008 [P] Add handler tests in `services/twig/internal/handler/task_test.go`: `dbTaskToProto` maps `snooze_until` (set + unset); `CreateTask`/`UpdateTask` round-trip the value; `UpdateTask` with no `snooze_until` clears it (full-replace) (depends on T007)

**Checkpoint**: The API now persists and returns `snooze_until`. Client work can begin (US1/US2 in sequence; US3 in parallel).

---

## Phase 3: User Story 1 - Snooze a task until a future day (Priority: P1) 🎯 MVP

**Goal**: From the TUI add/edit forms, set (or clear) a snooze-until day; future-snoozed tasks vanish from the default (pending) view and reappear automatically once their day arrives.

**Independent Test**: In the TUI, edit a task → set Snooze to tomorrow → it disappears from the pending list; clear the field → it returns; set it to today/past → it stays visible.

### Implementation for User Story 1

- [ ] T009 [US1] Add the Snooze field to the edit form in `internal/tui/edit.go`: introduce `focusSnooze`, add a `snooze textinput.Model` to `editFormModel`, shift `focusSave`/`focusCancel`/`focusCount`, add a blank-form input (placeholder e.g. `"YYYY-MM-DD (optional)"`), prefill it in `NewEditForm` from `task.SnoozeUntil` formatted as a **bare day** `2006-01-02` (not RFC3339), and render its label/field in `View`, `cycleFocus`, and `updateFocusedField`
- [ ] T010 [US1] Add `snoozeStr` to `editSavedMsg` and populate it from the snooze input on both the Ctrl+S and Enter-on-Save paths in `internal/tui/edit.go`
- [ ] T011 [US1] In `internal/tui/update.go`, parse `msg.snoozeStr` via `cli.ParseDue` and set `req.SnoozeUntil` in both `createTaskCmd` and `updateTaskCmd`; leave it unset when the string is empty so `UpdateTask`'s full-replace clears the snooze (un-snooze)
- [ ] T012 [US1] Hide future-snoozed tasks (and their whole subtree) from the default view in `internal/tui/tree.go`: add an `isSnoozed(task, today)` helper (UTC date of `snooze_until` strictly after local `today`), include it in `emitNode`'s skip condition and the `hasVisibleChildren` check, and thread a `today` value through `buildVisible`/`emitNode`/`visibleSiblings` (default `time.Now().Local()`), gated by the existing show-all flag
- [ ] T013 [P] [US1] Update `internal/tui/export_test.go` shim and add tests (`internal/tui/edit_test.go`, `internal/tui/tree_test.go`): form save emits `snoozeStr`; `buildVisible` with an injected `today` hides a future-snoozed node **and its descendants** in the pending view, and keeps tasks snoozed for today/past visible

**Checkpoint**: Snoozing from the TUI works end-to-end; the pending view honors it. This is the MVP.

---

## Phase 4: User Story 2 - Reveal and review snoozed tasks (Priority: P2)

**Goal**: The `c` toggle broadens to **"Show all" ↔ "Show only pending"**, revealing completed *and* snoozed tasks; snoozed tasks shown there are visually marked with 💤.

**Independent Test**: With a snoozed task present, press `c` → it appears with a 💤 marker and the state reads "Show all"; press `c` again → hidden, state reads "Show only pending".

**Depends on**: US1 (the snooze-hide logic in `tree.go` and the show-all flag it gates on).

### Implementation for User Story 2

- [ ] T014 [US2] Broaden the toggle's meaning and labels: in `internal/tui/model.go` and the toggle handler in `internal/tui/update.go` (~line 921), treat the flag as "show all" (completed **and** snoozed); render the state label as "Show all" / "Show only pending" wherever it appears in `internal/tui/view.go` (optionally rename `showCompleted` → `showAll` across the package for clarity)
- [ ] T015 [US2] Update the help text in `internal/tui/keymap.go`: change the `Filter` binding help from `"toggle completed"` to broadened wording (e.g. `"toggle show all"`), keeping the `c` key
- [ ] T016 [US2] Add a 💤 indicator to snoozed rows when shown in `internal/tui/view.go` (and the detail pane `internal/tui/details.go` if it surfaces status), drawn from the shared theme and not clashing with the completed indicator
- [ ] T017 [P] [US2] Tests in `internal/tui/view_test.go`: with show-all on, a snoozed task appears and renders the 💤 indicator; with show-all off it is absent

**Checkpoint**: Users can reveal and visually identify snoozed tasks in the TUI.

---

## Phase 5: User Story 3 - CLI and web honor the snooze (Priority: P2)

**Goal**: The CLI and web clients keep future-snoozed tasks out of their default views (no snooze-authoring UI), so the deferral set in the TUI is honored everywhere.

**Independent Test**: Snooze a task in the TUI, then run `twig task list` (absent) vs `twig task list --all` (present); open the web tree (absent in default).

**Depends on**: Foundational only — independent of US1/US2 (separate clients), so can run in parallel with them.

### Implementation for User Story 3

- [ ] T018 [US3] Regenerate web proto types: run `npm run gen` in `services/twig-web/` so `snoozeUntil` appears on the `Task` type in `services/twig-web/src/gen/`
- [ ] T019 [US3] CLI default view hides future-snoozed: in `internal/cli/render.go` add an `isSnoozed(task, today)` helper and extend `pruneIncomplete`'s "actionable" test to `incomplete && !snoozed` (inject `today`); confirm `--all` still shows snoozed and `--completed` is unaffected in `internal/cli/task.go`
- [ ] T020 [P] [US3] CLI tests in `internal/cli/task_test.go` / `internal/cli/render_test.go`: default list prunes future-snoozed (and its subtree), `--all` keeps them, with today/tomorrow/yesterday boundary cases via injected `today`
- [ ] T021 [US3] Web default view hides future-snoozed: in `services/twig-web/src/lib/filterTree.ts` extend `keep()`'s predicate to `!completed && !snoozed`, where snoozed = UTC calendar date of `snoozeUntil` strictly after local today; accept an optional reference-date arg for testability (depends on T018)
- [ ] T022 [P] [US3] Web visuals/copy: add a 💤 indicator to snoozed rows in `services/twig-web/src/components/TreeRow.tsx`, and make the empty-state copy snooze-aware in `services/twig-web/src/theme/messages.ts` / `services/twig-web/src/pages/TaskTreePage.tsx` (depends on T018)
- [ ] T023 [P] [US3] Web tests in `services/twig-web/src/lib/filterTree.test.ts` (+ a `TreeRow` test): `filterTree` hides future-snoozed with an injected reference date and keeps the subtree together; `TreeRow` renders 💤 for a snoozed task (depends on T018)

**Checkpoint**: All three clients honor the snooze; the deferral is consistent everywhere.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Tone, full-suite verification, and docs across all stories.

- [ ] T024 [P] Tone review of all new user-facing copy for warmth per Constitution Principle IV: TUI labels/help in `internal/tui/view.go` + `internal/tui/keymap.go`, and web copy in `services/twig-web/src/theme/messages.ts` + `services/twig-web/src/pages/TaskTreePage.tsx`
- [ ] T025 Run the full suite green: `go build -o twig ./cmd/twig`, `go test ./...`, `(cd services/twig && go test ./...)`, `(cd services/twig-web && npm test)`
- [ ] T026 Execute `quickstart.md` end-to-end across TUI, CLI, and web (snooze, auto-wake, reveal, subtree-hide, empty-state)
- [ ] T027 [P] Update docs if needed (e.g., note the Snooze form field / broadened filter wording) in `CLAUDE.md` or relevant readme

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup — **BLOCKS all user stories**.
- **US1 (Phase 3)**: Depends on Foundational.
- **US2 (Phase 4)**: Depends on Foundational **and US1** (reuses the snooze-hide logic + show-all flag).
- **US3 (Phase 5)**: Depends on Foundational only — **independent of US1/US2**, can run in parallel.
- **Polish (Phase 6)**: Depends on all targeted stories being complete.

### Within Foundational

- T002 → T005 (proto regen); T003 + T004 → T006 (sqlc regen); T005 + T006 → T007 → T008.

### Parallel Opportunities

- Foundational: **T002, T003, T004** are different files → run in parallel; then T005/T006 (regen) → T007 → T008.
- After Foundational: **US1+US2 (one track)** and **US3 (another track)** can proceed in parallel.
- Within US3: after T018, **T020 / T022 / T023** are parallelizable; T019 and T021 touch distinct files too.
- Test tasks marked [P] (T008, T013, T017, T020, T023) run alongside or right after their implementation.

---

## Parallel Example: Foundational contract + persistence

```bash
# Different files — safe to do together:
Task: "Add snooze_until to api/proto/task/v1/task.proto (T002)"
Task: "Write migration 000009_task_snooze.{up,down}.sql (T003)"
Task: "Add snooze_until to CreateTask/UpdateTask in db/queries/task.sql (T004)"
# Then regenerate:
Task: "make proto (T005)"   ;  Task: "sqlc generate (T006)"
```

## Parallel Example: US1 and US3 tracks

```bash
# Track A (TUI authoring + reveal): T009 → T010 → T011 → T012 → T013 → (US2) T014…T017
# Track B (other clients), in parallel: T018 → T019/T021 → T020/T022/T023
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 (Setup) → Phase 2 (Foundational — the field round-trips).
2. Phase 3 (US1) → snoozing from the TUI hides/reveals correctly.
3. **STOP and VALIDATE**: snooze, clear, today/past, subtree-hide, auto-wake.
4. Demo — this is a usable, shippable increment on its own.

### Incremental Delivery

1. Foundational → US1 (MVP: author + honor snooze in the TUI).
2. US2 → reveal + 💤 marker in the TUI.
3. US3 → CLI and web honor the snooze (parallelizable with US1/US2 after Foundational).
4. Polish → tone, full-suite green, quickstart, docs.

---

## Notes

- [P] = different files, no incomplete-task dependencies.
- Generated dirs (`api/gen/`, `services/twig/internal/db/`, `services/twig-web/src/gen/`) are produced by T005/T006/T018 — never hand-edit them.
- Inject a `today`/reference-date value into the snooze predicates (TUI/CLI/web) so tests are deterministic (research.md Decision 5).
- The auto-wake behavior (FR-005) needs **no task** — visibility is time-derived from `snooze_until`, so a task reappears as soon as the client re-evaluates on/after its day.
- Commit after each task or logical group.
