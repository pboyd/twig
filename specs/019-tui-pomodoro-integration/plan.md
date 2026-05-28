# Implementation Plan: TUI Pomodoro Integration

**Branch**: `019-tui-pomodoro-integration` | **Date**: 2026-05-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/019-tui-pomodoro-integration/spec.md`

## Summary

Replace the TUI's full-screen, blocking pomodoro takeover with an integrated background timer. Today, starting a pomodoro calls `tea.Exec` to suspend the entire TUI and hand the terminal to the CLI countdown renderer (`cli.RunPomodoroForTask`), which is why the user loses access to the task list and why an abandoned pomodoro never completes (its `on_complete` hook fires client-side inside that loop).

The new design makes the active pomodoro a piece of background `Model` state (task id, cached task name, server `start_at`, completed flag) driven by a one-second `tea.Tick`. Remaining time is derived locally via the existing `pomodoro.Remaining(start, now)`; the server stays the source of truth. The timer renders in the bottom status area (which grows to two lines while active), all normal task interactions keep working, and the timer transitions (start / cancel / complete) call the already-existing `StartPomodoro`, `CancelPomodoro`, `CompletePomodoro`, and `GetActivePomodoro` RPCs. Lifecycle hooks run off the UI goroutine with stdio detached so they cannot corrupt the alt-screen. This is a purely client-side change inside `services/todo/internal/tui/` — no proto, handler, db, or migration work.

## Technical Context

**Language/Version**: Go 1.22+ (existing module at `services/todo/`).

**Primary Dependencies**: Bubble Tea (`github.com/charmbracelet/bubbletea`) for the event loop and `tea.Tick`; Bubbles `help` (existing); Lipgloss for status-bar styling; existing ConnectRPC `TaskServiceClient`. No new dependencies.

**Storage**: N/A — pomodoro state is server-authoritative via existing RPCs; the TUI holds only a transient in-memory view of the active pomodoro.

**Testing**: `go test ./...` from `services/todo/`. Reducer/transition unit tests in `internal/tui/` following the existing `update_test.go` / `view_test.go` style: drive `Model.Update` directly with synthetic messages, inject `now` for time-dependent logic, and use a fake `TaskServiceClient` plus the `export_test.go` shim pattern. No DB or running-server dependency.

**Target Platform**: TTY (Linux/macOS terminals); alt-screen Bubble Tea program.

**Project Type**: CLI / TUI client (single Go binary at `cmd/todo`).

**Performance Goals**: Timer redraws once per second with no per-tick network call; displayed remaining time stays within 1s of the authoritative `start_at` (SC-003). Hook execution never blocks rendering (SC-004).

**Constraints**: Keyboard-only; no new dependencies; no server changes; lifecycle hooks must not write to the TUI-controlled terminal; must render legibly in unstyled (non-TTY-styled) output.

**Scale/Scope**: One active pomodoro at a time per user (server invariant). Change is confined to ~6 files under `internal/tui/`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Simplicity / YAGNI**: PASS. Removes machinery (the `tea.Exec` takeover: `funcExecCommand`, `execPomodoroStart/Resume`, `pomodoroRequestMsg`/`pomodoroDoneMsg`, and the `modePomodoro` view mode) and replaces it with a single nullable `*activePom` field plus a `tea.Tick`. No new packages or abstractions; reuses `pomodoro.Remaining`, the existing RPC client, and the existing status-bar rendering path. Out-of-scope ideas (configurable durations, breaks, server-side autonomous completion, continuous cross-client polling) are explicitly excluded in the spec's Assumptions.
- **II. API-First Design**: PASS. No contract changes. The feature consumes existing, already-implemented RPCs (`StartPomodoro`, `CancelPomodoro`, `CompletePomodoro`, `GetActivePomodoro`, `GetTask`) defined in `services/todo/proto/task/v1/task.proto`. `contracts/pomodoro-rpcs.md` documents the reused contract; no `make proto` step is required.

No violations. Complexity Tracking table is unused.

## Project Structure

### Documentation (this feature)

```text
specs/019-tui-pomodoro-integration/
├── spec.md                  # Feature spec (already created)
├── plan.md                  # This file
├── research.md              # Phase 0 output (timing engine, hook safety, status-area layout)
├── data-model.md            # Phase 1 output (TUI-side active-pomodoro state + transitions)
├── quickstart.md            # Phase 1 output (manual verification recipe)
├── contracts/
│   └── pomodoro-rpcs.md     # Phase 1 output (pointer to existing pomodoro RPC contract)
└── checklists/
    └── requirements.md      # From /speckit-specify
```

### Source Code (repository root)

```text
services/todo/internal/tui/
├── model.go        # MODIFY: remove modePomodoro; add `pom *activePom` field + activePom type (taskID, taskName, startAt, completed, banner)
├── pomodoro.go     # REWRITE: drop tea.Exec helpers (funcExecCommand, execPomodoroStart/Resume, request/done msgs);
│                   #          add tick (pomTickMsg) + command factories (startPomCmd, cancelPomCmd, completePomCmd,
│                   #          getActivePomCmd) and the terminal-safe hook runner
├── update.go       # MODIFY: Init also fires getActivePomCmd; handle pomTickMsg / pom*Msg; rewire `s` (start) and
│                   #          add `x` (cancel); remove `r` resume path; quit-guard on `q`; clear banner on keypress
├── keymap.go       # MODIFY: replace PomResume("r") with PomCancel("x"); keep PomStart("s"); update help groupings
├── view.go         # MODIFY: renderStatus grows to two lines while a pom is active (timer/banner line + help/error line);
│                   #          statusHeight() helper feeds the pane height math in every view function; quit-confirm overlay
├── help.go         # MODIFY: reflect new keys (start/cancel; resume removed)
└── *_test.go       # NEW/MODIFY: pomodoro_test.go (transitions w/ injected clock + fake client), update_test.go &
                    #             view_test.go (start/tick/complete/cancel, auto-attach, quit guard, two-line status, hook safety)
```

No changes outside `services/todo/internal/tui/`. No proto, handler, db, or migrations. The CLI `todo pom` subcommands and `internal/cli/pom.go` (`fireHook`, `RunPomodoroForTask`, `ResumeBackgroundedPomodoro`) are left untouched per FR-016.

**Structure Decision**: Single Go module, TUI-only change. Follows the existing pattern where pomodoro concerns live in `internal/tui/pomodoro.go` with a small surface integrated into `update.go` / `view.go` / `keymap.go`, mirroring how feature 017 (move task) was structured.

## Complexity Tracking

Not applicable — no Constitution violations.
