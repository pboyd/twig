# Quickstart: Hide Completed Unscheduled Plan Entries

How to build, exercise, and verify this feature. No server, proto, or DB changes are involved — only the two client surfaces.

## Prerequisites

- Stack running for manual checks: `make dev` (PostgreSQL + server on :8080).
- A provisioned user / `TWIG_API_KEY` + `TWIG_ADDR` for the CLI/TUI (see `CLAUDE.md`).

## Build & test

```bash
# TUI/CLI module (repo root)
go build -o twig ./cmd/twig
go test ./...

# Web app
cd services/twig-web
npm test
npm run build
```

## Manual verification — TUI

1. Launch the TUI (`./twig` on a TTY) and open the Planning tab.
2. Add an **untimed** task entry for the day (`p` from Tasks, or add without a start time) so the untimed pane shows it.
3. Highlight the untimed entry and press the complete key.
   - ✅ Expected: the entry stays in place, **crossed out**, and **still highlighted** (it does not vanish).
4. Move the highlight (Up/Down) to another entry.
   - ✅ Expected: the completed untimed entry is now **gone** from the plan.
5. Repeat step 3, then instead switch tabs (or change the day and come back).
   - ✅ Expected: on return, the completed untimed entry is gone (retained only while it was the active highlight).
6. Complete an untimed entry, and **before** navigating away, reopen it (uncomplete).
   - ✅ Expected: it is no longer crossed out and continues to display normally.
7. Add a **timed** entry, complete its task.
   - ✅ Expected: it remains visible on the grid (crossed out) — timed entries are never hidden.

## Manual verification — Web

1. `cd services/twig-web && npm run dev`, open http://localhost:5173, log in, open the day planner.
2. Ensure the day has an untimed task entry; confirm it appears under "Untimed".
3. Complete that task (from the task list / detail), then return to / refresh the planner.
   - ✅ Expected: the untimed entry no longer appears.
4. Confirm a completed **timed** entry still appears (marked done) on the same day.
5. Uncomplete the task and return to the planner.
   - ✅ Expected: the untimed entry reappears.

## Automated test coverage to add

- **Web** (`services/twig-web/src/lib/planView.test.ts`): `groupPlan` excludes completed untimed entries; keeps completed timed entries; `isEmpty` is true when the only untimed entries are completed and there are no timed entries.
- **TUI** (`internal/tui/*_test.go`):
  - Completing the highlighted untimed entry keeps it in the rendered plan (crossed out) while `pendingComplete` is set.
  - Navigating (cursor move / day change / tab switch) clears `pendingComplete` and removes the entry from the rendered plan.
  - A completed untimed entry that is **not** pending is absent from the rendered plan after load.
  - Timed completed entries and events are never filtered.
  - Uncompleting a pending entry restores normal (non-struck) rendering.

## Done when

- All acceptance scenarios in `spec.md` (US1–US3) pass manually and via the automated tests above.
- `go test ./...` and `npm test` are green; `npm run build` succeeds.
