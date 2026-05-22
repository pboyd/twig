# Tasks: CLI Help System

**Input**: Design documents from `specs/008-cli-help/`

**Branch**: `008-cli-help`

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to
- All source paths are relative to `services/todo/`

---

## Phase 1: Foundational (Blocking Prerequisite)

**Purpose**: Add the `runHelp()` dispatcher and refactor print functions to accept a writer, enabling all per-command help to be added independently.

**⚠️ CRITICAL**: All user story phases depend on this.

- [X] T001 Refactor `printRootUsage()`, `printTaskUsage()`, `printPomUsage()`, and `printPlanUsage()` to accept an `io.Writer` parameter; update all existing call sites to pass `os.Stderr` in `internal/cli/cli.go`, `internal/cli/task.go`, `internal/cli/pom.go`, `internal/cli/plan.go`
- [X] T002 Add `runHelp(args []string) int` function in `internal/cli/cli.go` that dispatches `todo help [command]` — for an empty args list it calls the root usage writer (stdout, exit 0); for a known command name calls that command's usage writer (stdout, exit 0); for an unknown command name prints an error and exits non-zero

**Checkpoint**: Foundation ready — per-command and root help can now be added in parallel

---

## Phase 2: User Story 1 — Root Help (Priority: P1) 🎯 MVP

**Goal**: `todo help` and `todo --help` display top-level usage and exit 0.

**Independent Test**: Run `todo help` and `todo --help`; confirm each lists `task`, `pom`, and `plan` with one-line descriptions, writes to stdout, and exits 0. Also confirm `todo` with no arguments still exits non-zero (unchanged).

- [X] T003 [US1] Wire `"help"` case and `"--help"` check into the `Run()` switch in `internal/cli/cli.go` so both invoke `runHelp()` with the remaining args (or no args for `--help`), exiting 0
- [X] T004 [US1] Improve the content of `printRootUsage()` in `internal/cli/cli.go` to include a one-line description for each top-level command (`task`, `pom`, `plan`) and a usage line showing `todo help [command]`

**Checkpoint**: `todo help` and `todo --help` work correctly; `todo` (no args) still exits non-zero

---

## Phase 3: User Story 2 — Per-Command Help (Priority: P1)

**Goal**: `todo help <command>` and `todo <command> --help` display detailed command usage and exit 0.

**Independent Test**: Run `todo help task`, `todo help pom`, `todo help plan`, `todo task --help`, `todo pom --help`, `todo plan --help`, and `todo help unknown`; verify each behaves per spec FR-003 through FR-008.

- [X] T005 [P] [US2] Add `--help` check at the top of `runTask()` in `internal/cli/task.go`; when `args[0] == "--help"`, call `printTaskUsage(os.Stdout)` and return 0. Improve `printTaskUsage()` content to document all subcommands (`add`, `rm`, `mod`, `complete`) with their flags (`--parent`, `--due`, `--completed`, `--all`)
- [X] T006 [P] [US2] Add `--help` check at the top of `runPom()` in `internal/cli/pom.go`; when `args[0] == "--help"`, call `printPomUsage(os.Stdout)` and return 0. Improve `printPomUsage()` content to document all subcommands (`estimate`, `start`, `resume`, `cancel`, `status`) with their flags
- [X] T007 [P] [US2] Add `--help` check at the top of `runPlan()` in `internal/cli/plan.go`; when `args[0] == "--help"`, call `printPlanUsage(os.Stdout)` and return 0. Improve `printPlanUsage()` content to document all subcommands and include time/duration format examples
- [X] T008 [US2] Handle `todo help <unknown>` in `runHelp()` in `internal/cli/cli.go`: print `"unknown command: <cmd>\nRun 'todo help' for usage."` to stderr and return 1

**Checkpoint**: All three `todo help <cmd>` and `todo <cmd> --help` paths work; `todo help unknown` exits non-zero with a useful error

---

## Phase 4: User Story 3 — Error Message Hints (Priority: P2)

**Goal**: Unknown command/subcommand errors tell users how to get help.

**Independent Test**: Run `todo badcmd` and `todo task badsubcmd`; confirm each error message includes a `Run 'todo help ...' for usage.` hint.

- [X] T009 [US3] Update the unknown-command error in `Run()` in `internal/cli/cli.go` to append `"Run 'todo help' for usage."` after the `"unknown command: ..."` line
- [X] T010 [P] [US3] Update the unknown-subcommand errors in `runTask()`, `runPom()`, and `runPlan()` in `internal/cli/task.go`, `internal/cli/pom.go`, and `internal/cli/plan.go` to append `"Run 'todo help <cmd>' for usage."` (substituting the appropriate command name)

**Checkpoint**: All error paths for unknown commands/subcommands include a help hint

---

## Phase 5: Polish & Cross-Cutting Concerns

- [X] T011 Add tests for help routing: verify `Run([]string{"help"})`, `Run([]string{"--help"})`, `Run([]string{"help", "task"})`, `Run([]string{"help", "unknown"})` return the expected exit codes in `internal/cli/cli_test.go`
- [X] T012 [P] Verify that `todo help`, `todo help task`, `todo help pom`, and `todo help plan` output contains no raw ANSI escape sequences when stdout is not a TTY (pipe the output to `cat` and inspect)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No external dependencies — start immediately
- **US1 (Phase 2)**: Depends on T001 + T002
- **US2 (Phase 3)**: Depends on T001 + T002 (T005, T006, T007 are parallel with each other and with US1)
- **US3 (Phase 4)**: Independent of US1 and US2 — depends only on Phase 1
- **Polish (Phase 5)**: Depends on all story phases complete

### User Story Dependencies

- **US1 (P1)**: Depends on Foundational only
- **US2 (P1)**: Depends on Foundational only; T005/T006/T007 can run in parallel with each other and with US1
- **US3 (P2)**: Depends on Foundational only; can run in parallel with US1 and US2

### Parallel Opportunities

- T005, T006, T007 can run in parallel (different files)
- T009, T010 can run in parallel (error-hint changes are independent per file)
- US1, US2, and US3 phases can all proceed once Phase 1 is complete

---

## Parallel Example: User Story 2

```
# After T001 + T002 complete, launch all three in parallel:
Task T005: "Add --help to runTask() and improve printTaskUsage() in internal/cli/task.go"
Task T006: "Add --help to runPom() and improve printPomUsage() in internal/cli/pom.go"
Task T007: "Add --help to runPlan() and improve printPlanUsage() in internal/cli/plan.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2 Only)

1. Complete Phase 1: Foundational (T001, T002)
2. Complete Phase 2: US1 (T003, T004)
3. Complete Phase 3: US2 (T005–T008, parallelize T005/T006/T007)
4. **STOP and VALIDATE**: `todo help`, `todo --help`, `todo help task/pom/plan`, `todo task/pom/plan --help` all work
5. Ship or continue to US3

### Incremental Delivery

1. Foundational → routing wired and writers refactored
2. US1 → root `todo help` works (MVP for discoverability)
3. US2 → per-command help works (fully useful help system)
4. US3 → error messages improved (quality of life)
5. Polish → tested and verified clean

---

## Notes

- All changes are in `services/todo/internal/cli/` — no new files, no new dependencies
- The `io.Writer` refactor (T001) is the only cross-cutting change; all other tasks are scoped to a single file
- Existing behavior when `todo` is run with no arguments (exit non-zero) must not change
- Help output for explicit `help` command always goes to stdout; usage shown on error continues to use stderr
