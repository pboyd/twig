---

description: "Task list for feature 007-cli-ux-improvements"
---

# Tasks: CLI UX Improvements

**Input**: Design documents from `/specs/007-cli-ux-improvements/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-commands.md, quickstart.md

**Tests**: This project already has unit tests for `internal/cli`; new test tasks are included to extend that suite. (Tests are included because the existing codebase has them — not as TDD-first.)

**Organization**: Tasks are grouped by user story. Story phase order matches priority from spec.md: US1 (P1) bug fix first, then US2/US3 (P2), then US4 (P3).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Maps to user story (US1–US4)
- Paths are repo-relative.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Make sure the local environment can build/test the existing CLI before any edits.

- [X] T001 Verify baseline build: run `go build ./...` and `go test ./internal/cli/...` from `services/todo/`; capture current pass/fail state in a scratch note (not committed).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: One shared helper for TTY-gated ANSI styling. All later stories that touch rendering depend on this.

- [X] T002 Add styling helper to `services/todo/internal/cli/render.go`: introduce a `styler` type (or two small functions `dimStrike(s string) string` and `wantStyled(w io.Writer) bool`) that emit `\x1b[2;9m…\x1b[0m` only when styling is enabled, and the raw string otherwise. TTY detection uses `golang.org/x/term.IsTerminal(int(os.Stdout.Fd()))`. Add `golang.org/x/term` to `services/todo/go.mod` direct requires (currently indirect). Keep the helper unexported.
- [X] T003 Plumb a `styled bool` (or equivalent) parameter through `renderRoots` and `renderTree` in `services/todo/internal/cli/render.go` so tests can force styled/plain output without depending on the process's stdout TTY state.

**Checkpoint**: Foundation ready — styling can be toggled deterministically in tests; subsequent stories can change rendering output independently.

---

## Phase 3: User Story 1 — Re-parent without renaming (Priority: P1) 🎯 MVP

**Goal**: Fix `todo task mod <id> --parent <p>` so it updates the parent and leaves the name alone, and accept flags in any position after the id.

**Independent Test**: Add a task with a known name and parent, run `todo task mod <id> --parent <other-id>`, then `todo task` and confirm the task's name is unchanged and its parent is updated. Unit tests in `task_test.go` cover the parsing paths without a backend.

### Tests for User Story 1

- [X] T004 [P] [US1] In `services/todo/internal/cli/task_test.go`, add a table-driven test for the `runMod` arg-parsing layer covering: (a) `mod 3` → "nothing to update" error; (b) `mod 3 "new name"` → name change only; (c) `mod 3 --parent 5` → parent change only, name preserved; (d) `mod 3 --parent 5 "new name"` → both; (e) `mod 3 "new name" --parent 5` → both; (f) `mod 3 ""` → empty-name error; (g) `mod 3 a b c` → unexpected-args error. Mock or stub the `TaskServiceClient` so the test asserts on the `UpdateTaskRequest` constructed, not on a real RPC.

### Implementation for User Story 1

- [X] T005 [US1] Rewrite `runMod` in `services/todo/internal/cli/task.go`:
  - Require `args[0]` as `<id>`; parse it as int64 first.
  - Call `fs.Parse(args[1:])`.
  - After parsing, `rest := fs.Args()`. If `len(rest) > 1` → usage error. If `len(rest) == 1` and `rest[0] == ""` → reject empty name. If `len(rest) == 0` AND no flags were set (`parentStr == "" && dueStr == ""`) → "nothing to update" error.
  - Fetch the existing task (`GetTask`), then build `UpdateTaskRequest` preserving fields and overriding only those the user provided. The name override happens only when `len(rest) == 1`.
  - Update the usage string in the error path to match the new shape: `usage: todo task mod <id> [<name>] [--parent <id>] [--due <timestamp>]`.

**Checkpoint**: `todo task mod 3 --parent 5` no longer renames task 3. US1 complete and the most painful bug is fixed; can be merged on its own as an MVP.

---

## Phase 4: User Story 2 — `todo task` lists by default (Priority: P2)

**Goal**: `todo task` (no subcommand) lists tasks. The `list` subcommand is removed.

**Independent Test**: Run `todo task` with no args and confirm it produces the same output as today's `todo task list`. Run `todo task list` and confirm an unknown-subcommand error.

### Tests for User Story 2

- [X] T006 [P] [US2] In `services/todo/internal/cli/cli_test.go`, add tests for the `runTask` dispatch: (a) empty `args` invokes the list path; (b) `["list"]` produces an unknown-subcommand error. Stub `runList` via the existing `export_test.go` hooks or by routing through a small indirection if needed; do not require a live backend.

### Implementation for User Story 2

- [X] T007 [US2] Edit `runTask` in `services/todo/internal/cli/cli.go`:
  - When `len(args) == 0`, build the `TaskServiceClient` and call `runList(client, nil)`. Mirror the empty-arg branch of `runPlan` in `plan.go:43-45`.
  - Remove the `case "list":` arm from the dispatch switch.
- [X] T008 [US2] Update `printTaskUsage` in `services/todo/internal/cli/cli.go` to drop the `list` entry and document the new default-list behavior per `contracts/cli-commands.md` § "Help text".

**Checkpoint**: Both US1 and US2 are independently functional. The CLI now matches the `plan` command's shape and the flag bug is fixed.

---

## Phase 5: User Story 3 — Gray + strikethrough completed tasks (Priority: P2)

**Goal**: Replace the `[x]`/`[ ]` checkbox markers with dim+strikethrough rendering for completed tasks; incomplete tasks render in the default style.

**Independent Test**: Create one complete and one incomplete task, run `todo task --all` in a TTY and visually confirm the completed line is dim and struck-through with no checkbox markers; run `todo task --all | cat` and confirm no ANSI escape codes appear in the captured output.

### Tests for User Story 3

- [X] T009 [P] [US3] In `services/todo/internal/cli/render_test.go`, add tests for `renderRoots` covering: (a) styled=true on a completed task wraps the post-id content with `\x1b[2;9m` and `\x1b[0m`; (b) styled=false on the same task produces a plain string with no escapes; (c) no `[x]` or `[ ]` marker appears in either case; (d) incomplete tasks are unstyled in both modes.

### Implementation for User Story 3

- [X] T010 [US3] In `services/todo/internal/cli/render.go`, remove `checkboxPrefix` and its call sites. Change `renderTree` and `renderRoots` so the per-line content is built as a plain string (id + name + estimate + suffixes), then the post-id portion is run through the styling helper from T002 when the task is completed and styling is enabled. The leading `[<id>] ` (or `connector + [<id>] `) MUST NOT be styled.

**Checkpoint**: US3 complete; piping output to a file produces no escape codes, and an interactive run shows completed tasks in dim+strikethrough.

---

## Phase 6: User Story 4 — Pomodoro estimate in parentheses (Priority: P3)

**Goal**: When a task has `estimate > 0`, render ` (N)` immediately after the name in list output.

**Independent Test**: With a task whose `estimate` is 3, run `todo task` and confirm its line includes `<name> (3)`. With a task whose `estimate` is 0, confirm no parenthesized number appears.

### Tests for User Story 4

- [X] T011 [P] [US4] In `services/todo/internal/cli/render_test.go`, add cases: (a) `estimate = 3` renders ` (3)` immediately after the name and before any `(due …)` / `(completed …)` suffix; (b) `estimate = 0` renders no parenthesized estimate; (c) for a completed task with `estimate = 2`, the `(2)` is inside the dim+strikethrough region when styled=true.

### Implementation for User Story 4

- [X] T012 [US4] In `services/todo/internal/cli/render.go`, within the line-building code shared by `renderRoots`/`renderTree`, append ` (%d)` after `task.Name` when `task.GetEstimate() > 0`. Ordering: name, then optional ` (N)`, then ` (due …)`, then ` (completed …)`.

**Checkpoint**: All four user stories functional.

---

## Phase 7: Polish & Cross-Cutting

- [X] T013 Run `go test ./...` from `services/todo/` and ensure the full suite passes. Fix any failures introduced by the rendering format change (some existing render assertions that look for `[x] ` / `[ ] ` will need updating to the new format).
- [X] T014 [P] Run the quickstart steps from `specs/007-cli-ux-improvements/quickstart.md` against a live backend and confirm each expected behavior.
- [X] T015 [P] `go vet ./...` and `gofmt -l services/todo/internal/cli` clean.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (T001)**: no deps.
- **Foundational (T002, T003)**: depends on T001. **Blocks US3 and US4** (both depend on the rendering pipeline change). US1 and US2 do not strictly need T002/T003, but T002/T003 are tiny and land first to keep render.go edits coherent.
- **US1 (T004–T005)**: depends only on Foundational.
- **US2 (T006–T008)**: depends only on Foundational.
- **US3 (T009–T010)**: depends on Foundational (T002, T003).
- **US4 (T011–T012)**: depends on Foundational. Independent of US3 in principle, but both edit `render.go`; T010 and T012 should be sequenced to avoid merge conflicts within the same file.
- **Polish (T013–T015)**: depends on all preceding stories.

### Within Each User Story

- Tests (T004, T006, T009, T011) are written before or alongside their implementation tasks.
- US3 implementation (T010) must remove `checkboxPrefix` *before* US4 (T012) edits the same line-builder.

### Parallel Opportunities

- T004, T006, T009, T011 are all in different test cases and can be drafted in parallel.
- T015 (vet/fmt) can run while T014 (manual quickstart) is in progress.
- Different stories can be split across developers after Foundational lands.

---

## Parallel Example: After Foundational

```bash
# US1 and US2 touch disjoint files (task.go vs cli.go) — fully parallel.
Task: "T005 Rewrite runMod parsing in services/todo/internal/cli/task.go"
Task: "T007/T008 Update runTask dispatch + usage in services/todo/internal/cli/cli.go"

# Tests for all four stories can be drafted in parallel before implementations land:
Task: "T004 runMod parsing tests in task_test.go"
Task: "T006 runTask dispatch tests in cli_test.go"
Task: "T009 styled rendering tests in render_test.go"
Task: "T011 estimate rendering tests in render_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. T001 → T002 → T003 → T004 → T005.
2. **Stop and validate**: the parent-bug is fixed and tests cover the new argparsing surface. Could merge here and ship the fix immediately.

### Incremental Delivery

1. MVP (US1) → merge.
2. Add US2 → merge.
3. Add US3 → merge.
4. Add US4 → merge.
5. Polish (T013–T015) → final merge.

### Notes on file conflicts

- `render.go` is touched by Foundational (T002, T003), US3 (T010), and US4 (T012). Land them sequentially in that order.
- `render_test.go` is touched by US3 (T009) and US4 (T011); these add independent test functions, so they're safely parallelizable as drafts but should be merged carefully if branched.
- `cli.go` is touched only by US2; `task.go` only by US1 — no conflicts between those stories.
