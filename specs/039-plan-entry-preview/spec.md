# Feature Specification: Live Plan Entry Preview with Overlap Indication

**Feature Branch**: `039-plan-entry-preview`

**Created**: 2026-06-05

**Status**: Draft

**Input**: User description: "when editing an entry in the TUI planning tab, the day planner should show a preview of the item as the form is filled out. A dashed border would be nice, but anything that visually indicates that it's only a preview is acceptable. There should also be some visual indication when the user enters values that will cause an overlap (ideally, the overlapping portion of the preview box would be red--but a simpler indicator will suffice if that's not feasible). The intention is to prevent mistakes from overlapping events, or leaving unexpected gaps in the schedule from the user entering a wrong value."

## Clarifications

### Session 2026-06-05

- Q: Which planning forms should show the live preview + overlap indication? → A: All three timed forms (Edit entry, Schedule task, Add event).
- Q: When a planning form is open, how should the day planner be displayed so the preview is visible? → A: Side-by-side two-pane — day planner in one pane, form in the other; preview renders in the planner pane.
- Q: When the form values overlap an existing entry, what gets the visual conflict marker? → A: The preview's overlapping portion only; saved entries are not restyled.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See where an edited entry will land before saving (Priority: P1)

A user is editing an existing plan entry on the Planning tab. As they change the Start time and Duration in the form, the day planner shows a live preview of the entry positioned at the time window the form values describe. The preview is visually marked as not-yet-saved (e.g., a dashed/distinct border), so the user can see exactly where the entry will sit in their day before committing the change.

**Why this priority**: This is the core of the feature. Seeing the entry's placement in context — relative to the rest of the day — is what lets the user catch a wrong time or duration before saving. Without it, the user is editing numbers blind and only discovers mistakes after the fact.

**Independent Test**: Open the Planning tab, select a scheduled entry, open its edit form, change the Start and/or Duration, and confirm a visually-distinct preview box appears in the day planner at the corresponding time window and moves/resizes as the values change. Fully testable on its own.

**Acceptance Scenarios**:

1. **Given** the day planner is visible alongside the edit form, **When** the user opens the edit form for an entry scheduled at 09:00 for 30m, **Then** a preview box appears in the day planner spanning 09:00–09:30, visually marked as a preview (distinct from saved entries).
2. **Given** the edit form is open, **When** the user changes the Start field to a different valid time, **Then** the preview box moves to the new time window without the user having to save.
3. **Given** the edit form is open, **When** the user changes the Duration field to a different valid value, **Then** the preview box grows or shrinks to match the new duration.
4. **Given** a preview box is shown, **When** the user looks at it next to saved entries, **Then** it is clearly distinguishable as a preview (e.g., dashed border or other distinct visual treatment) and not mistaken for an already-saved entry.
5. **Given** the edit form is open with a preview shown, **When** the user cancels the form, **Then** the preview disappears and the day planner reflects only the saved entries (unchanged).
6. **Given** the edit form is open with a preview shown, **When** the user saves valid changes, **Then** the preview is replaced by the saved entry at that time window.

---

### User Story 2 - Be warned when the new values overlap another entry (Priority: P1)

While editing or scheduling an entry, the user enters a Start/Duration combination that would overlap an existing timed entry on that day. The day planner gives a clear visual indication of the conflict — ideally by showing the overlapping portion of the preview in a warning color (red) — so the user notices and corrects the value before saving.

**Why this priority**: Preventing overlapping entries is the primary stated motivation. The plain preview (Story 1) already reveals overlaps to an attentive user, but an explicit conflict indicator is what reliably stops the mistake. It is co-P1 because the user called it out as a specific need.

**Independent Test**: With at least one existing timed entry on the day, open a form and enter values whose time window intersects that entry; confirm the day planner shows a distinct overlap/conflict indication on the preview. Testable independently of the gap case.

**Acceptance Scenarios**:

1. **Given** an existing entry occupies 10:00–11:00, **When** the user's form values describe a window of 10:30–11:30 (overlapping), **Then** the day planner shows a clear conflict indication tied to the preview (e.g., the overlapping portion rendered in a warning color, or an equivalent unambiguous indicator).
2. **Given** the form values describe a window that overlaps an existing entry, **When** the user adjusts Start/Duration so the window no longer overlaps, **Then** the conflict indication clears.
3. **Given** the user is editing an existing entry, **When** the preview occupies the same time window as that same entry's saved position, **Then** the entry is NOT reported as overlapping itself.
4. **Given** the form values describe a window that does not intersect any other entry, **When** the preview is shown, **Then** no conflict indication appears.

---

### User Story 3 - Catch unintended gaps from a wrong value (Priority: P3)

A user scheduling back-to-back work mistypes a Start time or Duration, which would leave an unexpected gap in the day. Because the live preview shows the entry's real placement relative to neighboring entries, the user sees the gap appear and corrects the value before saving.

**Why this priority**: Gap-avoidance is a secondary motivation the user mentioned. It is served by the same live preview as Story 1 rather than a separate mechanism, so it carries low independent priority — it is a benefit that falls out of the preview being accurate and visible.

**Independent Test**: With an existing entry ending at 10:00, open a form and enter a Start of 10:30 (instead of 10:00); confirm the preview visibly sits below the prior entry with a gap between them, making the unintended gap apparent. Testable via visual inspection of the rendered planner.

**Acceptance Scenarios**:

1. **Given** an existing entry ends at 10:00, **When** the user's form values place the new entry starting at 10:30, **Then** the preview is positioned so the half-hour gap between 10:00 and 10:30 is visible in the day planner.
2. **Given** the user notices the gap, **When** they change Start to 10:00, **Then** the preview moves up to sit flush against the prior entry with no gap.

---

### Edge Cases

- **No valid time yet**: While the Start field is empty or not yet a valid time (and the entry would therefore be untimed), no timed preview box is shown in the calendar portion of the planner. (The feature targets timed placement; untimed entries have no time window to preview.)
- **Invalid/unparseable input**: When the Start or Duration field contains an unparseable value, the preview reflects the last valid interpretation or shows no preview box, rather than rendering at a garbage position. No crash or misplacement occurs.
- **Zero/blank duration**: When duration is blank or zero on an otherwise-timed entry, the preview is shown using the planner's normal handling for a minimal/zero-length entry (consistent with how such entries already render), without breaking layout.
- **Preview outside the visible window**: When the form values place the entry at a time outside the currently visible portion of the day planner, the behavior is consistent with how the planner already handles entries outside its window (the preview need not force the window to scroll, but MUST NOT render at a wrong position).
- **Multiple overlaps**: When the form values overlap more than one existing entry, the conflict indication reflects that an overlap exists (it need not enumerate each conflicting entry).
- **Adjacent, non-overlapping windows**: An entry ending exactly when another begins (e.g., 10:00–10:30 next to 10:30–11:00) is touching but NOT overlapping and MUST NOT trigger the conflict indication.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: While a plan entry form that produces a timed entry is open, the day planner and the form MUST be displayed together in a side-by-side two-pane layout (planner in one pane, form in the other) so the preview is shown in context with existing entries.
- **FR-001a**: The live preview and overlap indication MUST apply to all three timed planning forms — Edit entry, Schedule task, and Add event.
- **FR-002**: The system MUST render a live preview of the entry-being-edited in the day planner at the time window described by the form's current Start and Duration values.
- **FR-003**: The preview MUST update in response to changes to the Start and Duration values as the user fills out the form, without requiring the user to save.
- **FR-004**: The preview MUST be visually distinguishable from saved entries (for example, a dashed or otherwise distinct border) so the user can tell it is an unsaved preview.
- **FR-005**: When the form's current values describe a time window that overlaps any other timed entry on the same day, the system MUST mark the overlapping portion of the preview with a clear conflict indication (preferred: a warning color such as red). The conflict marker applies to the preview only; saved/existing entries MUST NOT be restyled. A simpler unambiguous indicator on the preview is acceptable if highlighting only the overlapping portion is not feasible.
- **FR-006**: When editing an existing entry, the overlap check MUST NOT treat that entry's own saved time window as a conflict with its own preview.
- **FR-007**: The conflict indication MUST clear automatically when the user changes the values so the window no longer overlaps any other entry.
- **FR-008**: Two windows that merely touch at a boundary (one ends exactly when the next begins) MUST NOT be reported as overlapping.
- **FR-009**: When the form's current values do not describe a valid timed window (e.g., empty/invalid Start), the system MUST NOT render a misplaced preview box; it shows no timed preview in that state.
- **FR-010**: Cancelling the form MUST remove the preview and leave the day planner showing only the saved entries, unchanged.
- **FR-011**: Saving valid changes MUST replace the preview with the saved entry at the resulting time window.
- **FR-012**: The preview and overlap indication MUST NOT alter, reorder, or otherwise mutate any saved entry while the form is open.

### Key Entities *(include if feature involves data)*

- **Plan entry (existing)**: A scheduled item in a day's plan, with a name and — for timed entries — a start time and duration that define a time window. Existing concept; not changed by this feature.
- **Entry preview (transient, view-only)**: A non-persisted representation of the entry-being-edited at the time window implied by the in-progress form values. Exists only while a form is open; never stored.
- **Overlap/conflict state (derived)**: A computed indication of whether the preview's time window intersects any other timed entry on the same day, used to drive the conflict indication.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: When a user changes the Start or Duration in an open plan form, the day planner preview reflects the new time window with no perceptible lag (updates appear immediate as the user types/edits).
- **SC-002**: 100% of form value combinations that would overlap an existing timed entry produce a visible conflict indication, and 0% of non-overlapping combinations produce one (including the touching-boundary case, which is not a conflict).
- **SC-003**: In a usability check, users can correctly identify the preview as unsaved versus a saved entry on first look, without being told which is which.
- **SC-004**: Users can detect and correct an overlapping or mistakenly-gapped time value before saving, reducing post-save corrections of overlapping/gapped entries compared to the prior form-only workflow.
- **SC-005**: Opening, editing, cancelling, and saving via the form leaves the saved schedule identical to what the user explicitly committed — the preview never persists and never mutates other entries.

## Assumptions

- **Scope of forms** (confirmed — see Clarifications): The preview and overlap indication apply to all three timed planning forms — editing an existing entry, scheduling a task, and adding an event. See FR-001a.
- **Layout** (confirmed — see Clarifications): While a form is open, the day planner and form are shown together in a side-by-side two-pane layout, replacing the prior form-only view. See FR-001.
- **Gap handling**: Unintended-gap prevention is delivered by the accurate, live preview making gaps visually apparent (Story 3), not by a separate dedicated gap-warning indicator.
- **Overlap definition**: "Overlap" means the half-open time interval of the preview intersects the half-open interval of another timed entry on the same day; touching at a boundary is not an overlap. This matches the existing overlap semantics already present in the planner's domain logic.
- **Untimed entries**: Untimed/backlog entries have no time window and are out of scope for the timed preview box.
- **Visual treatment**: The exact preview styling (dashed border vs. another distinct treatment) and the exact conflict styling (red overlapping portion vs. a simpler indicator) are left to implementation, constrained by FR-004 and FR-005; both styled (color/ANSI) and plain rendering modes should convey the preview and conflict states in a way appropriate to each mode.
- **Day context**: Overlap is evaluated only against entries on the same day currently shown in the planner.
