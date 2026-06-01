# Quickstart: Untimed Plan Entries — Manual Verification

Prereqs: stack running (`make dev`), a provisioned user, `TWIG_API_KEY`/`TWIG_ADDR` set, and a built CLI (`go build -o twig ./cmd/twig` from `services/twig/`). Assume at least one task exists; note its `<task_id>`.

## Automated checks first

```bash
cd services/twig
go test ./...        # all packages green, including handler/cli/tui
```

## CLI — untimed add & list (US1)

```bash
# Untimed, default duration
twig plan task <task_id>
#   → grid prints; an "Untimed" section above it lists the new entry with a duration and no time.

# Untimed with an explicit duration via the null sentinel (any casing)
twig plan task <task_id> null 45m
twig plan task <task_id> NULL 45m      # case-insensitive

# Still works the old way (timed)
twig plan task <task_id> 09:00 30m     # appears on the grid, not the untimed section
```

## CLI — schedule / unschedule (US3)

```bash
twig plan                       # note an untimed entry's id, say 3
twig plan mv 3 10:30            # schedules it → moves to the grid at 10:30
twig plan mv 3 null            # unschedules → back to the untimed section, duration preserved
twig plan mv 3                  # also unschedules (start omitted)
```

## CLI — events stay timed (edge case)

```bash
twig plan event "Standup"       # → rejected: events need a time (friendly error), exit non-zero
twig plan event "Standup" 09:15 # → succeeds, appears on the grid
```

## TUI — untimed pane, navigation, editing (US2 + US3)

```bash
twig                            # launch TUI; Tab to the Planning view
```

- A **top-left pane** shows untimed entries as dark boxes — **one line per 15 minutes** of duration — above the day grid. Confirm it matches the grid's box styling.
- On a day with **no** untimed entries, the pane is **absent** and the grid uses the full area.
- Press ↑/↓: the highlight flows in **one unified cycle** through the untimed entries first, then into the grid, and wraps — **no** focus-switch key.
- Highlight an untimed entry, press Enter (edit): change the name/duration → applies to that entry only.
- In edit, type a time into **Start** → the entry moves onto the grid. Edit again and type `null` into Start → it returns to the untimed pane (duration kept).
- Press `t` (add task), pick a task, leave **Start** blank → creates an untimed entry.

## TUI — send from Tasks tab (US4)

- On the **Tasks** tab, highlight a task and press **`p`** → an untimed entry for it lands on **today's** plan (switch to Planning to confirm); a playful confirmation shows.
- Highlight a task and press **`ctrl+p`** → a date prompt appears **pre-filled with tomorrow**. Press Enter → untimed entry on tomorrow's plan. Try an invalid date (e.g. `2026-13-40`) → no entry, friendly error. Press Esc at the prompt → nothing added.

## Acceptance mapping

| Story | Covered by |
|-------|-----------|
| US1 — capture untimed | CLI add & list section |
| US2 — TUI pane & manage | TUI untimed-pane section |
| US3 — schedule/unschedule | CLI mv + TUI edit sections |
| US4 — send from Tasks tab | TUI Tasks-tab section |
| Events must be timed | CLI events edge-case section |
