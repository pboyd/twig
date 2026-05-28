# Data Model — TUI Completed Task Display

This feature is **view-only**. No persisted entities, no proto messages, no database tables, no API requests/responses are added or changed.

## View-model entities

### `Model.pendingComplete *int64` *(existing field — no schema change)*

- **Location**: `services/todo/internal/tui/model.go`
- **Meaning**: When non-nil, holds the task id of a "lingering just-completed" row that should remain visible in the tree even though the task's `CompletedAt` is now set.
- **Lifecycle**:
  - **Set**: in `handleListKey`'s `Complete` case, immediately when the user presses the complete key. Set before the `completeTaskCmd` is dispatched.
  - **Cleared**: in `handleListKey`'s `Up`, `Down`, and any other branch that intentionally moves `m.cursor` to a row whose id differs from `*m.pendingComplete`. Also cleared at the top of the user-triggered `Refresh` handler (FR-006).
  - **Not cleared**: by `Expand`, `Collapse` of a leaf (which is same-row no-op), `Edit`, `Help`, `Filter`, pomodoro key, digit-key estimate.
- **Lifetime**: View-only; never persisted, never sent over the wire, reset on TUI restart.

### `visibleRow` *(existing struct — no change)*

- **Location**: `services/todo/internal/tui/tree.go`
- Already carries the `*cli.TreeNode` whose `Task.CompletedAt` field is the source of truth for completion state used by the renderer.

## Derived state used by the renderer

For each `visibleRow` in `m.visible`, the renderer derives:

- `completed := row.node.Task.GetCompletedAt() != nil`
- `selected := i == m.cursor`

Strikethrough is applied iff `completed` (regardless of `selected`). Selection highlight is applied iff `selected` (regardless of `completed`). The two style layers compose: a completed selected row is both strikethrough and highlighted (Story 1's lingering row).

## State transitions

```text
                         user presses Complete
                                 │
incomplete ───────────────────────▶ lingering-completed
   (no styling,                       (strikethrough on name,
    not in m.visible if               selection retained,
    showCompleted=false)              m.pendingComplete = id)
                                       │
                                       │ user presses Up/Down/jump-to OR Refresh
                                       ▼
                                 completed-and-hidden
                                       (if showCompleted=false)
                                  OR completed-and-styled
                                       (if showCompleted=true)
                                  m.pendingComplete = nil
```

No other transitions are introduced.
