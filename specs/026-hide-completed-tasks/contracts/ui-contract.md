# Contracts: Hide Completed Tasks

## API contract — NO CHANGE

This feature makes **no changes** to any API, proto, or RPC contract.

- `task.v1.TaskService.ListTasks` is consumed as-is and continues to return **all** tasks
  (completed and incomplete), each carrying `completedAt`.
- No request fields are added (no `include_completed` flag — see research Decision 1).
- `CompleteTask` / `UncompleteTask` are unchanged; the feature only reacts to their existing
  React Query cache invalidation.

This record exists to satisfy Constitution Principle II / Quality Gate 1: the contract is
explicitly reviewed and confirmed unchanged before implementation begins.

## Client-side UI contract

Since the only new "interface" is within the React client, the contract below defines the
pure functions and the component behavior that implementation MUST satisfy. These are the
units that tasks/tests will target.

### `filterTree(nodes, showCompleted)` — `src/lib/filterTree.ts`

```text
filterTree(nodes: TaskNode[], showCompleted: boolean): TaskNode[]
```

| Guarantee | Behavior |
|-----------|----------|
| Identity passthrough | `showCompleted === true` ⇒ returns the input tree unchanged (same nodes, same order, same `depth`). |
| Drop completed leaf | An incomplete-free leaf (`completedAt` set, no children) is removed. |
| Keep incomplete leaf | A leaf with `completedAt` unset is kept. |
| Keep partial parent | An **incomplete** parent is kept; its completed children are removed, its incomplete children retained, order preserved. |
| Prune completed subtree | A **completed** parent whose entire subtree is completed is removed along with all descendants. |
| Keep completed ancestor of incomplete | A completed node that still has a kept (incomplete) descendant is retained, so the incomplete descendant is not orphaned. |
| Purity | Does not mutate input nodes; `depth` values are preserved (no re-parenting). |

### `showCompletedPref` — `src/lib/showCompletedPref.ts`

```text
readShowCompleted(): boolean      // localStorage 'twig-show-completed'; default false; never throws
writeShowCompleted(value: boolean): void   // best-effort; swallows storage errors
```

| Guarantee | Behavior |
|-----------|----------|
| Default | Missing/unreadable key ⇒ `false`. |
| Round-trip | `writeShowCompleted(true)` then `readShowCompleted()` ⇒ `true`. |
| Robust | Any non-`"true"` stored value ⇒ `false`; storage exceptions are caught. |

### `TaskTreePage` behavior

| Requirement | Observable behavior |
|-------------|---------------------|
| FR-001 | On mount with no saved preference, completed tasks are not rendered. |
| FR-002 | A single control toggles between hiding and showing completed tasks. |
| FR-005 | When shown, completed tasks render with the existing "done" styling (strikethrough / checked). |
| FR-006 | Completing/uncompleting a task updates the visible list with no manual reload. |
| FR-007 | The toggle initializes from `readShowCompleted()` and calls `writeShowCompleted` on change. |
| FR-008 | When `tasks.length > 0` but the filtered tree is empty, render the playful "all done — completed hidden" message + a way to reveal them (not the generic empty state). |
| FR-009 | The toggle is in the always-rendered header row, present even when no tasks are visible. |
