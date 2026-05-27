# Feature Specification: Plan Calendar Grid View

**Feature Branch**: `013-plan-calendar-grid`

**Created**: 2026-05-27

**Status**: Draft

**Input**: User description: "Change the way that the plan view looks. Each day should have a calendar with the hours marked in a grid… entries shown as boxes with heavier marks covering the hour grid. Each text line represents 15 minutes."

## Clarifications

### Session 2026-05-27

- Q: How should the visible time window of the calendar be determined? → A: Default 08:00–17:00; expand outward (rounded down/up to the hour) to include any entry that starts earlier or ends later.
- Q: How wide should the calendar render? → A: Fit to terminal width, with a sensible minimum (~60 columns); box and dividers stretch.
- Q: Should the calendar show a current-time indicator? → A: Yes, but only when rendering today; a subtle marker character in the left gutter at the current 15-minute row.
- Q: Should tasks and events render with different visual styles? → A: No — identical heavy box-drawing for both; the label conveys identity.
- Q: Should completed-task entries be visually distinguished? → A: Yes — apply strikethrough (and dim color on TTY) to the label text; box geometry unchanged.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View a day as an hour-by-hour calendar grid (Priority: P1)

When a user displays their daily plan, the day is rendered as a vertical calendar with each hour marked by a horizontal divider, and the space between hours divided into four 15-minute rows. This gives the user an at-a-glance, time-proportional view of their day rather than a flat list.

**Why this priority**: This is the core visual change requested. Without it, the rest of the feature has nothing to render onto.

**Independent Test**: Render a plan for a day that contains no entries. Verify that the output shows hour labels (e.g. `08:00`, `09:00`, …) on the left, horizontal hour dividers across the calendar width, and that the vertical space between consecutive hour labels accommodates exactly three blank text rows (representing the :15, :30, and :45 quarter-hours).

**Acceptance Scenarios**:

1. **Given** a day with no scheduled entries, **When** the plan view is rendered, **Then** the user sees an empty calendar grid showing each hour from the configured start hour through the configured end hour, with the hour label on the left and a horizontal divider on each hour row.
2. **Given** a day's plan, **When** rendered, **Then** every gap between consecutive hour rows contains three text rows representing :15, :30, and :45 respectively.

---

### User Story 2 - See entries as time-proportional boxes (Priority: P1)

Each scheduled entry on the day is drawn as a rectangle whose top edge is at the entry's start time and whose bottom edge is at the entry's end time. The box uses heavier (bold) box-drawing characters so it stands out against the lighter hour grid. The label inside the box shows the entry's ordinal, time range, and name; the name is truncated with an ellipsis when it does not fit on one line, but may wrap onto additional lines while space remains inside the box.

**Why this priority**: This is the primary information the calendar exists to communicate. Without it the grid is empty scaffolding.

**Independent Test**: Render a plan containing one 2-hour entry, one 30-minute entry, one entry that starts off the hour, one 15-minute entry on the hour, and one 15-minute entry off the hour. Verify each box is positioned correctly relative to the hour grid, sized correctly in 15-minute increments, and labelled with its ordinal, time range, and (possibly truncated) name.

**Acceptance Scenarios**:

1. **Given** an entry from 08:00 to 10:00, **When** rendered, **Then** the entry's top edge sits exactly on the 08:00 hour row, its bottom edge sits exactly on the 10:00 hour row, and the box draws across the intervening rows using heavy box-drawing characters.
2. **Given** an entry from 11:15 to 12:00, **When** rendered, **Then** the entry's top edge sits on the :15 row below 11:00 (not on the 11:00 hour row) and its bottom edge sits on the 12:00 hour row.
3. **Given** an entry from 13:00 to 13:15, **When** rendered, **Then** the entry is drawn as a single-row box with both heavy top and heavy bottom on the same line; the label appears inline on that single row.
4. **Given** an entry whose label exceeds the available width, **When** rendered, **Then** the label is truncated with an ellipsis (`...`) at the end of the last available text row inside the box.
5. **Given** an entry whose label can wrap onto additional rows inside the box, **When** rendered, **Then** the label wraps to fill the box's interior rows before any truncation occurs.

---

### User Story 3 - Adjacent entries share a single border line (Priority: P2)

When two entries are scheduled back-to-back (the first one's end time equals the second one's start time), their boxes share one horizontal border line rather than each drawing its own. The shared line uses the heavy box-drawing junction characters appropriate to a "bottom of upper box / top of lower box" joint.

**Why this priority**: Without this rule, touching entries would draw two adjacent border lines and either consume an extra row of vertical space or visually misrepresent the timing. Important for readability but the calendar still works without it.

**Independent Test**: Render two entries scheduled 13:00–13:15 and 13:15–13:30. Verify that exactly one heavy horizontal line appears between them at the :15 mark, and that the line uses a junction character indicating both boxes meet there.

**Acceptance Scenarios**:

1. **Given** an entry ending at 10:00 and another starting at 10:00, **When** rendered, **Then** the 10:00 hour row shows a single shared heavy horizontal line with the appropriate left/right junction characters connecting to the hour grid, rather than two stacked horizontal borders.
2. **Given** an entry ending at 13:15 and another starting at 13:15, **When** rendered, **Then** the :15 row shows a single shared heavy horizontal line between them.

---

### User Story 4 - Glanceable "now" indicator on today (Priority: P2)

When the rendered day is today, a small marker appears in the left gutter on the 15-minute row that contains the current time, so the user can immediately see what is happening now and what comes next without consulting a clock.

**Why this priority**: Real practical value for a tool used during the working day, but the calendar is usable without it.

**Independent Test**: Render today's plan at a known wall-clock time. Verify a marker appears in the left gutter on the row corresponding to the current 15-minute slot. Render a non-today date and verify no such marker appears.

**Acceptance Scenarios**:

1. **Given** the rendered day is today and the current local time is 10:37, **When** the plan is rendered, **Then** a marker appears in the left gutter on the row representing 10:30–10:45.
2. **Given** the rendered day is yesterday or tomorrow, **When** the plan is rendered, **Then** no current-time marker appears anywhere on the calendar.

---

### Edge Cases

- An entry's start or end time falls between 15-minute marks (for example 09:07): the system has no way to draw sub-15-minute precision, so the entry is snapped to the nearest 15-minute boundary for rendering purposes. The underlying entry data is not modified.
- An entry extends past the calendar's end hour or starts before its start hour: cannot occur under the visible-window rule (the window is always extended to contain every entry). No clipping behavior is required.
- Two entries overlap (one's end is strictly after another's start): out of scope for this feature. The current plan model assumes a single-track schedule; overlap handling is not introduced here.
- An entry has an empty or whitespace-only name: only the ordinal and time range are shown in the label.
- The terminal is narrower than the minimum width needed to draw the grid plus a useful label: the grid still renders, label truncation simply happens sooner.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-000**: When the rendered day is today (in the user's local time zone), the calendar MUST show a current-time indicator: a marker character in the left gutter, on the 15-minute row containing the current time. When the rendered day is not today, no current-time indicator MUST appear.

- **FR-001**: The plan view MUST render each day as a vertical calendar grid with one labelled hour row per hour and three intervening text rows per hour (one row per 15-minute increment).
- **FR-002**: Hour rows MUST display the hour label (e.g. `08:00`) on the left, followed by a horizontal divider that spans the calendar width using light box-drawing characters. The calendar width MUST adapt to the terminal width, subject to a minimum width (approximately 60 columns) below which rendering may degrade gracefully but must still produce a usable grid.
- **FR-003**: The calendar MUST render with a default visible window of 08:00 through 17:00. If any entry on the day starts before 08:00, the window's start hour MUST be extended downward to the hour containing that entry's start (rounded down to the nearest hour). If any entry ends after 17:00, the window's end hour MUST be extended upward to the hour containing that entry's end (rounded up to the nearest hour). Empty days MUST still render the full default 08:00–17:00 window.
- **FR-004**: Each plan entry MUST be rendered as a rectangular box drawn with heavy box-drawing characters, positioned so that the box's top edge aligns with the row corresponding to the entry's start time and its bottom edge aligns with the row corresponding to its end time.
- **FR-005**: Entry start and end times that do not fall on a 15-minute boundary MUST be snapped to the nearest 15-minute boundary for rendering purposes only; the stored entry data MUST NOT be modified.
- **FR-006**: Each entry box MUST display, on its first interior text row, the entry's ordinal (e.g. `[3]`), its start–end time range (e.g. `11:15-12:00`), and its name, in that order.
- **FR-007**: If the label is longer than the available interior width on a single row, the label MUST wrap onto subsequent interior rows of the same box where space allows.
- **FR-008**: If the label still does not fit after wrapping fills the box's interior rows, the label MUST be truncated and end with an ellipsis (`...`).
- **FR-009**: When two entries are temporally adjacent (one entry's end time equals another entry's start time after snapping), the two boxes MUST share a single heavy horizontal border line at the join, rather than drawing two stacked borders.
- **FR-010**: A 15-minute entry (occupying exactly one text row) MUST be drawn as a single row whose left and right ends use junction characters and whose horizontal extent uses heavy horizontal characters, with the label inline on that same row.
- **FR-011**: Where an entry box crosses an hour row, the box's vertical sides MUST replace the hour row's divider at the entry's column extent, while the divider continues outside the entry on both sides.
- **FR-013**: When a plan entry references a task whose status is completed, the entry's label text inside the box MUST be rendered with a strikethrough effect; on a TTY supporting color/style codes, the label MAY additionally be rendered dimmed. The box border characters MUST remain unchanged regardless of completion status. Entries that are not task references (e.g. events) are never marked as completed.

### Key Entities

- **Plan Day**: a day to be rendered. Has a start-of-day hour, an end-of-day hour, and an ordered collection of entries. The hours are existing configuration; no new attributes are introduced by this feature.
- **Plan Entry**: a scheduled item on a day. Has a start time, an end time, an ordinal (its position within the day), a name, and (for task-backed entries) a derived completion status from the referenced task. No new persisted attributes are introduced by this feature; completion status is read from the existing task model.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user viewing a day with at least one scheduled entry can identify, without reading any text, which times of day are occupied versus free, by visually scanning the vertical position and length of the heavy boxes against the hour grid.
- **SC-002**: For every entry shown, the entry's top-edge row corresponds to its rendered start time and its bottom-edge row corresponds to its rendered end time, verifiable by counting rows from the nearest hour label.
- **SC-003**: When two entries are temporally adjacent, the number of text rows consumed between the start of the earlier entry and the end of the later entry equals exactly the sum of their individual durations in 15-minute units — i.e., the shared border does not consume an extra row.
- **SC-004**: 100% of rendered entry labels either fit fully inside the box or end in an explicit ellipsis (`...`); no label is silently cut off mid-word with no indicator.

## Assumptions

- The plan view already has a notion of which range of hours to show for a day; this feature renders within that existing range and does not introduce new configuration for visible hours.
- The terminal can render Unicode box-drawing characters, including the heavy variants (`┏ ┓ ┗ ┛ ┃ ━ ┣ ┫`) and the light variants (`├ ┤ │ ─`). Plain-ASCII fallback is out of scope for this feature.
- The plan model is single-track: at most one entry occupies any given moment. Overlapping entries are out of scope.
- Sub-15-minute precision is intentionally not supported in this view; any entry whose times are finer-grained snaps to the nearest 15-minute boundary for display purposes only.
- The hour grid stretches to the terminal width with a sensible minimum (~60 columns); label truncation adapts to that width.
- This change replaces the current plan view rendering; no toggle is provided to switch between the old and new presentations.
- Task entries and event entries render identically; the visible distinction is only their label text.
