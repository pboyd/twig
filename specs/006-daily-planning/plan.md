# Implementation Plan: Daily Planning

**Branch**: `006-daily-planning` | **Date**: 2026-05-21 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/006-daily-planning/spec.md`

## Summary

Extend the existing `services/todo` backend and `todo` CLI with a daily plan. A new `plan_entries` table stores entries scoped to `(user_id, day)`: each row carries a per-day sequential `id`, an optional `task_id` foreign key (ON DELETE CASCADE, mirroring `pomodoros`), an optional `name` (which falls back to the linked task's name on display), and the time window expressed as `start_minute SMALLINT` (0–1439) plus `duration_minute SMALLINT` (1–1440) on `day DATE`. Times are stored as time-of-day on a specific date — never as `TIMESTAMPTZ` — so the "user's local timezone" requirement is satisfied without server-side TZ math, and the "no entry crosses midnight" rule is enforced with a CHECK constraint (`start_minute + duration_minute <= 1440`). Per-day overlap is enforced inside the handler in a single transaction (`SELECT ... FOR UPDATE` then INSERT/UPDATE), using half-open `[start, start+duration)` semantics so touching at the boundary is allowed (clarification 2026-05-21). The same transaction allocates the next `id` as `MAX(id)+1` for the user/day; gaps from removals are preserved. A new proto package `plan.v1` defines a `PlanService` with `AddPlanTask`, `AddPlanEvent`, `RemovePlanEntry`, `RenamePlanEntry`, `MovePlanEntry`, `ClearPlan`, and `ListPlanEntries` — one RPC per CLI subcommand plus the read needed for display. The CLI gains a `todo plan` command group with subcommands `task`, `event`, `rm`, `rename`, `mv`, `clear`, and a default (no-subcommand) render that prints an ASCII day-planner grid from the first entry's start to the last entry's end. Start-time and duration parsing (`13:15` / `1:15pm` / `01:15 PM`; `90m` / `1h30m` / `2h`) live in a new `internal/timeparse` helper package consumed only by the CLI; the server speaks integer minutes.

## Technical Context

**Language/Version**: Go 1.25 (existing `services/todo` module — no new module).

**Primary Dependencies**: Existing only — Connect-Go (`connectrpc.com/connect`), `pgx/v5`, `golang-migrate/v4`, `sqlc`, buf-generated protobuf bindings. The CLI's ASCII grid renderer uses the standard library only; no TUI framework, no terminal-control libraries. No new dependencies.

**Storage**: PostgreSQL via `pgx/v5`. New migration `000006_plan_entries`:
  - `CREATE TABLE plan_entries (`
    - `user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,`
    - `day DATE NOT NULL,`
    - `id INT NOT NULL,`
    - `task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE,`
    - `name VARCHAR(255),`
    - `start_minute SMALLINT NOT NULL CHECK (start_minute BETWEEN 0 AND 1439),`
    - `duration_minute SMALLINT NOT NULL CHECK (duration_minute > 0 AND duration_minute <= 1440),`
    - `CHECK (start_minute + duration_minute <= 1440),`
    - `CHECK (task_id IS NOT NULL OR name IS NOT NULL),`
    - `PRIMARY KEY (user_id, day, id)`
    - `);`
  - `CREATE INDEX plan_entries_task_id_idx ON plan_entries (task_id);`

  Overlap is **not** enforced by a DB constraint; the handler enforces it within a `BEGIN ... COMMIT` block on `SERIALIZABLE` isolation (consistent with the pattern from 005, where simplicity outweighed using an exclusion constraint with `btree_gist`).

  New `db/queries/plan.sql` holds: `ListPlanEntriesForDay`, `GetPlanEntry`, `LockPlanEntriesForDay` (`SELECT … FOR UPDATE`), `NextPlanEntryId`, `InsertPlanEntry`, `UpdatePlanEntryName`, `UpdatePlanEntryTime`, `DeletePlanEntry`, `DeletePlanEntriesFromMinute`, `TrimPlanEntryDuration`.

**Testing**: `go test` — table-driven handler tests against the test database covering each RPC's happy path; overlap rejection (including the touching-boundary allowance); per-day sequential id assignment that does not renumber after deletions; cascade-delete from `tasks` removing dependent plan entries; midnight-crossing rejection; rename/mv-by-id NotFound for unknown id; `MovePlanEntry` with no duration preserving the prior duration; `ClearPlan` straddle behaviour (entry that overlaps cutoff is shortened, later entries removed); concurrent `AddPlanTask` race (two goroutines for the same `(user_id, day)`, exactly one succeeds). CLI tests for: subcommand dispatch from `todo plan ...`; `--date` flag override vs. default; time parsing for the documented variants (`13:15`, `1:15pm`, `01:15 PM`); duration parsing for `90m` / `1h30m` / `2h`; the grid renderer (start/end bracketing, gap rows, task-name fallback when `name` is null); empty-plan message.

**Target Platform**: Same single Linux server binary (`cmd/server`) plus the `todo` CLI binary (`cmd/todo`), already established by features 001–005. No interactive terminal modes are introduced — the grid renderer simply writes lines to stdout.

**Project Type**: Web service (Connect-Go backend) plus its companion CLI — both already exist in one Go module; this feature extends both.

**Performance Goals**: All RPCs return in under 1 second locally; per SC-001 a user can compose a 5-entry plan in under 2 minutes. With at most a few dozen rows per `(user_id, day)`, the unindexed scans inside the transaction's `FOR UPDATE` are trivially fast.

**Constraints**: Times are stored as `(day DATE, start_minute SMALLINT, duration_minute SMALLINT)` — never as `TIMESTAMPTZ` — because the spec mandates local-timezone semantics and a strict single-day boundary. This pushes timezone interpretation entirely to the CLI side and makes the midnight-crossing rule a static CHECK. Half-open `[start, start+duration)` overlap semantics are codified in the `internal/plan` package's `Overlap(aStart, aDur, bStart, bDur int)` helper, used by the handler and by handler tests. Entry-id allocation and overlap checking happen in the same transaction; without that, a race between two concurrent adds could produce duplicate ids or overlapping windows.

**Scale/Scope**: Continues the one-to-two-users-per-instance posture. A plan is at most ~50 entries on a single day; queries are keyed on `(user_id, day)` (the PK prefix) so growth is bounded.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I — Simplicity / YAGNI

| Check | Status |
|---|---|
| Simplest solution that satisfies requirements | PASS — one new table, one new proto package with seven thin RPCs (one per CLI subcommand plus the read for display), one new CLI command group. No `Plan` aggregate is materialized — a plan is just the rows for a `(user_id, day)`. Time is stored as integer minutes on a date; no timezone math on the server. Overlap is checked in the same transaction that allocates the next id, avoiding both a DB extension (`btree_gist` exclusion constraint) and a second round-trip. |
| No speculative abstractions | PASS — no `PlanRepository` interface, no `Scheduler`, no recurring-event engine, no calendar/iCal export. The grid renderer is one function (`renderGrid([]PlanEntry) string`). The `timeparse` helper exists because three subcommands genuinely share the same start/duration parsing. |
| Reuse over invention | PASS — uses existing `sqlc`, `pgx`, Connect error mapping, auth interceptor, CLI client wiring. The `task_id` FK reuses the same `ON DELETE CASCADE` convention introduced for `pomodoros` in feature 005. |
| Complexity justified in tracking table | PASS — no violations; Complexity Tracking is empty. Storing time-of-day as two `SMALLINT`s rather than a `TIMESTAMPTZ` is a *simplification* (no TZ conversions), not a complexity addition. |

### Principle II — API-First Design

| Check | Status |
|---|---|
| Data contract defined before implementation | PASS — `contracts/plan.md` is authored in Phase 1 below and specifies the new `plan.v1` proto file in full: the `PlanEntry` message, the seven RPCs with request/response shapes, and the error codes each one can return. |
| Contract committed under `specs/006-daily-planning/contracts/` | PASS — `contracts/plan.md` is committed with the feature branch. The actual `proto/plan/v1/plan.proto` file lands in the implementation phase and matches the contract verbatim. |
| Implementation conforms to the contract | PASS — handler tests assert each RPC's request/response shape and error code mapping. The CLI converts `PlanEntry` rows to grid cells; it never invents a parallel local plan model. |

**Result**: PASS (initial). Re-evaluated after Phase 1 design — see "Post-Design Constitution Re-Check" below.

## Project Structure

### Documentation (this feature)

```text
specs/006-daily-planning/
├── plan.md                       # This file
├── research.md                   # Phase 0 output
├── data-model.md                 # Phase 1 output
├── quickstart.md                 # Phase 1 output
├── contracts/
│   └── plan.md                   # Phase 1 output — proto file, RPC contracts
├── checklists/
│   └── requirements.md           # Spec quality checklist (from /speckit-specify)
└── tasks.md                      # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/todo/
├── proto/plan/v1/
│   └── plan.proto                # NEW — PlanService + PlanEntry + seven RPC messages
├── gen/plan/v1/
│   ├── plan.pb.go                # GENERATED by buf
│   └── planv1connect/...         # GENERATED by buf
├── db/
│   ├── migrations/
│   │   ├── 000006_plan_entries.up.sql   # NEW — plan_entries table + task_id index
│   │   └── 000006_plan_entries.down.sql # NEW — reverse
│   └── queries/
│       └── plan.sql              # NEW — Insert / List / Get / Update / Delete / Lock / NextId / Trim
├── internal/
│   ├── plan/
│   │   └── plan.go               # NEW — Overlap(aStart, aDur, bStart, bDur int) bool
│   ├── db/
│   │   ├── models.go             # REGENERATED — PlanEntry model
│   │   └── plan.sql.go           # GENERATED — sqlc bindings for plan queries
│   └── handler/
│       ├── plan.go               # NEW — AddPlanTask, AddPlanEvent, RemovePlanEntry,
│       │                         #   RenamePlanEntry, MovePlanEntry, ClearPlan,
│       │                         #   ListPlanEntries; tx-wraps id allocation +
│       │                         #   overlap check; maps task FK violation -> NotFound
│       └── plan_test.go          # NEW — RPC tests + concurrency + cascade-delete
├── cmd/server/
│   └── main.go                   # MODIFIED — register PlanService on the Connect mux
└── internal/cli/
    ├── cli.go                    # MODIFIED — wire `plan` top-level command group
    ├── plan.go                   # NEW — runPlan dispatcher; runPlanTask, runPlanEvent,
    │                             #   runPlanRm, runPlanRename, runPlanMv, runPlanClear,
    │                             #   runPlanShow (default); --date flag handling
    ├── plan_grid.go              # NEW — renderGrid([]*planv1.PlanEntry) string
    ├── plan_test.go              # NEW — dispatcher + flag tests with fake client
    ├── plan_grid_test.go         # NEW — golden-output grid renderer tests
    └── timeparse/
        ├── timeparse.go          # NEW — ParseStart(s string) (minute int, err error);
        │                         #   ParseDuration(s string) (minute int, err error)
        └── timeparse_test.go     # NEW — table-driven tests for documented variants
```

**Structure Decision**: A new proto package (`plan.v1`) keeps the planning service domain-separated from `task.v1`, mirroring how the codebase already separates concerns by package rather than piling messages into one service. On the server, planning is two new files (`internal/handler/plan.go`, `internal/plan/plan.go`); on the CLI, planning is contained to three new files plus the `timeparse` sub-package. The `timeparse` helper sits under `internal/cli/timeparse` rather than at the module root because the server speaks integer minutes and has no business parsing human time formats — the format is a CLI surface concern.

## Complexity Tracking

> No Constitution Check violations. No entries required.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| _(none)_  | —          | —                                    |

## Post-Design Constitution Re-Check

Re-evaluated after Phase 1 artifacts (`research.md`, `data-model.md`, `contracts/plan.md`, `quickstart.md`) were produced:

- **Principle I** — Phase 1 confirmed: one table, one new handler file with seven RPCs that are thin wrappers around one to two queries each, one CLI command group with a small dispatcher, a 30-line grid renderer, and a 50-line parser. No new dependencies. The `internal/plan` package is a single function plus its test. **PASS.**
- **Principle II** — `contracts/plan.md` records the exact proto file before implementation, and every handler test asserts both the response shape and the Connect error code the CLI relies on. The CLI displays values straight from `ListPlanEntriesResponse` rows; no parallel model. **PASS.**

No violations introduced by the design. Ready for `/speckit-tasks`.

## Phase 0 — Research

See [research.md](./research.md). All technical decisions are resolved; no `NEEDS CLARIFICATION` items remain. Research records the notable implementation-level choices:
1. **Time storage representation** — `(day DATE, start_minute SMALLINT, duration_minute SMALLINT)` vs. `TIMESTAMPTZ` vs. `TIMESTAMP WITHOUT TIME ZONE`.
2. **Overlap enforcement** — handler-level `SERIALIZABLE` transaction vs. Postgres `EXCLUDE` constraint (requires `btree_gist`) vs. application-level mutex.
3. **Per-day id allocation** — `MAX(id)+1` inside the same transaction vs. a per-day sequence vs. a global sequence.
4. **Task FK behaviour on task deletion** — `ON DELETE CASCADE` vs. `ON DELETE SET NULL` vs. no action.
5. **Where time-format parsing lives** — server vs. CLI.
6. **One service vs. extending `TaskService`** — new `plan.v1` package vs. piling RPCs onto `task.v1`.

## Phase 1 — Design & Contracts

- [data-model.md](./data-model.md) — the `plan_entries` table, its constraints (PK on `(user_id, day, id)`, midnight-crossing CHECK, name-or-task CHECK), the half-open overlap semantics, and the lifecycle of an entry (`Inserted → optionally Renamed / Moved → Deleted`).
- [contracts/plan.md](./contracts/plan.md) — the full `proto/plan/v1/plan.proto` file (PlanEntry, seven RPCs) plus each RPC's request/response/error contract.
- [quickstart.md](./quickstart.md) — apply migration, add a task entry from the CLI, add an event, render the plan, move one entry, clear the rest of the day; observe the grid output at each step.
- Agent context: the `<!-- SPECKIT START -->` block in `CLAUDE.md` is updated to point to this plan.

## Phase 2 — Next Step

Run `/speckit-tasks` to generate `tasks.md`, the dependency-ordered implementation task list.
