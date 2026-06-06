# Consumed Contract: `plan.v1.PlanService.ListPlanEntries`

This feature introduces **no new API contract**. It consumes one existing, already-committed RPC. The source of truth is `api/proto/plan/v1/plan.proto`; generated client stubs already live at `services/twig-web/src/gen/plan/v1/`. This file documents the slice this view depends on, satisfying the constitution's requirement that the contract be recorded before implementation.

## RPC used

```proto
// PlanService manages day-scoped plan entries for the calling user.
service PlanService {
  // ListPlanEntries returns every entry on the given day, ordered by start_minute ascending.
  rpc ListPlanEntries(ListPlanEntriesRequest) returns (ListPlanEntriesResponse);
}

message ListPlanEntriesRequest { string day = 1; } // day in YYYY-MM-DD
message ListPlanEntriesResponse { repeated PlanEntry entries = 1; }

message PlanEntry {
  string day = 1;                    // YYYY-MM-DD
  int32  id = 2;                     // per-day sequential identity
  int64  task_id = 3;                // 0 = event (no task)
  string name = 4;                   // empty => fall back to linked task's name
  optional int32 start_minute = 5;   // minutes since local midnight (0..1439); absent => untimed
  int32  duration_minute = 6;        // > 0; start_minute + duration_minute <= 1440
  bool   completed = 7;              // server-populated; true only for completed task-linked entries
}
```

## Client usage

- **Transport**: shared `transport` (`src/lib/transport.ts`), same-origin, `credentials: "include"`. Auth handled by the existing ConnectRPC middleware; the transport interceptor redirects to `/login` on `Code.Unauthenticated`.
- **Call**: `useQuery(listPlanEntries, { day })` where `day` is the viewed `YYYY-MM-DD` (defaults to today, local). Re-runs when `day` changes.
- **Mutations**: none. This view calls **only** `ListPlanEntries` from `PlanService`. The Add/Remove/Rename/Move/Clear RPCs are intentionally NOT used (FR-013, read-only).

## Field handling expectations (verifiable)

| Field | Client obligation |
|-------|-------------------|
| `entries` order | Render in returned order (server sorts timed by `start_minute` asc). Do not re-sort. |
| `start_minute` absent | Treat as **untimed**; render in the untimed group (FR-009). |
| `start_minute` present | Render as wall-clock `HH:MM` with no timezone conversion (FR-020); show `start–end` (FR-005). |
| `task_id == 0` | Render as **event**, display-only / non-interactive (FR-007, FR-021). |
| `task_id != 0` | Render as **task**; tapping navigates to `/tasks/{task_id}` (FR-011). |
| `name == ""` | Fall back to linked task's name, then to a generic label (FR-006, FR-019). |
| `completed == true` | Mark the entry as completed (FR-008). |
| empty `entries` | Show the day's empty state (FR-012). |
| RPC error | Show retryable error (FR-017); `Unauthenticated` → redirect to login (existing interceptor). |

## Dev environment dependency

`vite.config.ts` proxy must forward `"/plan.v1"` → `http://localhost:8080` (mirroring the existing `/task.v1`, `/health.v1` entries) so the same-origin session cookie reaches the server in dev.
