# Quickstart: Plan Objectives and Notes

**Feature**: 074-plan-objective-notes | **Branch**: `074-plan-objective-notes`

How to build, run, and verify this feature. Read `plan.md` for the design and
`contracts/` for the interfaces being implemented.

## Environment

```bash
# Full stack: postgres + server, migrations run on server startup
make dev

# Provision a user if you don't have an API key yet
podman exec -it twig-server ./twig-server --provision-user quickstart:hunter2

export TWIG_API_KEY=<key from provisioning>
export TWIG_ADDR=http://localhost:8080
export DATABASE_URL='postgres://twig:twig@localhost:5432/twig?sslmode=disable'
```

`DATABASE_URL` matters: the handler tests skip **silently** without it, so
`go test ./...` reports PASS while exercising nothing.

## Build loop

```bash
# 1. Proto first (Principle II) — after editing api/proto/plan/v1/plan.proto
make proto

# 2. DB layer — after editing services/twig/db/queries/plan.sql or adding a migration
cd services/twig && sqlc generate && cd ../..

# 3. Server
cd services/twig && go test ./... && cd ../..

# 4. CLI + TUI
go test ./...
go build -o twig ./cmd/twig
```

Regenerated code (`api/gen/`, `services/twig/internal/db/`) is never hand-edited.

## Migration

The new migration is `services/twig/db/migrations/000015_plan_days.{up,down}.sql`.
`make dev` applies it on server start. To drive it by hand:

```bash
make migrate-up
make migrate-down   # verify the down migration actually drops plan_days
```

Confirm it landed:

```bash
psql "$DATABASE_URL" -c '\d plan_days'
```

## Verify: CLI

```bash
./twig plan --date 2026-08-06 objective                      # prints nothing, exits 0
./twig plan --date 2026-08-06 objective 'Ship the migration'
./twig plan --date 2026-08-06 objective                      # → Ship the migration
./twig plan --date 2026-08-06 objective ''                   # clears it
./twig plan --date 2026-08-06 objective                      # prints nothing again

./twig plan objective 'Today only'                           # no --date → today
./twig plan --date 2026-08-06 objective                      # still empty; days are independent

./twig plan --help                                           # objective appears in usage
```

Scriptability check — the read must emit bare text, no label, no ANSI:

```bash
./twig plan --date 2026-08-06 objective 'Ship it' >/dev/null
test "$(./twig plan --date 2026-08-06 objective)" = 'Ship it' && echo OK
```

Round-trip with an untouched day (no entries, no row) must succeed, not error.

## Verify: TUI

```bash
./twig      # opens on Tasks; shift+tab, shift+tab reaches Planning — or tab to it
```

Objective:

1. On a day with no objective, confirm **no** objective band — the grid starts directly
   under the tab bar and is as tall as before.
2. Press `o`. The editor appears in the band position, empty.
3. Type text, press Enter. The band appears with the rendered objective.
4. Press `o` again — pre-filled. Press Esc. Nothing changed.
5. Press `o`, clear the field, press Enter. The band disappears and the grid regains the height.
6. Press `[` and `]` to move between days: each day shows its own objective.

Notes:

1. Confirm the Notes pane sits in the right column below Details, empty on a fresh day.
2. Press `n`. The whole right column becomes the notes editor.
3. Type several lines — Enter inserts newlines and does **not** save.
4. Press `ctrl+g`. `$EDITOR` opens with the draft; edit, save, quit; the text returns to
   the editor, still unsaved.
5. Press `ctrl+s`. Details returns on top, the Notes pane below shows the rendered notes.
6. Press `n`, edit, press Esc — the previous notes are intact.

Markdown and layout:

1. Set notes containing a heading, a bullet list, `**bold**`, a link, and `` `code` ``.
   Confirm they render as on the Tasks tab, and that the editor shows raw source.
2. Set a very long objective and very long notes. Resize the terminal narrow and short.
   Pane borders, the grid, the tab bar and the status line must all stay intact.
3. Run with styling off (pipe the TUI's output or use a non-TTY path) and confirm the
   plain-text fallback reads correctly.

Guards:

1. Press `t` to open the task picker, then press `o` and `n` — neither opens an editor and
   the picker is undisturbed. Same with an entry form open.
2. On the Tasks tab, `n` still creates a subtask.
3. Press `?` on the Planning tab — `o` and `n` appear in the help.

Cross-surface (FR-028):

1. Set an objective in the TUI, then read it with `twig plan --date <day> objective`.
2. Set it from the CLI, then press `ctrl+r` in the TUI and confirm the band updates.

## Test targets

| Area | Where |
|---|---|
| Migration + queries | `services/twig/internal/handler/plan_test.go` (integration, needs `DATABASE_URL`) |
| RPC behaviour: upsert, clear, trim, length limit, isolation between the two fields | `services/twig/internal/handler/plan_test.go` |
| `ListPlanEntries` populates `day` for rows and no-rows alike | `services/twig/internal/handler/plan_test.go` |
| CLI read/write/clear, empty-day silence, usage text | `internal/cli/plan_test.go` |
| Objective band shown/omitted, height, markdown, edit keys | `internal/tui/plan_view_test.go`, `internal/tui/plan_update_test.go` |
| Notes pane split, empty pane, editor takeover, `ctrl+g` routing | `internal/tui/plan_view_test.go`, `internal/tui/plan_update_test.go` |
| Background refresh never clobbers an open draft; wrong-day discarded | `internal/tui/autorefresh_apply_test.go` |
| `n` on Tasks tab unchanged | `internal/tui/update_test.go` |

Unexported helpers are reached through the existing `export_test.go` shims.

## Definition of done

- `go test ./...` passes at the repo root **and** in `services/twig/` with `DATABASE_URL` set.
- `make proto` and `sqlc generate` produce no uncommitted diff.
- The down migration cleanly drops `plan_days`.
- Every checklist item in `checklists/requirements.md` still holds.
- New user-facing strings reviewed for Principle IV tone; the CLI read path is the one
  documented exception (bare text for scripting).
