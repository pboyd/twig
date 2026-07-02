# Feature Specification: Web Plan Tab Day-Planner Redesign

**Feature Branch**: `061-web-plan-redesign`

**Created**: 2026-07-02

**Status**: Draft

**Input**: User description: "the plan tab in the web app is functionally fine, but it's UI/UX is a bit of a mess. The schedule should look like a day planner, but it's just a list. Tasks should also be more consistent with the tasks tab. For instance, there's a empty circle to mark tasks 'done' on the tasks tab whereas the planning tab has a checkbox. I don't care really if it's a checkbox or a circle, but I do want them to be consistent."

## Clarifications

### Session 2026-07-02

- Q: What time window should the day-planner timeline display? → A: Auto-fit to entries — window spans first to last timed entry, expanded outward to hour boundaries, with a modest default window (e.g., 8 am–5 pm) when the day is empty or sparse (matches the terminal planner).
- Q: Should the done control on the Plan tab toggle both ways (complete and un-complete), like the Tasks tab's circle control? → A: Two-way toggle — tapping a completed entry's filled circle re-opens the task, exactly like the Tasks tab.
- Q: How should timed entries that overlap in time be laid out on the timeline? → A: Not applicable — the server rejects overlapping entries, so the timeline may assume timed entries never overlap.
- Q: Where should the untimed-entries section sit relative to the timeline? → A: Above the timeline — untimed checklist first, then the hour-ruled schedule below, matching the terminal UI's Plan tab.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the day as a planner, not a list (Priority: P1)

A user opens the Plan tab to see their day. Instead of a flat list of entries with time ranges printed above each one, they see a day-planner timeline: an hour-labeled time column down the side, with each scheduled entry drawn as a block whose vertical position corresponds to its start time and whose height corresponds to its duration. Gaps between appointments are visible as empty space, so the shape of the day — busy stretches, free stretches — is readable at a glance.

**Why this priority**: This is the core complaint. The schedule currently reads as an undifferentiated list; the whole point of a daily plan view is to convey *when* things happen and how the day is shaped.

**Independent Test**: Plan a day with two timed entries separated by a gap (e.g., 8:00–8:30 am and 11:00 am–12:00 pm). Open the Plan tab and verify the entries appear as blocks aligned to an hour-labeled timeline, with the 30-minute block visibly shorter than the 60-minute block and empty space between them.

**Acceptance Scenarios**:

1. **Given** a day with timed entries, **When** the user opens the Plan tab, **Then** each timed entry appears as a block positioned against a time axis at its start time, with visible hour labels/lines.
2. **Given** two entries of different durations, **When** viewing the schedule, **Then** the longer entry's block is proportionally taller than the shorter entry's block.
3. **Given** a gap between two timed entries, **When** viewing the schedule, **Then** the gap appears as empty timeline space between the two blocks.
4. **Given** a day with only untimed entries, **When** the user opens the Plan tab, **Then** the untimed entries are shown as a list (no timeline is required for a day with nothing scheduled).
5. **Given** a day with no entries at all, **When** the user opens the Plan tab, **Then** the existing empty-state message is shown.

---

### User Story 2 - Consistent task controls across tabs (Priority: P2)

A user who marks tasks done on the Tasks tab (by tapping the empty circle that fills in green with a check) switches to the Plan tab and finds the *same* control on plan entries that are tasks — not a different-looking checkmark button. Completed entries look the same in both places: filled circle plus struck-through name.

**Why this priority**: Inconsistent controls make the app feel disjointed and force users to re-learn the same action per tab. It's the second explicit complaint, but the feature is still usable without it — hence P2.

**Independent Test**: Place an incomplete task on today's plan. Compare its done-control on the Plan tab against the same task's control on the Tasks tab — they must be visually and behaviorally the same. Mark it done on the Plan tab and verify the completed rendering matches the Tasks tab's completed rendering.

**Acceptance Scenarios**:

1. **Given** an incomplete task on the plan, **When** viewing it on the Plan tab, **Then** its mark-done control uses the same visual affordance (empty circle) as the Tasks tab.
2. **Given** an incomplete task on the plan, **When** the user activates the done control, **Then** the task is completed and the entry renders in the same completed style used on the Tasks tab (filled circle with check, struck-through name).
3. **Given** a completed task entry, **When** viewing it on the Plan tab, **Then** its completed indicator matches the Tasks tab's completed indicator (no separate/different "completed" badge style).
4. **Given** a plan entry that is a free-form event (not a task), **When** viewing it, **Then** no done control is shown (events cannot be completed today; unchanged).
5. **Given** a task that cannot be completed because it has open subtasks, **When** the user attempts to complete it from the Plan tab, **Then** the existing explanatory message is shown (behavior unchanged).
6. **Given** a completed task entry on the plan, **When** the user activates its done control, **Then** the task is re-opened (marked incomplete), matching the Tasks tab's toggle behavior.

---

### User Story 3 - Orient within today (Priority: P3)

While viewing today's plan, the user can see where "now" falls on the timeline via a current-time indicator, making it obvious what's next and what's already passed.

**Why this priority**: A polish feature that reinforces the day-planner metaphor (and matches the terminal UI, which already marks the current time), but the redesign delivers value without it.

**Independent Test**: Open today's plan during a scheduled entry and verify a current-time marker appears at the correct position on the timeline; navigate to another day and verify the marker is absent.

**Acceptance Scenarios**:

1. **Given** the user is viewing today's plan, **When** the timeline is displayed, **Then** a visual indicator marks the current time at the correct position.
2. **Given** the user is viewing a day other than today, **When** the timeline is displayed, **Then** no current-time indicator is shown.

---

### Edge Cases

- **Very short entries** (e.g., 15 minutes): the block must still be tall enough to show the entry name and offer usable touch targets for its actions.
- **Long entry names** in a narrow viewport: names may truncate or wrap within their block but must not break the timeline layout.
- **Entries late in the day / spanning many hours**: the timeline must extend to contain every entry's full span.
- **Narrow (mobile-width) viewports**: the timeline, entry blocks, and action controls must remain usable at phone widths — the current UI is used at narrow sizes (see input screenshot).
- **Untimed entries**: remain in a separate section, now placed above the timeline, and are unaffected by the timeline layout aside from the consistent done control.
- **Action errors** (connectivity, blocked completion): per-entry error messages must still be presentable in the new layout.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Plan tab MUST render timed entries on a vertical timeline with labeled hour lines, where each entry is a block whose position corresponds to its start time and whose height is proportional to its duration.
- **FR-002**: The timeline's visible window MUST auto-fit the day's timed entries: it spans from the first to the last timed entry, expanded outward to hour boundaries, and falls back to a modest default window (e.g., 8 am–5 pm) when the day has no or few timed entries (consistent with the product's existing terminal planner).
- **FR-003**: Gaps between scheduled entries MUST be visually apparent as empty timeline space.
- **FR-004**: Task entries on the Plan tab MUST use the same mark-done control (same shape, states, and interaction) as task rows on the Tasks tab, including two-way toggling: activating a completed entry's control re-opens the task, subject to the same rules as the Tasks tab (e.g., blocked when the parent task is complete, with the existing explanatory message).
- **FR-005**: Completed task entries on the Plan tab MUST use the same completed-state presentation as the Tasks tab (filled indicator plus struck-through name), replacing the Plan tab's current separate completed badge.
- **FR-006**: Each entry MUST retain its existing actions and behavior: navigating to the task's detail page from a task entry, removing the entry from the plan, and completion rules (including the blocked-by-open-subtasks message).
- **FR-007**: Free-form event entries (non-task entries) MUST continue to display without a done control and remain removable.
- **FR-008**: Untimed entries MUST appear in a distinct section rendered above the timeline (matching the terminal UI's Plan tab layout), using the same consistent done control as FR-004.
- **FR-009**: When viewing today, the timeline MUST display a current-time indicator at the correct position; the indicator MUST NOT appear on other days.
- **FR-010**: Day navigation (previous/next/Today) and the loading, error, and empty states MUST continue to work as they do today.
- **FR-011**: The redesigned layout MUST remain fully usable at narrow (phone-width) viewports, including comfortably tappable entry actions.

### Key Entities

- **Plan entry**: an item on a day's plan; either a *task entry* (linked to a task, completable) or an *event* (free-form label, not completable). May be *timed* (start time + duration) or *untimed*.
- **Day plan**: the set of plan entries for a calendar day, split into timed (timeline) and untimed (list) groups.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user viewing a populated day can identify each entry's approximate start time, duration, and the free gaps in their day from the layout alone, without reading per-entry time-range text.
- **SC-002**: The mark-done control on Plan tab task entries is visually indistinguishable from the Tasks tab's control, in both incomplete and completed states, in light and dark themes.
- **SC-003**: 100% of existing Plan tab capabilities (complete a task, remove an entry, open a task's detail page, navigate between days, loading/error/empty states) remain available after the redesign, verified by the existing and updated test suite passing.
- **SC-004**: On a phone-width viewport, every per-entry action on the timeline can be activated on the first attempt (touch targets meet the app's existing minimum size convention).

## Assumptions

- "Day planner" means a single-day vertical timeline (hour-ruled agenda view), not a week/month calendar; day-to-day movement stays on the existing prev/next/Today navigation.
- The circle-style done control from the Tasks tab is the one to standardize on (the user expressed no preference, and the Tasks tab is the primary surface; the terminal UI is out of scope).
- The done control is fully consistent with the Tasks tab, including two-way toggling (clarified 2026-07-02): completing and re-opening are both possible from the Plan tab, with the same error handling as the Tasks tab.
- Untimed completed task entries remain hidden from the untimed section (current behavior); completed *timed* entries remain visible on the timeline.
- The timeline window follows the existing terminal-planner convention: sized to contain all timed entries, expanded to hour boundaries; a modest default window (e.g., 8 am–5 pm) is shown when there are no or few timed entries (clarified 2026-07-02).
- No backend or data-model changes are needed; this is a presentation-layer redesign of an existing, functionally complete feature.
- Timed entries never overlap — the server rejects overlapping plan entries (clarified 2026-07-02), so the timeline does not need an overlap layout strategy.
- Drag-to-reschedule, resizing blocks, or creating entries by tapping the timeline are out of scope; entry creation stays on the Tasks tab's add-to-plan control.
