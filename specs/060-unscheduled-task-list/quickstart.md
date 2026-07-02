# Quickstart: Verifying "Unscheduled Tasks as a List"

Presentation-only change to how untimed plan entries render in the Plan tab (and `twig plan` CLI).

## Build & test

```bash
# From repo root
go build -o twig ./cmd/twig
go test ./...            # cli + tui render tests, including updated untimed assertions
```

## Automated verification (primary)

The contract in `contracts/untimed-list-rendering.md` is enforced by:

- `internal/cli/plan_grid_test.go` — asserts `RenderUntimed` emits `☐`/`☑ name` rows (styled),
  name-only rows (plain), one line per entry, `""` for empty input, and selection highlighting via
  `GridOptions{SelectedID, SelectionStyle}`.
- `internal/tui/plan_view_test.go` — asserts the Plan tab composes the untimed list above the
  divider and grid, and that a no-unscheduled day is unchanged (FR-006).

Run just these while iterating:

```bash
go test ./internal/cli/ -run Untimed
go test ./internal/tui/ -run Plan
```

## Manual smoke test (TUI)

Requires a running server + provisioned user (see CLAUDE.md → Dev Environment).

```bash
export TWIG_API_KEY=...   TWIG_ADDR=http://localhost:8080
./twig
```

1. Go to the **Plan** tab.
2. Ensure the day has at least one **unscheduled** task and one **scheduled** block.
   (Add an unscheduled task via the plan flow if needed.)
3. Confirm:
   - [ ] Unscheduled tasks appear as `☐ name` / `☑ name` rows (not boxes) above the grid. *(FR-001, FR-004)*
   - [ ] A single divider line separates the list from the grid. *(FR-003)*
   - [ ] Selecting an unscheduled row highlights it; moving selection flows between list and grid as before. *(FR-005)*
   - [ ] Completing a selected unscheduled task flips ☐→☑ with the completed styling. *(FR-005)*
   - [ ] Scheduling an unscheduled task (assign a time) moves it onto the grid; it leaves the list. *(FR-007)*
   - [ ] Reordering unscheduled tasks and opening details behave exactly as before. *(FR-005)*
   - [ ] A day with **no** unscheduled tasks looks identical to before (no header, no empty list). *(FR-006)*
   - [ ] With many unscheduled tasks, the list grows and the grid shrinks without breaking layout. *(FR-008)*

## Manual smoke test (CLI, shared renderer)

```bash
./twig plan            # untimed section renders as a checkbox/name list (TTY) consistent with the TUI
./twig plan | cat      # plain output: name-only rows, no glyphs, pipe-clean
```

## Web app

No change expected. `services/twig-web` Plan view is explicitly out of scope (FR-009); do not modify.
