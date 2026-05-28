# Feature Specification: TUI Pomodoro Integration

**Feature Branch**: `019-tui-pomodoro-integration`

**Created**: 2026-05-28

**Status**: Draft

**Input**: User description: "Integrate the Pomodoro timer into the interactive TUI so it runs as a background fact alongside normal task navigation/editing, instead of suspending the TUI with a full-screen blocking countdown."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run a pomodoro without leaving the task view (Priority: P1)

A user browsing their tasks in the interactive TUI selects a task and starts a pomodoro. Instead of being dropped into a separate full-screen countdown, a live timer appears in the status bar and counts down second by second. The user continues navigating the tree, editing tasks, adding subtasks, and moving items — all while the timer keeps ticking and stays visible. If they change their mind, they cancel the pomodoro and the timer disappears.

**Why this priority**: This is the heart of the request. Today, starting a pomodoro hijacks the entire screen and makes the task list unusable until the timer is exited. Restoring the ability to work while a pomodoro runs is the single change that delivers the most value, and it is a viable improvement on its own even before completion notifications or cross-session resume are added.

**Independent Test**: Start a pomodoro on a selected task; confirm the timer renders and decrements in the status bar while the task list remains fully interactive (navigate, expand/collapse, edit, create); cancel and confirm the timer clears. Fully exercisable with an injected clock and a stub task service.

**Acceptance Scenarios**:

1. **Given** the task list is shown and a task is selected, **When** the user presses the start-pomodoro key, **Then** a countdown timer for that task appears in the status bar and begins decreasing once per second.
2. **Given** a pomodoro is running, **When** the user navigates the tree, expands/collapses, edits, creates, or moves tasks, **Then** all of those actions work normally and the timer continues to count down and remain visible.
3. **Given** a pomodoro is running, **When** the user presses the cancel-pomodoro key, **Then** the pomodoro ends, the configured cancel hook runs, and the timer is removed from the status bar.
4. **Given** a pomodoro is running on the selected task, **When** the user presses the start key again on that same task, **Then** the existing pomodoro continues uninterrupted (no restart, no error).
5. **Given** a pomodoro is running on one task, **When** the user starts a pomodoro on a different task, **Then** the first pomodoro ends and a new one begins on the newly selected task.

---

### User Story 2 - Automatic completion with notification (Priority: P2)

While the user works in the TUI, a running pomodoro reaches zero. The system records the pomodoro as completed, runs the user's completion hook (for example a desktop notification or sound), and shows a brief, non-blocking "Pomodoro complete" message in the status bar. The message clears on the next key press or after a few seconds. The user is never interrupted by a blocking dialog and never has to manually finalize the timer.

**Why this priority**: Automatic, reliable completion is what makes the integrated timer trustworthy. It directly fixes the existing defect where an unattended pomodoro is never completed and its completion hook never fires. It depends on the timer existing in the TUI (Story 1), so it is second.

**Independent Test**: With an injected clock advanced past the pomodoro length while the TUI is open, confirm the pomodoro is marked complete, the completion hook is invoked once, and a transient banner appears and then clears on key press or timeout.

**Acceptance Scenarios**:

1. **Given** a pomodoro is running and the TUI is open, **When** the remaining time reaches zero, **Then** the pomodoro is recorded as completed and the completion hook runs exactly once.
2. **Given** a pomodoro has just completed, **When** completion is recorded, **Then** a non-blocking "Pomodoro complete" message naming the task appears in the status bar without interrupting the user's current activity.
3. **Given** the completion message is showing, **When** the user presses any key or a few seconds elapse, **Then** the message clears and the status bar returns to its normal single-line state.
4. **Given** a completion hook is configured but fails to run, **When** completion occurs, **Then** the pomodoro is still recorded complete and the failure is surfaced as an ordinary status-bar error rather than corrupting the display.

---

### User Story 3 - Pick up an already-running pomodoro at launch (Priority: P2)

A user starts a pomodoro from the command line (or in a previous TUI session) and then opens the TUI. The TUI detects the active pomodoro and immediately shows it ticking in the status bar, with the correct remaining time, so the two surfaces stay in sync.

**Why this priority**: Pomodoro state is shared between the CLI and the TUI. Without auto-attach, the integrated timer would appear broken whenever a pomodoro was started elsewhere. It is valuable but secondary to starting and completing pomodoros from within the TUI.

**Independent Test**: With a stub service reporting an active pomodoro and a known start time, launch the TUI and confirm the timer appears immediately with the correct derived remaining time and the correct task name; with no active pomodoro, confirm the status bar starts in its normal state.

**Acceptance Scenarios**:

1. **Given** a pomodoro is active on the server, **When** the TUI launches, **Then** the timer appears in the status bar showing the correct remaining time for the correct task without any manual action.
2. **Given** no pomodoro is active on the server, **When** the TUI launches, **Then** no timer is shown and the status bar is in its normal state.

---

### User Story 4 - Quit safely with a pomodoro running (Priority: P3)

A user with a running pomodoro tries to quit the TUI. Rather than silently quitting, the TUI asks for confirmation, noting that the pomodoro is still running. If the user confirms, the TUI exits and the pomodoro keeps running so it can be picked up again later; if they decline, they stay in the TUI with the timer intact.

**Why this priority**: A guardrail that prevents accidentally walking away from a focus session. Useful polish, but the feature is usable without it, so it is lowest priority.

**Independent Test**: With a running (not-yet-completed) pomodoro, trigger quit and confirm a one-key confirmation prompt appears; confirm "yes" exits while "no"/escape returns to the list with the timer still running. With no pomodoro, confirm quit happens immediately.

**Acceptance Scenarios**:

1. **Given** a pomodoro is running, **When** the user presses the quit key, **Then** a confirmation prompt appears stating the pomodoro is still running and offering yes/no.
2. **Given** the quit confirmation is showing, **When** the user confirms, **Then** the TUI exits and the pomodoro remains active so it can be resumed later.
3. **Given** the quit confirmation is showing, **When** the user declines or cancels, **Then** the prompt is dismissed and the timer continues unchanged.
4. **Given** no pomodoro is running (or it has already completed), **When** the user presses the quit key, **Then** the TUI exits immediately with no prompt.

---

### Edge Cases

- **Start while another pomodoro is active**: starting on the same task leaves the running one alone; starting on a different task ends the old one and begins the new one. No interactive text prompt is used (the TUI cannot read line input mid-session).
- **Hook output and failures**: lifecycle hooks must not write to the terminal the TUI controls; their output must not corrupt the screen, a slow hook must not freeze the interface, and a failing hook must be reported without aborting the pomodoro transition.
- **Stale timer events**: a pending tick that arrives after the pomodoro has already been cancelled or completed must be ignored and must not restart a timer.
- **Plain (unstyled) terminals**: the timer and messages must render legibly without relying on color or styling when the output is not a styled terminal.
- **Server/network failure on start, cancel, or completion**: the failure is surfaced as a status-bar error and the TUI remains usable rather than crashing.
- **Quitting before completion vs. after completion**: the quit guard applies only while a pomodoro is actively counting down; once it has completed it must not block quitting.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The TUI MUST allow the user to start a pomodoro on the selected task and MUST display a live countdown for the active pomodoro without leaving or obscuring the task view.
- **FR-002**: The running timer MUST update at least once per second and MUST remain visible in the task-list, edit/new-form, and move views. The transient full-screen help overlay is exempt: it covers the whole screen and is dismissed by any key, so the timer need not be shown there.
- **FR-003**: While a pomodoro is running, all existing task interactions (navigate, expand/collapse, edit, create, delete, complete, move, set estimate, refresh, filter, help) MUST remain fully functional.
- **FR-004**: The timer MUST be shown in the bottom status area; while a pomodoro is active the status area MAY occupy an additional line, and the surrounding content MUST adjust so nothing is overlapped or pushed off-screen.
- **FR-005**: Remaining time MUST be derived from the pomodoro's authoritative start time and fixed length, so the displayed countdown stays consistent with the shared pomodoro state.
- **FR-006**: When the remaining time reaches zero while the TUI is open, the system MUST record the pomodoro as completed and MUST run the configured completion hook exactly once.
- **FR-007**: On completion the TUI MUST show a brief, non-blocking message identifying the completed task, and that message MUST clear on the next key press or after a short timeout. A key press that clears the message MUST still perform its normal action (clearing the message is a side effect, not a substitute for the keystroke).
- **FR-008**: The user MUST be able to cancel the active pomodoro from the TUI, which MUST run the configured cancel hook and remove the timer.
- **FR-009**: On launch the TUI MUST detect an already-active pomodoro and display it immediately with the correct remaining time and task, with no manual resume step.
- **FR-010**: When the user starts a pomodoro while one is already active, the system MUST, without prompting for typed input: leave the existing one running if it is the same task, or end it and start a new one if it is a different task.
- **FR-011**: When the user attempts to quit while a pomodoro is actively counting down, the TUI MUST request confirmation; confirming MUST exit while leaving the pomodoro running, and declining MUST return to the interface with the pomodoro intact.
- **FR-012**: Quitting with no active pomodoro, or after the active pomodoro has completed, MUST NOT prompt for confirmation.
- **FR-013**: Lifecycle hooks (start, cancel, complete) triggered from the TUI MUST run without writing to the terminal the TUI controls and MUST NOT block the interface; hook failures MUST be surfaced as ordinary status-area errors instead of raw terminal output.
- **FR-014**: A timer event that arrives after its pomodoro has been cancelled or completed MUST be ignored and MUST NOT cause a timer to resume.
- **FR-015**: Failures from start, cancel, or completion operations MUST be surfaced as status-area errors and MUST leave the TUI usable.
- **FR-016**: The command-line `todo pom` subcommands MUST retain their current behavior and hook handling; this feature changes only the in-TUI experience.
- **FR-017**: The manual resume action MUST be removed from the TUI, since active pomodoros are attached automatically on launch and continue running throughout a session.

### Key Entities *(include if feature involves data)*

- **Active pomodoro (TUI view of it)**: the currently running pomodoro as the TUI tracks it — the associated task, the task's display name, its authoritative start time, and whether it has just completed (so the transient completion message can be shown). This is a view onto the shared, server-authoritative pomodoro, not a new store of timer state.
- **Lifecycle hooks**: the user-configured start, cancel, and complete commands that run at the corresponding pomodoro transitions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Starting a pomodoro in the TUI never removes access to the task list — the user can perform every normal task action while a pomodoro runs (0 cases of being dropped into a separate full-screen view).
- **SC-002**: In 100% of sessions where the TUI remains open through a pomodoro's end, the pomodoro is recorded complete and its completion hook fires within 1 second of the timer reaching zero, with no manual action by the user.
- **SC-003**: The displayed remaining time matches the authoritative pomodoro state within 1 second, including immediately after launching the TUI onto an already-running pomodoro.
- **SC-004**: No lifecycle hook (including a slow or failing one) ever corrupts the display or freezes the interface; the interface remains responsive to input throughout hook execution.
- **SC-005**: A user can start, observe, complete, and cancel pomodoros entirely from within the TUI without ever invoking the separate command-line pomodoro flow.

## Assumptions

- The pomodoro length stays fixed at its current value; this feature does not introduce configurable durations or breaks.
- The server remains the single source of truth for pomodoro state (start time and active task); the TUI derives and displays remaining time but does not independently persist timer state, and no server-side changes are required.
- Completion hooks fire only while a client is attached and observing the timer; if the user quits before completion, the pomodoro keeps running and its completion is finalized when a client (TUI or CLI) next attaches and the timer is found expired. Server-side autonomous completion is out of scope.
- The start key, cancel key, and quit key bindings reuse the TUI's existing keybinding conventions; the cancel action takes over the binding previously used for manual resume.
- Cross-client divergence (for example, another client cancels the pomodoro while this TUI is open) is reconciled opportunistically on the next list refresh rather than by continuous polling.
- Lifecycle hook configuration continues to use the existing config-file mechanism; this feature does not change how hooks are defined.
