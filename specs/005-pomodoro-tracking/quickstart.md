# Quickstart — Pomodoro Tracking (005)

End-to-end smoke for the feature, assuming the rest of the project is already built and running (server on `:8080`, database migrated through 000004, a user logged in with `TODO_API_KEY` exported, at least one task created).

## 1. Apply the migration

```sh
cd services/todo
make migrate-up   # runs golang-migrate; should report applying 000005_pomodoros
```

Verify:

```sh
psql "$TODO_DB_URL" -c '\d tasks' | grep estimate
psql "$TODO_DB_URL" -c '\d pomodoros'
```

You should see `estimate` on `tasks` and the `pomodoros` table with the partial unique index.

## 2. Set an estimate

Assuming task id 1 exists:

```sh
todo task pom estimate 1 3
todo task list           # 'pomodoros: 0/3' shown next to task 1
todo task pom estimate 1 11
# error: estimate must be between 0 and 10; tasks larger than 10 pomodoros must be broken down further
```

## 3. Run a pomodoro to completion

```sh
todo task pom start 1
# Countdown UI opens:
#
#   Task: <task 1 name>          Estimate: 3   Completed: 0
#
#   24:59 remaining
#
#   Press 'c' to cancel, 'q' to quit (keeps timer running)
#
# After 25 minutes the UI closes; the pomodoro is marked complete server-side.
```

Verify:

```sh
todo task pom status      # 'No active pomodoro.'
todo task list            # 'pomodoros: 1/3' shown next to task 1
```

## 4. Quit-and-resume

```sh
todo task pom start 1     # ...press 'q' after a few seconds
todo task pom status      # shows active pomodoro on task 1 with remaining time
todo task pom resume      # re-attaches the countdown for the still-active pomodoro
```

## 5. External cancellation

Terminal A:

```sh
todo task pom start 1     # countdown running
```

Terminal B:

```sh
todo task pom cancel      # cancels the active pomodoro
```

Within 1–2 seconds Terminal A prints a "pomodoro canceled externally" message and exits with status 0.

## 6. `--exec` on completion

```sh
todo task pom start 1 --exec 'notify-send "Break!"'
# When the 25 minutes elapse, the pomodoro is marked complete and
# notify-send runs (stdout/stderr inherited). If notify-send fails, the
# CLI prints a warning naming the exit code but still exits 0.
```

## 7. Cascade delete

```sh
todo task pom start 2
todo task rm 2            # also removes every pomodoros row for task 2
psql "$TODO_DB_URL" -c 'SELECT count(*) FROM pomodoros WHERE task_id = 2;'   # 0
todo task pom status      # 'No active pomodoro.'  (the active one was deleted by cascade)
```

## 8. Single-active invariant

```sh
todo task pom start 1
todo task pom start 2
# error: you already have an active pomodoro on task 1 (cancel it or resume it first)
```

## What "done" looks like

- All API calls return in well under one second (SC-002).
- External cancel terminates a running countdown within 5 seconds (SC-005).
- The database never contains two rows with `end_at IS NULL` for the same `user_id` (SC-003) — the partial unique index makes this impossible.
- Out-of-range estimates are always rejected with the "break it down further" message (SC-006).
