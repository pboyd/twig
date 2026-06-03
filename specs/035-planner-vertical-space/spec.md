# Feature Specification: Planner Vertical Space

**Feature Branch**: `035-planner-vertical-space`

**Created**: 2026-06-03

**Status**: Draft

**Input**: User description: "improve the day planner's use of vertical space. The planning tab currently stops drawing the day planner grid at 5 PM (if there aren't more entries). The behavior is great for the CLI, but in the TUI it should instead fill the available space with more hours if there is space available. When there isn't enough space available, the planner should prioritize upcoming events, and (after the untimed entries) begin rendering the planner at the block at the current time."

## Clarifications

### Session 2026-06-03

- Q: When extra vertical space is available, which direction should the planner fill with more hours? → A: Fill later hours only (extend past 5 PM toward end of day); keep the 8 AM start unless an entry requires earlier.
- Q: When the grid anchors to the current-time block and earlier entries scroll above the top edge, how should the user interact with those hidden earlier entries? → A: Earlier entries are not shown; selection/navigation operates only on visible entries (no new scrolling model).
- Q: What baseline must fail to fit before the planner anchors to the current-time block instead of showing from the top? → A: Anchor to now only when untimed entries + the entry-extended default window (the view the CLI would render) exceed available rows.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Fill empty space with more hours (Priority: P1)

A person opens the Planning tab in the TUI on a tall terminal with a lightly
scheduled day. Today's last scheduled item ends at 2 PM, but the terminal has
room to show many more hours. Instead of the grid stopping at 5 PM and leaving a
large blank gap below it, the grid keeps drawing later hours until the available
vertical space is filled.

**Why this priority**: This is the headline improvement — the day planner
currently wastes the lower half of a tall window. Filling it with usable, later
hours immediately makes the planner more useful for scheduling the rest of the
day, and it is independently demonstrable on any tall terminal.

**Independent Test**: Open the Planning tab on a terminal tall enough to display
well past 5 PM with a day whose entries all end before 5 PM. Confirm the grid
draws hour rows beyond 5 PM, filling the available height rather than stopping at
5 PM with blank space below.

**Acceptance Scenarios**:

1. **Given** a Planning tab with vertical room for more than the default
   8 AM–5 PM window and a day with no entries after 5 PM, **When** the view
   renders, **Then** the grid continues drawing hours after 5 PM until the
   available vertical space is consumed.
2. **Given** the terminal is then made shorter (fewer rows), **When** the view
   re-renders, **Then** the number of hours shown decreases to match the new
   available space.
3. **Given** the terminal is exactly tall enough for the default 8 AM–5 PM
   window and nothing more, **When** the view renders, **Then** the grid shows
   the default window with no extra hours and no blank gap.

---

### User Story 2 - Prioritize upcoming events when space is tight (Priority: P1)

A person opens the Planning tab on a short terminal (few rows) while viewing
today, partway through the afternoon. There is not enough vertical space to show
the whole default window plus their untimed items. Rather than showing the
morning hours that have already passed, the planner shows the untimed entries
first and then begins the timed grid at the block containing the current time,
so the next upcoming events are what they see.

**Why this priority**: A short window that wastes its limited rows on hours that
have already elapsed hides exactly the information the user needs most — what is
coming next. This is the core "not enough space" behavior and is the other half
of making the planner respect vertical space.

**Independent Test**: On a short terminal, view today partway through the day
with entries both before and after the current time. Confirm untimed entries
appear first, then the timed grid starts at the current-time block (earlier
hours are not shown), and upcoming entries are visible.

**Acceptance Scenarios**:

1. **Given** today is in view, the current time is mid-afternoon, and the
   Planning tab is too short to show the full default window plus untimed
   entries, **When** the view renders, **Then** the timed grid's first visible
   row is the 15-minute block containing the current time, and hours before that
   block are not drawn.
2. **Given** the same constrained view, **When** the view renders, **Then** the
   untimed entries are shown above the timed grid and are not dropped in favor of
   timed hours.
3. **Given** a constrained view with an entry scheduled later today, **When** the
   view renders, **Then** that upcoming entry is reachable within the visible
   grid rather than scrolled off the bottom in favor of past hours.

---

### User Story 3 - Untimed entries remain visible (Priority: P2)

A person with several untimed (no start time) plan entries opens the Planning tab
on a short terminal. The untimed entries — which represent things to fit in
whenever — stay visible above the timed grid, and the timed grid takes whatever
space remains.

**Why this priority**: Untimed entries are already a first-class part of the
planner. Preserving their visibility under the new space-management rules keeps
the feature consistent and avoids regressing existing behavior, but it builds on
the prioritization established in User Story 2.

**Independent Test**: On a short terminal with multiple untimed entries, confirm
the untimed pane and its separator render in full before any timed grid rows, and
the timed grid fills the remaining rows.

**Acceptance Scenarios**:

1. **Given** untimed entries that fit within the available height, **When** a
   constrained view renders, **Then** all untimed entries and the separator are
   shown before the timed grid.
2. **Given** untimed entries plus a timed grid that together exceed the available
   height, **When** the view renders, **Then** the untimed entries are shown
   first and the timed grid is the part that is reduced to fit.

---

### Edge Cases

- **Viewing a day other than today**: There is no "current time" block within a
  past or future day. In a constrained view of a non-today day, the timed grid
  begins at the default window start (top of the day window) rather than a
  current-time block.
- **Current time outside the day window**: If the current time falls before the
  window start or after the window end (e.g., very early morning or late night),
  the constrained view begins the grid at the nearest applicable window boundary
  rather than an out-of-range block.
- **Entry in progress across the current-time block**: When the grid starts at
  the current-time block, an entry that began earlier and is still running may
  have its earlier rows above the visible top; only the portion from the
  current-time block down is shown.
- **Selection above the anchor**: If selection was on an entry that becomes
  hidden above the current-time anchor, selection MUST move to an entry within the
  visible window (or to no selection) rather than pointing at an undrawn entry.
- **Day extends past the available expansion limit**: Expansion to fill space
  does not draw hours beyond the end of the day; once the day's final hour is
  reached, no further rows are added even if vertical space remains.
- **Untimed entries alone exceed available height**: When untimed entries cannot
  all fit, the timed grid may be reduced to zero visible rows; behavior degrades
  gracefully without error.
- **Entries already extending past 5 PM**: The existing rule that the window
  expands to contain every entry's span still applies; space-filling only adds
  hours beyond what entries require, never removes hours an entry needs.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The TUI Planning tab MUST extend the visible timed grid beyond the
  default window end (5 PM) to fill unused vertical space, drawing additional
  later hours until the available height is consumed or the end of the day is
  reached.
- **FR-002**: The TUI Planning tab MUST NOT draw timed-grid hours beyond the end
  of the day when filling space.
- **FR-003**: When available vertical space is insufficient to show the untimed
  entries plus the entry-extended default timed window (the view the CLI would
  render), and today is in view, the TUI MUST begin the timed grid at the
  15-minute block containing the current time, omitting earlier blocks.
- **FR-004**: Under the constrained (current-time-start) behavior, the TUI MUST
  render untimed entries (and their separator) before the timed grid, prioritizing
  their visibility over earlier timed hours.
- **FR-005**: The constrained current-time-start behavior MUST apply only when
  the day in view is today; for other days the timed grid MUST begin at the
  default window start.
- **FR-006**: The TUI MUST recompute the visible window each time the view renders
  so that the displayed hours adjust to the current terminal height.
- **FR-007**: The existing rule that the window expands outward to contain every
  entry's snapped span MUST be preserved; space management MUST NOT hide hours
  that an entry occupies.
- **FR-008**: The CLI (non-TUI) day planner output MUST remain unchanged,
  retaining the default 8 AM–5 PM window extended only to contain entries.
- **FR-009**: When vertical space exactly matches or is less than the default
  window, the TUI MUST NOT introduce a permanent blank gap below the grid; any
  unused trailing space is the result of the day ending, not of stopping at 5 PM.
- **FR-010**: All space-management behavior MUST degrade gracefully on very short
  or very narrow terminals without errors or visual corruption.
- **FR-011**: Under the constrained current-time-start behavior, timed entries
  earlier than the visible window MUST NOT be drawn, and entry selection /
  navigation MUST operate only on visible entries; the feature MUST NOT introduce
  a scrolling model for reaching off-screen earlier entries.

### Key Entities *(include if feature involves data)*

- **Visible window**: The contiguous range of hours the timed grid draws, defined
  by a start block and an end block. Currently fixed by a default (8 AM–5 PM)
  expanded to contain entries; this feature makes the TUI's window
  height-aware.
- **Timed plan entry**: A plan entry with a start time and duration, occupying one
  or more 15-minute blocks in the grid.
- **Untimed plan entry**: A plan entry without a start time, shown in a stacked
  pane above the timed grid.
- **Current-time block**: The 15-minute block containing the present moment, used
  as the grid start when space is constrained and today is in view.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: When the Planning tab has more vertical space than the default
  window requires and the day has no late entries, at least 95% of the available
  grid rows are filled with hour content (no large permanent blank region below
  the grid).
- **SC-002**: When the Planning tab is too short to show the full day on the
  current day, the block containing the current time is always among the visible
  timed-grid rows.
- **SC-003**: Untimed entries that fit within the available height are fully
  visible in 100% of constrained renders, appearing before any timed grid rows.
- **SC-004**: CLI day-planner output is byte-for-byte identical to the previous
  behavior across all existing test cases (zero regressions).
- **SC-005**: Resizing the terminal taller or shorter changes the number of
  visible hours accordingly within a single render cycle, with no manual refresh
  required.

## Assumptions

- **Expansion direction**: Unused vertical space is filled by extending the
  window end forward (later hours) toward the end of the day, not by adding
  earlier morning hours before 8 AM. The default 8 AM start is preserved unless an
  entry requires an earlier start.
- **End-of-day bound**: The latest hour the grid will draw when filling space is
  the end of the day (midnight / 24:00).
- **Constrained trigger**: "Not enough space" means the combined untimed pane,
  separator, and entry-extended default timed window (the view the CLI would
  render) exceed the rows available to the grid in the Planning tab.
- **Non-today days**: Because there is no meaningful "current time" within a past
  or future day, the current-time-start behavior is limited to today; other days
  use the default window start when constrained.
- **Per-render recomputation**: The visible window is derived fresh on each render
  from current terminal height and the day's entries; it is not persisted across
  day navigation.
- **Scope**: Changes are confined to the TUI Planning tab's grid rendering /
  windowing. The CLI planner, plan data model, and server APIs are unchanged.
