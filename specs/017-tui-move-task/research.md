# Research: Change Task Parent in TUI

All Technical Context items resolved without NEEDS CLARIFICATION. The spec's clarifications already pinned the open UX questions (display = hierarchical tree, scope = all incomplete tasks, keys = Enter/Esc). This document records the small implementation-level decisions that drove the plan.

## Decision 1: Use the existing `UpdateTask` RPC; no proto changes

**Decision**: Call `taskv1.UpdateTaskRequest` with the task's current `name`, `description`, `due`, plus the new `parent_id` (unset for top-level).

**Rationale**: `UpdateTask` already supports reparenting and the handler (`services/todo/internal/handler/task.go:222-246`) already validates parent existence, rejects cycles, and rejects completed parents. Adding a separate `MoveTask` RPC would violate Principle I (Simplicity/YAGNI).

**Alternatives considered**:

- New `MoveTask(id, parent_id)` RPC: rejected — duplicates existing capability.
- Direct DB update: not applicable; the TUI is a client.

## Decision 2: Modal view mode (`modeMove`) following the existing edit/help pattern

**Decision**: Add `modeMove` to the `viewMode` enum in `model.go`. Routing in `update.go` dispatches key events to a new `move.go` handler when in this mode; `view.go` renders a centered overlay when in this mode.

**Rationale**: Matches how `modeEdit`, `modeHelp`, and `modePomodoro` already work — no new abstraction, just one more mode.

**Alternatives considered**:

- Inline editing on the main list (e.g., a status-line prompt): rejected — the spec calls for a dialog and a hierarchical browsable tree, which doesn't fit a one-line prompt.

## Decision 3: Reuse `tree.go`'s tree builder for the candidate list

**Decision**: Build the candidate tree from the already-loaded incomplete tasks using the same indentation/ordering logic the main view uses. Exclude (a) the task being moved and (b) its descendants from the candidate set; this both prevents the user from selecting a cycle-creating option and means the server's cycle check is a defense-in-depth backstop.

**Rationale**: Keeps the dialog visually consistent with the main TUI per the clarification, and avoids reimplementing tree layout.

**Alternatives considered**:

- Flat list: rejected by clarification Q1.
- Filter to currently visible tasks: rejected by clarification Q2 (scope is all incomplete tasks).

## Decision 4: Pre-selection rules

**Decision**: On open, set the dialog's cursor to the row for the task's current `parent_id`. If `parent_id` is unset, cursor lands on the "no parent" entry, which is rendered as the first row of the dialog.

**Rationale**: FR-003 and Story 2 AC2 require this; placing "no parent" at the top keeps it easy to find and gives a sensible "home" position.

## Decision 5: Error surfacing

**Decision**: If `UpdateTask` returns an error (e.g., the server rejects the change), keep the dialog open, preserve the current cursor position, and render the error message in a status line at the bottom of the dialog. The user can adjust their selection and retry or press Esc.

**Rationale**: FR-007 and the edge case "save fails" both call for in-dialog error surfacing without losing selection state.

## No outstanding NEEDS CLARIFICATION

All Technical Context fields are populated with concrete values. No items deferred.
