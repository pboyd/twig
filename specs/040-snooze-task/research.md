# Phase 0 Research: Snooze a Task

All unknowns from Technical Context are resolved below. No open `NEEDS CLARIFICATION`.

## Decision 1 — How to store the snooze day

**Decision**: Add a nullable `snooze_until TIMESTAMPTZ` column on `tasks`, exposed as a `google.protobuf.Timestamp snooze_until` field. The value is pinned to **midnight UTC of the chosen calendar day**, exactly how the existing `due` field treats a bare `YYYY-MM-DD` (`internal/cli/render.go` `ParseDue`: `time.Date(y, m, d, 0,0,0,0, time.UTC)`).

**Rationale**:
- Maximum reuse of an established, tested pattern (`due` is already `TIMESTAMPTZ` → proto `Timestamp`, parsed by `ParseDue`, formatted via `.UTC().Format`). Principle I (Simplicity) and III (Consistency).
- No new proto types, no new parsing/formatting helpers needed.

**Alternatives considered**:
- *Dedicated `DATE` column / proto `string` "YYYY-MM-DD"*: more honest about day-granularity, but protobuf has no date scalar, sqlc would surface a different Go type, and it diverges from `due`. Extra code for no user-visible benefit — rejected.
- *Store an absolute instant and snooze by time-of-day*: contradicts the spec's day-granularity decision; rejected.

## Decision 2 — Where snooze filtering happens

**Decision**: **Client-side**, in all three clients. `ListTasks` is unchanged and returns every task (including snoozed ones, with their `snooze_until`). Each client hides future-snoozed tasks from its default view.

**Rationale** (clarified with the user, 2026-06-06):
- Mirrors the existing completed-task filter, which is already client-side (`completed_at` is sent to clients; the TUI `buildVisible`, CLI `pruneIncomplete`, and web `filterTree` each filter locally).
- The TUI "Show all" must be able to reveal snoozed tasks; if the server filtered them out, that would require a new request parameter — avoided.
- The server never needs to know the user's timezone.

**Alternatives considered**:
- *Server-side filtering with an `include_hidden` request flag on `ListTasks`*: adds API surface, splits the filtering model from the completed case, and pushes timezone awareness into the server. Rejected.

## Decision 3 — Timezone semantics ("the day arrives")

**Decision**: A task is **snoozed (hidden)** iff the **UTC calendar date** of `snooze_until` is **strictly after** the client's **local calendar date** (`today` in the client's local timezone). Equivalently: visible iff `snoozeDate ≤ localToday` or `snooze_until` is unset.

**Rationale**:
- Because the stored value is midnight-UTC of the entered day, extracting its UTC calendar date recovers exactly the day the user typed, regardless of timezone.
- Comparing that day against the client's *local* today gives the spec's "becomes visible at the start of that day in the user's local timezone" behavior.
- Worked example (client at UTC−8, snooze = 2026-06-10 → stored `2026-06-10T00:00:00Z`): at local `2026-06-09 23:00` (UTC `2026-06-10 07:00`) localToday=`2026-06-09` < `2026-06-10` → hidden; one minute into `2026-06-10` local, localToday=`2026-06-10` = snoozeDate → visible. Wakes at local start-of-day. ✓

**Implementation notes per client**:
- **Go (TUI/CLI)**: `snoozeT := task.GetSnoozeUntil().AsTime().UTC()`; `snoozeDate := time.Date(snoozeT.Year(), snoozeT.Month(), snoozeT.Day(), 0,0,0,0, time.UTC)`; `localToday := <now>.Local()` truncated to its Y/M/D as a UTC-keyed date; hidden iff `snoozeDate.After(localTodayAsDate)`. Inject "now" so tests are deterministic (see Decision 5).
- **TypeScript (web)**: read `snoozeUntil` (a `Timestamp`), build a UTC `Date`, compare `getUTCFullYear/Month/Date` to the local `new Date()`'s `getFullYear/Month/Date`.

## Decision 4 — Un-snooze and past dates

**Decision**: Clearing the snooze field, or setting a day ≤ today, results in the task being treated as not snoozed. No separate "un-snooze" action is added.

**Rationale**:
- `UpdateTask` is full-replace: an omitted optional field is cleared server-side. The TUI edit form sending an empty snooze string ⇒ no `snooze_until` on the request ⇒ column set to NULL. Editing to clear is therefore free.
- A past/today value simply fails the "strictly after local today" predicate, so it shows — no validation/rejection needed (FR-004, spec Assumptions).

**Alternatives considered**: a dedicated `Snooze`/`Unsnooze` RPC — unnecessary given full-replace `UpdateTask`; rejected (Principle I).

## Decision 5 — Testability of "now"

**Decision**: The snooze visibility predicate must take an injected reference time so unit tests are deterministic (no real-clock flakiness), consistent with how the codebase already passes explicit times (e.g., CLI `runComplete` captures `before := time.Now()`).

**Rationale**: The TUI `buildVisible`/`emitNode` and CLI prune functions are unit-tested via `export_test.go` shims. Threading a `now`/`today` value (rather than calling `time.Now()` deep inside) keeps these tests hermetic. The web `filterTree` similarly should accept an optional reference date.

## Decision 6 — TUI filter state representation

**Decision**: Keep a single boolean toggle (the existing `m.showCompleted`) but broaden its meaning to "show everything" (completed **and** snoozed). Relabel user-facing strings to **"Show all" / "Show only pending"**; the `c` keybinding and help entry are updated. Optionally rename the field to `showAll` for clarity.

**Rationale**: The clarified design is a two-state toggle, so the existing boolean suffices — no new state machine. Principle I. The only required changes are the predicate (now also considers snooze) and the copy.

**Alternatives considered**: a three-state cycle (pending / +completed / +snoozed) — explicitly out of scope per the spec Assumptions; rejected.
