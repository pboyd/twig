# Quickstart: TUI Pomodoro Integration

Manual verification recipe for the integrated TUI pomodoro. Automated coverage lives in `internal/tui/*_test.go` (see plan.md Testing).

## Prerequisites

```bash
# From repo root: start postgres + server (auto-runs migrations)
make dev

# Provision a user to get an API key (inside the server container or a local binary)
./todo-server --provision-user me:secret

# Configure the CLI (env vars, or ~/.config/todo/config.toml)
export TODO_API_KEY=<key-from-provisioning>
export TODO_ADDR=http://localhost:8080
```

Optionally set lifecycle hooks in `~/.config/todo/config.toml` so you can see them fire without corrupting the screen:

```toml
[pomodoro]
on_start    = "notify-send 'pom started'"
on_complete = "notify-send 'pom done'"
on_cancel   = "notify-send 'pom cancelled'"
```

Build and launch the TUI:

```bash
cd services/todo && go build -o todo ./cmd/todo && ./todo
```

## Scenarios

### S1 — Run a pomodoro while working (Story 1 / P1)
1. Select a task and press `s`.
2. **Expect**: a `🍅 mm:ss · <task>` timer appears in the status bar and counts down; the status area is now two lines.
3. Navigate (`j`/`k`), expand/collapse, edit (`e`), add a subtask (`n`), move (`m`).
4. **Expect**: every action works normally and the timer keeps ticking and stays visible in each mode (including the help screen `?`).
5. Press `x`.
6. **Expect**: the pomodoro ends, the `on_cancel` hook fires, and the timer disappears (status area back to one line).

### S2 — Same-task / different-task start (Story 1 scenarios 1.4–1.5)
1. With a pomodoro running on task A, select task A and press `s` → **Expect**: nothing restarts; the existing countdown continues.
2. Select task B and press `s` → **Expect**: task A's pomodoro ends and a fresh countdown starts on task B.

### S3 — Automatic completion (Story 2 / P2)
1. Start a pomodoro. (For a fast check, temporarily shorten `pomodoro.Length`, or use the unit tests with an injected clock.)
2. Let it reach `0:00` while the TUI is open.
3. **Expect**: the pomodoro is recorded complete, `on_complete` fires once, and a non-blocking `🍅 Pomodoro complete! · <task>` banner appears.
4. Press any key (or wait ~5s) → **Expect**: the banner clears and the status area returns to one line.

### S4 — Auto-attach at launch (Story 3 / P2)
1. From another terminal: `todo pom start <task_id>`.
2. Launch the TUI.
3. **Expect**: the running pomodoro is already showing in the status bar with the correct remaining time and task name — no key press needed. (`r` no longer exists.)
4. Launch the TUI with no active pomodoro → **Expect**: normal one-line status bar.

### S5 — Quit guard (Story 4 / P3)
1. With a pomodoro running, press `q`.
2. **Expect**: a `🍅 mm:ss still running. Quit anyway? [y]es [n]o` confirm appears.
3. Press `n`/`esc` → **Expect**: back in the list, timer intact.
4. Press `q` then `y` → **Expect**: TUI exits; `todo pom status` in a shell shows the pomodoro is still running.
5. With no pomodoro running, press `q` → **Expect**: immediate exit, no prompt.

### S6 — Terminal-safe hooks (FR-013 / SC-004)
1. Set a hook that writes to stdout and sleeps, e.g. `on_complete = "echo done; sleep 2"`.
2. Trigger completion.
3. **Expect**: the screen is not corrupted by the hook's stdout, the UI stays responsive during the 2s sleep, and a failing hook (e.g. `on_complete = "false"`) surfaces as a status-bar error rather than raw terminal text.

### S7 — Plain terminal (edge case)
1. Run with output piped / `TERM=dumb` or otherwise non-styled.
2. **Expect**: the timer and banner render legibly as plain text without relying on color/emoji styling.

## Regression check

```bash
cd services/todo && go test ./... && go build -o todo ./cmd/todo
# Confirm `todo pom start|resume|cancel|status` still behave exactly as before (FR-016).
```
