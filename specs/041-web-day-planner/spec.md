# Feature Specification: Web Day Planner (View)

**Feature Branch**: `041-web-day-planner`

**Created**: 2026-06-06

**Status**: Draft

**Input**: User description: "Show the day planner in the web app. The web app is mostly a companion to the TUI for when the user is on-the-go, so the main use-case is still adding tasks from a mobile browser. But sometimes it would be nice to see the daily plan from there. We don't need to manage the daily plan from there, just view it. The expectation that the user will plan their day in the TUI, but check it later from the webapp."

## Clarifications

### Session 2026-06-06

- Q: How should the user navigate to days other than today? → A: Adjacent-day stepping (previous/next controls) plus a "back to today" affordance; no arbitrary date picker.
- Q: How should planned times display when the viewing device's timezone differs from where the plan was made? → A: Show the planned wall-clock time as-is, with no timezone conversion.
- Q: What happens when a user selects a standalone event entry (no linked task)? → A: Event entries are non-interactive (display only); only task-linked entries are selectable.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Check today's plan on the go (Priority: P1)

A user planned their day earlier in the terminal. Later, away from their computer, they open the web app on their phone to remind themselves what's next. They navigate to the day planner and see today's schedule laid out in time order: the time-blocked tasks and events that make up their day.

**Why this priority**: This is the entire point of the feature — letting the user glance at the plan they built in the TUI from a mobile browser. Viewing today's plan is the smallest slice that delivers the requested value on its own.

**Independent Test**: With a plan already created for today (in the TUI/CLI) containing a few timed entries, sign in on a mobile-sized screen, open the day planner, and confirm today's entries appear in chronological order with their times and names.

**Acceptance Scenarios**:

1. **Given** a signed-in user whose plan for today has entries from 09:00–10:00 and 13:00–14:00, **When** they open the day planner, **Then** both entries are shown in time order, each labeled with its start/end time and name.
2. **Given** a signed-in user, **When** they open the web app, **Then** the day planner is reachable from the app's primary navigation without leaving the signed-in experience.
3. **Given** a plan entry that is linked to a task, **When** the plan is displayed, **Then** the entry shows the task's name.
4. **Given** a plan entry that is a standalone event (e.g. "Lunch"), **When** the plan is displayed, **Then** the entry shows the event name and is visually distinguishable from task-linked entries.
5. **Given** a user with no plan or no entries for today, **When** they open the day planner, **Then** they see a clear empty state indicating today has no plan, rather than a broken or blank view.

---

### User Story 2 - Understand each entry at a glance (Priority: P2)

While viewing the plan, the user wants enough detail on each entry to act on it: how long it lasts, whether a task-linked entry is already done, and the ability to jump to a task's full details to read its description or sub-tasks.

**Why this priority**: Seeing the bare schedule (P1) is the core value; richer per-entry context makes the view genuinely useful but is a refinement on top of the basic display.

**Independent Test**: With a plan containing a timed task entry, a completed task entry, an untimed entry, and an event, open the day planner and confirm durations are shown, the completed task is marked done, untimed items are grouped clearly, and tapping a task-linked entry opens that task's detail view.

**Acceptance Scenarios**:

1. **Given** a timed entry with a start time and duration, **When** the plan is displayed, **Then** the entry's time span (start through end) is shown.
2. **Given** a plan that includes untimed entries (no fixed start time), **When** the plan is displayed, **Then** untimed entries are shown in a clearly separated grouping distinct from the timed schedule.
3. **Given** an entry linked to a completed task, **When** the plan is displayed, **Then** the entry is clearly marked as completed.
4. **Given** an entry linked to a task, **When** the user selects that entry, **Then** they are taken to the task's detail view.
5. **Given** a standalone event entry (no linked task), **When** the user attempts to select it, **Then** nothing happens — the entry is display-only.
6. **Given** a gap between two timed entries, **When** the plan is displayed, **Then** the user can tell the two entries are not contiguous (the open time is apparent).

---

### User Story 3 - Look at another day's plan (Priority: P3)

Occasionally the user wants to see a day other than today — to glance ahead at tomorrow's plan or back at what they had scheduled yesterday.

**Why this priority**: The stated use case is checking *today's* plan on the go; other days are a convenience that builds on the day view and is not required for the feature to be valuable.

**Independent Test**: With plans created for two different dates, open the day planner, move from today to an adjacent day, and confirm the displayed plan changes to that day's entries.

**Acceptance Scenarios**:

1. **Given** plans exist for today and the following day, **When** the user moves the day planner forward one day, **Then** the next day's entries are shown and the displayed date updates accordingly.
2. **Given** the user is viewing a day other than today, **When** they choose to return to today, **Then** today's plan is shown again.
3. **Given** the user navigates to a day that has no plan, **When** that day is displayed, **Then** the same empty state from User Story 1 is shown for that date.

---

### Edge Cases

- **No plan for the day**: The view shows a clear empty state rather than an error or blank screen.
- **Session expired / credentials revoked mid-use**: The user is returned to the sign-in screen, consistent with the rest of the web app.
- **Plan changes in the TUI after the web view loaded**: The web view reflects the latest server state on next load/refresh; it is not expected to update live.
- **Entry with no explicit name and no linked task**: The entry still renders with a sensible fallback label rather than appearing empty.
- **Task-linked entry whose task was deleted elsewhere**: The entry is shown gracefully (or omitted) without breaking the view; selecting it does not lead to a broken task page.
- **Very long plan spanning many hours**: The schedule remains readable and scrollable on a small screen.
- **Many untimed entries**: The untimed grouping remains readable and does not crowd out the timed schedule.
- **Connectivity drops while loading the plan**: The user is told the plan could not be loaded and can retry.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The web app MUST provide a day planner view that displays a single day's plan.
- **FR-002**: The day planner MUST be reachable from the web app's primary navigation by a signed-in user.
- **FR-003**: The day planner MUST default to showing today's plan when first opened.
- **FR-004**: The view MUST display the day's timed entries in chronological order.
- **FR-005**: For each timed entry, the view MUST show its time span (start time and end time or duration).
- **FR-006**: The view MUST display the entry's name; for an entry linked to a task with no separate name, it MUST show the linked task's name.
- **FR-007**: The view MUST visually distinguish task-linked entries from standalone events.
- **FR-008**: The view MUST indicate when a task-linked entry's task is completed.
- **FR-009**: The view MUST display untimed entries (entries without a fixed start time) in a grouping clearly separated from the timed schedule.
- **FR-010**: The view MUST make gaps between non-contiguous timed entries apparent.
- **FR-011**: Selecting a task-linked entry MUST navigate the user to that task's detail view within the web app.
- **FR-012**: When the selected day has no plan or no entries, the view MUST display a clear empty state identifying the date.
- **FR-013**: The day planner MUST be read-only: it MUST NOT offer creating, editing, rescheduling, reordering, or deleting plan entries.
- **FR-014**: The view MUST be usable on a mobile-sized screen (the primary target), including readable layout and scrolling for long plans.
- **FR-015**: The view MUST reflect the current server-side plan state when loaded or refreshed; live/automatic updates are not required.
- **FR-016**: Accessing the day planner MUST require an authenticated session, reusing the web app's existing sign-in/session mechanism.
- **FR-017**: When the plan cannot be loaded (e.g. connectivity or server error), the view MUST inform the user and allow a retry.
- **FR-018**: Users SHOULD be able to view a day other than today via previous/next day controls (one day at a time) and a control to return to today. An arbitrary date picker is out of scope. *(Supports User Story 3; may follow the P1 view.)*
- **FR-019**: The view MUST render entries with missing or empty names using a sensible fallback label rather than blank space.
- **FR-020**: Entry times MUST be displayed as the planned wall-clock time for the plan's date, without converting to the viewing device's timezone.
- **FR-021**: Standalone event entries MUST be non-interactive (display only); only task-linked entries are selectable/navigable (see FR-011).

### Key Entities *(include if feature involves data)*

- **Daily Plan**: The schedule for one specific calendar day belonging to the user. It is a collection of plan entries for that date. Created and managed elsewhere (the TUI/CLI); the web app only reads it.
- **Plan Entry**: A single item on a day's plan. May be *timed* (has a start time and a duration) or *untimed* (no fixed start). May be *task-linked* (references an existing task, deriving its name and completion status) or a *standalone event* (has its own name, e.g. "Lunch"). Has an order/identity within the day.
- **Task**: An existing user task that a plan entry may link to. The day planner reads a linked task's name and completion status and links through to its detail view. Tasks are managed by existing web app features.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A signed-in user can go from opening the web app to seeing today's plan in at most two interactions (e.g. one navigation tap).
- **SC-002**: For a plan built in the TUI, 100% of that day's entries (timed, untimed, task-linked, and events) are represented in the web day planner view.
- **SC-003**: A user can correctly identify, from the web view alone, the start time and name of any timed entry and whether a task-linked entry is complete, without consulting the TUI.
- **SC-004**: On a mobile-sized screen, the day planner is fully usable (all entries readable and reachable by scrolling) with no horizontal scrolling required.
- **SC-005**: When a day has no plan, users see an empty state (not an error) in 100% of such cases.
- **SC-006**: No action available in the day planner modifies plan data — the view is verifiably read-only.

## Assumptions

- The daily plan is authored and maintained in the TUI/CLI; the web app's role is strictly viewing. Managing the plan from the web is explicitly out of scope.
- The existing plan data (for a given user and date) is retrievable through the existing backend without server-side changes; if the current API does not expose what the view needs, surfacing that is a planning concern, but the product intent assumes no new management capabilities.
- Authentication and session behavior reuse the existing web app mechanism (long-lived session from prior features); no new auth is introduced.
- "Plan" means a single-day plan as defined by the existing daily-planning feature ([006-daily-planning](../006-daily-planning/spec.md)), including untimed entries ([027-untimed-plan-entries](../027-untimed-plan-entries/spec.md)).
- Times are shown as the planned wall-clock time for the plan's date (no timezone conversion), consistent with how the TUI presents the plan (see FR-020).
- The day planner presents the plan as a mobile-friendly chronological list/timeline rather than reproducing the terminal's fixed ASCII grid; the exact visual treatment is a design decision left to planning.
- Highlighting the "current" time or in-progress entry is a possible enhancement but not required for this feature.
