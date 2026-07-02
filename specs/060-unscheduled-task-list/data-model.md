# Phase 1 Data Model: Unscheduled Tasks as a List

This feature is presentation-only. **No new entities, fields, or persisted state are introduced.**
It re-renders data the Plan tab already loads. This document records the existing shapes the renderer
reads, and the render-time values derived from them.

## Existing entities consumed (no changes)

### PlanEntry (`api` — `plan.v1.PlanEntry`)

The Plan tab loads a day's entries into `m.plan.entries`. Fields the untimed renderer reads:

| Field | Type | Role in this feature |
|-------|------|----------------------|
| `Id` | int32 | Selection identity (`opts.SelectedID` match). Not displayed (no `[id]` prefix). |
| `Name` | string | The row label; rendered inline, truncated to available width. |
| `Completed` | bool | Chooses checkbox glyph: `true → ☑`, `false → ☐`; drives strike/dim on the name. |
| `StartMinute` | *int (nullable) | **Partition key** — `nil` ⇒ untimed (this list); non-nil ⇒ timed (grid). Unchanged. |
| `DurationMinute` | int32 | **No longer read** for untimed rendering (was used for box height). |
| `TaskId` | int32 | Unchanged; used elsewhere (details pane), not by the row renderer. |

**Partitioning** (unchanged): `splitPlanEntries` in `plan_view.go` splits `entries` into `untimed`
(`StartMinute == nil`) and `timed`. Only the *rendering* of the `untimed` slice changes.

## Render-time values (derived, not persisted)

These describe what one untimed row is composed of — see `contracts/untimed-list-rendering.md` for
the exact string contract.

| Value | Derivation |
|-------|-----------|
| `checkbox` | `entry.Completed ? "☑" : "☐"` (styled/TTY only; omitted in plain mode) |
| `name` | `md.RenderInline(entry.Name, …)` truncated to `width - prefixWidth`; struck when `Completed` |
| `selected` | `isTTY && opts.SelectedID != 0 && entry.Id == opts.SelectedID` |
| `line count` | Exactly `1` per untimed entry (was `max(1, DurationMinute/15)`) — feeds the grid height math |

## State transitions (unchanged behavior, FR-007)

No new transitions. For reference, the existing lifecycle that moves an entry between the two render
regions is preserved:

```
untimed (StartMinute == nil)  ──schedule (assign start time)──▶  timed (grid block)
        ▲                                                              │
        └────────────────── unschedule (clear start time) ────────────┘
```

Completion toggles `Completed` in place (row stays in the list, checkbox flips ☐↔☑). Reordering
changes the entry's position among untimed siblings (existing `reorderPlanEntryCmd`), unaffected by
the row format.

## Validation rules

None added. Existing visibility rules for completed untimed entries (show/hide) are applied upstream
of the renderer and are unchanged (FR-010).
