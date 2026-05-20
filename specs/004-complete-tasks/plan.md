# Implementation Plan: Complete Tasks

**Branch**: `004-complete-tasks` | **Date**: 2026-05-20 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/004-complete-tasks/spec.md`

## Summary

Add per-task completion to the existing `services/todo` backend and CLI. The `tasks` table gains a single nullable `completed_at TIMESTAMPTZ` column — `NULL` means incomplete, a value means complete at that moment. The `task.proto` contract gains a `completed_at` field on `Task` plus a new `CompleteTask` RPC; `ListTasks` is unchanged (the CLI applies completion filters client-side). The handler enforces the invariant "a completed task has no incomplete descendants" in three places: `CompleteTask` rejects when any descendant is incomplete (recursive CTE), `CreateTask` rejects a new subtask under a complete parent, and `UpdateTask` rejects re-parenting under a complete parent. The CLI adds `todo task complete <id>` and two mutually exclusive filter flags on `todo task list` (`--completed`, `--all`); the renderer prefixes every line with `[ ]` or `[x]` and appends the completion timestamp on completed rows. Default list view prunes any subtree whose tasks are all complete; an incomplete parent stays visible even if some of its children are complete.

## Technical Context

**Language/Version**: Go 1.25 (existing `services/todo` module — no new module)

**Primary Dependencies**: Existing only — Connect-Go (`connectrpc.com/connect` v1.19.2), `pgx/v5`, `golang-migrate/v4`, `sqlc` v1.31.1, generated `taskv1`/`taskv1connect`. No new dependencies.

**Storage**: PostgreSQL via `pgx/v5`; schema managed by `golang-migrate` file migrations. New migration `000004_complete` adds `tasks.completed_at TIMESTAMPTZ NULL`. `db/queries/task.sql` modified — `CreateTask`/`UpdateTask` returning columns updated, new `CompleteTask`, `HasIncompleteDescendants`, and `GetTaskCompletion` queries added.

**Testing**: `go test` — table-driven handler tests against a test database covering: `CompleteTask` happy path, idempotent re-complete (same timestamp returned), not-found, blocked-by-incomplete-descendants (one level deep, multi-level deep). `CreateTask`/`UpdateTask` extended to assert rejection when the parent is complete. CLI tests for `runComplete`, the two filter flags (and the mutually-exclusive error), and the renderer's checkbox / timestamp output and tree-pruning rule.

**Target Platform**: Same single Linux server binary (`cmd/server`) plus the `todo` CLI binary (`cmd/todo`), already established by features 001–003.

**Project Type**: Web service (Connect-Go backend) plus its companion CLI — both already exist in one Go module; this feature extends both.

**Performance Goals**: Per-request overhead negligible. The `HasIncompleteDescendants` check is a single recursive CTE keyed on `(user_id, parent_id)` and bounded by the depth of one user's task tree (small — single-user instance scale).

**Constraints**: The completion invariant — *a completed task has no incomplete descendants* — must hold at all times. It is enforced server-side at every mutation that could break it (`CompleteTask`, `CreateTask`, `UpdateTask`); the CLI is not trusted to preserve it. Completion timestamps are recorded in UTC (RFC 3339 on the wire), matching the existing due-date convention. Idempotent re-completion returns the original timestamp unchanged. CLI exit codes follow the established convention: `0` on success (including idempotent re-complete), `1` on any error.

**Scale/Scope**: Continues feature 003's posture — one to two users per instance, small task trees per user. The recursive CTE for descendants is comfortably within that envelope.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I — Simplicity / YAGNI

| Check | Status |
|---|---|
| Simplest solution that satisfies requirements | PASS — one nullable column, one new RPC, three small handler additions, one new CLI subcommand, two new flags on `list`. No new package, no new dependency, no abstraction layer. The completion state is `completed_at IS NULL`, not a separate enum or status table. |
| No speculative abstractions | PASS — no "task-status enum", no event log, no "reopen" hook, no policy registry for who can complete what. The handler validation lives as plain functions in `handler/task.go` alongside the existing `parentChainContains` helper. |
| Reuse over invention | PASS — existing `sqlc` generation, existing `pgx` types (`pgtype.Timestamptz`), existing Connect error mapping, existing CLI tree renderer extended in place. |
| Complexity justified in tracking table | PASS — no violations; Complexity Tracking is empty. The recursive CTE is the standard SQL pattern for descendant checks and is simpler than fetching the whole tree into Go. |

### Principle II — API-First Design

| Check | Status |
|---|---|
| Data contract defined before implementation | PASS — `contracts/task-completion.md` is authored in Phase 1, before any code change. It specifies the new `completed_at` field on `Task`, the new `CompleteTask` RPC, and the unchanged surface of `ListTasks`. |
| Contract committed under `specs/004-complete-tasks/contracts/` | PASS — `contracts/task-completion.md` is committed with the feature branch. The proto edit lands in the implementation phase, matching the contract verbatim. |
| Implementation conforms to the contract | PASS — handler tests assert the new RPC's request/response shape; existing `task.proto` history under `specs/001-task-crud-api/contracts/` is amended only via the diff documented in this feature's contracts file. |

**Result**: PASS (initial). Re-evaluated after Phase 1 design — see "Post-Design Constitution Re-Check" below.

## Project Structure

### Documentation (this feature)

```text
specs/004-complete-tasks/
├── plan.md                       # This file
├── research.md                   # Phase 0 output
├── data-model.md                 # Phase 1 output
├── quickstart.md                 # Phase 1 output
├── contracts/
│   └── task-completion.md        # Phase 1 output — proto delta and RPC contract
├── checklists/
│   └── requirements.md           # Spec quality checklist (from /speckit-specify)
└── tasks.md                      # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/todo/
├── proto/task/v1/
│   └── task.proto                # MODIFIED — add Task.completed_at; add CompleteTask RPC
├── gen/task/v1/
│   ├── task.pb.go                # REGENERATED by buf
│   └── taskv1connect/...         # REGENERATED by buf
├── db/
│   ├── migrations/
│   │   ├── 000004_complete.up.sql   # NEW — ALTER TABLE tasks ADD completed_at
│   │   └── 000004_complete.down.sql # NEW — reverse
│   └── queries/
│       └── task.sql              # MODIFIED — RETURNING includes completed_at;
│                                 #   new CompleteTask, HasIncompleteDescendants,
│                                 #   GetParentCompletion queries
├── internal/
│   ├── db/
│   │   ├── models.go             # REGENERATED — Task gains CompletedAt
│   │   └── task.sql.go           # REGENERATED — sqlc bindings for new queries
│   └── handler/
│       ├── task.go               # MODIFIED — dbTaskToProto includes completed_at;
│       │                         #   CompleteTask method; CreateTask/UpdateTask
│       │                         #   reject when parent is complete
│       └── task_test.go          # MODIFIED — new RPC tests + invariant tests
└── internal/cli/
    ├── task.go                   # MODIFIED — runComplete; list flag parsing
    ├── render.go                 # MODIFIED — checkbox prefix; completion timestamp;
    │                             #   default-mode subtree pruning
    ├── task_test.go              # MODIFIED — runComplete tests; filter-flag tests
    └── render_test.go            # MODIFIED — checkbox / timestamp / pruning tests
```

**Structure Decision**: The feature extends the existing single `services/todo` module in place — no new package, no new module. Every change is additive within the file each touches; the only structural change is one new migration file pair and one new contract document. Completion is a property of an existing entity (Task), so it lives on the existing table and message, not in a parallel "completion" subsystem.

## Complexity Tracking

> No Constitution Check violations. No entries required.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| _(none)_  | —          | —                                    |

## Post-Design Constitution Re-Check

Re-evaluated after Phase 1 artifacts (`research.md`, `data-model.md`, `contracts/task-completion.md`, `quickstart.md`) were produced:

- **Principle I** — Phase 1 confirmed: one nullable column, one new RPC, three short handler additions, two CLI flags, and a renderer that gains six lines for the checkbox and timestamp. No new dependency, no new package, no shared abstraction introduced. The recursive CTE is the only SQL of any sophistication and is documented inline in `data-model.md`. **PASS.**
- **Principle II** — `contracts/task-completion.md` records the exact proto delta and the `CompleteTask` request/response shape; the implementation will edit `proto/task/v1/task.proto` to match. `ListTasks` keeps its existing wire contract — filtering is a client-side render concern, so no server-side filter argument is added (avoids speculative API surface). **PASS.**

No violations introduced by the design. Ready for `/speckit-tasks`.

## Phase 0 — Research

See [research.md](./research.md). All technical decisions are resolved; no `NEEDS CLARIFICATION` items remain. Research records the four notable implementation-level choices: schema representation (nullable column vs. status enum), the descendant-completion check (recursive CTE vs. application-level walk), where list filtering lives (client-side vs. server-side proto field), and the renderer's default-mode tree-pruning rule.

## Phase 1 — Design & Contracts

- [data-model.md](./data-model.md) — the `tasks.completed_at` addition, the completion invariant, and the lifecycle of a task with respect to completion.
- [contracts/task-completion.md](./contracts/task-completion.md) — the proto delta (`Task.completed_at` field, new `CompleteTask` RPC) and its request/response/error contract.
- [quickstart.md](./quickstart.md) — how to apply the migration, complete a task end-to-end from the CLI, and observe each list-filter mode.
- Agent context: the `<!-- SPECKIT START -->` block in `CLAUDE.md` is updated to point to this plan.

## Phase 2 — Next Step

Run `/speckit-tasks` to generate `tasks.md`, the dependency-ordered implementation task list.
