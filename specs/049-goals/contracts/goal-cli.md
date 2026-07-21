# Surface Contract: `twig goal` CLI + TUI Goals tab

**Feature**: 049-goals | **Status**: contract — commit before implementation

## CLI: `twig goal`

Follows `twig task` conventions exactly: bare command lists, `add`/`rm`/`mod`
subcommands, `--help` at every level, TTY-gated styling via
`internal/cli/render.go`, exit code 1 with a stderr message on failure.

```
twig goal [--all]                                 List goals (default subcommand)
twig goal add [--due <date>] [--desc <text>] <name>   Create a goal (starts incubating)
twig goal show <id>                               Goal fields + associated task subtrees
twig goal mod <id> [<name>] [--due <date>] [--desc <text>]   Edit name/description/due
twig goal state <id> <incubating|in-progress|completed|archived>   Change state
twig goal rm <id>                                 Delete a goal (tasks survive)
```

### Listing format

Grouped by state, `position` order within each group. Default shows
**Committed** then **Incubating**; `--all` appends **Completed** and
**Archived**. Each row: id, name, due date when set (same date rendering as
task listings). Group headers use the shared heading style.

Empty default list (playful, Principle IV):

```
No goals on the horizon yet — plant one with 'twig goal add <name>'.
```

### `twig goal show <id>`

Fields (name, state, due, description), then `Tasks:` followed by the
associated task subtrees rendered with the existing task-tree renderer
(association roots and their descendants, from ListTasks + client-side
effective-goal computation). When no tasks:

```
No tasks attached yet — every great goal starts as a wish.
```

### Errors

- Unknown id: `twig: goal 42 isn't in my book — 'twig goal' lists what is.` (exit 1)
- Bad state arg: lists the four valid states. (exit 1)
- Empty name on add/mod: rejected, playful nudge to provide one. (exit 1)

### Task-side association (existing commands extended)

```
twig task add --goal <goal-id> …      Create a task already associated
twig task mod <id> --goal <goal-id>   Associate (calls SetTaskGoal, not UpdateTask)
twig task mod <id> --goal none        Clear the association
```

Task detail output gains a `Goal: <name>` line when the task has an effective
goal (own or inherited). `FailedPrecondition` from nesting rules surfaces as:
`twig: that subtree already belongs to "<goal>" — clear that link first.`

### Help text

`printRootUsage` gains `goal     Long-term goals (incubate, commit, complete)`
listed **before** `task` (Goals-first mirrors the TUI tab order). `twig help
goal` prints the subcommand usage; `runHelp` gains the `goal` case.

## TUI: Goals tab

### Tab bar & startup

`tabGoals` is the first tab in the bar, which reads `Goals · Tasks · Plan ·
Report`. The TUI still opens on the Tasks tab; the existing tab-cycle
bindings include Goals (one `shift+tab` from startup).

### Layout

Two-pane (same proportions and theming as Plan/Tasks tabs): left = goal list,
right = detail of the selected goal. Shared `theme.go` styles only; no new
colors.

**Left pane** — section header per visible state group (Committed, then
Incubating; plus Completed, Archived when show-all is on), goals in `position`
order: name + due date when set. Cursor moves with `↑/k`, `↓/j` across groups.

**Right pane** — selected goal's name, state, due, description, then its
associated task subtrees (read-only tree, reusing task-tree rendering).

Empty state (no visible goals):

```
A blank canvas! Press 'n' to plant your first goal.
```

### Key bindings (list pane)

| Key | Action | Precedent |
|---|---|---|
| `n` | New goal (form: name, description, due — reuses the task edit-form component) | `n` new task |
| `e`/`enter` | Edit selected goal (same form) | `e` edit task |
| `ctrl+d` | Delete selected goal (confirm prompt names surviving tasks: "Tasks attached to it will stick around.") | `ctrl+d` delete task |
| `i` / `o` / `d` / `v` | Set state: incubating / in-progress / completed (done) / archived | new; shown in help |
| `{` / `}` | Rank higher / lower within the state group (no-op at group edge) | `{`/`}` task rank |
| `c` | Toggle showing Completed + Archived groups | `c` toggle show all |
| `a` | Add a new task attached to this goal (opens task form; created with `--goal` semantics) | — |
| `L` | Link an existing task: opens the task picker (`pickerState`, as in Plan add-task); chosen task's subtree joins the goal | picker reuse |
| `U` | Unlink: pick from the goal's association roots; clears via SetTaskGoal | — |
| `?` | Help (entries added to `keymap.go` help sections) | global |

State-change keys call `SetGoalState`; the moved goal visibly leaves/joins
groups immediately (optimistic refetch, as other tabs do). Completing a goal
shows a transient notice: `Goal achieved — take a bow! 🎉` (single emoji,
within tone rules); archiving: `Tucked away. It'll be here if you change your
mind.`

### Task views (Tasks tab + task edit)

- Task detail pane gains `Goal: <name>` when the task has an effective goal
  (inherited goals show the same line — nearest self-or-ancestor rule).
- The task edit form gains a **Goal** field: cycles `none` → each visible
  (in-progress/incubating) goal by name; saving a change calls `SetTaskGoal`.
  Tree rows are unchanged (clarification 2026-06-11: detail only).

### Hidden-state behavior

Hidden groups stay hidden across restarts only insofar as the default is
always hidden (no persistence of the toggle — matches the Tasks tab's `c`
behavior).
