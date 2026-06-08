# Phase 0 Research: Auto-Schedule a Task

All Technical Context items are resolved; no open `NEEDS CLARIFICATION` remained after `/speckit-clarify`. This file records the codebase findings that shaped the approach.

## Decision: Reuse `MovePlanEntry` — no new RPC

- **Decision**: Auto-schedule performs its move through the existing `plan.v1.PlanService/MovePlanEntry` RPC via the existing `movePlanCmd` helper in `internal/tui/plan_update.go`.
- **Rationale**: The operation is exactly "give this existing entry a start time (timed) with its duration." `MovePlanEntry` already does this (`StartMinute *int32`, `DurationMinute int32`, `timed bool`). Reusing it honors Principle II (no new contract) and Principle I (no new code path). It also means the server, proto, and DB layers are untouched.
- **Alternatives considered**: A dedicated `AutoSchedulePlanEntry` RPC that computes the slot server-side. Rejected: it duplicates placement logic the client already has the data for, adds a proto/handler/DB round of changes, and provides no benefit since the client already holds the day's full entry list.

## Decision: Slot-finder is a pure function in `internal/cli`

- **Decision**: Add `AutoScheduleSlot(timed []*planv1.PlanEntry, durationMin, floorMin int, excludeID int32) (startMin int, ok bool)` to `internal/cli/plan_grid.go`, unit-tested in `plan_grid_test.go`.
- **Rationale**: `internal/cli` already hosts the pure, rendering-free plan-day helpers (`GridWindow`, `splitPlanEntries`-style logic, the 1440 end-of-day clamp). Placing the algorithm there keeps it independently table-testable without a Bubble Tea model or a database, matching the project's "no integration test infra" convention. The handler stays a thin wiring layer.
- **Alternatives considered**: Inlining the search in the `update.go` key handler. Rejected: harder to unit-test thoroughly across gap arrangements and couples algorithm to TUI state.

## Decision: Algorithm — earliest-fit forward scan

- **Decision**: Treat each *timed* entry (except the excluded one) as an occupied interval `[start, start+duration)`. Clip/union them against `[floorMin, 1440)`, then scan free intervals in ascending order; return the first interval whose length ≥ the needed block, with `startMin` = that interval's start. `ok=false` if none fits.
- **Rationale**: "Next available gap it fits into" in the spec means *earliest start*, not best-fit (Assumptions). A sort-by-start + sweep over a handful of entries is trivial and obviously correct.
- **Needed block length**: the task's `DurationMinute` if > 0, else the default **30** (FR-013). The task's stored duration is never mutated; the 30 is used only to choose the start.
- **Alternatives considered**: Best-fit / smallest-gap packing. Rejected: contradicts the spec's earliest-start definition and is more complex.

## Decision: Scheduling floor computed in the handler

- **Decision**: The handler computes `floorMin` = `480` (08:00) for any day, raised to the current local minute-of-day when `m.plan.day == time.Now().Local().Format("2006-01-02")` and that minute > 480.
- **Rationale**: The existing `planDayHeader` already determines "today" by comparing `m.plan.day` to `now.Format("2006-01-02")`; this reuses the same idiom. Handlers elsewhere in `update.go` already call `time.Now().Local()` directly (e.g. the go-to-task handler), so no plumbing of `now` is required.
- **Alternatives considered**: Always 8 AM (rejected during `/speckit-clarify` — would let tasks land in the past on today).

## Decision: "Highlight follows the task" is free

- **Decision**: No extra work for FR-014. `movePlanCmd` already returns `planMutatedMsg{highlightID: id}`, and `handlePlanEntriesMsg`/`listPlanHighlightCmd` re-select the entry with that id after the day reloads.
- **Rationale**: Auto-schedule reuses `movePlanCmd`, inheriting its post-move highlight behavior, which keeps the moved task selected.

## Decision: No-op detection for "already in place"

- **Decision**: When the highlighted task is already timed and its current `StartMinute` equals the computed `startMin`, skip the RPC and show no change (FR-011).
- **Rationale**: Avoids a spurious server write and a needless reload when pressing `a` would not move anything. Cheap equality check in the handler.

## Decision: User-facing messages (Principle IV)

- **Decision**: Use the existing `m.notice` channel for the playful one-liners, mirroring the event guards already present on the Complete/Pomodoro planning handlers.
- **Cases**:
  - Event highlighted → e.g. "That's an event — it's already pinned where it belongs. Auto-schedule only herds tasks."
  - No slot fits → e.g. "Couldn't find an open slot before midnight — the day's packed. Free something up and try again."
  - (Optional) no-duration task placed as a 30-min block → standard success path; no special message needed beyond the move taking effect.
- **Rationale**: Consistent tone and mechanism with the rest of the planning tab; final wording is polished during implementation/review.
