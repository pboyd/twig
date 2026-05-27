# Quickstart: Manually verify the calendar grid refinement

This recipe lets you eyeball the new rendering against the example in `spec.md`.

## Prerequisites

- The local stack is running (`make dev` from repo root).
- A provisioned user with `TODO_API_KEY` exported, and `TODO_ADDR=http://localhost:8080`.
- The CLI binary built (`cd services/todo && go build -o todo ./cmd/todo`).

## Steps

1. Create a plan for today with the example entries:

   ```bash
   ./todo plan add "a task spanning multiple hours" --start 08:00 --end 10:00
   ./todo plan add "a shorter task, truncate the name if it's too long" --start 10:00 --end 10:30
   ```

   (Substitute the actual `plan add` flags your CLI uses; the point is just to produce two adjacent entries — one multi-hour, one 30-minute.)

2. Render the plan:

   ```bash
   ./todo plan
   ```

3. Visually verify against `spec.md`:

   - The `08:00` row shows `08:00`, one space, the `▶` now-marker (if it is currently around 08:00), the left rail `├`, one light `─`, the heavy top-left corner `┏`, and the heavy top of box 1.
   - The `09:00` row inside the multi-hour box shows the light hour line on both sides of the heavy verticals (one `─` between each rail and each `┃`).
   - The `10:00` row shows a shared border: heavy T-junctions `┣` and `┫` with one light `─` of padding between each rail and the junction.
   - All non-hour rows show light `│` rails on both outer edges with one space of padding between each rail and the heavy box.
   - Below box 2 (the 10:30 entry's bottom) and through 11:00, the calendar still draws full `├─…─┤` hour lines and `│ … │` non-hour rows.

4. Render a non-today date and confirm the marker column is blank but the rail position is identical to today's rendering.

## Failure modes to watch for

- Heavy `┃` or `┏`/`┓` appearing in column 1 (means the rail was overwritten by the box — padding wasn't applied).
- The light hour line disappearing entirely on rows that contain an entry (means the grid is not always-on).
- The `▶` marker sitting immediately next to the hour text with no separating space (means the gutter spacing wasn't added).
