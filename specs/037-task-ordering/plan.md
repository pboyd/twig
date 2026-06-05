# Implementation Plan: User-Configurable Task Order

**Branch**: `037-task-ordering` | **Date**: 2026-06-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/037-task-ordering/spec.md`

## Summary

Tasks currently render in `id`-ascending order everywhere (server `ListTasks` does `ORDER BY id`; both `cli.BuildTree`/`SortNodes` and the web `buildTree` rely on that). This feature gives each task a user-controlled position **within its sibling group** (same `parent_id`, or the root group for parentless tasks) that persists in PostgreSQL and is therefore shared across every client.

Technical approach:

- Add a `position` column to `tasks`, unique-ordered per `(user_id, parent_id)` group, backfilled to preserve the current id-ascending order.
- Expose `position` on the `task.v1.Task` message and add one new RPC, `ReorderTask`, that moves a task immediately **before** or **after** a named sibling anchor. This single anchor-based contract serves both the TUI single-step move and the web drag-to-arbitrary-position gesture, and is naturally concurrency-safe.
- TUI: bind `{` (rank higher) and `}` (rank lower) in the Tasks list; each computes the visible-sibling anchor (skipping hidden tasks) and calls `ReorderTask`.
- Web: make sibling rows sortable via `@dnd-kit` drag-and-drop, calling `ReorderTask` on drop.
- CLI/server tree builders sort siblings by `position`; the CLI gains no new commands (read-only respect of the order).

## Technical Context

**Language/Version**: Go 1.23 (root CLI/TUI module + `services/twig` server module); TypeScript 5.8 / React 19 (`services/twig-web`)

**Primary Dependencies**: ConnectRPC, pgx/v5, sqlc, golang-migrate (server); bubbletea/lipgloss (TUI); `@connectrpc/connect-query`, `@tanstack/react-query`, and a new `@dnd-kit/core` + `@dnd-kit/sortable` (web)

**Storage**: PostgreSQL (`tasks` table); a new `position` column ordered per `(user_id, parent_id)` group

**Testing**: `go test ./...` (root + server, using `export_test.go` shims, no DB required); `npm test` (vitest) for web

**Target Platform**: Linux server (port 8080); terminal TUI/CLI; modern browsers for the SPA

**Project Type**: Full-stack — three Go modules (CLI/TUI, shared API, server) plus a React SPA

**Performance Goals**: Reorder reflected in under 1s after a TUI keypress or web drop (SC-006); sibling groups are personal-scale (tens of items), so a per-group renumber on reorder is well within budget

**Constraints**: No integration-test DB infra (handler tests use shims); generated code (`api/gen/`, `internal/db/`, web `src/gen/`) is never hand-edited; the SPA must stay same-origin with the API (session cookie)

**Scale/Scope**: Single personal-todo deployment; small sibling groups; three editing surfaces (TUI write, web write, CLI read-only)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One column, one RPC, per-group renumber-on-reorder (no rank-rebalancing/CRDT machinery). No speculative abstraction. `@dnd-kit` is a standard accessible-DnD library, not a home-grown framework. |
| II. API-First Design | ✅ | `position` field + `ReorderTask` defined in `api/proto/task/v1/task.proto` and `contracts/` before any implementation. Contract is the source of truth for server + both write clients. |
| III. UI/UX Consistency | ✅ | TUI bindings added to the shared `KeyMap`/help; reuses existing theme. Web drag affordances use existing `theme/tokens.ts` + components. CLI ordering stays consistent with existing tree rendering. |
| IV. Playful User Messages | ✅ | New user-facing text (reorder error/status copy, web drag aria/empty hints) routed through `src/theme/messages.ts` and TUI status styling with the established warm/playful tone. |

No violations — Complexity Tracking table intentionally left empty.

## Project Structure

### Documentation (this feature)

```text
specs/037-task-ordering/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── reorder-task.md   # ReorderTask RPC + position field contract
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
api/
├── proto/task/v1/task.proto          # ADD: Task.position field; ReorderTask rpc + messages
└── gen/task/v1/...                   # REGEN via `make proto` (do not hand-edit)

services/twig/
├── db/
│   ├── migrations/
│   │   ├── 000008_task_position.up.sql    # ADD column + backfill + index
│   │   └── 000008_task_position.down.sql
│   └── queries/task.sql              # ADD: position to CreateTask; sibling-group read/renumber queries
├── internal/db/...                   # REGEN via `sqlc generate` (do not hand-edit)
└── internal/handler/
    ├── task.go                       # ADD ReorderTask handler; position on dbTaskToProto;
    │                                 #     end-of-group position on Create and on reparent (UpdateTask)
    └── task_test.go                  # ADD reorder + position unit tests

internal/cli/
└── render.go                         # CHANGE: sort siblings by position (was id)

internal/tui/
├── keymap.go                         # ADD RankUp ("{") / RankDown ("}") bindings + help
├── update.go                         # ADD { / } handling in handleListKey; reorderTaskCmd
└── update_test.go / tree_test.go     # ADD ranking behavior tests (incl. hidden-skip, boundaries)

services/twig-web/
├── package.json                      # ADD @dnd-kit/core, @dnd-kit/sortable
├── src/gen/...                       # REGEN via `npm run gen` (do not hand-edit)
├── src/lib/tree.ts                   # CHANGE: sort siblings by position
├── src/lib/reorderAnchor.ts          # ADD: pure fn mapping a drop to before/after anchor (unit-tested)
├── src/components/TreeRow.tsx        # ADD sortable/draggable behavior
├── src/pages/TaskTreePage.tsx        # ADD DnD context + onDragEnd → ReorderTask
└── src/theme/messages.ts             # ADD reorder-related copy
```

**Structure Decision**: This is the existing full-stack layout (three Go modules + React SPA per `CLAUDE.md`). The feature touches the shared proto/contract, the server (migration + handler + queries), the shared CLI tree builder (consumed by both CLI and TUI), the TUI input layer, and the web SPA. No new modules or services are introduced.

## Complexity Tracking

> No Constitution Check violations — nothing to justify.
