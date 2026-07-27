# Quickstart: TUI Auto-Refresh

**Feature**: 069-tui-auto-refresh | **Date**: 2026-07-27

## What changes

Two triggers are added to the TUI. Nothing about the server, the CLI, the config file, or the on-screen layout changes.

1. Entering a tab always reloads it.
2. A once-a-minute heartbeat reloads the active tab when its data is over ten minutes old, staying silent and out of the way.

## Files touched

All work is in the root module, package `internal/tui`. No changes in `api/`, `services/twig/`, or `services/twig-web/`.

| File | Change |
|---|---|
| `internal/tui/model.go` | `tasksLastLoad` on `Model`; `lastLoad` on `goalState`, `planState`, `reportState` |
| `internal/tui/update.go` | `autoRefreshTickMsg` + `autoRefreshTickCmd`; heartbeat handler; `autoRefreshEligible`; `bg` on three messages; tab-entry fixes; handler guards |
| `internal/tui/plan_update.go` | `bg`/`bgDay` on `planEntriesMsg`; entry-ID cursor anchoring; `lastLoad` stamp |
| `internal/tui/export_test.go` | shims for the new unexported fields and predicate |

## Build and test

```bash
# From repo root
go build -o twig ./cmd/twig
go test ./internal/tui/...
go test ./...
```

## Verifying by hand

The feature is mostly invisible when it works, so two of the three checks need a deliberate setup.

### Tab-switch refresh (no waiting)

```bash
make dev          # stack up
./twig            # launch the TUI
```

1. Go to the Plan tab, complete a planned task.
2. Press `shift+tab` to reach the Tasks tab.
3. The task shows as completed. Before this feature it still showed as pending.
4. Visit the Goals tab, change a goal from another terminal with `twig goal ...`, return to the Goals tab. The change is visible. Before this feature the Goals tab only ever loaded once per session.

### Background refresh (needs a second client)

```bash
./twig            # terminal 1: leave it sitting on the Tasks tab, untouched
twig task add "something new"   # terminal 2, any time
```

Wait. Within eleven minutes of the last load, terminal 1 shows the new task with no keystroke. Leave the cursor on a task in the middle of the list first and confirm it has not moved when the new data lands.

To avoid waiting ten minutes during development, temporarily lower `autoRefreshInterval` — but note that the constant is deliberately not configurable at runtime (FR-009), so this is an edit-and-rebuild, not a flag.

### Non-disruption

1. Open an edit form on the Tasks tab and leave it open past the staleness threshold. Nothing happens; your typing is intact.
2. Stop the server (`podman-compose stop server`) and leave the TUI idle for an hour. The data stays on screen and no error text appears. Press `ctrl+r` and an error *does* appear — manual refresh still reports failures.

## Review checklist

Two invariants are easy to break later and worth checking in review:

- **`autoRefreshTickCmd` has exactly two call sites** in non-test code: `Init`, and the end of its own handler. The tick never self-terminates, so a third call site permanently doubles the heartbeat rate.
- **No load handler assigns to `m.mode`.** That is what keeps an open form safe from a result that lands underneath it.
