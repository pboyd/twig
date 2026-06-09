# Feature Specification: Basic Planning Edits on the Web App

**Feature Branch**: `045-web-plan-edits`

**Created**: 2026-06-09

**Status**: Draft

**Input**: User description: "Basic planning edits on the webapp. Allow users to add unscheduled tasks to a plan from the webapp. From the tasks list, the user should be able to add it to today's plan or a future plan (adding to a past plan does not need to be prevented, although there's no good reason for a user to do that). Most commonly, the user will be adding a task to that day's plan, so this should be the most convenient option. Users should be able to at least remove unscheduled entries from the plan (necessary to fix a mistake). If this can easily be extended to scheduled tasks and events then that is fine. Users should also be able to complete a task directly from the planning tab (currently, they may edit the task to mark it done, but it would be nice to complete the task directly from the plan without an extra step)."

## Clarifications

### Session 2026-06-09

- Q: When adding a task to a day other than today, how should the user pick the target day? → A: A "Today" one-tap shortcut, plus quick presets (e.g. Tomorrow) and a full date picker for any arbitrary day.
- Q: If the task is already on the chosen day's plan (as an untimed entry), what should adding it again do? → A: Prevent the duplicate and inform the user — matching the existing terminal behavior. (The server already enforces this: `AddPlanTask` rejects a second untimed entry for the same task+day with a clear message, so the web app surfaces that rejection rather than performing its own check.)
- Q: How should a successful "add to plan" be confirmed to the user? → A: A brief, auto-dismissing toast (naming the day for non-today adds); this introduces a small reusable transient-notification primitive in the web app.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add a task to today's plan from the tasks list (Priority: P1)

A user browsing their task list on the web app decides a task should be part of today. Without leaving the list, they trigger a single, obvious action that adds the task to today's plan as an unscheduled (untimed) entry. They get clear confirmation that it landed on today's plan.

**Why this priority**: This is the most common planning edit the user will make from the web app, and the user explicitly asked for it to be the most convenient option. It is the smallest slice that delivers the core value — turning the read-only web planner into something the user can feed from their phone.

**Independent Test**: Sign in, open the tasks list, choose the "add to today" action on a task, then open the day planner on today and confirm the task appears as an untimed entry.

**Acceptance Scenarios**:

1. **Given** a signed-in user viewing their tasks list, **When** they invoke the "add to today" action on a task, **Then** the task is added to today's plan as an untimed entry and the user sees a brief toast confirming it was added.
2. **Given** a task that was just added to today's plan, **When** the user opens the day planner for today, **Then** the task is shown in the untimed grouping.
3. **Given** the "add to today" action, **When** the user uses it, **Then** adding to today requires no additional day-selection step (it is a single action).
4. **Given** a task that already has an untimed entry on today's plan, **When** the user invokes "add to today" again for it, **Then** no duplicate entry is created and the user is told it is already on that day's plan.
5. **Given** the add action fails (e.g., connectivity drops), **When** the user invokes it, **Then** the user is told it did not succeed and the tasks list is unchanged.

---

### User Story 2 - Complete a task directly from the planning tab (Priority: P2)

While looking at the day planner, the user finishes a task that is on the plan. Instead of opening the task to edit it, they mark it done directly from the plan entry in one action, and the plan reflects the completion immediately.

**Why this priority**: The user can already complete tasks by editing them, so this is a convenience improvement rather than a new capability — but it removes a real point of friction in the day-planning flow. It is independent of the add/remove stories.

**Independent Test**: With a task-linked entry on today's plan, use the complete action on that entry from the planner and confirm the task becomes complete (verifiable in the tasks list) without opening a separate edit screen.

**Acceptance Scenarios**:

1. **Given** a task-linked entry on the displayed plan, **When** the user invokes the complete action on that entry, **Then** the linked task is marked complete without the user navigating to a separate screen.
2. **Given** an untimed task-linked entry that the user completes from the plan, **When** the plan refreshes, **Then** the entry is hidden from the untimed grouping (consistent with how completed untimed entries are already treated).
3. **Given** a timed task-linked entry that the user completes from the plan, **When** the plan refreshes, **Then** the entry remains visible and is clearly marked as completed.
4. **Given** a task that has incomplete sub-tasks, **When** the user tries to complete its plan entry, **Then** the action is refused with a clear message explaining its sub-tasks must be finished first, and the task stays incomplete.
5. **Given** a standalone event entry (no linked task), **When** the plan is displayed, **Then** no complete action is offered for it.

---

### User Story 3 - Add a task to a future day's plan (Priority: P2)

The user wants to plan ahead: from the tasks list they add a task to a day other than today (for example, tomorrow or later this week). They choose the target day and the task is added to that day's plan as an untimed entry.

**Why this priority**: Planning future days is valuable but secondary to the dominant "add to today" case. It builds directly on User Story 1 by adding day selection.

**Independent Test**: From the tasks list, choose the "add to another day" action on a task, pick a future date, then open the planner on that date and confirm the task appears as an untimed entry.

**Acceptance Scenarios**:

1. **Given** a signed-in user viewing their tasks list, **When** they choose to add a task to a day other than today and select a future date, **Then** the task is added to that day's plan as an untimed entry and the user sees a toast naming the day it was added to.
2. **Given** the day-selection step, **When** the user picks the target day, **Then** they can choose "Today", a quick preset such as "Tomorrow", or any arbitrary date via a date picker.
3. **Given** a task added to a future day, **When** the user opens the planner on that day, **Then** the task appears in the untimed grouping.
4. **Given** the day-selection step, **When** the user selects a past date, **Then** the system still adds the task to that day (past days are not blocked).

---

### User Story 4 - Remove an entry from the plan (Priority: P3)

The user added something to a plan by mistake (or no longer wants it on the plan) and removes that entry from the day planner. The entry disappears from the plan; the underlying task is untouched.

**Why this priority**: Removal is the safety net for the add stories — needed to fix mistakes — but the add and complete flows deliver value first. It is listed last because the worst case of a mis-add is a stray plan entry, not data loss.

**Independent Test**: With an entry on the displayed plan, use the remove action on it and confirm the entry is gone from the plan while the task still exists in the tasks list.

**Acceptance Scenarios**:

1. **Given** an untimed entry on the displayed plan, **When** the user removes it, **Then** the entry no longer appears on that day's plan.
2. **Given** a removed entry that is linked to a task, **When** the user checks the tasks list, **Then** the task still exists and is unchanged (removal affects only the plan entry).
3. **Given** a timed entry or a standalone event on the plan, **When** the user removes it, **Then** it is also removed (removal is available for all entry kinds, not only untimed task entries).
4. **Given** a remove action that fails, **When** the user invokes it, **Then** the user is told it did not succeed and the entry remains on the plan.

---

### Edge Cases

- **Task already on the chosen day's plan**: Adding a task that already has an untimed entry on that day is rejected; the user is told it is already on that day and no duplicate is created (the server enforces this).
- **Adding a completed task to a plan**: Permitted; the resulting untimed entry follows the existing rule that completed untimed entries are hidden, so it may not appear in the untimed grouping.
- **Completing the last visible untimed entry**: After completion the untimed grouping may become empty; the plan view shows the appropriate empty/grouping state without breaking.
- **Removing the only entry on a day**: The plan view falls back to its existing empty state for that day.
- **Concurrent change**: If a task or entry was already completed/removed elsewhere, the action resolves gracefully (no error surfaced for an already-true outcome where reasonable) and the view reflects the latest state on refresh.
- **Session expired mid-action**: The user is returned to the sign-in screen, consistent with the rest of the web app.
- **Connectivity drops during an edit**: The user is told the action could not be completed and can retry; no partial or incorrect state is shown.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: From the tasks list, the web app MUST let a signed-in user add a task to a day's plan as an untimed entry.
- **FR-002**: The web app MUST provide a single-action way to add a task to **today's** plan that requires no day-selection step.
- **FR-003**: The web app MUST let the user add a task to a day other than today by choosing the target day, offering a "Today" shortcut, at least one quick preset (e.g. "Tomorrow"), and a date picker for selecting any arbitrary day.
- **FR-004**: The system MUST NOT block adding a task to a past day.
- **FR-005**: After a successful add, the web app MUST confirm it with a brief, auto-dismissing toast notification; for non-today targets the toast MUST identify the day the task was added to.
- **FR-006**: When the user adds a task that already has an untimed entry on the chosen day, the web app MUST NOT create a duplicate and MUST tell the user the task is already on that day (surfacing the server's rejection).
- **FR-007**: When the user later views the target day's planner, a successfully added task MUST appear there as an untimed entry, subject to existing display rules for completed untimed entries.
- **FR-008**: From the day planner, the web app MUST let the user remove a plan entry, after which that entry no longer appears on the day's plan.
- **FR-009**: Removing a plan entry MUST NOT delete or otherwise modify any linked task.
- **FR-010**: Entry removal MUST be available for untimed entries at minimum, and SHOULD also be available for timed entries and standalone events.
- **FR-011**: From the day planner, the web app MUST let the user mark a task-linked entry's task complete in a single action, without navigating to a separate edit screen.
- **FR-012**: The complete action MUST NOT be offered for standalone event entries (which have no linked task).
- **FR-013**: When completing a task is refused because it has incomplete sub-tasks, the web app MUST show a clear message and leave the task incomplete.
- **FR-014**: After a completion or removal, the plan view MUST reflect the change without requiring a manual full-page reload.
- **FR-015**: When any add, remove, or complete action fails, the web app MUST inform the user and leave the prior state intact (no silent failure, no inconsistent display).

### Key Entities *(include if feature involves data)*

- **Plan entry**: A single item on a specific day's plan. May be linked to a task or be a standalone event; may be untimed (no fixed start time) or timed. This feature adds untimed task entries, removes entries, and reflects completion state.
- **Task**: A unit of work the user tracks. A plan entry may reference one. This feature adds a task to a plan and marks a task complete; it never deletes a task.
- **Day**: The date a plan entry belongs to — the unit the user selects when adding to a future day and the scope the planner displays.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can add a task to today's plan from the tasks list in a single action, with no intermediate day-selection step.
- **SC-002**: A user can add a task to a chosen future day's plan in no more than three actions (invoke add, choose day, confirm).
- **SC-003**: A user can mark a planned task complete from the day planner without opening any separate task-edit screen.
- **SC-004**: A user can remove a mistakenly added plan entry in no more than two actions from the day planner.
- **SC-005**: After a successful add, complete, or remove, the relevant view reflects the new state within the same session without a manual page reload.
- **SC-006**: 100% of failed add/remove/complete actions surface a visible message to the user rather than failing silently or showing stale or incorrect state.

## Assumptions

- The existing plan and task back-end operations are sufficient: adding an untimed task entry, removing an entry by day and id, and marking a task complete are all already available on the server, so no back-end changes are required.
- "Unscheduled" / "untimed" entries are the kind produced when adding from the tasks list (no start time is chosen during the add).
- Removing a plan entry takes effect immediately without a separate confirmation dialog; the low stakes (the task is untouched and can be re-added) make an explicit confirmation unnecessary.
- Completed untimed entries continue to be hidden from the planner per the existing behavior; this feature does not change that rule.
- The web app remains a companion to the terminal app; these edits bring the most common planning actions to the web without aiming for full plan-management parity (setting times, reordering, and renaming entries are out of scope here).
