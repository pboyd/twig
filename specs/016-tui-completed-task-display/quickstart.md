# Quickstart — Verify TUI Completed Task Display

## Build and run

```bash
cd services/todo
go test ./internal/cli/... ./internal/tui/...   # unit tests should pass
go build -o todo ./cmd/todo                      # build the CLI
make dev                                         # start postgres + server (if not running)
./todo                                           # launch TUI on a TTY
```

(`TODO_API_KEY` and `TODO_ADDR` must be set, or in `~/.config/todo/config.toml`. Use `./todo-server --provision-user name:password` inside the server container to create one.)

## Manual verification

### Story 1 — lingering completed task

1. Create or open a workspace with at least three incomplete tasks.
2. Launch the TUI (`./todo`).
3. Navigate to the **middle** task (not the first or last).
4. Press the `Complete` key (see in-TUI help with `?` if unsure).
5. **Expected**: the task stays where it was, the name is rendered with strikethrough, and the row is still the highlighted (selected) row. The detail pane on the right shows the same name **without** strikethrough and adds a `Completed: <timestamp>` line.
6. Press `Down` (or `Up`).
7. **Expected**: the just-completed task disappears from the tree and the adjacent task becomes highlighted normally.

### Story 1 — same-row actions do not remove the lingering task

1. Repeat steps 1–5 above.
2. Press `Expand` (or `Collapse`) — if the lingering row has no children this is a no-op for the row; otherwise it toggles its subtree. Either way, the lingering row stays.
3. Press `?` to open help, then close it.
4. **Expected**: lingering row is still visible, still selected, still struck through.
5. Now press `Up` → it disappears as in Story 1.

### Story 1 — refresh drops the lingering row

1. Repeat steps 1–5 above to enter the lingering state.
2. Press the `Refresh` key.
3. **Expected**: tree reloads and the lingering row is gone (assuming `showCompleted=false`); the cursor lands on a sensible nearby row.

### Story 2 — completed children render with strikethrough in the tree

1. Toggle the `Filter` (show-completed) key so completed tasks are visible.
2. **Expected**: every completed task in the tree — at any nesting depth — renders with strikethrough on its name. Incomplete tasks render normally.
3. Select a completed task.
4. **Expected**: the detail pane renders the name *without* strikethrough; the `Completed: <timestamp>` line is present.

## Non-TTY sanity check

```bash
./todo | cat   # pipe to disable TTY detection (not interactive — exits immediately or hangs)
```

In non-TTY contexts the `cli.Strike` helper is a no-op; any text-mode snapshot you take MUST contain no `\x1b[9m` codes.

## Automated test verification

```bash
cd services/todo
go test ./internal/cli/...       # covers cli.Strike helper
go test ./internal/tui/...       # covers pendingComplete lifecycle + render output
```

All tests in these two packages should pass.
