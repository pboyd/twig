# Feature Specification: TUI Completed Task Display

**Feature Branch**: `016-tui-completed-task-display`

**Created**: 2026-05-28

**Status**: Draft

**Input**: User description: "When a task is completed in the TUI it disappears immediately and the first task in the list is highlighted. Instead of a satisfied feeling of finishing a task, it's an annoyed feeling of moving back to where I was before. I would prefer that the task is re-styled as a completed task and remain highlighted. When the user highlights something else it should disappear.

Additionally, tasks that are completed don't appear completed in the tree. The name in the detail pane does not need to be styled differently, but that styling should be applied in the tree."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Completed task lingers with completion styling (Priority: P1)

When a user marks a task as completed in the TUI tree view, the task does not vanish immediately. Instead, it stays in its current position, takes on a "completed" visual style (e.g. strikethrough or muted/dim coloring), and remains the highlighted (selected) row. The task only disappears from the tree the next time the user moves the selection to a different task.

**Why this priority**: This is the core UX complaint from the user. Today, completing a task causes the cursor to jump back to the top of the list, which feels like punishment rather than reward. Fixing this delivers the emotional payoff the user wants from finishing a task and is the central reason this feature exists.

**Independent Test**: With at least three incomplete tasks in the tree, navigate to a middle task, mark it complete, and confirm that (a) the task still appears in the tree, (b) it shows the completed visual style, (c) it is still the selected row, and (d) it disappears from the tree as soon as the user moves the selection up or down to another task.

**Acceptance Scenarios**:

1. **Given** the tree shows three incomplete tasks and the middle task is highlighted, **When** the user marks that task as completed, **Then** the task remains at its current position in the tree, is styled as completed, and is still the highlighted row.
2. **Given** a task has just been completed and is still visible in the tree as the highlighted row, **When** the user moves the selection to another task, **Then** the previously completed task is removed from the tree on that movement and the newly selected task is highlighted normally.
3. **Given** a task has just been completed and is still visible in the tree as the highlighted row, **When** the user takes no further action and remains on that row, **Then** the completed task continues to be displayed with completed styling and stays highlighted.

---

### User Story 2 - Already-completed tasks render with completion styling in the tree (Priority: P2)

Tasks that are already completed (e.g. completed earlier in this session, or completed children of a parent task that is still being worked on) must visually indicate their completed state in the tree view, using the same completion styling as the "lingering completed task" from Story 1. The detail pane to the right of the tree continues to render the task name without that completion styling.

**Why this priority**: The user reported that completed subtasks under an incomplete parent currently render identically to incomplete tasks in the tree, which makes the tree hard to read. This is a visible defect but independent of the linger-on-completion behavior — fixing it on its own already delivers clearer tree rendering even before Story 1 ships.

**Independent Test**: Create a parent task with at least one completed child and one incomplete child. Open the tree view. Confirm that the completed child is visually distinguishable from the incomplete child (e.g. strikethrough/dim) using the same style applied in Story 1, and that selecting the completed child shows its name in the detail pane without that styling.

**Acceptance Scenarios**:

1. **Given** a parent task with one completed and one incomplete child, **When** the tree view renders, **Then** the completed child is shown with completion styling while the incomplete child is shown with the normal style.
2. **Given** a completed task is highlighted in the tree, **When** the detail pane shows that task's name, **Then** the name in the detail pane is rendered without completion styling.
3. **Given** a tree containing completed tasks at any nesting level (root or nested), **When** the tree is rendered, **Then** every completed task uses the completion styling regardless of depth.

---

### Edge Cases

- **Completing the only remaining task**: If the user completes the last incomplete task in the tree, the task stays visible with completed styling and remains highlighted. It disappears only if/when the user moves selection elsewhere; if there is nowhere to move, the row continues to display.
- **Completing a parent task with children**: When a parent is marked complete, the parent itself uses the linger-and-restyle behavior; any cascading completion of its children (if that already happens) follows the same "show completed styling" rendering rule from Story 2.
- **External update while lingering**: If the task list refreshes (e.g. due to a sync or an action that reloads tasks) while a just-completed task is still lingering, the just-completed task is removed on that refresh (treated equivalently to the user moving selection away). The cursor follows the normal post-refresh behavior.
- **Un-completing a lingering task**: If the user toggles the just-completed task back to incomplete while it is still the highlighted "lingering" row, it reverts to normal styling in place and stays highlighted as an incomplete task.
- **Moving selection then moving back**: Once the user moves off the completed task and it is removed from the tree, moving back does not restore it. The "linger" only applies until the first selection change.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: When a task is marked completed from the TUI tree view, the tree view MUST continue to display that task in its current position, with completion styling applied, and MUST keep it as the currently highlighted row.
- **FR-002**: The lingering completed task MUST be removed from the tree on the next user action that changes the selected row (e.g. moving up/down to another task), at which point the newly selected task is highlighted normally.
- **FR-003**: The tree view MUST render every task whose state is completed with a visually distinct "completed" style (consistent with how the lingering completed task is styled), regardless of nesting depth.
- **FR-004**: The detail pane MUST render the task name without the completed styling, even when the selected task is completed.
- **FR-005**: The completion styling in the tree MUST clearly distinguish a completed task from an incomplete task (so the user can tell at a glance which tasks are done).
- **FR-006**: If the tree is refreshed/reloaded while a just-completed task is in its lingering state, that task MUST be filtered out of the new render (the linger does not survive a reload).
- **FR-007**: If the user un-completes the lingering task before moving selection, the row MUST revert to normal (incomplete) styling in place and remain selected.

### Key Entities *(include if feature involves data)*

- **Task (tree row)**: Represents a single task displayed in the TUI tree. Already has a completion state. This feature adds a transient "just completed and still lingering" presentation state that is local to the current TUI view and does not need to be persisted.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After marking a task complete in the tree, the user's selected row does not jump to a different task in 100% of cases (the selection stays on the row they just completed until they move it themselves).
- **SC-002**: A user looking at the tree can identify which tasks are completed without opening the detail pane, with no false positives or false negatives across all visible rows.
- **SC-003**: After completing a task and then pressing a single navigation key (up or down), the completed task is gone from the tree and the newly selected task is the adjacent task in the navigation direction.
- **SC-004**: The user reports (qualitatively) that completing a task now feels rewarding rather than disorienting — the original frustration described in the input is no longer reproducible.

## Assumptions

- The TUI tree view is the only surface that needs this behavior change. Non-TUI surfaces (CLI commands that complete tasks) are out of scope.
- "Completion styling" can be any visually clear differentiation (strikethrough, dim color, etc.); the spec does not require a specific visual treatment, only that it be clearly distinguishable from incomplete tasks and consistent between the two stories.
- The lingering completed task is a view-only state — no new field is persisted to the database or sent over the API. It is reset whenever the tree is rebuilt from a fresh data source.
- The user's current keybinding for "complete task" in the TUI is preserved; this feature changes only what happens visually and to selection after completion.
- The detail pane already renders the task name independently of the tree's styling, so applying tree-only completion styling does not require structural changes to the detail pane.
