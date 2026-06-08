# Phase 1 Data Model: Auto-Schedule a Task

This feature introduces **no new persisted entities and no schema changes**. It operates entirely on the in-memory plan-day state the TUI already holds and writes back through the existing `MovePlanEntry` RPC. The "model" here is the conceptual shape the algorithm reasons over.

## Existing entities (reused, unchanged)

### PlanEntry (`planv1.PlanEntry`)

The unit being scheduled. Relevant fields:

| Field | Type | Meaning for this feature |
|---|---|---|
| `Id` | `int32` | Identifies the entry to move (and to exclude from its own obstacle set). |
| `TaskId` | `int64` | `0` ⇒ event (auto-schedule is a no-op); non-zero ⇒ task (eligible). |
| `Name` | `string` | Used only in user-facing notices. |
| `StartMinute` | `*int32` | `nil` ⇒ untimed; non-nil ⇒ timed at that minute-of-day. Auto-schedule sets this. |
| `DurationMinute` | `int32` | Block length to fit. `0` ⇒ use the 30-minute default for fitting (value left unchanged on write). |

### Plan day state (`planState` in `internal/tui/model.go`)

| Field | Type | Role |
|---|---|---|
| `day` | `string` (`YYYY-MM-DD`) | The day being scheduled; compared against today for the floor. |
| `entries` | `[]*planv1.PlanEntry` | Full entry list for the day; split into timed (obstacles) and untimed. |
| `cursor` | `int` | Index of the highlighted entry — the auto-schedule subject. |

## Derived / transient concepts (computed, not stored)

### Scheduling floor (`floorMin int`)

- `480` (08:00) when `day != today`.
- `max(480, currentLocalMinuteOfDay)` when `day == today`.
- Lower bound (inclusive) for any chosen start.

### Obstacle interval

For each timed entry `e` where `e.Id != excludeID`: the half-open interval `[*e.StartMinute, *e.StartMinute + e.DurationMinute)`. Overlapping obstacles are treated as their union. Portions before `floorMin` are irrelevant; portions are bounded above by `1440`.

### Free slot / gap

A maximal sub-interval of `[floorMin, 1440)` covered by no obstacle. The task **fits** a gap when `neededBlock ≤ (gapEnd − gapStart)` where `neededBlock = DurationMinute > 0 ? DurationMinute : 30`.

### Result (`startMin int, ok bool`)

- `ok == true`: `startMin` is the start of the earliest fitting gap → passed to `MovePlanEntry` as the new `StartMinute` with `timed = true` and the entry's existing `DurationMinute`.
- `ok == false`: no gap fits before `1440` → plan unchanged, playful notice shown.

## Validation rules (from requirements)

1. Eligible only when the highlighted entry exists and `TaskId != 0` (FR-001, FR-009, FR-010).
2. `startMin ≥ floorMin` always (FR-002, FR-003).
3. `startMin + neededBlock ≤ 1440` — the block must end by end of day (FR-006).
4. Chosen `startMin` does not overlap any obstacle (FR-002, FR-004).
5. The moved entry excludes itself from obstacles (FR-005).
6. If `startMin == currentStartMinute` (already-timed task), treat as no-op (FR-011).
7. The entry's stored `DurationMinute` is never altered by the 30-minute fitting default (FR-013).

## State transition

```
highlighted task entry (untimed OR timed at X)
        │  press `a`, AutoScheduleSlot → (startMin, ok=true), startMin ≠ X
        ▼
timed task entry at startMin (duration unchanged), highlight follows it
```

No-op transitions: highlighted event, no highlight, `ok=false`, or `startMin == X` → state unchanged (notice for the event / no-fit cases).
