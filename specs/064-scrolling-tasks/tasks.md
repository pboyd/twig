---
description: "Task list for Scrolling Task List (064-scrolling-tasks)"
---

# Tasks: Scrolling Task List

**Input**: Design documents from `specs/064-scrolling-tasks/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/tui-interaction.md, quickstart.md

**Tests**: Included. The repo follows a test-driven style (`internal/tui/*_test.go` with `export_test.go` shims), and the spec/quickstart call out automated coverage. Test tasks precede the implementation they cover.

**Organization**: Tasks are grouped by user story. All work is confined to the root module's `internal/tui/` package (no server, proto, or DB changes).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files / independent, no dependency on incomplete tasks)
- **[Story]**: US1 or US2 (Setup/Foundational/Polish have no story label)
- Paths are repository-relative.

## Path Conventions

- Root CLI/TUI module: `internal/tui/`
- Tests live beside source in `internal/tui/*_test.go`; unexported access via `internal/tui/export_test.go`.

---

## Phase 1: Setup

- [x] T001 Confirm baseline builds and tests are green before changes: run `go build -o twig ./cmd/twig` and `go test ./internal/tui/...` from the repo root and note the current pass state.

---

## Phase 2: Foundational (blocking prerequisites for US1 and US2)

**Purpose**: The scroll offset state and the two pure/near-pure helpers both user stories depend on. No user-visible behavior changes yet.

- [x] T002 Add `listScroll int` field to the `Model` struct in `internal/tui/model.go` (place near `cursor`; add a short comment: "top visible row index of the Tasks list viewport"). Default zero value is correct.
- [x] T003 [P] Write unit tests for the pure `windowOffset(off, cursor, height, n int) int` helper in `internal/tui/view_test.go`: cursor above window pulls offset up; cursor below window pulls offset down (`cursor-height+1`); clamps to `[0, max(0, n-height)]`; `n <= height` returns 0; `n == 0` returns 0. Tests fail to compile until T004.
- [x] T004 Implement `windowOffset(off, cursor, height, n int) int` in `internal/tui/view.go` (alongside `splitLines`/`padRightAnsi`) per the rule in `data-model.md`. Make T003 pass.
- [x] T005 [P] Write unit tests for `func (m Model) listViewportHeight() int` in `internal/tui/view_test.go`: styled uses `m.height - 2 - statusHeight() - tabBarHeight`; plain uses `m.height - 1 - ...`; result is clamped to a minimum of 1; account for an active pomodoro raising `statusHeight()` to 2. Tests fail to compile until T006.
- [x] T006 Implement `func (m Model) listViewportHeight() int` in `internal/tui/view.go` mirroring the pane-height arithmetic already used in `viewList`/`viewWithForm`/etc., branching on `m.styled`. Make T005 pass.

**Checkpoint**: Helpers exist and are unit-tested; the app still behaves exactly as before (nothing calls them yet).

---

## Phase 3: User Story 1 — Keep the selected task visible while navigating a long list (Priority: P1) 🎯 MVP

**Goal**: The Tasks list windows to one screen and keeps the selection cursor visible as it moves with `↑`/`↓`/`home`/`end`, scrolling long lists and never showing blank space below the last task.

**Independent Test**: In a short terminal with more tasks than fit, navigate top→bottom→top with arrows; every task is reachable and the selected row is always visible; short lists look identical to before.

### Tests for User Story 1

- [x] T007 [P] [US1] In `internal/tui/view_test.go`, add render tests for `renderList`: for a `Model` with `len(visible) > listViewportHeight()`, assert the output has at most `listViewportHeight()` task rows AND always contains the cursor row, across representative cursor positions (top, middle, last) and a mid-list `listScroll`. Add an "exactly fits" case asserting output is unchanged from the full render (FR-005). Use `export_test.go` to construct a `Model` with populated `visible`/`cursor`/`height`/`styled` (extend the shim if needed).
- [x] T008 [P] [US1] In `internal/tui/update_test.go`, add tests that after `Down`/`Up`/`First`/`Last` key handling on a tall list, `listScroll` is in range and `cursor ∈ [listScroll, listScroll+height)`; and that on `tea.WindowSizeMsg` shrinking the height, `listScroll` re-clamps so the cursor stays visible.

### Implementation for User Story 1

- [x] T009 [US1] Window the render in `renderList` (`internal/tui/view.go`): compute `h := m.listViewportHeight()` and `off := windowOffset(m.listScroll, m.cursor, h, len(m.visible))`, then iterate only absolute indices `[off, min(off+h, len(m.visible)))`, preserving the existing `i == m.cursor` cursor styling and per-row markdown/tree-prefix rendering. Keep the existing empty-list early return. Make T007 pass.
- [x] T010 [US1] Persist the reconciled offset after Tasks-tab key handling: in `handleKey` (`internal/tui/update.go`), wrap the `modeList` dispatch so that after `handleListKey` returns a `Model`, set `m.listScroll = windowOffset(m.listScroll, m.cursor, m.listViewportHeight(), len(m.visible))` before returning. (Covers `Up`/`Down`/`First`/`Last`/`Expand`/`Collapse` including their early returns.)
- [x] T011 [US1] Re-clamp on resize: in the `tea.WindowSizeMsg` case in `internal/tui/update.go` (after `m.height`/`m.width` are set), recompute `m.listScroll` via `windowOffset(...)`.
- [x] T012 [US1] Re-clamp after async list rebuilds: in the message handlers that rebuild `m.visible` (task-load/refresh, complete/uncomplete, toggle-all, filter-apply — around the `buildVisible(...)` calls in `internal/tui/update.go`), recompute `m.listScroll` via `windowOffset(...)` so a shrinking list never strands the viewport (FR-006). Make T008 pass.

**Checkpoint**: Arrow/home/end navigation scrolls correctly; long lists fully reachable; short lists unchanged. US1 is independently shippable (MVP).

---

## Phase 4: User Story 2 — Page through the list with PgUp / PgDn (Priority: P2)

**Goal**: `PgUp`/`PgDn` move the selection by ~one screen, bounded at the first/last task, consistent with the goal-status reader.

**Independent Test**: On a list several screens tall, `PgDn` advances ~one screen per press and stops at the last task; `PgUp` moves back symmetrically and stops at the first task.

### Tests for User Story 2

- [x] T013 [P] [US2] In `internal/tui/update_test.go`, add tests: `PageDown` moves `cursor` toward the end by `listViewportHeight()` (clamped to `len(visible)-1`) and `PageUp` moves toward the start by the same (clamped to 0); pressing `PgUp` at the top and `PgDn` at the bottom are no-ops; after each, `listScroll` keeps the cursor visible.

### Implementation for User Story 2

- [x] T014 [US2] Add `PageUp` (`key.WithKeys("pgup")`, help `"pgup"/"page up"`) and `PageDown` (`key.WithKeys("pgdown")`, help `"pgdown"/"page down"`) bindings to `KeyMap` in `internal/tui/keymap.go`.
- [x] T015 [US2] Add `PageUp`/`PageDown` cases to `handleListKey` in `internal/tui/update.go`: move `m.cursor` by `±m.listViewportHeight()` and clamp with `clampCursor(_, len(m.visible))`. Let the T010 wrapper reconcile `listScroll`. Make T013 pass.
- [x] T016 [US2] Surface the new keys in Tasks-tab help in `internal/tui/keymap.go`: add `PageUp`/`PageDown` to the non-Goal/non-Report/non-Planning branch of `ShortHelp` and/or `FullHelp` so they are discoverable (Constitution III).

**Checkpoint**: Paging works and is discoverable; both user stories complete.

---

## Phase 5: Polish & Cross-Cutting Concerns

- [x] T017 [P] Update the help-content/keymap tests if the project asserts on `ShortHelp`/`FullHelp` contents (e.g. `internal/tui/*_test.go`) to include the new `PageUp`/`PageDown` bindings; otherwise skip.
- [ ] T018 Run the full manual verification in `specs/064-scrolling-tasks/quickstart.md` against a `go build -o twig ./cmd/twig` binary in a short terminal (scroll down/up, home/end, PgUp/PgDn, short-list no-op, resize, list-shrink).
- [x] T019 Run `go test ./...` from the repo root and confirm all packages pass; then `go vet ./...` on the touched package.

---

## Dependencies & Execution Order

- **Setup (T001)** → **Foundational (T002–T006)** → **US1 (T007–T012)** → **US2 (T013–T016)** → **Polish (T017–T019)**.
- **Foundational blocks everything**: `windowOffset`, `listViewportHeight`, and `listScroll` are used by both stories.
- **US1 before US2**: US2's paging relies on the viewport windowing and the T010 offset-reconciliation wrapper landed in US1.
- **US1 is independently deliverable** as the MVP (scrolling via arrows/home/end) without US2.

### Within-phase parallelism

- Foundational: T003 and T005 (test authoring for the two helpers) are `[P]` — different concerns in the same test file; if edited by one agent, do them sequentially to avoid file conflicts. T004 depends on T003; T006 depends on T005.
- US1: T007 and T008 are `[P]` (different test files: `view_test.go` vs `update_test.go`). Implementation T009–T012 touch `view.go`/`update.go`; T010/T011/T012 all edit `update.go` so run them sequentially.
- US2: T013 (test) `[P]` with US1 tests conceptually, but sequence after US1 implementation exists.

## Parallel Execution Example

```text
# After Foundational is complete, US1 test authoring can proceed together:
T007 [P] [US1] renderList windowing tests   (internal/tui/view_test.go)
T008 [P] [US1] nav/resize scroll tests       (internal/tui/update_test.go)
# Then implement US1 sequentially (shared files): T009 → T010 → T011 → T012
```

## Implementation Strategy

- **MVP = Phases 1–3** (Setup + Foundational + US1): delivers a scrolling Tasks list that keeps the cursor visible — the core defect fixed and shippable on its own.
- **Increment 2 = Phase 4** (US2): adds `PgUp`/`PgDn` paging.
- **Finish = Phase 5**: help-test updates and full manual + automated verification.

## Notes

- Entire change is TUI-only in `internal/tui/`; no `make proto`, no `sqlc generate`, no server/web changes.
- No new user-facing prose is required; existing empty-state copy is preserved (Constitution IV).
- Keep `windowOffset` a pure function so it is trivially unit-testable and reusable by both the renderer and the Update-path persistence.
