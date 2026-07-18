# Feature Specification: Show Actual Times in Plan Grid Labels

**Feature Branch**: `067-fix-plan-time-labels`

**Created**: 2026-07-18

**Status**: Draft

**Input**: User description: "The TUI's planning tab rounds the times to 15-minute increments. This is the right behavior for the boxes, but it's also applying to the times in the boxes. For example: the times for those entries are 1300-1350 and 1350-1410. The boxes in the grid are correct, but it's displaying the rounded times in the grid boxes. This problem also affects the preview, where an overlap is reported in the grid."

## Overview

The plan grid draws each entry as a box aligned to a 15-minute grid. Snapping the **box geometry** to 15-minute boundaries is correct and intended — it keeps the grid readable and aligned to the hour gutter. However, the same snapped values are currently reused for two things they should not drive:

1. The **time range printed inside the box**, which misreports the entry's real schedule.
2. The **overlap/conflict detection** shown while editing, which reports conflicts between entries that do not actually overlap.

This feature separates "where the box is drawn" (snapped, unchanged) from "what the entry actually is" (exact, corrected).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Grid labels report the real scheduled times (Priority: P1)

A user schedules a plan entry from 13:00 to 13:50 and another from 13:50 to 14:10. When they view the Plan tab, each box shows the times they actually entered, not times rounded outward to the nearest quarter hour.

**Why this priority**: This is the core defect. The grid is the primary read surface for the day's plan, and it currently states times that are simply wrong — a user reading "13:00-14:00" has no way to know the entry really ends at 13:50. Everything else in this feature is secondary to the labels telling the truth.

**Independent Test**: Create two entries at 13:00-13:50 and 13:50-14:10, open the Plan tab, and confirm the labels read `13:00-13:50` and `13:50-14:10` while the boxes remain aligned to the 15-minute grid rows.

**Acceptance Scenarios**:

1. **Given** an entry scheduled 13:00-13:50, **When** the plan grid renders, **Then** its label reads `13:00-13:50` and its box still spans the grid rows from 13:00 through 14:00.
2. **Given** an entry scheduled 13:50-14:10, **When** the plan grid renders, **Then** its label reads `13:50-14:10` and its box still spans the grid rows from 13:45 through 14:15.
3. **Given** an entry whose start and end already fall on 15-minute boundaries (e.g. 09:00-09:30), **When** the plan grid renders, **Then** its label is unchanged from current behavior.
4. **Given** entry IDs are shown (ID prefix not hidden), **When** the plan grid renders, **Then** the ID prefix is preserved and only the time portion of the label reflects exact times.

---

### User Story 2 - Overlap warnings reflect real overlaps (Priority: P1)

A user edits an entry that runs 13:00-13:50, immediately followed by another entry at 13:50-14:10. The live preview does not warn about a conflict, because the two entries merely touch — they do not overlap.

**Why this priority**: A false conflict warning trains the user to ignore conflict warnings entirely, which destroys the value of the feature. Back-to-back scheduling is the single most common plan shape, so this false positive fires constantly.

**Independent Test**: With an entry at 13:50-14:10 present, open the edit form for an entry and enter a 13:00 start with a 50-minute duration; confirm no conflict is indicated anywhere in the grid.

**Acceptance Scenarios**:

1. **Given** an existing entry at 13:50-14:10, **When** the user previews an entry at 13:00-13:50, **Then** no conflict is reported.
2. **Given** an existing entry at 13:50-14:10, **When** the user previews an entry at 13:00-14:00, **Then** a conflict is reported, because the intervals genuinely overlap from 13:50 to 14:00.
3. **Given** an existing entry at 13:10-13:20, **When** the user previews an entry at 13:00-13:05, **Then** no conflict is reported, even though both fall inside the same 13:00 grid slot.
4. **Given** two entries where one ends at the exact minute the other begins, **When** either is previewed, **Then** the touching boundary is not treated as an overlap.

---

### User Story 3 - The preview label also shows exact times (Priority: P2)

While a user is filling in the start and duration fields for a new or edited entry, the transient preview box in the grid shows the exact times being typed, so the preview label matches what the user just entered.

**Why this priority**: It follows directly from User Story 1 and shares its fix, but it affects a transient state rather than the persistent read surface, so it is slightly less critical.

**Independent Test**: Open the entry form, type a start of 13:00 and a duration of 50 minutes, and confirm the preview box label reads `13:00-13:50`.

**Acceptance Scenarios**:

1. **Given** the user has typed a start of 13:00 and duration of 50, **When** the preview renders, **Then** the preview label reads `13:00-13:50`.
2. **Given** the user changes the duration to 20, **When** the preview re-renders, **Then** the label updates to `13:00-13:20` and the box geometry updates to the corresponding snapped rows.

---

### Edge Cases

- **Entry shorter than one grid slot** (e.g. 13:05-13:10): the box must still occupy at least one visible grid row, while the label reports the exact 5-minute range.
- **Zero or missing duration**: the box continues to use the existing minimum-height behavior; the label reports the entry's actual end as derived from its stored duration.
- **Entry with no start time**: unscheduled entries are unaffected by this feature.
- **Entry extending past the grid's last rendered hour**: the label reports exact times; box clipping behavior is unchanged.
- **Narrow terminal**: when the label is truncated to fit, the exact time range is subject to the same existing truncation rules as today — this feature does not change truncation policy.
- **Conflict spanning a partial slot**: when two entries overlap by only a few minutes within one grid slot, that slot is marked as conflicting, since a grid slot is the finest available unit for the visual marker.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The plan grid MUST render each entry's label using the entry's exact start and end times, formatted as `HH:MM-HH:MM`.
- **FR-002**: The plan grid MUST continue to compute box position and height by snapping the entry's start down and its end up to the nearest 15-minute boundary — box geometry behavior is unchanged.
- **FR-003**: The transient preview entry shown while a timed form is open MUST label itself with the exact times derived from the form's current start and duration values.
- **FR-004**: Conflict detection MUST report a conflict only when the previewed entry's exact interval genuinely overlaps another entry's exact interval, using half-open intervals so that touching boundaries (one entry's end equal to another's start) do not conflict.
- **FR-005**: When a genuine overlap exists, the system MUST mark the grid slots that the overlapping region falls within, so the user can see where the conflict occurs.
- **FR-006**: Conflict detection MUST continue to exclude the entry being edited from the set of entries it is compared against, so an entry never conflicts with itself.
- **FR-007**: Entry labels MUST preserve all existing non-time content and decoration, including the optional ID prefix, the entry name, completion styling, selection styling, and preview/conflict styling.
- **FR-008**: The hour gutter on the left edge of the grid MUST continue to display 15-minute boundary times — it labels the grid itself, not any entry.

### Key Entities

- **Plan Entry**: A scheduled item with a name, an exact start minute, and an exact duration in minutes. It has two distinct time representations that this feature keeps separate: its *exact interval* (used for labels and conflict math) and its *snapped interval* (used only for box geometry).
- **Preview Entry**: A transient, unsaved plan entry derived from the currently open form's fields. Behaves like a plan entry for rendering and conflict purposes but is never persisted.
- **Conflict Slot**: A 15-minute grid slot marked to indicate that the previewed entry genuinely overlaps an existing entry somewhere within that slot.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For any entry whose start or end does not fall on a 15-minute boundary, the times shown in its grid label match the times the user entered exactly, with zero minutes of discrepancy.
- **SC-002**: Two entries scheduled back-to-back produce zero conflict warnings.
- **SC-003**: Every conflict warning shown corresponds to a genuine overlap of at least one minute between two entries; the false-positive rate is zero.
- **SC-004**: Every genuine overlap of at least one minute produces a visible conflict indication; the false-negative rate is zero.
- **SC-005**: Box positions and heights in the rendered grid are unchanged from current behavior for every entry — this change is visible only in label text and conflict marking.
- **SC-006**: A user can read the plan grid and reconstruct the exact schedule they entered without opening any entry's detail or edit view.

## Assumptions

- Snapping box geometry to 15-minute increments is correct and desired; only the label text and the conflict math are defective. The user confirmed "the boxes in the grid are correct."
- The label time format remains `HH:MM-HH:MM` in 24-hour form, matching what is displayed today; only the values change, not the format.
- Conflict markers remain slot-granular, because the grid's finest visual unit is one 15-minute row. A sub-slot overlap marks the whole slot containing it. Only the *decision* to mark becomes exact, not the marker's resolution.
- Half-open intervals `[start, start+duration)` are the correct semantics for overlap, so an entry ending at 13:50 and one starting at 13:50 do not conflict. This matches the intent already documented in the existing conflict-detection logic.
- Existing minimum-height behavior for very short entries (rendering at least one slot) is retained.
- This is a display-and-validation fix in the terminal interface only. No stored data, no server behavior, and no scheduling rules change. Entries already hold exact times; they are merely being reported incorrectly.
- The web interface's plan timeline is out of scope for this feature.
