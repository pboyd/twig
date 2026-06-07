# Phase 0 Research: Show Scheduled Days in Task Details

## Unknowns extracted from Technical Context

The only material unknown is **how the TUI obtains the set of days a task is scheduled for**, because the existing `PlanService` is day-scoped (`ListPlanEntries(day)`) and offers no reverse lookup (task → days). The clarification in the spec ("all days, no horizon; today-or-later by local date") rules out an unbounded client-side scan, so the data source must be decided here.

---

## Decision 1: Add a server-side reverse query (new RPC) rather than client-side day scanning

**Decision**: Add a new `PlanService.ListScheduledDays` RPC backed by a single `SELECT DISTINCT task_id, day` query over `plan_entries`, filtered to `day >= from_day`. The client passes its **local** current day as `from_day`.

**Rationale**:
- The spec requires listing **every** current/future day with **no upper horizon**. A client-side approach would have to call `ListPlanEntries(day)` for an unbounded forward range of days — impossible to bound and O(days) round trips.
- The data already lives in one table (`plan_entries`) keyed by `(user_id, day, task_id)`. A reverse query is a single indexed `WHERE user_id = ? AND task_id IS NOT NULL AND day >= ?`, which is trivial and cheap.
- Aligns with **Principle II (API-First)**: the new contract is defined in `contracts/` before implementation.
- The `from_day` cutoff is applied server-side using the value the client computes from its **local** clock (per the Session 2026-06-07 clarification), keeping "past" semantics local without the server needing timezone knowledge.

**Alternatives considered**:
- *Client scans `ListPlanEntries` over a fixed future window (e.g., next 90 days)*: Rejected — imposes an artificial horizon the spec explicitly forbids ("all of them"), and is far chattier.
- *Add a `scheduled_days` field to the `Task` message in `TaskService.ListTasks`*: Rejected — couples plan data into the task contract, makes every task list pay for a join even when the details pane isn't shown, and crosses a service boundary (plan data on a task message). Keeping it on `PlanService` is cleaner and more cohesive.

---

## Decision 2: Fetch all scheduled days in bulk into a Model-held map, refreshed with task loads

**Decision**: `ListScheduledDays` returns one flat `(task_id, day)` list for **all** of the caller's tasks on/after `from_day`. The TUI groups it into a `map[int64][]string` (task ID → ascending days) stored on `Model`, and the details pane renders from that map **synchronously**. The map is (re)loaded alongside the task list: at init, on `Ctrl-R` refresh, when switching to the Tasks tab, and after a send-to-plan (`ctrl+p`) action.

**Rationale**:
- The details pane re-renders on every cursor move. A lazy per-selection RPC would add a round trip to each arrow-key press, introducing latency and async-ordering complexity (stale responses racing cursor movement).
- A single bulk fetch mirrors the existing pattern where `ListTasks` is loaded up front and rendered synchronously, satisfying **Principle I (Simplicity)** at the render hot path.
- Refreshing the map at the same points the task list refreshes (plus tab-switch and `ctrl+p`) keeps the display fresh per **SC-004** without a polling mechanism.
- The payload is naturally bounded: only scheduled tasks, only today-or-future days.

**Alternatives considered**:
- *Lazy fetch on task selection*: Rejected for the latency/race complexity above.
- *Fold the map into `listTasksResultMsg`*: Considered; kept as a **separate** command/message so the schedule load is independent of the task-list load and can be refreshed on tab-switch without re-fetching tasks. Slightly more code, but clearer separation and avoids changing the task-list message shape.

---

## Decision 3: "Past" determined by the client's local date (carried from spec clarification)

**Decision**: The client computes `from_day` from its **local** current date (`time.Now()` local, formatted `2006-01-02`) and the server returns `day >= from_day`. Days are compared as `YYYY-MM-DD` strings, which sorts chronologically.

**Rationale**:
- Plan days are defined relative to local midnight (per `plan.proto`), so a local cutoff is the consistent and intuitive choice (Session 2026-06-07 clarification).
- This intentionally differs from the existing `Snooze:` line, which formats in UTC. The divergence is documented in the spec Assumptions so a reviewer doesn't "fix" it into an inconsistency.

**Alternatives considered**:
- *UTC cutoff (matching the snooze line)*: Rejected by clarification — would show/hide a day a few hours off from the user's local sense of "today".

---

## Decision 4: Rendering follows existing details-pane conventions

**Decision**: Render a `Scheduled for: <days>` line in `renderDetails` (both the plain and styled paths), using the existing `labelStyle` (dim) for the label, comma-space joined days. Omit the line entirely when the task has no today-or-future days. Place it near the other date-oriented fields (e.g., after `Due:` / before or alongside `Snooze:`).

**Rationale**:
- Reuses the established label/value structure and theme tokens, satisfying **Principle III (UI/UX Consistency)**.
- The label text `Scheduled for:` is taken verbatim from the user's spec. It is longer than the existing column-aligned 4–10 char labels; we follow the same `labelStyle.Render(...) + " " + value` form without forcing it into the narrower column alignment (a single neutral field label, not a tone-bearing message, so **Principle IV** imposes no new copy).

**Alternatives considered**:
- *A list-row badge/icon in the tree (like the 💤 snooze marker)*: Out of scope — the spec asks specifically for the details pane. Could be a future enhancement.

---

## Test approach (informs Phase 1 / tasks)

- **Server handler test** (`services/twig/internal/handler/plan_test.go`): follows the existing pattern — `newTestPlanPool` against `DATABASE_URL`, skipped when unset. Insert plan entries across past/today/future days (timed and untimed, plus duplicate same-day entries and a second user) and assert `ListScheduledDays` returns distinct, ordered, isolation-correct, horizon-filtered results.
- **TUI render test** (`internal/tui/details_test.go`): unit-test `renderDetails` (and any pure helper that formats the scheduled-days line) with a supplied day list — no client needed. Covers: single day, multiple days (ascending, comma-space), de-duplication already handled server-side, empty → no line, both styled and plain paths.
- **TUI wiring test** (`internal/tui/*_test.go`): use the existing `fakePlanClient` pattern to assert the schedule map is loaded/refreshed at the right moments and that `from_day` is the local today.
