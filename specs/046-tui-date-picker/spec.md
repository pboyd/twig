# Feature Specification: TUI Calendar Date Picker

**Feature Branch**: `046-tui-date-picker`

**Created**: 2026-06-10

**Status**: Draft

**Input**: User description: "The date fields in the TUI are simple text inputs right now. It's functional, but it would be nice to have a calendar widget for the date picker."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Pick a date from a calendar (Priority: P1)

While editing a task in the interactive TUI, a user reaches a date field (Due or Snooze until) and, instead of typing a date string from memory, opens a month calendar, navigates to the desired day, and selects it. The chosen date is filled into the field in the format the form already accepts.

**Why this priority**: This is the core value of the feature — removing the burden of recalling and typing exact date formats. Without it, nothing else in this spec matters.

**Independent Test**: Can be fully tested by opening the task edit form, focusing a date field, opening the calendar, selecting a day, and confirming the field now contains that date and the task saves with it.

**Acceptance Scenarios**:

1. **Given** the task edit form is open and a date field is focused, **When** the user opens the calendar widget, **Then** a month view appears showing the current month (or the month of the field's existing date, if set) with today / the existing date visually indicated.
2. **Given** the calendar is open, **When** the user moves the selection to another day and confirms, **Then** the calendar closes and the date field contains the selected date in an accepted format.
3. **Given** the calendar is open, **When** the user dismisses it without confirming, **Then** the calendar closes and the date field's prior value is unchanged.
4. **Given** a date field already contains a valid date, **When** the user opens the calendar, **Then** the calendar opens with that date pre-selected.

---

### User Story 2 - Keyboard navigation across days, months, and years (Priority: P2)

A user wants to schedule something several weeks or months out. From the calendar they move day by day, week by week, jump between months, and jump between years — entirely from the keyboard, since the TUI is keyboard-driven.

**Why this priority**: A calendar that only shows the current month barely beats typing. Efficient keyboard movement is what makes the widget genuinely faster than text entry.

**Independent Test**: Can be tested by opening the calendar and verifying each navigation key moves the selection as expected, including across month and year boundaries.

**Acceptance Scenarios**:

1. **Given** the calendar is open, **When** the user presses directional keys, **Then** the selection moves by one day (left/right) or one week (up/down).
2. **Given** the selection is on the last day of a month, **When** the user moves forward one day, **Then** the calendar advances to the first day of the next month.
3. **Given** the calendar is open, **When** the user uses the month-jump keys, **Then** the view moves to the same day in the previous/next month (clamping to the last day when the target month is shorter).
4. **Given** the calendar is open, **When** the user uses the year-jump keys, **Then** the view moves to the same date in the previous/next year.

---

### User Story 3 - Text entry still works (Priority: P3)

A user who knows exactly the date they want — or wants a precise time of day on the Due field — types it directly into the field as they do today, never opening the calendar.

**Why this priority**: The calendar is an enhancement, not a replacement. Power users and scripts of habit must not lose the faster path, and the Due field's time-of-day precision is only reachable by typing.

**Independent Test**: Can be tested by editing a task and typing dates into both date fields without ever invoking the calendar, then saving successfully.

**Acceptance Scenarios**:

1. **Given** a date field is focused, **When** the user types a date in an accepted format and saves, **Then** the task saves exactly as before this feature existed.
2. **Given** the Due field contains a timestamp with a time of day, **When** the user picks a different date from the calendar, **Then** the date portion changes and the existing time of day is preserved.
3. **Given** a date field is focused and the calendar is closed, **When** the user presses ordinary text keys, **Then** they edit the field as plain text (the calendar does not open uninvited).

---

### User Story 4 - Clear an optional date (Priority: P3)

Both date fields are optional. A user who set a due date by mistake removes it, either by clearing the text field as today or via an explicit "clear" action in the calendar.

**Why this priority**: Optional fields must stay optional; a picker that can only set dates but never unset them would be a regression in usability.

**Independent Test**: Can be tested by setting a date, clearing it, saving, and confirming the task has no date.

**Acceptance Scenarios**:

1. **Given** a date field contains a date, **When** the user clears the field (by text deletion or a clear action), **Then** the field is empty and the task saves with no date set.

---

### Edge Cases

- February 29: navigating year-forward from a leap day must land on a valid date (e.g., clamp to Feb 28).
- Month jumps from day 31 into a 30-day (or 28/29-day) month must clamp to the month's last day rather than producing an invalid date.
- Opening the calendar when the field contains malformed or partially typed text: the calendar opens at a sensible default (today) instead of failing.
- The terminal window is very small: the calendar must either fit or degrade gracefully without corrupting the rest of the form.
- The calendar's open/navigate/confirm keys must not conflict with existing form keys (field cycling, save, cancel).
- Dates in the past are allowed (both fields legitimately accept past dates today; the calendar must not forbid them).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Users MUST be able to open a calendar widget from any date field in the task edit form (Due and Snooze until) via a discoverable keystroke.
- **FR-002**: The calendar MUST display one month at a time, indicating today's date and the currently selected date.
- **FR-003**: When the field already holds a valid date, the calendar MUST open with that date selected; otherwise it MUST open on today's date.
- **FR-004**: Users MUST be able to move the selection by day, by week, by month, and by year using only the keyboard.
- **FR-005**: Confirming a selection MUST close the calendar and write the selected date into the field in a format the form already accepts for that field.
- **FR-006**: Dismissing the calendar without confirming MUST leave the field's value unchanged.
- **FR-007**: Direct text entry into date fields MUST continue to work exactly as it does today; the calendar is an optional aid, not a gate.
- **FR-008**: When the Due field holds a timestamp with a time of day, selecting a date from the calendar MUST preserve the existing time of day.
- **FR-009**: Users MUST be able to end up with an empty (unset) date after interacting with the calendar or field, since both dates are optional.
- **FR-010**: Month and year navigation MUST clamp to valid dates (shorter months, leap years) and never produce an invalid selection.
- **FR-011**: The calendar's key bindings MUST be visible or discoverable (e.g., in the form's help/footer area) and MUST not conflict with existing form key bindings.

### Key Entities

- **Date field**: A form input holding an optional calendar date (Snooze until) or an optional date-with-optional-time (Due).
- **Calendar widget**: A transient month-view selector attached to a date field; it has a viewed month, a selected day, and produces either a confirmed date or a cancellation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can set a due date in the current month via the calendar in under 10 seconds without typing any date text.
- **SC-002**: A user can reach any date within ±2 years in 15 or fewer keystrokes from opening the calendar.
- **SC-003**: 100% of dates selectable in the calendar are accepted by the form's existing validation (no selection ever produces a format error).
- **SC-004**: Users who prefer typing experience zero change: every previously valid typed input remains valid and behaves identically.
- **SC-005**: Date-format errors on save (mistyped dates) drop to zero for users who use the calendar.

## Assumptions

- The calendar is keyboard-driven; mouse support is not required (consistent with the rest of the TUI).
- The calendar supplements text entry rather than replacing it — both input methods coexist on the same fields (hybrid model).
- The calendar selects calendar dates only; time-of-day on the Due field remains a typed concern, with any existing time preserved across calendar selections (midnight/UTC default for newly picked dates, matching current behavior for date-only input).
- Scope is the task edit form in the TUI (Due and Snooze until fields). Other surfaces (CLI flags, web frontend) are out of scope.
- Week layout and month names follow the existing locale conventions already used elsewhere in the TUI (English, weeks starting Sunday or Monday per the chosen widget's default).
- No new persistence, API, or server changes are needed — the feature only changes how a date string gets into the existing fields.
