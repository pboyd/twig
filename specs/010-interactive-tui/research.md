# Phase 0 Research: Interactive TUI

## R1 — TUI framework

**Decision**: Use Bubble Tea (`github.com/charmbracelet/bubbletea`) with Bubbles components and Lipgloss styling.

**Rationale**:
- Active, well-documented, and the de facto modern Go TUI stack.
- Elm-style MVU model means `Update(msg) -> (Model, Cmd)` is a pure function; we can unit-test every keypress and message without spinning up a real terminal.
- Bubbles ships ready-made `list`, `viewport`, `textinput`, and `key.Binding` components, eliminating most of the boilerplate this feature would otherwise need.
- Lipgloss styles compose with the strings the existing CLI already produces (we already use ANSI escapes in `dimStrike`); no rewrites of the styling helpers needed.

**Alternatives considered**:
- `rivo/tview` — older, more "widget framework" style, harder to unit test because event handlers are imperative callbacks.
- `gdamore/tcell` directly — too low-level; we'd be rebuilding list/viewport/text-input from scratch, violating Principle I.
- Hand-rolled curses-style implementation — gratuitously complex for an internal tool.

## R2 — Reuse of existing CLI helpers

**Decision**: The TUI's tree builder reuses `internal/cli.buildTree`, `internal/cli.sortNodes`, `internal/cli.formatDue`, `internal/cli.formatCompletedAt`, and the styling helpers (`dimStrike`). Either we export them (rename to `BuildTree`, `SortNodes`, etc.) or expose them via an internal alias file inside `internal/cli`.

**Rationale**:
- Clarification Q1 nailed task ordering to "match `todo task list`" — the only way to guarantee that across CLI and TUI is to call the same code. Re-implementing sort logic in the TUI would drift over time.
- Completed-task styling (strikethrough + dim) is already implemented in `dimStrike`; reusing it satisfies SC-004 ("CLI familiarity") by construction.

**Alternatives considered**:
- Duplicate the helpers inside `internal/tui`. Rejected: violates Principle I and creates two sources of truth for ordering/styling.
- Move the helpers into a new `internal/taskview` package. Rejected as out of scope; we'd only do that refactor if it had a second consumer. For now, simple exports from `internal/cli` are enough.

**Action**: Promote a small set of helpers from package-private to exported, keeping them inside `internal/cli`. Documented as a task in `/speckit-tasks`.

## R3 — TTY detection / non-interactive fallback

**Decision**: Use `golang.org/x/term.IsTerminal(int(os.Stdout.Fd()))` to gate TUI launch. If not a TTY, fall back to the existing `printRootUsage` output (current no-args behavior) and exit 0 (or 1 — match what `cli.Run` does today on no-args, which is exit 1; keep exit code stable to avoid breaking scripts).

**Rationale**:
- `golang.org/x/term` is already a direct dependency (see `go.mod`).
- Matches the project's existing TTY detection pattern in `wantStyled`.

**Alternatives considered**:
- Try-launch and bail on failure — wasteful and produces ugly garbage in pipes.
- Detect via `$TERM` env var — unreliable; users in tmux/screen may have `dumb` or unset values but still be in a real terminal.

## R4 — Pomodoro integration

**Decision**: Reuse the existing `internal/cli.runPomStart` and `runPomResume` flows but adapt them so they don't call `os.Exit`. Two practical options:
1. Refactor those functions to return an `error` instead of an exit code, with the existing CLI wrappers translating that back to an exit code.
2. Spawn the pomodoro subprocess from the TUI via `exec.Command` and re-enter the TUI when it exits.

Pick option 1 (refactor in-place). The pomodoro logic already lives in `internal/pomodoro`; the CLI wrapper just owns the countdown rendering. We add a `RunPomodoroForTask(ctx, taskID)` function in `internal/cli` that the TUI invokes; Bubble Tea suspends rendering during the pomodoro (via `tea.ExecProcess` or by exiting and re-entering the alt-screen). On completion the function returns and the TUI re-enters its main loop.

**Rationale**:
- One source of truth for pomodoro behavior; no risk of drift between TUI and CLI flows.
- `tea.ExecProcess` is exactly the Bubble Tea pattern for "yield the terminal to a subprocess (or function), then resume" — battle-tested.

**Alternatives considered**:
- Reimplement pomodoro inside the TUI as its own Bubble Tea model. Rejected: duplicates `pom_countdown.go`, violates Principle I.

## R5 — External-change refresh strategy

**Decision** (locks in clarification Q2): No background poller. The TUI calls `TaskService.ListTasks` exactly at:
- Launch.
- After every mutation it performs (create/edit/delete/complete/estimate).
- When the user presses `Ctrl-R`.

**Rationale**: Simplest correct behavior; no goroutine lifecycle to manage; no spurious "task changed under your cursor" surprises. Matches user expectation that a CLI shows a snapshot until it acts.

**Alternatives considered**: Periodic polling and server-sent events — both rejected per Q2 clarification.

## R6 — Help screen content source

**Decision**: Help text is generated from the `keymap.go` `key.Binding` definitions, not hand-maintained in a separate file. Bubbles' `help` component does this out of the box.

**Rationale**:
- Single source of truth: change a binding, the help screen updates automatically.
- Satisfies SC-005 ("100% of keybindings discoverable via help") by construction.

## R7 — Edit form field strategy

**Decision**: Edit form is a vertical stack of `textinput` components, one per editable field (name, description, due date as text, parent — read-only display, pomodoro estimate). Date is a free-text field validated on save using the existing `parseDue` helper from `internal/cli/task.go`. Tab/Shift-Tab cycle focus; Enter on a single-line field advances; on the description (multi-line) we use a textarea component.

**Rationale**:
- `parseDue` already exists and is tested; reusing it keeps date-handling consistent with `todo task add --due`.
- No date-picker widget needed in v1; users already know the text format from the CLI.

**Alternatives considered**:
- Custom date picker widget — overkill for v1; can be added later if friction emerges.
