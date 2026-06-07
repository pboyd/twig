# Quickstart: Show Scheduled Days in Task Details

## What you're building

A `Scheduled for: 2026-06-07, 2026-06-08` line in the TUI **Tasks-tab details pane**, listing the current/future days a task appears in the daily plan. Past days (by the user's **local** date) are hidden; no line when there are none.

## Prerequisites

- Stack running: `make dev` (Postgres + server).
- A provisioned user with `TWIG_API_KEY` / `TWIG_ADDR` set.
- `buf` CLI (for `make proto`) and `sqlc` CLI.

## Build order (API-first)

1. **Proto** — add `ListScheduledDays` RPC + `ListScheduledDaysRequest/Response/ScheduledDay` to `api/proto/plan/v1/plan.proto` (see `contracts/plan-service.md`). Run `make proto`.
2. **SQL** — add `ListScheduledDaysForTasks` to `services/twig/db/queries/plan.sql` (see `data-model.md`). Run `sqlc generate` in `services/twig/`.
3. **Handler** — implement `(*handler.Plan).ListScheduledDays` in `services/twig/internal/handler/plan.go`: validate `from_day` with `parseDay`, call the query with the context user id, map rows to `ScheduledDay`. Register the RPC (it's part of the already-registered `PlanService`, so no new mount — just the method).
4. **TUI data** — add `scheduledDays map[int64][]string` to `Model`; add `listScheduledDaysCmd(planClient, localToday)` + a result message that builds the map; issue it at init, on `Ctrl-R`, on switch to Tasks tab, and after `ctrl+p` send-to-plan.
5. **TUI render** — in `internal/tui/details.go`, emit the `Scheduled for:` line in both the plain and styled paths from the map; omit when empty.

## Manual verification

```bash
# Build CLI
go build -o twig ./cmd/twig

# 1. Create a task, note its id
./twig task add "Write the quarterly report"

# 2. Schedule it on today and a future day (past day to prove it's hidden)
#    From the TUI Tasks tab, press ctrl+p on the task and enter each date,
#    or use the planning tab. Use: yesterday, today, today+1.

# 3. Launch the TUI, select the task, open details:
./twig
#    Expect: "Scheduled for: <today>, <today+1>"  (yesterday NOT shown)

# 4. A task scheduled only in the past shows NO "Scheduled for:" line.
```

## Automated checks

```bash
# From repo root — CLI/TUI tests (render + wiring, no DB needed)
go test ./...

# Server tests (handler); needs DATABASE_URL for the DB-backed test
cd services/twig && DATABASE_URL=postgres://twig:twig@localhost:5432/twig?sslmode=disable go test ./...
```

## Done when

- `renderDetails` shows the line per spec (single/multiple/empty, ascending, comma-space, deduped, both styled+plain).
- Past (local) days are excluded; today is included.
- `ListScheduledDays` returns distinct, ordered, user-isolated, horizon-filtered rows.
- `go test ./...` (root) and the server tests pass.
