# Contract Delta: `plan.v1.PlanService`

This feature changes only the **start-time presence** on three messages. No RPCs are added or removed. `make proto` regenerates `gen/`; this document is the source of truth the implementation must conform to (Constitution II).

## Proto changes (`services/twig/proto/plan/v1/plan.proto`)

```proto
message PlanEntry {
  string day = 1;
  int32 id = 2;
  int64 task_id = 3;
  string name = 4;
  optional int32 start_minute = 5;   // CHANGED: was `int32`. Absent ⇒ untimed.
  int32 duration_minute = 6;
  bool completed = 7;
}

message AddPlanTaskRequest {
  string day = 1;
  int64 task_id = 2;
  optional int32 start_minute = 3;   // CHANGED: was `int32`. Absent ⇒ untimed entry.
  int32 duration_minute = 4;         // 0 ⇒ server default (unchanged)
}

message MovePlanEntryRequest {
  string day = 1;
  int32 id = 2;
  optional int32 start_minute = 3;   // CHANGED: was `int32`. Absent ⇒ unschedule (make untimed).
  int32 duration_minute = 4;         // 0 ⇒ keep existing duration (unchanged)
}

// UNCHANGED — events must always be timed:
message AddPlanEventRequest {
  string day = 1;
  string name = 2;
  int32 start_minute = 3;            // required
  int32 duration_minute = 4;
}
```

Field numbers are preserved, so the change is wire-compatible (a present `optional int32` encodes identically to the old `int32`).

## Behavioral contract

### `AddPlanTask`
- `start_minute` **present**: unchanged — validate `0..1439`, compute/honor duration, overlap-check against the day, insert timed.
- `start_minute` **absent**: create an **untimed** entry.
  - Skip start-range validation, the `start + duration <= 1440` check, and the overlap check (an untimed entry occupies no grid time).
  - Duration default is unchanged (`(estimate − completed_pomodoros) * 30` if positive, else `30`).
  - Response `PlanEntry` has `start_minute` **absent**.
- Task must exist (`NotFound` otherwise) — unchanged.

### `MovePlanEntry`
- `start_minute` **present**: unchanged — schedule/move; `duration_minute = 0` keeps existing duration; overlap-checked. A previously untimed entry becomes timed.
- `start_minute` **absent**: **unschedule** — set the stored `start_minute` to NULL, preserve duration (`duration_minute = 0` keeps it; a positive value overrides), no overlap check. Entry becomes untimed.
- Entry must exist (`NotFound` otherwise) — unchanged.

### `AddPlanEvent`
- Unchanged. `start_minute` is required; an absent/invalid start is rejected (`InvalidArgument`). The DB CHECK `task_id IS NOT NULL OR start_minute IS NOT NULL` is a backstop.

### `ListPlanEntries`
- Returns timed and untimed entries together, ordered `start_minute ASC NULLS FIRST, id ASC` (untimed first, by creation order).
- Untimed entries return with `start_minute` absent and a populated `duration_minute`.

### `ClearPlan`
- Unchanged. Only timed task entries at/after the cutoff are affected; untimed entries are never cleared (NULL start fails the `>= cutoff` predicate).

## Error tone (Constitution IV)

Handler-level rejections keep the existing warm phrasing (e.g. the overlap message `No room there — that bumps into entry %d.`). Any new message (e.g. an event missing a time) is authored in the same playful-but-clear register.
