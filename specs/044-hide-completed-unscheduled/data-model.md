# Phase 1 Data Model: Hide Completed Unscheduled Plan Entries

This feature introduces **no new persisted entities** and **no schema or proto changes**. It consumes an existing field and adds one piece of transient TUI client state. The model below documents the entities as they bear on the display filter.

## Existing entities (unchanged)

### PlanEntry (`plan.v1.PlanEntry`)

| Field | Type | Relevance to this feature |
|-------|------|---------------------------|
| `day` | string (YYYY-MM-DD) | The plan day being viewed. |
| `id` | int32 | Per-day identity; used as the TUI highlight/pending key. |
| `task_id` | int64 | `0` ⇒ event (no link). Non-zero ⇒ task-linked entry. |
| `start_minute` | optional int32 | **Absent ⇒ untimed/unscheduled** (the only entries this feature can hide). Present ⇒ timed (never hidden). |
| `duration_minute` | int32 | Unused by the filter. |
| `completed` | bool | **The filter input.** `true` iff the linked task is complete. Always `false` for events. Populated server-side by `ListPlanEntries`; ignored on writes. |

No fields are added or modified.

### Task

Unchanged. Its completion state is the source of `PlanEntry.completed`; completing/uncompleting a task (via `CompleteTask`/`UncompleteTask`) is what flips the entry's `completed` value on the next `ListPlanEntries`.

## Derived classification (view-time, not stored)

An entry is **hideable by this feature** iff all hold:

- `start_minute` is absent (untimed), AND
- `task_id != 0` (task-linked — implied, since only task entries can be completed), AND
- `completed == true`.

Timed entries and events are never hideable.

## New transient client state (TUI only)

### `planState.pendingComplete *int32`

- **Holds**: the `id` of an untimed entry whose task was just completed from the planning view and which should remain visible (crossed out, highlighted) until the user navigates away.
- **Lifecycle**:
  - **Set** to the entry id when the user completes a (untimed) task-linked entry in the planning view.
  - **Honored** by the display filter: a completed untimed entry whose id equals `pendingComplete` is *not* hidden.
  - **Cleared** (`nil`) on any navigation that re-renders without that entry as the active highlight: cursor up/down, previous/next/today day change, switching tabs, go-to-task.
- **Not persisted**: in-memory only; resets across reloads. (Web has no equivalent — completion there originates outside the planner.)

## State transitions (TUI displayed list)

```text
incomplete untimed entry ──complete──► completed, pendingComplete=id
    (shown normally)                     (shown crossed out, highlighted)
                                              │
                              ┌───────────────┼────────────────┐
                       navigate away      reopen (uncomplete)
                              │                                │
                              ▼                                ▼
                    pendingComplete=nil               completed=false
                    → entry filtered out              pendingComplete=nil
                      (hidden)                         → shown normally again
```

## Validation / invariants

- The filter MUST NOT remove timed entries regardless of `completed` (FR-004).
- The filter MUST NOT remove events (`completed` is always false for them) (FR-005).
- Hiding is view-time only: no `RemovePlanEntry` is issued; the entry persists and reappears on uncomplete (FR-006, FR-007).
- TUI and web apply the same untimed-completed exclusion for identical data (FR-003).
