# Quickstart: Manual Verification — TUI Planning Tab

Prerequisites: a running server with a provisioned user, `TODO_API_KEY` and `TODO_ADDR` set (see CLAUDE.md). Build the CLI: `cd services/todo && go build -o todo ./cmd/todo`.

Launch the TUI on a TTY: `./todo`.

## 1. Tab navigation and grid (User Story 1 — P1)

1. The TUI opens on the **Tasks** tab; a tab bar reads `Tasks │ Planning` at the top.
2. Press **Tab** → the **Planning** tab activates and today's plan renders as the day-planner grid (time rails, hour dividers, entry boxes), with the in-view day shown in the header.
3. Confirm entry boxes show `HH:MM-HH:MM <name>` with **no `[id]` prefix**.
4. Confirm a **now-marker** points at the current 15-minute slot.
5. If any entry references a completed task, confirm it renders **struck through**; events never appear completed.
6. Press **Shift+Tab** (or **Tab**) → back on **Tasks**; confirm cursor and expansion are exactly as before.
7. Cross-check: run `./todo plan` in another shell — the grid matches the TUI's (modulo the id prefix and the on-screen highlight).

## 2. Pomodoro spans both tabs

1. On the Tasks tab, start a pomodoro (`s`) on a task.
2. Switch to the Planning tab — the timer keeps counting down in the status bar and stays visible.

## 3. Now-marker auto-advances (FR-008a)

1. Stay on the Planning tab (today) with **no pomodoro running** and **no key presses**.
2. Wait until the wall clock crosses a 15-minute boundary; confirm the now-marker moves to the new slot on its own.

## 4. Build the day (User Story 2 — P2)

1. On Planning, press **a** (add task) → the task picker opens.
2. Select a task; enter a start time (e.g. `9:00`) and a duration (e.g. `1h`); submit.
   - Confirm a new box for that task appears spanning 09:00–10:00 and is highlighted.
3. Press **a** again, pick a task, enter only a start time (blank duration); submit.
   - Confirm the entry appears using the server's default duration.
4. Press **e** (add event) → enter a name (e.g. `Lunch`), start `12:00`, duration `30m`; submit.
   - Confirm the event box appears.
5. Enter an invalid time (e.g. `25:99`) in any prompt → confirm a readable error in the status bar and **no** entry created; the prompt stays open.
6. Open a prompt and press **Esc** → confirm nothing changes.

## 5. Edit in place (User Story 3 — P3)

1. With ≥2 entries, press **↓/↑** (or `j`/`k`) → the highlight moves between entries in time order.
2. **r** → rename the highlighted entry; confirm the label updates.
3. **m** → move it to a new start and/or duration; confirm it relocates.
4. **Ctrl+D** → remove it; confirm it disappears and the highlight settles on a neighbor.
5. **c** → clear from a chosen time (default = now); confirm entries at/after that time are gone and earlier ones remain.

## 6. Plan another day (User Story 4 — P3)

1. Press **]** → the header advances to tomorrow and shows that day's (separate) entries.
2. Add an entry; confirm it lands on tomorrow (cross-check `./todo plan --date <tomorrow>`).
3. Confirm **no now-marker** is shown for a non-today day.
4. Press **t** → returns to today.
5. Press **[** repeatedly → can reach past days and edit them.

## 7. Tab-switch is blocked while a modal is open (FR-023)

1. Open any prompt (e.g. press **a** to start add-task, or **r** to rename).
2. Press **Tab** → confirm the tab does **not** switch; the prompt keeps focus (Tab moves between fields where applicable).
3. Cancel (**Esc**) or submit, then **Tab** switches tabs normally.

## 8. Refresh / consistency (FR-022, FR-024)

1. On the Planning tab, in another shell run `./todo plan task <id> 15:00 30m`.
2. Press **Ctrl+R** (or switch away and back) → the externally-added entry appears.

## Automated checks

From `services/todo/`: `go test ./...` — covers grid options (`internal/cli/plan_grid_test.go`), and the planning reducer/view (`internal/tui/plan_update_test.go`, `plan_view_test.go`) plus tab dispatch in the existing `update_test.go`/`view_test.go`.
