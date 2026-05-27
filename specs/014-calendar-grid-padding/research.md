# Research: Calendar Grid Padding & Gutter Refinement

This document records the design decisions resolved during `/speckit-clarify`. The feature is a small visual refinement, so research is correspondingly narrow.

## Decision 1: Heavy/light intersections

**Decision**: Where a heavy entry box edge meets the light hour grid, the light hour line terminates one column before the heavy box. No heavy/light Unicode junction glyphs (e.g. `┝`, `┥`, `┿`) are used.

**Rationale**: This is exactly what the user's example in `spec.md` shows. It keeps the rendering legible without relying on less-common Unicode glyphs whose font support varies across terminals, and it reinforces the "one column of padding" reading uniformly on every row — hour rows and non-hour rows alike.

**Alternatives considered**:

- Use literal heavy/light junctions (`┝`, `┥`, `┿`) so the light hour line appears to pass through the heavy box edge. Rejected: deviates from the user's example, depends on terminal font coverage of less-common glyphs, and visually competes with the heavy corners/T-junctions of the box.

## Decision 2: Width budget

**Decision**: Total calendar width is unchanged from feature 013 (terminal width, minimum ~60 cols). The new padding (one column inside each rail) and the extra gutter space come out of the existing width budget; each box's interior is therefore ~2 columns narrower than in 013.

**Rationale**: Preserves the existing layout contract from 013 (calendar fills the terminal). The cost is at most a few fewer columns inside boxes, and 013 already has a label-truncation rule that handles narrower interiors gracefully.

**Alternatives considered**:

- Grow the calendar by the new padding + gutter columns (~3 columns wider) and raise the minimum width accordingly. Rejected: breaks the "fits terminal" contract for users on narrow terminals, and visually pushes the calendar around when terminals are at the minimum width.
