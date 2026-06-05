# Phase 1 Data Model: Live Plan Entry Preview with Overlap Indication

This feature introduces **no persisted data** and **no proto/DB changes**. The "entities" are transient, view-only constructs used during rendering.

## Entity: Preview Entry (transient)

A synthetic plan entry representing the entry-being-edited at the time window implied by the form's current values. Built fresh on each render from `m.plan.form` state; never stored, never sent to the server.

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| `Id` | `int32` | constant `previewID` (negative sentinel, e.g. `-1`) | Lets `RenderGrid` recognize the preview; cannot collide with real positive IDs. |
| `Name` | `string` | Name field (Edit/Event) or linked task name (TaskTime) | May be empty; the box still renders. |
| `StartMinute` | `*int32` | `timeparse.ParseStart(startField)` | Built **only** when Start parses to a valid minute; otherwise no preview entry is produced (FR-009). |
| `DurationMinute` | `int32` | `timeparse.ParseDurationOrEnd(durField, start)` | Blank/unparseable → minimal slot (grid enforces ≥15-min visual height). |

**Construction rules**
- Produced only when `m.plan.mode ∈ {planTaskTime, planEventForm, planEdit}`.
- Start invalid/empty ⇒ **no** preview entry (untimed is out of scope for the timed preview box).
- For `planEdit`, the original entry (`Id == m.plan.form.entryID`) is **excluded** from the rendered timed slice and replaced by the preview.

**Validation**: none beyond parse success — the preview intentionally reflects in-progress (even partially invalid) input by simply not appearing until Start is valid.

## Entity: Conflict Slot Set (derived)

The set of 15-minute slot-start minutes where the preview overlaps another timed entry on the same day.

| Aspect | Definition |
|--------|------------|
| Element | A slot-start minute `t` (multiple of 15) covered by the preview. |
| Membership | `t` is in the set iff some *other* timed entry's half-open interval `[start, start+dur)` intersects `[t, t+15)`. |
| Comparison set | All timed entries for the day **except** the edit target (`form.entryID`) — self-overlap excluded (FR-006). |
| Empty set | No conflict ⇒ no conflict styling rendered (FR-007 covers clearing as values change). |
| Touching boundary | Intervals that only touch (`aEnd == bStart`) do **not** produce membership (FR-008). |

Computed in `internal/tui/plan_preview.go`; passed to the renderer as `GridOptions.PreviewConflictSlots`.

## Extension: `cli.GridOptions` (internal Go API)

New fields added to the existing struct in `internal/cli/plan_grid.go`. Full semantics in [contracts/grid-preview.md](./contracts/grid-preview.md).

| Field | Type | Meaning |
|-------|------|---------|
| `PreviewID` | `int32` | `0` = no preview. Otherwise the `Id` of the entry to render with dashed runes as an unsaved preview. |
| `PreviewStyle` | `func(string) string` | Optional; styles the preview's non-conflicting runes (e.g., `dim`). `nil` ⇒ dashed runes only (plain mode). |
| `ConflictStyle` | `func(string) string` | Optional; styles the preview's conflicting rows (e.g., `errorColor` red). `nil` ⇒ plain-mode gutter `!` marker fallback. |
| `PreviewConflictSlots` | `map[int]bool` | Slot-start minutes of the preview that overlap another entry. Empty/nil ⇒ no conflict marking. |

## Relationships & flow

```text
m.plan.form.fields (live textinputs)
        │  parse (timeparse)
        ▼
Preview Entry (transient *planv1.PlanEntry, Id = previewID)
        │  + day's other timed entries (edit target excluded)
        ├────────────► Conflict Slot Set  (interval overlap, 15-min granularity)
        ▼                         │
timed slice (others + preview) ───┤
        │                         ▼
        └──► cli.RenderGrid(..., GridOptions{
                 PreviewID, PreviewStyle, ConflictStyle, PreviewConflictSlots,
                 WindowStartMin/EndMin (from GridWindow incl. preview) })
```

## State transitions (lifecycle)

| From | Event | Effect on preview |
|------|-------|-------------------|
| `planList` | open Edit/Schedule/Add-event form | Preview begins rendering (once Start valid). |
| timed form | edit Start/Duration | Preview re-derived next frame; moves/resizes; conflict set recomputed. |
| timed form | `Esc` / Cancel | Mode → `planList`; preview vanishes; saved entries unchanged (FR-010). |
| timed form | Save (valid) | Mode → `planList`; reload renders the saved entry; preview gone (FR-011). |

No field is added to `planState`; the preview has no stored lifecycle to manage.
