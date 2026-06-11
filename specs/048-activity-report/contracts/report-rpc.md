# API Contract: CountCompletedPomodoros RPC

**Feature**: 048-activity-report | **Status**: contract — commit before implementation

The only server-side addition for the activity report. Added to the existing
`task.v1.TaskService` in `api/proto/task/v1/task.proto` (pomodoro RPCs already
live there). Regenerate stubs with `make proto`.

## Proto additions

```proto
service TaskService {
  // …existing RPCs unchanged…

  // CountCompletedPomodoros returns how many of the calling user's pomodoros
  // were completed within the half-open UTC instant range [start, end).
  // A pomodoro counts when complete = true and start <= end_at < end,
  // regardless of whether its task is complete.
  rpc CountCompletedPomodoros(CountCompletedPomodorosRequest)
      returns (CountCompletedPomodorosResponse);
}

message CountCompletedPomodorosRequest {
  // Inclusive lower bound (UTC). Required.
  google.protobuf.Timestamp start = 1;
  // Exclusive upper bound (UTC). Required; must be after start.
  google.protobuf.Timestamp end = 2;
}

message CountCompletedPomodorosResponse {
  // Number of pomodoros completed in [start, end). 0 when none.
  int64 count = 1;
}
```

## Semantics

| Aspect | Behavior |
|---|---|
| Scope | Calling user only (from auth middleware), like all other RPCs |
| Range | Half-open `[start, end)` on `pomodoros.end_at`; timestamps are UTC instants — the client converts local-day boundaries before calling |
| Counted rows | `complete = TRUE` only; cancelled pomodoros never count |
| Empty range result | `count = 0` (not an error) |

## Errors

| Condition | Code |
|---|---|
| `start` or `end` missing | `InvalidArgument` |
| `end <= start` | `InvalidArgument` |
| Unauthenticated | `Unauthenticated` (existing middleware) |

## Backing query (sqlc, `services/twig/db/queries/pomodoro.sql`)

```sql
-- name: CountCompletedPomodorosInRange :one
SELECT count(*)::bigint AS count FROM pomodoros
WHERE user_id = $1 AND complete AND end_at >= $2 AND end_at < $3;
```

No schema migration required.
