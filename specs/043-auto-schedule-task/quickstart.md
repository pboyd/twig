# Quickstart: Auto-Schedule a Task

## What it does

In the TUI **planning tab**, highlight a task and press **`a`**. The task jumps to the earliest open slot in the day that fits it — never before 8 AM (and never in the past when you're planning today). Events are left alone.

## Try it

```bash
# Build and run the TUI (needs TWIG_API_KEY / TWIG_ADDR or ~/.config/twig/config.toml)
go build -o twig ./cmd/twig
./twig            # launches the interactive TUI on a TTY
```

1. Press `tab` to reach the **Plan** tab.
2. Add a couple of tasks/events with times (`t` / `e`) to create some gaps.
3. Add or highlight a task that has no time yet.
4. Press **`a`** — it lands in the earliest gap at/after 8 AM that fits its duration, and stays highlighted.
5. Highlight an **event** and press `a` — nothing moves; you get a playful nudge that auto-schedule is for tasks.

## Verify

```bash
# From repo root — runs the new slot-finder unit tests and the TUI handler tests
go test ./...

# Narrower, while iterating:
go test ./internal/cli/ -run AutoScheduleSlot
go test ./internal/tui/ -run AutoSchedule
```

## Acceptance walk-through (maps to spec)

| Scenario | Steps | Expected |
|---|---|---|
| Empty day (US1.4) | Empty day, highlight a 45-min task, press `a` | Task placed at 08:00 |
| Gap after a block (US1.2) | Entry 08:00–09:00, highlight a 30-min task, press `a` | Task placed at 09:00 |
| Too-small gap skipped (US1.3) | Entries 08:00–09:00 and 09:15–10:00, highlight 30-min task, press `a` | Task placed at 10:00 |
| Re-home earlier (US2.1) | Task at 14:00 with 08:00 free, press `a` | Task moves to 08:00 |
| Already in place (US2.3 / FR-011) | Task already at the earliest fit, press `a` | No change, no spurious write |
| Event guard (US3) | Highlight an event, press `a` | No move + playful notice |
| Today, afternoon (FR-003) | Plan = today, it's 14:00, highlight a task, press `a` | Placed no earlier than 14:00, never at 08:00 |
| No room left (Edge) | Day full from floor to midnight, press `a` | No change + "day's packed" notice |

## Where the code lives

- `internal/cli/plan_grid.go` — `AutoScheduleSlot(...)` (pure earliest-fit search; unit-tested)
- `internal/tui/keymap.go` — `PlanAutoSchedule` binding (`a`) + help entry
- `internal/tui/update.go` — `handlePlanKey` case: guards, floor computation, calls `AutoScheduleSlot`, dispatches the existing `movePlanCmd`
