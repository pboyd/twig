# Data Model: Plan Task Actions

**No schema change.** This feature reads existing fields and invokes existing task-service operations. Documented here for the invariants the actions depend on.

## Entities

### PlanEntry (`plan.v1.PlanEntry`, existing)

| Field | Type | Role in this feature |
|-------|------|----------------------|
| `Id` | `int32` | Per-day entry identifier; used as `highlightID` to re-select the entry after a completion reload. |
| `TaskId` | `int64` | Link to a task. **`0` means the entry is an event** (no linked task) — the discriminator for both actions' event guard. |
| `Name` | `string` | Display name; for a task-linked entry this is the task's name, passed to `startPomCmd` as the pomodoro label. |
| `Completed` | `bool` | **Mirrors the linked task's completion**, computed server-side on `ListPlan`. Read to decide toggle direction; redrawn after reload. Not independently writable. |

**Invariant (relied upon, not introduced)**: `PlanEntry.Completed` is a projection of the linked task's completion state at read time. The planning tab never writes `Completed` directly; it completes/uncompletes the **task**, and a subsequent `ListPlan` reload reflects the change on every entry linked to that task.

### Task (`task.v1.Task`, existing)

The unit of work behind a task-linked entry. Its completion state (`completed_at`) is what the complete action toggles via `CompleteTask` / `UncompleteTask`. Completion is owned by the task and shared by every plan entry that links to it — there is no per-entry completion concept.

### Pomodoro (existing)

A single-active focus timer associated with a task, started via the existing pomodoro-start operation. Lifecycle, single-active-timer rule, lifecycle hooks (`on_start`/`on_cancel`/`on_complete`), and cross-tab running-state display are unchanged; this feature only adds a new trigger point.

## State transitions

| Selected entry state | Action (`space`) | Result |
|----------------------|------------------|--------|
| Task-linked, incomplete | toggle complete | linked task → completed; day reloads; entry (and any sibling entry for the same task) shows completed |
| Task-linked, completed | toggle complete | linked task → incomplete; day reloads; entry shows incomplete |
| Event (`TaskId == 0`) | toggle complete | no-op; playful notice |
| No entry selected (empty plan) | toggle complete | no-op |

| Selected entry state | Action (`s`) | Result |
|----------------------|--------------|--------|
| Task-linked (complete or not) | start pomodoro | pomodoro starts for the linked task; matches Tasks-tab behavior incl. any running-timer interaction |
| Event (`TaskId == 0`) | start pomodoro | no-op; playful notice |
| No entry selected (empty plan) | start pomodoro | no-op |
