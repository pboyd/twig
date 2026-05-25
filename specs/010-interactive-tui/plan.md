# Implementation Plan: Interactive TUI

**Branch**: `010-interactive-tui` | **Date**: 2026-05-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/010-interactive-tui/spec.md`

## Summary

Launch an interactive terminal UI when `todo` is invoked with no subcommand. The TUI presents a two-pane layout: a hierarchical task tree on the left (with `[+]`/`[-]` markers and the existing CLI's tree-line styling) and a details pane on the right showing the highlighted task's full fields. Users can navigate, expand/collapse, edit, create, delete, complete, filter, start/resume pomodoros, and view help — all via keyboard. All mutations call the existing ConnectRPC handlers (`TaskService`, `PomodoroService`); no new server code is needed.

Technical approach: build the TUI as a new `internal/tui` package using the **Bubble Tea** framework (Charmbracelet's MVU library) plus **Bubbles** (off-the-shelf list, viewport, textinput components) and **Lipgloss** (styling). Reuse `internal/cli`'s existing helpers (`buildTree`, `sortNodes`, `formatDue`, `dimStrike`, etc.) so completed-task styling and ordering match the CLI exactly. Wire a new `Run()` branch in `internal/cli/cli.go` that launches the TUI when `args` is empty and stdout is a TTY.

## Technical Context

**Language/Version**: Go 1.25.0 (matches existing `services/todo/go.mod`)

**Primary Dependencies**:
- `github.com/charmbracelet/bubbletea` — TUI runtime (Elm-style MVU)
- `github.com/charmbracelet/bubbles` — reusable components (list, viewport, textinput, key)
- `github.com/charmbracelet/lipgloss` — terminal styling (reused for completed-task strikethrough/dim)
- `connectrpc.com/connect` — already in use; client calls go through the existing `taskv1connect`, `pomodorov1connect` clients
- `golang.org/x/term` — already in use; for TTY detection

**Storage**: N/A — all state lives in-memory for the TUI session; persistence is via existing ConnectRPC handlers to PostgreSQL.

**Testing**: `go test` with table-driven tests. Bubble Tea models are pure functions of state + message → state, so unit tests drive `Update(msg)` directly without a real terminal. We use the same `export_test.go` shim pattern that the rest of `internal/cli` uses.

**Target Platform**: Linux/macOS terminals capable of ANSI styling. A non-TTY invocation (pipe/redirect) falls back to the existing root-usage banner — no TUI launch.

**Project Type**: CLI / desktop (terminal) application — single-binary Go module.

**Performance Goals**: All keypress → redraw transitions perceived as immediate (<50 ms) on a local-network server. Re-fetching the tree after a mutation completes within one round-trip; no auto-polling.

**Constraints**:
- No new server code. All mutations and reads use existing ConnectRPC handlers.
- Completed-task styling MUST match `todo task list` exactly (reuse `dimStrike`).
- Task ordering MUST match `todo task list` exactly (reuse `buildTree` + `sortNodes`).
- Maintain Go 1.25.0 / no breaking changes to existing CLI subcommands.

**Scale/Scope**: Single user per process. Task counts in the hundreds; tree depth typically 2–4. No virtualization needed for the list — Bubbles' list/viewport handle this natively.

## Constitution Check

*Principle I — Simplicity / YAGNI*: PASS.
- One new internal package (`internal/tui`); no speculative abstractions.
- Reuses existing CLI helpers (`buildTree`, `dimStrike`, `formatDue`, ConnectRPC clients) rather than duplicating logic.
- Bubble Tea + Bubbles + Lipgloss is the smallest viable Go TUI stack; we add no in-house framework.
- No auto-refresh layer (clarification Q2 explicitly chose manual + post-mutation refresh).

*Principle II — API-First Design*: PASS.
- This feature adds no new server endpoints. The "contract" surface for this feature is the keymap and the data shape the TUI consumes from the existing protobuf services. These are documented in `contracts/` (keymap.md, view-model.md) before implementation.

Gate: **PASSING** for both principles. No Complexity Tracking entries required.

## Project Structure

### Documentation (this feature)

```text
specs/010-interactive-tui/
├── plan.md                  # This file
├── spec.md                  # Feature specification (with clarifications)
├── research.md              # Phase 0: TUI library + reuse decisions
├── data-model.md            # Phase 1: TUI session state shape
├── quickstart.md            # Phase 1: how to try it locally
├── contracts/
│   ├── keymap.md            # Keybinding contract (one source of truth)
│   └── view-model.md        # In-memory task-tree model the TUI consumes
└── tasks.md                 # /speckit-tasks output (created later)
```

### Source Code (repository root)

```text
services/todo/
├── cmd/
│   ├── server/main.go                  # unchanged
│   └── todo/main.go                    # unchanged (calls cli.Run)
├── internal/
│   ├── cli/
│   │   ├── cli.go                      # MODIFIED: route empty args to TUI when TTY
│   │   ├── task.go                     # unchanged
│   │   ├── render.go                   # unchanged (TUI reuses dimStrike, buildTree, etc.)
│   │   └── ...
│   ├── tui/                            # NEW package
│   │   ├── tui.go                      # Run entrypoint, Bubble Tea program wiring
│   │   ├── model.go                    # Model struct + Init/Update/View dispatch
│   │   ├── tree.go                     # task tree view model + visible-row flattening
│   │   ├── tree_test.go
│   │   ├── keymap.go                   # key bindings (Bubbles `key.Binding` map)
│   │   ├── details.go                  # details pane renderer
│   │   ├── edit.go                     # edit form (Bubbles textinput chain)
│   │   ├── edit_test.go
│   │   ├── help.go                     # help overlay
│   │   ├── pomodoro.go                 # in-TUI pomodoro start/resume bridge
│   │   ├── client.go                   # ConnectRPC client construction (mirrors cli's)
│   │   ├── update.go                   # Update() message dispatch
│   │   ├── update_test.go
│   │   └── export_test.go              # test shim
│   ├── handler/                        # unchanged
│   ├── pomodoro/                       # unchanged
│   └── ...
└── go.mod                              # MODIFIED: add bubbletea/bubbles/lipgloss
```

**Structure Decision**: Single Go module, single binary. The TUI is a new internal package wired into the existing CLI entry point at exactly one site (`cli.Run` with empty args). No new servers, no new persistence. Per Principle I, we add no abstraction layer between Bubble Tea and the ConnectRPC client — `client.go` is a thin helper that returns ready-to-use clients using the same env vars (`TODO_API_KEY`, `TODO_ADDR`) as the existing CLI.

## Complexity Tracking

> Constitution Check passes; no violations. Section intentionally left without entries.
