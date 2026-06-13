# Design: Goal Status Updates

**Date**: 2026-06-12
**Feature**: `051-goal-status-updates`
**Spec**: `specs/051-goal-status-updates/spec.md`

Status updates let a user record the latest state of a goal as a timestamped,
potentially long markdown note and read it back later. A goal accumulates a
history of updates; the newest is the "latest status". TUI only for this
feature (no CLI/web). This document captures the UI and architecture decisions
reached during brainstorming, before implementation planning.

## Decisions reached (brainstorming)

- **Reading model**: the goal detail pane shows a "Latest status" section; a
  dedicated, scrollable **status-history view** (opened with `s`) shows the
  full history.
- **History view shape**: a master list (timestamp + first line, newest first)
  with a cursor; `enter` opens a **full reader** for the selected update
  (scrollable, markdown-rendered).
- **Adding**: quick-add with `S` from the goal detail (opens `$EDITOR`); also
  `n` inside the history view. Compose and edit both use `$EDITOR` (temp `.md`),
  consistent with how descriptions are edited today.
- **API shape**: embed a read-only latest update in the `Goal` message; load
  full history lazily when the history view opens.

## Architecture / layers

Standard twig flow: `TUI (cmd/twig) → ConnectRPC → handler/ → db/ → PostgreSQL`.
Modules touched the usual way:
- `api/` — proto + regenerated stubs (`make proto`)
- `services/twig/` — migration, sqlc queries, handler
- root — TUI

No CLI or web surface (per spec scope).

## Data model

New table `goal_status_updates`:

| Column       | Type        | Notes                                                  |
|--------------|-------------|--------------------------------------------------------|
| `id`         | bigserial   | PK                                                     |
| `goal_id`    | bigint      | NOT NULL, FK → `goals(id)` **ON DELETE CASCADE**       |
| `body`       | text        | NOT NULL; validated non-empty (trimmed)               |
| `created_at` | timestamptz | NOT NULL DEFAULT now()                                 |

- Index on `(goal_id, created_at DESC, id DESC)` for newest-first listing.
- User scoping inherited via the goal: queries filter/join on the owning user,
  matching existing goal queries. Cascade delete satisfies FR-012.

## API contract (extends `GoalService`)

- `Goal` gains a **read-only** field `StatusUpdate latest_status_update`,
  populated by `ListGoals` and `GetGoal`. `UpdateGoal` ignores it (full-replace
  semantics for name/description/due are unchanged).
- New message:
  `StatusUpdate { int64 id; int64 goal_id; string body; google.protobuf.Timestamp created_at; }`
- New RPCs:
  - `ListGoalStatusUpdates(goal_id) → [StatusUpdate]` — newest-first
  - `AddGoalStatusUpdate(goal_id, body) → StatusUpdate` — empty/whitespace body → `InvalidArgument`
  - `UpdateGoalStatusUpdate(id, body) → StatusUpdate` — edits text; `created_at` unchanged; same empty-body validation
  - `DeleteGoalStatusUpdate(id) → {}`
- All scoped to the caller; referencing another user's goal/update → `NotFound`.

## TUI components & flow

**Detail pane** (`renderGoalDetail`): add a "Latest status" section after the
description — relative timestamp (e.g. "2d ago") + newest update rendered with
`m.md.Render`, plus a dim hint `[s] N updates`. Empty state when none: a playful
line (Principle IV), e.g. *"No status yet — how's it going?"*

**Status-history view** (new mode + render fn; replaces the two panes like other
full views):
- Master list: `timestamp · first line` (markdown-inline), newest first, cursor
  (up/down).
- `enter` → **full reader**: scrollable, markdown-rendered view of the selected
  update; `esc` returns to the list. The reader introduces the scrolling logic
  (viewport pattern).
- `n` → compose new (`$EDITOR`); `e` → edit selected (`$EDITOR` prefilled);
  `d` → delete selected (y/n confirm, warm copy); `esc` → back to goal detail.

**Quick-add**: `S` in the goal detail opens `$EDITOR` directly; on save,
`AddGoalStatusUpdate`, refresh, stay on the goal (new update becomes latest).

**Compose mechanism**: reuse `openEditorCmd` (temp `.md` → `$EDITOR`). An
empty/whitespace result is discarded with a notice; no empty update is created.

## Key bindings (Goals tab)

- `s` = open status history
- `S` = add status update

New `KeyMap` bindings + help entries. The history list and reader get their own
mode key handlers, so `e` / `enter` / `d` mean update-actions there rather than
goal-actions.

## Error / empty / edge handling

- Empty body rejected at client (discard) and server (`InvalidArgument`).
- Goal with no updates → friendly empty state; deleting the only update returns
  to that state.
- Long body (5000+ chars) → reader scrolls/wraps; the detail-pane "latest" shows
  within the pane's budget and relies on the `[s]` hint for full reading.
- Deleting a goal cascades to its updates (DB-level).

## Testing

- **Handler tests** (no DB; `export_test.go` shims): add/list/update/delete,
  empty-body rejection, cross-user `NotFound`, latest-update population in
  `ListGoals`/`GetGoal`.
- **TUI tests** (table-style like `goal_view_test.go`): detail-pane latest
  rendering + empty state, history list ordering, reader scrolling, key-handler
  mode transitions, `$EDITOR` save → add flow (mocked).
- **Migration** up/down.
