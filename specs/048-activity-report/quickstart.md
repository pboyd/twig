# Quickstart: Activity Report

How to exercise the feature end-to-end once implemented.

## Setup

```bash
make dev                                   # postgres + server via podman-compose
# provision a user if you don't have one, then:
export TWIG_API_KEY=<key>
export TWIG_ADDR=http://localhost:8080
go build -o twig ./cmd/twig
```

## Seed some history

```bash
./twig task add "Write quarterly summary"
./twig task add "Web app polish"
./twig task add --parent <web-app-id> "Wire up the download page"
./twig task complete <subtask-id>
./twig task complete <summary-id>
# optionally: ./twig pom start <id> … and complete a pomodoro
```

To test past days, complete tasks and then adjust `completed_at` directly:

```bash
psql postgres://twig:twig@localhost:5432/twig \
  -c "UPDATE tasks SET completed_at = NOW() - INTERVAL '1 day' WHERE id = <id>"
```

## CLI checks

```bash
./twig report                       # recent (yesterday + today), day-grouped
./twig report yesterday             # only yesterday's completions
./twig report last-week             # Monday–Sunday of last week
./twig report quarter               # > 14 days → accomplishment-grouped layout
./twig report --from 2026-01-01 --to 2026-03-31
./twig report | cat                 # piped → no ANSI styling
./twig report --from 2026-03-31 --to 2026-01-01   # → playful validation error, exit 1
./twig help report
```

Verify:
- Most recent day appears first in day-grouped output.
- Completed subtasks show their parent context.
- Summary line counts tasks and pomodoros; pomodoros on still-open tasks count.
- A period with no completions prints the playful empty state and exits 0.
- `task uncomplete` removes a task from the report on the next run.

## TUI checks

```bash
./twig          # launches the TUI on a TTY
```

- `tab` twice → Report tab; default period is "recent".
- `←/→` cycle presets; header label and totals update.
- A near-midnight completion lands on the correct local day.
- `r` refreshes after completing a task in another terminal.

## Tests

```bash
go test ./...                                   # CLI/TUI/report unit tests
cd services/twig && go test ./...               # handler tests (skip without DATABASE_URL)
cd services/twig && DATABASE_URL=postgres://twig:twig@localhost:5432/twig?sslmode=disable go test ./internal/handler/
```
