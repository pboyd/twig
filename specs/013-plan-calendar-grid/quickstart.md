# Quickstart — Plan Calendar Grid View

End-to-end loop for someone implementing this feature.

## 1. Update the proto

Edit `services/todo/proto/plan/v1/plan.proto` and add field 7 to `PlanEntry` per `contracts/plan.proto.patch.md`. Then regenerate:

```bash
make proto
```

This updates `gen/plan/v1/`. Do not hand-edit generated files.

## 2. Update the SQL and regenerate sqlc

Edit `services/todo/db/queries/plan.sql` so the `ListPlanEntries` query LEFT JOINs `tasks` on `task_id` and projects a `completed` boolean (see `data-model.md`). Then regenerate:

```bash
cd services/todo && sqlc generate
```

## 3. Populate `completed` in the handler

In `services/todo/internal/handler/plan.go`, set `entry.Completed` from the new row column when mapping db rows to `*planv1.PlanEntry`.

## 4. Rewrite the renderer

Replace `services/todo/internal/cli/plan_grid.go` with the calendar renderer. Signature:

```go
// RenderGrid renders the day's plan as a calendar grid.
// day:    the rendered date (YYYY-MM-DD); used to decide whether to draw a now-marker.
// now:    the current wall-clock time, injectable for tests.
// width:  total terminal width; the function picks the calendar width itself.
// isTTY:  whether to emit ANSI styling for the completed-task strikethrough.
func RenderGrid(entries []*planv1.PlanEntry, day string, now time.Time, width int, isTTY bool) string
```

Wire it up at the single call site in `services/todo/internal/cli/plan.go` by passing `time.Now()`, the detected terminal width, and the existing TTY check used by `render.go`.

## 5. Tests

Replace the goldens in `services/todo/internal/cli/plan_grid_test.go`. Cover at minimum:

- Empty day → renders the default 08:00–17:00 window with no boxes.
- A single 08:00–10:00 entry (matches the mock in the spec).
- A 10:00–10:30 entry adjacent to the prior entry (shared heavy border at 10:00).
- An 11:15–12:00 entry with a wrapping label.
- A 13:00–13:15 entry plus an adjacent 13:15–13:30 entry on a single row each.
- An entry whose start sits outside business hours (window extends).
- A completed-task entry with `isTTY = false` (no SGR; label still readable).
- A now-marker on `day == today` with an injected `now` time.

Run the suite:

```bash
cd services/todo && go test ./...
```

## 6. Eyeball it

With the dev stack up (`make dev`), provision a user, schedule a couple of entries, complete one, and run `todo plan` against `localhost:8080`. Verify the rendered calendar matches the spec mockup.
