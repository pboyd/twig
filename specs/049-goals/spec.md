# Feature Specification: Goals

**Feature Branch**: `049-goals`

**Created**: 2026-06-11

**Status**: Draft

**Input**: User description: "goals — The task list should always be split up into things that can be done... I propose a new object in twig: goals. Goals have an ID, name, description, and due date. Users can rank goals. Goals are not nested. Goals are in one of four states: Incubating, Committed, Completed, Archived. Tasks can be associated to goals. In the TUI, Goals should be the first tab with a 2-pane UI; completed and archived goals hidden by default; goals grouped by state. In the CLI, a `twig goal` subcommand manages goals. No web app support needed right now."

## Clarifications

### Session 2026-06-11

- Q: When a task with subtasks is associated with a goal, how do the subtasks relate to the goal? → A: Subtree inherits — associating a task brings its entire subtree into the goal; descendants implicitly belong to the same goal and cannot be associated with a different one; the goal detail view shows the full subtree.
- Q: In the TUI, where can the user manage a task's goal association? → A: Both sides — from the goal detail pane (create a new task attached to the goal, link/unlink existing tasks) and from the task edit views (set or clear the goal).
- Q: Where should a task's associated goal be visible in the task views? → A: Detail only — the task detail view (TUI detail pane, CLI task show) displays the associated goal's name; task tree/list rows stay unchanged.
- Q: Where does a goal land in the rank order when created or when it changes state? → A: Bottom of group — new goals append to the bottom of the Incubating group; a goal that changes state appends to the bottom of its new group; existing curated order is never disturbed.
- Q: Does "Goals is the first tab" make it the startup tab? → A: No — Goals appears first in the tab bar, but the TUI continues to open on the Tasks tab.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Capture and track long-term goals (Priority: P1)

A user has long-term aspirations like "buy a new car" that aren't actionable today. Instead of cluttering the task list (or repeatedly snoozing them), the user records each one as a goal with a name, an optional description, and an optional due date. The user opens the Goals tab in the TUI — the first tab — and sees their goals grouped by state. As life progresses, the user moves a goal between states: it starts as an idea (Incubating), becomes a commitment (Committed), and eventually is either achieved (Completed) or set aside (Archived). Completed and archived goals disappear from the default view so the list stays focused, but the user can reveal them on demand.

**Why this priority**: This is the core of the feature — getting non-actionable, long-term items out of the task list and into a dedicated home so the task list stays empowering. Without this, nothing else in the feature matters.

**Independent Test**: Can be fully tested by creating goals in the TUI, editing their fields, moving them through all four states, and confirming the grouping and default hiding behavior — with no task association or CLI involvement.

**Acceptance Scenarios**:

1. **Given** the user opens the TUI, **When** the app loads, **Then** Goals appears as the first tab in the tab bar (the app still opens on the Tasks tab), and switching to it shows a two-pane layout (goal list on one side, selected goal's details on the other), consistent with the Tasks and Plans tabs.
2. **Given** the Goals tab is open, **When** the user creates a goal with a name, **Then** the goal appears in the Incubating group with its name, and the user can later add a description and due date.
3. **Given** an Incubating goal, **When** the user changes its state to Committed, **Then** the goal moves to the Committed group immediately.
4. **Given** a Committed goal, **When** the user marks it Completed (or Archived), **Then** the goal disappears from the default goal list.
5. **Given** completed and archived goals exist, **When** the user toggles the show-hidden view, **Then** Completed and Archived groups become visible with their goals, and toggling again hides them.
6. **Given** a goal is selected in the list pane, **When** the user views the detail pane, **Then** it shows the goal's name, description, due date, and state.
7. **Given** a Completed goal, **When** the user changes its state back to Committed, **Then** the goal reappears in the default view under Committed (state changes are reversible).

---

### User Story 2 - Associate tasks with goals (Priority: P2)

A user incubating "buy a new car" wants to attach research tasks ("compare insurance quotes", "test drive the EV") to it. The user links existing or new tasks to a goal. Looking at a goal, the user sees which tasks belong to it. Tasks remain ordinary tasks — they appear in the task list as before — and tasks never have to belong to a goal.

**Why this priority**: Linking actionable steps to a goal is what turns a goal from a parking lot into a plan, but goals are useful on their own without it.

**Independent Test**: Can be tested by associating a task with a goal, viewing the goal's task list, dissociating the task, and confirming unlinked tasks behave exactly as today.

**Acceptance Scenarios**:

1. **Given** an existing task and an existing goal, **When** the user associates the task with the goal, **Then** the goal's detail view lists that task.
2. **Given** a task associated with a goal, **When** the user removes the association, **Then** the task no longer appears under the goal but is otherwise unchanged.
3. **Given** a task with no goal association, **When** the user works with it anywhere in the product, **Then** it behaves exactly as tasks do today.
4. **Given** a goal with associated tasks, **When** the goal is completed, archived, or deleted, **Then** the tasks themselves are not completed or deleted (deleting a goal only removes the association).
5. **Given** a task associated with goal A, **When** the user associates it with goal B, **Then** it belongs to goal B only (a task belongs to at most one goal).

---

### User Story 3 - Manage goals from the CLI (Priority: P3)

A user who lives in the terminal manages goals without opening the TUI: listing goals, creating one, viewing details, editing fields, changing state, and deleting — via a `twig goal` subcommand, consistent in feel with the existing `twig task` commands.

**Why this priority**: The CLI is a first-class interface in twig and parity is expected, but the TUI alone delivers the feature's value.

**Independent Test**: Can be tested entirely from a shell: run goal subcommands and verify output and resulting state, including that changes made via CLI are visible in the TUI and vice versa.

**Acceptance Scenarios**:

1. **Given** goals exist, **When** the user runs the goal list command, **Then** goals are shown grouped by state with Completed and Archived omitted unless an option requests them.
2. **Given** a shell, **When** the user creates a goal via the CLI with a name (and optionally a description and due date), **Then** the goal is created in the Incubating state and is visible from both CLI and TUI.
3. **Given** an existing goal, **When** the user changes its state, edits its fields, or deletes it via the CLI, **Then** the change takes effect and is reflected everywhere.
4. **Given** a goal ID that doesn't exist, **When** the user references it in a goal command, **Then** the command fails with a clear error message.

---

### User Story 4 - Rank goals (Priority: P4)

A user with several goals in a state group puts the most important ones first. In the TUI, the user moves the selected goal up and down with the same `{` and `}` keys used to rank tasks. The order persists across sessions and is reflected wherever goals are listed.

**Why this priority**: Ordering is a quality-of-life refinement; goals are fully usable in default order.

**Independent Test**: Can be tested by creating several goals in one state, reordering them with `{`/`}`, restarting the TUI, and confirming the order persisted and appears in CLI listings too.

**Acceptance Scenarios**:

1. **Given** three goals in the same state group, **When** the user presses `{` on the bottom goal repeatedly, **Then** it moves up one position per press within its group.
2. **Given** a goal at the top of its state group, **When** the user presses `{`, **Then** the order is unchanged (ranking does not move a goal across state groups).
3. **Given** goals have been reordered, **When** the user restarts the TUI or lists goals in the CLI, **Then** the saved order is shown.

---

### Edge Cases

- A goal is completed, archived, or deleted while it still has incomplete associated tasks: the tasks are untouched and remain in the task list; only the association is removed on delete.
- A goal's due date is in the past: the goal is displayed with its overdue date; nothing is enforced or auto-changed.
- The user has no goals yet: the Goals tab shows a friendly empty state rather than a blank pane.
- Every goal is Completed or Archived: the default view shows the empty state until the user toggles hidden goals on.
- A goal name is empty: creation/edit is rejected with a clear message (name is required; description and due date are optional).
- Two users' goals: a user only ever sees and modifies their own goals, consistent with tasks today.
- Ranking keys are pressed when only one goal exists in the group: nothing happens, no error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support a goal object with a unique ID, a required name, an optional description, and an optional due date.
- **FR-002**: Each goal MUST be in exactly one of four states: Incubating, Committed, Completed, or Archived. New goals start as Incubating.
- **FR-003**: Users MUST be able to change a goal's state to any other state at any time (all transitions allowed, including reversals such as Completed → Committed).
- **FR-004**: Goals MUST NOT nest — there is no parent/child relationship between goals.
- **FR-005**: Users MUST be able to create, view, edit (name, description, due date), and delete goals.
- **FR-006**: Users MUST be able to associate a task with at most one goal, and to remove that association. Task–goal association MUST remain optional; tasks without a goal behave exactly as they do today.
- **FR-006a**: Associating a task with a goal MUST include its entire subtree: descendants implicitly belong to the same goal, cannot be associated with a different goal, and the goal's detail view shows the full subtree.
- **FR-007**: Deleting a goal MUST remove its task associations without deleting or otherwise modifying the tasks. Completing or archiving a goal MUST NOT change its associated tasks.
- **FR-008**: Goals MUST be scoped to the owning user; a user can only see and modify their own goals.
- **FR-009**: The TUI MUST present Goals as the first tab in the tab bar (the TUI continues to open on the Tasks tab), using a two-pane layout (goal list + detail of the selected goal) consistent with the Tasks and Plans tabs.
- **FR-010**: The TUI goal list MUST group goals by state, ordered Committed first, then Incubating, then (when shown) Completed, then Archived.
- **FR-011**: Completed and Archived goals MUST be hidden by default in the TUI and CLI listings, with an explicit control (TUI toggle, CLI option) to show them.
- **FR-012**: The goal detail view MUST show the goal's name, description, due date, state, and its associated tasks.
- **FR-013**: Users MUST be able to rank goals within their state group in the TUI using the `{` and `}` keys (matching task ranking), and the order MUST persist and apply to all goal listings.
- **FR-013a**: A newly created goal MUST append to the bottom of the Incubating group's rank order; a goal whose state changes MUST append to the bottom of its new group, leaving the existing order of other goals undisturbed.
- **FR-014**: The CLI MUST provide a `twig goal` subcommand covering listing, creating, showing, editing, changing state, and deleting goals.
- **FR-015**: Users MUST be able to manage task–goal associations from both sides: in the goal detail pane (create a new task attached to the goal, link/unlink existing tasks) and in the interfaces where tasks are edited (TUI task views and CLI).
- **FR-016**: Goal commands referencing a nonexistent goal MUST fail with a clear, user-friendly error.
- **FR-017**: Task detail views (TUI detail pane, CLI task show) MUST display the associated goal's name when one is set; task tree/list rows remain unchanged.

### Key Entities

- **Goal**: A long-term aim that is not directly actionable today. Attributes: ID, name (required), description (optional), due date (optional), state (Incubating | Committed | Completed | Archived), user-defined rank, owning user. Goals do not nest.
- **Task** (existing, extended): Gains an optional reference to a single goal. Association is set on a task and covers its entire subtree — descendants implicitly belong to the same goal and cannot be associated with a different one. All current task behavior is unchanged when no goal is set.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can move a non-actionable item off their task list and into a goal (create goal, delete or re-home the task) in under one minute.
- **SC-002**: A user can create a goal and see it in the goal list, correctly grouped by state, in under 30 seconds.
- **SC-003**: 100% of goal operations (create, view, edit, state change, delete, list) are available from both the TUI and the CLI, and a change made in one is visible in the other.
- **SC-004**: With completed and archived goals present, the default goal view shows 0 of them; the show-hidden control reveals all of them.
- **SC-005**: Goal ordering set by the user survives a restart and is identical across TUI and CLI listings.
- **SC-006**: Existing task workflows are unaffected: a user who never touches goals sees no change in task behavior.

## Assumptions

- New goals start in the Incubating state; all state transitions are allowed in any direction (no enforced lifecycle order).
- A task can belong to at most one goal at a time; assigning a new goal replaces the previous association.
- Completing a goal does not auto-complete its tasks, and completing all of a goal's tasks does not auto-complete the goal — state changes are always explicit user actions.
- Goals can be deleted outright (in addition to being archived); deletion removes task associations but never deletes tasks.
- State groups are displayed in the order Committed, Incubating, Completed, Archived — commitments are the most actionable and appear first.
- Ranking reorders a goal relative to other goals in the same state group; it never moves a goal between groups.
- Goals follow the same ownership and authentication model as tasks (per-user, API-key/session authenticated).
- Out of scope for this feature: web app support, goal data in the daily plan, goal data in the activity report, and notifications/reminders for goal due dates.

