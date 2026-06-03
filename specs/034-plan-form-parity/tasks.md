---
description: "Task list for Plan Entry Form Parity with Task Form"
---

# Tasks: Plan Entry Form Parity with Task Form

**Input**: Design documents from `specs/034-plan-form-parity/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/edit-form-ui.md, quickstart.md

**Tests**: Included — the project follows TDD and the existing `internal/tui` suite is the executable spec for form behavior (string/table tests, no DB).

**Organization**: Tasks are grouped by user story. Because all three planning forms share one renderer (`renderPlanFormView`), one focus model (`planFormState`), and one key handler (`handlePlanFormKey`), the shared presentation/interaction changes are **Foundational** (blocking both stories). US1 adds the edit-only pre-fill + submit semantics; US2 verifies and polishes parity on the schedule/add-event forms.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1 / US2 (Setup, Foundational, Polish carry no story label)
- All paths are relative to repo root; the Go module lives at `services/twig/`.

## Path Conventions

- TUI source: `services/twig/internal/tui/`
- Time parsing helper: `services/twig/internal/cli/timeparse/`
- Run tests from `services/twig/` (`go test ./...`)

---

## Phase 1: Setup

**Purpose**: Establish a known-good baseline before changes.

- [ ] T001 Confirm baseline is green: from `services/twig/` run `go test ./internal/tui/... ./internal/cli/timeparse/...` and record that it passes before any edits.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared form mechanics used by all three planning modes. MUST complete before US1 and US2. After this phase, all three forms render with title/spacing/buttons/help-line parity, Enter no longer saves, and focus cycles through the buttons.

- [ ] T002 [P] Add `FormatDuration(minutes int) string` (compact form: `30m`, `1h30m`) to `services/twig/internal/cli/timeparse/timeparse.go`, and add a round-trip unit test (`ParseDuration(FormatDuration(m)) == m` for representative values incl. 0/30/90/600) in `services/twig/internal/cli/timeparse/timeparse_test.go`.
- [ ] T003 Extend the focus model in `services/twig/internal/tui/plan_update.go`: treat focus indices `len(fields)` and `len(fields)+1` as virtual Save/Cancel slots; update `cyclePlanFormFocus` to wrap over `len(fields)+2` and `Blur()` all fields when a button slot is focused. Add helper predicates `planFocusSave`/`planFocusCancel` (in `services/twig/internal/tui/plan_view.go` or `model.go`).
- [ ] T004 Update `renderPlanFormView` in `services/twig/internal/tui/plan_view.go`: emit the (currently discarded) per-mode `title` followed by a blank line; insert one blank line between adjacent fields; render `[ Save ]  [ Cancel ]` below the fields with `[>Save<]`/`[>Cancel<]` when the corresponding button slot is focused; replace the help line with exactly `Ctrl+S: save  Esc: cancel  Tab: next field` (preceded by a blank line). Conform to `contracts/edit-form-ui.md` R1–R4.
- [ ] T005 Update `handlePlanFormKey` in `services/twig/internal/tui/update.go`: remove `|| msg.Type == tea.KeyEnter` from the Save case; add Enter handling that submits when Save is focused, cancels when Cancel is focused, else advances focus (`cyclePlanFormFocus(1)`); guard the text-input forwarding so a focused **button** slot (`focus >= len(fields)`) is not indexed into `fields`. Keep `keys.Save` (Ctrl+S) submitting from any focus. Conform to contract key-binding table.
- [ ] T006 [P] Remove sentinel hint text in `services/twig/internal/tui/plan_view.go` (`planFieldLabel` for `planEdit` → plain `Start` / `Duration`) and `services/twig/internal/tui/plan_update.go` (neutral placeholders, e.g. `e.g. 09:00` / `e.g. 30m`). No `blank=keep` or `null=unschedule` may remain (contract R5).
- [ ] T007 Add foundational render/interaction tests in `services/twig/internal/tui/plan_view_test.go` and `services/twig/internal/tui/plan_update_test.go`: assert the title line, `[ Save ]`/`[ Cancel ]`, and the exact help line render; Tab/Shift-Tab cycles focus through the buttons; Enter in a focused text field does not submit (mode stays open). Cover all three modes where practical.

**Checkpoint**: All three planning forms now look and behave like the task form (presentation + key handling). US1 and US2 can proceed in parallel.

---

## Phase 3: User Story 1 - Edit a plan entry with a familiar, polished form (Priority: P1) 🎯 MVP

**Goal**: The edit form pre-fills the entry's current Name/Start/Duration, and saving maps cleanly to rename/reschedule/unschedule with unchanged-save as a no-op.

**Independent Test**: Open Planning → select a scheduled entry → Enter. Verify pre-filled `09:00` / `30m`, no `blank=keep`, no-op save leaves it unchanged, clearing Start unschedules, and rename works.

### Tests (write first — they must fail before T010/T011)

- [ ] T008 [P] [US1] Test in `services/twig/internal/tui/plan_update_test.go`: `initEditForm` pre-fills Name verbatim, Start as `HH:MM`, Duration as compact form for a scheduled entry; leaves Start/Duration empty for an untimed entry; placeholders contain no `blank=keep` text.
- [ ] T009 [P] [US1] Test in `services/twig/internal/tui/plan_update_test.go`: `submitEditForm` change-detection — unchanged form dispatches no rename/move and closes; name-only change dispatches rename only; clearing Start on a scheduled entry dispatches unschedule and ignores Duration; changed start/duration dispatches a scheduled move; invalid start or duration keeps the form open with an error.

### Implementation

- [ ] T010 [US1] Update `initEditForm` in `services/twig/internal/tui/plan_update.go`: pre-fill Start (`fmt.Sprintf("%02d:%02d", …)`) and Duration (`timeparse.FormatDuration`) from the selected entry; set neutral placeholders; keep Name pre-fill and `entryID`.
- [ ] T011 [US1] Rewrite `submitEditForm` in `services/twig/internal/tui/plan_update.go`: compare trimmed fields against the original entry; rename only on name change; on empty Start unschedule a scheduled entry (ignore Duration) / no-op if already untimed; on present Start, parse and dispatch a move only if start/duration changed; remove the `null`-keyword path; preserve empty-name validation; keep the "no commands → close form" no-op path.
- [ ] T012 [US1] Repair pre-existing edit tests in `services/twig/internal/tui/plan_update_test.go` that assumed `blank=keep`/`null` semantics (e.g. `TestEditForm_RenameOnNameChange` setup comment and move-count assertions) so they reflect pre-filled values + change-detection.

**Checkpoint**: US1 is independently testable and constitutes the MVP — the requested edit-form fix is fully delivered.

---

## Phase 4: User Story 2 - Consistent presentation across all planning forms (Priority: P2)

**Goal**: The schedule-task and add-event forms share the same parity treatment delivered in Foundational; verify and polish them.

**Independent Test**: Open the Schedule-task form and the Add-event form; verify each shows its title, blank-line spacing, Save/Cancel buttons, the new help line, and that Enter in a field does not submit.

### Tests

- [ ] T013 [P] [US2] Test in `services/twig/internal/tui/plan_view_test.go`: the Schedule-task form renders title `Schedule task` and the Add-event form renders title `Add event`, each with `[ Save ]`/`[ Cancel ]` and the exact help line.
- [ ] T014 [P] [US2] Test in `services/twig/internal/tui/plan_update_test.go`: pressing Enter in a text field of the schedule and add-event forms does not submit (mode stays `planTaskTime` / `planEventForm`); Ctrl+S still submits.

### Implementation

- [ ] T015 [US2] In `services/twig/internal/tui/plan_update.go`, align the add-event/schedule placeholders with the parity tone and confirm the 2-field schedule form (`planTaskTime`) cycles focus correctly across Start, Duration, Save, Cancel (no out-of-range field access). Make only the minimal changes needed for consistency.

**Checkpoint**: All three planning forms verified consistent with the task form.

---

## Phase 5: Polish & Cross-Cutting Concerns

- [ ] T016 [P] Guard check: from repo root, `! grep -rn "blank=keep\|null=unschedule\|ctrl+s/enter" services/twig/internal/tui/*.go` returns clean (contract R5 + help-line FR-008).
- [ ] T017 [P] From `services/twig/`, run `go test ./...` and `go build -o twig ./cmd/twig`; both must pass.
- [ ] T018 Walk through `quickstart.md` against `./twig` and confirm Constitution Principle III (visual/interaction parity) and IV (message tone of any validation copy) per quality gates 5–6.

---

## Dependencies & Execution Order

```
Phase 1 (Setup: T001)
        │
        ▼
Phase 2 (Foundational: T002–T007)   ← BLOCKS everything below
        │
        ├───────────────┬───────────────┐
        ▼               ▼
Phase 3 (US1: T008–T012)   Phase 4 (US2: T013–T015)   ← independent of each other
        └───────────────┴───────────────┘
        ▼
Phase 5 (Polish: T016–T018)
```

- **Foundational ordering**: T002 [P] is standalone. T003 (focus predicates) precedes T004 and T005 (both consume the focus model). T004 (`plan_view.go`) and T005 (`update.go`) touch different files and can run in parallel once T003 lands. T006 [P] is largely independent (labels/placeholders). T007 comes after T003–T006.
- **US1 vs US2**: both depend only on Foundational and are independent of each other — different developers can take them in parallel.
- **Within a story**: write the test tasks first (TDD), then implementation.

## Parallel Execution Examples

- Foundational kickoff: run **T002** and **T006** in parallel (different files/concerns).
- After T003: run **T004** and **T005** in parallel (different files).
- US1 tests: **T008** and **T009** in parallel.
- US2 tests: **T013** and **T014** in parallel.
- Polish: **T016** and **T017** in parallel.

## Implementation Strategy

- **MVP = Phase 1 + Phase 2 + Phase 3 (US1).** This delivers the explicitly requested edit-form fix; because the renderer is shared, the schedule/add-event forms already inherit the parity presentation.
- **Incremental**: Phase 4 (US2) then adds explicit regression coverage and minor polish for the other two forms; Phase 5 finalizes guards, full build/test, and the manual constitution review.
- Keep changes minimal and within the five named files (Principle I); do not introduce a new form abstraction.
