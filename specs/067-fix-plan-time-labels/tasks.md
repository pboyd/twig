---

description: "Task list for 067-fix-plan-time-labels"
---

# Tasks: Show Actual Times in Plan Grid Labels

**Input**: Design documents from `/specs/067-fix-plan-time-labels/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/grid-label.md, quickstart.md

**Tests**: Test tasks ARE included. The spec did not request TDD in so many words, but research.md establishes that the existing suite uses only 15-minute-aligned times and therefore *passes against the buggy code* — both defects shipped precisely because no test could see them. Failing tests written first are the only way to prove either fix works.

**Organization**: Grouped by user story. US1 and US2 touch different files with no shared state and can be done in either order or simultaneously.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1, US2, US3 per spec.md

## Path Conventions

Root Go module (`github.com/pboyd/twig`). All paths are repo-relative. The `api/` and `services/twig/` modules are not touched by this feature.

---

## Phase 1: Setup (Baseline)

**Purpose**: Establish a known-good starting point so any later failure is attributable to this feature.

- [X] T001 Run `go test ./...` from repo root and confirm the suite passes before any change; note the current pass count
- [X] T002 Build and launch the TUI (`go build -o twig ./cmd/twig && ./twig`), create plan entries at 13:00 (50 min) and 13:50 (20 min), and confirm both defects reproduce: labels read `13:00-14:00` / `13:45-14:15`, and editing the first entry shows a red conflict marker at the 13:45 slot

---

## Phase 2: Foundational

**Purpose**: None required.

This feature adds no shared infrastructure, no new types, and no migrations. The two defects live in separate files (`internal/cli/plan_grid.go` and `internal/tui/plan_preview.go`) and neither fix depends on the other. User story work begins immediately after the baseline.

**Checkpoint**: Baseline captured — US1 and US2 can proceed in parallel.

---

## Phase 3: User Story 1 - Grid labels report the real scheduled times (Priority: P1) 🎯 MVP

**Goal**: Entry labels in the plan grid show the entry's exact start and end times while box geometry stays byte-identical.

**Independent test**: Render a grid containing entries at 13:00+50min and 13:50+20min; labels read `13:00-13:50` and `13:50-14:10`, boxes still span 13:00→14:00 and 13:45→14:15.

### Tests for User Story 1

- [X] T003 [P] [US1] Add a failing test in `internal/cli/plan_grid_test.go` asserting an entry at 13:50 with duration 20 renders the label text `13:50-14:10` (currently renders `13:45-14:15`)
- [X] T004 [P] [US1] Add a failing test in `internal/cli/plan_grid_test.go` asserting an entry at 13:00 with duration 50 renders `13:00-13:50` and that its box still occupies the rows from 13:00 through 14:00 (geometry unchanged — this is the SC-005 guard)
- [X] T005 [P] [US1] Add a test in `internal/cli/plan_grid_test.go` covering `HideID: false`, asserting the `[<id>] ` prefix is preserved ahead of the exact time range (FR-007)
- [X] T006 [P] [US1] Add a test in `internal/cli/plan_grid_test.go` for a 15-minute-aligned entry (09:00, duration 30) asserting the label `09:00-09:30` is unchanged from current behavior (no-regression guard)

### Implementation for User Story 1

- [X] T007 [US1] In the `layouts` loop of `RenderGrid` in `internal/cli/plan_grid.go` (~line 121), add exact-interval locals `es := int(e.GetStartMinute())` and `ee := es + int(e.DurationMinute)`, leaving `sn`/`se` untouched for `tl`/`bl` row placement
- [X] T008 [US1] In `internal/cli/plan_grid.go` lines 123 and 125, replace the `sn`/`se` format arguments with `es`/`ee` in both the `HideID` and ID-prefixed `fmt.Sprintf` calls; do not alter the format string, the branch structure, or the following `wrapLabel` call
- [X] T009 [US1] Verify no other call site changed: confirm `internal/cli/plan_grid.go` lines 64-65, 101-102, 156, 244, 702-703, and 741 still use snapped values (per the research.md inventory — line 244 is the hour gutter and must stay snapped per FR-008)

**Checkpoint**: `go test ./internal/cli/` passes; grid labels tell the truth. This alone is a shippable increment.

---

## Phase 4: User Story 2 - Overlap warnings reflect real overlaps (Priority: P1)

**Goal**: Conflicts are reported only for genuine overlaps of at least one minute, and marked slots point at where the overlap actually is.

**Independent test**: With an entry at 13:50–14:10 present, preview an entry at 13:00–13:50 and observe no conflict anywhere in the grid.

**Independence note**: Touches only `internal/tui/plan_preview.go`. Fully parallel with Phase 3.

### Tests for User Story 2

- [X] T010 [P] [US2] Add a failing test in `internal/tui/plan_preview_test.go`: preview 13:00–13:50 against an existing entry at 13:50–14:10 yields no conflicts (the user-reported false positive)
- [X] T011 [P] [US2] Add a test in `internal/tui/plan_preview_test.go`: preview 13:00–14:00 against 13:50–14:10 yields exactly `{825}` — the overlap region 13:50–14:00 maps to the 13:45 slot only, proving marking is scoped to the overlap and not the whole preview
- [X] T012 [P] [US2] Add a test in `internal/tui/plan_preview_test.go`: preview 13:00–13:05 against 13:10–13:20 yields no conflicts, covering two non-overlapping entries inside one 15-minute slot
- [X] T013 [P] [US2] Add a test in `internal/tui/plan_preview_test.go`: preview 10:00–11:00 fully contained by an entry at 09:00–12:00 yields `{600, 615, 630, 645}`
- [X] T014 [P] [US2] Add a test in `internal/tui/plan_preview_test.go` asserting every returned key is a multiple of 15 and lies within the preview's snapped box (contract §3 requirement 7 — a key outside the box would silently never render)

### Implementation for User Story 2

- [X] T015 [US2] Rewrite the `others` loop body in `planPreviewConflicts` in `internal/tui/plan_preview.go` (~lines 122-142) to compute `ovStart := max(pStart, oStart)` and `ovEnd := min(pEnd, oEnd)`, `continue` when `ovStart >= ovEnd`, and otherwise mark slots from `snapDown15(ovStart)` while `t < ovEnd` stepping 15
- [X] T016 [US2] Remove both artificial `if xEnd <= xStart { xEnd = xStart + 15 }` clamps (preview at ~line 117, other entry at ~line 128) per research.md Decision 3; retain `snapDown15`, which is still needed to map the overlap region onto slot keys
- [X] T017 [US2] Update the `planPreviewConflicts` doc comment in `internal/tui/plan_preview.go` to describe exact-interval comparison and overlap-region slot marking, replacing the current slot-iteration description; keep the note that self-exclusion is the caller's responsibility
- [X] T018 [US2] Confirm the pre-existing `TestPlanPreviewConflicts_Overlap` and `TestPlanPreviewConflicts_Touching` still pass unmodified — both use aligned times and remain valid under the new logic

**Checkpoint**: `go test ./internal/tui/` passes; back-to-back entries no longer cry wolf.

---

## Phase 5: User Story 3 - The preview label also shows exact times (Priority: P2)

**Goal**: The transient preview box labels itself with the exact times the user is typing.

**Delivered by US1**: The preview is an ordinary `planv1.PlanEntry` (with `Id = -1`) passed through the same `RenderGrid` label path, so T007–T008 fix it automatically. This phase adds verification only — no production code changes. Do not write a second label path.

### Tests for User Story 3

- [X] T019 [P] [US3] Add a test in `internal/tui/plan_view_test.go` (or `plan_preview_test.go`, wherever preview rendering is currently exercised) asserting a preview built from start `13:00` and duration `50` renders the label `13:00-13:50`
- [X] T020 [P] [US3] Extend that test to re-render with duration `20` and assert the label updates to `13:00-13:20`, confirming the preview label tracks live form edits

**Checkpoint**: Preview labels match what the user typed.

---

## Phase 6: Polish & Verification

- [X] T021 Run `go test ./...` from repo root and confirm the full suite passes with no regressions against the T001 baseline
- [X] T022 Rebuild and re-run the TUI scenario from T002; confirm labels now read `13:00-13:50` and `13:50-14:10`, no conflict marker appears when editing the first entry, and box positions are visually identical to the T002 screenshots (SC-005)
- [X] T023 In the running TUI, verify a genuine overlap still warns: edit the first entry to 13:00–14:00 and confirm the conflict marker appears at the 13:45 slot (SC-004 — guards against over-correcting into silence)
- [X] T024 Verify the hour gutter still shows 15-minute boundary times (FR-008) and that a short entry (13:05, duration 5) labels as `13:05-13:10` while still drawing one full slot
- [X] T025 Re-read `specs/067-fix-plan-time-labels/contracts/grid-label.md` and confirm the implementation matches every normative requirement in §1 and §3 (constitution Quality Gate 1)

---

## Dependencies

```text
Phase 1 (T001-T002)  baseline
        ├─→ Phase 3 US1 (T003-T009)  ─┐
        └─→ Phase 4 US2 (T010-T018)  ─┤   independent, parallel
                                       ↓
              Phase 5 US3 (T019-T020)  requires US1 (T007-T008) only
                                       ↓
              Phase 6 (T021-T025)      requires all
```

- **US1 ⊥ US2**: different files, no shared symbols. Either can ship alone.
- **US3 → US1**: verification-only; its assertions cannot pass until T008 lands.
- Within each story, tests (written first, failing) precede implementation.

## Parallel Execution Examples

**Two developers, or one agent with two work streams:**

```text
Stream A: T003, T004, T005, T006 (all [P], same file — write together, then) → T007 → T008 → T009
Stream B: T010, T011, T012, T013, T014 (all [P], same file) → T015 → T016 → T017 → T018
```

Both streams converge at T019.

**Note on [P] within a file**: T003–T006 all edit `plan_grid_test.go` and T010–T014 all edit `plan_preview_test.go`. They are marked `[P]` because they are independent *test cases* that can be authored in one pass; they are not separate concurrent file writes.

## Implementation Strategy

**MVP = Phase 3 (US1) alone.** It fixes the defect the user actually opened with — the grid stating times that are wrong — and is independently shippable in about three lines of production code.

**Recommended order**: Do US2 immediately after. Both are P1, and the false conflict warning is arguably the more corrosive of the two, since it teaches users to ignore a signal that will matter later.

**Total**: 25 tasks — 7 US1, 9 US2, 2 US3, 2 setup, 5 polish. Production code changed: ~10 lines across 2 files. The task count is dominated by tests, which is appropriate for a defect class that the existing suite provably could not detect.
