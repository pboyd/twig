# Feature Specification: Interactive TUI

**Feature Branch**: `010-interactive-tui`

**Created**: 2026-05-25

**Status**: Draft

**Input**: User description: "The current CLI works OK, but it would be better to have a TUI. The TUI should run if the CLI is run without a command. ..."

## Clarifications

### Session 2026-05-25

- Q: What determines the display order of root tasks and of siblings under a parent? → A: Match the order used by the existing `todo task list` CLI.
- Q: How should the TUI handle external changes (tasks modified by another client during a session)? → A: Show data as-of-launch; re-fetch after every mutation; provide `Ctrl-R` for a manual full reload.
- Q: What key dismisses the help overlay? → A: Both `?` and `Esc` dismiss it.
- Q: Can the edit form be saved/cancelled without tabbing to the buttons? → A: Yes — `Esc` cancels from anywhere in the form, `Ctrl-S` saves from anywhere; the Save/Cancel buttons still work via tab.
- Q: When the user toggles the completed-task filter with `C`, where should the highlight land? → A: Keep highlight on the same task if still visible; otherwise fall back to the first visible task.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse and view tasks in a tree (Priority: P1)

A user runs the `todo` command with no subcommand and is dropped into an interactive screen that shows their incomplete tasks as a tree. The first task is highlighted, and a details pane to the right shows that task's id, name, due date, description, and other fields. Subtask trees start collapsed; the user can navigate up and down through the visible tasks, expand and collapse subtree branches, and see the details pane update in lockstep with the highlight.

**Why this priority**: This is the foundational interaction — without a working tree view and details pane, no other TUI feature is reachable. It also delivers immediate value on its own: a faster, more glanceable replacement for `todo task list`.

**Independent Test**: Launch `todo` with no arguments against a database that has at least one task with subtasks. Verify the tree displays, the first task is highlighted, the details pane reflects it, and the arrow / vim keys move the highlight and expand/collapse subtrees as described.

**Acceptance Scenarios**:

1. **Given** the user has incomplete tasks with subtasks, **When** they run `todo` with no arguments, **Then** the TUI launches with the incomplete tasks listed as a tree, all subtask branches collapsed, and the first task highlighted.
2. **Given** the TUI is open and a task with subtasks is highlighted, **When** the user presses `Right` or `L`, **Then** that task's subtask branch expands and its tree marker switches from `[+]` to `[-]`.
3. **Given** the TUI is open and a task with expanded subtasks is highlighted, **When** the user presses `Left` or `H`, **Then** that task's subtask branch collapses and its tree marker switches from `[-]` to `[+]`.
4. **Given** the TUI is open, **When** the user presses `Down` / `J` or `Up` / `K`, **Then** the highlight moves to the next or previous *visible* task, skipping over tasks inside collapsed branches.
5. **Given** the highlight changes to a different task, **When** the screen redraws, **Then** the details pane updates to show that task's id, name, due date, description, and other fields.
6. **Given** a completed task is shown in the list (e.g. while toggled on via `C`), **When** it is rendered, **Then** its name is drawn with strikethrough in a lighter color, matching the style used by `todo task` for completed tasks.
7. **Given** a task has no subtasks, **When** it is rendered, **Then** its marker shows as `[-]` with no subtree beneath it (the marker indicates "no children to hide", consistent with the example tree).

---

### User Story 2 - Edit, create, and delete tasks from the TUI (Priority: P1)

A user can mutate their task list without leaving the TUI: edit the highlighted task in a form, create a new subtask under it, create a new root task, delete the highlighted task, and toggle its completion state. After every mutation the tree reflects the change immediately and the highlight lands somewhere sensible.

**Why this priority**: A read-only TUI would force users back to the CLI for every change. Editing, creating, and deleting are core task-management actions and are needed for the TUI to fully replace the existing subcommands for daily use.

**Independent Test**: With the TUI open, exercise each of: editing a field via `E`, adding a subtask via `N`, adding a root task via `Ctrl-N`, deleting via `Ctrl-D`, and toggling completion via `Space`. Verify each action persists (visible after re-launching the TUI) and that the highlight behavior described below holds.

**Acceptance Scenarios**:

1. **Given** a task is highlighted, **When** the user presses `E`, **Then** focus moves to the details pane in edit mode, the fields show the current values, `Tab` cycles between fields, and `Save` / `Cancel` buttons sit at the bottom of the form.
2. **Given** the user is in the edit form, **When** they activate `Save`, **Then** the changes persist, focus returns to the task list, and the same task remains highlighted.
3. **Given** the user is in the edit form, **When** they activate `Cancel`, **Then** no changes are persisted and focus returns to the task list with the same task highlighted.
4. **Given** a task is highlighted, **When** the user presses `N`, **Then** focus moves to a blank details/edit pane for a new subtask of the highlighted task.
5. **Given** a new subtask is being created and the user activates `Save`, **When** focus returns to the task list, **Then** the parent's subtree is expanded if necessary and the newly created subtask is highlighted.
6. **Given** a new subtask is being created and the user activates `Cancel`, **When** focus returns to the task list, **Then** no new task is created and the originally highlighted task remains highlighted.
7. **Given** any task is highlighted, **When** the user presses `Ctrl-N`, **Then** a blank edit form opens for a new root-level task; on `Save` the new task is added at the root and highlighted, and on `Cancel` the original highlight is preserved.
8. **Given** a task is highlighted, **When** the user presses `Ctrl-D`, **Then** that task is deleted immediately without a confirmation prompt and the highlight moves to a sensible neighbor (the next visible task, or the previous one if there is no next).
9. **Given** a task is highlighted, **When** the user presses `Space`, **Then** the task's completion state toggles. If the list is currently filtered to incomplete tasks only and the task was just completed, it remains visible with the completed styling until the user moves the highlight away, after which it disappears from the list.

---

### User Story 3 - Pomodoro integration (Priority: P2)

From the TUI a user can start a pomodoro on the highlighted task, set its pomodoro estimate, and resume a previously backgrounded pomodoro — without exiting to the shell. When a pomodoro finishes, the user lands back on the task list rather than being kicked out of the program.

**Why this priority**: Pomodoros are a primary workflow but the TUI is still useful without them; users can fall back to `todo pom start` while the integration is being built.

**Independent Test**: With the TUI open and a task highlighted, press `S` and confirm the pomodoro starts using the same flow as `todo pom start`; when it ends, confirm the TUI returns to the task list. Press a digit `0`–`9` and confirm the highlighted task's estimate updates. Background a pomodoro from the CLI, re-enter the TUI, press `R`, and confirm it resumes.

**Acceptance Scenarios**:

1. **Given** a task is highlighted, **When** the user presses `S`, **Then** a pomodoro starts for that task using the same behavior as `todo pom start`, and on completion control returns to the task list (not to the shell).
2. **Given** a task is highlighted, **When** the user presses a digit key `0`–`9`, **Then** that task's pomodoro estimate is set to the digit's value.
3. **Given** a pomodoro was previously backgrounded, **When** the user presses `R` from the task list, **Then** the backgrounded pomodoro resumes inside the TUI.

---

### User Story 4 - Filtering, help, and exit (Priority: P2)

A user can toggle whether completed tasks are shown, open a help screen listing every keybinding, and quit the TUI cleanly.

**Why this priority**: These are quality-of-life shortcuts; the TUI works without them but feels incomplete without them. Help in particular reduces the learning curve significantly.

**Independent Test**: Launch the TUI, press `C` and confirm completed tasks appear (styled as struck-through and dimmed) and disappear when toggled off; press `?` and confirm a help overlay lists every shortcut from this spec; press `Q` and confirm the program exits cleanly back to the shell.

**Acceptance Scenarios**:

1. **Given** the TUI is open with the default view (incomplete tasks only), **When** the user presses `C`, **Then** completed tasks become visible in the tree with strikethrough and lighter color.
2. **Given** completed tasks are currently shown, **When** the user presses `C` again, **Then** completed tasks are hidden and only incomplete tasks remain.
3. **Given** the TUI is open, **When** the user presses `?`, **Then** a help screen lists every keyboard shortcut from this spec, and dismissing it returns to the task list with the same task highlighted.
4. **Given** the TUI is open and no modal/form is active, **When** the user presses `Q`, **Then** the program exits and the user returns to their shell.

---

### Edge Cases

- **No tasks exist**: The TUI launches with an empty task list and an empty details pane; `Ctrl-N` is still available to create the first task.
- **Completed task highlighted while filter hides completed tasks**: After `Space` completes the highlighted task in incomplete-only mode, the task stays visible with completed styling until the highlight moves; when it moves, the just-completed task is removed from the list.
- **Deleting the only task / last visible task**: After `Ctrl-D` the tree may be empty; the details pane clears and only global shortcuts (`Ctrl-N`, `C`, `?`, `Q`, `R`) are meaningful.
- **Deleting a task with subtasks**: The subtree is removed with the parent (consistent with however `todo task delete` already behaves on the server).
- **Pressing `H` is ambiguous in the source request** (both "collapse subtree" and "show help"). This spec resolves the conflict by binding help to `?` (common TUI convention) and reserving `H` for the collapse action, matching the `H`/`L` vim-direction pair.
- **Non-TTY invocation**: If `todo` is run with no arguments and stdout is not a TTY (piped, redirected, or in a script), the program should not attempt to render a TUI — it should either print usage / the current CLI help, or exit with a clear message. The TUI is interactive-only.
- **Pomodoro digit keys while filter or other state is active**: Digits `0`–`9` always set the highlighted task's estimate; they are not used for any other purpose in the task list view.
- **Terminal resize**: The two-pane layout reflows; the highlight and expansion state are preserved.

## Requirements *(mandatory)*

### Functional Requirements

#### Launch and layout

- **FR-001**: Running the `todo` CLI with no subcommand and an interactive terminal MUST launch the TUI.
- **FR-002**: Existing subcommands (e.g. `todo task ...`, `todo pom ...`, `todo plan ...`) MUST continue to behave exactly as today; only the no-argument invocation changes.
- **FR-003**: The TUI MUST display a two-pane layout: a task tree on the left and a details pane on the right.
- **FR-004**: On launch, the TUI MUST show incomplete tasks only, with all subtask branches collapsed, and MUST highlight the first task.
- **FR-004a**: Root tasks and sibling subtasks MUST be displayed in the same order used by the existing `todo task list` CLI, so the TUI is visually consistent with the existing CLI output.

#### Task tree rendering

- **FR-005**: Each task in the tree MUST display only its name (the details pane carries the rest of the fields).
- **FR-006**: Each task MUST be prefixed with a marker indicating subtree state: `[+]` when the task has subtasks and the branch is collapsed, `[-]` when subtasks are expanded *or* when the task has no subtasks.
- **FR-007**: Subtasks MUST be drawn using the same tree-line style as the existing `todo task` CLI output (`├──`, `└──`, indentation).
- **FR-008**: Completed tasks, when shown, MUST be rendered with strikethrough text in a lighter color, matching the existing `todo task` completed-task styling.

#### Details pane

- **FR-009**: The details pane MUST always reflect the currently highlighted task and MUST update whenever the highlight changes.
- **FR-010**: The details pane MUST display the task's id, name, due date, description, and the remaining fields surfaced by the existing `todo task` detail view.

#### Navigation (global keys)

- **FR-011**: `Down` arrow and `J` MUST move the highlight to the next *visible* task (skipping tasks hidden inside collapsed branches).
- **FR-012**: `Up` arrow and `K` MUST move the highlight to the previous visible task with the same skipping rule.
- **FR-013**: `?` MUST open a help screen listing every shortcut defined in this spec. Either `?` (toggle) or `Esc` MUST dismiss it, returning to the task list with the prior highlight intact.
- **FR-014**: `Ctrl-N` MUST open a blank edit form for a new root-level task.
- **FR-015**: `C` MUST toggle the list filter between "incomplete tasks only" and "all tasks (including completed)". After the toggle, the highlight MUST remain on the same task if it is still visible; otherwise it MUST fall back to the first visible task.
- **FR-016**: `Q` MUST exit the TUI cleanly when no modal or form is focused.
- **FR-017**: `R` MUST resume a previously backgrounded pomodoro, using the same behavior as the existing pomodoro resume flow.
- **FR-017a**: `Ctrl-R` MUST trigger a full reload of the task tree from the server, preserving the current highlight if the task still exists (otherwise falling back to the first visible task).

#### Highlighted-task actions

- **FR-018**: `Left` arrow and `H` MUST collapse the highlighted task's subtree.
- **FR-019**: `Right` arrow and `L` MUST expand the highlighted task's subtree.
- **FR-020**: `E` MUST move focus to the details pane in edit mode, pre-filled with the highlighted task's current values; `Tab` MUST cycle focus through the editable fields; `Save` and `Cancel` buttons MUST sit at the bottom of the form, and activating either MUST return focus to the task list. Additionally, `Esc` MUST cancel the form from any field, and `Ctrl-S` MUST save the form from any field, without requiring the user to tab to the buttons.
- **FR-021**: On `Save` from edit mode, the changes MUST persist to the server and the same task MUST remain highlighted.
- **FR-022**: On `Cancel` from edit mode, no changes MUST be persisted and the same task MUST remain highlighted.
- **FR-023**: `S` MUST start a pomodoro on the highlighted task with the same behavior as `todo pom start`, except that on completion the TUI MUST return to the task list instead of exiting the program.
- **FR-024**: Pressing any digit `0`–`9` while a task is highlighted MUST set that task's pomodoro estimate to the digit's value.
- **FR-025**: `Ctrl-D` MUST delete the highlighted task immediately, with no confirmation prompt.
- **FR-026**: After `Ctrl-D`, the highlight MUST move to the next visible task; if there is no next visible task, it MUST move to the previous visible task.
- **FR-027**: `N` MUST open a blank edit form for a new subtask of the highlighted task; on `Save` the new subtask MUST be created, its parent's subtree MUST be expanded if needed, and the new subtask MUST become the highlighted task; on `Cancel` no subtask MUST be created and the originally highlighted task MUST remain highlighted.
- **FR-028**: `Space` MUST toggle the completion state of the highlighted task. When the current filter is "incomplete only" and the task was just completed, the task MUST remain visible with completed styling until the highlight moves to a different task, after which it MUST be removed from the list.

#### Persistence and consistency

- **FR-029**: All mutations performed in the TUI (edits, creates, deletes, completion toggles, estimate changes, pomodoro records) MUST use the same server APIs that the existing CLI subcommands use, so behavior and authorization are identical.
- **FR-030**: After any mutation, the visible tree MUST reflect the new server state without requiring the user to restart the TUI.
- **FR-031**: The TUI MUST NOT auto-poll the server for changes. External modifications made by other clients during a session are accepted as stale until the next mutation triggers a re-fetch, or until the user presses `Ctrl-R` for a full reload.

### Key Entities

- **Task**: Same entity used by the existing CLI — id, name, description, due date, completion state, pomodoro estimate, parent task. Tasks form a tree via parent/child relationships.
- **Pomodoro session**: Same entity used by `todo pom` — has a target task, a duration, and a running/backgrounded/completed state.
- **TUI session state** (in-memory only, not persisted): which task is highlighted, which subtrees are expanded, whether the completed-task filter is on, and the in-list "just-completed but still showing" status of any single task.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user with an existing task tree can launch the TUI and reach any task (including a 3-level-deep subtask) using only the keyboard in under 5 seconds.
- **SC-002**: After learning the shortcuts (or consulting the help screen once), a user can perform every action listed in this spec — edit, create root task, create subtask, delete, toggle completion, start pomodoro, set estimate, toggle filter, resume pomodoro — without leaving the TUI.
- **SC-003**: Every keystroke that causes a visible change (navigation, expand/collapse, completion toggle, filter toggle) produces a redraw fast enough that the user perceives it as immediate (no perceptible lag on a local-network server).
- **SC-004**: A user familiar with the existing CLI (`todo task`, `todo pom`) can identify what each task tree row represents at a glance, because completed-task styling and tree-line characters match the existing CLI output exactly.
- **SC-005**: 100% of the keybindings listed in this spec are discoverable via the in-app help screen.

## Assumptions

- The TUI replaces the no-argument invocation only. Today, `todo` with no arguments prints a help banner; that behavior changes to launching the TUI. All other subcommands keep working as before.
- The TUI is interactive-only: when stdout is not a TTY, `todo` with no arguments falls back to printing the current CLI help (or exiting with a clear message) rather than rendering an unusable TUI.
- The conflict in the source request — `H` listed for both "collapse subtree" and "help" — is resolved by binding help to `?` (a near-universal TUI convention) and keeping `H` as the collapse key alongside `Left`. The vim-style `H`/`L` pair for collapse/expand is preserved.
- The pomodoro `S` action uses the existing `todo pom start` behavior, including whatever backgrounding / completion notification logic that command already implements. The TUI only changes what happens after completion (return to list instead of exit).
- The digit-key behavior (`0`–`9` sets estimate) is mutually exclusive with any other digit-based interaction in the task list; if "vim-style count prefixes" are added later, they will be designed around this binding.
- `Ctrl-D` deletes without confirmation per the user's explicit request; an undo facility is out of scope for this feature.
- All TUI operations go through the existing ConnectRPC handlers; no new persistence layer is introduced.
- The detail pane shows the same fields the existing CLI surfaces for a single task; no new task fields are introduced by this feature.

## Dependencies

- Existing ConnectRPC handlers for task CRUD, pomodoro start/resume, and completion toggling must remain backward-compatible.
- A terminal UI library suitable for Go (selection deferred to `/speckit-plan`) — out of scope for this spec.
