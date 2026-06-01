# Contract: Plan Task Actions

## API delta: none

This feature adds **no** proto messages, services, or RPCs, and changes **no** request/response shapes. It is recorded here so the Constitution Principle II quality gate has an explicit artifact to verify.

The two planning-tab actions are implemented entirely on the client (TUI) by invoking operations that already exist and are already exercised by the Tasks tab:

| Planning-tab action | Existing operation invoked | Request (unchanged) | Response (unchanged) |
|---------------------|----------------------------|---------------------|----------------------|
| Toggle complete — complete | `task.v1.TaskService/CompleteTask` | `CompleteTaskRequest{ id = entry.TaskId }` | `CompleteTaskResponse` |
| Toggle complete — uncomplete | `task.v1.TaskService/UncompleteTask` | `UncompleteTaskRequest{ id = entry.TaskId }` | `UncompleteTaskResponse` |
| Start pomodoro | existing pomodoro-start call (`startPomCmd`, `task.v1`) | unchanged (`taskId = entry.TaskId`, label `= entry.Name`) | unchanged (`pomStartedMsg`) |

After a successful complete/uncomplete, the TUI re-reads the day with the existing `plan.v1.PlanService/ListPlan` (via `listPlanHighlightCmd`); `PlanEntry.Completed` is recomputed by the server, so no new field or flag is needed.

## Client-side behavioral contract (TUI, planList mode)

These are interaction guarantees, not wire-format changes:

1. `space` on a task-linked entry toggles the linked task's completion (direction chosen from `entry.Completed`) and reloads the day, re-selecting the same entry.
2. `s` on a task-linked entry starts a pomodoro for that task, identical to the Tasks tab.
3. Either key on an entry with `TaskId == 0` (event) is a no-op that emits a playful `notice` and changes no task/timer state.
4. Either key with an empty plan (no selected entry) is a no-op.
5. On RPC failure, the error surfaces via `m.plan.err` (planning tab) and the displayed completion state is not falsely advanced (the entry only redraws after a successful reload).
