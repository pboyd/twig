# Phase 1 Data Model: Basic Planning Edits on the Web App

This feature adds **no persistent data structures** — it consumes existing server entities and adds a few client-side view models. The authoritative wire types are in `api/proto/plan/v1/plan.proto` and `api/proto/task/v1/task.proto`.

## Server entities (existing, unchanged)

### PlanEntry (`plan.v1.PlanEntry`)
Identity is `(day, id)`.

| Field | Type | Used by this feature |
|-------|------|----------------------|
| `day` | string `YYYY-MM-DD` | scope of all plan reads/writes |
| `id` | int32 (per-day) | target of `RemovePlanEntry` |
| `task_id` | int64 (0 = none) | distinguishes task entry (completable) vs event |
| `name` | string | display fallback handled by `resolveEntries` |
| `start_minute` | optional int32 | **omitted on add** → untimed entry |
| `duration_minute` | int32 | server default when adding (`0`) |
| `completed` | bool (read-only) | drives "done" styling / 044 hide rule |

### Task (`task.v1.Task`)
| Field | Type | Used by this feature |
|-------|------|----------------------|
| `id` | int64 | source of `AddPlanTask.task_id` and `CompleteTask.id` |
| `name` | string | toast label / row label |
| `completedAt` / `complete` | timestamp / bool | reflects completion after `CompleteTask` |

## Client view models

### `ResolvedEntry` (existing, `lib/planView.ts`) — unchanged
Already exposes `id`, `displayName`, `kind` (`"task" | "event"`), `taskId?`, `completed`, `timed`. The planner edits read from this directly:
- **Complete** is offered when `kind === "task" && taskId !== undefined && !completed`.
- **Remove** is offered for every `ResolvedEntry`.

### `DayTarget` (new, `lib/planDays.ts`)
A small pure helper module for choosing the add target day. No type ceremony beyond string helpers:

| Helper | Signature | Notes |
|--------|-----------|-------|
| `todayIso()` | `() => string` | local `YYYY-MM-DD` (reuse/extend `planView.todayString`) |
| `tomorrowIso()` | `() => string` | `addDays(todayIso(), 1)` |
| `dayPickerLabel(iso)` | `(string) => string` | "Today" / "Tomorrow" / `formatDayLabel` for toasts |

**Validation rules**:
- Target day must be a valid `YYYY-MM-DD`. The native `<input type="date">` guarantees the format; empty selection disables the **Add** confirm.
- Past days are permitted (FR-004) — no lower bound.

### Toast model (new, `context/ToastProvider.tsx`)
| Field | Type | Notes |
|-------|------|-------|
| `id` | number | auto-increment key for list rendering |
| `message` | string | playful copy from `messages.ts` |
| `tone` | `"success" \| "error"` (optional) | styling via tokens; defaults to success |

Lifecycle: enqueued by `useToast().show(message, tone?)` → rendered in a fixed overlay → auto-removed after ~3s (or on tap-to-dismiss). No persistence.

## Request shapes used (no new messages)

| Action | RPC | Request |
|--------|-----|---------|
| Add to today | `AddPlanTask` | `{ day: todayIso(), taskId, durationMinute: 0 }` (no `startMinute`) |
| Add to chosen day | `AddPlanTask` | `{ day: <picked iso>, taskId, durationMinute: 0 }` |
| Remove entry | `RemovePlanEntry` | `{ day, id }` |
| Complete task | `CompleteTask` | `{ id: taskId }` |

## State transitions

- **Add**: (task not on day) → untimed `PlanEntry` created. (task already untimed on day) → no change; server returns `FailedPrecondition`.
- **Complete**: task `incomplete` → `complete`; on the planner an untimed entry then disappears (044), a timed entry stays and shows done. Blocked if the task has incomplete sub-tasks (`FailedPrecondition`).
- **Remove**: `PlanEntry (day,id)` exists → deleted; linked `Task` unaffected.
