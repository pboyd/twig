# Data Model: TUI Auto-Refresh

**Feature**: 069-tui-auto-refresh | **Date**: 2026-07-27

This feature stores nothing. There is no database table, no proto message, and no config key. The "data model" is the in-memory TUI state added to `Model` and the fields added to four existing Bubble Tea messages.

## Entity: Tab load timestamp

Per-tab record of when that tab's displayed data was last loaded or last had a background load dispatched. Drives the staleness decision in FR-007.

| Field | Type | Location | Zero value meaning |
|---|---|---|---|
| `tasksLastLoad` | `time.Time` | `Model` (Tasks state is flat on `Model`) | never loaded — treated as stale |
| `lastLoad` | `time.Time` | `goalState` | never loaded — treated as stale |
| `lastLoad` | `time.Time` | `planState` | never loaded — treated as stale |
| `lastLoad` | `time.Time` | `reportState` | never loaded — treated as stale |

**Write points** (all use `m.nowOrDefault()` so tests can inject time):

1. On successful arrival in each of the four result handlers, regardless of what triggered the load.
2. At the moment a background refresh is dispatched, before the command runs (see research Decision 3).

**Read point**: the `autoRefreshTickMsg` handler, for the active tab only.

**Validation rule**: a tab is stale when `m.nowOrDefault().Sub(lastLoad) > 10 * time.Minute`. A zero `time.Time` is always stale, which is harmless — the tab-switch and `Init` paths load every tab before any beat can act on it.

**Why not one shared timestamp**: the background refresh targets the active tab only (FR-008), and tabs load at different moments. A single timestamp would let a Plan-tab load suppress a stale Tasks-tab refresh.

## Entity: Refresh trigger

Not a struct — a boolean carried on each result message, distinguishing "the user asked for this" from "the clock asked for this".

| Message | Added field | Type |
|---|---|---|
| `listTasksResultMsg` | `bg` | `bool` |
| `listGoalsResultMsg` | `bg` | `bool` |
| `planEntriesMsg` | `bg`, `bgDay` | `bool`, `string` |
| `reportResultMsg` | `bg` | `bool` |

`bgDay` records which day the background Plan refresh was dispatched for, so a late result cannot overwrite a different day (research Decision 4). It is empty for user-initiated loads.

**Behavior the flag selects**, per FR-014 through FR-021:

| Concern | `bg == false` | `bg == true` |
|---|---|---|
| Error on failure | assigned to the tab's `err` field | discarded; handler returns early |
| Existing error on success | cleared, as today | left in place |
| Result after tab switch | applied, as today | discarded |
| Report scroll offset | reset to 0 | preserved |
| Cursor anchoring | by ID (newly uniform) | by ID |
| Open interactive mode | untouched, as today | untouched |

The last two rows are identical by design. Cursor anchoring becomes ID-based for both flavors (research Decision 5), and no load path forces `m.mode`, so nothing extra is needed to satisfy FR-014. Only three rows actually branch.

## Entity: Heartbeat

| Item | Type | Notes |
|---|---|---|
| `autoRefreshTickMsg` | `struct{}` | Carries nothing; the handler reads all it needs from `Model`. |
| `autoRefreshTickCmd()` | `tea.Cmd` | `tea.Tick(time.Minute, …)`. Started once in `Init`; rescheduled only by its own handler. |

**Invariant (FR-010)**: exactly two references to `autoRefreshTickCmd` may exist in non-test code — the one in `Init` and the one at the end of its own handler. Any third reference is a defect, because the tick never self-terminates and a second chain would permanently double the beat rate.

## Constants

| Name | Value | Requirement |
|---|---|---|
| `autoRefreshInterval` | `10 * time.Minute` | FR-007, FR-009 — the staleness threshold, not user-configurable |
| `autoRefreshTickRate` | `time.Minute` | FR-005 — how often staleness is evaluated |

## State transitions

The heartbeat handler is a pure decision over existing state:

```text
autoRefreshTickMsg received
  │
  ├─ not m.autoRefreshEligible()  ──────────────→ reschedule only        (FR-012, FR-013)
  │
  ├─ active tab's lastLoad within 10 minutes ───→ reschedule only        (FR-007)
  │
  └─ otherwise ─→ stamp lastLoad = now                                   (Decision 3)
                  dispatch active tab's fetch command with bg = true     (FR-008)
                  reschedule                                             (FR-010)
```

`autoRefreshEligible()` is true only when `m.mode == modeList`, `!m.confirmingQuit`, `!m.confirmingDiscard`, and the active tab's own mode is at rest (`m.goal.mode == goalList` for Goals, `m.plan.mode == planList` for Plan; Tasks and Report have no sub-mode of their own).

## Relationships to existing state

- **`pendingComplete`** (`Model` and `planState`): read by `buildVisible` and `displayedPlanEntries`, both of which already run inside the load handlers. A background refresh therefore preserves just-completed-item retention with no extra work.
- **`expanded`**, **`filterExpr`**, **`filteredIDs`**, **`listScroll`**: untouched by the load handlers today; FR-017 is satisfied by not changing that.
- **`nowFunc`**: existing test-only time injection, reused for staleness comparison.
- **`filterGen`**: unrelated, but the precedent for guarding stale async responses.
