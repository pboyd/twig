# Contract: Task API additions for Snooze

**Source of truth**: `api/proto/task/v1/task.proto`. This document specifies the additions; after editing the proto, run `make proto` (Go stubs) and `npm run gen` (web). API-first per Constitution Principle II — committed before implementation.

## Proto additions

### `Task` message

Add one field (next available number is **10**; `position = 9` is the current max):

```proto
message Task {
  // ... existing fields 1–9 ...

  // Optional day before which this task is hidden from default ("pending")
  // views. Pinned to midnight UTC of the chosen calendar day. Unset when the
  // task is not snoozed. Clients hide the task while this day is strictly after
  // the client's local current day; on/after that day the task is shown.
  google.protobuf.Timestamp snooze_until = 10;
}
```

### `CreateTaskRequest`

```proto
message CreateTaskRequest {
  string name = 1;
  string description = 2;
  google.protobuf.Timestamp due = 3;
  optional int64 parent_id = 4;
  // Optional snooze day (midnight UTC). Unset = not snoozed.
  google.protobuf.Timestamp snooze_until = 5;
}
```

### `UpdateTaskRequest`

```proto
message UpdateTaskRequest {
  int64 id = 1;
  string name = 2;
  string description = 3;
  google.protobuf.Timestamp due = 4;
  optional int64 parent_id = 5;
  // Full-replace: unset clears the snooze (un-snoozes the task).
  google.protobuf.Timestamp snooze_until = 6;
}
```

No other messages change. **`ListTasks` is intentionally unchanged** — it returns every task (snoozed included) so clients can filter locally and the TUI "Show all" can reveal snoozed tasks.

## Semantics

| Aspect | Behavior |
|---|---|
| Encoding | A calendar day `YYYY-MM-DD` is stored/sent as `YYYY-MM-DDT00:00:00Z` (reuses `ParseDue`'s bare-date rule). |
| Create | `snooze_until` set ⇒ new task starts snoozed until that day. Unset ⇒ not snoozed. |
| Update | Full-replace: present ⇒ set/overwrite; absent ⇒ cleared (un-snooze). Consistent with how `due` and `parent_id`-less updates already clear fields. |
| Read | `Task.snooze_until` is populated from the column when set; omitted when NULL. |
| Validation | None beyond a well-formed timestamp. A day ≤ today is accepted and is inert (task shows). No new error codes. |
| Visibility | **Not** enforced by the server. Each client hides a task while `UTC-date(snooze_until) > client-local-today` (see data-model.md). |

## Handler mapping (`services/twig/internal/handler/task.go`)

- `dbTaskToProto`: after the `completed_at` block, add
  `if t.SnoozeUntil.Valid { pt.SnoozeUntil = timestamppb.New(t.SnoozeUntil.Time) }`.
- `CreateTask` / `UpdateTask`: mirror the existing `due` handling —
  `if req.Msg.SnoozeUntil != nil { params.SnoozeUntil = pgtype.Timestamptz{Time: req.Msg.SnoozeUntil.AsTime(), Valid: true} }`.

## Backward compatibility

- Adding optional proto fields is wire-compatible; older clients ignore `snooze_until` and simply never hide snoozed tasks (acceptable — they predate the feature).
- The new DB column is nullable with no default, so existing rows are `NULL` (not snoozed). The migration is additive and reversible.
