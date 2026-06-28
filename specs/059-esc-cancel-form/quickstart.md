# Quickstart: Esc Cancels TUI Forms

## What changed

`Esc` now cancels any TUI form. If you've typed or edited anything that isn't
saved, it first asks `Toss out your unsaved edits? [y]es  [n]o — keep editing`.
If you haven't changed anything, it just backs out.

## Build & run

```bash
# From repo root
go build -o twig ./cmd/twig
./twig            # launches the TUI on a TTY
```

(Server must be reachable; see CLAUDE.md for `TWIG_API_KEY` / `TWIG_ADDR`.)

## Manual verification

1. **Clean cancel (new task)**: On the Tasks tab press `n` to open the new-task
   form, type nothing, press `Esc`. → Form closes immediately; no task created.
2. **Dirty cancel — keep**: Press `n`, type a name, press `Esc`. → Discard prompt
   appears. Press `n` (or `Esc`). → Back in the form with your text intact.
3. **Dirty cancel — discard**: Repeat, then press `y` at the prompt. → Form closes;
   no task created; cursor is where it was.
4. **Edit, no change**: Highlight a task, press `e`, change nothing, press `Esc`. →
   Closes immediately, no prompt, nothing updated.
5. **Edit, revert**: Press `e`, change the name, change it back to the original,
   press `Esc`. → Closes immediately (no prompt), nothing updated.
6. **Calendar layering**: Press `e`, focus the Due field, press `ctrl+g` to open the
   calendar, press `Esc`. → Calendar closes, form stays. Press `Esc` again. → Clean/
   dirty cancel rules apply.
7. **Goals tab**: Press `N`/`e` to create/edit a goal, type, `Esc` → prompt; same
   behavior. Same for `a` (new task on a goal).
8. **Planning tab**: Open a timed-task / event / edit form, type a value, `Esc` →
   prompt; with no changes, `Esc` cancels immediately.

## Automated tests

```bash
go test ./internal/tui/...
```

Expect new/updated cases in `edit_test.go`, `update_test.go`, and
`plan_update_test.go` covering: clean vs dirty `Esc`, revert-to-original, whitespace-
only, discard confirm/decline (Tasks, Goals, Plan), and calendar-overlay layering.

## Out of scope (unchanged)

- Status-update compose uses your external `$EDITOR`; cancel/discard there is the
  editor's job (e.g. `:cq`).
- Date prompt, move/link/unlink pickers, and existing `[y]es/[n]o` confirmations
  keep their current immediate-`Esc` behavior.
