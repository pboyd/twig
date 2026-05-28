---

description: "Task list for TUI Theme Pass"
---

# Tasks: TUI Theme Pass

**Input**: Design documents from `/specs/018-tui-theme-pass/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ui-rendering-contract.md

**Tests**: INCLUDED. The spec's "Testing" guidance and the design doc explicitly
request tests (glyph mapping, border width, detail alignment, plain unstyled
path). Test tasks are written before the implementation they cover.

**Organization**: Tasks grouped by user story (US1=P1, US2=P2, US3=P3) for
independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- All paths are relative to repo root; package is `services/todo/internal/tui/`

## ⚠️ Shared-file note (read before parallelizing)

`view.go` is touched by US1 (borders/layout), US2 (row glyphs + cursor area), and
US3 (cursor + footer). Tasks editing `view.go` are **not** mutually `[P]` and must
be serialized in the order given. Cross-file work (`theme.go`, `tree.go`,
`details.go`, and their `_test.go` files) is genuinely parallelizable and marked
`[P]`.

---

## Phase 1: Setup

**Purpose**: Establish a green baseline before any change.

- [X] T001 Run `cd services/todo && go test ./internal/tui/...` and confirm the existing suite passes; record the current set of tests (esp. `TestRenderList_NoStrikethroughWhenUnstyled`, `TestRenderDetails_CompletedTaskNoStrikethrough`, and the `tree_test.go` `marker` assertions) as the baseline to preserve or migrate.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The centralized palette every story references. **No user story can begin until this is done.**

**⚠️ CRITICAL**: US1 borders, US2 glyph colors, and US3 header/cursor/footer all depend on the palette.

- [X] T002 Create `services/todo/internal/tui/theme.go` defining semantic styles as `lipgloss.AdaptiveColor` (light+dark): `accent`, `border`, `borderActive` (= accent), `dim`, `completed` (dim green), `errorColor`, `cursorBar`, `cursorBg`. Move `highlightStyle` and `errorStyle` out of `view.go` into `theme.go` and redefine them in terms of the palette (keep their exported behavior identical for now so T001 stays green).
- [X] T003 [P] Add `services/todo/internal/tui/theme_test.go`: assert each palette color is an `AdaptiveColor` with non-empty Light and Dark values, and that `errorStyle`/`highlightStyle` resolve from the palette.

**Checkpoint**: Palette available; stories can begin (respecting the `view.go` serialization note).

---

## Phase 3: User Story 1 - Clearly framed, focus-aware panels (Priority: P1) 🎯 MVP

**Goal**: Render the task list and the second pane (details/form/move) as rounded-bordered, titled boxes joined horizontally, with the focused pane's border in `accent` and the inactive pane's in `border`. Applies to all three list-mode layouts.

**Independent Test**: On a TTY, open the TUI: two titled rounded boxes appear; the active pane border is accent-colored, the other dim; switching to the move/edit views keeps the framed treatment. `lipgloss.Width` of each pane equals its allotted width.

### Tests for User Story 1 ⚠️ (write first, ensure they fail)

- [X] T004 [P] [US1] In `services/todo/internal/tui/view_test.go`, add a test that each rendered pane's visible width (`lipgloss.Width` per pane, before join) equals its allotted outer width for a representative size (e.g. 80×24), and that content does not overflow the frame (contract C4.1/C4.2).
- [X] T005 [P] [US1] In `services/todo/internal/tui/view_test.go`, add a test (styled path) asserting the focused pane border uses the `accent` color and the inactive pane uses `border` (contract C3.3); and a styled-vs-unstyled test that border box-drawing runes (`╭ ─ ╮ ╰ ╯ │`) appear only when `styled == true` (contract C1).

### Implementation for User Story 1

- [X] T006 [US1] In `services/todo/internal/tui/view.go`, add a pane-box helper that wraps content in `lipgloss.RoundedBorder()` with an embedded title and selects `borderActive` (accent) vs `border` by a `focused bool`, sizing via `.Width()`/`.Height()` using inner dims (outer − 2 borders − padding), clamping inner width ≥ 0 and inner height ≥ 1 (contract C3, C4.4).
- [X] T007 [US1] In `services/todo/internal/tui/view.go`, refactor `viewList` (styled path) to render the list and detail panes via the box helper and combine with `lipgloss.JoinHorizontal(lipgloss.Top, ...)`, routing inner width into `renderList`/`renderDetailPane`; remove the `splitLines`/`padRightAnsi` stitching for the styled path. Title the list pane `Tasks`. List pane is focused in `modeList`.
- [X] T008 [US1] In `services/todo/internal/tui/view.go`, apply the box helper + `JoinHorizontal` to `viewWithForm`; the form pane is the focused (accent) pane in edit/new modes (contract C3.4).
- [X] T009 [US1] In `services/todo/internal/tui/view.go`, apply the box helper + `JoinHorizontal` to `viewWithMove`; the move pane is the focused (accent) pane in `modeMove` (contract C3.4).
- [X] T010 [US1] Preserve the unstyled path: when `styled == false`, emit no borders/titles/box runes — keep today's plain joined output. Confirm `TestRenderList_NoStrikethroughWhenUnstyled` and the new C1 assertion (T005) pass.

**Checkpoint**: All three list-mode views are framed and focus-aware; widths are correct; unstyled output unchanged. MVP demoable.

---

## Phase 4: User Story 2 - Honest fold and completion indicators (Priority: P2)

**Goal**: Render rows as `{treePrefix}{chevron-or-space} {checkbox} {name}` — chevron `▾`/`▸` only on expandable rows, checkbox `☐`/`☑` on every row — in the styled path; unstyled path stays plain.

**Independent Test**: Build a tree with collapsible parents, leaves, and mixed completion; styled output shows a chevron only on expandable rows and a checkbox matching completion on every row; unstyled output contains none of those glyph runes.

### Tests for User Story 2 ⚠️ (write first, ensure they fail)

- [X] T011 [P] [US2] In `services/todo/internal/tui/tree_test.go`, migrate the existing `rows[0].marker` assertions (lines ~107–214) to the new fields: assert `expandable`/`expanded` instead of `[+]`/`[-]` (collapsed-with-visible-children ⇒ `expandable==true, expanded==false`; expanded ⇒ `expanded==true`; leaf / all-children-filtered ⇒ `expandable==false`).
- [X] T012 [P] [US2] In `services/todo/internal/tui/view_test.go`, add a styled-path test: a chevron rune (`▾`/`▸`) is present **iff** the row is expandable, and a checkbox rune (`☐`/`☑`) is present on every row matching `CompletedAt` (contract C2.1/C2.2).
- [X] T013 [P] [US2] In `services/todo/internal/tui/view_test.go`, add an unstyled-path test asserting NO decorative glyph runes (`▾ ▸ ☐ ☑`) appear when `styled == false` (contract C1/C2 unstyled).

### Implementation for User Story 2

- [X] T014 [US2] In `services/todo/internal/tui/tree.go`, change `visibleRow`: remove `marker string`; add `expandable bool` and `expanded bool`. In `emitNode`, set `expandable = hasExpandableChildren` and `expanded = isExpanded`; delete the `marker`/`[+]`/`[-]` string construction (computation of `hasExpandableChildren` is unchanged).
- [X] T015 [US2] In `services/todo/internal/tui/view.go` `renderList`, replace `prefix := row.treePrefix + row.marker + " "` with `{treePrefix}{chevron-or-space} {checkbox} ` where (styled path) chevron = `▾` if `expandable&&expanded`, `▸` if `expandable&&!expanded`, else `" "`; checkbox = `☑`/`☐` from `node.Task.GetCompletedAt()`. Keep completed-name strikethrough/dim-green (use `completed` palette color). Account for the new glyph columns in the inner-width truncation math.
- [X] T016 [US2] Preserve the unstyled path in `renderList`: when `styled == false`, keep today's plain output (no chevron/checkbox glyphs, name only, strikethrough suppressed) so SC-003 holds. Confirm T013 passes.

**Checkpoint**: Fold and completion are shown as independent, honest indicators; tree tests migrated; unstyled output still plain. US1 + US2 both work.

---

## Phase 5: User Story 3 - Polished details, footer, and consistent palette (Priority: P3)

**Goal**: Detail pane gets a bold-accent header, dimmed column-aligned labels, wrapped description, dim-green completed timestamp; the help line becomes a full-width footer; the cursor softens to a left accent bar + tint.

**Independent Test**: View a task with description/due/estimate/completed: name reads as a header, labels are dim and aligned, description wraps to inner width, footer reads as a full-width help bar, cursor row shows the `▎` bar + tint (not a solid fill); light-terminal legibility holds; unstyled detail output stays plain.

### Tests for User Story 3 ⚠️ (write first, ensure they fail)

- [X] T017 [P] [US3] In `services/todo/internal/tui/details_test.go`, add a styled-path test for the header (bold `accent`, no strike/dim on the name) and dim, column-aligned `ID/Due/Est/Completed` labels; keep `TestRenderDetails_CompletedTaskNoStrikethrough` green (unstyled path: no codes).
- [X] T018 [P] [US3] In `services/todo/internal/tui/view_test.go`, add a cursor test: the cursor row contains the accent bar `▎` and a background tint (`cursorBg`) rather than a full-width solid fill, and the completed-name strikethrough is still present on the cursor row (contract C6).
- [X] T019 [P] [US3] In `services/todo/internal/tui/view_test.go`, add a footer test: the status/help line renders full-width with the footer tint (styled path), and an error sets the `error` color via `renderStatus` (contract C7).

### Implementation for User Story 3

- [X] T020 [US3] Change `renderDetails` signature in `services/todo/internal/tui/details.go` to `renderDetails(task *taskv1.Task, width int, styled bool)`; update callers: `renderDetailPane` (`view.go`, pass `m.styled`), `ExportRenderDetails` (`export_test.go`, add `styled` param), and `details_test.go` call sites (pass `false` for the existing plain-path test).
- [X] T021 [US3] In `services/todo/internal/tui/details.go`, when `styled`: render the task name as a bold `accent` header on top, then `ID/Due/Est` as `dim` column-aligned labels, description wrapped to inner width, and a `Completed:` line in `completed` (dim green). When `!styled`: keep today's plain output exactly.
- [X] T022 [US3] In `services/todo/internal/tui/view.go` `renderList`, soften the cursor: replace the full-width `highlightStyle` fill with a left accent bar `▎` (`cursorBar`) + subtle `cursorBg` background tint + bold text on the cursor row, ensuring completed-name strikethrough on that row is preserved (contract C6). (Edits `view.go` — sequence after T015/T016.)
- [X] T023 [US3] In `services/todo/internal/tui/view.go` `renderStatus` (and `help.go`/`theme.go` as needed), style the help line as a full-width footer with a faint background tint via the bubbles `help` model styles; keep error messages using the `error` palette color (contract C7).

**Checkpoint**: Detail pane, footer, and cursor are polished; all three stories functional; palette consistent across the app.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T024 [P] Run `cd services/todo && go test ./...` and confirm the full suite is green (all migrated/new tests pass; no unstyled-path regressions).
- [X] T025 Run `cd services/todo && go build -o todo ./cmd/todo` and perform the `quickstart.md` manual checklist on a TTY (framed panes, focus accent, chevron-iff-expandable, checkbox per row, dim-green completed, `▎` cursor, full-width footer, small-terminal no-panic); verify `./todo | cat` produces plain output (no ANSI, no glyph runes).
- [X] T026 [P] Verify no leftover references to the removed `visibleRow.marker` field anywhere in `services/todo/internal/tui/` (`grep -rn "\.marker\|marker:" internal/tui/` returns nothing).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none — start immediately.
- **Foundational (Phase 2)**: depends on Setup — **blocks all user stories** (palette).
- **User Stories (Phase 3–5)**: each depends on Foundational. They share `view.go`, so prefer the priority order P1 → P2 → P3 (see shared-file note). Cross-file tasks within/across stories marked `[P]` may parallelize.
- **Polish (Phase 6)**: after the desired stories are complete.

### User Story Dependencies

- **US1 (P1)**: after Foundational. Independent; delivers the MVP framing.
- **US2 (P2)**: after Foundational. Logically independent, but its `renderList` edit (T015) shares `view.go` with US1's `viewList` refactor (T007) — sequence T007 before T015.
- **US3 (P3)**: after Foundational. `details.go` work (T020–T021) is file-independent of `view.go` and can run in parallel with US1/US2; the cursor (T022) and footer (T023) edits share `view.go` and must follow US2's `renderList` edits.

### Within Each User Story

- Write the story's tests first and see them fail, then implement.
- US1: box helper (T006) before the view refactors (T007–T009).
- US2: `tree.go` field change (T014) before `renderList` glyphs (T015).
- US3: signature change (T020) before detail styling (T021).

### Parallel Opportunities

- T003 (theme_test) ∥ nothing else needed but independent of later phases.
- US1 tests T004 ∥ T005 (same file `view_test.go` — different test funcs, safe to author together but commit carefully).
- US3 detail work (T020–T021, `details.go`) ∥ US1/US2 `view.go` work, since different files.
- Polish T024 ∥ T026.
- **Not parallel**: any two tasks both editing `view.go` (T007, T008, T009, T015, T016, T022, T023) — serialize in listed order.

---

## Parallel Example: cross-file work after Foundational

```bash
# These touch different files and can proceed concurrently:
Task: "T021 [US3] Style detail header + labels in internal/tui/details.go"
Task: "T014 [US2] Add expandable/expanded fields in internal/tui/tree.go"
# while the view.go refactor (T006–T010, then T015–T016, then T022–T023)
# proceeds on its own serial track.
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 Setup → 2. Phase 2 Foundational (palette) → 3. Phase 3 US1 (framed, focus-aware panes) → 4. **STOP & VALIDATE** on a TTY → demo. This alone delivers the biggest legibility win.

### Incremental Delivery

1. Setup + Foundational → palette ready.
2. US1 → framed panes (MVP) → validate/demo.
3. US2 → honest fold/completion glyphs → validate/demo.
4. US3 → detail/footer/cursor polish → validate/demo.
5. Polish → full suite green + manual quickstart.

---

## Notes

- This is a presentation-only feature: **no behavior, keybinding, proto, or DB change** (contract C9). Every change is gated by the existing `styled` flag.
- The single hardest invariant to keep green is the **plain unstyled path** (SC-003): re-run `TestRenderList_NoStrikethroughWhenUnstyled` and the new C1/C2/C5 unstyled assertions after each `view.go`/`details.go` edit.
- Commit after each task or logical group; stop at any checkpoint to validate a story independently.
