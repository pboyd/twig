# Quickstart: Goal Status Updates

**Feature**: 051-goal-status-updates | **Date**: 2026-06-12

How to build, run, and manually verify the feature. TUI only.

## Build & regenerate

```bash
# After editing api/proto/goal/v1/goal.proto
make proto

# After editing services/twig/db/queries/goal_status_update.sql
cd services/twig && sqlc generate

# Build the CLI/TUI
go build -o twig ./cmd/twig

# Tests
go test ./...                 # root module (TUI)
cd services/twig && go test ./...   # server (handlers)
```

## Run the stack

```bash
make dev   # postgres + server (auto-runs migrations, incl. 000011)
export TWIG_API_KEY=...        # from --provision-user
export TWIG_ADDR=http://localhost:8080
./twig                          # launches the TUI; shift+tab from Tasks → Goals
```

## Manual verification (maps to user stories)

### US1 — record & read the latest status (P1)

1. On the Goals tab, select a goal.
2. Press `S`. Your `$EDITOR` opens an empty `.md` file. Type
   `Narrowed to **two** EVs; test drive next week.`, save, quit.
3. The detail pane shows a "Latest status" section: a relative timestamp and the
   text with `**two**` rendered bold (not raw markup).
4. Press `S` again, add a newer note, save. The newer note replaces the one shown
   as latest, with its own timestamp.
5. Press `S`, save an empty file → nothing is recorded; a notice explains why.
6. On a goal with no updates, the detail pane shows the friendly empty state.

### US2 — browse history (P2)

1. After recording several updates, press `s` to open the status-history view.
2. Updates are listed newest-first, each with its timestamp.
3. Move the cursor with ↑/↓; press `enter` on one to open the full reader.
4. For a long entry (paste 5000+ chars), the reader scrolls through all of it; no
   truncation. Press `esc` to return to the list, `esc` again to the goal detail.

### US3 — correct or remove (P3)

1. In the history view, select an update and press `e`. `$EDITOR` opens prefilled
   with its current text; edit, save → the updated text shows (markdown rendered),
   timestamp unchanged.
2. Select an update and press `d`; confirm `y`. It disappears; siblings remain.
3. Delete the last remaining update → the goal detail returns to the empty state.

### Edge / scoping

- Delete a goal that has updates → its updates are gone too (DB cascade). Verify
  via `psql` that no orphan rows remain.
- A second user cannot see or modify the first user's updates (RPCs return
  `NotFound`).

## Key bindings added (Goals tab)

| Key | Action |
|---|---|
| `S` | Add a status update (opens `$EDITOR`) |
| `s` | Open the status-history view |
| (in history) `n` | New update |
| (in history) `e` | Edit selected update |
| (in history) `d` | Delete selected update (confirm) |
| (in history) `enter` | Open the full reader |
| (in history/reader) `esc` | Back |
