# Feature Specification: Plan Tab Task Navigation

**Feature Branch**: `038-plan-tab-task-nav`

**Created**: 2026-06-05

**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Jump to Task from Plan Entry (Priority: P1)

A user is reviewing today's plan in the Planning tab and notices that a linked task has the wrong name or is in the wrong place in the task tree. Rather than switching tabs manually and scrolling to find the task, the user presses a shortcut key on the highlighted plan entry. The application switches to the Tasks tab and positions the cursor directly on that task, expanding the tree as needed to make it visible.

**Why this priority**: This is the entire value of the feature — eliminating the friction of manually navigating between tabs.

**Independent Test**: Open the Planning tab with at least one plan entry linked to a task. Highlight that entry and press `ctrl+t`. Verify the Tasks tab is now active and the cursor is on the linked task.

**Acceptance Scenarios**:

1. **Given** the Planning tab is active and the highlighted entry has a linked task, **When** the user presses `ctrl+t`, **Then** the Tasks tab becomes active and the cursor is positioned on the linked task.
2. **Given** the linked task is a subtask whose parent is collapsed, **When** the user presses `ctrl+t`, **Then** all ancestor nodes are expanded so the task is visible, and the cursor lands on it.
3. **Given** the Planning tab is active and the highlighted entry is an event (no linked task), **When** the user presses `ctrl+t`, **Then** nothing happens (the keypress is silently ignored).

---

### User Story 2 - No Action for Unlinked Entries (Priority: P2)

A user presses `ctrl+t` on a plan entry that is a calendar event (not linked to any task). The application does nothing, avoiding confusion.

**Why this priority**: Correct no-op behavior prevents surprising side effects. Lower priority because the happy path is more important.

**Independent Test**: Press `ctrl+t` on a plan event entry. Confirm the tab and cursor do not change.

**Acceptance Scenarios**:

1. **Given** the highlighted plan entry has no linked task, **When** the user presses `ctrl+t`, **Then** the active tab and cursor position remain unchanged.

---

### Edge Cases

- What happens when the linked task has been deleted since the plan entry was created? The task will not be found in the tree; the navigation silently does nothing.
- What if the linked task is a completed task and completed tasks are currently hidden? The task may not be visible; navigation should still attempt to position the cursor on it (completed tasks exist in the tree even when filtered), or do nothing if the task cannot be made visible.
- What if the Planning tab is not in `planList` mode (e.g., the edit form is open)? The shortcut should be inactive while a sub-form is open.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: While the Planning tab is in list mode (no sub-form open), pressing `ctrl+t` on a plan entry linked to a task MUST switch the active tab to the Tasks tab.
- **FR-002**: After switching to the Tasks tab, the cursor MUST be positioned on the linked task.
- **FR-003**: If any ancestor node in the task tree is collapsed and would hide the linked task, those ancestors MUST be expanded before the cursor is positioned.
- **FR-004**: If the highlighted plan entry has no linked task (event entry), pressing `ctrl+t` MUST produce no visible change.
- **FR-005**: If the linked task cannot be found in the in-memory task tree (e.g., deleted), pressing `ctrl+t` MUST produce no visible change.
- **FR-006**: The shortcut MUST be inactive when the Planning tab is in any mode other than `planList` (e.g., edit form, add-task picker, add-event form).

### Key Entities

- **Plan Entry**: A row in the daily plan. May or may not be linked to a task via a task ID.
- **Task**: An item in the task tree. Has an ID, may have a parent, and may be nested arbitrarily deep.
- **Task Tree**: The hierarchical structure of tasks displayed in the Tasks tab, with expand/collapse state per node.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can navigate from any linked plan entry to its task in the Tasks tab in a single keypress.
- **SC-002**: The task is immediately visible (not hidden behind a collapsed parent) after the navigation.
- **SC-003**: Pressing the shortcut on an unlinked plan entry produces no state change, verified by confirming the active tab and cursor position are identical before and after.

## Assumptions

- Option 1 (tab switch + tree reveal) is the preferred implementation; Option 2 (inline edit form) is a fallback only if Option 1 proves unreasonably complex.
- The feature does not need to work for already-completed tasks when completed tasks are hidden from the tree view (acceptable degradation).
- The shortcut key `ctrl+t` is not already bound to another action in the Planning tab; if a conflict is found during implementation, a replacement key will be chosen.
- The task tree is already loaded in memory when the user is in the Planning tab (both tabs are loaded at startup).
