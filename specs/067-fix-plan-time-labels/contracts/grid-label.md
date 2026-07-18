# Contract: Plan Grid Entry Labels and Conflict Slots

**Feature**: 067-fix-plan-time-labels
**Status**: Normative for `internal/cli.RenderGrid` and `internal/tui.planPreviewConflicts`
**Amends**: `specs/039-plan-entry-preview/contracts/grid-preview.md` §4 (conflict-slot semantics). Everything that document specifies about preview rendering — dashed runes, style hooks, plain-mode fallback marker — remains in force unchanged.

This is a UI rendering contract, not a wire contract. No protobuf, HTTP, or ConnectRPC surface is affected by this feature.

## 1. Entry label format

Each timed entry drawn in the grid carries a single-line label, wrapped and truncated to the box's content width by existing rules.

```text
HideID = true:   "HH:MM-HH:MM <name>"
HideID = false:  "[<id>] HH:MM-HH:MM <name>"
```

**Normative requirements**:

1. `HH:MM-HH:MM` MUST be derived from the entry's **exact** interval: start is `StartMinute`, end is `StartMinute + DurationMinute`.
2. The label MUST NOT use snapped values. Snapping remains confined to geometry.
3. Times are 24-hour, zero-padded, in the plan's local day. Format is unchanged from the previous implementation; only the values change.
4. When `DurationMinute <= 0`, the end equals the start and is rendered as such (e.g. `13:05-13:05`).
5. The ID prefix, name, and all styling hooks (selection, completion, preview, conflict) are unaffected by this contract.

**Non-normative examples**:

| Start | Duration | Label (HideID) | Box rows |
|---|---|---|---|
| 13:00 | 50 | `13:00-13:50` | 13:00 → 14:00 |
| 13:50 | 20 | `13:50-14:10` | 13:45 → 14:15 |
| 09:00 | 30 | `09:00-09:30` | 09:00 → 09:30 |
| 13:05 | 5 | `13:05-13:10` | 13:00 → 13:15 |

The "Box rows" column is stated only to show that geometry is independent of the label; it is specified by the pre-existing grid contract and is not changed here.

## 2. Hour gutter

The left gutter MUST continue to display 15-minute boundary times. It labels grid rows, not entries, so snapped values are correct there and this contract does not apply.

## 3. Conflict slot semantics

`GridOptions.PreviewConflictSlots` remains `map[int]bool` keyed by slot-start minute (a multiple of 15). The renderer's use of it is unchanged. What changes is which keys the producer puts in it.

Given a preview interval `P = [pStart, pEnd)` and each other timed entry `O = [oStart, oEnd)`, both exact and half-open:

1. Compute `ovStart = max(pStart, oStart)` and `ovEnd = min(pEnd, oEnd)`.
2. If `ovStart >= ovEnd`, entry `O` contributes **no** slots. Touching boundaries land here.
3. Otherwise add every `t` from `snapDown15(ovStart)` while `t < ovEnd`, stepping by 15.

**Normative requirements**:

1. A conflict MUST be reported only when two exact intervals overlap by at least one minute.
2. Entries that merely touch (one's end equals the other's start) MUST NOT conflict.
3. Two entries within the same 15-minute slot that do not overlap MUST NOT conflict.
4. Marked slots MUST be limited to those spanning the overlap region, not the whole preview.
5. Producers MUST exclude the entry being edited from `others`; self-exclusion stays the caller's responsibility (`buildTimedSliceWithPreview` in `internal/tui/plan_view.go`).
6. No artificial minimum duration is applied to either interval.
7. Every emitted key MUST be a multiple of 15 and MUST fall within the preview's snapped box. Requirement 3's construction guarantees this, since the overlap region lies inside the preview interval.

**Non-normative examples** (preview vs. one other entry):

| Preview | Other | Conflict slots | Why |
|---|---|---|---|
| 13:00–13:50 | 13:50–14:10 | none | touching |
| 13:00–14:00 | 13:50–14:10 | `{825}` | overlap 13:50–14:00 → slot 13:45 |
| 13:00–13:05 | 13:10–13:20 | none | same slot, no overlap |
| 10:00–11:00 | 10:30–11:30 | `{630, 645}` | overlap 10:30–11:00 |
| 10:00–11:00 | 09:00–12:00 | `{600, 615, 630, 645}` | preview fully contained |
