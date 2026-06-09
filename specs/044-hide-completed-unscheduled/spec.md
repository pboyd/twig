# Feature Specification: Hide Completed Unscheduled Plan Entries

**Feature Branch**: `044-hide-completed-unscheduled`

**Created**: 2026-06-09

**Status**: Draft

**Input**: User description: "When an unscheduled task (that is, a plan entry without a start time, with a linked task) is completed, it should no longer appear in the plan view on the TUI or the webapp."

## Clarifications

### Session 2026-06-09

- Q: In the TUI, after an unscheduled entry is completed and stays crossed-out + highlighted, which navigation should cause it to drop out of the plan? → A: Any re-render where it is no longer the highlighted entry — moving the highlight, switching tabs, or changing the displayed day all hide it (retained only while it is the active highlight, matching the tasks tab).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Completing an unscheduled task clears it from the plan (Priority: P1)

A user is working through their day in the planner. They have a handful of unscheduled (untimed) tasks parked in their plan for the day — small, must-do-today items they did not commit to a specific time. As they finish each one and mark it complete, it drops out of the plan view, leaving only the unscheduled work that still needs doing. The planner stays focused on what is left rather than accumulating finished clutter.

**Why this priority**: This is the entire feature. An unscheduled plan entry exists to remind the user a task still needs doing today; once the task is done, the reminder has served its purpose and only adds noise. Removing completed unscheduled entries from the plan view delivers the full value on its own.

**Independent Test**: Place an unscheduled task (a plan entry with a linked task and no start time) on a day's plan, view that day's plan, confirm the entry appears, then mark the linked task complete and confirm the entry no longer appears in the plan view.

**Acceptance Scenarios**:

1. **Given** a day's plan containing an unscheduled entry linked to an incomplete task, **When** the user opens the plan view, **Then** the unscheduled entry is displayed.
2. **Given** an unscheduled entry linked to a task, **When** the linked task is marked complete and the plan is next shown afresh, **Then** the entry no longer appears in the plan view for that day.
3. **Given** a day's plan containing several unscheduled entries, **When** one of their linked tasks is completed, **Then** only that entry is removed from the plan view and the remaining unscheduled entries stay visible.

---

### User Story 2 - Completing an unscheduled task in the TUI keeps it in place until you move on (Priority: P1)

A user is in the TUI planning view and marks the highlighted unscheduled task complete. Rather than the entry vanishing out from under the cursor, it stays where it is, shown as done (crossed out) and still highlighted — exactly the way completing a task on the tasks tab behaves. This confirms the action visually and avoids a jarring jump. Only when the user navigates the highlight away from that entry does it drop out of the plan.

**Why this priority**: Completion is the trigger for the whole feature, and in the TUI the *interaction* matters as much as the outcome. An entry that disappears instantly on completion would move the highlight unexpectedly and feel inconsistent with the tasks tab. Deferring the hide until the user navigates away makes completing tasks in the planner predictable and consistent with the rest of the TUI.

**Independent Test**: In the TUI planning view, highlight an unscheduled entry and mark its task complete. Confirm the entry stays visible, crossed out, and still highlighted. Then move the highlight to another entry and confirm the completed entry is now gone from the plan.

**Acceptance Scenarios**:

1. **Given** a highlighted unscheduled entry in the TUI planning view, **When** the user marks its task complete, **Then** the entry remains visible, is shown crossed out, and stays highlighted.
2. **Given** a just-completed unscheduled entry that is still highlighted, **When** the user moves the highlight to another entry, **Then** the completed entry is removed from the plan view.
3. **Given** a just-completed unscheduled entry that is still highlighted, **When** the user switches to another tab or changes the displayed day and then returns to the plan, **Then** the completed entry is no longer shown (it is retained only while it is the active highlight).
4. **Given** a just-completed unscheduled entry that is still highlighted, **When** the user reopens (uncompletes) it before navigating away, **Then** the entry is no longer crossed out and continues to be shown normally.

---

### User Story 3 - Completed-then-reopened task returns to the plan (Priority: P2)

A user completes an unscheduled task — it disappears from the plan — and then realizes the work is not actually finished. They reopen (uncomplete) the task. The unscheduled entry reappears in the plan view so they can continue tracking it as outstanding work for the day.

**Why this priority**: Hiding completed work is only safe if reopening a task brings its plan entry back; otherwise an accidental completion would silently lose the day's planning. It depends on Story 1 and is secondary to it, but is needed for the behavior to be trustworthy.

**Independent Test**: Complete an unscheduled task so its entry leaves the plan view, then reopen the task and confirm the unscheduled entry reappears in the plan view for that day.

**Acceptance Scenarios**:

1. **Given** an unscheduled entry hidden because its linked task is complete, **When** the user reopens (uncompletes) the task, **Then** the entry reappears in the plan view for that day.
2. **Given** a reopened task whose entry has reappeared, **When** the user views the plan, **Then** the entry behaves like any other unscheduled entry (highlightable and editable).

---

### Edge Cases

- **Scheduled (timed) entries are unaffected**: Completing a task linked to a *timed* plan entry (one with a start time, shown on the day grid) does not hide that entry — only unscheduled entries are removed on completion. The timed entry remains so the user retains a record of what they did and when.
- **Events have no linked task**: Event entries (which by definition have no linked task and cannot be completed) are never affected by this behavior.
- **Same task on multiple days**: If the same task is linked to unscheduled entries on more than one day, completing the task removes the unscheduled entries on all affected days; reopening restores them.
- **Last entry of the day**: When completing a task removes the only remaining unscheduled entry, the unscheduled area behaves exactly as it does for a day that never had any unscheduled entries (e.g., the dedicated pane hides itself per existing behavior).
- **Underlying plan entry is preserved**: Completion hides the entry from view; it does not delete the plan entry. The association between the day, the entry, and the task is retained so reopening can restore it.
- **Completing the currently highlighted entry in the TUI**: The completed entry is not removed while it is still highlighted — it stays in place, crossed out, until the highlight moves off it. This avoids the highlight jumping unexpectedly and matches the tasks-tab completion interaction.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The plan view MUST exclude any unscheduled entry (a plan entry with a linked task and no start time) whose linked task is complete.
- **FR-002**: The plan view MUST continue to display unscheduled entries whose linked task is incomplete.
- **FR-003**: The exclusion MUST apply consistently in both the TUI planning view and the web planner.
- **FR-004**: The exclusion MUST NOT affect timed plan entries (entries with a start time); a timed entry linked to a completed task MUST still appear in the plan view.
- **FR-005**: The exclusion MUST NOT affect event entries (entries with no linked task).
- **FR-006**: When a previously completed task linked to an unscheduled entry is reopened (uncompleted), the plan view MUST again display that entry.
- **FR-007**: Hiding a completed unscheduled entry MUST NOT delete or otherwise destroy the underlying plan entry or its link to the task and day.
- **FR-008**: The behavior MUST apply to the plan for any day being viewed (past, present, or future), based on the current completion state of each entry's linked task.
- **FR-009**: In the TUI planning view, completing the linked task of a currently highlighted unscheduled entry MUST NOT remove the entry immediately; the entry MUST remain visible, shown as done (crossed out), and stay highlighted.
- **FR-010**: In the TUI, a completed unscheduled entry is retained only while it is the active highlight. It MUST be removed from the plan on any subsequent render where it is no longer the highlighted entry — including moving the highlight to another entry, switching to a different tab, or changing the displayed day — matching the tasks-tab completion interaction.
- **FR-011**: If a retained-and-highlighted completed unscheduled entry is reopened (uncompleted) before the highlight moves away, the TUI MUST stop showing it as done and keep displaying it normally.

### Key Entities *(include if feature involves data)*

- **Plan entry**: An item placed on a specific day's plan. May be *timed* (has a start time, rendered on the day grid) or *unscheduled/untimed* (no start time). May be linked to a task, or be a standalone event with no task link.
- **Task**: A unit of work that can be marked complete or incomplete. A task may be linked to one or more plan entries across one or more days.
- **Completion state**: Whether a task is complete or incomplete. This state determines whether the task's *unscheduled* plan entries are shown in the plan view.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After a user marks an unscheduled task complete, its entry is gone from the plan view the next time the plan is shown, with no manual refresh or extra step required by the user.
- **SC-002**: 100% of completed unscheduled entries are excluded from the plan view, while 100% of incomplete unscheduled entries and all timed entries (regardless of completion) remain visible.
- **SC-003**: The plan view shows identical inclusion/exclusion results for the same underlying data in both the TUI and the web planner.
- **SC-004**: Reopening a completed task restores its unscheduled entry to the plan view, so no planning information is permanently lost by an accidental completion.

## Assumptions

- "Completed" refers to the linked task's completion state (the same notion of completion used elsewhere in the product, e.g. the task list), not a separate per-plan-entry status.
- Hiding is a *view-time* filter: the plan entry and its links are preserved, which is what makes reopening able to restore the entry. No data is deleted on completion.
- This feature does not add a reveal/show-completed toggle to the plan view (unlike the task-list "show completed" control); completed unscheduled entries are simply not shown in the planner. Surfacing them there is out of scope.
- The scope is limited to *unscheduled* entries. Timed entries deliberately remain visible when their task is completed, preserving an as-done record on the day grid; changing that behavior is out of scope.
- The web planner referenced here is the existing day planner in the web app; this feature changes only which unscheduled entries it displays, not how entries are rendered.
- The deferred-hide interaction (entry stays crossed out and highlighted until the user navigates away) is specific to the TUI, to match the existing tasks-tab completion behavior. The web planner is not required to reproduce this keyboard-highlight interaction; it applies the same view-time exclusion on its normal render/refresh.
