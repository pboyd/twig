# Feature Specification: Goal Handling When Moving Tasks

**Feature Branch**: `072-move-task-goal-rules`

**Created**: 2026-08-01

**Status**: Draft

**Input**: User description: "clean up the logic around moving a task with a goal — moving a task with a goal under another task with a goal is blocked even when the goals match; moving a task with a goal under a task without a goal leaves a sub-task holding a goal; moving a sub-task out to the top level loses the goal it inherited from its parent."

## Context

Twig lets a person attach a long-term goal to a task, and tasks nest into a tree. The product rule is that **only top-level tasks carry a goal association**; a nested task shows the goal of its nearest goal-bearing ancestor. Moving a task around the tree is where that rule currently breaks down in three different ways: one move is refused outright, one produces a state the rule forbids, and one silently throws away the goal the person could see on the task a moment earlier.

This feature makes moving a task always succeed and always leave the tree in a state that matches the rule, without asking the person to do goal bookkeeping first.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Move a goal-linked task under another task (Priority: P1)

Someone is reorganizing their task list and drags a top-level task that is linked to a goal underneath another task. Today this is either refused with an error telling them to unlink the goal first, or it succeeds and leaves a nested task still holding its own goal link. Either way, they have to think about goal plumbing in the middle of a reorganization.

After this change the move simply happens. The moved task gives up its own goal link, and from then on it shows whatever goal its new ancestors carry — which may be a goal, or none.

**Why this priority**: This is the case that produces a hard error today. It blocks a routine action, and the error text asks the person to perform a separate step that the system can perform for them. Fixing it alone removes the visible failure.

**Independent Test**: Link a goal to two separate top-level tasks, then move one under the other. The move succeeds, the moved task no longer holds its own goal link, and the tree shows it under the destination's goal.

**Acceptance Scenarios**:

1. **Given** top-level task A linked to goal "Fitness" and top-level task B linked to goal "Health", **When** the person moves A under B, **Then** the move succeeds, A no longer holds its own goal link, and A is shown as belonging to "Health".
2. **Given** top-level task A and top-level task B both linked to goal "Fitness", **When** the person moves A under B, **Then** the move succeeds with no error and A is shown as belonging to "Fitness".
3. **Given** top-level task A linked to goal "Fitness" and top-level task B with no goal, **When** the person moves A under B, **Then** the move succeeds, A no longer holds a goal link, and A is shown with no goal.
4. **Given** top-level task A linked to goal "Fitness" with sub-tasks beneath it, **When** the person moves A under goal-linked task B, **Then** A's sub-tasks move with it and none of them holds its own goal link.
5. **Given** any task being moved under any destination task, **When** the move is performed, **Then** the system never refuses the move for a goal-related reason.

---

### User Story 2 - Promote a sub-task to the top level (Priority: P1)

Someone has a sub-task sitting under a goal-linked parent. In the task list that sub-task visibly belongs to the parent's goal. They decide it deserves to stand on its own and move it to "no parent". Today it arrives at the top level with no goal at all, and the goal connection they could see a moment ago is gone with no warning.

After this change the sub-task keeps the goal it was showing: on promotion to the top level it is given its own link to the goal it had been inheriting.

**Why this priority**: This is silent data loss from the person's point of view. Nothing tells them the goal was dropped, so it is discovered later, if at all. It is as important to fix as the visible error in User Story 1.

**Independent Test**: Link a goal to a top-level task, add a sub-task under it, then move the sub-task to "no parent". The sub-task arrives at the top level linked to that same goal.

**Acceptance Scenarios**:

1. **Given** top-level task A linked to goal "Fitness" with sub-task B beneath it, **When** the person moves B to no parent, **Then** B becomes a top-level task linked to "Fitness".
2. **Given** top-level task A linked to goal "Fitness", sub-task B beneath it, and sub-sub-task C beneath B, **When** the person moves C to no parent, **Then** C becomes a top-level task linked to "Fitness" — the goal inherited from its nearest goal-bearing ancestor.
3. **Given** top-level task A with no goal and sub-task B beneath it, **When** the person moves B to no parent, **Then** B becomes a top-level task with no goal, and no error is raised.
4. **Given** top-level task A linked to goal "Fitness", sub-task B beneath it, and sub-tasks beneath B, **When** the person moves B to no parent, **Then** B is linked to "Fitness", its sub-tasks move with it, and they show "Fitness" through B without holding their own links.
5. **Given** a top-level task already linked to goal "Fitness", **When** the person moves it to no parent (a no-op move), **Then** it stays linked to "Fitness".

---

### User Story 3 - The rule holds after every move (Priority: P2)

Whatever combination of moves someone performs, the task tree afterwards should satisfy the product rule: goal links live only on top-level tasks, and every nested task's goal is the one its nearest goal-bearing ancestor carries. This covers pre-existing rows that already violate the rule — a nested task holding its own goal link from before this change gets cleaned up as soon as it or its ancestor is moved.

**Why this priority**: Users of Stories 1 and 2 get correct behavior for the moves they perform; this story is about the invariant holding across sequences of moves and over legacy data. It is verification-heavy rather than a distinct user-visible action.

**Independent Test**: Perform a series of moves that mix goal-linked and goal-free tasks at several depths, then inspect the resulting tree — no nested task holds its own goal link, and every task's displayed goal traces to a top-level ancestor.

**Acceptance Scenarios**:

1. **Given** a nested task that already holds its own goal link (pre-existing data from before this change), **When** it is moved under another task, **Then** its own goal link is cleared.
2. **Given** a task with a nested descendant that holds its own goal link, **When** that task is moved under another task, **Then** the descendant's own goal link is cleared as well.
3. **Given** any sequence of moves, **When** the resulting tree is inspected, **Then** no task that has a parent holds its own goal link.

---

### Edge Cases

- **The inherited goal is closed.** A sub-task is promoted to the top level and the goal it was inheriting has since been completed or archived. The promoted task is still linked to that goal — this preserves what the person could already see rather than silently dropping it, even though linking to a closed goal is not something they can newly request.
- **The destination is an unrelated part of the tree.** Moving a task under a destination that has no goal-bearing ancestor at any depth leaves the moved task with no goal, and this is not an error.
- **Moving between two positions under the same parent.** Reordering within a group is not a change of parent and must not touch any goal link.
- **A move that is rejected for a non-goal reason.** Moves are still refused when they would create a cycle, when the destination does not exist, or when the destination is already completed. In those cases no goal link changes.
- **Deeply nested subtrees.** A moved task may carry many levels of descendants; the goal rule must be applied to all of them, not only direct children.
- **Concurrent edits.** If the destination's goal link changes at the same moment as the move, the resulting tree still satisfies the rule — the moved task holds no goal link of its own, so it reflects whatever the ancestor ends up carrying.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow a task to be moved under any valid destination task regardless of the goal links held by the moved task, its descendants, the destination, or the destination's ancestors. No move may be refused for a goal-related reason.
- **FR-002**: When a task is moved so that it has a parent, the system MUST clear the moved task's own goal link, if it had one.
- **FR-003**: When a task is moved so that it has a parent, the system MUST clear the own goal links of every descendant of the moved task, at any depth.
- **FR-004**: When a task is moved to the top level (no parent) and it did not already hold its own goal link, the system MUST link it to the goal it was inheriting — the goal held by its nearest goal-bearing ancestor before the move.
- **FR-005**: When a task is moved to the top level and it was not inheriting any goal, the system MUST leave it with no goal link, and MUST NOT treat this as an error.
- **FR-006**: When a task that already holds its own goal link is moved to the top level, the system MUST leave that link unchanged.
- **FR-007**: The goal assigned on promotion to the top level MUST be applied even when that goal is completed or archived, so that no visible goal association is lost by the move.
- **FR-008**: Reordering a task among its existing siblings — a change of position without a change of parent — MUST NOT change any goal link.
- **FR-009**: When a move is refused for a non-goal reason (cycle, missing destination, completed destination), the system MUST leave all goal links unchanged.
- **FR-010**: The system MUST continue to refuse a direct request to link a goal to a task that has a parent, or to a task that has a goal-linked ancestor or descendant. This feature changes what moves do, not what explicit goal-linking requests are allowed.
- **FR-011**: All goal changes caused by a move MUST be applied together with the move itself, so no partially-applied state is ever observable — a failure at any point leaves both the tree and the goal links as they were.
- **FR-012**: The error message telling the person to "clear the goal link first" before moving MUST be removed, since that situation no longer arises.

### Key Entities

- **Task**: An item of work. Has a name, an optional parent task, a position among its siblings, and an optional goal link. The rule this feature enforces is that the goal link is populated only when the task has no parent.
- **Goal**: A long-term objective a person is working toward. May be open, completed, or archived. A goal is linked to zero or more top-level tasks.
- **Effective goal**: Not stored — derived. For a top-level task it is its own goal link; for a nested task it is the goal link of its nearest ancestor that has one, or none.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of task moves that were previously refused with the nested-goal error now succeed.
- **SC-002**: After any move, 0% of tasks that have a parent hold their own goal link.
- **SC-003**: Promoting a nested task to the top level preserves the goal it was displaying in 100% of cases where it was displaying one.
- **SC-004**: A person can reorganize their task tree without ever being asked to unlink a goal as a prerequisite — the number of manual goal-unlink steps required before a move drops to zero.
- **SC-005**: The goal shown against a task never changes without a corresponding move by the person; no move silently drops a visible goal.

## Assumptions

- The existing product rule — goal links belong only on top-level tasks, and nested tasks display the goal of their nearest goal-bearing ancestor — is correct and stays as-is. This feature makes moves respect that rule rather than changing it.
- Moving a task under a destination is understood as "the moved task now belongs to the destination's goal, if any." The moved task's previous goal is therefore not worth preserving, and clearing it is the right outcome rather than a loss. This matches the user's stated intent for the first two reported issues.
- Promotion to the top level is the reverse: the goal the task was displaying is worth keeping, so it is materialized as an explicit link.
- No one-time repair of existing data is in scope. Nested tasks that already hold their own goal link are corrected when they or an ancestor are next moved. The application already tolerates the inconsistent state in the meantime.
- The behavior applies uniformly wherever a move can be initiated — command line, terminal interface, and web — because the rule is enforced where the move is processed rather than in each interface.
- No new confirmation prompt or warning accompanies the automatic goal changes; the moves are reversible and the resulting goal is visible in the task tree immediately.
