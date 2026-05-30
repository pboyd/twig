# Feature Specification: Task Web App

**Feature Branch**: `023-task-web-app`

**Created**: 2026-05-30

**Status**: Draft

**Input**: User description: "Task web app — primarily targeting mobile browsers for capturing and managing tasks while away from the terminal. A user should be able to log in with username and password, see the tree of existing tasks, view task details, add a task or sub-task, modify an existing task, and mark a task complete."

## Clarifications

### Session 2026-05-30

- Q: Should marking a task complete be reversible from the web app? → A: Completion is a toggle — a user can mark complete and reopen (un-complete) a task.
- Q: What happens when a user completes a parent task whose sub-tasks are still incomplete? → A: Blocked — a parent cannot be completed until all its sub-tasks are complete (the server already enforces this; the web app surfaces it).
- Q: How long should a sign-in last? → A: Reuse the existing browser session from feature [003-user-auth](../003-user-auth/spec.md) — a long-lived, persistent session (configurable lifetime, default 30 days) that survives reloads and revisits until the user signs out or the session expires. No new session mechanism is introduced.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Capture a task on the go (Priority: P1)

A user is away from their computer and thinks of something that needs to get done. They open the web app on their phone, sign in, and add a new task (optionally as a sub-task under an existing one) so the idea is safely captured before they forget it.

**Why this priority**: This is the core motivation for the web app. The terminal remains the primary interface, so the single most valuable thing the web version adds is the ability to capture tasks quickly from a phone. Login is a prerequisite for any task action, so it is bundled into this first journey.

**Independent Test**: Sign in with valid credentials on a mobile-sized screen, create a new top-level task, then create a sub-task under it. Confirm both appear in the user's task list. Delivers value on its own: a user can capture work even with no other features present.

**Acceptance Scenarios**:

1. **Given** a user with valid credentials, **When** they enter their username and password, **Then** they are signed in and shown their tasks.
2. **Given** a signed-in user, **When** they add a new top-level task with a title, **Then** the task is saved and appears in their task tree.
3. **Given** a signed-in user viewing an existing task, **When** they add a sub-task under it, **Then** the sub-task is saved and appears nested beneath its parent.
4. **Given** a user who enters invalid credentials, **When** they attempt to sign in, **Then** they are told the credentials are incorrect and are not signed in.

---

### User Story 2 - Review existing tasks (Priority: P2)

A signed-in user wants to see what is already on their plate. They browse the tree of existing tasks, expand and collapse branches, and open a task to read its full details.

**Why this priority**: Reviewing tasks is highly valuable but secondary to capture — a user can add tasks (P1) without first browsing. Viewing builds confidence that captured items landed correctly and lets the user orient before editing.

**Independent Test**: With a set of existing tasks (including nested sub-tasks), open the app and confirm the hierarchy renders correctly, branches can be expanded/collapsed, and tapping a task shows its details (title, description, status, and any sub-tasks).

**Acceptance Scenarios**:

1. **Given** a signed-in user with existing tasks, **When** they open the app, **Then** their tasks are displayed as a tree reflecting parent/child relationships.
2. **Given** a task tree with nested sub-tasks, **When** the user expands or collapses a branch, **Then** the visible tasks update accordingly.
3. **Given** a signed-in user, **When** they select a task, **Then** they see its details including title, description, completion status, and sub-tasks.
4. **Given** a user with no tasks, **When** they open the app, **Then** they see a clear empty state inviting them to add their first task.

---

### User Story 3 - Update and complete tasks (Priority: P3)

A signed-in user reviews a task and needs to change it — fix a typo in the title, add detail, or mark it done.

**Why this priority**: Editing and completion round out task management but depend on tasks existing (P1) and being viewable (P2). They are valuable but the lowest-priority slice for an initial release focused on capture.

**Independent Test**: Open an existing task, change its title and description, save, and confirm the changes persist. Separately, mark a task complete and confirm its status updates and is reflected in the tree.

**Acceptance Scenarios**:

1. **Given** a signed-in user viewing a task, **When** they edit the title or description and save, **Then** the updated values are persisted and shown.
2. **Given** a signed-in user viewing an incomplete task, **When** they mark it complete, **Then** its status changes to complete and the change is reflected in the task tree.
3. **Given** a completed task, **When** the user views it, **Then** its completed state is clearly indicated.
4. **Given** a user editing a task, **When** they discard their changes before saving, **Then** the task retains its original values.

---

### Edge Cases

- What happens when the user's session expires or their credentials are revoked mid-use? They should be returned to the sign-in screen with their unsaved input preserved where feasible.
- How does the system handle submitting a task with an empty title? Submission is blocked with a clear validation message.
- How does the system behave when connectivity drops while saving? The user is informed the save did not complete and can retry; data is not silently lost.
- What happens when two clients (e.g., the terminal and the web app) modify the same task? The web app reflects the latest server state on next load; last write is accepted (no merge-conflict UI in this round).
- How are deeply nested or very large task trees presented on a small screen without becoming unusable?
- What happens when a user tries to act on a task that was deleted elsewhere? They are informed the task no longer exists.
- What happens when a user tries to complete a parent task with incomplete sub-tasks? The action is rejected and the user is told the sub-tasks must be completed first.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow a user to sign in with a username and password using the same accounts as the existing terminal application.
- **FR-002**: System MUST reject invalid credentials with a clear, non-revealing error message and MUST NOT grant access.
- **FR-003**: System MUST keep a user signed in using the existing browser session mechanism from feature 003-user-auth — a persistent session (configurable lifetime, default 30 days) that survives page reloads and revisits until the user signs out or the session expires — and MUST provide a way to sign out.
- **FR-004**: System MUST restrict all task data to the authenticated user; a user MUST only see and modify their own tasks.
- **FR-005**: System MUST display the user's tasks as a hierarchical tree reflecting parent/child relationships.
- **FR-006**: Users MUST be able to expand and collapse branches of the task tree.
- **FR-007**: Users MUST be able to view the details of a selected task, including its title, description, completion status, and sub-tasks.
- **FR-008**: Users MUST be able to add a new top-level task by providing at least a title.
- **FR-009**: Users MUST be able to add a sub-task under an existing task.
- **FR-010**: System MUST prevent creation or saving of a task with an empty title and explain why.
- **FR-011**: Users MUST be able to modify an existing task's editable fields (at minimum title and description) and persist the changes.
- **FR-012**: Users MUST be able to toggle a task's completion state — marking an incomplete task complete and reopening (un-completing) a completed task. The completed state MUST be clearly indicated in both the detail view and the task tree.
- **FR-017**: System MUST prevent completing a parent task while any of its sub-tasks remain incomplete, and MUST communicate this constraint to the user (the rule is enforced by the existing backend).
- **FR-013**: System MUST reflect changes made from other clients (e.g., the terminal app) when the user next loads or refreshes their tasks.
- **FR-014**: System MUST present a usable layout on mobile browser screen sizes as the primary target.
- **FR-015**: System MUST inform the user when an action fails (e.g., lost connectivity, expired session) rather than failing silently, and allow them to retry.
- **FR-016**: System MUST show a clear empty state when the user has no tasks.

### Key Entities *(include if feature involves data)*

- **User**: An individual with credentials (username/password) shared with the existing terminal application. Owns a private set of tasks. Authenticated state is represented by a session.
- **Task**: A unit of work owned by a user. Key attributes: title (required), description (optional), completion status, and an optional parent task forming a hierarchy. A task may have zero or more sub-tasks.
- **Session**: Represents an authenticated user's active sign-in on the web app; governs access to task operations and can expire or be ended via sign-out.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A returning, signed-in user can capture a new task from their phone in under 20 seconds from opening the app.
- **SC-002**: A first-time user can sign in and add their first task within 1 minute without external help.
- **SC-003**: 95% of task create, edit, and complete actions succeed on the first attempt under normal connectivity.
- **SC-004**: The task tree, detail view, add, edit, and complete flows are all fully operable on a typical mobile phone screen without horizontal scrolling or pinch-zoom.
- **SC-005**: Changes made in the web app are visible in the terminal application (and vice versa) after a refresh, confirming a single shared source of task data.
- **SC-006**: 90% of users can locate and view the details of a specific existing task within their first session without instruction.

## Assumptions

- **Shared backend and accounts**: The web app uses the existing task service and the same user accounts/credentials as the terminal application; no separate user store or sign-up flow is introduced in this round. Authentication reuses the browser session model defined in feature 003-user-auth (secure, script-inaccessible session cookie, CSRF-protected, configurable lifetime); this feature does not change that mechanism.
- **Scope is task management only**: Pomodoro tracking, daily planning, and other terminal-only features are explicitly out of scope for the web app in this round.
- **Online-only**: The app requires network connectivity; offline capture and local queuing are out of scope for this round. The motivating "capture before I forget" need is met by a phone with normal connectivity, not offline storage.
- **Mobile-first, not mobile-only**: Mobile browsers are the primary target; the app should remain usable on a desktop browser, but desktop polish is not a priority.
- **Task fields**: Editable fields in this round are title and description; completion status is toggled via the complete action. Other attributes that may exist in the terminal app (e.g., scheduling/planning metadata) are not edited here.
- **Concurrency**: Simultaneous edits from multiple clients use last-write-wins; no conflict-resolution UI is provided this round.
- **Deletion out of scope**: Deleting tasks is not part of this round (the listed capabilities are view, add, modify, and complete).
- **Single-user perspective**: No sharing, collaboration, or multi-user task visibility is in scope.
