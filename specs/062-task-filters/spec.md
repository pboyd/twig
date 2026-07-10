# Feature Specification: Task Filters

**Feature Branch**: `062-task-filters`

**Created**: 2026-07-10

**Status**: Draft

**Input**: User description: "task filters — As users add more tasks they have an increasingly hard time finding the one they're looking for. We need to add filters. In the TUI, filters should be accessed with `/` and progressively search as the user types. If a child task matches, then its parent should also be shown. TUI-only for now, but implemented server-side so the Web app can use it later. The filter format should be expressive and similar to SQL (text search, completed/snoozed/parent_id/goal_id conditions, date comparisons, transitive `^` matching). The interaction with the existing 'show all' mode is likely to be a bit thorny."

## Clarifications

### Session 2026-07-10

- Q: With "show all" off and a filter of `completed=true` (no `snoozed` condition), should completed tasks that are also snoozed appear? → A: No — the override is per-attribute. An explicit `completed` condition replaces only the completion default; snoozed tasks stay hidden unless the expression also mentions `snoozed` (or "show all" is on).
- Q: After `Enter` applies a filter and focus returns to the list, how does the user modify or remove it? → A: `/` reopens the filter input pre-filled with the current expression for editing; `Esc` pressed in the list clears the active filter.
- Q: Can a free-text term be combined with attribute conditions in one expression? → A: Yes — a text term is a valid `AND` operand, e.g. `groceries AND completed=false AND ^goal_id=2`.
- Q: When a parent task matches, are its non-matching subtasks shown? → A: No — the view shows only matching tasks plus their ancestor chain; non-matching descendants of a match stay hidden.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find a task by typing text (Priority: P1)

A user with hundreds of tasks presses `/` in the Tasks tab, types a few characters, and watches the task list narrow with every keystroke to only the tasks whose text contains what they typed. Matching tasks keep their place in the tree: when a subtask matches, its parent chain stays visible so the user understands where the match lives.

**Why this priority**: Text search is the single interaction that solves the stated problem — "users have an increasingly hard time finding the task they're looking for." It delivers value on its own even if no structured filtering ever ships.

**Independent Test**: Can be fully tested by creating a nested set of tasks, pressing `/`, typing a term that matches only a deeply nested subtask, and confirming that the subtask and all of its ancestors are the only tasks shown.

**Acceptance Scenarios**:

1. **Given** a task list containing a task titled "Buy groceries", **When** the user presses `/` and types `groc`, **Then** the list shows "Buy groceries" and hides non-matching tasks — updating as each character is typed.
2. **Given** a subtask "Order flowers" nested two levels below "Plan wedding", **When** the user filters for `flowers`, **Then** "Order flowers" is shown along with both of its ancestors, preserving the tree structure.
3. **Given** "show all" is **off** and a completed task contains the text "foo", **When** the user filters for `foo`, **Then** the completed task is *not* shown (only incomplete, unsnoozed matches appear).
4. **Given** "show all" is **on** and a completed task contains the text "foo", **When** the user filters for `foo`, **Then** the completed task *is* shown alongside incomplete and snoozed matches.
5. **Given** an active filter, **When** the user presses `Enter`, **Then** focus returns to the task list with the filter still applied, and normal navigation/shortcut keys work against the filtered list.
6. **Given** an active filter, **When** the user clears it (e.g. presses `Esc` in the filter input), **Then** the full task list is restored.

---

### User Story 2 - Filter by task attributes (Priority: P2)

A power user types a structured, SQL-like expression into the same `/` filter input to slice the list by task attributes: completion status, snooze status, completion date, direct parent, or goal — combining conditions with `AND`.

**Why this priority**: Attribute filters answer questions text search cannot ("what did I complete before January?", "which direct subtasks of task 12 are still open?"). They build on the P1 input mechanism and matching pipeline.

**Independent Test**: Can be tested by entering each documented expression form and confirming the returned set matches a hand-computed expected set.

**Acceptance Scenarios**:

1. **Given** task 1 has both completed and incomplete direct subtasks, **When** the user filters with `completed=true AND parent_id=1`, **Then** only the completed direct subtasks of task 1 are shown (task 1 itself appears as ancestor context).
2. **Given** tasks completed on various dates, **When** the user filters with `completed < 2026-01-01`, **Then** only tasks completed before 1 Jan 2026 are shown.
3. **Given** a mix of snoozed and unsnoozed tasks, **When** the user filters with `snoozed=true`, **Then** only currently snoozed tasks are shown — even though "show all" is off — because the expression explicitly asks for them.
4. **Given** "show all" is **off**, **When** the user filters with `parent_id=1` (no completion or snooze condition), **Then** only *incomplete, unsnoozed* direct subtasks of task 1 are shown, because the expression doesn't override the default visibility.
5. **Given** a half-typed or malformed expression (e.g. `completed=`), **When** the user pauses, **Then** the list retains the last valid result set and the input visibly indicates the expression is not yet valid — the list never crashes or goes blank mid-typing.

---

### User Story 3 - Filter across an entire subtree or goal (Priority: P3)

A user prefixes a relationship field with `^` to match transitively: `^parent_id=1` matches every descendant of task 1 (not just direct children), and `^goal_id=1` matches every task anywhere in the tree of any task attached to goal 1.

**Why this priority**: Subtree/goal scoping is the most powerful query form but depends on the structured-filter foundation from P2. It converts the filter from "find a task" into "focus my whole view on one project."

**Independent Test**: Can be tested by building a three-level task tree under one goal and confirming `^parent_id` and `^goal_id` expressions return the full subtree while their non-`^` counterparts return only direct children.

**Acceptance Scenarios**:

1. **Given** task 1 has a child task 2, which has a child task 3, **When** the user filters with `completed=false AND ^parent_id=1`, **Then** incomplete tasks 2 *and* 3 are shown; with `parent_id=1` only task 2 would match.
2. **Given** goal 1 has tasks with nested subtasks, **When** the user filters with `completed=false AND snoozed=false AND ^goal_id=1`, **Then** the whole incomplete, unsnoozed tree under goal 1 is shown, not just top-level tasks.

---

### Edge Cases

- **No matches**: the filtered list is empty — the TUI shows a clear "no tasks match this filter" state rather than an empty screen.
- **Nonexistent ID**: `parent_id=99999` where no such task exists returns an empty result, not an error.
- **Invalid date**: `completed < 2026-13-45` is treated as an invalid expression (last valid results retained, invalid indicator shown).
- **Reserved words in text**: a user searching for literal text that collides with the expression syntax (e.g. a task titled "AND review") can quote the term (`"AND review"`) to force text matching.
- **Task changes while filtered**: completing, snoozing, or editing a task while a filter is active re-evaluates the filter; a task that no longer matches disappears from the view.
- **Filter + other shortcuts**: while the filter input has focus, printable keys go to the input, not to list shortcuts (`c`, `u`, etc.); after `Enter` returns focus to the list, shortcuts work normally against the filtered view.
- **Toggling "show all" while a filter is active**: the filter is re-evaluated under the new default visibility (explicit completion/snooze conditions in the expression continue to override).
- **Ancestor context rows**: ancestors shown only for context (they don't themselves match) are still real tasks — selecting and acting on them behaves normally.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Pressing `/` in the TUI Tasks tab MUST open a filter input; typing MUST progressively re-filter the visible task list on each change without requiring the user to press Enter.
- **FR-002**: A bare text term MUST match tasks whose user-visible text fields (title and notes/description) contain the term, case-insensitively. Terms containing spaces or reserved syntax MUST be expressible via quoting.
- **FR-003**: When a task matches the filter, all of its ancestor tasks MUST also be displayed, preserving tree structure and indentation, even if the ancestors do not themselves match. The reverse does not hold: non-matching descendants of a matching task MUST stay hidden — the view shows only matches plus their ancestor chains.
- **FR-004**: The filter language MUST support conditions on at least these attributes: `completed` (boolean equality, or date comparison against the completion date), `snoozed` (boolean equality), `parent_id` (task ID equality), and `goal_id` (goal ID equality).
- **FR-005**: The filter language MUST support the comparison operators `=` and `!=` for boolean and ID values, and `=`, `!=`, `<`, `<=`, `>`, `>=` for date values (dates written as `YYYY-MM-DD`).
- **FR-006**: The filter language MUST support combining multiple conditions with `AND`. A free-text term is itself a valid `AND` operand, so text search and attribute conditions can be mixed in one expression (e.g. `groceries AND completed=false AND ^goal_id=2`).
- **FR-007**: Prefixing `parent_id` or `goal_id` with `^` MUST make the condition transitive: `^parent_id=N` matches all descendants of task N at any depth; `^goal_id=N` matches all tasks in the tree of any task belonging to goal N.
- **FR-008**: Default visibility ("show all" interaction): when a filter expression contains **no** explicit `completed` or `snoozed` condition, the current "show all" toggle governs — off means only incomplete, unsnoozed tasks are eligible to match; on means all tasks are eligible. When the expression **does** contain an explicit `completed` or `snoozed` condition, that condition overrides the toggle's default **for that attribute only** — e.g. with the toggle off, `completed=true` shows completed tasks but still excludes snoozed ones, because the expression doesn't mention `snoozed`.
- **FR-009**: While the typed expression is syntactically invalid or incomplete, the TUI MUST retain the most recent valid result set, MUST visibly indicate the expression is invalid, and MUST NOT crash, blank the list, or surface a raw error.
- **FR-010**: Pressing `Enter` in the filter input MUST apply the filter and return focus to the task list (filter remains active and visible); pressing `Esc` in the filter input MUST clear the filter and restore the unfiltered view. While a filter is applied and the list has focus, pressing `/` MUST reopen the filter input pre-filled with the current expression for editing, and pressing `Esc` MUST clear the active filter.
- **FR-011**: Filter evaluation MUST be performed by the backend service (not solely inside the TUI client) so that future clients — specifically the web app — can reuse the same filtering capability with identical semantics.
- **FR-012**: A filter that matches nothing MUST produce a distinct empty state, and a filter referencing a nonexistent task or goal ID MUST return an empty result rather than an error.
- **FR-013**: Task mutations performed while a filter is active (complete, snooze, edit, create) MUST cause the filtered view to re-evaluate so it stays consistent with the filter expression.
- **FR-014**: An active filter MUST be visually indicated (the expression remains visible) so the user always knows the list is filtered.

### Key Entities

- **Task**: the item being filtered. Relevant attributes: title, notes/description (text-searchable), completion state and completion date, snooze state, parent task (tree position), and associated goal.
- **Goal**: a long-term objective that tasks belong to; used as a filter scope via `goal_id` / `^goal_id`.
- **Filter Expression**: user-entered query composed of a free-text term and/or attribute conditions joined by `AND`; each condition names an attribute, an operator, and a value, optionally marked transitive with `^`.
- **Show-all Toggle**: existing per-session visibility mode (`c` key) that determines the *default* eligibility of completed/snoozed tasks when a filter doesn't explicitly constrain them.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can locate a specific task in a list of 500+ tasks in under 10 seconds using text search (press `/`, type, spot the task).
- **SC-002**: The visible list updates within 300 ms of each keystroke for task lists up to 1,000 tasks, so filtering feels instantaneous.
- **SC-003**: Every documented expression form (bare text, boolean equality, ID equality, date comparison, `AND` combination, `^` transitive matching) returns exactly the hand-computed expected task set in acceptance testing — 100% of the example expressions in this spec behave as described.
- **SC-004**: No sequence of keystrokes in the filter input — including malformed, incomplete, or nonsensical expressions — crashes the TUI or leaves the task list in a blank/unrecoverable state.
- **SC-005**: The same filter expression evaluated on behalf of any client yields the same result set, verified by exercising the filtering capability directly against the backend without the TUI.

## Assumptions

- The filter applies to the **Tasks tab only**; Goals, Plan, and Report tabs are unchanged.
- Bare text is matched as a single case-insensitive substring (including any spaces), not as multiple independent words; quoting is only needed when the text collides with expression syntax.
- Only `AND` conjunction is supported in this version; `OR`, `NOT`, and parenthesized grouping are deferred (see Out of Scope).
- `snoozed` is a boolean condition ("currently snoozed or not"); comparing against the snooze-until date is not required in this version.
- `completed` used with a date operator compares the completion timestamp by calendar date; `completed < 2026-01-01` means "completed before 1 Jan 2026" and implies the task is completed (so it also satisfies the explicit-condition rule in FR-008).
- The filter is session-scoped: it does not persist across TUI restarts, and there is no saved/named filter facility in this version.
- Ancestors displayed purely as context are rendered as normal task rows (no special dimming required, though an implementation may choose to add it).
- The exact surface syntax may be adjusted during design (per the request, "the exact syntax used here is not especially important") as long as every capability expressed in the examples remains expressible.
- Server-side evaluation is a deliberate constraint requested so the web app can adopt filtering later; the web UI itself is out of scope now.

## Out of Scope

- Web app filter UI (the server-side capability must support it later, but no web UI work now).
- CLI (non-TUI) filter flags or commands.
- `OR` / `NOT` operators and parenthesized grouping.
- Saved, named, or persistent filters.
- Sorting or ordering changes to the filtered results.
- Filtering by attributes not listed in FR-004 (e.g. creation date, scheduled/plan date, snooze-until date).
