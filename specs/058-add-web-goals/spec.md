# Feature Specification: Goals in the Web App

**Feature Branch**: `058-add-web-goals`

**Created**: 2026-06-22

**Status**: Draft

**Input**: User description: "Add goals to the web app. The TUI has goals, but they have not been implemented in the web app. For now, add the ability to view goals and work with status update (read, write and edit status updates)."

## Clarifications

### Session 2026-06-22

- Q: Should deleting a status update be in scope (alongside read/write/edit)? → A: Yes — the web app supports adding, editing, and deleting status updates, matching the existing full capability.
- Q: Should the web goal detail show a goal's associated tasks? → A: Yes, read-only — the goal detail lists associated tasks for context; linking/unlinking tasks is not part of this feature.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse goals in the web app (Priority: P1)

A user who tracks long-term goals in twig opens the web app and wants to see those same goals there, not just in the terminal. They navigate to a Goals area, see their goals grouped by state (the same grouping they know from the TUI), and select a goal to read its full details — name, description, due date, and current state. Completed and archived goals stay out of the way by default so the list reflects what the user is actively pursuing, but the user can reveal them when they want the full picture.

**Why this priority**: Seeing goals on the web is the foundation of the whole feature. Without a way to list and open a goal, there is nowhere to read or record status updates. This story alone delivers value: a user can finally review their goals from a browser.

**Independent Test**: Can be fully tested by opening the web app with existing goals, confirming they appear grouped by state with completed/archived hidden by default, toggling the hidden ones into view, and opening a goal to see its full details — with no status-update interaction.

**Acceptance Scenarios**:

1. **Given** the user is signed in to the web app, **When** they open the Goals area, **Then** their goals are listed grouped by state, with Committed and Incubating shown and Completed and Archived hidden by default.
2. **Given** the user has completed and/or archived goals, **When** they activate the "show hidden" control, **Then** the Completed and Archived groups appear; activating it again hides them.
3. **Given** the user has no goals, **When** they open the Goals area, **Then** a friendly empty state is shown rather than a blank screen.
4. **Given** a goal is shown in the list, **When** the user selects it, **Then** the goal's detail view shows its name, description (rendered as markdown), due date, and state.
5. **Given** a goal has a description containing markdown, **When** the user views the goal detail, **Then** the markdown is rendered (not shown as raw markup), consistent with how other text is rendered in the web app.
6. **Given** a goal has associated tasks, **When** the user views the goal detail, **Then** the associated tasks are listed for context (read-only — the user cannot link or unlink tasks here).

---

### User Story 2 - Read a goal's status updates (Priority: P1)

A user opens a goal to remind themselves where things stand. They see the most recent status update front and center, with its timestamp, rendered as markdown. They can also browse the full history of updates for that goal, newest first, reading even long entries in full.

**Why this priority**: Reading the latest status is the single most common reason to open a goal — it answers "where am I on this?" It is tied with browsing the list as core value: the user explicitly asked to read status updates.

**Independent Test**: Can be tested by opening a goal that already has one or more status updates and confirming the latest update is shown with its timestamp and rendered markdown, and that the full history is viewable newest-first with long entries fully readable.

**Acceptance Scenarios**:

1. **Given** a goal with no status updates, **When** the user views its detail, **Then** a friendly "no status updates yet" indication is shown.
2. **Given** a goal with one or more status updates, **When** the user views its detail, **Then** the most recent update is shown with its timestamp, rendered as markdown.
3. **Given** a goal with multiple status updates, **When** the user opens the status history, **Then** all updates are listed newest first, each labeled with a human-readable timestamp.
4. **Given** a status update longer than the visible area, **When** the user reads it, **Then** the full text is accessible (wraps and/or scrolls) with no silent truncation.

---

### User Story 3 - Record a new status update (Priority: P2)

A user has made progress on a goal and wants to capture it. From the goal's detail in the web app, they write a free-form status update and save it. It is stamped with the current time and immediately becomes the goal's latest status.

**Why this priority**: Writing updates is the "work with" half of the request and turns the web app from a read-only viewer into a place to journal progress. It depends on being able to view a goal (US1) but is independently valuable once viewing exists.

**Independent Test**: Can be tested by opening a goal, writing and saving a status update, and confirming it appears as the goal's latest status with the current timestamp; and that attempting to save an empty update is rejected.

**Acceptance Scenarios**:

1. **Given** a goal is open, **When** the user writes a status update and saves it, **Then** it is stored with the current date/time and immediately shown as the goal's latest status.
2. **Given** the user is writing a status update, **When** they try to save it with empty/whitespace-only text, **Then** the save is rejected with a clear message and nothing is recorded.
3. **Given** a goal already has a latest status, **When** the user records a newer update, **Then** the newer update becomes the latest status and the previous one moves into history.

---

### User Story 4 - Edit or delete an existing status update (Priority: P3)

A user spots a typo in an update, or recorded one they no longer want. From the status history they edit an update's text and save, or delete the update outright.

**Why this priority**: Correcting and removing updates is a quality-of-life refinement; the feature is already useful when updates can be read and appended. It rounds out full management parity with the existing capability.

**Independent Test**: Can be tested by editing an existing update's text and confirming the change is shown and rendered, and by deleting an update and confirming it disappears while the goal's other updates remain.

**Acceptance Scenarios**:

1. **Given** an existing status update, **When** the user edits its text and saves, **Then** the new text is shown and rendered as markdown under the update's original timestamp.
2. **Given** the user edits an update to empty/whitespace-only text, **When** they try to save, **Then** the save is rejected with a clear message and the original text is retained.
3. **Given** an existing status update, **When** the user deletes it, **Then** it no longer appears in the goal's history and the goal's other updates are unaffected.
4. **Given** a goal whose only status update is deleted, **When** the user views its detail, **Then** the detail returns to the "no status updates yet" state.

---

### Edge Cases

- A user only has completed/archived goals: the default view shows the empty state until "show hidden" is activated.
- A goal's due date is in the past: it is displayed as-is (overdue); nothing is enforced or auto-changed.
- A status update contains malformed or unusual markdown: it renders safely and legibly, consistent with other rendered text in the web app.
- A very long status update (e.g., several thousand characters): it is fully readable via wrapping/scrolling with no truncation.
- A goal selected in the detail is changed or deleted from another interface (TUI/CLI) in the meantime: the web app surfaces a clear not-found/refresh state rather than failing silently or showing stale data indefinitely.
- Network or server error while loading goals or saving an update: the user sees a clear error and can retry; no partial/ambiguous success is implied.
- Concurrent edit: two updates saved in quick succession each get their own timestamp and both appear in history.
- A goal's associated tasks include subtasks: the associated tasks are shown consistent with how association works elsewhere (a task's subtree belongs to the same goal).

## Requirements *(mandatory)*

### Functional Requirements

#### Viewing goals

- **FR-001**: The web app MUST provide a dedicated Goals area, reachable from the app's primary navigation.
- **FR-002**: The Goals area MUST list the signed-in user's goals grouped by state, using the same state grouping as the existing interfaces (Committed, Incubating, and — when revealed — Completed, Archived).
- **FR-003**: Completed and Archived goals MUST be hidden by default, with an explicit control to reveal and re-hide them.
- **FR-004**: Goals MUST be presented in their saved rank order within each state group.
- **FR-005**: Selecting a goal MUST show a detail view with the goal's name, description, due date, and state.
- **FR-006**: A goal's description MUST be rendered as markdown when viewed, consistent with how other text is rendered in the web app.
- **FR-007**: When the user has no goals (or none in the visible groups), the Goals area MUST show a friendly empty state.
- **FR-008**: The goal detail view MUST list the goal's associated tasks for context, read-only; the web app does NOT provide linking or unlinking of tasks in this feature.

#### Reading status updates

- **FR-009**: The goal detail view MUST show the goal's most recent status update, including a human-readable timestamp, rendered as markdown.
- **FR-010**: When a goal has no status updates, the detail view MUST show a friendly "no status updates yet" indication.
- **FR-011**: The user MUST be able to view the full history of a goal's status updates, ordered newest first, each labeled with a human-readable timestamp.
- **FR-012**: A status update's text MUST be fully readable regardless of length (wrapping and/or scrolling), with no silent truncation.

#### Writing, editing, and deleting status updates

- **FR-013**: The user MUST be able to add a new status update to a goal from the web app; it is timestamped with the current time automatically (not user-entered) and becomes the goal's latest status.
- **FR-014**: A status update with empty or whitespace-only text MUST be rejected on add or edit, with a clear message and no change recorded.
- **FR-015**: The user MUST be able to edit the text of an existing status update; the edit replaces the text under the update's original timestamp (no separate "edited at" time).
- **FR-016**: The user MUST be able to delete an individual status update; the goal's other updates are unaffected.

#### Scope and consistency

- **FR-017**: Goals and their status updates MUST be scoped to the signed-in user; a user only ever sees and modifies their own goals and updates.
- **FR-018**: Changes made in the web app MUST be consistent with the existing interfaces — a status update added/edited/deleted on the web is reflected in the TUI/CLI and vice versa.
- **FR-019**: For this feature, the web app MUST treat goals themselves as read-only: creating, editing, deleting, changing the state of, or reordering goals is out of scope and remains available only in the existing interfaces.
- **FR-020**: Loading and saving failures (network/server errors, not-found) MUST surface a clear message and allow the user to recover (retry/refresh) without ambiguous success.

### Key Entities *(include if feature involves data)*

- **Goal** *(existing)*: A long-term goal with a name, optional description, optional due date, a lifecycle state (Incubating, Committed, Completed, Archived), a rank position within its state group, and a most-recent status update. Read-only in the web app for this feature. Scoped to the owning user.
- **Status Update** *(existing)*: A timestamped, free-form markdown text entry recording a goal's state at a point in time. Belongs to exactly one goal, ordered newest-first. Created, edited, and deleted from the web app in this feature. Inherits ownership from its goal.
- **Associated Task** *(existing)*: A task linked to a goal (its subtree belongs to the same goal). Displayed read-only in the goal detail for context.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A signed-in user with existing goals can open the web app and see those goals grouped by state, with completed/archived hidden by default, without using the TUI or CLI.
- **SC-002**: A user can open any goal and read its latest status update (rendered, with timestamp) and browse its full history newest-first, with 100% of an update's text readable regardless of length.
- **SC-003**: A user can record a new status update from the web app and see it become the goal's latest status within the same view, and that update is retrievable later with identical text and timestamp across sessions and interfaces.
- **SC-004**: A user can correct a status update's text and delete an unwanted update from the web app, with changes reflected immediately and consistently in the other interfaces.
- **SC-005**: Markdown in goal descriptions and status updates is rendered (not shown as raw markup) in 100% of web views, matching the rendering of other text in the web app.
- **SC-006**: Empty/whitespace-only status updates are rejected on both add and edit in 100% of attempts, with a clear message and no data recorded.

## Assumptions

- **Backend already provides goal capabilities**: The server already exposes the goal and status-update operations used by the TUI/CLI (list/get goals, and list/add/edit/delete status updates). This feature consumes those existing capabilities; the backend is not modified.
- **Authentication is reused**: The web app's existing sign-in/session handling governs access; no new auth work is in scope.
- **Markdown rendering reuses existing capability**: Goal descriptions and status updates are rendered with the web app's existing markdown rendering (feature 055), so behavior is consistent and no new rendering rules are defined here.
- **Goals are read-only in this iteration**: Creating, editing, deleting, state changes, and ranking of goals are intentionally out of scope for the web app for now; the user asked to "view goals." Those remain available in the TUI/CLI.
- **Associated tasks are display-only**: The goal detail lists associated tasks for context; linking/unlinking tasks from the web is out of scope.
- **Status-update history matches existing semantics**: Updates are append-with-edit/delete (history log), newest-first; edits preserve the original creation timestamp, consistent with the existing behavior.
- **No reminders/notifications**: Recording an update is a manual act; the feature does not prompt or schedule reminders.
- **Single-user-per-view scope**: As with tasks and plans today, the web app shows only the signed-in user's data.

## Dependencies

- Existing server goal and status-update operations (the same ones backing the TUI/CLI goals features).
- The web app's existing authentication/session layer.
- The web app's existing markdown rendering (feature 055).
- The web app's existing task data (tasks carry an optional goal association) for listing a goal's associated tasks read-only.
