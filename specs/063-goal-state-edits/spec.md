# Feature Specification: Goal State Edits

**Feature Branch**: `063-goal-state-edits`

**Created**: 2026-07-12

**Status**: Draft

**Input**: User description: "goal states updates — Goal states need a few tweaks. First, we need a new status called Hold. This is for goals which the user still wants to pursue, but isn't working on right now. Goals on hold should not be shown by default. Second, there are lots of keys to remember to change the state of a goal, and users aren't changing states so often that hotkeys are even necessary. So, let's drop all the state change keys except to complete a goal. Currently, `d` marks a goal as complete, but it should be changed to `Space` (to match tasks). Then we need a State option in the edit form to switch between the goal states."

## Clarifications

### Session 2026-07-12

- Q: When Space is pressed on a goal that is already Completed, what should happen? → A: Space toggles like a task — completing sets Completed, and pressing Space again returns the goal to Committed.

### Session 2026-07-27

- Q: Should Hold goals still be hidden from the default goals view? → A: Reversed — Hold goals are now shown by default (grouped below Incubating), same as any active goal. Only Completed and Archived remain hidden behind "show all".

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Change a goal's state from the edit form (Priority: P1)

Today, changing a goal's state means remembering a different single-letter key for each state (one for incubating, one for committing, one for archiving, and so on). A user rarely changes a goal's state, so these keys are hard to recall and easy to press by mistake. Instead, the user opens the goal in the edit form — the same place they change the goal's other details — and picks the state from a labeled **State** option, then saves. There is now one obvious place to manage a goal's state, and no scattered hotkeys to memorize.

**Why this priority**: This is the core of the change. State changes are infrequent, so moving them into the edit form (and removing the per-state hotkeys) is what makes the goals tab simpler to use. It is also the only way to set most states once the hotkeys are gone, so nothing else in this feature is usable without it.

**Independent Test**: Can be fully tested by opening a goal's edit form, changing the State option to a different value, saving, and confirming the goal now shows that state — with no reliance on the Hold state or on the complete key.

**Acceptance Scenarios**:

1. **Given** a goal is selected, **When** the user opens the edit form, **Then** the form shows a State option set to the goal's current state.
2. **Given** the edit form is open, **When** the user changes the State option to another value and saves, **Then** the goal's state is updated to the chosen value and the change is reflected immediately in the goals list.
3. **Given** the edit form is open, **When** the user views the State option, **Then** every available state (Incubating, Committed, Hold, Completed, Archived) can be selected.
4. **Given** the edit form is open, **When** the user leaves the State option unchanged and saves other edits, **Then** the goal keeps its existing state.
5. **Given** the user is on the Goals tab, **When** they press one of the former state-change keys (the keys previously used for incubating, committing, or archiving), **Then** the goal's state does not change.

---

### User Story 2 - Put a goal on hold so it's out of sight but not gone (Priority: P2)

A user has a goal they still care about — "learn to sail" — but they are not working on it right now and don't want it mixed in with goals that are actively being worked on. They set the goal to the new **Hold** state. It stays visible in the default goals view — the list isn't just Committed and Incubating — but it's grouped in its own **Hold** section below Incubating, so it reads as parked rather than active. Completed and Archived goals are the ones that disappear from the default view; the user reaches those with the existing "show all" toggle.

**Why this priority**: Hold is the new capability users asked for, but it builds on the ability to set state (US1). It delivers real value on its own — a place to park goals without losing sight of them — once state setting exists.

**Independent Test**: Can be tested by setting a goal to Hold and confirming it remains visible in the default goals view under its own Hold group, positioned below Incubating and above the finished states.

**Acceptance Scenarios**:

1. **Given** a goal is set to Hold, **When** the user views the default goals view, **Then** the goal is shown, grouped under a Hold header below Incubating.
2. **Given** a goal is set to Hold, **When** the user turns on "show all", **Then** the on-hold goal is still shown in its Hold group (show all only affects Completed/Archived visibility).
3. **Given** the default view is active with an on-hold goal visible, **When** the user turns "show all" on and back off, **Then** the on-hold goal remains visible throughout.
4. **Given** a visible goal, **When** the user changes its state to Hold and saves, **Then** the goal moves into the Hold group immediately, still in the default view.
5. **Given** an on-hold goal, **When** the user changes its state back to an active state (e.g., Committed), **Then** the goal moves out of the Hold group and into the appropriate active group.

---

### User Story 3 - Complete a goal with the same key used for tasks (Priority: P3)

A user finishes a goal and wants to mark it done. Rather than recalling a goal-specific key, they press **Space** — exactly the key they already use to complete a task — while the goal is selected, and the goal is marked complete. The muscle memory carries over between tasks and goals.

**Why this priority**: Completing goals is common enough to keep a dedicated key, and aligning it with the task key removes the last piece of goal-specific key trivia. It is a refinement on top of the state model, so it comes after Hold.

**Independent Test**: Can be tested by selecting a goal, pressing Space, and confirming it becomes Completed; then pressing Space again and confirming it returns to an active state — matching how Space toggles a task.

**Acceptance Scenarios**:

1. **Given** an active (non-completed) goal is selected, **When** the user presses Space, **Then** the goal is marked Completed.
2. **Given** a Completed goal is selected, **When** the user presses Space, **Then** the goal returns to the Committed state.
3. **Given** the Goals tab is shown, **When** the user consults the keybinding hints, **Then** Space is shown as the complete key and the former complete key (`d`) is not advertised.

---

### Edge Cases

- A goal is put on Hold while the default view is active: it moves into the Hold group immediately, without leaving the default view.
- The only goals are on hold (all others are completed/archived) and "show all" is off: the goals view shows the Hold group; the normal empty state only applies when there are no goals at all in the default-visible states.
- A user completes an on-hold goal with Space: it becomes Completed (leaving Hold), consistent with completing a goal in any other state.
- Existing goals created before this change: their current states are preserved and none is automatically moved to Hold.
- The former state-change keys (incubate, commit, archive, and the old done key `d`) are pressed out of habit: they have no effect on the goal's state.
- Non-state keys on the Goals tab (ranking, "show all" toggle, edit, delete, add/link task) continue to work unchanged.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST support a "Hold" goal state, representing a goal the user still intends to pursue but is not actively working on right now.
- **FR-002**: Goals in the Hold state MUST be shown in the default goals view, grouped separately from Completed and Archived goals (which remain hidden by default).
- **FR-003**: The existing "show all" toggle MUST continue to reveal only Completed and Archived goals; it has no effect on Hold visibility, since Hold is always shown.
- **FR-004**: On-hold goals MUST be grouped and labeled by their Hold state consistently with the other states, positioned after the actively-pursued states and before the finished (Completed/Archived) states.
- **FR-005**: The goal edit form MUST include a State option that lets the user select any available state (Incubating, Committed, Hold, Completed, Archived), defaulting to the goal's current state.
- **FR-006**: Changing the State option in the edit form and saving MUST update the goal's state, and the change MUST be reflected immediately in the goals list (including moving the goal into or out of the Hold group, or into/out of the hidden Completed/Archived groups, as appropriate).
- **FR-007**: The dedicated per-state hotkeys on the Goals tab other than complete — the keys previously used to set Incubating, Committed, and Archived, and the former Done key — MUST be removed and MUST no longer change a goal's state.
- **FR-008**: Users MUST be able to complete the selected goal by pressing Space, consistent with the key used to complete a task.
- **FR-009**: Pressing Space on an already-completed goal MUST return it to the Committed state, mirroring the complete/uncomplete toggle behavior used for tasks.
- **FR-010**: Contextual keybinding hints on the Goals tab MUST reflect the removed hotkeys and the new Space-to-complete binding, and MUST not advertise removed keys.
- **FR-011**: Existing goals and their current states MUST be unaffected by this change; no goal is automatically moved to Hold.
- **FR-012**: A goal's state, including Hold, MUST persist across sessions.
- **FR-013**: Non-state Goals-tab keys (ranking, "show all" toggle, edit, delete, add/link task) MUST continue to function unchanged.

### Key Entities *(include if feature involves data)*

- **Goal** *(existing)*: Gains one additional possible state value, **Hold** (a goal actively intended but not currently being worked on), joining the existing Incubating, Committed, Completed, and Archived states. A goal's state becomes settable through the edit form's State option. Unlike Completed and Archived, Hold is visible in the default goals view — it gets its own group, positioned below Incubating. All other goal attributes are unchanged.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can move a goal to Hold and it stays in the default goals view, grouped under its own Hold header below Incubating.
- **SC-002**: A user can change a goal to any of its states using only the edit form — with no per-state hotkey involved.
- **SC-003**: The number of goal-state hotkeys a user must remember drops to exactly one (complete, via Space); all other state changes go through the edit form.
- **SC-004**: Pressing Space completes the selected goal, and pressing Space again returns it to an active state, using the same key that completes a task, in 100% of attempts.
- **SC-005**: After the change ships, every pre-existing goal retains the exact state it had before; no goal is silently reassigned (including to Hold).

## Assumptions

- **TUI scope**: The interaction changes (Hold visibility behavior, the edit-form State option, hotkey removal, Space-to-complete) apply to the TUI Goals tab, consistent with prior goal features. The CLI and web goal experiences are out of scope for these interaction changes; where they already display goal states, they treat Hold like any other state they already handle.
- **Space toggles completion**: Completing sets a goal to Completed; pressing Space on a Completed goal returns it to the Committed state, mirroring how Space toggles a task between complete and incomplete (confirmed in Clarifications, Session 2026-07-12).
- **"Show all" toggle unaffected by Hold**: The existing "show all" toggle continues to reveal only Completed and Archived goals; Hold is not gated by it (confirmed in Clarifications, Session 2026-07-27).
- **Non-state keys unchanged**: Ranking (`{`/`}`), the "show all" toggle (`c`), edit, delete, and add/link-task keys are unaffected — only the per-state change keys are removed.
- **State selector lists all states**: The edit-form State option offers all five states and starts on the goal's current state; it does not restrict which transitions are allowed.
- **No migration of existing goals**: Adding the Hold state does not move any existing goal; Hold is only ever entered explicitly by the user.
