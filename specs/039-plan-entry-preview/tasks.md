---
description: "Task list for Live Plan Entry Preview with Overlap Indication"
---

# Tasks: Live Plan Entry Preview with Overlap Indication

**Input**: Design documents from `/specs/039-plan-entry-preview/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/grid-preview.md, quickstart.md

**Tests**: INCLUDED. The repo follows strong table-driven test conventions (`internal/cli/plan_grid_test.go`, `internal/tui/plan_view_test.go`, `export_test.go` shims) and the plan/quickstart list new test files as deliverables. Test tasks are written first within each story.

**Organization**: Tasks are grouped by user story. US2 builds on US1 (a preview must exist before its overlap can be marked); US3 is satisfied by US1's accurate preview and adds verification only.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1 / US2 / US3 (Setup, Foundational, Polish carry no story label)
- File paths are repo-relative from `/home/user/dev/twig`

## Path Conventions

Single root Go module (`github.com/pboyd/twig`). Renderer lives in `internal/cli/`; TUI/form/model in `internal/tui/`. No `services/` or `services/twig-web/` changes (TUI-only feature).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm a clean baseline before touching rendering code.

- [ ] T001 Confirm baseline build and tests are green: run `go build ./... && go test ./internal/cli/... ./internal/tui/...` from repo root and note the current pass state (no code changes).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The shared `GridOptions` surface both stories depend on. Adding fields without behavior keeps existing output byte-for-byte identical.

**⚠️ CRITICAL**: Must complete before US1/US2 rendering work begins.

- [ ] T002 Extend `GridOptions` in `internal/cli/plan_grid.go` with the four preview fields per `contracts/grid-preview.md` §1: `PreviewID int32`, `PreviewStyle func(string) string`, `ConflictStyle func(string) string`, `PreviewConflictSlots map[int]bool` (struct + doc comments only; no rendering behavior yet).
- [ ] T003 [P] Add a backward-compatibility test in `internal/cli/plan_grid_test.go` asserting that `RenderGrid`/`RenderUntimed` output is unchanged when `PreviewID == 0` and `PreviewConflictSlots` is empty (locks contract §5 before behavior is added).

**Checkpoint**: `GridOptions` carries the preview fields; all existing tests still pass.

---

## Phase 3: User Story 1 - See where an edited entry will land before saving (Priority: P1) 🎯 MVP

**Goal**: While any timed form (Edit/Schedule/Add-event) is open, render a dashed, visually-distinct preview of the entry-being-edited in the day-planner grid at the form's current Start/Duration window; it moves/resizes live and disappears on cancel/save.

**Independent Test**: Open the Edit form for a scheduled entry, change Start and Duration, and confirm a dashed-bordered box appears at the matching window and updates without saving; cancel and confirm it disappears.

### Tests for User Story 1 ⚠️ (write first, expect failure)

- [ ] T004 [P] [US1] Add preview-construction tests in `internal/tui/plan_preview_test.go` (new file; expose helpers via `internal/tui/export_test.go` as needed): `buildPlanPreview` returns an entry with `Id == previewID`, name + start + duration derived from form fields for each of `planTaskTime`/`planEventForm`/`planEdit`; returns `nil` when Start is empty/invalid (FR-009); returns `nil` when `mode == planList`.
- [ ] T005 [P] [US1] Add dashed-rune rendering test in `internal/cli/plan_grid_test.go`: an entry whose `Id == PreviewID` renders its box with `┅`/`┇` (and corners `┏┓┗┛`), while sibling saved entries keep solid `━`/`┃` (contract §1 invariants 1–2).

### Implementation for User Story 1

- [ ] T006 [US1] Create `internal/tui/plan_preview.go` with `const previewID int32 = -1` and `func (m Model) buildPlanPreview() *planv1.PlanEntry` per `contracts/grid-preview.md` §2: gated on timed-form modes, parses Start via `timeparse.ParseStart` (nil if invalid), duration via `timeparse.ParseDurationOrEnd`, name from Name field or `findTask(m.tree, taskID)` for `planTaskTime`.
- [ ] T007 [US1] Implement dashed-rune substitution in `RenderGrid` (`internal/cli/plan_grid.go`): for the entry matching `opts.PreviewID`, emit `┅`/`┇` in place of `━`/`┃` across its top/interior/bottom/single/shared-border rows, and wrap its runes with `opts.PreviewStyle` when non-nil (contract §1.2, decision 2).
- [ ] T008 [US1] Wire preview rendering in `internal/tui/plan_grid.go` `planGridOptions`: set `opts.PreviewID = previewID` and, in styled mode, `opts.PreviewStyle` from the shared `dim` color in `internal/tui/theme.go` (Principle III).
- [ ] T009 [US1] Inject the preview into the rendered timed slice in `internal/tui/plan_view.go` (`renderPlanGrid` styled path and `renderPlanGridContent` plain path): build `timedOthers` excluding `m.plan.form.entryID` (self-exclusion, FR-006), append `buildPlanPreview()` when non-nil, and pass this combined slice to both `cli.GridWindow` (so the window grows to include the preview, decision 5) and `cli.RenderGrid`.
- [ ] T010 [US1] Confirm lifecycle correctness (FR-010/011/012) is purely derived: verify no `planState` field is added and `m.plan.entries` is never mutated (the exclusion/append operate on a local slice). Add a `plan_view_test.go` assertion that opening then cancelling a form leaves `m.plan.entries` unchanged.

**Checkpoint**: US1 fully functional — dashed preview tracks form input and vanishes on cancel/save. MVP deliverable.

---

## Phase 4: User Story 2 - Be warned when the new values overlap another entry (Priority: P1)

**Goal**: When the preview's window overlaps another timed entry on the same day, mark the overlapping portion of the preview in red (styled) or with a gutter `!` (plain); clear it when no longer overlapping; never flag self-overlap or touching boundaries.

**Independent Test**: With an existing entry at 10:00–11:00, enter a window of 10:30–11:30 and confirm the overlapping preview rows are marked; adjust to 11:00–12:00 (touching) and confirm the marking clears.

**Dependency**: Requires US1 (the preview entry + its slot rows must exist to mark).

### Tests for User Story 2 ⚠️ (write first, expect failure)

- [ ] T011 [P] [US2] Add conflict-detection tests in `internal/tui/plan_preview_test.go`: `planPreviewConflicts` returns the overlapping 15-min slot-start minutes for an overlapping preview, an empty/nil set for a touching boundary (FR-008), and excludes the edit target from comparison (FR-006).
- [ ] T012 [P] [US2] Add conflict-rendering tests in `internal/cli/plan_grid_test.go`: preview rows whose slot is in `PreviewConflictSlots` are wrapped with `ConflictStyle` in styled mode, and show a gutter `!` marker when `ConflictStyle == nil` (contract §1.4, §4); non-conflicting preview rows are not marked.

### Implementation for User Story 2

- [ ] T013 [US2] In `internal/tui/plan_preview.go`, add a local half-open interval overlap helper and `func planPreviewConflicts(preview *planv1.PlanEntry, others []*planv1.PlanEntry) map[int]bool` per `contracts/grid-preview.md` §2 (15-min slot granularity; `others` must already exclude the edit target).
- [ ] T014 [US2] Render conflict marking in `RenderGrid` (`internal/cli/plan_grid.go`): for preview rows at a slot in `opts.PreviewConflictSlots`, apply `opts.ConflictStyle` when non-nil; otherwise place `!` in the gutter marker column (the `▶` now-marker column), with the conflict `!` taking precedence on a coinciding row (contract §4).
- [ ] T015 [US2] Wire conflict options in `internal/tui/plan_grid.go` `planGridOptions` (and pass the edit-target-excluded `timedOthers` from `plan_view.go` per T009): set `opts.PreviewConflictSlots = planPreviewConflicts(preview, timedOthers)` and, in styled mode, `opts.ConflictStyle` from the shared `errorColor` in `internal/tui/theme.go`.

**Checkpoint**: US2 functional — overlapping portion shown in red (styled) / `!` (plain), clears on edit, self-overlap and touching excluded.

---

## Phase 5: User Story 3 - Catch unintended gaps from a wrong value (Priority: P3)

**Goal**: An unintended gap from a mistyped Start/Duration is visible because the live preview sits accurately relative to neighboring entries. No new production code beyond US1 — this phase verifies the behavior.

**Independent Test**: With an entry ending at 10:00, set the preview Start to 10:30 and confirm the preview sits with a visible half-hour gap; change Start to 10:00 and confirm it sits flush.

**Dependency**: Requires US1 (accurate live preview placement).

- [ ] T016 [US3] Add a gap-visibility test in `internal/cli/plan_grid_test.go` (or `internal/tui/plan_view_test.go`): a preview at 10:30 after an entry ending 10:00 renders empty grid rows for the 10:00–10:30 slots between them, and a preview at 10:00 renders flush (no empty slot) — asserting the preview occupies exactly its snapped window.

**Checkpoint**: Gap scenarios verified via the existing preview placement.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, consistency, and manual confirmation.

- [ ] T017 [P] Run `gofmt -l` and `go vet ./...`; fix any formatting/vet issues introduced in `internal/cli/plan_grid.go`, `internal/tui/plan_preview.go`, `internal/tui/plan_view.go`, `internal/tui/plan_grid.go`.
- [ ] T018 Run the full suite `go test ./...` from repo root and confirm all packages pass, including the T003 backward-compat guard.
- [ ] T019 [P] Verify Principle III compliance: confirm preview color = `dim` and conflict color = `errorColor` are the only palette references added, both from `internal/tui/theme.go` (no ad-hoc colors); and Principle IV: confirm no new user-facing prose was introduced (visual-only feature).
- [ ] T020 Walk through `specs/039-plan-entry-preview/quickstart.md` Scenarios A–D against `./twig` (styled) and once with `NO_COLOR=1` (plain) to confirm dashed preview, red/`!` overlap marking, untimed/invalid no-show, and save-replaces-preview.

---

## Dependencies & Execution Order

```text
Phase 1 (Setup: T001)
        │
        ▼
Phase 2 (Foundational: T002 → T003)   ← GridOptions fields; blocks all rendering
        │
        ▼
Phase 3 (US1 — MVP: T004,T005 [P tests] → T006 → T007 → T008 → T009 → T010)
        │
        ▼
Phase 4 (US2: T011,T012 [P tests] → T013 → T014 → T015)   ← needs US1 preview
        │
        ▼
Phase 5 (US3: T016)   ← needs US1 placement
        │
        ▼
Phase 6 (Polish: T017 [P], T018, T019 [P], T020)
```

**Story independence notes**:
- US1 is the standalone MVP (preview without conflict marking).
- US2 strictly extends US1 (overlap marking on the existing preview).
- US3 requires no new production code; it is a verification phase over US1.

**Same-file sequencing** (cannot be parallel):
- `internal/cli/plan_grid.go` is edited by T002 → T007 → T014 (struct, then dashed, then conflict) — strictly sequential.
- `internal/tui/plan_preview.go` is edited by T006 → T013 — sequential.
- `internal/tui/plan_grid.go` is edited by T008 → T015 — sequential.
- `internal/tui/plan_view.go` is edited by T009 (and asserted by T010) — single editor.

## Parallel Execution Examples

- **Phase 2**: T003 (test, different file) can be written alongside/after T002's struct change.
- **US1 tests**: T004 (`plan_preview_test.go`) and T005 (`plan_grid_test.go`) are different files → run in parallel before implementation.
- **US2 tests**: T011 (`plan_preview_test.go`) and T012 (`plan_grid_test.go`) → parallel.
- **Polish**: T017 (fmt/vet) and T019 (palette/prose audit) → parallel; T018 and T020 run after.

## Implementation Strategy

1. **MVP = Phase 1 + 2 + 3 (US1)**: ships the live dashed preview — already delivers the core "see where it lands" value and the gap-avoidance benefit (US3) implicitly.
2. **Increment 2 = Phase 4 (US2)**: adds the explicit overlap marking — the second P1.
3. **Verify = Phase 5 + 6**: gap-scenario test, full suite, manual quickstart in both styled and plain modes.

Total: **20 tasks** — Setup 1, Foundational 2, US1 7 (incl. 2 tests), US2 5 (incl. 2 tests), US3 1, Polish 4.
