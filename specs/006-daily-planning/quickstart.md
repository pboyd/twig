# Quickstart: Daily Planning

This walkthrough exercises every CLI subcommand of the new `todo plan` group end-to-end against a local dev server.

## 0. Apply the migration

```sh
cd services/todo
make migrate-up        # or: migrate -path db/migrations -database "$DATABASE_URL" up
```

You should see `000006_plan_entries` applied. Verify:

```sh
psql "$DATABASE_URL" -c "\d plan_entries"
```

The output should show the composite PK `(user_id, day, id)` and the three CHECKs.

## 1. Schedule a task

Assume task `5` is "Write the daily-planning data model" and has `estimate = 3, completed = 1`.

```sh
todo plan task 5 9:00am
# → entry 1
```

Default duration is `(3 - 1) * 30 = 60 minutes`. Confirm:

```sh
todo plan
```

```
09:00 │ 1  Write the daily-planning data model
09:15 │ ░
09:30 │ ░
09:45 │ ░
10:00 │
```

The grid starts at 09:00 (the first entry) and ends at 10:00 (entry 1's end). Hours before 09:00 are omitted.

## 2. Add an event with no linked task

```sh
todo plan event "Lunch" 12:00pm 45m
# → entry 2
```

```sh
todo plan
```

```
09:00 │ 1  Write the daily-planning data model
09:15 │ ░
09:30 │ ░
09:45 │ ░
10:00 │
10:15 │
10:30 │
10:45 │
11:00 │
11:15 │
11:30 │
11:45 │
12:00 │ 2  Lunch
12:15 │ ░
12:30 │ ░
12:45 │
```

The gap between entry 1's end (10:00) and entry 2's start (12:00) renders as empty rows.

## 3. Move and rename

```sh
todo plan mv 1 09:30 1h30m
todo plan rename 1 "Draft data-model.md"
```

The first entry now starts at 09:30 and runs for 90 minutes. `mv` accepts the documented duration formats; this one is `1h30m`.

## 4. Overlap rejection

```sh
todo plan event "Standup" 10:00 45m
# Error: entry would overlap entry 1 (09:30–11:00)
```

The new entry is not saved; the existing entries are untouched.

## 5. Clear the rest of the day

Suppose the local time is 13:15.

```sh
todo plan clear
# Cleared 0 entries; trimmed 0 straddling entries.
```

Nothing is cleared because every entry already ended before 13:15. Now clear from an earlier minute:

```sh
todo plan clear 12:15
# Cleared 0 entries; trimmed 1 straddling entry (entry 2 shortened to end at 12:15).
```

Entry 2 had been 12:00–12:45; it is now 12:00–12:15. No later entries existed, so `deleted_count` is 0.

## 6. View another day

```sh
todo plan --date 2026-05-22
# Plan is empty for 2026-05-22.
```

The `--date` flag overrides "today" in every subcommand.

## 7. Cleanup

Drop the migration to reset:

```sh
make migrate-down
```
