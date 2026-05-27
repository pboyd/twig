# Feature Specification: Calendar Grid Padding & Gutter Refinement

**Feature Branch**: `014-calendar-grid-padding`

**Created**: 2026-05-27

**Status**: Draft

**Input**: User description: "A minor update to the 013-plan-calendar-grid feature. The hour grid lines should always be present, and the heavy boxes should look like they're inside them (one-character of padding). Let's also add an extra space in the gutter so that the time indicator is separated from the hour markers."

## Overview

A small visual refinement to the calendar grid introduced in feature 013. The hour grid (the light horizontal lines and the light vertical side rails) is always drawn end-to-end, and each entry box is inset by one character on the left and one character on the right so the heavy box visually sits *inside* the lighter grid rather than replacing it. The "now" indicator in the left gutter is separated from the hour label by one extra space so the marker no longer touches the hour text.

Example of the intended rendering:

```
08:00 ▶├─┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓─┤
       │ ┃ [1] 08:00-10:00 a task spanning multiple hours                     ┃ │
       │ ┃                                                                    ┃ │
       │ ┃                                                                    ┃ │
       │ ┃                                                                    ┃ │
09:00  ├─┃                                                                    ┃─┤
       │ ┃                                                                    ┃ │
       │ ┃                                                                    ┃ │
       │ ┃                                                                    ┃ │
10:00  ├─┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫─┤
       │ ┃ [2] 10:00-10:30 a shorter task, truncate the name if it's too l... ┃ │
       │ ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛ │
       │                                                                        │
11:00  ├────────────────────────────────────────────────────────────────────────┤
```

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Hour grid always visible behind entries (Priority: P1)

When a user views their daily plan as a calendar, the light hour grid (horizontal hour lines and the left/right vertical rails of the calendar area) is always present across the full visible time range, including the hours and rows that are covered by entry boxes. Entry boxes are drawn one character inside the rails, so the light grid frames each heavy box on both sides.

**Why this priority**: This is the core visual change requested. It makes the hour structure of the day legible even when entries fill the space, and it gives the calendar a consistent, contained look.

**Independent Test**: Render a plan containing at least one multi-hour entry that crosses one or more hour boundaries. Verify (a) every hour row, including those inside the entry, shows the light hour divider extending to the outer rails on both sides; (b) the heavy entry box is positioned one column to the right of the left rail and one column to the left of the right rail; (c) on hour rows that intersect an entry, the heavy box's left/right edges meet the light grid via the appropriate junction characters.

**Acceptance Scenarios**:

1. **Given** a day with a 2-hour entry from 08:00 to 10:00, **When** the plan is rendered, **Then** the 09:00 hour line is drawn across the calendar with light horizontal characters on both sides of the heavy box, and the heavy box's left and right edges sit one column inside the outer vertical rails on every row of the entry.
2. **Given** any rendered day in the visible time range, **When** the plan is rendered, **Then** each hour row shows a continuous light horizontal divider from the left rail to the right rail, broken only where it intersects a heavy box (where it joins via a junction character rather than ending).
3. **Given** a row inside an entry that does not coincide with an hour boundary, **When** rendered, **Then** the left and right rails (light vertical lines) are still drawn at the outer edges of the calendar, with one space of padding between each rail and the corresponding heavy vertical side of the box.

---

### User Story 2 - Hour grid spans empty rows too (Priority: P1)

For hours and quarter-hour rows that contain no entries, the calendar still draws the light hour line at each hour boundary and the light left/right rails on every row, so the grid forms a continuous frame for the entire visible day.

**Why this priority**: Without this, the grid would only appear where entries are absent and disappear where they are present, defeating the purpose of an always-on grid.

**Independent Test**: Render a day containing one entry from 09:00 to 10:00 within an 08:00–11:00 visible window. Verify the 08:00, 09:00, 10:00, and 11:00 hour lines are all drawn from left rail to right rail, and that the :15/:30/:45 rows in the empty 08:00 and 10:00–11:00 hours show the light rails on both sides with empty interior.

**Acceptance Scenarios**:

1. **Given** an hour with no entry in it, **When** rendered, **Then** the hour boundary row shows a continuous light horizontal line and the three intervening quarter-hour rows each show only the two light vertical rails with empty space between them.
2. **Given** the first and last hour rows of the visible window, **When** rendered, **Then** both rows show the full light horizontal hour line edge-to-edge.

---

### User Story 3 - "Now" indicator separated from hour label (Priority: P2)

When the rendered day is today, the "now" marker appears in the left gutter with one extra space of separation between it and the hour-label column, so the marker is visually distinct from the time text and aligns under a consistent gutter slot.

**Why this priority**: This is a small legibility fix; the indicator works without it but reads more clearly with the separation.

**Independent Test**: Render today's plan at a known wall-clock time. Verify the "now" marker appears with at least one blank column between it and the hour-label text on the marker row, and that on non-marker rows the gutter still reserves space so the rails align vertically with the marker row.

**Acceptance Scenarios**:

1. **Given** today's plan is rendered at 08:00 with the visible window starting at 08:00, **When** the row containing 08:00 is drawn, **Then** the hour label `08:00` is followed by exactly one space, then the marker glyph, then one more space before the left rail.
2. **Given** any row in the calendar where the "now" marker is not present, **When** rendered, **Then** the gutter still allocates the same column width as the marker row, so the left rail of the calendar is in the same column on every row.
3. **Given** a non-today date is rendered, **When** the plan is drawn, **Then** the marker column is blank on every row but the gutter still allocates the column so the left rail position is unchanged from today's rendering.

---

### Edge Cases

- An entry whose top edge sits on an hour boundary: the heavy top of the box must use a junction character that connects horizontally to the light hour line on both sides (one character of light line between the rail and the corner of the heavy box).
- An entry whose top edge sits *off* the hour (e.g., :15 start): the heavy top of the box appears on a non-hour row; on that row the light rails are still drawn at the outer edges and there is exactly one space of padding between each rail and the heavy box.
- Two back-to-back entries sharing a horizontal border (carried over from feature 013): the shared heavy line still meets the light grid via the appropriate junction characters, with one column of light line between each rail and the heavy junction.
- The "now" marker falling on a row that also contains an entry box: the gutter rendering (marker + extra space) is unchanged; the entry box still starts one column inside the left rail.
- A single-row entry (heavy top and heavy bottom on the same row): the padding rule still applies — one column between each rail and the box on that row.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The plan calendar view MUST draw the light horizontal hour divider on every hour row across the full width of the calendar, regardless of whether that row is covered by an entry box.
- **FR-002**: The plan calendar view MUST draw the light left and right vertical rails on every row within the visible time window, including rows covered by entry boxes.
- **FR-003**: Every heavy entry box MUST be inset by one column from the left rail and one column from the right rail; the column between the rail and the box on every row of the entry MUST be a single space character (or, on hour rows, a single character of the light horizontal hour line) and never a heavy box-drawing character.
- **FR-004**: Where a heavy entry box edge intersects the light hour grid, the intersection MUST use junction box-drawing characters that join heavy and light lines (rather than truncating either line), so the light grid visually passes "behind" the heavy box on hour rows.
- **FR-005**: The left gutter MUST place exactly one blank column between the "now" indicator glyph and the hour-label column, and the marker column MUST be reserved on every row so the position of the left rail is identical across all rows of the rendering.
- **FR-006**: On days other than today, the marker column MUST be drawn as blank space on every row; the rail and hour-label positions MUST be identical to today's rendering.
- **FR-007**: All other visual rules from feature 013 (15-minute row resolution, ordinal/time/name labelling inside boxes, label truncation/wrapping, strikethrough on completed entries, dynamic visible window, shared borders for back-to-back entries) MUST continue to apply unchanged.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a rendered plan with at least one multi-hour entry, 100% of hour rows in the visible window show a continuous light horizontal divider from the left rail to the right rail (joining via junction characters where intersected by a heavy box).
- **SC-002**: In a rendered plan, 100% of entry boxes have exactly one column of padding between the box and each outer rail on every row of the box, with no exceptions for single-row entries or off-hour starts.
- **SC-003**: When rendering today's plan, the column index of the left rail is identical on every row, and the "now" marker (when present) is separated from the hour-label text by exactly one blank column.
- **SC-004**: A user reviewing a printed/rendered plan can identify each hour boundary inside an entry box without needing to count rows, because the light hour line is visible flanking the box on hour rows.

## Assumptions

- This refinement applies only to the calendar view introduced in feature 013; the list-style plan rendering (if still reachable) is out of scope.
- The total calendar width (rails + padding + box interior) continues to scale to terminal width, with the new padding columns counted as part of the calendar width budget rather than added on top of the previous width.
- The light grid uses the existing single-line box-drawing characters from feature 013 (`│`, `─`, `├`, `┤`); heavy/light junction characters (e.g. `┝`, `┥`, or the appropriate variants where a heavy line meets a light line) are available in the same character set already in use.
- The "now" marker glyph itself (e.g. `▶`) and its color/styling rules are unchanged from feature 013; only its horizontal position relative to the hour label changes.
- No new configuration options are introduced; the padding and gutter spacing are fixed parts of the rendering.

## Dependencies

- Feature 013 (`specs/013-plan-calendar-grid/`) must be implemented; this feature modifies its rendering rules and assumes its underlying calendar data model and entry layout are in place.
