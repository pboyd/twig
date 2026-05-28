# Implementation Plan: Change Task Parent in TUI

**Branch**: `017-tui-move-task` | **Date**: 2026-05-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/017-tui-move-task/spec.md`

## Summary

Add an `m` keybinding to the TUI that opens a modal "move task" dialog over the task list. The dialog shows all of the user's incomplete tasks as a hierarchical tree (plus a "no parent" entry at the top), pre-selects the selected task's current parent, and on Enter persists the chosen parent via the existing `UpdateTask` ConnectRPC call. Esc cancels with no change. The server already implements `UpdateTask` with `parent_id` (including cycle and completed-parent rejection), so this feature is purely client-side: a new view mode, a new key binding, a new component file, and rendering of the modal over the existing tree view.

## Technical Context

**Language/Version**: Go 1.22+ (existing module)

**Primary Dependencies**: Bubble Tea (`github.com/charmbracelet/bubbletea`), Bubbles (`github.com/charmbracelet/bubbles`), Lipgloss for styling, ConnectRPC client (existing).

**Storage**: N/A — server-side persistence via existing `UpdateTask` RPC.

**Testing**: `go test ./...` from `services/todo/`; same patterns as existing TUI tests (`update_test.go`, `view_test.go`, `tree_test.go`) using direct model updates rather than real terminal.

**Target Platform**: TTY (Linux/macOS terminals).

**Project Type**: CLI / TUI client (single Go binary at `cmd/todo`).

**Performance Goals**: Dialog open in <50ms (no extra network round-trip — uses already-loaded task list).

**Constraints**: Keyboard-only interaction; no new dependencies; reuse the existing `tree.go` rendering helpers so the dialog tree looks like the main list.

**Scale/Scope**: Up to ~hundreds of incomplete tasks per user (existing list rendering already scales here).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Simplicity / YAGNI**: PASS. No new abstractions — adds one view mode and one component file alongside the existing `edit.go`/`help.go` pattern. Reuses the existing tree builder and the already-supported `UpdateTask` RPC. No new server work, no new contract, no new persistence.
- **II. API-First Design**: PASS. The required contract (`UpdateTask` with `parent_id`) already exists in `proto/task/v1/task.proto`. No contract changes are needed; the feature is a TUI-only consumer of an existing API.

No violations. Complexity Tracking table is unused.

## Project Structure

### Documentation (this feature)

```text
specs/017-tui-move-task/
├── spec.md                  # Feature spec (already created)
├── plan.md                  # This file
├── research.md              # Phase 0 output
├── data-model.md            # Phase 1 output (TUI-side state model only)
├── quickstart.md            # Phase 1 output (manual verification recipe)
├── contracts/
│   └── update-task.md       # Phase 1 output (pointer to existing RPC contract)
└── checklists/
    └── requirements.md      # From /speckit-specify
```

### Source Code (repository root)

```text
services/todo/internal/tui/
├── keymap.go                # ADD: Move binding ("m")
├── model.go                 # ADD: modeMove viewMode constant + move-related state on Model
├── move.go                  # NEW: move dialog component (build candidate list, navigation, confirm/cancel)
├── move_test.go             # NEW: unit tests for tree building, pre-selection, key handling, cycle prevention surfaced via error
├── update.go                # MODIFY: route `m` from list mode → enter modeMove; route key events in modeMove to move.go
├── view.go                  # MODIFY: render move dialog as overlay when mode == modeMove
├── tree.go                  # REUSE: candidate tree builder (extract a small helper if needed for "filter to incomplete + exclude moving task & descendants")
└── help.go                  # MODIFY: include `m` in the help screen
```

No changes outside `services/todo/internal/tui/`. No proto, no handler, no db, no migrations.

**Structure Decision**: Single Go module, TUI-only change. Follows the existing pattern where each modal feature (edit, help, pomodoro) lives in its own file with a small surface integrated into `update.go` and `view.go`.

## Complexity Tracking

Not applicable — no Constitution violations.
