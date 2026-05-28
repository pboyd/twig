# Phase 1 Data Model: TUI Pomodoro Integration

This feature introduces **no persisted data** and **no new wire types**. It adds transient, in-memory TUI state that is a view onto the server-authoritative pomodoro. All durable pomodoro data (`Pomodoro.start_at`, `task_id`, `complete`) already exists in `proto/task/v1/task.proto` and is unchanged.

## TUI-side state

### `activePom` (new)

Holds the TUI's view of the one running pomodoro. Stored on `Model` as `pom *activePom` (nil ⇒ no pomodoro).

| Field | Type | Source | Purpose |
|---|---|---|---|
| `taskID` | `int64` | `StartPomodoroResponse` / `GetActivePomodoroResponse` | Identifies the task the pomodoro is for; used for the cancel/complete RPCs and conflict detection. |
| `taskName` | `string` | task tree / `GetTask` | Cached label for the status-bar line and completion banner (avoids a lookup every tick). |
| `startAt` | `time.Time` | `Pomodoro.start_at` | Authoritative start time; remaining is derived as `pomodoro.Remaining(startAt, now)`. |
| `completed` | `bool` | set by completion transition | Guards "fire once" (R2), stops further ticks, and switches the status line to banner mode. |
| `banner` | `string` (or expiry timestamp) | set on completion | The transient "🍅 Pomodoro complete! · <task>" text; cleared on next keypress or timeout. |

### `Model` field changes

| Change | Field | Notes |
|---|---|---|
| ADD | `pom *activePom` | nil when idle. |
| ADD | `confirmingQuit bool` | true while the quit-confirm overlay is shown (R8). |
| REMOVE | `modePomodoro` enum value | A pomodoro is no longer a `viewMode`; it coexists with every mode. |

## Messages (Bubble Tea)

| Message | Direction | Carries | Replaces |
|---|---|---|---|
| `pomTickMsg` | tick → Update | (nothing; reads clock) | — (new) |
| `pomStartedMsg` | startPomCmd → Update | `taskID, taskName, startAt, err` | — (new) |
| `pomActiveMsg` | getActivePomCmd → Update | `*activePom or nil, err` | — (new) |
| `pomCancelledMsg` | cancelPomCmd → Update | `err` | — (new) |
| `pomCompletedMsg` | completePomCmd → Update | `taskName, err` | — (new) |
| `pomHookErrMsg` | hook runner → Update | `err` | — (new) |
| `pomBannerExpireMsg` | tea.Tick(~5s) → Update | (nothing) | — (new) |
| ~~`pomodoroRequestMsg`~~ | — | — | **removed** |
| ~~`pomodoroDoneMsg`~~ | — | — | **removed** |

## State transitions

```text
            (s on selected task)                  tick → remaining==0
   idle ───────────────────────────▶ running ───────────────────────────▶ completed
    ▲   (launch w/ active pom: auto-attach)  │                                  │
    │                                        │  (x cancel)                      │ (keypress or ~5s)
    │                                        ▼                                  ▼
    └──────────────────────────────────── idle ◀───────────────── banner cleared → idle
```

- **idle → running**: `s` on selected task, or auto-attach at launch. Sets `pom`, fires `on_start` hook (start path only), schedules the tick.
- **running → running**: each `pomTickMsg` recomputes/redraws remaining and reschedules the tick.
- **running → completed**: tick observes `remaining == 0`; calls `CompletePomodoro`, fires `on_complete`, sets `completed=true` + `banner`, schedules `pomBannerExpireMsg`, stops ticking.
- **running → idle (cancel)**: `x`; calls `CancelPomodoro`, fires `on_cancel`, clears `pom`.
- **completed → idle**: next keypress or `pomBannerExpireMsg` clears `banner` and sets `pom = nil`.
- **running → running (start conflict)**: `s` on the same task is a no-op; `s` on a different task cancels then starts (R4).

## Validation / invariants

- At most one `activePom` at a time (enforced by the server; the TUI mirrors it).
- A `pomTickMsg` is ignored when `pom == nil` or `pom.completed` (R3 / FR-014).
- Remaining time is never negative (`pomodoro.Remaining` clamps to 0).
- The quit guard applies only while `pom != nil && !pom.completed` (FR-012).
