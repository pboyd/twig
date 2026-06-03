# Quickstart: Verifying Planner Vertical Space

Manual and automated checks for the height-aware Planning grid.

## Prerequisites

```bash
cd services/twig
go build -o twig ./cmd/twig
# Server + a provisioned user running (see CLAUDE.md "Dev Environment").
export TWIG_API_KEY=...    # from --provision-user
export TWIG_ADDR=http://localhost:8080
```

Seed today with a light morning-only schedule plus one untimed item, e.g.:

```bash
./twig plan add --at 09:00 --for 60 "Standup + email"
./twig plan add --at 11:00 --for 30 "1:1"
./twig plan add "Read design doc"          # untimed
```

## Scenario A — Fill empty space (US1)

1. Make the terminal **tall** (e.g., 50+ rows). Launch the TUI: `./twig`.
2. Open the **Planning** tab.
3. **Expect**: the grid draws hours past 17:00 (e.g., down toward 22:00–23:00),
   filling the pane. No large blank gap below the last drawn hour.
4. Shrink the terminal height and watch the grid show **fewer** later hours.

✅ Pass: later hours appear beyond 5 PM and track the window height (SC-001, SC-005).

## Scenario B — Anchor to now when tight (US2)

1. Use a **short** terminal (e.g., ~12 rows) so the full day can't fit.
2. Ensure it is **today** and `now` is mid-afternoon; have an entry later today
   (`./twig plan add --at 16:00 --for 30 "Sync"`).
3. Open Planning.
4. **Expect**: untimed entries render first; the timed grid's first row is the
   block containing the current time; morning hours are **not** shown; the later
   entry is visible.

✅ Pass: grid starts at the now-block, upcoming entry reachable (SC-002).

## Scenario C — Untimed stays visible (US3)

1. Short terminal, add several untimed entries.
2. **Expect**: all untimed boxes + separator render before any timed row; the
   timed grid is the part that shrinks.

✅ Pass: untimed fully visible before timed rows (SC-003).

## Scenario D — Non-today day

1. Navigate to a **past or future** day on a short terminal.
2. **Expect**: grid starts at the default top (08:00 / earliest entry), trimmed to
   fit — no "anchor to now" (there is no now in that day).

✅ Pass: FR-005 honored.

## Scenario E — CLI parity (FR-008)

```bash
./twig plan            # non-TTY / piped also fine:
./twig plan | cat
```

**Expect**: identical 08:00–17:00 (entry-extended) output as before — no extra
hours, no anchoring.

## Automated regression

```bash
cd services/twig
go test ./...
```

Key suites:
- `internal/cli` — `TestRenderGrid_EmptyDay`, `*_WindowExtension*` (CLI parity),
  plus new `GridWindow` / override tests.
- `internal/tui` — fill mode, anchor mode, non-today, tiny-terminal grid tests.

✅ Pass: all green; existing default-window assertions unchanged (SC-004).
