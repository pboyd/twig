# Feature Specification: Plan Task Actions

**Feature Branch**: `029-plan-task-actions`

**Created**: 2026-06-01

**Status**: Draft

**Input**: User description: "users should be able to interact with the linked tasks from the planning tab. Two basic actions need to be performed: Complete a linked task (more properly, toggle complete, but the primary use-case is to complete a task not un-complete one) and Start a pomodoro for a linked task"

## User Scenarios & Testing *(mandatory)*

A plan is a day's worth of entries. Some entries are **linked to a task** (they were created by sending a task to the plan); others are bare **events** with no underlying task. Today the planning tab can show, add, edit, move, and remove entries, but it cannot act on the task behind a task-linked entry — to mark that task done or to start working on it, the user has to leave the planning tab, find the task in the Tasks tab, and act there. This feature lets the user act on the linked task directly from the planning tab, using the same gestures the Tasks tab already uses.

### User Story 1 - Complete a linked task from the plan (Priority: P1)

While reviewing the day's plan, the user finishes a planned task and marks it done without leaving the planning tab. They select the entry for that task and trigger "complete"; the task is marked complete and the entry shows as completed in place.

**Why this priority**: Marking work done as you go is the core daily-planning loop. It is the most frequent interaction and delivers the headline value of the feature on its own. Completing tasks where the user is already looking (the plan) removes the most common reason to bounce back to the Tasks tab.

**Independent Test**: With a plan containing at least one task-linked entry, select that entry and trigger complete. Verify the linked task becomes complete (reflected both in the planning tab and in the Tasks tab) and that triggering it again un-completes the task.

**Acceptance Scenarios**:

1. **Given** the planning tab is focused and a task-linked entry for an incomplete task is selected, **When** the user triggers the complete action, **Then** the linked task is marked complete and the entry is shown as completed.
2. **Given** a task-linked entry for an already-completed task is selected, **When** the user triggers the complete action, **Then** the linked task is un-completed (the action toggles).
3. **Given** a task was completed from the planning tab, **When** the user views that task in the Tasks tab, **Then** it appears completed there too (the change is persisted, not view-local).
4. **Given** the same task is linked by more than one entry on the day (e.g., a timed and an untimed entry), **When** the user completes it from one entry, **Then** every entry linked to that task reflects the completed state.

---

### User Story 2 - Start a pomodoro for a linked task from the plan (Priority: P2)

While reviewing the day's plan, the user decides to start working on a planned task and starts a pomodoro for it directly from the planning tab, without first switching to the Tasks tab to find it.

**Why this priority**: Starting focused work from the plan is the natural next step after completing items, and closes the loop between planning and doing. It is high-value but secondary to completion, which is the more frequent action.

**Independent Test**: With a plan containing a task-linked entry, select it and trigger "start pomodoro". Verify a pomodoro starts for that linked task, identical to starting it from the Tasks tab.

**Acceptance Scenarios**:

1. **Given** the planning tab is focused and a task-linked entry is selected, **When** the user triggers the start-pomodoro action, **Then** a pomodoro begins for the linked task and the running-pomodoro indicator reflects it.
2. **Given** a pomodoro is already running, **When** the user starts a pomodoro for a linked task from the planning tab, **Then** the behavior matches starting a pomodoro from the Tasks tab while one is running (no new, divergent rule is introduced).
3. **Given** a pomodoro started from the planning tab is running, **When** the user switches to the Tasks tab, **Then** the same pomodoro is shown as running there.

---

### Edge Cases

- **Selected entry is an event (no linked task)**: The complete and start-pomodoro actions do not apply. The user receives clear feedback that the action is unavailable for an event, and no task state changes.
- **No entry is selected**: Triggering either action does nothing and (where appropriate) gives feedback; nothing changes.
- **Starting a pomodoro for a task whose entry is already marked complete**: Allowed; behavior matches the Tasks tab (the feature introduces no new restriction here).
- **Action fails server-side** (e.g., network/permission error): The user is informed and the displayed state does not falsely show success.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Users MUST be able to toggle the completion state of the task linked to the selected plan entry, from within the planning tab, without navigating to the Tasks tab.
- **FR-002**: The complete action MUST be a toggle: completing an incomplete linked task marks it complete, and triggering it again un-completes it. The action is optimized for the common case of marking a task complete.
- **FR-003**: A completion change made from the planning tab MUST be persisted to the underlying task so it is reflected consistently everywhere the task appears (the planning tab, the Tasks tab, and other plan entries linked to the same task).
- **FR-004**: Users MUST be able to start a pomodoro for the task linked to the selected plan entry, from within the planning tab, without navigating to the Tasks tab.
- **FR-005**: A pomodoro started from the planning tab MUST behave identically to one started from the Tasks tab, including how it interacts with any already-running pomodoro and how its running state is displayed across tabs.
- **FR-006**: Both actions MUST be invoked using the same gestures/commands the Tasks tab already uses for "toggle complete" and "start pomodoro", so users do not learn new controls for the same operations.
- **FR-007**: When the selected plan entry is an event with no linked task, both actions MUST be inert (no task is created or modified) and the user MUST receive feedback that the action is not available for that entry.
- **FR-008**: When no entry is selected, triggering either action MUST do nothing and leave all state unchanged.
- **FR-009**: The completed state of a task-linked entry MUST remain visible in the planning tab after completion (it is marked completed in place rather than removed from the day's plan).
- **FR-010**: User-facing feedback for these actions (success, unavailable-for-event, and error) MUST be presented in the application's established warm/playful tone and surfaced through the existing status/notice channel.

### Key Entities *(include if feature involves data)*

- **Plan entry**: An item on a given day's plan. May be **task-linked** (carries a reference to a task and mirrors that task's completed state) or an **event** (no linked task). The actions in this feature apply only to task-linked entries.
- **Task**: The unit of work that a task-linked entry refers to. Has a completion state that this feature toggles. Completion is a property of the task, shared by every entry linked to it.
- **Pomodoro**: A focus timer associated with a task. This feature lets one be started for the task behind a selected entry; its lifecycle and single-active-timer rules are unchanged from the Tasks tab.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From the planning tab, a user can mark a linked task complete with a single action, without switching tabs or opening any additional form.
- **SC-002**: From the planning tab, a user can start a pomodoro for a linked task with a single action, without switching tabs.
- **SC-003**: The result of completing a task or starting a pomodoro from the planning tab is indistinguishable from performing the same action in the Tasks tab (same persisted state, same running-timer behavior) in 100% of cases.
- **SC-004**: A completion made in the planning tab is visible as completed in the Tasks tab in the same session with no separate manual reconciliation step required by the user.
- **SC-005**: When a user triggers either action on an event entry or with no entry selected, no task or timer state changes, and the user understands why from the feedback shown.

## Assumptions

- **Scope is the interactive TUI planning tab.** These actions are added where the user already views the plan interactively. The web frontend (`twig-web`, which consumes only the task service) is out of scope, as is adding equivalent non-interactive CLI subcommands (the CLI can already complete tasks and start pomodoros directly).
- **Same controls as the Tasks tab.** The complete action reuses the Tasks-tab "toggle complete" gesture and the start-pomodoro action reuses the Tasks-tab "start pomodoro" gesture, rather than introducing new keys, satisfying UI/UX consistency.
- **Completion is the task's state, not a plan-entry-local flag.** A task-linked entry already mirrors its task's completion; this feature toggles the task, and the entry reflects it. No new per-entry completion concept is introduced.
- **Pomodoro semantics are inherited, not redefined.** Single-active-pomodoro behavior, lifecycle hooks, and running-state display are exactly as they already work when starting a pomodoro from the Tasks tab; this feature only adds a new place to trigger the start.
- **Completed entries stay on the plan.** Consistent with how the plan already displays completed task-linked entries, completing from the plan marks the entry completed in place rather than hiding or removing it.
- **No proto or database schema change is anticipated.** The underlying complete-task and start-pomodoro operations already exist; this feature wires the planning tab to them.
