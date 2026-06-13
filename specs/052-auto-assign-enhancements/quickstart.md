# Quickstart: Auto-Assign Enhancements

## What changed

Pressing `a` in the TUI planning tab now:

1. Places tasks on tidy 15-minute boundaries (`:00/:15/:30/:45`).
2. Bumps a task that is already in its earliest slot forward into the next free gap, instead of doing nothing.

## Run the tests

```bash
# From repo root
go test ./internal/cli/ ./internal/tui/
# or everything
go test ./...
```

Key tests:
- `internal/cli/plan_grid_test.go` — `AutoScheduleSlot` boundary rounding + `NextGapFloor`.
- `internal/tui/plan_update_test.go` — `a`-handler floor down-rounding, bump-to-next-gap, and no-room-when-no-later-gap.

## Try it by hand

```bash
go build -o twig ./cmd/twig
TWIG_API_KEY=... TWIG_ADDR=http://localhost:8080 ./twig   # launches the TUI
```

In the **Plan** tab:

1. **Quarter-hour rounding (today)**: at a non-:00/:15/:30/:45 time (e.g. 9:03) with an open morning, highlight a task and press `a`. It lands at **9:00** — the boundary at or just before now — not 9:03.
2. **Rounding after an entry**: add an entry that ends off-boundary (e.g. 9:07), highlight a task that fits after it, press `a`. It lands at **9:15** (rounded up, no overlap).
3. **Bump**: highlight a task already sitting in its earliest slot (e.g. 8:00) with a later free gap after some entry, press `a`. It jumps forward into that next gap rather than staying put. Press `a` again to keep stepping it forward.
4. **No-room bump**: highlight a task in the last free stretch of an otherwise full day and press `a`. It stays put and shows the "Day's packed…" notice.

## Regression checks (should be unchanged from feature 043)

- Highlight an **event** and press `a` → nothing moves; event notice.
- A **packed day** → "Day's packed…" notice.
- A task placed late (e.g. 14:00) with a clean earlier slot open → `a` moves it **earlier**.
- After any successful move, the **highlight follows** the task.
