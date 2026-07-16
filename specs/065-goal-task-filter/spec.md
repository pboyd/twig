# Feature Specification: Goal Task Filter Shortcut

**Feature Branch**: `065-goal-task-filter`

**Created**: 2026-07-16

**Status**: Draft

**Input**: User description: "shortcut to filter tasks by goal — In the TUI, if a user presses `ctrl+t`, they should be taken to the tasks tab with a filter enabled to limit the displayed tasks to only that goal (`^goal_id=<id>`). Users can technically filter tasks by goal now, but they would need to know the goal ID, which isn't displayed anywhere."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Jump from a goal to its tasks (Priority: P1)

A user is looking at their goals. They want to see everything they still have to do for one of them. They put the cursor on that goal, press `ctrl+t`, and land on the Tasks tab showing only the tasks belonging to that goal — including tasks nested anywhere beneath a task attached to the goal. They never have to know or type the goal's ID.

**Why this priority**: This is the entire feature. Filtering tasks by goal already works, but it is unreachable in practice because goal IDs are never displayed. This story removes the only thing standing between the user and an existing capability.

**Independent Test**: Can be fully tested by creating a goal with attached tasks plus at least one unrelated goal with its own tasks, selecting the first goal in the Goals tab, pressing `ctrl+t`, and confirming the Tasks tab shows only the first goal's tasks with the corresponding filter visible.

**Acceptance Scenarios**:

1. **Given** the Goals tab is showing with the cursor on a goal that has attached tasks, **When** the user presses `ctrl+t`, **Then** the active tab becomes Tasks and only tasks belonging to that goal are listed.
2. **Given** the user has jumped to the Tasks tab via `ctrl+t`, **When** they look at the filter bar, **Then** it displays the filter expression that is in effect, scoping the list to the selected goal.
3. **Given** a goal whose attached task has nested subtasks, **When** the user presses `ctrl+t` on that goal, **Then** the subtasks are shown too, not only the directly attached tasks.
4. **Given** a goal with no attached tasks, **When** the user presses `ctrl+t`, **Then** the Tasks tab opens with the goal's filter applied and shows the standard "no tasks match" empty state rather than a blank screen or an error.
5. **Given** the Tasks tab already has a different filter applied, **When** the user returns to the Goals tab and presses `ctrl+t` on a goal, **Then** the previous filter is replaced by the selected goal's filter.

---

### User Story 2 - Adjust or clear the goal filter after arriving (Priority: P2)

Having landed on the Tasks tab scoped to a goal, the user treats that filter like any other: they refine it, or they drop it to get their whole task list back. Nothing about arriving via the shortcut makes the filter special or sticky.

**Why this priority**: Depends on P1 existing, but without it the shortcut is a trap — the user gets scoped in with no obvious way out. The existing filter controls already provide the mechanism; this story is about confirming the shortcut plugs into them rather than bypassing them.

**Independent Test**: Can be tested by pressing `ctrl+t` on a goal and then exercising each existing filter control (reopen for editing, clear) to confirm the goal filter behaves exactly like a hand-typed one.

**Acceptance Scenarios**:

1. **Given** the user arrived on the Tasks tab via `ctrl+t`, **When** they reopen the filter input, **Then** it is pre-filled with the goal filter expression, ready to be edited or extended.
2. **Given** the user arrived on the Tasks tab via `ctrl+t`, **When** they clear the active filter, **Then** the full task list is restored.
3. **Given** the user arrived on the Tasks tab via `ctrl+t`, **When** they navigate, edit, complete, or start a pomodoro on a task, **Then** those actions work against the filtered list exactly as they do under a manually typed filter.

---

### Edge Cases

- **Goals list is empty**: `ctrl+t` with no goal under the cursor does nothing — no tab switch, no error.
- **Goal filter matches nothing visible under current visibility settings**: a goal whose tasks are all completed shows the "no tasks match" state while "show all" is off, and reveals the completed tasks once "show all" is turned on — the filter scopes by goal and defers to the existing completion/snooze defaults.
- **Shortcut pressed outside the goal list**: while a goal form, task form, goal picker, status history, or help view is open on the Goals tab, `ctrl+t` is not treated as the jump shortcut; the open view keeps its own key handling.
- **Goal deleted after jumping**: a filter referencing a goal that no longer exists returns an empty result, not an error.
- **Key collision**: `ctrl+t` already means "go to task" on the Plan tab. The Goals-tab binding is a distinct binding on a distinct tab and must not change Plan-tab behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Goals tab MUST provide a keyboard shortcut (`ctrl+t`) that switches the user to the Tasks tab with the task list filtered to the goal under the cursor.
- **FR-002**: The applied filter MUST scope the list to tasks attached to the selected goal *and* their descendant subtasks, matching the behavior of the existing transitive goal filter (`^goal_id=<id>`).
- **FR-003**: The applied filter MUST be identical in kind to a user-typed filter — visible in the filter bar, editable via the existing filter input, and clearable via the existing clear action.
- **FR-004**: The shortcut MUST replace any filter previously active on the Tasks tab rather than combining with it.
- **FR-005**: The shortcut MUST resolve the selected goal's identifier internally, so that the user never needs to know, see, or type a goal ID.
- **FR-006**: The shortcut MUST take no action when there is no goal under the cursor (for example, an empty goals list).
- **FR-007**: The shortcut MUST only apply when the Goals tab is in its normal list view, leaving key handling untouched while a form, picker, history, or help view is open.
- **FR-008**: The shortcut MUST NOT alter the meaning of `ctrl+t` on the Plan tab, where it already jumps to the task linked to the selected plan entry.
- **FR-009**: The applied filter MUST defer to the existing completion/snooze visibility defaults, so that "show all" continues to govern whether completed and snoozed tasks in the goal appear.
- **FR-010**: The Goals tab help listing MUST document the shortcut so it is discoverable.

### Key Entities

- **Goal**: A long-term objective the user tracks on the Goals tab. Has an identifier that is used internally by the filter but is not surfaced in the interface. Tasks may be attached to it.
- **Task**: A unit of work that may be attached to a goal and may have nested subtasks. A task is considered part of a goal's scope if it is attached to the goal or descends from a task that is.
- **Filter expression**: The user-facing text that scopes the task list. The shortcut authors one on the user's behalf; it is otherwise indistinguishable from one the user typed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From the Goals tab, a user can see the open tasks for a chosen goal with a single keystroke, without typing anything and without any prior knowledge of goal identifiers.
- **SC-002**: 100% of goals reachable in the goals list can be filtered on this way, including goals with no attached tasks (which produce a clear empty state rather than an error).
- **SC-003**: The resulting task set is identical to what the user would get by manually typing the equivalent goal filter — the shortcut is a shortcut, not a second code path with its own behavior.
- **SC-004**: After jumping, a user can return to their unfiltered task list using the filter-clearing action they already know, with no goal-specific escape hatch to learn.
- **SC-005**: Existing Plan-tab and Tasks-tab keyboard behavior is unchanged; no previously working shortcut changes meaning.

## Assumptions

- The filter expression applied is `^goal_id=<id>` for the selected goal, exactly as the user specified — with no completion or snooze condition attached. Per the established filter semantics, an expression that names neither `completed` nor `snoozed` leaves the default visibility in force, so the user sees incomplete, unsnoozed tasks for the goal unless "show all" is on. This matches what "show me this goal's tasks" is expected to mean.
- The existing task-filter capability (expression syntax, transitive `^goal_id` matching, filter bar, editing, and clearing) is already in place and is reused as-is. This feature adds a way to reach it, not a new filtering engine.
- The goals list already tracks which goal is under the cursor; the shortcut reads that selection rather than introducing a new selection concept.
- `ctrl+t` is unbound on the Goals tab and is available. Its existing Plan-tab meaning ("go to task") is close enough in spirit that reusing the key aids the mnemonic rather than confusing it.
- Scope is the TUI only. The web app is untouched, and no server-side or API change is expected, since the underlying filter is already evaluated server-side.
- The goal's ID remains hidden from the interface; this feature deliberately does not solve "display goal IDs" — it removes the need to know them.
