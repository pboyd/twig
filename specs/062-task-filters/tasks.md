---
description: "Task list for Task Filters feature implementation"
---

# Tasks: Task Filters

**Input**: Design documents from `/specs/062-task-filters/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/filter-tasks.md, contracts/filter-grammar.md, quickstart.md

**Tests**: INCLUDED. The spec mandates them — SC-003 (every documented expression form returns the hand-computed set), SC-004 (no keystroke sequence crashes/blanks the list), SC-005 (identical results via the backend directly) — and the plan calls for table-driven `internal/filter` tests, `export_test.go`-shim handler tests, and `internal/tui/*_test.go` behavior tests.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 (text search), US2 (attribute filters), US3 (transitive matching)
- File paths are repo-relative

## Path Conventions

Three Go modules (per plan.md):
- `api/` — shared proto + generated stubs
- `services/twig/` — server (handler + new `internal/filter` package)
- repo root — CLI/TUI (`internal/tui`)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Land the API contract both the server and TUI compile against.

- [ ] T001 Add the `FilterTasks` RPC plus `FilterTasksRequest` (`expression`, `show_all`, `today`) and `FilterTasksResponse` (`repeated int64 task_ids`) to `api/proto/task/v1/task.proto`, exactly per `specs/062-task-filters/contracts/filter-tasks.md` (leave all existing RPCs untouched).
- [ ] T002 Regenerate stubs with `make proto` and confirm `api/gen/` now exposes `FilterTasks` on `task.v1.TaskService` (do not hand-edit generated code).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the `internal/filter` package surface and wire the handler so both modules compile before any story logic exists.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T003 [P] Create the `services/twig/internal/filter` package with the AST + token type definitions from `data-model.md` in `services/twig/internal/filter/ast.go`: `Expression`, the `Condition` interface, and the `TextCondition`, `BoolCondition`, `DateCondition`, `RelCondition` structs with their field/operator enums; add package-level `Parse(expr string) (Expression, error)` and `Evaluate(e Expression, tasks []db.Task, showAll bool, today civil.Date) ([]int64, error)` signatures as stubs in `services/twig/internal/filter/filter.go`.
- [ ] T004 Implement the `FilterTasks` handler in `services/twig/internal/handler/task.go`: load the caller's tasks via the existing `Queries.ListTasks`, parse `today`, call `filter.Parse` then `filter.Evaluate`, return `FilterTasksResponse` with ascending `task_ids`, and map any `filter.Parse`/validation error to `connect.NewError(connect.CodeInvalidArgument, …)` with a human-readable message (never any other code for expression problems). Depends on T002, T003.

**Checkpoint**: Both modules build; `FilterTasks` returns (empty) results through the real request path.

---

## Phase 3: User Story 1 - Find a task by typing text (Priority: P1) 🎯 MVP

**Goal**: Press `/` in the Tasks tab, type text, and watch the list narrow per keystroke to matching tasks plus their ancestor chains; `Enter` applies, `Esc` clears.

**Independent Test**: Create a nested task set, press `/`, type a term that matches only a deeply nested subtask, and confirm that subtask and all its ancestors are the only rows shown; verify show-all off hides a completed "foo" match and show-all on reveals it.

### Tests for User Story 1 ⚠️ (write first, ensure they FAIL)

- [ ] T005 [P] [US1] Table-driven lexer + text-term/`AND`/quoting parser tests (bare word, quoted `"AND review"`, multi-word bareword, empty expression → error) in `services/twig/internal/filter/parser_test.go`.
- [ ] T006 [P] [US1] Evaluator tests for text matching (case-insensitive substring across name + description) and FR-008 default visibility with `show_all` on/off (worked-examples rows `foo`/off and `foo`/on), asserting ascending id output, in `services/twig/internal/filter/eval_test.go`.
- [ ] T007 [P] [US1] `FilterTasks` handler test (valid text expression returns sorted matched ids; malformed expression → `CodeInvalidArgument`; empty result on no match) using the `export_test.go` shim convention in `services/twig/internal/handler/task_test.go`.
- [ ] T008 [P] [US1] TUI behavior test: `/` opens the filter bar, per-keystroke input yields a `FilterTasks` command, matched ids render with ancestor chains (non-matching descendants hidden), `Esc` restores the full list, `Enter` returns focus to the list with the filter still applied, in `internal/tui/update_test.go`.

### Implementation for User Story 1

- [ ] T009 [US1] Implement the full tokenizer (identifiers, quoted strings, `AND` keyword, `^`, `=` `!=` `<` `<=` `>` `>=`, `YYYY-MM-DD` dates, integers, `true`/`false`, whitespace) in `services/twig/internal/filter/lexer.go`. Depends on T003.
- [ ] T010 [US1] Implement the recursive-descent parser for a text term and `AND`-joined conditions producing `TextCondition`s, with quoting forcing text interpretation and empty/unterminated-string → error, in `services/twig/internal/filter/parser.go`. Depends on T009.
- [ ] T011 [US1] Implement the evaluator: case-insensitive text match over name + description, FR-008 implicit `completed=false`/`snoozed=false` when `show_all` is off (no explicit conditions yet), returning the ascending matched-id set, in `services/twig/internal/filter/eval.go`. Depends on T009.
- [ ] T012 [P] [US1] Add filter state fields (`filterInput`, `filterFocused`, `filterExpr`, `filterMatches`, `filterInvalid`, `filterGen`) per data-model.md to the model in `internal/tui/model.go`.
- [ ] T013 [P] [US1] Add a `FilterTasks` client call helper (sends `expression`, `show_all`, `today`) in `internal/tui/client.go`.
- [ ] T014 [US1] Register the `/` binding and filter-mode bindings and document them in `internal/tui/keymap.go` and `internal/tui/help.go`.
- [ ] T015 [US1] Handle filter-mode key/message flow in `internal/tui/update.go`: `/` opens the bar (pre-filled with `filterExpr` when a filter is applied), printable keys route to the input, each change fires a `FilterTasks` command tagged with an incremented `filterGen` (discard responses from a stale generation), a successful response stores `filterMatches`, an `InvalidArgument` response sets `filterInvalid` while retaining the previous matches, `Enter` applies and returns focus to the list, and `Esc` clears the filter. Depends on T012, T013.
- [ ] T016 [US1] Extend `buildVisible` with a filtered mode that emits exactly the matched tasks plus every ancestor of a match, preserving tree order and indentation, ignoring collapse/expansion state, in `internal/tui/tree.go`. Depends on T012.
- [ ] T017 [US1] Render the filter bar (active expression always visible, invalid indicator when `filterInvalid`) and the distinct no-matches empty state in `internal/tui/view.go`. Depends on T012.
- [ ] T018 [US1] Re-fire the active filter after task mutations (complete, snooze, edit, create) and after a `show_all` toggle so the filtered view stays consistent (FR-013), in `internal/tui/update.go`. Depends on T015.
- [ ] T019 [US1] Add the playful user-facing copy from quickstart.md (no-matches empty state, invalid-expression indicator, filter-bar placeholder) in `internal/tui/view.go`, honoring Principle IV.

**Checkpoint**: Text search is fully functional end-to-end — the MVP is shippable on its own.

---

## Phase 4: User Story 2 - Filter by task attributes (Priority: P2)

**Goal**: Type SQL-like attribute conditions (`completed`, `snoozed`, `parent_id`, `goal_id`, date comparisons on `completed`) combined with `AND`, mixing free text and attributes.

**Independent Test**: Enter each documented expression form (`completed=true AND parent_id=1`, `completed < 2026-01-01`, `snoozed=true`, `parent_id=1` with show-all off) and confirm the returned set matches the hand-computed set, including per-attribute override of default visibility; a malformed `completed=` keeps the last valid results.

### Tests for User Story 2 ⚠️ (write first, ensure they FAIL)

- [ ] T020 [P] [US2] Parser tests for field conditions (bool `completed`/`snoozed`, date `completed`, integer `parent_id`/`goal_id`), operator/type validation (unknown field, wrong op for type, malformed date `2026-13-45`, non-integer id, `!=` forms) → `InvalidArgument`, in `services/twig/internal/filter/parser_test.go`.
- [ ] T021 [P] [US2] Evaluator tests for the attribute worked-examples rows (`completed=true AND parent_id=1`/off, `completed < 2026-01-01`/off, `snoozed=true`/off, `parent_id=1`/off) plus per-attribute FR-008 override (explicit `completed=true` still excludes snoozed when show-all off) and date-implies-completed, in `services/twig/internal/filter/eval_test.go`.

### Implementation for User Story 2

- [ ] T022 [US2] Extend the parser to build `BoolCondition`, `DateCondition` (on `completed`), and direct `RelCondition` (`parent_id`, `goal_id`) with the type/operator rules from `contracts/filter-grammar.md`, in `services/twig/internal/filter/parser.go`. Depends on T010.
- [ ] T023 [US2] Extend the evaluator to satisfy bool (`completed`/`snoozed` vs `today`), date (`completed_at` vs `D@00:00 UTC`, implying completed), and direct `parent_id`/`goal_id` conditions, and to apply the FR-008 default only for attributes the expression does not mention, in `services/twig/internal/filter/eval.go`. Depends on T011.

**Checkpoint**: Text search (US1) and attribute filters both work independently.

---

## Phase 5: User Story 3 - Filter across an entire subtree or goal (Priority: P3)

**Goal**: Prefix `parent_id`/`goal_id` with `^` for transitive matching — `^parent_id=N` matches all descendants of N; `^goal_id=N` matches the whole subtree of any association root for goal N.

**Independent Test**: Build a three-level tree under one goal and confirm `^parent_id=1` returns the full descendant set (vs. `parent_id=1` returning only direct children) and `^goal_id=1` returns the whole subtree including roots.

### Tests for User Story 3 ⚠️ (write first, ensure they FAIL)

- [ ] T024 [P] [US3] Parser tests: `^` accepted only on `parent_id`/`goal_id` (any other field or a text term with `^` → `InvalidArgument`), in `services/twig/internal/filter/parser_test.go`.
- [ ] T025 [P] [US3] Evaluator tests for the transitive worked-examples rows (`completed=false AND ^parent_id=1` returns descendants at any depth excluding N; `completed=false AND snoozed=false AND ^goal_id=1` returns the whole subtree incl. roots; `groceries AND ^goal_id=2`) and nonexistent-id → empty, in `services/twig/internal/filter/eval_test.go`.

### Implementation for User Story 3

- [ ] T026 [US3] Extend the parser to accept and validate the `^` transitive prefix on `parent_id`/`goal_id`, setting `RelCondition.Transitive`, in `services/twig/internal/filter/parser.go`. Depends on T022.
- [ ] T027 [US3] Extend the evaluator to build the derived `children`, `byID`, and goal-roots maps and resolve `^parent_id` (all descendants) and `^goal_id` (association roots + their subtrees), in `services/twig/internal/filter/eval.go`. Depends on T023.

**Checkpoint**: All three query forms work independently and compose.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validate the whole feature against the spec's success criteria.

- [ ] T028 [P] Review/finalize the playful copy strings against Principle IV (empty state, invalid indicator, placeholder) and the constitution's UI/UX tone, in `internal/tui/view.go`.
- [ ] T029 Run `go test ./...` at the repo root and `cd services/twig && go test ./...`; confirm the `internal/filter` suite covers every row of the worked-examples table in `contracts/filter-grammar.md` (SC-003) and no keystroke sequence blanks/crashes the TUI (SC-004).
- [ ] T030 Run the quickstart.md `curl` against `FilterTasks` to confirm identical results via the backend directly (SC-005) and that an invalid expression returns HTTP 400 `invalid_argument`.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 → T002 (regen needs the proto change). No other dependencies.
- **Foundational (Phase 2)**: depends on Setup; T003 and T004 unblock all stories. BLOCKS every user story.
- **User Stories (Phase 3–5)**: all depend on Foundational.
  - US1 delivers the input mechanism, handler flow, lexer, and TUI — the MVP.
  - US2 and US3 extend only `parser.go` + `eval.go`; no TUI changes.
- **Polish (Phase 6)**: depends on the stories you intend to ship.

### User Story Dependencies

- **US1 (P1)**: independent — the only story that touches the TUI, handler wiring, and the lexer. Fully testable alone.
- **US2 (P2)**: independent to test, but its parser/eval edits build on the US1 `parser.go`/`eval.go` scaffolding (same files), so schedule it after US1.
- **US3 (P3)**: independent to test; its `^` handling builds on US2's `RelCondition` parsing/eval (same files), so schedule it after US2.

Because US1/US2/US3 all edit `services/twig/internal/filter/parser.go` and `eval.go`, the stories are **sequential** (P1 → P2 → P3), not parallelizable across developers — but each remains an independently verifiable increment.

### Within Each User Story

- Tests are written first and must FAIL before implementation.
- Lexer (T009) before parser (T010); parser + evaluator can then proceed.
- Model state (T012) before update/tree/view wiring (T015–T018).

### Parallel Opportunities

- Setup is strictly sequential (T001 → T002).
- Foundational: T003 [P] can start immediately; T004 follows once T002 + T003 land.
- **Within US1**: the four test tasks T005–T008 are [P] (different files). During implementation, T012 (model) and T013 (client) are [P]; the filter-package tasks T009–T011 run in the server module while the TUI tasks T012–T019 run in the root module — the two modules can progress in parallel once T004 defines the request path.
- **Within US2**: T020 [P] and T021 [P] (different test files).
- **Within US3**: T024 [P] and T025 [P] (different test files).

---

## Parallel Example: User Story 1

```bash
# Write all US1 tests together (different files):
Task: "Lexer + text-term parser tests in services/twig/internal/filter/parser_test.go"
Task: "Evaluator text + default-visibility tests in services/twig/internal/filter/eval_test.go"
Task: "FilterTasks handler test in services/twig/internal/handler/task_test.go"
Task: "TUI filter-mode behavior test in internal/tui/update_test.go"

# Then split by module: server filter package vs. TUI wiring
Task: "Implement lexer/parser/evaluator (T009–T011) — server module"
Task: "Add model state + client helper (T012, T013) — root module"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 Setup → Phase 2 Foundational.
2. Phase 3 US1 (text search + full TUI integration).
3. **STOP and VALIDATE**: locate a nested task by text; verify ancestors show, show-all interaction, Enter/Esc, and that mashing keys never blanks the list.
4. Ship — text search alone solves the stated problem.

### Incremental Delivery

1. Setup + Foundational → request path live.
2. US1 → text search MVP → demo.
3. US2 → attribute filters → demo.
4. US3 → transitive subtree/goal scoping → demo.
5. Polish → run the SC-003/004/005 validations.

---

## Notes

- [P] tasks = different files, no dependencies.
- The three stories share `parser.go`/`eval.go`, so they are sequential across stories even though each is independently testable.
- No database schema changes — evaluation is in-memory over `ListTasks` rows.
- `ListTasks` and all existing RPCs stay untouched.
- Verify each story's tests FAIL before implementing; commit after each task or logical group.
