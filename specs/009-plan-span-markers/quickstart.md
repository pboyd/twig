# Quickstart — Plan Span Markers

Manual verification after implementation.

## Run unit tests

```bash
cd services/todo
go test ./internal/cli/... -run RenderGrid
```

Expected: all table-driven cases in `plan_grid_test.go` pass, including the new cases for single-slot, two-slot, and adjacent-block scenarios.

## Render against a live plan

Requires the dev stack and a provisioned user (see `CLAUDE.md`).

```bash
make dev                                                        # start postgres + server
cd services/todo && go build -o todo ./cmd/todo
export TODO_API_KEY=<provisioned-key>
export TODO_ADDR=http://localhost:8080
./todo plan add "field day"             --start 07:45 --duration 4h45m
./todo plan add "mine deep for diamonds" --start 13:30 --duration 1h45m
./todo plan add "create an empty enchanting room" --start 15:30 --duration 15m
./todo plan
```

Expected output shape:

```
07:45 │ ┌ 1 field day
08:00 │ │
08:15 │ │
... (intermediate │ rows)
12:15 │ │
12:30 │ └
12:45 │
13:00 │
13:15 │
13:30 │ ┌ 2 mine deep for diamonds
13:45 │ │
... (intermediate │ rows)
15:15 │ └
15:30 │ ─ 3 create an empty enchanting room
```

## Verification checklist (maps to acceptance scenarios in spec.md)

- [ ] **Scenario 1**: the 4-slot+ block ("field day") shows `┌` once with name, `│` for all middle rows, `└` for the final row. Name appears exactly once.
- [ ] **Scenario 2**: the 1-slot entry ("create an empty enchanting room") shows `─` with name — not `┌` and not `└`.
- [ ] **Scenario 3**: no two entries are adjacent in this example, but verify by adding `./todo plan add "x" --start 12:30 --duration 1h` (adjacent to "field day"). Confirm the first block ends with `└` on the line above where the second block's `┌` begins.
- [ ] **Scenario 4**: the rows for 12:45, 13:00, 13:15 are blank to the right of `│`.

## Cleanup

```bash
./todo plan clear     # if the command exists; otherwise delete entries individually
podman-compose down   # stop dev stack
```
