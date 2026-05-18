# Phase 1 Data Model: Todo CLI Tool

**Feature**: 002-todo-cli | **Date**: 2026-05-18

The CLI stores nothing of its own. It consumes the `Task` entity defined by the
backend contract and builds one transient in-memory structure (the render tree).
This document describes the data as the CLI sees and shapes it.

## Consumed entity: Task

Defined by `specs/001-task-crud-api/contracts/task.proto`; the CLI receives it
as the generated `taskv1.Task`. Only the fields the tool uses are listed.

| Field | Generated Go type | Used for | Notes |
|---|---|---|---|
| `id` | `int64` | display, `<id>`/`--parent` arguments | Server-assigned, positive. |
| `name` | `string` | display, `add`/`mod` input | Required; backend enforces non-blank, ≤ 255 chars. |
| `description` | `string` | preserved by `mod` | Never displayed or set by this tool; carried through `mod` unchanged. |
| `due` | `*timestamppb.Timestamp` | display, `--due` input | `nil` when absent → blank due column. |
| `parent_id` | `*int64` | tree placement, `--parent` input | `nil` → root task. |

The CLI treats `description` as opaque: it reads it only so that `mod` can send
it back, satisfying the spec's no-data-loss assumption.

## Transient structure: render tree

Built by `list` from the flat `ListTasks` response; never persisted.

```text
treeNode
├── task      *taskv1.Task     // the task this node represents
└── children  []*treeNode      // child nodes, ordered by task.id ascending
```

Construction:

1. Receive `[]*taskv1.Task` from `ListTasks` (backend returns it ordered by `id`).
2. Index every task by `id`.
3. For each task: if `parent_id` is `nil`, it is a root; otherwise append it to
   its parent's `children`.
4. Roots and every `children` slice are kept in `id`-ascending order (FR-004).
   Because the backend already returns tasks `id`-ascending, a single pass
   preserves order with no explicit sort.

Rendering walks the roots depth-first, emitting one line per node with
`tree`-style connector glyphs (see `contracts/cli.md`). Depth is unbounded
(FR-002). The backend's foreign key guarantees every non-`nil` `parent_id`
references an existing task, so no orphan or dangling-parent case can occur.

## Command input models

Each subcommand parses argv into a small request shape, then maps it onto a
generated request message. No persisted model — these live only for the
duration of one invocation.

| Command | Inputs | Maps to |
|---|---|---|
| `task list` | none | `ListTasksRequest{}` |
| `task add` | `name` (positional, required); `--parent` (int, optional); `--due` (timestamp, optional) | `CreateTaskRequest{Name, ParentId?, Due?}` |
| `task rm` | `id` (positional, required, int) | `DeleteTaskRequest{Id}` |
| `task mod` | `id` (positional, required, int); `name` (positional, required); `--parent` (int, optional); `--due` (timestamp, optional) | `GetTaskRequest{Id}` then `UpdateTaskRequest{Id, Name, Description (from fetch), Due, ParentId}` |

### Validation rules (client-side, before any RPC)

| Rule | Applies to | Requirement | On failure |
|---|---|---|---|
| `<id>` and `--parent` parse as integers | `rm`, `mod`, `add`/`mod` `--parent` | FR-013 | Usage error, exit 1, no RPC |
| `--due` parses as RFC 3339 date-time or bare `YYYY-MM-DD` date | `add`, `mod` | FR-013, clarified `--due` formats | Usage error naming both forms, exit 1, no RPC |
| `name` positional is present | `add`, `mod` | FR-006, FR-010 | Usage error, exit 1, no RPC |
| A recognized `task` subcommand is given | all | FR-001, FR-017 | Usage help, exit 1 |

Existence of a task or `--parent`, name length, and parent-cycle prevention are
**not** checked client-side — they are enforced by the backend and surfaced via
error mapping (see `research.md` Decision 4).

### `mod` field-merge semantics

For `mod`, the `UpdateTaskRequest` is assembled as:

| Field | Value sent |
|---|---|
| `Id` | the positional `<id>` |
| `Name` | the positional `name` (always supplied) |
| `Description` | copied from the `GetTask` result (preserved unchanged) |
| `Due` | the `--due` flag value if supplied, otherwise the fetched `due` |
| `ParentId` | the `--parent` flag value if supplied, otherwise the fetched `parent_id` |

This guarantees FR-012: only fields named on the command line change. Clearing
`due` or detaching a parent is intentionally not possible in this version
(spec Assumptions).
