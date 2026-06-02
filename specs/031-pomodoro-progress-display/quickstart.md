# Quickstart: Pomodoro Progress Display

## Build & test

```bash
cd services/twig

# After editing proto + queries:
make proto            # regenerate gen/ (Task.completed_pomodoro_count)
sqlc generate         # regenerate internal/db (CountCompletedPomodorosByTask)

go test ./...         # all tests, incl. new pomodoro_row_test.go
go build -o twig ./cmd/twig
```

## Run the stack

```bash
make dev              # postgres + server via podman-compose (auto-migrates)
# provision a user / API key if needed (see CLAUDE.md), then:
TWIG_API_KEY=... TWIG_ADDR=http://localhost:8080 ./twig
```

## Verify the glyph row (task tree)

1. Launch the TUI (`twig` with no args on a TTY).
2. Select a task and set an estimate (e.g. 5) — the details pane shows five dimmed 🍅.
3. Start and complete a pomodoro on that task. After completion the tree reloads;
   the first glyph turns bold red → `🍅(red) 🍅🍅🍅🍅(dim)`.
4. Complete pomodoros until completed == estimate: all five bold red.
5. Complete one more (6th): the row grows to six glyphs, five bold red + one
   bold **yellow** — matching the spec's worked example.
6. Clear the estimate and completions (a fresh task with neither): the row is
   absent from the details pane.

## Verify the glyph row (planning tab)

1. Add a plan entry linked to a task that has an estimate and some completions.
2. Select that entry — the planning detail pane shows the same glyph row,
   colored identically to the task-tree pane (consistency check, Principle III).
3. An event entry (no linked task) shows no glyph row.

## Verify the active-pomodoro glyph (US3)

1. Start a pomodoro. The status bar shows `🍅 MM:SS · <task>  [x] cancel`.
2. The leading 🍅 is rendered red and bold.

## Plain-mode check (FR-010)

```bash
./twig | cat        # or pipe to a non-TTY / NO_COLOR
```

Confirm details still show the correct number of 🍅 glyphs with no ANSI styling
and no layout breakage.

## Acceptance mapping

- US1 / FR-001,003–007,009 → task-tree steps above.
- US2 / FR-002 → planning-tab steps above.
- US3 / FR-008 → active-glyph step above.
- FR-010 → plain-mode check.
- SC-005 → the 5→2→6 worked example in the task-tree steps.
