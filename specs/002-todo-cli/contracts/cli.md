# CLI Contract: Todo CLI Tool

**Feature**: 002-todo-cli | **Date**: 2026-05-18

This is the source-of-truth contract for the `todo` command-line interface — the
surface the tool presents to its user. The backend wire contract it consumes is
`specs/001-task-crud-api/contracts/task.proto` (unchanged by this feature).

## Invocation

```text
todo task <subcommand> [arguments] [flags]
```

- `<subcommand>` is one of `list`, `add`, `rm`, `mod` (FR-001).
- Backend address: the `TODO_ADDR` environment variable; defaults to
  `http://localhost:8080` when unset.
- With no arguments, an unknown command group, or an unknown subcommand, the
  tool prints usage help and exits `1` (FR-017).

## Exit codes

| Code | Meaning |
|---|---|
| `0` | The command succeeded. |
| `1` | Any failure — usage error, malformed argument, not-found, validation, parent cycle, or unreachable backend (FR-015; single non-zero code, per clarification). |

Error messages are written to **stderr**; normal command output is written to
**stdout**.

## `todo task list`

List every task as a tree.

- **Arguments**: none.
- **Flags**: none.
- **Backend call**: `ListTasks`.
- **Output**: one line per task, depth-first from each root, drawn with
  `tree`-style ASCII connectors (`├──`, `└──`, `│`). Tasks with no parent are
  roots; children are nested under their parent to unbounded depth. Siblings
  (and roots) appear in `id`-ascending order (FR-002, FR-003, FR-004).
- **Per-line content**: the task's `id`, `name`, and `due` date. A task with no
  due date shows a blank/empty due field rather than erroring (FR-003).
- **Due format**: RFC 3339 in UTC, e.g. `2026-06-01T17:00:00Z`.
- **Empty result**: when there are no tasks, print a plain message such as
  `no tasks` and exit `0` (FR-005).

Illustrative output:

```text
[1] Write the report (due 2026-06-01T17:00:00Z)
├── [2] Draft section 1
│   └── [4] Gather sources (due 2026-05-25T00:00:00Z)
└── [3] Draft section 2
[5] Buy groceries
```

(The exact column/label styling is an implementation detail; the connector
glyphs, ordering, and the id/name/due fields are the contract.)

## `todo task add`

Create a task.

```text
todo task add [--parent <id>] [--due <timestamp>] <name>
```

- **Arguments**: `<name>` — required positional, the task name (FR-006).
- **Flags**:
  - `--parent <id>` — optional; integer id of the task to nest under (FR-007).
  - `--due <timestamp>` — optional; RFC 3339 date-time (`2026-06-01T17:00:00Z`)
    or a bare calendar date (`2026-06-01`, interpreted as `00:00:00Z` UTC).
- **Backend call**: `CreateTask`.
- **Success**: prints a confirmation naming the new task's id (FR-008), exit `0`.
- **Failures** (exit `1`, message to stderr):
  - `<name>` missing → usage error, no RPC made.
  - `--parent`/`--due` malformed → usage error, no RPC made (FR-013).
  - `--parent` references no existing task → backend error surfaced.

## `todo task rm`

Delete a task.

```text
todo task rm <id>
```

- **Arguments**: `<id>` — required positional integer (FR-009).
- **Flags**: none.
- **Backend call**: `DeleteTask`. The backend cascades the delete to all
  descendants of the task (the whole subtree is removed).
- **Success**: prints a confirmation naming the deleted task's id, exit `0`.
- **Failures** (exit `1`, message to stderr):
  - `<id>` malformed or missing → usage error, no RPC made.
  - `<id>` references no existing task → backend not-found error surfaced.

## `todo task mod`

Modify an existing task.

```text
todo task mod [--parent <id>] [--due <timestamp>] <id> <name>
```

- **Arguments**:
  - `<id>` — required positional integer, the task to modify (FR-010).
  - `<name>` — required positional, the task's new name (FR-010).
- **Flags**:
  - `--parent <id>` — optional; new parent task id (FR-011).
  - `--due <timestamp>` — optional; new due date, same formats as `add`.
- **Backend calls**: `GetTask` then `UpdateTask` (fetch-then-update). Fields not
  named on the command line — including the task's description — are preserved
  (FR-012). This version cannot clear a due date or detach a parent.
- **Success**: prints a confirmation naming the updated task's id, exit `0`.
- **Failures** (exit `1`, message to stderr):
  - `<id>`/`<name>` missing, or `<id>`/`--parent`/`--due` malformed → usage
    error, no RPC made (FR-013).
  - `<id>` references no existing task → backend not-found error surfaced; no
    `UpdateTask` is attempted.
  - `--parent` references no existing task → backend error surfaced.
  - `--parent` would make the task its own ancestor → backend cycle error
    surfaced.

## Error message contract (FR-014, FR-016, SC-004)

Every failure produces a message that identifies the cause:

| Cause | Source | Message basis |
|---|---|---|
| Missing/extra argument, bad subcommand | local | usage text for the command |
| Non-integer `<id>` or `--parent` | local | states the argument must be an integer |
| Unparseable `--due` | local | names the accepted timestamp formats |
| Unknown task / unknown parent / cycle / name too long | backend | the backend `connect.Error` message, surfaced verbatim |
| Backend unreachable | local (transport) | states the backend address could not be reached |

All of the above exit `1`.
