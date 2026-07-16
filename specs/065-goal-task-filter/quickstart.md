# Quickstart: Goal Task Filter Shortcut

**Feature**: 065-goal-task-filter | **Date**: 2026-07-16

How to build, run, and manually verify this feature. Automated coverage is specified in
[plan.md](./plan.md) Change 3; this walkthrough is the human check that the shortcut actually feels right.

## Build and run

```bash
# From repo root — the stack must be up (postgres + server)
make dev

# Unit tests (no database required)
go test ./...

# Focused on this feature
go test ./internal/tui/

# Build and launch the TUI
go build -o twig ./cmd/twig
./twig                       # no args → TUI; opens on the Tasks tab
```

The CLI needs `TWIG_API_KEY` and `TWIG_ADDR` (defaults to `http://localhost:8080`), from the environment
or `~/.config/twig/config.toml`. See `CLAUDE.md` for provisioning a user.

## Fixture setup

Set this up once via the Goals tab (`shift+tab` from Tasks), so each scenario below has something to bite
on:

| Goal | Tasks needed |
|------|--------------|
| **Goal A** | A task with at least one **nested subtask** (verifies transitive matching), plus one **completed** task (verifies show-all) |
| **Goal B** | At least one open task, unrelated to Goal A (verifies scoping excludes it) |
| **Goal C** | **No tasks at all** (verifies the empty state) |

Attach tasks to a goal with `n` (new task on the goal) or `L` (link an existing task) from the Goals tab.

## Verification scenarios

### 1. The jump — FR-001, FR-002, FR-005

1. `shift+tab` to the Goals tab.
2. Put the cursor on **Goal A**.
3. Press `ctrl+t`.

**Expect**: You land on the Tasks tab. Only Goal A's tasks are listed — Goal B's are gone. The nested
subtask is shown, not just the directly attached task. The filter bar reads `^goal_id=<n>` for some `n`
you never had to look up.

### 2. The filter is ordinary — FR-003, US2

With Goal A's filter applied:

1. Press `/` → the input reopens **pre-filled** with `^goal_id=<n>`, ready to edit.
2. Press `esc` to leave the input, then `esc` in the list → the full task list returns.
3. Press `ctrl+t` from Goal A again, then edit the filter to `^goal_id=<n> AND completed=false` and press
   Enter → it works like any hand-typed expression.

**Expect**: Nothing about the shortcut's filter behaves specially. It is text in the normal filter bar.

### 3. Show-all still governs completion — FR-009

With Goal A's filter applied and show-all **off**:

1. Confirm Goal A's completed task is **hidden**.
2. Press `c` (toggle show all).

**Expect**: The completed task **appears**, still scoped to Goal A. Press `c` again and it hides. This is
the check that the expression didn't smuggle in `completed=false`.

### 4. Replacement, not merge — FR-004

1. On the Tasks tab, press `/`, type any other filter (e.g. `completed=false`), press Enter.
2. `shift+tab` to Goals, cursor on **Goal B**, press `ctrl+t`.

**Expect**: The filter bar reads only `^goal_id=<b>`. The previous expression is gone, not `AND`-ed on.

### 5. Empty goal — FR-004 edge case

Cursor on **Goal C** (no tasks), press `ctrl+t`.

**Expect**: The Tasks tab opens with Goal C's filter applied and shows the "no tasks match" state. Not a
blank screen, not an error, not a crash.

### 6. Empty goals list — FR-006

With no goals at all (or cursor past the end), press `ctrl+t` on the Goals tab.

**Expect**: Nothing happens. You stay on Goals. No error flashes.

### 7. Inert in sub-modes — FR-007

From the Goals tab, press `ctrl+t` while each of these is open:

- A goal edit form (`e`)
- A new-task form (`n`)
- The link-task picker (`L`)
- Status history (`s`)
- The help overlay (`?`)

**Expect**: No tab switch in any case. The open view keeps handling keys — in a text field, `ctrl+t` does
whatever the input already did.

### 8. Plan tab is untouched — FR-008 (regression guard)

1. `tab` to the Plan tab with at least one task entry scheduled.
2. Cursor on a task entry, press `ctrl+t`.

**Expect**: The **old** behavior — jump to the Tasks tab with the cursor on that entry's task, **no filter
applied**. This is the shared-key regression check; treat a failure here as a blocker.

### 9. Help is discoverable — FR-010

On the Goals tab, press `?`.

**Expect**: `ctrl+t` — "show goal's tasks" is listed among the Goals shortcuts.

## What "done" looks like

- `go test ./...` passes from the repo root.
- All nine scenarios behave as described, scenario 8 especially.
- The filter bar always shows the expression a user could have typed — because that is precisely what the
  shortcut typed for them.
