---
description: "Task list for Planner Vertical Space"
---

# Tasks: Planner Vertical Space

**Input**: Design documents from `/specs/035-planner-vertical-space/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/grid-window.md, quickstart.md

**Tests**: Included. This is a rendering change guarded by a hard **CLI byte-for-byte parity** invariant (FR-008) and the codebase already maintains table-driven suites in `internal/cli` and `internal/tui`.

**Organization**: Tasks are grouped by user story (from spec.md) so each story is an independently testable increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 = fill space, US2 = anchor to now, US3 = untimed priority

## Path Conventions

Single Go module at `services/twig/`. All paths below are relative to repo root. Run `go` commands from `services/twig/`.

---

## Phase 1: Setup

**Purpose**: Confirm a green baseline before touching the shared rendering code.

- [x] T001 Confirm baseline green: run `go test ./...` from `services/twig/` and note current pass state of `internal/cli` and `internal/tui` (establishes the CLI-parity reference for later diffs).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The window-override plumbing in `RenderGrid` that every user story relies on. Until this is done, the TUI cannot pass a computed window.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T002 Add `WindowStartMin *int` and `WindowEndMin *int` optional fields to `GridOptions` in `services/twig/internal/cli/plan_grid.go`, with doc comments per `specs/035-planner-vertical-space/contracts/grid-window.md`.
- [x] T003 Modify `RenderGrid` window initialization in `services/twig/internal/cli/plan_grid.go` to honor the overrides: when `WindowStartMin`/`WindowEndMin` are non-nil use them as `winStart`/`winEnd`, and skip the auto-expansion of any overridden side inside the entry loop (nil side still expands as today). (depends on T002)
- [x] T004 [P] Add CLI-parity + override tests in `services/twig/internal/cli/plan_grid_test.go`: assert `GridOptions{}` output is unchanged, plus start-only / end-only / both-set overrides verifying no auto-expansion occurs on the fixed side. Keep `TestRenderGrid_EmptyDay`, `TestRenderGrid_WindowExtensionEarly/Late` green and unchanged. (depends on T003)

**Checkpoint**: `RenderGrid` can draw any explicit window; CLI default path proven identical.

---

## Phase 3: User Story 1 - Fill empty space with more hours (Priority: P1) 🎯 MVP

**Goal**: On a tall Planning tab, the timed grid extends past 17:00 to fill available vertical space instead of leaving a blank gap.

**Independent Test**: With a tall terminal and a day whose entries all end before 17:00, the grid draws hour rows beyond 17:00 filling the pane; shrinking the terminal reduces the hours shown.

### Implementation for User Story 1

- [x] T005 [US1] Implement `GridWindow(entries, now, day, availableRows) (startMin, endMin int)` in `services/twig/internal/cli/plan_grid.go`: extract the entry-extended default window (`baseStart`/`baseEnd`) into a small helper shared with `RenderGrid` (no duplication), then implement **FILL** mode (`start=baseStart`, `end=min(start+(availableRows-1)*15, 1440)`); for the constrained case fall back to top-truncate (`start=baseStart`) as a placeholder until US2. (depends on T003)
- [x] T006 [US1] Wire the styled live path `renderPlanGrid` in `services/twig/internal/tui/plan_view.go`: render untimed + separator, count their lines, compute `availableRows = max(1, height - untimedLines - sepLines)`, call `cli.GridWindow`, and pass `WindowStartMin: &start, WindowEndMin: &end` in `GridOptions`; keep the existing `lines[:height]` trim as a backstop. (depends on T005)
- [x] T007 [US1] Apply the same wiring to the unstyled live path `renderPlanGridContent` and the test-only helper `renderPlanningView` in `services/twig/internal/tui/plan_view.go`, accounting for their inline header line in `availableRows`. (depends on T005, T006)
- [x] T008 [P] [US1] FILL unit tests in `services/twig/internal/cli/plan_grid_test.go`: `availableRows > baseRows` extends the end to later hours; end clamped at 24:00; `availableRows == baseRows` returns the default window; 08:00 start preserved. (depends on T005)
- [x] T009 [US1] TUI test in `services/twig/internal/tui/plan_view_test.go`: a tall height renders hour rows past 17:00 and leaves no large blank gap below the grid. (depends on T006)

**Checkpoint**: Tall terminals fill space — MVP demoable.

---

## Phase 4: User Story 2 - Prioritize upcoming events when space is tight (Priority: P1)

**Goal**: On a short terminal viewing today, the timed grid begins at the current-time block (after untimed entries); earlier hours are omitted.

**Independent Test**: On a short terminal, today, mid-afternoon, with entries before and after now — the grid's first row is the now-block, morning hours are absent, and an upcoming entry is visible.

### Implementation for User Story 2

- [x] T010 [US2] Add **ANCHOR** and **TOP-TRUNCATE** branches to `GridWindow` in `services/twig/internal/cli/plan_grid.go`: when `availableRows < baseRows` AND `day` is today (`now.Format("2006-01-02")==day`) AND `nowBlock >= baseStart` → `start = snapDown15(nowBlock)`; otherwise `start = baseStart`; `end = min(start+(availableRows-1)*15, 1440)`. Replaces the US1 placeholder fallback. (depends on T005)
- [x] T011 [P] [US2] Unit tests in `services/twig/internal/cli/plan_grid_test.go`: constrained + today anchors to the now-block; constrained + non-today starts at `baseStart`; now before window → `baseStart`; late now anchors late; end clamped at 24:00. (depends on T010)
- [x] T012 [US2] TUI test in `services/twig/internal/tui/plan_view_test.go`: short height on today starts the grid at the current-time block, omits earlier hours, and keeps a later entry visible. (depends on T010, T007)
- [x] T013 [US2] TUI test in `services/twig/internal/tui/plan_view_test.go`: short height on a non-today day starts at the default top (no anchoring). (depends on T010, T007)

**Checkpoint**: Short terminals surface upcoming events; today vs. non-today behave per FR-005.

---

## Phase 5: User Story 3 - Untimed entries remain visible (Priority: P2)

**Goal**: Untimed entries (and their separator) always render before the timed grid; the timed grid takes only the remaining rows.

**Independent Test**: On a short terminal with several untimed entries, all untimed boxes + separator appear before any timed row; when untimed alone exceeds the height, nothing crashes and the timed grid is reduced.

### Implementation for User Story 3

- [x] T014 [US3] Verify/harden `availableRows` accounting in all three paths in `services/twig/internal/tui/plan_view.go` so untimed pane + separator lines are always subtracted before computing the timed window, and the underflow case (untimed alone ≥ inner height) clamps to ≥1 row (or zero timed rows) without overflow. (depends on T006, T007)
- [x] T015 [P] [US3] TUI tests in `services/twig/internal/tui/plan_view_test.go`: a constrained view renders all untimed entries + separator before any timed row; an over-tall untimed set degrades gracefully (no panic, no overflow past the pane). (depends on T014)

**Checkpoint**: All three user stories independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T016 [P] Execute `specs/035-planner-vertical-space/quickstart.md` scenarios A–E against a freshly built `./twig` binary and confirm each expectation.
- [x] T017 Run `go test ./...` from `services/twig/` and confirm all suites green, with the CLI-parity assertions (`TestRenderGrid_EmptyDay`, `*_WindowExtension*`) unchanged.
- [x] T018 [P] Self-review against Constitution Principles I (no new abstraction beyond the one shared baseline helper) and III (reuses existing grid geometry/theme); confirm no new user-facing copy was introduced (Principle IV n/a).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none.
- **Foundational (Phase 2)**: after Setup. BLOCKS all user stories.
- **US1 (Phase 3)**: after Foundational. Introduces the shared `GridWindow` helper and TUI wiring.
- **US2 (Phase 4)**: after US1 (extends `GridWindow`; reuses US1 TUI wiring).
- **US3 (Phase 5)**: after US1 wiring (hardens the row accounting that US1 introduced).
- **Polish (Phase 6)**: after all desired stories.

### Critical path

T001 → T002 → T003 → T005 → T006 → T007 → (T010 for US2) / (T014 for US3) → T017

### Within each story

- `GridWindow` logic (cli) before TUI wiring that calls it.
- Implementation before its tests where the test asserts behavior; cli unit tests `[P]` run alongside tui tests (different files).

### Parallel Opportunities

- T004 (cli test) is independent of US-phase tui work once T003 lands.
- T008 (cli FILL tests) ∥ T009 (tui fill test) — different files.
- T011 (cli anchor tests) ∥ T012/T013 (tui tests) — different files; T012 and T013 share `plan_view_test.go` so run sequentially.
- T016 and T018 (polish) can run in parallel.

---

## Parallel Example: User Story 1

```bash
# After T005 lands, the cli and tui tests touch different files:
Task: "T008 FILL unit tests in services/twig/internal/cli/plan_grid_test.go"
Task: "T009 tall-height TUI test in services/twig/internal/tui/plan_view_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 Setup → green baseline.
2. Phase 2 Foundational → window override + parity tests.
3. Phase 3 US1 → fill mode + TUI wiring. **STOP and validate** on a tall terminal.

### Incremental Delivery

1. Foundational ready → override plumbing proven CLI-safe.
2. US1 → tall terminals fill space (MVP, demo).
3. US2 → short terminals anchor to now (demo).
4. US3 → untimed priority hardened (demo).
5. Polish → quickstart + full regression.

---

## Notes

- [P] = different files, no incomplete dependency.
- The single most important guard: `GridOptions{}` (CLI) output must never change — re-run the parity tests after every cli change.
- `internal/cli/plan_grid.go` is touched by T002, T003, T005, T010 — these are sequential (same file).
- `internal/tui/plan_view.go` is touched by T006, T007, T014 — sequential (same file).
- `internal/cli/plan_grid_test.go` is touched by T004, T008, T011 — sequential (same file).
- `internal/tui/plan_view_test.go` is touched by T009, T012, T013, T015 — sequential (same file).
- Commit after each task or logical group.
