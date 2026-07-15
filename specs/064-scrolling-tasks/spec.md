# Feature Specification: Scrolling Task List

**Feature Branch**: `064-scrolling-tasks`

**Created**: 2026-07-15

**Status**: Draft

**Input**: User description: "scrolling tasks — The task list can be very long (especially when viewing completed tasks), but it currently can't scroll. We need scrolling there. Also `PgUp` and `PgDn` should work like users expect."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Keep the selected task visible while navigating a long list (Priority: P1)

A user is working in the Tasks tab with more tasks than fit on screen. As they move the selection cursor down (or up) through the list, the view keeps the currently selected task visible, revealing tasks that were previously off-screen instead of losing track of where the cursor is.

**Why this priority**: This is the core defect. Without it, any task list longer than the visible area is partially unreachable/unviewable, making the tab unusable for real workloads — especially when completed tasks are shown and the list grows large.

**Independent Test**: Populate the Tasks tab with more tasks than the terminal can display, then move the cursor from the top item toward the bottom. Verify every task can be reached and the selected task always stays visible on screen.

**Acceptance Scenarios**:

1. **Given** a task list taller than the visible area with the cursor on the first visible task, **When** the user moves the selection down past the bottom of the visible area, **Then** the list scrolls so the newly selected task remains visible.
2. **Given** the list has been scrolled down, **When** the user moves the selection back up above the top of the visible area, **Then** the list scrolls up so the selected task remains visible.
3. **Given** a task list that fits entirely within the visible area, **When** the user navigates up and down, **Then** the list does not scroll and behavior is unchanged from today.
4. **Given** the user is viewing completed tasks and the list is very long, **When** they navigate to the last task, **Then** the last task is reachable and displayed.

---

### User Story 2 - Page through the list with PgUp / PgDn (Priority: P2)

A user with a long task list presses `PgDn` to jump forward roughly one screen at a time, and `PgUp` to jump back, matching the behavior they expect from other scrolling interfaces (including twig's own Report and goal-reader views).

**Why this priority**: Line-by-line navigation is tedious on long lists. Paging is the expected fast-scroll affordance and the user explicitly called it out, but it builds on the scrolling foundation from Story 1.

**Independent Test**: With a task list several screens tall, press `PgDn` repeatedly and confirm the view advances by approximately one screen each time until the end is reached; press `PgUp` and confirm it moves back symmetrically to the top.

**Acceptance Scenarios**:

1. **Given** a task list taller than one screen positioned at the top, **When** the user presses `PgDn`, **Then** the view advances by approximately one screen.
2. **Given** the list scrolled part-way down, **When** the user presses `PgUp`, **Then** the view moves back by approximately one screen.
3. **Given** the view is near the bottom with less than a full screen remaining, **When** the user presses `PgDn`, **Then** the view stops at the bottom without scrolling past the last task.
4. **Given** the view is near the top with less than a full screen above, **When** the user presses `PgUp`, **Then** the view stops at the top without scrolling above the first task.

---

### Edge Cases

- **Empty list**: With no tasks, scrolling and paging keys are no-ops and do not error or produce a blank/scrolled-away view.
- **List exactly fills the screen**: No scrolling occurs; paging keys are no-ops.
- **Terminal resize**: If the terminal is made shorter or taller, the visible portion and scroll position adjust so the selected task stays visible and the view never shows blank space below the last task.
- **Jump to top/bottom** (if such shortcuts exist): The view scrolls to keep the newly selected first/last task visible.
- **Filtering/collapsing changes the list length**: After the visible set of tasks changes, the scroll position stays valid (clamped) so it never strands the view past the end of the list.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Tasks tab MUST allow the task list to scroll when it contains more content than fits in the visible area.
- **FR-002**: As the selection cursor moves, the system MUST keep the selected task visible within the viewport, scrolling the list as needed.
- **FR-003**: The system MUST support `PgDn` to advance the task list by approximately one visible screen and `PgUp` to move back by approximately one visible screen.
- **FR-004**: Scrolling MUST be bounded — the view MUST NOT scroll above the first task or below the last task, and MUST NOT display blank space beyond the end of the list.
- **FR-005**: When the entire task list fits within the visible area, the system MUST NOT scroll and MUST preserve current behavior.
- **FR-006**: The scroll position MUST remain valid (clamped in range) when the list length changes due to filtering, showing/hiding completed tasks, expanding/collapsing, or terminal resize.
- **FR-007**: Paging and scrolling behavior in the Tasks tab SHOULD be consistent with the paging/scrolling already present elsewhere in the TUI (e.g., Report tab and goal-status reader) so the interaction feels familiar.

### Key Entities

- **Task list view**: The scrollable region of the Tasks tab, characterized by the total number of displayed task rows, the height of the visible area, the current scroll offset, and the currently selected task.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can reach and view every task in a list of any length, including lists several times taller than the terminal.
- **SC-002**: The currently selected task is visible on screen 100% of the time during up/down navigation.
- **SC-003**: `PgUp` and `PgDn` move the view by approximately one screen and never scroll past the first or last task.
- **SC-004**: For task lists that fit on screen, there is no observable change in behavior compared to before this feature.
- **SC-005**: The view never displays blank space below the last task, regardless of scroll position, list length changes, or terminal resizing.

## Assumptions

- "The task list" refers to the task tree shown in the Tasks tab of the interactive TUI, which is the region the user identified as unable to scroll.
- The Tasks tab has an existing selection cursor; scrolling is driven by keeping that cursor visible rather than introducing an independent scroll cursor.
- "Approximately one screen" for paging means the number of visible task rows in the current viewport (matching the paging convention used elsewhere in the TUI); a small overlap or exact-page step are both acceptable as long as it is consistent.
- `PgUp`/`PgDn` refer to the standard Page Up and Page Down keys; existing line-by-line navigation keys (arrows / `j`/`k`) continue to work as they do today.
- This change is limited to the terminal TUI; the web frontend is out of scope.
