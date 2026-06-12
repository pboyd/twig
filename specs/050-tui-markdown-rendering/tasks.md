---
description: "Task list for Markdown Rendering in TUI Text Fields"
---

# Tasks: Markdown Rendering in TUI Text Fields

**Input**: Design documents from `/specs/050-tui-markdown-rendering/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/markdown-package.md, quickstart.md

**Tests**: Included. This repo is test-driven (every package ships `_test.go`), and the package contract specifies styled/plain golden assertions. Test tasks are written before the implementation they cover.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1 / US2 / US3 (Setup, Foundational, Polish carry no story label)
- All paths are repo-root-relative (root Go module `github.com/pboyd/twig`)

## Path Conventions

Single project (CLI/TUI client). New renderer package: `internal/markdown/`. Integration sites: `internal/tui/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the dependency and stand up the package skeleton everything else builds on.

- [ ] T001 Add `github.com/yuin/goldmark` v1.7.13 as a direct dependency (`go get github.com/yuin/goldmark@v1.7.13` then `go mod tidy`) and confirm it lands in `go.mod`/`go.sum`
- [ ] T002 Create the `internal/markdown` package skeleton in `internal/markdown/markdown.go`: define `Theme`, `Options`, `Renderer` (unexported `md`, `theme`, `cache`), `cacheKey{text,width,styled,inline}`, `NewRenderer(theme Theme) *Renderer` (build `goldmark.New(goldmark.WithExtensions(extension.GFM))`), and `Render`/`RenderInline` method stubs matching `contracts/markdown-package.md`

**Checkpoint**: package compiles with stub methods; goldmark wired.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared rendering primitives used by BOTH long-field (US1) and short-field (US2) rendering, plus the TUI bridge. No user story can be completed until this phase is done.

**⚠️ CRITICAL**: complete this phase before starting Phase 3 or 4.

- [ ] T003 [P] Implement display-width-aware soft-wrap helper (using `rivo/uniseg`/`mattn/go-runewidth`, NOT byte length) in `internal/markdown/wrap.go`
- [ ] T004 [P] Implement link rendering helper in `internal/markdown/link.go`: styled → wrap text in `ansi.SetHyperlink(url)` with underline+accent; plain → `text (url)` (per FR-007 amended + `contracts/markdown-package.md`)
- [ ] T005 Implement the inline node renderer in `internal/markdown/inline.go`: walk strong/emphasis/strikethrough/code-span/link/text nodes → `lipgloss/v2` styles drawn only from `Theme`; plain mode emits ZERO escape sequences; strips markup chars (depends on T004)
- [ ] T006 Implement memoized dispatch in `internal/markdown/markdown.go`: `Render`/`RenderInline` consult the bounded `cache` keyed by `cacheKey`, render on miss, store, and return; correctness must not depend on the cache (depends on T002)
- [ ] T007 [P] Add `buildMarkdownTheme() markdown.Theme` mapping palette vars (`accent`, `dim`, code-bg derived from `cursorBg`) → `markdown.Theme` in `internal/tui/theme.go` (no new ad-hoc colors — Principle III)
- [ ] T008 Add a `*markdown.Renderer` field to `Model` and construct it via `markdown.NewRenderer(buildMarkdownTheme())` in `NewModel` in `internal/tui/model.go` (depends on T002, T007)
- [ ] T009 [P] Unit tests for the inline renderer and link helper in `internal/markdown/inline_test.go` and `internal/markdown/link_test.go`: styled output carries styling; plain output contains no `\x1b`; no leftover `*`/`` ` ``/`~`; link styled→OSC 8 present, plain→`text (url)` (covers T004, T005)

**Checkpoint**: shared primitives + TUI renderer wiring ready; both user stories can proceed.

---

## Phase 3: User Story 1 - Read formatted long-text fields (Priority: P1) 🎯 MVP

**Goal**: Long-text fields (task description, goal description) render full GFM — headings, lists, tables, blockquotes, code, rules, links — in display mode.

**Independent Test**: Open a task whose description contains a heading, nested list, table, blockquote, fenced code block, and a link; confirm each renders recognizably within the pane width, with no leftover markup for supported inline styles.

### Tests for User Story 1 ⚠️ (write first, ensure they fail)

- [ ] T010 [P] [US1] Block render tests in `internal/markdown/block_test.go`: heading (styled distinct), unordered/ordered/nested/task lists (Unicode markers), blockquote bar, code block verbatim, horizontal rule, image→alt placeholder, raw HTML pass-through; width wrapping; malformed input does not panic; plain mode zero-escape
- [ ] T011 [P] [US1] Table render tests in `internal/markdown/table_test.go`: column alignment, width reflow/shrink to fit, plain-mode ASCII grid

### Implementation for User Story 1

- [ ] T012 [US1] Implement the block AST walker in `internal/markdown/block.go`: headings (bold accent, level-prefixed), paragraphs, unordered/ordered/task lists with nesting, blockquote left bar, code block (verbatim, not reflowed), horizontal rule, image alt placeholder (warm copy), raw HTML pass-through — calling the inline renderer (T005) and wrap helper (T003)
- [ ] T013 [US1] Implement GFM table rendering via `lipgloss/v2` `table` (width-aware) in `internal/markdown/table.go` and dispatch table nodes to it from the block walker (depends on T005, T012)
- [ ] T014 [US1] Implement public `Render()` in `internal/markdown/markdown.go`: parse GFM, walk blocks, join with no trailing newline, behind the memoized dispatch (depends on T012, T013, T006)
- [ ] T015 [US1] Render the task description via `m.md.Render(desc, markdown.Options{Width: width, Styled: m.styled})` in `renderDetails` (both styled and plain paths), replacing `wrapDescription` for the description in `internal/tui/details.go` (depends on T014, T008)
- [ ] T016 [US1] Render the goal description via `m.md.Render(...)` in `renderGoalDetail`, replacing `wrapDescription` for the description in `internal/tui/goal_view.go` (depends on T014, T008)
- [ ] T017 [P] [US1] Integration tests asserting rendered descriptions (styled carries styling, plain is escape-free, content stays within width) in `internal/tui/details_test.go` and `internal/tui/goal_view_test.go`

**Checkpoint**: US1 independently functional — long fields render full markdown. This is the MVP.

---

## Phase 4: User Story 2 - Inline formatting in short fields (Priority: P2)

**Goal**: Short single-line fields (task name, goal name) render inline emphasis (bold, italic, strikethrough, inline code, links) consistently everywhere they appear.

**Independent Test**: Set a task name to `**Ship** the *report*`; confirm it renders bold+italic on one line in the tree row and the detail header, with no `*` and no line break.

### Tests for User Story 2 ⚠️ (write first, ensure they fail)

- [ ] T018 [P] [US2] `RenderInline` tests in `internal/markdown/markdown_test.go`: single-line output (no `\n`); block syntax (leading `# `, `- `) flattened inline without breaking layout; markup chars stripped; plain mode zero-escape

### Implementation for User Story 2

- [ ] T019 [US2] Implement public `RenderInline()` in `internal/markdown/markdown.go`: parse, walk inline descendants only (block containers contribute flattened inline children), emit a single line with no newlines, behind the memoized dispatch (depends on T005, T006)
- [ ] T020 [US2] Render task names via `m.md.RenderInline(name, markdown.Options{Width: width, Styled: m.styled})` at the tree-row site in `internal/tui/view.go` (preserve existing cursor/highlight + `padRightAnsi` behavior) (depends on T019, T008)
- [ ] T021 [US2] Render goal names via `RenderInline` in `renderGoalList`, the `renderGoalDetail` name header, `renderGoalTaskTree`, and `renderGoalPicker` in `internal/tui/goal_view.go` (depends on T019, T008)
- [ ] T022 [US2] Render plan-entry / task-name displays via `RenderInline` in `internal/tui/plan_view.go` (depends on T019, T008)
- [ ] T023 [P] [US2] Integration tests for inline-rendered names in `internal/tui/view_test.go`, `internal/tui/goal_view_test.go`, and `internal/tui/plan_view_test.go` (consistent rendering across all display contexts — FR-009)

**Checkpoint**: US2 independently functional — names carry inline emphasis everywhere.

---

## Phase 5: User Story 3 - Edit raw source without content loss (Priority: P3)

**Goal**: Editing a field shows raw markdown; rendering never mutates stored text.

**Independent Test**: Edit a field containing markdown, confirm the editor shows raw markdown characters, save without changes, and confirm stored content is byte-for-byte identical.

### Tests for User Story 3 ⚠️ (write first, ensure they fail)

- [ ] T024 [P] [US3] Round-trip test in `internal/tui/edit_test.go`: entering edit mode shows raw markdown source (not rendered); saving an unchanged field leaves the stored text byte-for-byte identical (SC-003)

### Implementation for User Story 3

- [ ] T025 [US3] Verify and, if needed, adjust the edit/textarea paths in `internal/tui/edit.go` so editing always presents raw markdown source and no rendering is applied on the edit path (display-only invariant, FR-010) (depends on T024)

**Checkpoint**: US3 verified — round-trip is safe; rendering is strictly display-only.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Edge cases, degradation, tone, cleanup, and final verification.

- [ ] T026 [P] Verify the Report tab renders generated text as-is (no unintended markdown interpretation of computed output) in `internal/tui/report_view.go`, adding a guard/test if needed
- [ ] T027 [P] Add edge-case tests in `internal/markdown/markdown_test.go`: non-ASCII/emoji wrap on display width; empty/whitespace-only field renders empty without error; literal `*`/`_`/`#` not forming markup are preserved
- [ ] T028 [P] Review the image-alt / unrenderable-content placeholder copy for warm tone (Principle IV) in `internal/markdown/block.go`
- [ ] T029 Remove or scope down `wrapDescription` in `internal/tui/details.go` if fully superseded by the renderer; run `gofmt` and `go vet ./...`
- [ ] T030 Run the full suite (`go test ./...`) and perform the manual TUI verification steps from `quickstart.md` (build, view a markdown-rich task, narrow-resize, `NO_COLOR=1` plain check)

---

## Dependencies & Execution Order

- **Setup (Phase 1)** → blocks everything.
- **Foundational (Phase 2)** → blocks US1, US2, US3. T005 depends on T004; T006 on T002; T008 on T002+T007.
- **US1 (Phase 3)** and **US2 (Phase 4)** both depend only on Foundational and can largely proceed in parallel — EXCEPT both edit `internal/tui/goal_view.go` (T016 description vs. T021 name), so coordinate those two (not `[P]` against each other).
- **US3 (Phase 5)** depends on Foundational; independent of US1/US2 (verification-focused).
- **Polish (Phase 6)** runs after the stories it touches; T030 runs last.

### Story completion order (by priority)

US1 (P1, MVP) → US2 (P2) → US3 (P3). Each is independently testable at its checkpoint.

## Parallel Execution Examples

- **Phase 2 kickoff**: T003, T004, T007 in parallel (different files); then T005 (after T004), T006, T008; T009 after T004/T005.
- **Phase 3 tests**: T010 and T011 in parallel before T012–T014.
- **US1 + US2 overlap**: after Phase 2, one track does T010–T015 (US1) while another does T018–T020/T022 (US2); serialize only the shared `goal_view.go` edits (T016, T021).
- **Polish**: T026, T027, T028 in parallel; then T029; then T030.

## Implementation Strategy

**MVP = Phase 1 + Phase 2 + Phase 3 (US1).** That alone delivers the highest-value slice: long descriptions render full markdown. Ship/verify, then layer US2 (inline names) and US3 (round-trip guarantee) as incremental, independently testable additions.

## Format validation

All 30 tasks use `- [ ]`, sequential IDs T001–T030, `[P]` only where files are independent, `[US#]` labels on story-phase tasks only (Setup/Foundational/Polish unlabeled), and every task names concrete file path(s).
