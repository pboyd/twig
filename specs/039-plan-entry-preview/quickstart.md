# Quickstart: Verifying the Plan Entry Preview

Manual verification of the live preview + overlap indication in the TUI Planning tab.

## Prerequisites

```bash
# From repo root — bring up server + postgres
make dev

# Provision a user if you don't have a key yet (inside the server container or
# against a local binary with DATABASE_URL set):
#   ./twig-server --provision-user me:secret

export TWIG_API_KEY=<your key>
export TWIG_ADDR=http://localhost:8080

# Build and launch the TUI
go build -o twig ./cmd/twig
./twig
```

In the TUI, press `Tab` to reach the **Planning** tab.

## Scenario A — Live preview while editing (US1)

1. Ensure the day has at least one scheduled entry (e.g. an event 10:00–11:00). Add one
   with the Add-event form if needed.
2. Select a scheduled entry and open the **Edit** form.
3. **Expect**: a box with a **dashed border** appears in the day-planner pane at the
   entry's current window — visibly distinct from the solid saved entries.
4. Change **Start** to `13:00`.
   **Expect**: the dashed preview box jumps to the 13:00 row without saving.
5. Change **Duration** to `1h30m`.
   **Expect**: the dashed box grows to span 13:00–14:30.
6. Press `Esc` (Cancel).
   **Expect**: the preview disappears; the grid shows only saved entries, unchanged.

## Scenario B — Overlap indication (US2)

1. With an existing entry at 10:00–11:00, open **Add event** (or Edit another entry).
2. Enter Name `Standup`, **Start** `10:30`, **Duration** `1h` (→ 10:30–11:30, overlaps 10:00–11:00).
   **Expect (styled terminal)**: the overlapping portion of the dashed preview (the
   10:30–11:00 rows) is rendered in **red**.
   **Expect (plain `NO_COLOR`/non-TTY)**: those rows show a `!` marker in the left gutter.
3. Change **Start** to `11:00` (→ 11:00–12:00, merely touches the 11:00 boundary).
   **Expect**: the conflict indication **clears** (touching is not overlapping).
4. Editing an existing entry over its *own* original window:
   open Edit on the 10:00–11:00 entry and leave the time unchanged.
   **Expect**: **no** conflict indication (an entry never conflicts with itself).

## Scenario C — Untimed / invalid input (FR-009)

1. Open the **Edit** form for an entry and clear the **Start** field.
   **Expect**: no dashed preview box in the timed grid (the entry would be untimed).
2. Type a nonsense Start like `99:99`.
   **Expect**: no misplaced preview box appears.

## Scenario D — Save replaces preview (US1 / FR-011)

1. Open Edit, set **Start** `09:00`, **Duration** `45m`, press `Ctrl+S`.
   **Expect**: the dashed preview is replaced by a normal solid entry at 09:00–09:45.

## Automated checks

```bash
# Renderer + TUI unit tests (no DB required)
go test ./internal/cli/... ./internal/tui/...

# Full suite
go test ./...
```

Expected: new tests in `internal/cli/plan_grid_test.go` (dashed preview runes,
conflict-row styling/marker, backward-compat with PreviewID==0) and
`internal/tui/plan_preview_test.go` (preview construction, self-exclusion, conflict
detection, touching-boundary) pass, alongside the existing suite.
