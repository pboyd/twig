# Implementation Plan: Task CRUD API

**Branch**: `001-task-crud-api` | **Date**: 2026-05-16 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-task-crud-api/spec.md`

## Summary

Add task management to the `services/todo` backend: a `TaskService` exposing Create, Get, List, Update, and Delete RPCs over Connect-Go, backed by a new `tasks` table in PostgreSQL. Tasks carry a name (required), optional description and due date, and an optional `parent_id` reference to another task — forming a hierarchy of unrestricted depth. The PostgreSQL self-referencing foreign key with `ON DELETE CASCADE` delivers both parent-existence enforcement and cascade deletion; a recursive CTE checks for cycles on re-parenting. The feature follows the existing `HealthService` pattern exactly: proto → buf → `gen/`, SQL → sqlc → `internal/db/`, a handler holding `*db.Queries` directly with no service layer.

## Technical Context

**Language/Version**: Go 1.25

**Primary Dependencies**: Connect-Go (`connectrpc.com/connect` v1.19.2), pgx/v5 (`github.com/jackc/pgx/v5`), golang-migrate v4, Buf (remote plugins: `protocolbuffers/go`, `connectrpc/go`), sqlc v1.31.1

**Storage**: PostgreSQL 17 (one new table, `tasks`; migration `000002_tasks`)

**Testing**: `go test` — table-driven unit tests for handler validation; integration tests exercising the handlers against a real PostgreSQL instance

**Target Platform**: Linux server (distroless container), HTTP/2 h2c on `:8080`

**Project Type**: Single backend web service (`services/todo`)

**Performance Goals**: All five RPCs return within 1 second under single-user load (spec SC-005)

**Constraints**: No authentication/authorization (single trusted caller); flat list response (no pagination/filtering/sorting); hard delete only

**Scale/Scope**: Single-user todo; one service, one new table, five RPCs, ~hundreds of tasks expected

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I — Simplicity / YAGNI

| Check | Status |
|---|---|
| Simplest solution that satisfies requirements | PASS — handler holds `*db.Queries` directly; no service layer, no repository pattern, no domain model, matching the existing `HealthService`. |
| No speculative abstractions | PASS — five sqlc queries plus two helper queries (existence, cycle check); nothing generalized for hypothetical needs. |
| Reuse over invention | PASS — parent existence (FR-014) and cascade delete (FR-017) are enforced by one self-referencing FK with `ON DELETE CASCADE` rather than hand-written application logic. |
| Complexity justified in tracking table | PASS — no violations; Complexity Tracking is empty. |

### Principle II — API-First Design

| Check | Status |
|---|---|
| Data contract defined before implementation | PASS — `task.proto` is authored and committed to `contracts/` in Phase 1, before any handler code. |
| Contract committed under `specs/001-task-crud-api/contracts/` | PASS — see Phase 1 output. |
| Implementation conforms to the contract | PASS — handler types are generated from the proto by Buf; the contract is the source of truth. |

**Result**: PASS (initial). Re-evaluated after Phase 1 design — see "Post-Design Constitution Re-Check" below.

## Project Structure

### Documentation (this feature)

```text
specs/001-task-crud-api/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── task.proto       # Phase 1 output — API contract
├── checklists/
│   └── requirements.md  # Spec quality checklist (from /speckit-specify)
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/todo/
├── proto/task/v1/
│   └── task.proto                  # NEW — copied from contracts/task.proto
├── gen/task/v1/                    # NEW — buf-generated (committed)
│   ├── task.pb.go
│   └── taskv1connect/task.connect.go
├── db/
│   ├── migrations/
│   │   ├── 000002_tasks.up.sql     # NEW — create tasks table
│   │   └── 000002_tasks.down.sql   # NEW — drop tasks table
│   └── queries/
│       └── task.sql                # NEW — sqlc queries for tasks
├── internal/
│   ├── db/                         # sqlc regenerates: models.go gains Task;
│   │   ├── models.go               #   task.sql.go is added
│   │   └── task.sql.go             # NEW (generated)
│   └── handler/
│       ├── health.go               # unchanged
│       ├── task.go                 # NEW — TaskService handler
│       └── task_test.go            # NEW — handler tests
└── cmd/server/
    └── main.go                     # MODIFIED — register TaskService handler
```

**Structure Decision**: Single Go service. No new top-level directories — every new file slots into the existing `services/todo` layout established by the backend-foundation feature. The `health` slices (`proto/health`, `gen/health`, `internal/handler/health.go`) are the working template; the `task` slices mirror them one-for-one.

## Complexity Tracking

> No Constitution Check violations. No entries required.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _(none)_  | —          | —                                   |

## Post-Design Constitution Re-Check

Re-evaluated after Phase 1 artifacts (`research.md`, `data-model.md`, `contracts/task.proto`, `quickstart.md`) were produced:

- **Principle I** — The design adds exactly one table, one proto file, two migration files, one query file, and one handler file. The only non-CRUD logic is a single recursive-CTE query for cycle detection, which is the minimum needed to satisfy FR-016; no alternative is simpler while still correct. No service/repository layers introduced. **PASS.**
- **Principle II** — `contracts/task.proto` fully defines the request/response schema for all five RPCs and is committed before implementation. **PASS.**

No violations introduced by the design. Ready for `/speckit-tasks`.

## Phase 0 — Research

See [research.md](./research.md). All technical decisions are resolved; no `NEEDS CLARIFICATION` items remain.

## Phase 1 — Design & Contracts

- [data-model.md](./data-model.md) — `tasks` table schema, field types, constraints, the `000002` migration SQL, and the sqlc query set.
- [contracts/task.proto](./contracts/task.proto) — `TaskService` contract: five RPCs and their message types.
- [quickstart.md](./quickstart.md) — how to generate code, run migrations, start the stack, and exercise each RPC.
- Agent context: the `<!-- SPECKIT START -->` block in `CLAUDE.md` is updated to point to this plan.

## Phase 2 — Next Step

Run `/speckit-tasks` to generate `tasks.md`, the dependency-ordered implementation task list.
