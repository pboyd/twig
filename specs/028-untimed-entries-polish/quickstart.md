# Quickstart: Untimed Plan Entries — Polish & Bugfixes

Manual verification for all four fixes. Assumes a provisioned user, the server running (`make dev`), and `TWIG_API_KEY` / `TWIG_ADDR` set. Pick an existing task id (call it `T`) and a free day (e.g. today).

## Build & test

```bash
cd services/twig
go test ./...
go build -o twig ./cmd/twig
```

## Fix 2 — No duplicate untimed entries (CLI, server-enforced)

```bash
# First untimed add succeeds.
./twig plan task <today> T            # no start → untimed entry created

# Second untimed add for the SAME task/day is rejected (no entry created).
./twig plan task <today> T            # expect a friendly FailedPrecondition error

# A TIMED add for the same task/day still works.
./twig plan task <today> T 09:00      # timed entry created alongside the untimed one

# Clearing a timed entry's start onto an existing untimed duplicate is rejected.
./twig plan mv <id-of-09:00-entry> null   # expect rejection; entry keeps 09:00

# Verify the day: exactly one untimed entry for T, plus the timed one.
./twig plan show <today>
```

Expected: at most one untimed entry for `T` on the day, at every step; rejections carry a warm, actionable message.

## Fix 3 — Untimed entries render like gridded entries (CLI)

```bash
# Create two adjacent multi-line untimed entries on a day.
./twig plan task <day> T   60         # 60-min untimed (multi-row)
./twig plan task <day> T2  45         # 45-min untimed for a different task
./twig plan show <day>
```

Check the untimed pane output:
- [ ] Each entry's **title sits inside the box** — it does not overlap or replace the top border (`┏━━━┓` present above the title).
- [ ] The two entry boxes **join** — they share a `┣━━━┫` boundary like adjacent grid entries, rather than each drawing its own `┗┛`/`┣┫`.
- [ ] **Colors are consistent on every line**, including the last line of each entry (compare to a gridded entry of the same duration/state — they should be indistinguishable).

## Fix 4 — Separation between untimed pane and grid (CLI + TUI)

```bash
./twig plan show <day>     # day WITH untimed entries
```
- [ ] A clear visual separation appears between the untimed pane and the day planner grid.

```bash
./twig plan show <empty-day>   # day WITHOUT untimed entries
```
- [ ] No untimed pane and no separator; the grid uses the full area (unchanged from before this feature).

## Fix 1 — Status-bar feedback on send (TUI)

```bash
./twig            # launch TUI, stay on the Tasks tab
```
- [ ] Highlight a task, press `p` → a **confirmation** appears in the status bar naming the task and "today".
- [ ] Press `ctrl+p`, accept/edit the date, confirm → a confirmation names the task and that **date**.
- [ ] Press `p` again on a task that already has an untimed entry today → the status bar shows the **duplicate rejection** message (not a success), and no second entry is added (switch to the Planning tab to confirm).
- [ ] Messages are in the app's warm/playful tone and persist until the next action (auto-dismiss is intentionally out of scope).

## Regression sweep

- [ ] Timed entries: add / move / rename / remove / clear behave exactly as before.
- [ ] Planning view unified Up/Down highlight still flows untimed → grid and wraps.
- [ ] `go test ./...` is green.
