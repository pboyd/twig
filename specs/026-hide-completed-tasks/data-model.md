# Phase 1 Data Model: Hide Completed Tasks

This feature introduces **no new persisted server-side entities**. It reuses the existing
`Task` shape and adds one client-only preference plus a derived view model.

## Existing entity (reused, unchanged): Task

From `task.v1.Task` (generated TS in `src/gen/task/v1/task_pb`), as consumed by `buildTree`:

| Field | Type | Relevance to this feature |
|-------|------|---------------------------|
| `id` | `bigint` | Node identity (unchanged) |
| `name` | `string` | Display (unchanged) |
| `parentId` | `bigint \| undefined` | Tree structure; determines which subtree a completed task belongs to |
| `completedAt` | timestamp \| unset | **Completion state.** A task is "completed" iff `completedAt` is set (truthy). This is the sole input to the hide/show filter. |

No fields are added or modified. The CLI/TUI consume the same `Task`; they are out of scope.

## New client-only state: ShowCompleted preference

A single boolean persisted per-browser.

| Property | Value |
|----------|-------|
| Conceptual name | `showCompleted` |
| Type | boolean |
| Storage | `localStorage`, key `twig-show-completed` (stored as `"true"` / `"false"`) |
| Default (key absent / unreadable) | `false` (completed tasks hidden) |
| Lifecycle | Written whenever the user toggles; read on page mount to initialize view state |

**Validation / robustness rules**:
- Any value other than the literal `"true"` is treated as `false` (default-safe).
- Read and write are wrapped in `try/catch`; storage failure falls back to the default and
  must not throw (mirrors existing `readExpandedIds`/`writeExpandedIds`).

## Derived view model: filtered task tree

Not persisted — computed during render from `(tree, showCompleted)`.

**Inputs**: `nodes: TaskNode[]` (output of `buildTree`), `showCompleted: boolean`.

**Output**: `TaskNode[]` — the same tree when `showCompleted` is `true`; otherwise a pruned
copy.

**Pruning rule (when `showCompleted === false`)**, applied bottom-up per node:

```
keepNode(node):
  filteredChildren = filter children with keepNode, preserving order
  isIncomplete = node.task.completedAt is unset
  keep node IF (isIncomplete OR filteredChildren is non-empty)
  if kept, node's children become filteredChildren
```

Equivalently: a completed leaf is dropped; a completed node is dropped only if all its
descendants are also dropped; an incomplete node is always kept (with its surviving
children). `depth` values from `buildTree` are preserved (filtering never re-parents nodes).

### State transitions / derived states

| Condition | Resulting view |
|-----------|----------------|
| `showCompleted = true` | Full tree, completed tasks visibly marked as done |
| `showCompleted = false`, some incomplete tasks exist | Pruned tree (incomplete branches + their kept ancestors) |
| `showCompleted = false`, tasks exist but all are completed | Empty filtered tree → **"all done / completed hidden"** state (FR-008) |
| No tasks exist at all | Generic empty state (existing `EmptyState`) |

The distinction between the last two rows is made by: `tasks.length > 0` (raw) but
`filteredTree.length === 0`.
