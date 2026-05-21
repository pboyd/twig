# CLI Command Contract

This document is the source of truth for the user-facing CLI shape changed by feature `007-cli-ux-improvements`. Implementation MUST conform.

## `todo task` (new no-arg behavior)

```
todo task [--completed | --all]
```

- Equivalent to the previous `todo task list [--completed | --all]`.
- Default behavior: print incomplete tasks as a tree.
- `--completed` and `--all` flags are mutually exclusive (existing behavior, preserved).
- The `list` subcommand is **removed**; `todo task list` produces an unknown-subcommand error.

### Output format (per line)

```
[<id>] <name>[ (<estimate>)][ (due <RFC3339>)][ (completed <RFC3339>)]
```

- The `[ ] ` and `[x] ` checkbox prefixes are **removed** entirely.
- When `<task>.estimate > 0`, the ` (<estimate>)` suffix is appended immediately after `<name>`.
- When the task is completed (`completed_at != nil`) AND stdout is a TTY, the entire content of the line **after the `[<id>] ` prefix** is wrapped with ANSI escape sequences `\x1b[2;9m` (dim + strikethrough) and reset with `\x1b[0m`. The leading `[<id>] ` prefix and any tree-style connectors (`├──`, `└──`, `│`) are NOT styled.
- When stdout is not a TTY (pipe/redirect/test capture), no escape sequences are emitted; the styled segment is the plain string.

### Subcommands of `todo task` (unchanged shape; documented for completeness)

```
todo task add [--parent <id>] [--due <timestamp>] <name>
todo task rm <id>
todo task mod <id> [<name>] [--parent <id>] [--due <timestamp>]    # see below
todo task complete <id>
```

## `todo task mod` (revised argument parsing)

```
todo task mod <id> [<name>] [--parent <id>] [--due <timestamp>]
```

- `<id>` is required and MUST be the first positional argument.
- `<name>` is **optional**. If omitted, the task's name is preserved.
- Flags (`--parent`, `--due`) may appear in any position after `<id>` — before, after, or interleaved with `<name>`.
- If `<id>` is the only argument and no flags are set, the command is an error: `nothing to update`.
- An explicitly empty name (`""`) is rejected.

### Examples

| Invocation | Result |
|---|---|
| `todo task mod 3` | error: nothing to update |
| `todo task mod 3 "New name"` | name → "New name"; parent/due unchanged |
| `todo task mod 3 --parent 5` | parent → 5; name unchanged |
| `todo task mod 3 --parent 5 "New name"` | name → "New name"; parent → 5 |
| `todo task mod 3 "New name" --parent 5` | name → "New name"; parent → 5 |
| `todo task mod 3 --parent 5 --due 2026-06-01` | parent → 5; due → 2026-06-01T00:00:00Z; name unchanged |
| `todo task mod 3 ""` | error: name cannot be empty |
| `todo task mod 3 a b c` | error: unexpected extra arguments |

## Help text

`printTaskUsage` MUST drop the `list` line. New text:

```
Usage: todo task [<subcommand>] [arguments]

Subcommands (if omitted, lists tasks):
  add [--parent <id>] [--due <timestamp>] <name>
                   Add a new task
  rm <id>          Remove a task
  mod <id> [<name>] [--parent <id>] [--due <timestamp>]
                   Modify a task (name optional)
  complete <id>    Mark a task complete

When invoked with no subcommand, lists tasks as a tree:
  todo task [--completed | --all]
```
