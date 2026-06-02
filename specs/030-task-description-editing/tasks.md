---
description: "Task list for Multiline Task Descriptions & External Editor"
---

# Tasks: Multiline Task Descriptions & External Editor

**Input**: Design documents from `specs/030-task-description-editing/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/tui-description-editing.md, quickstart.md

**Tests**: Included. This codebase is test-first by convention (table-driven unit tests in `internal/tui/` using in-package `_test.go` files), and the plan/quickstart define an explicit requirement→test coverage map. Tests are written before implementation per story.

**Organization**: Tasks are grouped by user story. US1 (display) and US2 (editor) touch mostly disjoint files; US1 is sequenced first as the MVP, and because both stories' tests live in `package tui`, completing US1 before adding US2's symbols keeps the package compiling between checkpoints.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2
- All paths are under `services/twig/` (the Go module). Run `go` commands from `services/twig/`.

## Path Conventions

All source for this feature lives in `services/twig/internal/tui/`. Tests are in-package files (`package tui`) in the same directory, so they call unexported helpers (`wrapDescription`, `renderDetails`, `resolveEditor`, …) directly — no `export_test.go` additions are needed.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish a known-green baseline before changing rendering or input handling.

- [X] T001 Confirm baseline is green: from `services/twig/` run `go build ./...` and `go test ./internal/tui/...`; record that the suite passes before changes.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared prerequisites that must exist before user-story work.

**⚠️ CRITICAL**: None. This feature has no blocking shared infrastructure — US1 and US2 are self-contained and independently testable. US1 edits `details.go`; US2 adds `editor.go` and edits `keymap.go`/`edit.go`/`update.go`. Proceed directly to Phase 3.

**Checkpoint**: Foundation trivially ready — user story implementation can begin.

---

## Phase 3: User Story 1 - Whitespace and newlines preserved on display (Priority: P1) 🎯 MVP

**Goal**: Task descriptions display with the user's newlines, blank lines, and per-line leading indentation intact (long lines still wrap), instead of being flattened by `strings.Fields()`. Fixes existing tasks retroactively.

**Independent Test**: Render (or view in the TUI) a task whose description has multiple lines, a blank-line paragraph break, and an indented line; confirm all line breaks and the indentation appear in the details pane and that a too-long line wraps.

### Tests for User Story 1 ⚠️ (write first, ensure they FAIL)

- [X] T002 [US1] Add failing table-driven tests in `internal/tui/details_test.go` for the new `wrapDescription(text, width)` and for `renderDetails` description output, covering: (a) `"Line one\nLine two"` → two lines; (b) blank line between paragraphs preserved; (c) leading indentation preserved on a line; (d) a line longer than `width` wraps while other line breaks remain; (e) mid-line multi-space run may collapse (`"a     b"` → `"a b"`); (f) `width <= 0` returns text unchanged. Assert exact line structure per `contracts/tui-description-editing.md` §C1.

### Implementation for User Story 1

- [X] T003 [US1] In `internal/tui/details.go`, replace `wordWrap` with `wrapDescription(text string, width int) string` implementing the line-aware algorithm from `research.md` R2 (split on `\n`; preserve empty/blank lines; preserve each line's leading indentation; wrap each line's words to `width`; `width<=0` passthrough). Update both `renderDetails` call sites (styled + unstyled) to call `wrapDescription`. Grep the package for any other `wordWrap` references and update them.
- [X] T004 [US1] From `services/twig/` run `go test ./internal/tui/...` and confirm T002 tests pass; do a quick manual check per `quickstart.md` (multiline + blank line + indented description renders correctly).

**Checkpoint**: User Story 1 is fully functional — multiline descriptions display correctly, including pre-existing data. MVP shippable.

---

## Phase 4: User Story 2 - Edit a description in the external editor (Priority: P2)

**Goal**: While the Description field is focused in the edit/create form, `ctrl+g` shells out to `$EDITOR` (args supported; `vim` fallback) pre-loaded with the current text; on save the edited text returns into the field, with one trailing newline normalized away. Failures preserve the form.

**Independent Test**: Edit a task, focus Description, press `ctrl+g`; confirm the configured editor opens with the current text, edit/save/quit, and the changed text appears in the field. Verify `EDITOR="code --wait"` passes args, unset `EDITOR` uses `vim`, and a bogus editor shows a friendly error without losing the description.

### Tests for User Story 2 ⚠️ (write first, ensure they FAIL)

> Note (Go): these in-package tests reference symbols added in T008–T009; until those exist the `tui` package won't compile (a red state). Add the tests, then implement, within this phase. Do not start this phase until US1 (Phase 3) is green.

- [X] T005 [P] [US2] Add `internal/tui/editor_test.go` with table tests for `resolveEditor(editorEnv, path)` (cases: `""`→`vim [path]`; `"vim"`→`vim [path]`; `"code --wait"`→`code [--wait path]`; `"  emacsclient  -nw "`→`emacsclient [-nw path]`; path always last) and for the trailing-newline normalization (`"hello\n"`→`"hello"`, `"hello"`→`"hello"`, `"a\n\nb\n"`→`"a\n\nb"`, `""`→`""`) per `contracts/tui-description-editing.md` §C3/§C4.
- [X] T006 [P] [US2] Add tests in `internal/tui/edit_test.go`: pressing `ctrl+g` while `focusIndex == focusDescription` returns a non-nil `tea.Cmd` from `editFormModel.Update`; pressing `ctrl+g` while another field (Name/Due/Estimate) is focused does NOT trigger the editor (key forwarded/ignored).
- [X] T007 [P] [US2] Add tests in `internal/tui/update_test.go`: an `editorFinishedMsg{content}` (no error) while in `modeEdit` updates `m.edit.description` value and clears `m.err`; an `editorFinishedMsg{err}` leaves the description unchanged and sets a non-nil `m.err`.

### Implementation for User Story 2

- [X] T008 [US2] In `internal/tui/keymap.go`, add `Editor key.Binding` to `KeyMap`, bound to `ctrl+g` with help `"ctrl+g", "edit in $EDITOR"`, initialized in `DefaultKeyMap()`.
- [X] T009 [US2] Create `internal/tui/editor.go` with: `editorFinishedMsg{content string; err error}`; `resolveEditor(editorEnv, path string) (name string, args []string)` (R4); a trailing-newline normalizer (strip exactly one `\n`); and `openEditorCmd(text string) tea.Cmd` that writes `text` to an `os.CreateTemp("", "twig-desc-*.md")`, builds the `exec.Cmd` via `resolveEditor(os.Getenv("EDITOR"), path)`, and returns `tea.ExecProcess(cmd, cb)` where `cb` reads+normalizes the file on success / passes the error through, and removes the temp file in both paths (per `research.md` R3/R5).
- [X] T010 [US2] In `internal/tui/edit.go`, in `editFormModel.Update`, before field-forwarding, handle `key.Matches(keyMsg, keys.Editor)` and `f.focusIndex == focusDescription` by returning `f, openEditorCmd(f.description.Value())`. Update `View` so the Description hint line advertises `ctrl+g: open editor` when Description is focused.
- [X] T011 [US2] In `internal/tui/update.go` `Model.Update`, add `case editorFinishedMsg`: when `m.mode` is one of `modeEdit`/`modeNewSubtask`/`modeNewRoot` — on `err == nil` call `m.edit.description.SetValue(msg.content)` and clear `m.err`; on error set a playful `m.err` (warm tone, matching existing notices) and leave the field unchanged; otherwise ignore. Return `m, nil`.
- [X] T012 [US2] From `services/twig/` run `go test ./internal/tui/...` (confirm T005–T007 pass), then `go build -o twig ./cmd/twig` and do the manual editor round-trip from `quickstart.md` (success, no-op idempotency, `code --wait` args, bogus-editor error).

**Checkpoint**: Both user stories work independently — multiline display (US1) and external-editor authoring (US2).

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and consistency.

- [X] T013 [P] Review all new user-facing strings (the `ctrl+g` hint and the editor-failure error) for the warm/playful tone used elsewhere in the TUI (e.g. plan notices in `update.go`); adjust wording if flat.
- [X] T014 Run the full suite and build from `services/twig/`: `go test ./...` and `go build ./...`; complete the `quickstart.md` manual verification checklist end-to-end (including the existing-data regression in US1).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: No tasks; non-blocking.
- **User Story 1 (Phase 3)**: After Setup. Independent of US2.
- **User Story 2 (Phase 4)**: After Setup. Sequenced after US1 (Phase 3 green) only to keep the shared `package tui` compiling between checkpoints — there is no functional dependency on US1.
- **Polish (Phase 5)**: After US1 and US2 are complete.

### User Story Dependencies

- **US1 (P1)**: No dependencies. Self-contained in `details.go` + `details_test.go`.
- **US2 (P2)**: No functional dependency on US1; only the compile-ordering note above.

### Within Each User Story

- Tests written first and failing → implementation → run green + manual check.
- US1: test (T002) → impl (T003) → verify (T004).
- US2: tests (T005–T007) → keymap (T008) → editor core + message (T009) → form wiring (T010) → model routing (T011) → verify (T012). T009 must precede T010/T011 (they use `openEditorCmd`/`editorFinishedMsg`); T008 must precede T010 (uses `keys.Editor`).

### Parallel Opportunities

- US2 test authoring T005, T006, T007 are in three different files → can be written in parallel.
- T013 (tone review) is independent of T014 ordering only in that the build/test run is authoritative.
- US1 and US2 could be developed by two people in parallel on separate branches; if sharing one working tree, follow the sequential order to avoid `package tui` compile breakage.

---

## Parallel Example: User Story 2 tests

```bash
# From services/twig/ — author these three test files together (different files):
Task: "T005 resolveEditor + normalize tests in internal/tui/editor_test.go"
Task: "T006 ctrl+g focus-gating tests in internal/tui/edit_test.go"
Task: "T007 editorFinishedMsg routing tests in internal/tui/update_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup (T001).
2. Phase 2: Foundational (none).
3. Phase 3: User Story 1 (T002–T004).
4. **STOP and VALIDATE**: multiline descriptions render correctly, including existing tasks.
5. Ship — this alone resolves the primary complaint.

### Incremental Delivery

1. Setup → baseline green.
2. US1 → test independently → ship (MVP: whitespace preserved).
3. US2 → test independently → ship (external-editor authoring).
4. Polish → full suite + quickstart sign-off.

---

## Notes

- [P] = different files, no incomplete-task dependencies.
- Tests are in-package (`package tui`) and call unexported helpers directly — no `export_test.go` changes.
- No proto/handler/DB/web changes; descriptions already store newlines (the fix is render-time + an input convenience).
- Commit after each task or logical group; stop at any checkpoint to validate a story independently.
