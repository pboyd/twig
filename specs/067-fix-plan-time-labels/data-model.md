# Phase 1 Data Model: Show Actual Times in Plan Grid Labels

No persisted schema changes. No protobuf changes. No migrations. Plan entries already carry exact times; this feature only corrects how two consumers interpret them.

What follows documents the in-memory distinction the fix establishes.

## Plan Entry (`planv1.PlanEntry`) — unchanged

| Field | Type | Notes |
|---|---|---|
| `Id` | `int32` | Server-assigned. `-1` is reserved for the transient preview. |
| `Name` | `string` | Displayed after the time range in the label. |
| `StartMinute` | `*int32` | Minutes since midnight. `nil` ⇒ untimed; such entries never reach the grid. |
| `DurationMinute` | `int32` | Exact duration. Server defaults a blank form field to 30. |

### Derived intervals

The central point of this feature: one entry yields **two** intervals, and each has exactly one legitimate consumer set.

| Interval | Definition | Consumed by | Changed? |
|---|---|---|---|
| **Exact** | `[start, start + duration)` | Entry label text; conflict detection | Newly used by both |
| **Snapped** | `[snapDown15(start), snapUp15(start + duration))`, widened to at least one slot | Row placement, window sizing, "now" anchor, gutter | No change |

**Invariant**: Snapped always contains Exact. Consequently any slot derived from a sub-range of Exact is guaranteed to fall inside the entry's drawn box — the property that lets conflict slots render safely.

**Validation rules** (from the requirements):

- Exact interval is half-open, so an entry ending at minute *m* and one starting at minute *m* do not overlap (FR-004).
- Exact end may equal exact start when duration is 0; the label reports this honestly, and the snapped interval still guarantees a visible one-slot box (research.md, Decision 2).
- Only the snapped interval may influence geometry. Only the exact interval may influence label text and conflict decisions. No value crosses over — this separation *is* the fix.

## Preview Entry — unchanged shape

A transient `planv1.PlanEntry` built by `buildPlanPreview` from live form fields, carrying `Id = previewID (-1)`. Never persisted, never sent to the server. It obeys the same two-interval model as a saved entry.

## Conflict Slot Set — semantics changed

Type is unchanged: `map[int]bool`, keyed by slot-start minute (a multiple of 15), passed to the renderer as `GridOptions.PreviewConflictSlots`. The renderer's consumption at `internal/cli/plan_grid.go:193` and `:215` is untouched.

| | Before | After |
|---|---|---|
| Membership test | slot `[t, t+15)` intersects another entry | exact overlap region intersects slot `[t, t+15)` |
| Slots considered | every slot of the preview | only slots spanning the overlap region |
| Touching entries | false conflict | no conflict |
| Sub-slot neighbors (13:00–13:05 vs 13:10–13:20) | false conflict | no conflict |

**State transitions**: The set is recomputed from scratch on every render while a timed form is open (`planTaskTime`, `planEventForm`, `planEdit`); it is `nil` otherwise. No accumulation, no caching, no lifecycle to manage.
