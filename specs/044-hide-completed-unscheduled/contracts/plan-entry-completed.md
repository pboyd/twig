# Contract: `PlanEntry.completed` (consumed, not changed)

This feature adds **no new API surface**. It relies entirely on an existing field of the existing `plan.v1.PlanService.ListPlanEntries` response. This document records the contract the clients depend on, satisfying the API-First gate (the contract pre-exists and is unchanged).

## Source of truth

`api/proto/plan/v1/plan.proto`:

```proto
message PlanEntry {
  string day = 1;
  int32 id = 2;
  int64 task_id = 3;            // 0 ⇒ event (no task link)
  string name = 4;
  optional int32 start_minute = 5;  // absent ⇒ untimed (task entries only)
  int32 duration_minute = 6;
  // True iff this entry references a completed task (task_id != 0 and
  // tasks.completed_at IS NOT NULL). Always false for event entries.
  // Populated server-side by ListPlanEntries; ignored on writes.
  bool completed = 7;
}
```

## Guarantees relied upon

1. `ListPlanEntries` returns every entry for the day, each carrying an accurate `completed` value.
2. `completed == false` for every event entry (`task_id == 0`).
3. `completed` reflects the linked task's current completion state at fetch time, so completing/uncompleting a task (via `task.v1.TaskService.CompleteTask` / `UncompleteTask`) changes the value on the next `ListPlanEntries`.
4. `start_minute` absence is the authoritative signal that an entry is untimed/unscheduled.

## Client obligations introduced by this feature

| Client | Obligation |
|--------|-----------|
| TUI (`internal/tui`) | Exclude from the displayed plan list any entry with `start_minute` absent AND `completed == true`, **except** the entry currently held in `pendingComplete`. Never exclude timed entries or events. |
| Web (`services/twig-web`) | In `groupPlan`, exclude from `untimed` any entry with `completed == true`. Never exclude timed entries. |

## Explicitly out of scope (no contract change)

- No new RPC, request/response field, or DB migration.
- Writes (`AddPlanTask`, `MovePlanEntry`, `RemovePlanEntry`, …) are unchanged; the feature never deletes an entry to hide it.
