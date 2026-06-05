# Quickstart: User-Configurable Task Order

How to build, run, and verify this feature end-to-end. Assumes the dev stack from
`CLAUDE.md`.

## 1. Regenerate code after the contract change

```bash
# After editing api/proto/task/v1/task.proto (add Task.position + ReorderTask):
make proto                       # regenerates api/gen/ (Go)

cd services/twig-web && npm run gen   # regenerates web src/gen/
```

## 2. Apply the migration

`make dev` auto-runs migrations on server startup. To run manually:

```bash
export DATABASE_URL='postgres://twig:twig@localhost:5432/twig?sslmode=disable'
make migrate-up      # applies 000008_task_position
# make migrate-down  # to roll back
```

## 3. Build & run

```bash
make dev                         # postgres + server (port 8080)
go build -o twig ./cmd/twig      # CLI/TUI
cd services/twig-web && npm run dev   # SPA on http://localhost:5173
```

## 4. Manual verification

### TUI (User Story 1)

```bash
TWIG_API_KEY=... TWIG_ADDR=http://localhost:8080 ./twig
```

- Highlight a task among siblings, press `{` → it moves one position higher; press `}` → one lower.
- Highlight the first sibling, press `{` → no change, no error (boundary no-op).
- With completed-task hiding on (`c`), `{`/`}` should jump past hidden siblings so the move is always visible.
- Quit and relaunch → the new order persists.

### Web app (User Story 2)

- Open the task tree, drag a task above/below a sibling, drop it → order updates immediately.
- Reload the page → order persists.
- Drag and drop back onto the original spot (or press Esc mid-drag) → no change.
- Confirm a task cannot be dropped under a different parent (reorder-only).

### Cross-client consistency (User Story 3)

- Reorder in the TUI, then run `./twig task` (CLI list) → siblings print in the new order.
- Open/refresh the web app → same order.

## 5. Automated tests

```bash
go test ./...                         # root: TUI ranking, CLI sort-by-position
cd services/twig && go test ./...     # server: ReorderTask handler, create/reparent positions
cd services/twig-web && npm test      # web: tree sort, reorderAnchor mapping
```

Expected new coverage:
- Server: `ReorderTask` before/after, anchor validation errors, group renumber, end-position on create & reparent.
- TUI: `{`/`}` swap, boundary no-ops, hidden-sibling skip, highlight preserved.
- CLI/builder: siblings sorted by `position` then `id`.
- Web: `buildTree` sibling sort by `position`; `reorderAnchor` drop→before/after mapping.
