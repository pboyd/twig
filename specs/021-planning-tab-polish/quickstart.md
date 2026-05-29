# Quickstart: Verifying Planning Tab Polish

Manual verification walkthrough. Automated coverage lives in `internal/tui/*_test.go` and `internal/cli/plan_grid_test.go`.

## Setup

```bash
cd services/todo
go build -o todo ./cmd/todo
# Requires a running server + provisioned user (see CLAUDE.md "Dev Environment").
export TODO_API_KEY=<key>
export TODO_ADDR=http://localhost:8080
./todo            # launches the TUI on a TTY
```

Seed a few tasks (at least one with sub-tasks) and a plan day with both a task entry and an event entry.

## 1. Pomodoro cancel on Planning (FR-001/002, SC-002)

1. On the Tasks tab, start a pomodoro (`s`).
2. Switch to Planning (`tab`). Confirm the status bar shows the running timer and `[x] cancel`.
3. Press `x`. **Expect**: the pomodoro stops; the status bar reverts to the help line.
4. Switch back to Tasks. **Expect**: no running pomodoro there either.
5. With nothing running, press `x` on Planning. **Expect**: nothing happens, no error.
6. Start a pomodoro again, switch to Planning, press `q`. **Expect**: a "still running — quit anyway?" prompt; `n` cancels the quit, `y` quits.

## 2. Planning help screen (FR-003/004, SC-001)

1. On Planning, press `?`. **Expect**: a full-screen help listing Planning keys (nav, day nav, add task/event, rename, move, remove, clear, today, refresh, tab, help, quit) in the same style as the Tasks help.
2. Press `?` again. **Expect**: help closes; the grid returns with the same entry selected as before.

## 3. Sub-tasks in the add-task picker (FR-005/006, SC-003)

1. On Planning, press `a`. **Expect**: the picker lists tasks as a tree, with sub-tasks nested under their parents (the same hierarchy the move dialog shows on the Tasks tab via `m`).
2. Select a sub-task, press `enter`, give it a start time, save. **Expect**: an entry for that sub-task appears on the grid.

## 4. Visual consistency (FR-007/008/009/010, SC-004/005)

1. Compare the two tabs side by side. **Expect**:
   - Planning uses the same blue accent (selected entry highlighted with the accent, not plain bold/monotone).
   - The tab bar is polished and clearly marks the active tab.
   - Neither tab shows a redundant pane title (no `"Tasks"`/`"Planning"` label inside the left pane); the right pane still reads `"Details"`; the Planning date header remains.
   - The status/help line sits on the bottom row on both tabs, at tall and short terminal heights.

## 5. Two-pane layout & details (FR-011/012/013, SC-006)

1. On Planning, confirm the grid is on the left and a details pane on the right.
2. Move the selection (`↑`/`↓`). **Expect**: the right pane updates to the selected entry — name, `HH:MM–HH:MM` window, duration; for a task entry, the linked task and completion state; an event omits task-only fields.
3. Navigate to an empty day (`]` until empty). **Expect**: details pane shows an empty/placeholder state, no error.

## 6. Graceful degradation (FR-014)

1. Run in a non-ANSI context: `./todo | cat` is not a TTY, but to test the styled fallback resize a real terminal very narrow.
2. **Expect**: the two-pane split collapses without corrupting the grid or details; no stray escape sequences.

## 7. Non-regression (FR-015, SC-007)

```bash
cd services/todo && go test ./...
```

**Expect**: all existing tests pass, including `internal/cli` grid tests (proving the additive `GridOptions.Styled` change left CLI output unchanged) and the prior Planning tab tests.
