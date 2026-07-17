# RPC Usage Contract: Add New Tasks to a Plan

**Feature**: 066-add-task-to-plan | **Date**: 2026-07-17

## No wire-contract change

This feature changes **no** `.proto` file. `make proto` is not run; `api/gen/` is untouched; no
server handler, query, or migration changes.

It is recorded here anyway because Constitution Principle II makes the contract the source of truth
for the tiers — and here the point of the contract is precisely that the existing one already
suffices, and *how* this feature must call it.

## Consumed RPCs (both already shipped)

### `task.v1.TaskService.CreateTask`

Called exactly as today. The plan choice contributes nothing to this request — no new field, and the
task entity remains ignorant of plans.

### `plan.v1.PlanService.AddPlanTask`

Called only when the user picked a day, and only after `CreateTask` returns an id.

```protobuf
AddPlanTaskRequest {
  string day            = 1;  // resolved ISO YYYY-MM-DD (local at save time)
  int64  task_id        = 2;  // id from the CreateTask response
  optional int32 start_minute = 3;  // OMITTED — untimed entry (FR-004)
  int32  duration_minute = 4;  // 0 — defer to the server's existing default
}
```

**Field-by-field obligations**:

| Field | Value | Why |
|---|---|---|
| `day` | resolved ISO day | FR-005: resolution happens client-side at save; the wire never carries "today" |
| `task_id` | the new id | ordering: this is why the plan call cannot precede create |
| `start_minute` | **omitted** | the user chose a day, not a time. Per `plan.proto:68`, absent ⇒ untimed |
| `duration_minute` | `0` | per `plan.proto:70`, 0 ⇒ server picks: `(estimate - completed) * 30` if positive, else 30. The create form must not invent a duration |

### Error contract

| Server response | Meaning | Required client behavior |
|---|---|---|
| `FAILED_PRECONDITION` | task already on that day's plan | show `messages.alreadyOnPlan`. Task stays created. Not a hard error |
| any other error | plan write failed | FR-007 partial success: keep the task, report created-but-not-planned |
| (create fails first) | — | FR-006: never call `AddPlanTask` at all |

`AddToPlanControl.tsx:38` already distinguishes `FAILED_PRECONDITION` this way; the create path must
match that handling rather than treating a duplicate as a generic failure.

## Call sequence

```
CreateTask(name, description, due?, snooze?, parent?)
   │  fails ─────────────────────────────► stop. no plan call. (FR-006)
   ▼
task.id
   │  planDay == "" ─────────────────────► done (default path, FR-003)
   ▼
AddPlanTask(day=planDay, task_id=task.id, start_minute=absent, duration_minute=0)
   │  ok               ──► success message + refresh tree and that day's plan (FR-008)
   │  FAILED_PRECONDITION ──► alreadyOnPlan message; task kept
   └  other error      ──► partial-success message; task kept (FR-007)
```

**Non-atomic by design.** Two calls, no transaction. Justified in `research.md` R1: a dedicated
combined RPC would add proto surface, a handler, and a transaction to serve one UI convenience,
violating Principle I. The cost is the partial-failure window, which is handled explicitly and
visibly rather than papered over — and which leaves the user with their task, never with silent data
loss.

## Refresh obligations after a successful plan write

| Surface | Must refresh |
|---|---|
| TUI | the task tree (via the existing `fetchAfterMutation` path); the Plan tab re-fetches on view as it already does |
| Web | invalidate `listTasks`, and `listPlanEntries` for `day` — the key shape at `AddToPlanControl.tsx:30` |
