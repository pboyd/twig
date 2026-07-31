---
description: "Task list for 071-plan-task-descriptions"
---

# Tasks: Task Descriptions on the Planning Tab

**Input**: Design documents from `/specs/071-plan-task-descriptions/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/plan-detail-contract.md, quickstart.md

**Tests**: Test tasks ARE included. `research.md` R6 specifies the test coverage, `plan.md` sequences it, and the repo's testing conventions expect it. Tests are written before the implementation they cover.

**Organization**: Grouped by user story so each is independently verifiable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel — different files, no dependency on incomplete work
- **[Story]**: US1, US2, US3 — maps to the user stories in `spec.md`
- Exact file paths are given in every task

## Path Conventions

Root CLI module (`github.com/pboyd/twig`), package `internal/tui`. Only three files are touched:

- `internal/tui/plan_view.go` — the detail renderer
- `internal/tui/view.go` — the two-pane layout
- `internal/tui/plan_view_test.go` — tests

**A note on parallelism**: this feature is small and concentrated in two production files. Genuine `[P]` opportunities are therefore rare, and marking same-file tasks as parallel would be false. Only three tasks below carry `[P]`. That is the honest count, not an oversight.

---

## Phase 1: Setup

**Purpose**: Establish a known-good baseline before changing anything.

- [ ] T001 Run `go test ./internal/tui/` from the repo root and confirm it passes before any edit, so later failures are attributable to this feature

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: None required.

This phase is intentionally empty, and that is a finding rather than a gap. Everything the feature depends on already exists and needs no preparation:

- `renderPlanDetail` (`internal/tui/plan_view.go:304`) already receives both the linked `*taskv1.Task` and the `*markdown.Renderer` — no signature change.
- `internal/tui/plan_view.go` already imports `fmt`, `strings`, and `internal/markdown` — no import changes.
- `m.tree` and `m.md` are already populated on the `Model` — no new state.
- No proto, schema, or dependency work — no `make proto`, no `sqlc generate`.

**Checkpoint**: User story work can begin immediately after T001.

---

## Phase 3: User Story 1 — Read a scheduled task's description without leaving the plan (Priority: P1) 🎯 MVP

**Goal**: Selecting a plan entry linked to a task shows that task's description in the Details pane; every other case shows nothing at all.

**Independent Test**: Schedule a task that has a description, select its entry on the Planning tab, and confirm the description appears without leaving the tab. Select an event and a description-less task and confirm the pane is unchanged.

### Tests for User Story 1

> Write these first and confirm they FAIL before T005.

- [ ] T002 [US1] Add `TestRenderPlanDetail_DescriptionShown` to `internal/tui/plan_view_test.go`: a `PlanEntry` with `TaskId != 0` and a `Task` whose `Description` is non-empty, `md == nil`, asserting the description text appears in the output (rule D5, contract C1.1)
- [ ] T003 [US1] Add absence tests to `internal/tui/plan_view_test.go` covering rules D2–D4 and contract C1.2–C1.4: an event entry (`TaskId == 0`, `task == nil`), an entry whose task is missing from the tree (`TaskId != 0`, `task == nil`), a task with `Description == ""`, and a task with a whitespace-only description. Each asserts no description block AND no trailing blank separator line
- [ ] T004 [US1] Extend the empty-description case in `internal/tui/plan_view_test.go` to assert output is byte-identical to a fixture captured from the current renderer, pinning contract C1.4's "unchanged pane" guarantee

### Implementation for User Story 1

- [ ] T005 [US1] In the unstyled path of `renderPlanDetail` (`internal/tui/plan_view.go`, after the pomodoro row near line 347), append the description block guarded by `entry.TaskId != 0 && task != nil && strings.TrimSpace(task.GetDescription()) != ""`, emitting one blank line then `md.Render(desc, markdown.Options{Width: width, Styled: false})` with a `wrapDescription(desc, width)` fallback when `md == nil`
- [ ] T006 [US1] Apply the same block to the styled path of `renderPlanDetail` (`internal/tui/plan_view.go`, after the pomodoro row near line 372), passing `Styled: true`
- [ ] T007 [US1] Delete the now-obsolete `_ = width` marker at `internal/tui/plan_view.go:313` — `width` is genuinely used once T005 lands
- [ ] T008 [US1] Run `go test ./internal/tui/` and confirm T002–T004 now pass and all nine pre-existing `renderPlanDetail` test functions still pass unmodified (contract C5.4)

**Checkpoint**: The description is on screen and the absence rules hold. This is a shippable MVP.

---

## Phase 4: User Story 2 — Description renders as formatted text, not raw markup (Priority: P1)

**Goal**: The description renders through the shared markdown renderer, producing output identical to the Tasks tab.

**Independent Test**: Give a task a description with a heading, a bullet list, bold text, and inline code; render it on both tabs at the same width and compare.

**Implementation note, stated plainly**: T005/T006 already call `md.Render`, so this story shares US1's production code rather than adding its own. What Phase 4 independently delivers is the *verified guarantee* — the tests that make contract C2.3's byte-identical claim real instead of assumed. If any of these tests fail, the fix lands here.

### Tests for User Story 2

- [ ] T009 [US2] Add `TestRenderPlanDetail_DescriptionMatchesTaskDetail` to `internal/tui/plan_view_test.go`: construct a real `markdown.NewRenderer(buildMarkdownTheme())`, render a description containing a heading, bullet list, bold text, inline code, and a link through both `renderPlanDetail` and `renderDetails` at the same width, and assert the description portion is identical (contract C2.3, SC-003)
- [ ] T010 [US2] Add `TestRenderPlanDetail_DescriptionUnstyledNoANSI` to `internal/tui/plan_view_test.go`: with a real renderer and `styled == false`, assert the output contains no ANSI escape sequences, extending the guarantee at `plan_view_test.go:423` to the new block (contract C2.5)
- [ ] T011 [US2] Add `TestRenderPlanDetail_DescriptionNilRendererFallback` to `internal/tui/plan_view_test.go`: with `md == nil`, assert the description is present and wrapped to width via `wrapDescription` (contract C2.4)
- [ ] T012 [US2] Add `TestRenderPlanDetail_DescriptionBlockLevel` to `internal/tui/plan_view_test.go`: assert a multi-line markdown list renders as multiple lines, proving `Render` was used rather than `RenderInline` — which the entry *name* still uses (contract C2.1)

### Implementation for User Story 2

- [ ] T013 [US2] Run `go test ./internal/tui/`. If T009–T012 reveal any divergence from `renderDetails`, correct the call in `internal/tui/plan_view.go` so `Options.Width` and `Options.Styled` match `details.go:25-30` exactly. Do NOT introduce a feature-local renderer, theme, or options struct (contract C2.2, Principle III)

**Checkpoint**: Descriptions render identically on the Planning and Tasks tabs. Both P1 stories are complete.

---

## Phase 5: User Story 3 — Description fits the available space without breaking the layout (Priority: P2)

**Goal**: A description of any length or width leaves the two-pane layout intact.

**Independent Test**: Select an entry whose task has a description far longer and wider than the pane, and confirm the grid, pane borders, and status line stay aligned.

**Why this needs real work**: `paneBox` (`internal/tui/view.go:706`) applies `lipgloss.Height`, which pads short content but does **not** truncate tall content — verified empirically in `research.md` R4. The left grid pane already trims itself (`plan_view.go:253-257`); the right pane never needed to, until now.

### Tests for User Story 3

- [ ] T014 [US3] Add `TestPlanView_LongDescriptionDoesNotExceedHeight` to `internal/tui/plan_view_test.go`: build a `Model` on the Planning tab in `planList` mode with `styled == true`, fixed `width`/`height`, and a selected entry whose task has a 200-line description; assert the rendered view's line count does not exceed `m.height` (FR-006, SC-004, contract C4.2)
- [ ] T015 [US3] Add `TestPlanView_LongDescriptionPreservesGridAlignment` to `internal/tui/plan_view_test.go`: with the same long description, assert every line of the joined view has the same visible width via `lipgloss.Width`, proving the grid pane and detail pane stay aligned
- [ ] T016 [US3] Add `TestRenderPlanDetail_DescriptionWrapsToWidth` to `internal/tui/plan_view_test.go`: at a narrow width, assert no rendered description line exceeds the pane's inner width, including a case with a long unbroken URL (contract C3.4, spec edge case)

### Implementation for User Story 3

- [ ] T017 [US3] Add a `clampLines(s string, n int) string` helper to `internal/tui/view.go` that truncates to at most `n` lines without padding — distinct from the existing `splitLines`, which pads to exactly `n` and returns a slice. Document the distinction in a comment so the two are not confused later
- [ ] T018 [US3] In the styled Planning path of `internal/tui/view.go` (near line 104–114), clamp `rightContent` to `innerH` with `clampLines` before passing it to `paneBox`. Apply the clamp to whichever content occupies the pane so forms and pickers are covered too (contract C4.1)
- [ ] T019 [US3] Confirm the plain-text Planning path at `internal/tui/view.go:142` still clamps via `splitLines(rightContent, maxLines)` and leave it unchanged (contract C4.3)
- [ ] T020 [US3] Run `go test ./internal/tui/` and confirm T014–T016 pass and the existing form, picker, and move-pane tests are unaffected (contract C5.1)

**Checkpoint**: All three user stories are complete and independently verified.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T021 Run `go test ./...` from the repo root and `cd services/twig && go test ./...`, confirming the server module is untouched as `plan.md` claims
- [ ] T022 [P] Run `gofmt -l internal/tui/` and confirm no files are listed
- [ ] T023 Build with `go build -o twig ./cmd/twig` and walk the full verification table in `quickstart.md` against a running stack (`make dev`): formatted description, Tasks-tab comparison, event entry, description-less task, cursor movement, 200-line description, ~60-column terminal, and non-TTY plain output
- [ ] T024 [P] Re-verify the Constitution Check in `plan.md`: no new abstraction beyond `clampLines` (I), contract committed before implementation (II), shared `m.md` renderer and theme with no ad-hoc styles (III), and **no new user-facing string added** — confirm no cheerful empty-state copy crept in, since that would violate contract C1.4 (IV)
- [ ] T025 [P] Confirm the pre-existing Tasks-tab height overflow at `internal/tui/view.go:245-248` was deliberately left unfixed, and that `research.md` R4 and `quickstart.md` still record it as out of scope. Do not fix it here

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies
- **Foundational (Phase 2)**: empty — does not block anything
- **US1 (Phase 3)**: starts after T001
- **US2 (Phase 4)**: verification depends on US1's implementation (T005–T006) being in place
- **US3 (Phase 5)**: independent of US1/US2 in principle — the clamp is a layout fix — but its long-description tests need the description to render, so run it after Phase 3
- **Polish (Phase 6)**: after all desired stories

### User Story Dependencies

- **US1 (P1)**: no dependencies. Fully shippable alone.
- **US2 (P1)**: shares production code with US1 (see the Phase 4 note). Its tests are the independent deliverable.
- **US3 (P2)**: layout-only. Could technically land before US1, but its regression tests are meaningless until a description can overflow the pane.

### Within Each Story

- Tests before implementation; confirm they fail first
- Unstyled path before styled path — the unstyled path is easier to assert on and catches logic errors before styling noise
- Story complete and green before moving to the next priority

### Parallel Opportunities

Deliberately few. `plan_view_test.go` carries almost every test task and `plan_view.go` carries almost every implementation task, so same-file tasks are sequential by necessity. The only true `[P]` tasks are T022, T024, and T025 in Polish, which touch nothing and read different artifacts.

The realistic concurrency here is between people, not tasks: one developer could take Phase 5 (`view.go`) while another takes Phases 3–4 (`plan_view.go`), with only the shared test file needing coordination.

---

## Implementation Strategy

### MVP First

1. T001 — baseline green
2. Phase 3 (T002–T008) — description on screen, absence rules holding
3. **STOP and VALIDATE**: schedule a described task, open the TUI, select it
4. Shippable: this alone satisfies the original request

### Incremental Delivery

1. Phase 3 → the feature works → demo
2. Phase 4 → fidelity with the Tasks tab is proven, not assumed → demo
3. Phase 5 → long descriptions can no longer break the layout → demo
4. Phase 6 → full suite, manual walkthrough, constitution re-check

---

## Notes

- `[P]` means different files and no incomplete dependency. Only T022, T024, T025 qualify.
- Every task names its file; `plan.md` line references are anchors from the current `main`, so re-locate by symbol if the file has shifted.
- Commit after each phase, not each task — the tasks within a phase are small and interdependent.
- Do not extract a shared description-rendering helper across `details.go` and `plan_view.go`. `research.md` R2 rejected it under Principle I; revisit only if a third caller appears.
- Do not add scrolling to the detail pane. Explicitly out of scope, and it would make the pane behave differently from its Tasks-tab sibling (Principle III).
