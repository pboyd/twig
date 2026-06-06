# Implementation Plan: Snooze a Task

**Branch**: `040-snooze-task` | **Date**: 2026-06-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/040-snooze-task/spec.md`

## Summary

Add an optional **snooze-until day** to a task so it can be hidden from the working ("pending") view until a future calendar day, then reappears automatically. The TUI's add/edit forms gain a snooze field; the TUI's `c` filter broadens from a completed-only toggle to a two-state **"Show only pending" ↔ "Show all"** toggle that also reveals snoozed tasks (marked with 💤). The CLI and web clients gain no snooze input but honor the snooze by hiding future-snoozed tasks from their default views.

**Technical approach** (per clarifications): snooze is one nullable timestamp column on `tasks`, surfaced as a `snooze_until` field on the `Task` proto and on `CreateTaskRequest`/`UpdateTaskRequest` — reusing the exact pattern already used by `due`. The value encodes a calendar day as midnight UTC (same as `ParseDue`'s bare-date handling). **All filtering is client-side**: `ListTasks` is unchanged and continues to return every task; each client decides visibility by comparing the snooze day against its own local current day. This mirrors how completed-task filtering already works and keeps the API surface minimal.

## Technical Context

**Language/Version**: Go 1.x (CLI/TUI + server), TypeScript / React 19 (web)

**Primary Dependencies**: ConnectRPC + protobuf (`buf`), bubbletea/bubbles/lipgloss v2 (TUI), pgx/v5 + sqlc + golang-migrate (server), Vite + connect-es (web)

**Storage**: PostgreSQL — one new nullable column `snooze_until TIMESTAMPTZ` on `tasks`

**Testing**: `go test ./...` (root + `services/twig`), `npm test` (web). No DB required for handler/CLI/TUI tests (existing convention).

**Target Platform**: Linux/macOS terminal (CLI/TUI), Linux server (HTTP/2 :8080), modern browsers (web SPA)

**Project Type**: Multi-module full-stack (Go CLI/TUI + Go server + React SPA), all sharing the `api/` proto module

**Performance Goals**: No new performance-sensitive paths; snooze filtering is O(n) over the already-fetched task list, identical in cost to existing completed filtering.

**Constraints**: No new API request parameters (client-side filtering); `UpdateTask` full-replace semantics must clear snooze when the field is omitted; snooze comparison must honor the user's **local** day, not the server's.

**Scale/Scope**: Personal task lists (tens–hundreds of tasks). Surfaces touched: proto, DB migration + queries, one handler mapping, TUI form/tree/keymap/view, CLI list filter, web filter/row.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses the existing `due` column/field/parse pattern; one nullable column, one proto field per message; no new abstractions, services, or API params. |
| II. API-First Design | ✅ | Proto contract (`snooze_until` on `Task`, `CreateTaskRequest`, `UpdateTaskRequest`) is defined in `contracts/` and committed before implementation; both Go modules and the web client regenerate from it. |
| III. UI/UX Consistency | ✅ | Snooze field reuses the form's existing field/label styling and date-input convention; 💤 indicator and the relabeled filter use the shared theme; `c` keybinding unchanged. Web/CLI deliberately add no snooze input (consistent: only the TUI authors tasks via forms). |
| IV. Playful User Messages | ✅ | New copy (filter labels, help text, snoozed/empty-state messaging) reviewed for warm tone; web reuses/extends the existing playful `allCompletedHidden` style. |

No violations — Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/040-snooze-task/
├── plan.md              # This file
├── research.md          # Phase 0 — decisions (storage type, timezone, filter location)
├── data-model.md        # Phase 1 — Task entity delta + filter rules
├── quickstart.md        # Phase 1 — how to exercise the feature end-to-end
├── contracts/
│   └── task-api.md      # Phase 1 — proto additions + semantics (API-first)
├── checklists/
│   └── requirements.md  # Spec quality checklist (from /speckit-specify)
└── tasks.md             # Phase 2 — /speckit-tasks output (NOT created here)
```

### Source Code (repository root)

Existing multi-module layout; this feature edits the following real paths:

```text
api/
├── proto/task/v1/task.proto          # + snooze_until on Task, CreateTaskRequest, UpdateTaskRequest
└── gen/...                            # regenerated via `make proto` (do not hand-edit)

services/twig/
├── db/migrations/000009_task_snooze.{up,down}.sql   # + snooze_until column
├── db/queries/task.sql                # CreateTask / UpdateTask include snooze_until
├── internal/db/...                    # regenerated via `sqlc generate` (do not hand-edit)
└── internal/handler/task.go           # dbTaskToProto + Create/UpdateTask map snooze_until

internal/
├── cli/
│   ├── render.go                      # snooze-aware default prune (hide future-snoozed)
│   └── task.go                        # default list view hides snoozed; --all shows them
└── tui/
    ├── edit.go                        # + Snooze form field (focus index, prefill, save msg)
    ├── update.go                      # create/update cmds parse+send snooze; toggle relabel
    ├── tree.go                        # emitNode hides future-snoozed subtree unless "show all"
    ├── keymap.go                      # help text "toggle completed" → broadened wording
    ├── model.go                       # showCompleted → "show all" semantics (rename or repurpose)
    └── view.go / details.go           # 💤 indicator on snoozed rows

services/twig-web/
├── src/gen/...                        # regenerated via `npm run gen` (do not hand-edit)
├── src/lib/filterTree.ts              # hide future-snoozed when not "show all"
├── src/components/TreeRow.tsx         # 💤 indicator on snoozed rows
├── src/pages/TaskTreePage.tsx         # label/empty-state copy (broadened if desired)
└── src/theme/messages.ts             # snooze-aware empty-state copy if needed
```

**Structure Decision**: No structural change. The feature threads a single new field through the existing proto → handler → db pipeline and adds client-side filter logic at each of the three presentation surfaces. The `api/` proto module remains the single source of truth shared by all three.

## Phase 0 — Research

See [research.md](./research.md). Key decisions, all resolved (no open NEEDS CLARIFICATION):

1. **Storage type** — reuse `due`'s `TIMESTAMPTZ` column + proto `Timestamp`, with the value pinned to midnight UTC of the chosen day (matching `ParseDue`). Rejected a dedicated `DATE` column (no proto date type; diverges from `due`; more code for no benefit).
2. **Filter location** — client-side in all three clients (clarified). `ListTasks` unchanged. Rejected server-side filtering (would prevent the TUI "Show all" from revealing snoozed tasks without a new request param, and would force the server to know the user's timezone).
3. **Timezone / "day arrives"** — a task is snoozed iff the UTC calendar date of `snooze_until` is **strictly after** the client's local calendar date. This yields local-day semantics (wakes at the user's local start-of-day) while keeping encode/decode in UTC.
4. **Un-snooze & past dates** — clearing the field (or a date ≤ today) un-snoozes. `UpdateTask` full-replace already clears omitted fields, so editing the form with an empty snooze field clears it for free.

## Phase 1 — Design & Contracts

- **Contract**: [contracts/task-api.md](./contracts/task-api.md) — the proto additions and their semantics (API-first; committed before implementation per Principle II).
- **Data model**: [data-model.md](./data-model.md) — the `Task` entity delta and the precise client-side visibility predicate shared (in spirit) by all three clients.
- **Quickstart**: [quickstart.md](./quickstart.md) — end-to-end manual verification across TUI, CLI, and web.
- **Agent context**: `CLAUDE.md` SPECKIT block updated to point at this plan.

### Post-Design Constitution Re-Check

All four principles remain ✅ after design. The contract is defined before implementation (II); the design adds no abstractions (I); presentation reuses shared theme/conventions and the existing `c` binding (III); new copy is slated for tone review (IV).

## Complexity Tracking

No Constitution violations; no entries required.
