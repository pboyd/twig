# Contract: TUI View Model

This document pins down the in-memory shape that `internal/tui` consumes from the existing ConnectRPC services. It is the contract between "data fetched from the server" and "what the TUI renders." If the protobuf shape changes, this document must change with it.

## Inputs from the server

The TUI uses these existing ConnectRPC methods. **No new methods are introduced by this feature.**

| Method | Purpose in TUI |
|---|---|
| `TaskService.ListTasks` | Initial load, post-mutation reload, `Ctrl-R` reload |
| `TaskService.CreateTask` | `N` (subtask) and `Ctrl-N` (root) Save |
| `TaskService.UpdateTask` | `E` Save, digit-key estimate updates |
| `TaskService.DeleteTask` | `Ctrl-D` |
| `TaskService.CompleteTask` (or whatever toggles completion) | `Space` |
| `PomodoroService.StartPomodoro` (existing) | `S` |
| `PomodoroService.ResumePomodoro` (existing) | `R` |

If any of these methods do not currently exist with these exact names, the TUI uses the existing equivalents — implementation tasks resolve naming at codeing time. No new RPCs are added.

## Tree construction

```text
ListTasks response (flat) ──▶ internal/cli.BuildTree ──▶ []*treeNode ──▶ TUI Model.tree
                                                          (same order as `todo task list`)
```

`BuildTree` is the existing helper used by the CLI; reusing it guarantees ordering parity (per clarification Q1).

## Visible-row flattening

The `Model.visible` slice is derived purely from `tree`, `expanded`, `showCompleted`, and `pendingComplete`:

```text
for each root node in tree (in order):
  emit(root, depth=0)
  if expanded[root.id] and root has children:
    recurse into children
where emit(node, depth):
  if node.completed and not showCompleted and node.id != *pendingComplete:
    skip
  else:
    append visibleRow{node, depth, treePrefix, marker}
```

`treePrefix` follows the same `├──` / `└──` / `│  ` / `   ` rules as `internal/cli.renderTree`.

`marker`:
- `[+]` if `node has children and expanded[node.id] == false`
- `[-]` otherwise (expanded, or no children)

## Details pane

The details pane reads from `visible[cursor].node.task` and renders:

- ID (raw integer; or the existing post-id format from `buildPostID` if appropriate)
- Name (struck-through + dim if completed; reuse `dimStrike`)
- Due date (via `formatDue`)
- Pomodoro estimate (raw integer)
- Description (multi-line; wrapped to pane width)
- Completed-at timestamp (via `formatCompletedAt`) — only if completed and shown

If `visible` is empty, the details pane shows an empty/placeholder string.

## Mutation flows

Every mutation follows this pattern:

1. Optimistically transition the UI mode (close the form, exit edit mode).
2. Send the ConnectRPC request synchronously inside a `tea.Cmd`.
3. On response, re-fetch `ListTasks`, rebuild `tree`, rebuild `visible`, and resolve highlight per the contract rules above.
4. On error, surface in `Model.err`, do not change `tree`/`visible`, and keep cursor where it was.

## Pomodoro bridge

`S` causes the TUI to:

1. Exit the alt-screen (`tea.ExecProcess` or equivalent).
2. Call the in-CLI pomodoro function for the highlighted task ID (refactored from existing `runPomStart`; see research R4).
3. On return, re-enter the alt-screen and re-fetch tasks (a pomodoro may have updated counters server-side).

`R` likewise calls the existing `runPomResume`-equivalent function.

## What this contract does NOT cover

- The exact protobuf field names — those are the existing schema, untouched.
- Wire-level retry / timeout policy — inherits whatever the existing `connect` client already does.
- Authentication — TUI uses the same `TODO_API_KEY` env var the CLI already requires.
