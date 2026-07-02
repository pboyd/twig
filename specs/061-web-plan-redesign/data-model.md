# Data Model: Web Plan Tab Day-Planner Redesign

**Date**: 2026-07-02 | **Spec**: [spec.md](./spec.md)

No persistent data changes. All types below are frontend view-model types in `services/twig-web/src/lib/planView.ts`; wire types (`plan.v1.PlanEntry`, `task.v1.Task`) are unchanged.

## Existing (unchanged)

### ResolvedEntry

Already defined in `planView.ts`; produced by `resolveEntries(entries, taskNameById)`.

| Field | Type | Notes |
|---|---|---|
| `id` | `number` | Plan-entry id (remove key) |
| `displayName` | `string` | Entry name, falling back to task name |
| `kind` | `"task" \| "event"` | Events have no done control (FR-007) |
| `taskId` | `bigint?` | Present for task entries; complete/uncomplete + detail link |
| `completed` | `boolean` | Drives toggle state + strikethrough |
| `timed` | `boolean` | Timeline vs untimed section |
| `startMinute` / `endMinute` | `number?` | Minutes since midnight (timed only) |
| `timeLabel` | `string?` | Existing formatted range |
| `durationMinute` | `number` | Duration |

### GroupedPlan

`groupPlan(resolved)` — unchanged: `timed` (all, including completed), `untimed` (incomplete only), `isEmpty`.

## New view-model helpers (pure functions, unit-tested)

### TimelineWindow

```ts
interface TimelineWindow {
  startMinute: number; // inclusive, hour-aligned (multiple of 60)
  endMinute: number;   // exclusive, hour-aligned
}
```

**Validation/rules**:
- `computeWindow(timed: ResolvedEntry[]): TimelineWindow`
- Default/minimum window: 480–1020 (8:00–17:00) when no timed entries or as span floor.
- Otherwise: `startMinute = floorToHour(min(snapDown15(start)))`, `endMinute = ceilToHour(max(snapUp15(end)))`, each merged with the default window (union).
- Invariants: `startMinute < endMinute`; both multiples of 60; window contains every entry's snapped span.

### Slot geometry

```ts
const SLOT_MINUTES = 15;
const SLOT_PX = 44; // spacing.touchTarget

snapDown15(min: number): number
snapUp15(min: number): number
slotIndex(window: TimelineWindow, minute: number): number   // (snapped minute − startMinute) / 15
slotCount(window: TimelineWindow): number                   // (end − start) / 15
hourLabels(window: TimelineWindow): { minute: number; label: string }[] // reuses formatMinute()
```

**Rules**: an entry occupies grid rows `[slotIndex(snapDown15(start)), slotIndex(snapUp15(end)))`; minimum one row. Now-indicator offset: `(nowMinute − startMinute) / SLOT_MINUTES × SLOT_PX`, rendered only when `startMinute ≤ nowMinute < endMinute` and the viewed day is today.

## State transitions (per task entry)

| State | Action | Result |
|---|---|---|
| Incomplete | Toggle activated | `CompleteTask` RPC → completed (filled circle, strikethrough); blocked-by-subtasks → inline `completeBlockedBySubtasks` message, state unchanged |
| Completed | Toggle activated | `UncompleteTask` RPC → incomplete; blocked-by-parent → inline `reopenBlockedByParent` message, state unchanged |
| Any | Remove (trash) | `RemovePlanEntry` RPC → entry leaves the day plan (task itself untouched); `entryRemoved` toast |

Both mutations invalidate the `listPlanEntries` and `listTasks` queries (existing pattern).
