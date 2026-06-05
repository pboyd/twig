# Contract: Grid Preview Rendering (internal Go API)

This feature exposes no external (proto/HTTP/CLI) interface. Its contract is the
**internal Go API** between the TUI layer (`internal/tui`) and the grid renderer
(`internal/cli`). Per Constitution Principle II, this contract is fixed before
implementation; both sides must conform to it.

## 1. `cli.GridOptions` additions

Added to the existing struct in `internal/cli/plan_grid.go`:

```go
type GridOptions struct {
    // ...existing fields (HideID, SelectedID, Styled, SelectionStyle,
    //    WindowStartMin, WindowEndMin)...

    // PreviewID, when non-zero, marks the entry (by Id) that is an unsaved,
    // in-progress preview. Its box is drawn with dashed runes instead of solid.
    PreviewID int32

    // PreviewStyle, when non-nil, styles the preview entry's non-conflicting
    // runes (e.g. a dim foreground). When nil, the dashed runes are emitted
    // unstyled (plain/non-TTY mode still distinguishes the preview structurally).
    PreviewStyle func(string) string

    // ConflictStyle, when non-nil, styles the preview rows that fall in
    // PreviewConflictSlots (e.g. a red foreground). When nil, the renderer
    // applies the plain-mode fallback marker (see §4).
    ConflictStyle func(string) string

    // PreviewConflictSlots holds the slot-start minutes (multiples of 15) of the
    // preview entry that overlap another timed entry. Empty/nil ⇒ no conflict.
    PreviewConflictSlots map[int]bool
}
```

### Invariants `RenderGrid` MUST uphold

1. **Preview identity**: only the entry with `Id == PreviewID` (when `PreviewID != 0`)
   is drawn with dashed runes. All other entries render exactly as today (byte-for-byte
   when `PreviewID == 0`, preserving existing tests).
2. **Dashed rune substitution** for the preview box:
   | Solid (saved) | Dashed (preview) |
   |---|---|
   | `━` horizontal | `┅` |
   | `┃` vertical | `┇` |
   | `┏ ┓ ┗ ┛ ┣ ┫` corners/junctions | unchanged |
3. **Conflict styling scope**: conflict styling/markers apply **only** to the preview
   entry's rows. Saved/existing entries are never restyled by overlap (FR-005, clarified).
4. **Conflict rows**: a preview row at slot-start minute `t` is conflicting iff
   `PreviewConflictSlots[t]` is true. Styled mode wraps that row's content/border with
   `ConflictStyle`; plain mode uses the §4 fallback.
5. **No mutation**: `RenderGrid` does not modify the entries slice or any entry; the
   preview is supplied by the caller as an ordinary slice element.
6. **Selection vs preview**: the preview entry is not the selection cursor target while a
   form is open; if both `SelectedID` and `PreviewID` were ever set to the same id,
   preview styling takes precedence (defensive — not expected in practice).

## 2. Preview construction (TUI side, `internal/tui/plan_preview.go`)

```go
const previewID int32 = -1

// buildPlanPreview returns the transient preview entry for the active form, or
// nil when no timed preview should be shown (not a timed-form mode, or Start
// does not parse to a valid time).
func (m Model) buildPlanPreview() *planv1.PlanEntry

// planPreviewConflicts returns the set of 15-min slot-start minutes where the
// given preview overlaps any timed entry in `others` (which MUST exclude the
// edit target). Returns nil when preview is nil or no overlap exists.
func planPreviewConflicts(preview *planv1.PlanEntry, others []*planv1.PlanEntry) map[int]bool
```

### Construction contract

- Returns `nil` unless `m.plan.mode ∈ {planTaskTime, planEventForm, planEdit}`.
- Returns `nil` if the Start field does not parse via `timeparse.ParseStart`.
- `Id = previewID`; `StartMinute = &start`; `DurationMinute` from
  `timeparse.ParseDurationOrEnd` (blank/invalid ⇒ `0`, rendered as a minimal slot).
- `Name`: Edit/Event → Name field value; TaskTime → linked task name (`findTask`),
  fallback to a neutral non-empty label.
- For `planEdit`, callers MUST exclude `m.plan.form.entryID` from `others` (self-exclusion).

### Overlap contract

- Uses half-open intervals: slot `[t, t+15)` conflicts with entry `[s, s+d)` iff
  `t < s+d && s < t+15`. Touching boundaries do **not** conflict (FR-008).
- `others` MUST contain only timed entries and MUST NOT contain the edit target.

## 3. Wiring (TUI side, `internal/tui/plan_view.go` + `plan_grid.go`)

- `renderPlanGrid` and `renderPlanGridContent` build the rendered timed slice as:
  `timedOthers` (edit target removed) `+ [preview]` when a preview exists; this slice
  is passed to both `cli.GridWindow` (so the window grows to include the preview) and
  `cli.RenderGrid`.
- `planGridOptions` sets `PreviewID = previewID`, `PreviewConflictSlots` from
  `planPreviewConflicts`, and—in styled mode—`PreviewStyle` (dim) and `ConflictStyle`
  (errorColor red), both sourced from `internal/tui/theme.go`.

## 4. Plain-mode conflict fallback

When `ConflictStyle == nil` (non-styled path), `RenderGrid` marks each conflicting
preview row by placing `!` in the gutter marker column (the same column that renders
the `▶` now-marker). If a conflicting row coincides with the now-marker row, the
conflict `!` takes that column.

## 5. Backward compatibility

With `PreviewID == 0` and `PreviewConflictSlots` empty (the `planList` case and all
non-form rendering), `RenderGrid` and `RenderUntimed` MUST produce output identical to
the pre-feature behavior. Existing `internal/cli/plan_grid_test.go` cases MUST pass
unchanged.
