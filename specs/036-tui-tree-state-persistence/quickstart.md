# Quickstart: TUI Tree State Persistence

## What this feature does

The TUI now remembers which task subtrees you had expanded or collapsed. Quit, relaunch, and your tree comes back the way you left it — per account, stored locally.

## Try it

```bash
# Build the CLI/TUI from repo root
go build -o twig ./cmd/twig

# Launch the TUI (default profile)
./twig
```

1. Expand a few tasks (`l` / `→`) and collapse others (`h` / `←`).
2. Quit (`q`).
3. Relaunch `./twig` — the same tasks are expanded/collapsed as before.

With a named profile, state is kept separate:

```bash
./twig --profile work     # remembers the "work" account's tree independently
```

## Where state lives

```
$XDG_STATE_HOME/twig/tree-state.json     # or ~/.local/state/twig/tree-state.json by default
```

A small JSON file mapping each profile to its expanded task IDs, e.g. `{"default":[1,4,9],"work":[12]}`. Safe to delete — the TUI just starts fresh (everything collapsed) and rebuilds it. See `contracts/tree-state-file.md` for the format.

## Verify (developer)

```bash
# From repo root
go test ./internal/tui/...     # unit tests for load/save/prune + per-profile isolation
go test ./...                  # full suite
```

Manual checks aligned to the spec's success criteria:

- **SC-001 / SC-002**: Curate the tree, relaunch, confirm identical expansion with zero manual toggles.
- **SC-003**: `echo 'garbage' > ~/.local/state/twig/tree-state.json`, relaunch — TUI opens normally (all collapsed), no error.
- **SC-004**: Expand a task, delete it (elsewhere), relaunch and re-save — its ID no longer appears in the file.
- **FR-009**: Curate different trees under two profiles; confirm each `--profile` shows its own state.
```
