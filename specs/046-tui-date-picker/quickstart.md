# Quickstart: TUI Calendar Date Picker

**Feature**: 046-tui-date-picker

## Build & test

```bash
# From repo root — all changes are in the root CLI/TUI module
go test ./internal/tui/...
go build -o twig ./cmd/twig
```

No server, proto, sqlc, or web changes — `make dev` is only needed if you don't already
have a server running to try the TUI against.

## Try it

```bash
# Requires a running server (make dev) and credentials:
export TWIG_API_KEY=<key>          # from ./twig-server --provision-user name:password
export TWIG_ADDR=http://localhost:8080

./twig          # launches the TUI on a TTY
```

1. Highlight a task, press `enter` to edit (or `ctrl+n` for a new task).
2. `tab` to the **Due** or **Snooze until** field.
3. Press `ctrl+g` — the calendar unfolds beneath the field.
4. Navigate: arrows/`hjkl` (day/week), `[`/`]` (month), `{`/`}` (year), `t` (today).
5. `enter` writes the date into the field; `esc` backs out without touching it.
6. `ctrl+s` saves the task as usual.

## Verify the key scenarios

| Check | Expectation |
|---|---|
| Open on a field containing `2026-07-04` | Calendar opens with July 4 2026 selected |
| Open on an empty/garbled field | Calendar opens on today |
| `esc` after navigating | Field text unchanged |
| Due field had `2026-06-10T15:00:00Z`, pick June 12 | Field becomes `2026-06-12T15:00:00Z` |
| Snooze pick | Field gets plain `YYYY-MM-DD` |
| `]` from Jan 31 | Lands on Feb 28/29, never an invalid date |
| Typing dates with calendar closed | Behaves exactly as before the feature |
