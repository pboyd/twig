# Implementation Plan: CLI Help System

**Branch**: `008-cli-help` | **Date**: 2026-05-22 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/008-cli-help/spec.md`

## Summary

Add a `help` command and `--help` flag support to the todo CLI so users can discover usage for any command without reading source code. The implementation extends the existing `printXUsage()` functions and adds routing for `todo help [command]` and `todo [command] --help` in the CLI entry points.

## Technical Context

**Language/Version**: Go (module at `services/todo/`)

**Primary Dependencies**: `flag`, `fmt`, `os` from the standard library — no new dependencies

**Storage**: N/A

**Testing**: `go test ./internal/cli/...`

**Target Platform**: Linux terminal (single user)

**Project Type**: CLI

**Performance Goals**: N/A — help output is instant

**Constraints**: No new dependencies; no new abstractions; reuse and improve existing `printXUsage()` functions

**Scale/Scope**: 3 top-level commands, ~10 subcommands total

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | **PASS** | Extend existing `printXUsage()` functions; add `help` case to existing switch statements. No new abstractions, no library. |
| II. API-First Design | **PASS (N/A)** | No new API endpoints or proto changes. Purely CLI presentation. |

## Project Structure

### Documentation (this feature)

```text
specs/008-cli-help/
├── plan.md              # This file
├── research.md          # Phase 0 output
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

No `data-model.md`, `quickstart.md`, or `contracts/` needed — no new data or API surface.

### Source Code

```text
services/todo/
└── internal/cli/
    ├── cli.go           # Add help routing + --help at root; improve printRootUsage()
    ├── task.go          # Add --help handling in runTask(); improve printTaskUsage()
    ├── pom.go           # Add --help handling in runPom(); improve printPomUsage()
    ├── plan.go          # Add --help handling in runPlan(); improve printPlanUsage()
    └── cli_test.go      # Tests for help routing and --help flag
```

**Structure Decision**: All changes are within the existing `internal/cli` package. No new files needed.

## Phase 0: Research

### Unknowns

No significant unknowns. The codebase is well-understood:

- Help routing fits naturally into the existing `switch args[0]` in `Run()` and each `runXTop()`.
- The existing `printXUsage()` functions already produce the right content; they need accuracy improvements and stdout routing.
- `--help` at the root level must be intercepted before the switch (check `args[0] == "--help"`).
- `--help` at the command level must be checked at the top of each `runXTop()` before any other work.

### Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Help library | None — use `fmt.Fprintln` | Keeps zero new dependencies; existing pattern is already in place |
| Output stream for `help` command | stdout | Spec FR-011: help invoked explicitly goes to stdout |
| Output stream for no-args usage | stderr (unchanged) | Spec FR-012: current behavior retained; exits non-zero |
| `--help` scope | Root + per-command only | Spec assumption: `todo task add --help` is out of scope |
| Error hint format | `"Run 'todo help' for usage."` or `"Run 'todo help <cmd>' for usage."` | Consistent, discoverable |
