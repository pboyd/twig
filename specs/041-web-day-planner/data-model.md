# Phase 1 Data Model: Web Day Planner (View)

This is a read-only client view over existing server data. No new persisted entities are introduced. The model below describes the data the client consumes and the small view-model it derives for rendering.

## Source entities (server, unchanged)

### PlanEntry (`plan.v1.PlanEntry`)

One scheduled segment of a single day. Identity is `(day, id)`. Returned by `ListPlanEntries`, ordered by `start_minute` ascending.

| Field | Type | Notes (consumed by this view) |
|-------|------|-------------------------------|
| `day` | string (`YYYY-MM-DD`) | The day the entry belongs to. |
| `id` | number (int32) | Per-day sequential id; stable React key within a day. |
| `taskId` | bigint (int64) | `0` ⇒ event (no task). Non-zero ⇒ task-linked; used for name fallback and navigation target. |
| `name` | string | Display-name override. Empty ⇒ fall back to linked task's name (FR-006), then to a generic label (FR-019). |
| `startMinute` | number? (0..1439) | Minutes since local midnight. **Absent ⇒ untimed** (task entries only). Wall-clock; rendered without timezone conversion (FR-020). |
| `durationMinute` | number (int32, >0) | Length in minutes. End = `startMinute + durationMinute`. |
| `completed` | boolean | Server-populated; `true` only for task-linked entries whose task is complete (FR-008). Always `false` for events. |

### Task (`task.v1.Task`) — referenced, unchanged

Read via the existing `ListTasks` query solely to resolve names for task-linked entries whose `name` is empty, and to validate navigation targets. Relevant fields: `id` (bigint), `name` (string). (Completion is already carried on `PlanEntry.completed`, so the task lookup is not needed for completion state.)

## Derived view model (client, in `lib/planView.ts`)

These are pure, unit-tested transforms — no React, no network.

### `ViewDay`
- `dayString: string` — the `YYYY-MM-DD` currently being viewed (state; defaults to today, local).
- Helpers: `todayString()`, `addDays(dayString, delta)`, `isToday(dayString)`, `formatDayLabel(dayString)` (e.g. "Today · Sat Jun 6").

### `ResolvedEntry`
A `PlanEntry` enriched for rendering:
- `id: number`
- `displayName: string` — `name` || linked task name || fallback label.
- `kind: "task" | "event"` — `taskId !== 0n ? "task" : "event"`.
- `taskId?: bigint` — present when `kind === "task"`; navigation target.
- `completed: boolean`
- `timed: boolean` — `startMinute !== undefined`.
- `startMinute?: number`, `endMinute?: number` (= start + duration) — present when timed.
- `timeLabel?: string` — `"9:00 am – 10:00 am"` style (12-hour), present when timed.
- `durationMinute: number`

### `GroupedPlan`
Result of grouping a day's `ResolvedEntry[]`:
- `timed: ResolvedEntry[]` — entries with `startMinute`, in ascending start order (server already sorts; client preserves).
- `untimed: ResolvedEntry[]` — entries without `startMinute`.
- `isEmpty: boolean` — `timed.length === 0 && untimed.length === 0`.

## Key transforms / rules

- **Name resolution** (FR-006, FR-019): `displayName = entry.name.trim() || taskNameById.get(entry.taskId) || "Untitled entry"`.
- **Kind & interactivity** (FR-007, FR-011, FR-021): task entries are selectable → `/tasks/:taskId`; event entries render display-only.
- **Time formatting** (FR-005, FR-020): `formatMinute(min)` maps `0..1439` → 12-hour `H:MM am/pm` with no `Date`/UTC involvement. `timeLabel` joins start and end.
- **Grouping** (FR-009): partition by presence of `startMinute`.
- **Gap visibility** (FR-010): emerges from showing each timed entry's own start–end; consecutive entries with `prev.endMinute < next.startMinute` visibly differ. (No synthetic gap rows required.)
- **Ordering**: rely on server's ascending `start_minute`; do not re-sort timed entries. Untimed entries keep server order.

## State (PlanPage)

| State | Type | Purpose |
|-------|------|---------|
| viewed day | `string` (`YYYY-MM-DD`) | Which day is shown; mutated by prev/next/today controls (FR-018). |
| plan query | React Query (`listPlanEntries`, input `{ day }`) | Source of entries; re-runs when `day` changes. |
| tasks query | React Query (`listTasks`) | Name resolution map (cached/shared). |

No client persistence is required (unlike the task tree's expand-state). The viewed day resets to today on each visit, consistent with FR-003.
