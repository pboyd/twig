# Contract: Auto-Schedule a Task

This feature exposes no new network API. Its contracts are (1) a TUI key-binding behavior, (2) a pure internal function signature, and (3) reuse of an existing RPC. All three are committed here before implementation per the project constitution (Principle II / Quality Gate 1).

## 1. TUI keybinding contract (planning tab)

| Aspect | Contract |
|---|---|
| Key | `a` |
| Binding name | `PlanAutoSchedule` in `internal/tui/keymap.go` |
| Context | Active only in the planning tab list view (`planMode == planList`), not while a planning modal/form is open |
| Help | Appears in the planning `ShortHelp`/`FullHelp` listing alongside `t` (add task), `e` (add event), `enter` (edit), etc. |
| Help label | `"a"`, `"auto-schedule"` |

**Behavior:**

| Precondition | Effect |
|---|---|
| Highlighted entry is a task (`TaskId != 0`) and a fitting slot exists | Entry is moved (timed) to the earliest fitting start ≥ floor; highlight follows the entry; no notice required on success |
| Highlighted entry is a task but no slot fits before midnight | No change; playful notice explaining the day is full |
| Highlighted entry is an event (`TaskId == 0`) | No change; playful notice that auto-schedule applies only to tasks |
| No entry highlighted / empty day | No change; no error |
| Fitting slot equals the task's current start | No change; no RPC issued (no-op) |

All notices use the warm/witty house tone (Principle IV) via the existing `m.notice` field.

## 2. Internal function contract (`internal/cli`)

```go
// AutoScheduleSlot returns the earliest start minute (minute-of-day) at or after
// floorMin where a block of needed length fits without overlapping any timed entry,
// within a single day ending at 1440 (midnight).
//
//   timed       — the day's timed entries (StartMinute != nil). Entries with
//                 Id == excludeID are ignored (the entry being moved).
//   durationMin — the task's own duration; if <= 0, a 30-minute block is used for fitting.
//   floorMin    — earliest allowed start (e.g. 480 for 08:00, or current minute on today).
//   excludeID   — entry Id to exclude from obstacles; pass 0 to exclude nothing.
//
// Returns ok == false when no qualifying slot exists before 1440.
func AutoScheduleSlot(timed []*planv1.PlanEntry, durationMin, floorMin int, excludeID int32) (startMin int, ok bool)
```

**Guarantees:**

- `ok == true` ⇒ `floorMin ≤ startMin` and `startMin + max(durationMin,30→) ≤ 1440` and `[startMin, startMin+block)` overlaps no obstacle.
- Earliest-fit: no smaller `startMin` satisfies the guarantees.
- Pure: no I/O, no mutation of inputs (does not change any entry's `DurationMinute`).
- Obstacles are the union of `[start, start+duration)` over timed entries (excluding `excludeID`); overlapping obstacles handled as union.

## 3. Reused RPC contract (unchanged)

`plan.v1.PlanService/MovePlanEntry` — invoked via the existing `movePlanCmd`:

```
MovePlanEntryRequest {
  day:             <plan day, YYYY-MM-DD>
  id:              <entry Id>
  start_minute:    <startMin>        // set (timed = true)
  duration_minute: <entry's existing DurationMinute>
}
```

On success the client emits `planMutatedMsg{highlightID: id}`, which reloads the day and re-highlights the moved entry. No change to the proto, server handler, or database is made by this feature.

## Contract test mapping

- §2 guarantees → `internal/cli/plan_grid_test.go` table tests: empty day, gap-after-obstacle, too-small-gap-skipped, exclude-self, no-fit, floor respected, zero-duration→30-min block, block-must-end-by-1440.
- §1 behavior → `internal/tui/update_test.go`: task moved + correct request, event no-op + notice, no-fit notice, today-floor (no past placement), already-in-place no-op.
