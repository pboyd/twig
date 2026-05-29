# Quickstart: Verify Planning Tab Refinements

Manual verification walkthrough. All changes are TUI client-side; no server/DB changes. Automated coverage runs with `go test ./...` from `services/todo`.

## Prerequisites

```bash
cd services/todo
go build -o todo ./cmd/todo
# Server + a provisioned user as per CLAUDE.md (TODO_API_KEY, TODO_ADDR).
# Seed a day with a couple of entries (one task-linked, one event) and a task that has sub-tasks.
./todo            # launches the TUI on a TTY
```

Switch to the Planning tab with `tab`.

## 1. Cell-only selection highlight (US4)

- Move the selection (`↑`/`↓`) between entries.
- **Expect**: the selected entry's text block shows **bold white text on a blue background** — identical to selecting a row on the Tasks tab.
- **Expect**: the hour marker (e.g. `07:00`) and the box border around the entry are **not** recolored — only the inner text cell changes.
- Compare directly: press `tab` to the Tasks tab, select a task — the highlight should look the same.

## 2. Edit an entry with Enter, in the right pane (US1, US3)

- On the Planning tab, select an entry and press `enter`.
- **Expect**: an **Edit** form opens in the **right pane** (grid still visible on the left), prefilled with the entry's **Name**, and **Start** / **Duration** fields.
- Change the name and the start time; press `ctrl+s` (or `enter`).
- **Expect**: the grid updates — the entry shows the new name at the new time; selection stays on it.
- Press `enter` again, change nothing, `esc`.
- **Expect**: no change; the right pane returns to the read-only **Details** view.
- Press `enter` on an empty slot / empty day.
- **Expect**: nothing happens, no error.

## 3. Add task with `t`, add event with `e` — right pane (US3, US5)

- Press `t`.
- **Expect**: the task picker opens in the **right pane** (grid still visible), showing tasks as a tree including sub-tasks. Select a sub-task, enter a time, save.
- **Expect**: the entry appears on the grid.
- Press `e`, fill the event form (right pane), save.
- **Expect**: the event appears; grid stayed visible the whole time (no full-screen takeover).

## 4. Jump-to-today moved to `.` (US5)

- Press `]` a few times to move to a future day.
- Press `.`.
- **Expect**: the grid returns to today (the day title shows "(today)").
- Press `t` on a non-today day.
- **Expect**: it opens the add-task picker (does **not** jump to today).

## 5. Clear is gone (US6)

- Press `c` on the Planning tab.
- **Expect**: nothing happens (no "clear from time" form).
- Open help (`?`).
- **Expect**: no `clear` entry; help lists `t add task`, `enter edit entry`, `. today`, `ctrl+d remove entry`, and no `rename`/`move entry`.
- Remove an entry with `ctrl+d`.
- **Expect**: per-entry delete still works.

## 6. Enter edits on the Tasks tab (US2)

- Switch to the Tasks tab. Select a task, press `enter`.
- **Expect**: the task edit form opens (right pane, as before).
- Press `e`.
- **Expect**: the edit form does **not** open.
- Open help — **Expect**: edit is advertised as `enter`, not `e`.

## 7. Degradation checks

- Resize the terminal narrow while a Planning form is open.
- **Expect**: grid/form split collapses without corrupting either pane (same as Tasks tab).
- Run with styling off (e.g. pipe / non-TTY paths covered by tests): selection and forms render as plain text with no stray escape sequences.

## Automated

```bash
cd services/todo && go test ./...
```

Key suites: `internal/cli` grid-selection tests (cell-only styling, width invariance, `SelectedID==0` golden), `internal/tui` `plan_view_test.go` / `view_test.go` (right-pane form layout), `plan_update_test.go` (Enter→edit, `t`/`.` rebinds, merged submit composes rename+move, clear removed), `update_test.go` (Tasks `enter` edits, `e` no longer edits).
