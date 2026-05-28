# Phase 0 Research: TUI Pomodoro Integration

All major decisions were settled in a prior brainstorming session; this document records them in decision/rationale/alternatives form. No open `NEEDS CLARIFICATION` items remain.

## R1. Timing engine: local tick vs. server polling vs. timer component

- **Decision**: Drive a one-second `tea.Tick` (rescheduled only while a pomodoro is active). Each tick recomputes `remaining := pomodoro.Remaining(m.pom.startAt, time.Now())` purely locally. The server's `start_at` (captured at start or at auto-attach) is the source of truth; the TUI never persists its own duration.
- **Rationale**: Matches how the codebase already models pomodoros — the server stores `start_at` and clients derive remaining time via `pomodoro.Remaining`. No per-second network traffic, no new failure modes on the hot path, and `pomodoro.Remaining` already has test coverage. Aligns with Constitution Principle I (simplicity).
- **Alternatives considered**:
  - *Poll `GetActivePomodoro` every tick* — robust to another client cancelling out from under us, but chatty and adds a network failure path once per second. Rejected; cross-client drift is rare and handled opportunistically (R5).
  - *`bubbles/timer` component* — keeps its own duration state, which double-books against the server `start_at` and complicates auto-attach (the elapsed time must be reconstructed). More moving parts, less control. Rejected.

## R2. Completion detection and the "fire exactly once" guarantee

- **Decision**: On each tick, if `remaining == 0` and the pomodoro is not already marked completed, dispatch the completion transition (`CompletePomodoro` + `on_complete` hook + banner) and set `m.pom.completed = true`, which also stops further ticks. A guard on `m.pom != nil && !m.pom.completed` prevents a second firing.
- **Rationale**: Satisfies FR-006 ("exactly once") and SC-002 (fires within 1s of zero). Tick cadence of one second bounds the latency.
- **Alternatives considered**: Schedule a single `tea.Tick` for exactly the remaining duration instead of per-second ticks — but we need per-second ticks anyway to redraw the countdown, so a separate completion timer is redundant.

## R3. Stale tick / late message handling

- **Decision**: Any `pomTickMsg` (and any completion/cancel result message) is ignored when `m.pom == nil` or when it does not correspond to the current pomodoro. The next tick is only scheduled while a pomodoro is active and not completed.
- **Rationale**: Directly addresses FR-014 and the "stale timer events" edge case — a tick already in flight when the user cancels must not resurrect a timer.

## R4. Start-conflict policy without interactive prompts

- **Decision**: Pressing `s` issues `StartPomodoro`. On `CodeAlreadyExists`, extract the active task id from the error detail (existing `extractActiveTaskID` logic): if it is the **same** task, attach to the running pomodoro (no restart); if a **different** task, call `CancelPomodoro` then `StartPomodoro` for the new task. No stdin prompt.
- **Rationale**: The TUI cannot read line input mid-session, and this mirrors the non-interactive policy already encoded in `cli.RunPomodoroForTask`. Satisfies FR-010 and acceptance scenarios 1.4 / 1.5.
- **Alternatives considered**: An in-TUI confirm dialog for the conflict — rejected as unnecessary ceremony; the brainstorm chose the auto policy.

## R5. Auto-attach at launch and cross-client reconciliation

- **Decision**: `Init` fires `getActivePomCmd` alongside the existing `listTasksCmd`. `getActivePomCmd` calls `GetActivePomodoro`; if a pomodoro is returned, it also resolves the task name (via the already-loaded tree or `GetTask`) and seeds `m.pom`, starting the tick. Cross-client divergence (e.g., the CLI cancels mid-session) is reconciled opportunistically on the next list refresh rather than by polling.
- **Rationale**: Satisfies FR-009 / Story 3 and SC-003 (correct remaining time immediately at launch). Keeps the hot path network-free (R1) while still self-correcting on natural refresh points. The manual `r` resume becomes unnecessary and is removed (FR-017).

## R6. Terminal-safe lifecycle hooks

- **Decision**: Add a TUI-specific hook runner that runs `sh -c <cmd>` with stdio **detached** (not `os.Stdin/Stdout/Stderr`), executed inside a `tea.Cmd` (off the UI goroutine). Empty hook strings are no-ops. A non-zero exit returns a `pomHookErrMsg` that is surfaced through the existing `m.err` status-area path; it never writes raw text to the terminal and never aborts the pomodoro transition itself.
- **Rationale**: The existing `cli.fireHook`/`execHook` attach hook stdio to the process's terminal and print warnings to stderr — both would scribble over the alt-screen, and a slow synchronous hook would freeze the UI. Running detached and async satisfies FR-013, SC-004, and the "hook output and failures" edge case. The CLI's `fireHook` is left untouched for `todo pom` (FR-016).
- **Alternatives considered**: Reuse `cli.fireHook` directly — rejected because it targets the terminal and is synchronous.

## R7. Status-area layout (one line → two lines)

- **Decision**: While `m.pom != nil`, the bottom status area renders two lines: a pomodoro line (`🍅 mm:ss · <task>  [x] cancel`, or the completion banner) above the existing help/error line. A `statusHeight()` helper returns 1 or 2 and feeds the pane height math (`m.height - 3` styled / `maxLines := m.height - 2` plain) in every view function so panes shrink by one row instead of overflowing. Emoji/color are gated by `m.styled` for legible plain-terminal output.
- **Rationale**: Satisfies FR-002, FR-004, and the "plain terminals" edge case. The status area is shared by the list/edit/move view functions, so the timer stays visible across those modes. The full-screen help overlay (`viewHelp`) does not use the shared status area and is exempt per FR-002 (it is a whole-screen, any-key-to-dismiss overlay).
- **Alternatives considered**: A fixed two-line status area even when idle — rejected to avoid permanently shrinking the panes when no pomodoro is running.

## R8. Quit guard

- **Decision**: `q` with `m.pom != nil && !m.pom.completed` sets a `confirmingQuit` flag and renders a one-key confirm overlay (`🍅 mm:ss still running. Quit anyway? [y]es [n]o`). `y` → `tea.Quit` (pomodoro keeps running server-side, since no cancel/complete is called). `n`/`esc` → dismiss. `q` with no active or an already-completed pomodoro quits immediately.
- **Rationale**: Satisfies FR-011 / FR-012 / Story 4. No server changes — quitting simply stops observing; the pomodoro is finalized when a client next attaches and finds it expired (per spec Assumptions).
- **Alternatives considered**: Cancel-on-quit and server-side autonomous completion were both considered in brainstorming and rejected (the former loses work, the latter is a much larger, security-sensitive change).
