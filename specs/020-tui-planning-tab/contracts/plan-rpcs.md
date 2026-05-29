# Contract: PlanService RPCs (reused, unchanged)

This feature introduces **no new contract** and requires no `make proto`. It consumes the existing `PlanService`, defined in `services/todo/proto/plan/v1/plan.proto` and already implemented in `internal/handler` (the same service the CLI `todo plan` command uses). This document records exactly what the TUI relies on, so the implementation conforms to the contract (Constitution Principle II).

## Service

```
service PlanService {
  rpc ListPlanEntries(ListPlanEntriesRequest) returns (ListPlanEntriesResponse);
  rpc AddPlanTask(AddPlanTaskRequest)         returns (AddPlanTaskResponse);
  rpc AddPlanEvent(AddPlanEventRequest)       returns (AddPlanEventResponse);
  rpc RenamePlanEntry(RenamePlanEntryRequest) returns (RenamePlanEntryResponse);
  rpc MovePlanEntry(MovePlanEntryRequest)     returns (MovePlanEntryResponse);
  rpc RemovePlanEntry(RemovePlanEntryRequest) returns (RemovePlanEntryResponse);
  rpc ClearPlan(ClearPlanRequest)             returns (ClearPlanResponse);
}
```

Client: `planv1connect.PlanServiceClient`, constructed in `internal/tui/client.go` with the same HTTP client, base URL, gzip, and bearer-token interceptor as the existing `TaskServiceClient`.

## PlanEntry (response shape)

```
message PlanEntry {
  string day = 1;             // YYYY-MM-DD
  int32  id = 2;              // server-assigned, per-day sequential, never renumbered
  int64  task_id = 3;         // 0 ⇒ event (no linked task)
  string name = 4;            // display override; empty ⇒ fall back to linked task name
  int32  start_minute = 5;    // 0..1439, minutes since local midnight
  int32  duration_minute = 6; // > 0; start + duration <= 1440
  bool   completed = 7;       // derived server-side: task_id != 0 AND task completed; always false for events
}
```

TUI usage notes:
- `completed` is **read-only and server-derived** — the TUI never sets it and offers no complete-toggle (FR-009; spec Assumptions). It drives the strikethrough styling only.
- `name` empty ⇒ the grid label already falls back to the linked task's name (existing `RenderGrid` behavior; unchanged).

## Requests the TUI issues

| RPC | Request fields the TUI sets | Notes |
|-----|-----------------------------|-------|
| `ListPlanEntries` | `day` | Called on tab activation, after each mutation, on day-nav, and on manual refresh (FR-024). Response: `repeated PlanEntry entries`. |
| `AddPlanTask` | `day`, `task_id`, `start_minute`, `duration_minute` | `duration_minute = 0` ⇒ **server chooses**: `(estimate − completed) * 30` if positive, else 30. Response: `AddPlanTaskResponse{ entry }` → used to highlight the new entry. |
| `AddPlanEvent` | `day`, `name`, `start_minute`, `duration_minute` | `duration_minute = 0` ⇒ server uses the 30-minute default. Response: `AddPlanEventResponse{ entry }`. |
| `RenamePlanEntry` | `day`, `id`, `name` | `NotFound` if the id is gone (surface as error). |
| `MovePlanEntry` | `day`, `id`, `start_minute`, `duration_minute` | `duration_minute = 0` ⇒ **keep existing duration unchanged**. |
| `RemovePlanEntry` | `day`, `id` | `NotFound` handled as a readable error. |
| `ClearPlan` | `day`, `start_minute` | Removes entries at/after `start_minute`; may trim one straddling entry. Response: `ClearPlanResponse{ deleted_count, trimmed_straddling_entry }` (TUI just refetches). |

## Error handling

ConnectRPC error codes map to readable status-bar messages (FR-021), reusing the CLI's existing `UserMessage`/precondition handling semantics:
- `NotFound` → "entry not found".
- `InvalidArgument` / `FailedPrecondition` → the server's user-facing message (bad time range, overlap, etc.).
- other → generic "error: …".

On any error the TUI leaves `plan.entries` at the last good state and does not advance the sub-mode in a way that loses the user's input where recoverable.

## Time/duration input formats (client-side, before RPC)

Parsed by `internal/cli/timeparse` (same as the CLI, FR-016) into the integer minute fields above:
- Start: `13:15`, `1315`, `1:15pm`, `01:15 PM` → `start_minute`.
- Duration: `90m`, `1h`, `1h30m` → `duration_minute`; or an **end time** in any start format → converted to a duration relative to start.
- Empty duration → `0` (server default per the table).
