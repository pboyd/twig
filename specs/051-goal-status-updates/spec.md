# Feature Specification: Goal Status Updates

**Feature Branch**: `051-goal-status-updates`

**Created**: 2026-06-12

**Status**: Draft

**Input**: User description: "status updates for goals — Users want to record the latest status of a goal and be able to read it again later. Status updates are (potentially long) text entries associated to a goal. They should be timestamped. These should be viewable in the TUI, and (like other text fields) should render markdown when viewed."

## Clarifications

### Session 2026-06-12

- Q: How should a goal's status be modeled over time? → A: History log — a goal accumulates many timestamped updates; the newest is shown as the "latest status" and older updates remain readable as history (not a single overwritten status field).
- Q: Which interfaces should manage status updates in this feature? → A: TUI only — add/view/edit status updates only in the TUI Goals tab; CLI (`twig goal`) and web support are out of scope for this feature.
- Q: Should existing status updates be editable, or append-only? → A: Editable + deletable — users can edit a past update's text and delete updates; edited text replaces the original under its original creation timestamp (no separate "last edited" time).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Record and read the latest status of a goal (Priority: P1)

A user pursuing a long-term goal like "buy a new car" wants to capture where things stand right now — "Narrowed it down to two EVs; waiting on a test-drive appointment next week." The user opens the goal in the Goals tab, adds a status update as free-form text, and the update is stamped with the moment it was recorded. From then on, whenever the user views that goal, they see the most recent status — formatted as markdown — so they can pick up where they left off without re-reading everything.

**Why this priority**: This is the heart of the feature — getting the current state of a goal out of the user's head and into a place they can return to. Without the ability to record an update and read it back, nothing else in the feature has value.

**Independent Test**: Can be fully tested by opening a goal, adding a status update, and confirming the goal's detail view shows that update with its timestamp and renders its markdown — with no history browsing or editing involved.

**Acceptance Scenarios**:

1. **Given** a goal with no status updates, **When** the user views the goal's detail, **Then** the detail shows a friendly indication that there are no status updates yet.
2. **Given** a goal is selected, **When** the user adds a status update with some text, **Then** the update is saved with the current date and time and the goal's detail view shows it as the latest status.
3. **Given** a status update containing markdown (headings, lists, emphasis, code), **When** the user views it, **Then** it is rendered the same way other text fields are rendered, not shown as raw markup.
4. **Given** a goal already has a status update, **When** the user adds a newer one, **Then** the newer update becomes the one shown as the latest status, with its own timestamp.
5. **Given** the user starts adding a status update but leaves the text empty, **When** they try to save, **Then** the empty update is rejected with a clear message and nothing is recorded.

---

### User Story 2 - Browse the history of status updates (Priority: P2)

Over weeks, the user records several updates on a goal. To remember how things have progressed — and to recall a detail from an earlier note — the user reviews the full history of updates for the goal, each with its timestamp, newest first, and reads through even lengthy entries.

**Why this priority**: The running history is what turns status updates into a journal of progress rather than a single sticky note. It is valuable, but the feature already delivers value with just the latest update (US1).

**Independent Test**: Can be tested by recording several updates on one goal over time, then opening the history and confirming every update appears with its timestamp in newest-first order and that long entries can be read in full.

**Acceptance Scenarios**:

1. **Given** a goal with multiple status updates, **When** the user views the goal's status history, **Then** all updates are listed newest first, each labeled with its timestamp.
2. **Given** a status update whose text is longer than the visible area, **When** the user reads it, **Then** the full text is accessible (it wraps and/or scrolls) without truncation.
3. **Given** a goal with many status updates, **When** the user browses the history, **Then** the user can move through all of them.

---

### User Story 3 - Correct or remove a status update (Priority: P3)

The user notices a typo in a status update, or recorded one against the wrong goal. The user edits the text of an existing update, or deletes an update entirely.

**Why this priority**: Fixing mistakes is a quality-of-life refinement; the feature is usable without it because new updates can always be appended.

**Independent Test**: Can be tested by editing an existing update's text and confirming the change is shown, and by deleting an update and confirming it no longer appears while the goal's other updates remain.

**Acceptance Scenarios**:

1. **Given** an existing status update, **When** the user edits its text and saves, **Then** the updated text is shown and rendered as markdown.
2. **Given** an existing status update, **When** the user deletes it, **Then** it no longer appears in the goal's status history and the goal's other updates are unaffected.
3. **Given** a goal whose only status update is deleted, **When** the user views the goal's detail, **Then** the detail returns to the "no status updates yet" state.

---

### Edge Cases

- A goal has no status updates: the detail view shows a friendly empty state, not a blank area or error.
- A status update's text is very long: the text wraps and scrolls so the whole entry is readable; nothing is silently truncated.
- A status update contains malformed or unusual markdown: it is rendered safely and legibly, consistent with how other text fields handle markdown.
- A goal with status updates is deleted: its status updates are deleted along with it (they have no meaning without the goal).
- A goal changes state (e.g., Incubating → Completed) while it has status updates: the updates are retained and remain viewable.
- Two users' goals: a user only ever sees and modifies status updates on their own goals, consistent with goals and tasks today.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A goal MUST be able to have zero or more status updates; each status update is associated with exactly one goal.
- **FR-002**: Each status update MUST have a free-form text body that supports potentially long content.
- **FR-003**: Each status update MUST be timestamped with the moment it was recorded; the timestamp is assigned automatically and not entered by the user.
- **FR-004**: Users MUST be able to add a status update to a goal.
- **FR-005**: A status update with an empty text body MUST be rejected; the body is required.
- **FR-006**: The goal detail view in the TUI MUST display the goal's most recent status update, including its timestamp.
- **FR-007**: Users MUST be able to view the full history of a goal's status updates, ordered newest first, each labeled with its timestamp.
- **FR-008**: Status update text MUST be rendered as markdown when viewed, consistent with how other text fields are rendered in the TUI.
- **FR-009**: Timestamps MUST be presented to the user in a human-readable form.
- **FR-010**: A status update's text MUST be fully readable regardless of length (wrapping and/or scrolling), with no silent truncation.
- **FR-011**: Status updates MUST be scoped to the owning user; a user can only see and modify status updates on their own goals.
- **FR-012**: Deleting a goal MUST delete its status updates.
- **FR-013**: Users MUST be able to edit the text of an existing status update and to delete an individual status update. *(Priority P3 — see User Story 3.)*
- **FR-014**: Status updates MUST persist across sessions, retaining their text, timestamp, and ordering.

### Key Entities *(include if feature involves data)*

- **Status Update**: A timestamped, free-form (markdown) text entry recording the state of a goal at a point in time. Belongs to exactly one goal. Key attributes: text body, creation timestamp. Ordered by timestamp (newest first) within a goal. Inherits ownership from its goal.
- **Goal** *(existing)*: Gains an associated collection of status updates. Otherwise unchanged. Deleting a goal removes its status updates.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can record a status update on a goal and see it reflected as the goal's latest status without leaving the Goals tab.
- **SC-002**: Every status update a user records can be retrieved and read again later, with the same text and the timestamp it was recorded at, in 100% of cases across restarts.
- **SC-003**: When a goal has multiple status updates, they are always presented newest-first, and a user can locate a specific past update by scanning the timestamped history.
- **SC-004**: A status update of at least 5,000 characters is fully readable (via wrapping/scrolling) with no loss of content.
- **SC-005**: Markdown in a status update is rendered (not shown as raw markup) in 100% of views, matching the rendering of other text fields.

## Assumptions

- **Markdown rendering reuses existing capability**: Rendering of status update text uses the same markdown rendering introduced for other TUI text fields (feature 050), so behavior is consistent and no new rendering rules are defined here.
- **No reminders or notifications**: Recording a status update is a manual act; the feature does not prompt, schedule, or remind users to post updates.
- **No size limit beyond practicality**: Updates may be long; no explicit maximum length is imposed beyond what is reasonable for storage and display.
- **Latest status surfaced in detail only**: Like task–goal associations in the Goals feature, the latest status (and history) appears in the goal detail view; the goal list rows are unchanged.

See the Clarifications section above for resolved decisions on the data model (history log), surfaces (TUI only), and editability (editable + deletable).
