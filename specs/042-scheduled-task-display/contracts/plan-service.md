# Contract: `PlanService.ListScheduledDays`

**Service**: `plan.v1.PlanService` (existing) — adds one RPC. No existing messages change.

**Source of truth**: `api/proto/plan/v1/plan.proto`. After editing the proto, regenerate with `make proto` (updates `api/gen/...`). This contract MUST be committed before implementation per Principle II.

## Purpose

Return every `(task, day)` pair on or after a caller-supplied cutoff day, so a client can show which current/future days each task is scheduled for. This is the reverse of the existing day-scoped `ListPlanEntries`.

## Proto additions

```proto
service PlanService {
  // ... existing RPCs unchanged ...

  // ListScheduledDays returns, for every task the caller has scheduled, each
  // distinct day on or after from_day on which a plan entry links that task.
  // Untimed entries count. Results are ordered by task_id then day ascending,
  // with no duplicate (task_id, day) pairs.
  rpc ListScheduledDays(ListScheduledDaysRequest) returns (ListScheduledDaysResponse);
}

message ListScheduledDaysRequest {
  // The caller's local current day, YYYY-MM-DD. Only days >= from_day are
  // returned, so callers using their local date get a local "past" cutoff.
  string from_day = 1;
}

message ListScheduledDaysResponse {
  // Flat list of scheduled (task, day) pairs, sorted by task_id then day asc.
  repeated ScheduledDay days = 1;
}

message ScheduledDay {
  // The scheduled task. Always non-zero (entries without a task are excluded).
  int64 task_id = 1;
  // The scheduled day, YYYY-MM-DD.
  string day = 2;
}
```

## Behavior

| Aspect | Contract |
|---|---|
| Auth | Caller identified from session/API key (context), as with all `PlanService` RPCs. Results scoped to that user only. |
| `from_day` validation | MUST be `YYYY-MM-DD`. Malformed → `INVALID_ARGUMENT`. |
| Filtering | Only entries with a non-null `task_id` and `day >= from_day` are returned. |
| Deduplication | At most one row per `(task_id, day)` even if multiple entries exist for that task that day. |
| Ordering | `task_id` ascending, then `day` ascending. |
| Empty result | Returns an empty `days` list (not an error) when the caller has no qualifying scheduled days. |
| Side effects | None (read-only). |

## Consumer expectations (TUI)

- The CLI/TUI client sends `from_day` = its **local** current date (`time.Now()` local, `2006-01-02`).
- The client groups the response into `map[int64][]string`; per-task day slices are already ascending and de-duplicated, so no client-side sorting/dedup is required.
- The Tasks-tab details pane renders `Scheduled for: <day[, day...]>` from the map; absent/empty ⇒ no line.

## Out of scope

No changes to `TaskService`, no new auth endpoints, no schema migration, no write paths.
