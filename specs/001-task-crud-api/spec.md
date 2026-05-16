# Feature Specification: Task CRUD API

**Feature Branch**: `001-task-crud-api`

**Created**: 2026-05-16

**Status**: Draft

**Input**: User description: "I want to add CRUD endpoints to the backend API for tasks. A task has the following fields: id (incrementing integer primary key), name (short description of what needs to be done), description (longer text field describing the task), due (timestamp for when the task must be done). Only id and name are required." (Amended 2026-05-16: a task may also reference an optional parent task, forming a hierarchy of unrestricted depth; the system must prevent cycles so a task can never be its own ancestor.)

## Clarifications

### Session 2026-05-16

- Q: When a task is updated, does the request carry the complete desired state of all editable fields, or only the fields to change? → A: Full replace — the update request supplies name, description, due, and parent; any field left empty is stored as empty/cleared.
- Q: In what order should the task list be returned? → A: By identifier ascending (creation order, oldest first).
- Q: What is the maximum allowed length for a task name? → A: 255 characters maximum.
- Q: When a task that has child tasks is deleted, what happens to its descendants? → A: Cascade delete — deleting a task also deletes all of its descendant tasks.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Capture and review tasks (Priority: P1)

A person using the todo system needs to record things they have to do and see what they have recorded. They add a new task by providing at least a short name, optionally including a longer description and a due date. They can then retrieve the full list of their tasks, and look up any single task by its identifier.

**Why this priority**: Without the ability to create and view tasks, the system delivers no value. This story is the minimum usable product — a person can record work and see it again later.

**Independent Test**: Add several tasks (some with only a name, some with all fields) and confirm each one is returned correctly when the full list is requested and when looked up individually by identifier.

**Acceptance Scenarios**:

1. **Given** an empty task list, **When** a task is created with only a name, **Then** the task is stored, assigned a unique identifier, and returned with an empty description, no due date, and no parent.
2. **Given** an empty task list, **When** a task is created with a name, description, and due date, **Then** the task is stored and returned with all provided values intact.
3. **Given** a creation request with no name, **When** the request is submitted, **Then** it is rejected with a validation error and no task is stored.
4. **Given** several stored tasks, **When** the full task list is requested, **Then** every stored task is returned.
5. **Given** a stored task, **When** that task is requested by its identifier, **Then** the matching task is returned.
6. **Given** an identifier that matches no task, **When** that task is requested, **Then** a not-found error is returned.

---

### User Story 2 - Update an existing task (Priority: P2)

As the details of a task change — a clearer description, a new deadline, a corrected name — the person needs to amend a task they already recorded without recreating it.

**Why this priority**: Editing builds on stored tasks and is highly valuable, but the system is still usable for capture and review without it. It ranks below creation and reading.

**Independent Test**: Create a task, update each of its editable fields, and confirm the retrieved task reflects the new values while keeping the same identifier.

**Acceptance Scenarios**:

1. **Given** a stored task, **When** its name, description, and due date are updated, **Then** the task is returned with the new values and an unchanged identifier.
2. **Given** a stored task that has a due date, **When** it is updated to remove the due date, **Then** the task is returned with no due date.
3. **Given** an update request that clears the name, **When** the request is submitted, **Then** it is rejected with a validation error and the stored task is left unchanged.
4. **Given** an identifier that matches no task, **When** an update is requested for it, **Then** a not-found error is returned and nothing is stored.

---

### User Story 3 - Organize tasks into a hierarchy (Priority: P2)

A person wants to break large pieces of work into smaller ones and group related tasks together, so they nest a task underneath a parent task. A subtask can itself have subtasks, to any depth. As priorities shift, the person can move a task to a different parent or promote it back to a top-level task. The system never allows a task to be nested inside itself, directly or indirectly.

**Why this priority**: Nesting makes the todo system genuinely useful for real, multi-step work, but capturing and reviewing tasks (US1) and editing them (US2) remain usable without it. It builds directly on creation and update.

**Independent Test**: Create several tasks, assign parents to build a multi-level hierarchy, retrieve the tasks and confirm each reports the correct parent, then attempt to create a cycle and confirm it is rejected.

**Acceptance Scenarios**:

1. **Given** two stored tasks, **When** one is assigned the other as its parent, **Then** it is stored and returned with that parent reference.
2. **Given** a task already nested under a parent, **When** a third task is assigned that nested task as its parent, **Then** a three-level hierarchy exists and each task returns its correct parent.
3. **Given** a task created with no parent, **When** it is retrieved, **Then** it is returned as a top-level task with no parent reference.
4. **Given** a stored task, **When** an update assigns it itself, or one of its own descendants, as its parent, **Then** the request is rejected with a validation error and the task's parent is left unchanged.
5. **Given** a create or update request referencing a parent identifier that matches no task, **When** the request is submitted, **Then** it is rejected with a validation error and nothing is stored or altered.
6. **Given** a task nested under one parent, **When** it is updated with a different valid parent, **Then** it is returned nested under the new parent.
7. **Given** a task nested under a parent, **When** it is updated with no parent reference, **Then** it is returned as a top-level task.

---

### User Story 4 - Delete a task (Priority: P3)

When a task is finished or no longer relevant, the person needs to remove it so it no longer clutters their list.

**Why this priority**: Removal keeps the list relevant over time, but the system delivers value for capturing, reviewing, editing, and organizing tasks without it. It is the lowest priority of the four.

**Independent Test**: Create a task, delete it, and confirm it no longer appears in the list and can no longer be retrieved by its identifier.

**Acceptance Scenarios**:

1. **Given** a stored task with no children, **When** it is deleted, **Then** it is removed and no longer appears in the task list.
2. **Given** a deleted task, **When** it is requested by its identifier, **Then** a not-found error is returned.
3. **Given** an identifier that matches no task, **When** a delete is requested for it, **Then** a not-found error is returned.
4. **Given** a stored task that has child tasks (and possibly deeper descendants), **When** it is deleted, **Then** the task and every one of its descendant tasks are removed and no longer appear in the task list or are retrievable by identifier.

---

### Edge Cases

- A creation or update request whose name is empty or contains only whitespace is treated the same as a missing name and rejected.
- A name longer than the allowed maximum length is rejected with a validation error.
- A due date in the past is accepted — overdue tasks are valid and expected.
- Requesting, updating, or deleting a task with an identifier that has never existed, or that belonged to an already-deleted task, returns a not-found error.
- Requesting the task list when no tasks exist returns an empty list, not an error.
- Identifiers of deleted tasks are not reused by later task creations.
- A request that sets a task's parent to itself is rejected as a cycle.
- A request that sets a task's parent to one of its own descendants is rejected as a cycle.
- A request that sets a parent to an identifier matching no existing task is rejected with a validation error.
- A task may be moved to a different parent through an update, provided the move does not create a cycle.
- Deleting a task that has child tasks also deletes every descendant of that task (cascade delete).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow a task to be created with a name, and optionally a description, a due date, and a parent task.
- **FR-002**: System MUST require a non-empty name on every task and reject creation or update requests that omit it or supply only whitespace.
- **FR-003**: System MUST treat description, due date, and parent task as optional; a task may exist with none of them.
- **FR-004**: System MUST assign each created task a unique identifier that is a positive, increasing integer, and MUST NOT reuse the identifier of a deleted task.
- **FR-005**: System MUST allow retrieval of a single task by its identifier and return a not-found error when no task matches.
- **FR-006**: System MUST allow retrieval of all stored tasks ordered by identifier ascending (creation order, oldest first), returning an empty result when none exist.
- **FR-007**: System MUST allow the name, description, due date, and parent task of an existing task to be updated, identified by its identifier, while preserving the identifier.
- **FR-008**: System MUST allow the description, due date, and parent task of a task to be cleared through an update; clearing the parent makes the task a top-level task.
- **FR-009**: System MUST allow a task to be deleted by its identifier and return a not-found error when no task matches.
- **FR-010**: System MUST reject a name longer than 255 characters with a validation error.
- **FR-011**: System MUST persist tasks, including their parent relationships, so they remain available across server restarts.
- **FR-012**: System MUST return a clear, distinguishable error for each failure type: validation failure versus task not found.
- **FR-013**: System MUST allow a task to reference at most one other task as its parent; a task with no parent reference is a top-level task.
- **FR-014**: System MUST reject a create or update request whose parent reference does not match an existing task.
- **FR-015**: System MUST place no limit on the depth of the parent-child hierarchy.
- **FR-016**: System MUST reject any create or update that would make a task its own ancestor — a task cannot be its own parent, nor be nested (directly or indirectly) beneath itself — thereby preventing cycles, and MUST leave existing data unchanged when it does so.
- **FR-017**: System MUST cascade deletion through the hierarchy: deleting a task also deletes all of its descendant tasks (its children, their children, and so on).

### Key Entities

- **Task**: A single unit of work to be done. Attributes: a unique identifier (positive increasing integer, system-assigned); a name (required, short text describing what needs to be done); a description (optional, longer free text); a due date (optional timestamp for when the task must be completed); a parent task (optional reference, by identifier, to another task under which this task is nested). Tasks form a hierarchy of unrestricted depth; no task may appear anywhere in its own chain of ancestors.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A person can create a task and then retrieve it — by list or by identifier — and see every value they supplied returned unchanged, 100% of the time.
- **SC-002**: Every invalid request (missing name, over-length name, unknown identifier, unknown parent reference) is rejected with an error that identifies the cause, and no such request results in stored or altered data.
- **SC-003**: A created or updated task remains retrievable with its values, including its parent relationship, intact after a full restart of the system.
- **SC-004**: A deleted task — and every descendant of it — is absent from the task list and unretrievable by identifier immediately after deletion, 100% of the time.
- **SC-005**: All four operations (create, retrieve, update, delete) return a result within 1 second under normal single-user conditions.
- **SC-006**: A task can be nested beneath a parent to any depth, and each task's parent reference is returned unchanged on later retrieval, 100% of the time.
- **SC-007**: Every attempt to nest a task within its own ancestry (a cycle) is rejected, 100% of the time, and no cycle is ever persisted.

## Assumptions

- The system serves a single user or trusted caller; authentication, authorization, and per-user task ownership are out of scope for this feature.
- The task list is returned in full on every list request as a flat collection; each task carries its parent reference, and the hierarchy is derived from those references rather than from nested list responses. Pagination, filtering, and sorting are out of scope for this version.
- Deletion is permanent (hard delete); there is no archive, trash, or restore capability.
- An update request supplies the complete intended state of a task's editable fields (name, description, due date, parent); omitted optional fields are interpreted as cleared.
- A parent is referenced by the parent task's identifier; re-parenting (changing a task's parent) is permitted via update as long as it does not create a cycle.
- The maximum name length is 255 characters; description has no enforced length limit beyond the underlying storage capacity.
- Due dates are stored and returned as absolute timestamps in UTC.
- This feature builds on the existing backend foundation service (`services/todo`) and its PostgreSQL database; it adds a task schema and task endpoints alongside the existing health endpoint.
