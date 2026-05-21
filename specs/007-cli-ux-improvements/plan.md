# Implementation Plan: CLI UX Improvements

**Branch**: `007-cli-ux-improvements` | **Date**: 2026-05-21 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/007-cli-ux-improvements/spec.md`

## Summary

Four small UX improvements to the `todo` Go CLI: (1) `todo task` with no subcommand defaults to listing (matching `plan`), and the `list` subcommand is removed; (2) completed tasks render in dim/gray with strikethrough instead of `[x]`/`[ ]` checkbox markers, with graceful degradation when not a TTY; (3) `todo task mod` parses flags correctly when placed after the id and makes the name positional argument optional; (4) when a task has an `estimate > 0`, the value is appended as `(N)` after the task name in list output.

All changes are localized to `services/todo/internal/cli/` (`cli.go`, `task.go`, `render.go` and their tests). No protobuf, server, or storage changes.

## Technical Context

**Language/Version**: Go 1.25 (per `services/todo/go.mod`)

**Primary Dependencies**: `connectrpc.com/connect`, `golang.org/x/term` (already a transitive dep; used directly for TTY detection)

**Storage**: N/A (CLI only; reads `Task.estimate` from existing protobuf, which is already populated by the server)

**Testing**: `go test ./...` under `services/todo`; existing tests in `cli_test.go`, `task_test.go`, `render_test.go` are extended.

**Target Platform**: Linux/macOS terminal (ANSI-capable). Non-TTY output (pipes, redirects, `go test`) must emit plain text with no escape sequences.

**Project Type**: CLI subcommand of an existing Go service.

**Performance Goals**: N/A — formatting is O(tasks); rendering happens once per invocation.

**Constraints**: No new third-party dependencies. Reuse `golang.org/x/term.IsTerminal` for TTY detection. Output to non-TTY streams must contain no ANSI escape codes.

**Scale/Scope**: ~5 functions touched, ~3 test files updated. Single-user CLI.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Simplicity / YAGNI**: PASS. No new abstractions or packages. Styling is a thin helper (a function returning ANSI codes when stdout is a TTY, empty strings otherwise). The flag-parsing fix re-uses the existing `flag.FlagSet` pattern, taking the id positionally and reslicing args before `fs.Parse`.
- **II. API-First Design**: PASS (N/A in spirit). No network contracts change. The CLI command schema *is* the user-facing contract; it is documented in `contracts/cli-commands.md` below before implementation. The `Task.estimate` field already exists in `proto/task/v1/task.proto`; no proto edits.

No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/007-cli-ux-improvements/
├── plan.md              # This file
├── spec.md              # Feature spec
├── research.md          # Phase 0: TTY detection + ANSI styling decisions
├── data-model.md        # Phase 1: no new entities (notes only)
├── quickstart.md        # Phase 1: manual verification recipe
├── contracts/
│   └── cli-commands.md  # CLI command schema (the contract)
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (created by /speckit-tasks)
```

### Source Code (repository root)

```text
services/todo/
├── cmd/todo/main.go                          # entrypoint (unchanged)
└── internal/cli/
    ├── cli.go                                # EDIT: dispatch `task` with no args to list; remove `list` case; update usage
    ├── task.go                               # EDIT: rewrite runMod arg parsing; runList stays but is called from runTask directly
    ├── render.go                             # EDIT: replace checkboxPrefix with style-based rendering; append estimate
    ├── cli_test.go                           # EDIT: cover new dispatch behavior
    ├── task_test.go                          # EDIT: cover `task mod` flag-after-id and optional-name cases
    └── render_test.go                        # EDIT: cover styled vs plain rendering and estimate suffix
```

**Structure Decision**: Single Go module (`services/todo`). All work happens in the existing `internal/cli` package — no new packages or files. This matches existing conventions and Principle I (simplicity).

## Complexity Tracking

> Constitution Check passes with no violations. Table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|

## Phase 0 — Research

See [research.md](./research.md). Key decisions:

- **ANSI styling without a new dependency**: emit raw escape sequences (`\x1b[2;9m` for dim+strikethrough, `\x1b[0m` to reset). Gate behind `term.IsTerminal(int(os.Stdout.Fd()))`. Tests exercise both branches via an injectable `styler` value passed into `renderRoots`/`renderTree`.
- **Flag-after-positional parsing**: Go's stdlib `flag` package stops at the first non-flag argument. Solution: take the id from `args[0]`, parse the rest with `fs.Parse(args[1:])`, then treat `fs.Args()` as the optional name(s). This permits all of `mod <id>`, `mod <id> "name"`, `mod <id> --parent 5`, `mod <id> --parent 5 "name"`, `mod <id> "name" --parent 5`.
- **Estimate rendering**: `Task.estimate` is `int32`; `0` means "no estimate" (per `proto/task/v1/task.proto:77`). Append ` (N)` only when `> 0`.

## Phase 1 — Design & Contracts

- [data-model.md](./data-model.md) — no entity changes; describes which existing `Task` fields drive rendering.
- [contracts/cli-commands.md](./contracts/cli-commands.md) — updated CLI command schema (the user-facing contract).
- [quickstart.md](./quickstart.md) — manual verification steps.
- Agent context: `CLAUDE.md` updated to point at this plan.

### Post-design Constitution Re-Check

- **I. Simplicity**: still PASS — no extra packages introduced; styling is one helper function and one TTY check.
- **II. API-First**: still PASS — `contracts/cli-commands.md` documents the new shape before implementation begins.
