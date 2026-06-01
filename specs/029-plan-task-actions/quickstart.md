# Quickstart: Plan Task Actions — Manual Verification

Verifies completing/uncompleting a linked task and starting a pomodoro for a linked task from the TUI planning tab, plus the event/empty no-op cases.

## Prerequisites

```bash
# Stack up (postgres + server), with a provisioned user's TWIG_API_KEY / TWIG_ADDR exported.
make dev
cd services/twig && go build -o twig ./cmd/twig
```

Seed: have at least one incomplete task and send it to today's plan (Tasks tab → `p`), plus one **event** on today's plan (planning tab → `e`). For FR-004, send the same task to today's plan as both a timed and an untimed entry.

Launch the TUI on a TTY and switch to the planning tab (`tab`):

```bash
./twig
```

## US1 — Complete a linked task from the plan (P1)

1. Move the selection to a task-linked entry for an **incomplete** task. Press `space`.
   - **Expect**: a playful confirmation in the status line; the entry redraws as **completed** in place (stays on the plan).
2. Switch to the Tasks tab (`tab`). Find that task (toggle "show completed" with `c` if needed).
   - **Expect**: the task shows **completed** there too — the change persisted (acceptance US1-3).
3. Back on the planning tab, select the same (now completed) entry and press `space` again.
   - **Expect**: the task **un-completes**; the entry redraws as incomplete (toggle, FR-002).
4. **FR-004 (multiple entries, same task)**: with the task linked by two entries on the day, complete it from one entry.
   - **Expect**: **both** entries redraw as completed after the reload.

## US2 — Start a pomodoro for a linked task from the plan (P2)

1. Select a task-linked entry and press `s`.
   - **Expect**: a pomodoro starts for that task; the running-pomodoro indicator appears.
2. Switch to the Tasks tab.
   - **Expect**: the **same** pomodoro is shown running (acceptance US2-3).
3. With a pomodoro already running, select another linked entry and press `s`.
   - **Expect**: behavior matches starting a pomodoro from the Tasks tab while one runs (no new, divergent rule).
4. Cancel with `x` when done.

## Edge cases

1. **Event entry**: select the event (no linked task). Press `space`, then `s`.
   - **Expect**: each is a no-op with a playful "that's an event" notice; no task/timer state changes.
2. **Empty plan**: navigate (`[`) to a day with no entries. Press `space`, then `s`.
   - **Expect**: nothing happens; no error.
3. **Help**: press `?` on the planning tab.
   - **Expect**: the help lists `space` toggle-complete and `s` start-pomodoro alongside the existing planning keys.

## Automated checks

```bash
cd services/twig && go test ./internal/tui/...
```

- New table-driven `tea.KeyMsg` tests cover: incomplete→complete, complete→uncomplete, event no-op (both keys), empty-plan no-op (both keys), and pomodoro start on a linked entry.
- Existing planning-tab tests (navigation, add/edit/move/remove, send-to-plan) remain green — no regression to current behavior.
