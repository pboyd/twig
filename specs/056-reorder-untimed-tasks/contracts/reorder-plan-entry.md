# Contract: ReorderPlanEntry RPC

**Service**: `plan.v1.PlanService` (existing — one method added)
**Source of truth**: `api/proto/plan/v1/plan.proto` (regenerate stubs with `make proto`)
**Auth**: same as all ConnectRPC routes — wrapped by `auth.Middleware`; the caller's user is taken from the session/API key.

This contract MUST be defined and committed before handler or TUI implementation (Constitution Principle II).

## Proto additions

```proto
service PlanService {
  // ... existing methods (ListPlanEntries, AddPlanTask, AddPlanEvent,
  //     RemovePlanEntry, RenamePlanEntry, MovePlanEntry, ClearPlan,
  //     ListScheduledDays) ...

  // ReorderPlanEntry repositions an untimed entry within the day's untimed
  // group by placing it immediately before or after a sibling untimed anchor.
  // Both the entry and the anchor must be untimed (no start_minute) and on the
  // same day. Timed entries cannot be reordered or used as anchors.
  rpc ReorderPlanEntry(ReorderPlanEntryRequest) returns (ReorderPlanEntryResponse);
}

message ReorderPlanEntryRequest {
  // The day the entry belongs to (YYYY-MM-DD), matching ListPlanEntriesRequest.day.
  string day = 1;

  // The untimed entry to move (per-day entry id).
  int32 id = 2;

  // Where to place it, relative to a sibling untimed anchor on the same day.
  // Exactly one must be set.
  oneof anchor {
    // Place id immediately before this untimed sibling.
    int32 before_id = 3;
    // Place id immediately after this untimed sibling.
    int32 after_id = 4;
  }
}

message ReorderPlanEntryResponse {
  // The day's untimed entries (including the moved one) in their new order,
  // each carrying its updated position. Ordered as the client should display
  // them in the untimed pane.
  repeated PlanEntry untimed = 1;
}
```

> `PlanEntry` is the existing message (`user_id`/`day`/`id`/`task_id`/`name`/`optional start_minute`/`duration_minute`). No new field is exposed on `PlanEntry` itself; `position` stays a server-side ordering detail and the returned slice is already in order.

## Behavior

| Condition | Result |
|---|---|
| Valid move, entry not at the relevant boundary | Untimed group renumbered `0..n-1`; response returns the new order. `OK`. |
| Move first-higher or last-lower (boundary no-op) | Positions unchanged; response returns the current order. `OK`. (FR-005) |
| Neither / both anchor fields set | `INVALID_ARGUMENT` — "exactly one of before_id or after_id must be set". |
| `anchor == id` | `INVALID_ARGUMENT` — anchor must differ from the moved entry. |
| Moved entry or anchor is timed (`start_minute` set) | `INVALID_ARGUMENT` — only untimed entries can be reordered. (FR-004/FR-007) |
| Moved entry or anchor not found on `day` for the user | `NOT_FOUND`. |
| Anchor on a different day than the entry | `INVALID_ARGUMENT` — anchor must be in the same day's untimed group. |
| `day` not a valid `YYYY-MM-DD` | `INVALID_ARGUMENT`. |

All mutations run in a single transaction with the day's untimed entries locked `FOR UPDATE`, so the returned set is exactly the input set, renumbered (SC-003), and concurrent reorders converge.

## Client usage (TUI)

- `{` (RankUp) on an untimed entry → `ReorderPlanEntry{day, id, before_id: <prev visible untimed entry id>}`.
- `}` (RankDown) on an untimed entry → `ReorderPlanEntry{day, id, after_id: <next visible untimed entry id>}`.
- When the highlighted entry is timed, or is the first/last visible untimed entry for the chosen direction, the TUI does not send a request (local no-op).
- After a successful response the TUI reloads the day (or applies the returned `untimed` slice) and re-highlights the moved entry id (FR-003, FR-008).

## Out of scope

- No CLI command and no web control invoke this RPC for now (FR-009). Those surfaces only display the resulting order via the existing `ListPlanEntries`/listing query.
