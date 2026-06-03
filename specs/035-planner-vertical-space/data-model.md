# Phase 1 Data Model: Planner Vertical Space

This feature adds no persistent entities, no DB tables, and no proto messages. The
only "data" is an in-memory, per-render value object: the **visible window**.

## Visible Window (value object)

Computed fresh on every render; never stored.

| Field | Type | Meaning |
|-------|------|---------|
| `startMin` | int (minutes since midnight) | First minute drawn by the timed grid. Aligned to a 15-minute block. |
| `endMin` | int (minutes since midnight) | Last minute drawn (inclusive of the closing hour line). `endMin > startMin`. |

Derived quantities (already used by `RenderGrid`):
- `totalLines = (endMin - startMin)/15 + 1`
- A row is an hour divider when its minute `% 60 == 0`.

### Invariants

- `0 <= startMin < endMin <= 24*60` (1440).
- In **fill** and **top-truncate** modes, `startMin == baseStart` (the entry-
  extended 08:00 default).
- In **anchor** mode, `startMin == snapDown15(nowMinutes)`.
- The window never excludes a row an entry occupies **when** that side is not
  deliberately overridden (FR-007). Anchor mode intentionally clips rows earlier
  than `now` (FR-003).

## Inputs to the window computation

| Input | Source | Notes |
|-------|--------|-------|
| `entries` | timed plan entries for the day | Used to compute `baseStart`/`baseEnd` via existing snap-and-extend logic. |
| `now` | `time.Now()` passed through the view | Anchor block = `snapDown15(now.Hour()*60 + now.Minute())`. |
| `day` | `m.plan.day` (`YYYY-MM-DD`) | Anchor mode applies only when `day == now`'s date. |
| `availableRows` | inner pane height − untimed pane lines − separator lines | Rows the timed grid may occupy. |

## Mode decision (state-free)

```
baseStart, baseEnd = entryExtendedDefault(entries)   // 08:00..17:00 expanded to entries
baseRows           = (baseEnd - baseStart)/15 + 1

if availableRows >= baseRows:           # FILL
    start = baseStart
    end   = min(start + (availableRows-1)*15, 1440)
elif isToday(day, now) and nowBlock >= baseStart:   # ANCHOR
    start = snapDown15(nowBlock)
    end   = min(start + (availableRows-1)*15, 1440)
else:                                   # TOP-TRUNCATE (non-today or now before window)
    start = baseStart
    end   = start + (availableRows-1)*15
```

No transitions/lifecycle — the window is recomputed from scratch each frame
(FR-006).

## Untimed entries

Unchanged data model. They are rendered by the existing `RenderUntimed` and are
counted (their line total) to derive `availableRows`. They are always laid out
above the timed grid and take priority for vertical space (FR-004).
