# Phase 0 Research: Planner Vertical Space

All Technical Context unknowns are resolved (the feature is a localized rendering
change in an existing, well-understood code path). This document records the key
design decisions and the alternatives weighed.

## Decision 1: Where the window decision lives

**Decision**: Add a pure function `GridWindow(entries, now, day, availableRows)
→ (startMin, endMin)` in `internal/cli` (same package/file as `RenderGrid`). The
TUI calls it; the CLI does not.

**Rationale**: Keeping the decision pure and integer-only makes it trivially
unit-testable (no terminal, no DB) and co-locates it with the existing snap/
window math (`snapDown15`, the 08:00–17:00 default). The CLI path is untouched,
so its output cannot drift.

**Alternatives considered**:
- *Compute inside `RenderGrid` from a height arg* — would force a signature change
  on the CLI call site and entangle the (height-free) CLI behavior with TUI logic.
  Rejected: higher risk to CLI parity, harder to test in isolation.
- *Compute in the TUI package* — would duplicate the entry-extension math that
  already exists in `RenderGrid`. Rejected: duplication and drift risk.

## Decision 2: How the TUI tells RenderGrid which window to draw

**Decision**: Add two optional fields to `GridOptions`:
`WindowStartMin *int` and `WindowEndMin *int`. When non-nil they override the
computed window and disable `RenderGrid`'s auto-expansion **on that side only**.

**Rationale**: `GridOptions` is already the extension point and is passed by both
CLI and TUI. The CLI constructs `GridOptions{}` (both fields nil) → existing
behavior preserved byte-for-byte. Pointers (vs. a sentinel like `-1`) make
"unset" unambiguous and keep the zero value meaning "CLI default".

**Alternatives considered**:
- *Sentinel ints (0 / -1)* — 0 is a valid minute (00:00); ambiguous. Rejected.
- *Separate `RenderGridWindowed` function* — duplicates the body of `RenderGrid`.
  Rejected on Simplicity (Principle I).

**Parity guard**: The existing tests `TestRenderGrid_EmptyDay`,
`TestRenderGrid_WindowExtensionEarly/Late` already pin the default-window output;
they must remain green unchanged.

## Decision 3: The fill / anchor / top-truncate algorithm

**Decision**: `GridWindow` computes the entry-extended default window
`[baseStart, baseEnd]` (the same expansion `RenderGrid` does today), then:

| Condition | Window returned |
|-----------|-----------------|
| `availableRows >= baseRows` (room to spare) | **Fill**: `start = baseStart`; `end = min(start + (availableRows-1)*15, 24:00)` |
| `availableRows < baseRows` AND day is today AND `now` within/after window | **Anchor**: `start = snapDown15(now)`; `end = min(start + (availableRows-1)*15, 24:00)` |
| `availableRows < baseRows` AND (not today OR `now` before window) | **Top-truncate**: `start = baseStart`; `end = start + (availableRows-1)*15` |

where `baseRows = (baseEnd - baseStart)/15 + 1`.

**Rationale**: Directly encodes the three spec scenarios and the clarifications:
fill extends *later only* (Q1=A, 08:00 start preserved); anchor triggers only when
the entry-extended default view doesn't fit (Q3=A); anchoring applies only to
today (FR-005). Top-truncate matches today's behavior for past/future days.

**Alternatives considered**:
- *Always anchor to now when constrained, even on other days* — meaningless when
  the day has no "now". Rejected per FR-005 / clarification.
- *Fill earlier hours too* — rejected per Q1=A.

## Decision 4: Clipping entries that straddle the window top

**Decision**: Rely on `RenderGrid`'s existing per-row lookup maps. Entries whose
rows fall outside `[start, end]` simply never match a drawn row index; an entry
that began before `start` and ends after it renders only its in-window rows
(interior/bottom), with no top border.

**Rationale**: This is exactly the spec's "entry in progress across the
current-time block: only the portion from the current-time block down is shown"
edge case — achieved with **no extra clipping code**. Selection on a now-hidden
earlier entry is handled by the TUI moving selection into the visible set
(FR-011, "Selection above the anchor" edge case).

**Alternatives considered**:
- *Explicitly drop straddling entries entirely* — would hide an in-progress
  event the user cares about. Rejected.

## Decision 5: Measuring available rows in the TUI

**Decision**: At each grid call site, render the untimed pane + separator first,
count their lines, subtract from the pane's inner height, and pass the remainder
as `availableRows` to `GridWindow`. The existing final `lines[:height]` trim
stays as a defensive backstop.

**Rationale**: Untimed entries are prioritized (FR-004); the timed grid gets only
the leftover rows. The three live/test call sites
(`renderPlanGrid` styled, `renderPlanGridContent` unstyled, `renderPlanningView`
test-only) share this pattern, so the change is uniform.

**Open risk**: If untimed entries alone exceed the inner height, `availableRows`
≤ 0 → the TUI renders no timed rows; the backstop trim guarantees no overflow.
Verified as the "untimed entries alone exceed available height" edge case.
