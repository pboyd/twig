# Research: CLI Help System

**Feature**: `008-cli-help`

## Summary

No external research required. All decisions are based on reading the existing CLI source.

## Findings

### Current State

All four entry points (`Run`, `runTask`, `runPomTop`/`runPom`, `runPlan`) already have a `printXUsage()` function and call it on error. The existing content is mostly accurate but not reachable via a user-friendly `help` command.

Key gaps:
1. No `todo help` command exists — usage text only appears on error.
2. Usage text is written to stderr; when invoked explicitly it should go to stdout.
3. Error messages for unknown commands don't tell users how to get help.
4. `--help` flag is not recognized at any level.

### Design Choices

| Decision | Choice | Rationale | Alternatives Considered |
|----------|--------|-----------|------------------------|
| Help library | None | Zero deps; fmt.Fprintln already in use everywhere | `cobra`, `urfave/cli` — over-engineered for a single-user tool |
| Output stream | stdout for explicit help; stderr for error-triggered usage | FR-011/FR-012; enables piping help output | stdout-only would require changing error-path behavior |
| `--help` scope | Root + `todo <cmd> --help` only | Spec assumption; keeps scope small | Per-subcommand `--help` (out of scope) |
| Error hint | Append `"Run 'todo help [cmd]' for usage."` | Consistent, low-noise | Printing full usage on every error (too verbose) |

### Implementation Approach

1. In `cli.go / Run()`:
   - Add `"--help"` check before the switch → call `printRootHelp()` → exit 0
   - Add `"help"` case → dispatch to `runHelp(args[1:])`
   - `runHelp(args)` routes to per-command help or root help
   - Update `printRootUsage()` → `printRootHelp()`, write to stdout, exit 0
   - Update error messages to append hint

2. In `task.go`, `pom.go`, `plan.go`:
   - Check `args[0] == "--help"` at the top of the command handler → print command help to stdout → exit 0
   - Improve `printXUsage()` content for completeness

No new packages, no new files.
