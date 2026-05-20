# Feature Specification: Pomodoro Tracking

**Feature Branch**: `005-pomodoro-tracking`

**Created**: 2026-05-20

**Status**: Draft

**Input**: User description: "Add pomodoro tracking to tasks. A pomodoro is a 25-minute session of focused work."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run a focused work session on a task (Priority: P1)

A user picks a task they want to work on and starts a pomodoro. They see a live countdown showing the task name, their estimate, and how many pomodoros they have already completed for the task. When the 25 minutes elapse, the session is recorded as completed and contributes to the task's pomodoro history.

**Why this priority**: This is the core value of the feature. Without the ability to start and finish a focused session against a task, none of the surrounding functionality is useful.

**Independent Test**: Start a pomodoro for an existing task, observe the countdown UI showing task name, estimate, and previously completed pomodoros, let the timer reach zero, and verify a completed pomodoro is recorded against that task.

**Acceptance Scenarios**:

1. **Given** an existing task and no active pomodoro, **When** the user starts a pomodoro for that task, **Then** a new pomodoro record is created with the current start time and a `null` end time, and the countdown UI displays the task name, the task's estimate, and the count of previously completed pomodoros for that task.
2. **Given** an active pomodoro running in the foreground, **When** 25 minutes elapse, **Then** the pomodoro is marked complete with an end time equal to start + 25 minutes and the `complete` flag set to true.
3. **Given** an active pomodoro and `--exec` was provided when starting, **When** the timer finishes, **Then** the provided command is executed after the pomodoro is marked complete.

---

### User Story 2 - Cancel, quit, and resume sessions (Priority: P2)

A user is in the middle of a pomodoro and needs to step away or stop. They can either cancel the session (which records an incomplete pomodoro ending now) or quit the countdown UI without stopping the underlying session, and later resume the countdown for the still-active pomodoro on the same task. If they try to start a new pomodoro on a task that is already in progress, they are asked whether to restart it or resume.

**Why this priority**: Real workflows are interrupted. Supporting cancel, quit, and resume keeps the data model honest (canceled sessions don't count as completions) and lets users move between terminals or tasks without losing state.

**Independent Test**: Start a pomodoro, quit the countdown UI, run resume on the same task, and confirm the countdown picks up from the correct remaining time; separately, start a pomodoro, press `c` to cancel, and confirm an incomplete pomodoro record is saved with end time = now.

**Acceptance Scenarios**:

1. **Given** an active pomodoro displayed in the countdown UI, **When** the user presses `c`, **Then** the active pomodoro's end time is set to the current time, the `complete` flag remains false, and the UI exits.
2. **Given** an active pomodoro displayed in the countdown UI, **When** the user presses `q`, **Then** the UI exits but the pomodoro record remains active (end time still `null`).
3. **Given** a pomodoro is already active for task T, **When** the user runs resume for task T, **Then** the countdown displays the time remaining until start + 25 minutes.
4. **Given** a pomodoro was started more than 25 minutes ago and not yet ended, **When** the user runs resume for that task, **Then** the pomodoro is immediately marked complete and the UI exits.
5. **Given** a pomodoro is already active for task T, **When** the user runs start for task T, **Then** the user is asked whether to restart (cancel the current one and start a new one) or resume.
6. **Given** a pomodoro is active for task A, **When** the user runs start for task B, **Then** the user is asked whether to cancel task A's pomodoro; canceling it then starts a new pomodoro for task B.
7. **Given** an active pomodoro is being displayed in a countdown UI, **When** that pomodoro is canceled externally (e.g., from another terminal), **Then** the countdown UI detects the cancellation within a short polling interval and exits.
8. **Given** a task has no active or prior pomodoro, **When** the user runs resume for that task, **Then** an error is returned indicating the task was never started.

---

### User Story 3 - Estimate pomodoros for a task (Priority: P2)

A user records how many pomodoros they think a task will take, so they can later compare estimate vs. actual and decide whether a task needs to be broken down.

**Why this priority**: Estimates are required input for the countdown display in Story 1 and underpin the "break it down" guidance, but the system is still useful without them (estimate defaults to 0 / unset).

**Independent Test**: Set an estimate of 3 on an existing task, then read the task back and confirm the estimate field is 3; attempt to set an estimate of 11 and confirm it is rejected.

**Acceptance Scenarios**:

1. **Given** an existing task with no estimate, **When** the user sets the estimate to an integer between 1 and 10, **Then** the task's `estimate` field is updated to that value, replacing any prior value.
2. **Given** a task with an existing estimate, **When** the user sets a new estimate, **Then** the new value overwrites the old one.
3. **Given** any task, **When** the user attempts to set an estimate outside the 0–10 range, **Then** the operation is rejected with an error indicating estimates above 10 require the task to be broken down further.
4. **Given** a task that has never had an estimate set, **When** the task is read, **Then** its `estimate` is reported as 0.

---

### User Story 4 - Inspect current pomodoro status (Priority: P3)

A user wants to quickly see whether they have a pomodoro running and, if so, what task it belongs to and how much time is left.

**Why this priority**: Convenience and discoverability. It is not required to use the core feature but reduces friction, especially when a user has quit the countdown UI.

**Independent Test**: With no active pomodoro, run status and confirm the output says none is active; start a pomodoro, run status, and confirm the active task and remaining time are shown.

**Acceptance Scenarios**:

1. **Given** no active pomodoro, **When** the user runs status, **Then** the output indicates that no pomodoro is currently active.
2. **Given** an active pomodoro, **When** the user runs status, **Then** the output includes the task name/ID, the start time, and the time remaining until start + 25 minutes.

---

### Edge Cases

- The user attempts to start a pomodoro on a task that does not exist → the operation fails with a not-found error and no pomodoro record is created.
- The user attempts to cancel a pomodoro for a task that has no active pomodoro → the operation fails with an error; existing completed/canceled records are not modified.
- The system clock moves backwards between start and end (e.g., NTP correction) → end time is still recorded as the current time at cancel/complete; the duration may be shorter than expected but the record is preserved.
- A pomodoro record's start time is more than 25 minutes in the past and it is still active → on resume it is immediately completed; on status it is reported as still active until the next state-changing operation, since the API does not run timers itself.
- Two clients race to start a pomodoro for the same user → only one succeeds; the other receives an error indicating an active pomodoro already exists.
- The user provides `--exec` with a command that fails → the pomodoro is still recorded as complete; the command's failure is surfaced to the user but does not change the pomodoro record.
- The user presses `c` after the timer has already reached zero → the pomodoro is treated as complete (not canceled).

## Requirements *(mandatory)*

### Functional Requirements

**Task estimate**

- **FR-001**: A task MUST carry an integer `estimate` field representing the user's estimated number of pomodoros to complete it.
- **FR-002**: The `estimate` field MUST accept integer values from 0 through 10 inclusive; values outside this range MUST be rejected.
- **FR-003**: The default value of `estimate` for a task MUST be 0, which represents "no estimate".
- **FR-004**: Setting an estimate on a task MUST overwrite any prior estimate.
- **FR-005**: When a user attempts to set an estimate greater than 10, the system MUST reject the request and communicate that tasks larger than 10 pomodoros must be broken down further.

**Pomodoro records**

- **FR-010**: A pomodoro record MUST contain a `task_id`, `start` timestamp, `end` timestamp (nullable), and a boolean `complete` flag.
- **FR-011**: A pomodoro is considered "active" when its `end` timestamp is `null`.
- **FR-012**: For a completed pomodoro, `end` MUST equal `start` + 25 minutes and `complete` MUST be true.
- **FR-013**: For a canceled pomodoro, `end` MUST be the time at which cancel was requested (which will be less than 25 minutes after `start`) and `complete` MUST be false.
- **FR-014**: Each user MUST have at most one active pomodoro at any time; attempts to create a second active pomodoro for the same user MUST be rejected.
- **FR-015**: The system MUST NOT run any internal timer to advance pomodoros from active to complete; state transitions only occur via explicit API operations.

**API operations**

- **FR-020**: The system MUST provide a Start operation that creates a new pomodoro for a given `task_id`, with `start` set to the current time, `end` set to `null`, and `complete` set to false; this operation MUST fail if the user already has an active pomodoro or if the task does not exist.
- **FR-021**: The system MUST provide a Cancel operation that sets `end` on the user's active pomodoro to the current time and leaves `complete` false; this operation MUST fail if there is no active pomodoro.
- **FR-022**: The system MUST provide a Complete operation that sets `end` on the user's active pomodoro to the current time and sets `complete` to true; this operation MUST fail if there is no active pomodoro.
- **FR-023**: The system MUST expose, for a given task, the number of pomodoros previously completed against it.
- **FR-024**: The system MUST expose the current user's active pomodoro (if any), including `task_id`, `start`, and enough information to compute the remaining time until `start` + 25 minutes.

**CLI behavior**

- **FR-030**: The CLI MUST provide an estimate command that takes a task ID and an integer `n`, validates `n` against FR-002, and persists it via the API.
- **FR-031**: The CLI MUST provide a start command that begins a new pomodoro for a given task, displaying a live countdown UI showing the task name, the task's `estimate`, and the count of previously completed pomodoros for that task.
- **FR-032**: While the countdown is displayed, pressing `c` MUST cancel the pomodoro (via the Cancel operation) and exit the UI.
- **FR-033**: While the countdown is displayed, pressing `q` MUST exit the UI without changing the underlying pomodoro record.
- **FR-034**: When the countdown reaches zero, the CLI MUST invoke the Complete operation; if `--exec <cmd>` was provided, the CLI MUST then execute that command.
- **FR-035**: When start is invoked for a task that already has an active pomodoro, the CLI MUST prompt the user to either restart (cancel the existing one and start anew) or resume (behave equivalently to the resume command).
- **FR-036**: When start is invoked while the user has an active pomodoro on a different task, the CLI MUST prompt the user to cancel the other pomodoro (behaving equivalently to the cancel command) before starting the new one.
- **FR-037**: The CLI MUST provide a resume command that attaches a countdown UI to the user's existing active pomodoro for the given task; it MUST fail with an error if no pomodoro has ever been started for that task.
- **FR-038**: If, at resume time, the active pomodoro's `start` is more than 25 minutes in the past, the CLI MUST immediately mark it complete (via Complete) and exit.
- **FR-039**: While a countdown UI is displayed, the CLI MUST detect external cancellation/completion of the underlying pomodoro (e.g., from another terminal) within a short polling interval and exit gracefully.
- **FR-040**: The CLI MUST provide a cancel command that cancels the user's active pomodoro on the given task.
- **FR-041**: The CLI MUST provide a status command that prints information about the active pomodoro (task name/ID, start time, time remaining) or reports that none is active.

### Key Entities *(include if feature involves data)*

- **Task**: Existing task record, extended with an `estimate` integer (0–10) representing the user's predicted number of pomodoros to complete it.
- **Pomodoro**: A user's focused work session against a specific task. Attributes: `task_id` (link to Task), `start` (timestamp), `end` (timestamp, nullable while active), `complete` (boolean, true only when the full 25 minutes elapsed). Implicitly owned by the user who created it. At most one pomodoro per user may be active (`end == null`) at any time.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can start, run to completion, and have a pomodoro recorded against a task without any manual data correction in 100% of normal-path runs.
- **SC-002**: Starting, canceling, completing, and querying status of a pomodoro each return a result in under 1 second under normal usage.
- **SC-003**: For any user, at any moment in time, the system contains zero or one pomodoro records with `end == null` (the single-active-pomodoro invariant holds for 100% of recorded states).
- **SC-004**: After quitting (`q`) the countdown UI and later resuming on the same task, the resumed countdown's displayed remaining time differs from the true remaining time (start + 25 minutes − now) by less than 2 seconds.
- **SC-005**: External cancellation of the active pomodoro causes any running countdown UI to exit within 5 seconds.
- **SC-006**: 100% of attempts to set a task estimate outside the 0–10 range are rejected and communicate the "break it down" guidance.

## Assumptions

- Pomodoros are scoped per user; "active pomodoro" means the requesting user's active pomodoro, not a global one. The existing task ownership / user identification mechanism in the project is reused.
- Tasks are identified by integer IDs that already exist in the system; the estimate field is added to the existing task entity.
- The CLI talks to the same backing data store/API as other `task` subcommands; no new transport or auth mechanism is introduced.
- A "short polling interval" for detecting external cancellation is on the order of 1–5 seconds — fast enough that the UI feels responsive without overwhelming the API.
- "Time remaining" in the countdown is computed from `start + 25 minutes − now` against the API's stored start time, not from a local stopwatch, so quitting and resuming yields an accurate remaining time.
- The `--exec` command is run with the user's normal shell semantics; its success or failure does not affect whether the pomodoro is recorded as complete (it is already complete before `--exec` runs).
- Existing API conventions (error codes, response shapes, permission checks) carry over to the new pomodoro endpoints without redesign.
