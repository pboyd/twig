# Contract: TUI Tree-Row and Detail-Pane Rendering

This feature has no external API surface (no proto, no HTTP, no CLI flag). The "contract" captured here is the internal rendering contract between the TUI model, the visible-row flattener, and the two pane renderers. It is the source of truth that the implementation MUST conform to, and it is what the tests assert.

## C-1: `cli.Strike(s string) string`

**Location**: `services/todo/internal/cli/render.go`

- When `isTTY == true`: returns `s` wrapped with the ANSI strikethrough open code (`\x1b[9m`) and the reset code (`\x1b[0m`), with no dim or color attribute.
- When `isTTY == false`: returns `s` unchanged.

Companion to `DimStrike`; same gating convention.

## C-2: `tui.Model.pendingComplete` setter contract

**Trigger**: in `handleListKey` for `key.Matches(msg, m.keys.Complete)`, immediately before `completeTaskCmd` is returned:

```go
id := m.visible[m.cursor].node.Task.Id
m.pendingComplete = &id
m.err = nil
return m, completeTaskCmd(m.client, id)
```

**Invariant**: When `pendingComplete != nil`, the value points to a task id that was visible and selected at the moment the user pressed Complete.

## C-3: `tui.Model.pendingComplete` clearer contract

`pendingComplete` MUST be set to `nil` at the start of these handlers:

- `key.Matches(msg, m.keys.Up)` — already done.
- `key.Matches(msg, m.keys.Down)` — already done.
- `key.Matches(msg, m.keys.Refresh)` — **NEW**: add `m.pendingComplete = nil` before `listTasksCmd(m.client)`.

`pendingComplete` MUST NOT be cleared by:

- `Expand` / `Collapse` handlers (same-row UX state change).
- `Edit`, `Help`, `Filter`, pomodoro, digit-key estimate handlers.
- `refreshedMsg` arriving from `fetchAfterMutation` (the post-complete redraw).

## C-4: Tree-row rendering contract (`renderList`)

**Location**: `services/todo/internal/tui/view.go::renderList`

For each `row` in `m.visible`:

1. Compose the unstyled line: `treePrefix + marker + " " + row.node.Task.Name`.
2. If `row.node.Task.GetCompletedAt() != nil`, the **name portion only** is passed through `cli.Strike` before composition. The tree connector and marker are NOT struck through.
3. Truncate to `width` (working on the visible-width measurement, not the byte length, so ANSI codes aren't cut mid-sequence — reuse `lipgloss.Width` if needed).
4. If `i == m.cursor`, wrap the full padded line in `highlightStyle`.
5. Pad and emit.

The strikethrough must be visible on every completed row, regardless of nesting depth, regardless of whether the row is the lingering-completed one, the cursor row, or just a `showCompleted == true` row.

## C-5: Detail-pane rendering contract (`renderDetails`)

**Location**: `services/todo/internal/tui/details.go::renderDetails`

- The `Name:` line MUST render `task.Name` raw — no `cli.DimStrike` or `cli.Strike` wrapping, even when `task.CompletedAt != nil`.
- The `Completed: <timestamp>` line continues to render unchanged.
- All other fields (`ID`, `Due`, `Est`, `Description`) are unchanged.

## C-6: `buildVisible` contract (unchanged — already correct)

For completeness: `buildVisible(tree, expanded, showCompleted, pendingComplete)` already keeps a completed task in the visible rows when its id equals `*pendingComplete`. No change required; this is the foundation Story 1 relies on.

## Test obligations

The tests added in Phase 2 MUST cover at minimum:

- **T-A**: `cli.Strike` wraps with `\x1b[9m` when `isTTY=true`, returns input unchanged when `isTTY=false`.
- **T-B**: After pressing Complete on a selected row, `m.pendingComplete` points to that row's id and `buildVisible` keeps the row visible after the post-complete refresh.
- **T-C**: Pressing `Up` (or `Down`) after a Complete clears `m.pendingComplete` and the next `buildVisible` no longer emits the now-completed row (assuming `showCompleted=false`).
- **T-D**: Pressing `Expand`/`Collapse`/`Edit`/`Help`/`Filter` after a Complete leaves `m.pendingComplete` non-nil and the row still visible.
- **T-E**: Pressing the user-facing `Refresh` key clears `m.pendingComplete`.
- **T-F**: `renderList` emits ANSI strikethrough around the *name segment* for a completed row, and does not emit it for an incomplete row.
- **T-G**: `renderDetails` emits the task name without strikethrough even when `CompletedAt != nil`.
