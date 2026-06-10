# Tasks: TUI Calendar Date Picker

**Input**: Design documents from `/specs/046-tui-date-picker/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/calendar-widget.md, quickstart.md

**Tests**: Included — research.md R6 defines the test strategy and the contract's behavioral guarantees are test-verified (round-trip safety, lossless cancel, text-path regression). Tests are written before the implementation they cover, following the package's table-driven conventions.

**Organization**: Tasks are grouped by user story so each story is an independently testable increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)
- All paths are relative to the repository root; all work is in the root CLI/TUI module

## Phase 1: Setup

**Purpose**: Confirm a green baseline on the feature branch — no scaffolding is needed (existing package, no new dependencies)

- [ ] T001 Verify baseline build and tests pass on branch `046-tui-date-picker`: `go test ./internal/tui/...` and `go build ./cmd/twig`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared pieces every user story builds on — the key binding and the widget's core state per data-model.md

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T002 [P] Add `Calendar` key binding (`ctrl+g`, help "pick a date") to `KeyMap` and `DefaultKeyMap()` in internal/tui/keymap.go
- [ ] T003 [P] Create `calendarModel` skeleton in internal/tui/calendar.go: struct fields per data-model.md (`selected`, `today`, `keepTime`, `rfc3339Out`), constructor `newCalendar(fieldValue string, now time.Time, rfc3339Field bool)` that parses the field value via `cli.ParseDue` and falls back to today on empty/unparseable input (FR-003)

**Checkpoint**: Foundation ready — `calendarModel` exists and can be constructed from any field value

---

## Phase 3: User Story 1 - Pick a date from a calendar (Priority: P1) 🎯 MVP

**Goal**: From a focused Due or Snooze field, `ctrl+g` opens an inline month calendar; the user moves the selection and confirms with `enter` (field gets the date) or backs out with `esc` (field untouched).

**Independent Test**: Open the edit form, focus a date field, press `ctrl+g`, move the selection a few days, press `enter` — the field contains the picked date and the task saves. Re-open, press `ctrl+g` then `esc` — the field is unchanged.

### Tests for User Story 1

> Write these first; they MUST fail before the implementation tasks below

- [ ] T004 [P] [US1] Table-driven constructor tests in internal/tui/calendar_test.go: opens on field's date for `YYYY-MM-DD` and RFC3339 values, opens on today for empty and garbage input (US1 scenario 4, FR-003, spec edge case "malformed text")
- [ ] T005 [P] [US1] Render tests in internal/tui/calendar_test.go: header shows month+year, weekday row `Su Mo Tu We Th Fr Sa`, selected day wrapped in `highlightStyle`, today marked, out-of-month padding cells dimmed, total width ≈22 columns (contract "Visual contract")
- [ ] T006 [P] [US1] Form integration tests in internal/tui/edit_test.go: `ctrl+g` on Due/Snooze opens the calendar (and does nothing on Name/Description/Estimate); open → move → `enter` writes `YYYY-MM-DD` into the field; open → `esc` leaves the field byte-for-byte unchanged and the form still open; while open, form keys (`ctrl+s`, text runes) are swallowed; `tab`/`shift+tab` close the calendar unchanged and cycle focus (FR-001, FR-005, FR-006, contract guarantees 2, 3)

### Implementation for User Story 1

- [ ] T007 [US1] Implement day/week selection movement in internal/tui/calendar.go: `←/h` `→/l` ±1 day, `↑/k` `↓/j` ±7 days, flowing across month boundaries via `time.Time.AddDate` (US1 scenario 2; US2 scenarios 1–2 get month/year jumps later)
- [ ] T008 [US1] Implement `View()` month grid rendering in internal/tui/calendar.go using only the shared palette from internal/tui/theme.go (`accent`, `dim`, `highlightStyle`) per the contract's visual rules
- [ ] T009 [US1] Implement confirm/cancel result handling in internal/tui/calendar.go: `enter` yields the formatted date string (`YYYY-MM-DD` for now), `esc` yields cancellation
- [ ] T010 [US1] Wire the calendar into internal/tui/edit.go: add `calendar *calendarModel` + `calendarFor int` to `editFormModel`; `ctrl+g` on `focusDue`/`focusSnooze` opens it; while open, route every `tea.KeyPressMsg` to the calendar first; on confirm `SetValue` the owning field and close; render the grid directly beneath the owning field in `View(width)`
- [ ] T011 [US1] Add the contract's copy in internal/tui/edit.go: hint `ctrl+g: summon the calendar` under a focused date field when closed, and the in-calendar key-hint line `enter: pick  esc: never mind` (extended in US2) — tone per Principle IV

**Checkpoint**: MVP — calendar opens, navigates by day/week, confirms and cancels correctly; `go test ./internal/tui/...` green

---

## Phase 4: User Story 2 - Keyboard navigation across days, months, and years (Priority: P2)

**Goal**: Fast travel — `[`/`]` jump months, `{`/`}` jump years, `t` returns to today, all with day-of-month clamping so the selection is never invalid.

**Independent Test**: Open the calendar, press `]` from Jan 31 and land on Feb 28/29; press `}` from Feb 29 and land on Feb 28; press `t` and land on today.

### Tests for User Story 2

- [ ] T012 [P] [US2] Table-driven navigation tests in internal/tui/calendar_test.go: `[`/`]` month jumps incl. clamping (Jan 31 → Feb 28, Mar 31 → Apr 30, leap-year Feb 29), `{`/`}` year jumps incl. Feb 29 → Feb 28 clamp, `t` jumps to today, day movement across month/year boundaries (US2 scenarios 1–4, FR-004, FR-010, spec edge cases "February 29" and "month jumps from day 31")

### Implementation for User Story 2

- [ ] T013 [US2] Implement month/year jumps with day clamping and `t` (today) in internal/tui/calendar.go: compute target month length explicitly (no `AddDate` overflow rollover) per data-model.md invariants
- [ ] T014 [US2] Extend the in-calendar hint line in internal/tui/edit.go to the full contract copy: `enter: pick  esc: never mind  t: today  [/]: month  {/}: year`

**Checkpoint**: Any date within ±2 years reachable in ≤15 keystrokes (SC-002); all clamping tests green

---

## Phase 5: User Story 3 - Text entry still works (Priority: P3)

**Goal**: The typed path is untouched, and picking a date on a Due field that already holds an RFC3339 timestamp preserves its time of day.

**Independent Test**: Type dates into both fields with the calendar closed and save — identical behavior to before the feature. Put `2026-06-10T15:00:00Z` in Due, pick June 12 from the calendar — field reads `2026-06-12T15:00:00Z`.

### Tests for User Story 3

- [ ] T015 [P] [US3] Output-format tests in internal/tui/calendar_test.go: confirm on a Due field opened from `2026-06-10T15:00:00Z` emits `2026-06-12T15:00:00Z` for June 12 (time preserved, RFC3339 out); confirm from a date-only or empty Due emits `YYYY-MM-DD`; Snooze always emits `YYYY-MM-DD`; round-trip property — every emitted string is accepted by `cli.ParseDue` (FR-005, FR-008, SC-003, contract guarantees 1, 4)
- [ ] T016 [P] [US3] Text-path regression tests in internal/tui/edit_test.go: with the calendar closed, typing/cursor keys on date fields behave exactly as today and the calendar never opens uninvited; typed values flow into `editSavedMsg` unchanged (US3 scenarios 1, 3; FR-007; contract guarantees 2, 7)

### Implementation for User Story 3

- [ ] T017 [US3] Implement `keepTime`/`rfc3339Out` output formatting in internal/tui/calendar.go: capture the clock-time offset and format flag at open time (Due field only), re-apply on confirm so only the date portion changes (FR-008)

**Checkpoint**: All three input flavors (typed date, typed timestamp, picked date) coexist; round-trip property test green

---

## Phase 6: User Story 4 - Clear an optional date (Priority: P3)

**Goal**: Optional stays optional — a date set by mistake can be removed and the task saves with no date.

**Independent Test**: Set a due date, clear the field text, save — the task has no due date.

### Tests for User Story 4

- [ ] T018 [US4] Clearing test in internal/tui/edit_test.go: pick a date via the calendar, then delete the field text and save — `editSavedMsg.dueStr`/`snoozeStr` are empty and the existing save path treats them as unset (US4 scenario 1, FR-009); confirms no calendar state lingers after a clear

**Checkpoint**: All four user stories independently verified

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final verification against the constitution's quality gates and the quickstart checklist

- [ ] T019 [P] Run the full suite and build from repo root: `go test ./...` and `go build -o twig ./cmd/twig`
- [ ] T020 [P] Review all new user-facing copy in internal/tui/edit.go and internal/tui/calendar.go for Principle IV tone and Principle III consistency (palette-only colors, binding conventions) against contracts/calendar-widget.md
- [ ] T021 Walk the manual scenario table in specs/046-tui-date-picker/quickstart.md against a running server (`make dev`) and record results

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — T002 and T003 are parallel ([P], different files)
- **User Stories (Phases 3–6)**: All depend on Phase 2
  - US2, US3, US4 build on US1's wiring (same files: calendar.go, edit.go) — sequential execution in priority order is the practical path for a single developer
  - US3's T015 only needs calendar.go from US1 (not US2) and could start right after Phase 3
- **Polish (Phase 7)**: Depends on Phases 3–6

### User Story Dependencies

- **US1 (P1)**: Only Foundational — delivers the MVP
- **US2 (P2)**: Extends US1's `calendarModel` navigation (calendar.go)
- **US3 (P3)**: Extends US1's output formatting (calendar.go); independent of US2
- **US4 (P3)**: Test-only story exercising US1's wiring (edit_test.go); independent of US2/US3

### Within Each User Story

- Test tasks first; they must fail before the implementation tasks they cover
- calendar.go logic before edit.go wiring (T007–T009 before T010)

### Parallel Opportunities

- Phase 2: T002 ∥ T003 (keymap.go vs calendar.go)
- Phase 3: T004 ∥ T005 ∥ T006 (test authoring in different files/areas)
- Phase 5: T015 ∥ T016 (calendar_test.go vs edit_test.go)
- Phase 7: T019 ∥ T020

## Parallel Example: User Story 1

```bash
# Author all US1 tests together (different files/test groups):
Task: "Constructor tests in internal/tui/calendar_test.go"        # T004
Task: "Render tests in internal/tui/calendar_test.go"             # T005
Task: "Form integration tests in internal/tui/edit_test.go"       # T006
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 → Phase 2 (T001–T003)
2. Phase 3 (T004–T011): calendar opens, day/week navigation, confirm/cancel
3. **STOP and VALIDATE**: `go test ./internal/tui/...` + quickstart steps 1–6 manually
4. This alone delivers the headline feature — a usable calendar picker

### Incremental Delivery

1. + US2 (T012–T014): month/year fast travel → validate clamping edge cases
2. + US3 (T015–T017): time preservation + text-path regression proof
3. + US4 (T018): clearing regression proof
4. Phase 7 (T019–T021): full-suite run, tone/consistency review, manual quickstart walk

## Notes

- Single contract file `contracts/calendar-widget.md` governs all stories; T006, T012, T015, T016 explicitly verify its behavioral guarantees (constitution Quality Gate 1)
- No new dependencies, no API/proto/sqlc changes anywhere in this feature
- Commit after each task or logical group; every checkpoint is a safe stopping point
