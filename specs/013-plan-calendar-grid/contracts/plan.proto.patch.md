# Contract: `PlanEntry.completed`

## File

`services/todo/proto/plan/v1/plan.proto`

## Change

Add a new field to the existing `PlanEntry` message:

```proto
message PlanEntry {
  string day             = 1;
  int32  id              = 2;
  int64  task_id         = 3;
  string name            = 4;
  int32  start_minute    = 5;
  int32  duration_minute = 6;

  // True iff this entry references a task (task_id != 0) whose underlying
  // task has been marked completed (tasks.completed_at IS NOT NULL).
  // Always false for event entries (task_id == 0).
  // Populated server-side by ListPlanEntries; ignored on the request side.
  bool   completed       = 7;
}
```

## Producers and consumers

- **Producer**: `internal/handler.PlanHandler.ListPlanEntries` (server). Read from a LEFT JOIN against `tasks` in the sqlc-generated query.
- **Consumer**: `internal/cli.RenderGrid` (CLI). Used to decide whether to render the entry label with strikethrough + dim.

## Compatibility

- Field number `7` is new and not previously used.
- This is a purely additive change — protobuf wire format guarantees old clients silently ignore the new field and old servers send the zero value (`false`) which the new client treats as "not completed."

## Validation

- No client write path uses `completed`; it is output-only on `ListPlanEntries`. Add/move/rename/remove requests do not need to include this field.
- A unit test in `internal/handler` MUST cover: (a) event entry → `completed == false`, (b) task entry with no completion → `completed == false`, (c) task entry with completion → `completed == true`.
