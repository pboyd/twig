# Feature Specification: TUI Tree State Persistence

**Feature Branch**: `036-tui-tree-state-persistence`

**Created**: 2026-06-04

**Status**: Draft

**Input**: User description: "I'd like the TUI to remember the state of the tasks tree across restarts of the application (that is, which tasks have their sub-tasks expanded and which are collapsed)."

## Clarifications

### Session 2026-06-04

- Q: Where should the tree's expand/collapse state be stored? → A: Local only (per-machine; not synced through the server across devices/clients)
- Q: Should remembered state be separate per account/profile? → A: Per-profile (each profile/account remembers its own state)
- Q: How much TUI view state should be remembered? → A: Expand/collapse only (cursor/selection, active tab, and scroll position are out of scope)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Resume the tree exactly as I left it (Priority: P1)

A user works with a deep task tree in the TUI, expanding the branches they care about and collapsing the ones they don't, to focus their view. They quit the TUI (or it is closed) and later relaunch it. The tree reappears with the same branches expanded and collapsed as when they left, so they don't have to re-navigate and re-expand everything to get back to their working context.

**Why this priority**: This is the core value of the feature. Without it, every launch forces the user to manually rebuild their preferred view, which is the exact frustration the feature exists to remove. Delivering only this story is already a complete, useful feature.

**Independent Test**: Expand and collapse a known set of tasks in the TUI, quit, relaunch, and confirm the same tasks are expanded/collapsed as before.

**Acceptance Scenarios**:

1. **Given** a task with sub-tasks that the user has expanded, **When** the user quits and relaunches the TUI, **Then** that task is shown expanded with its sub-tasks visible.
2. **Given** a task with sub-tasks that the user has collapsed, **When** the user quits and relaunches the TUI, **Then** that task is shown collapsed with its sub-tasks hidden.
3. **Given** a mix of expanded and collapsed tasks across multiple levels of the tree, **When** the user relaunches the TUI, **Then** every task's expansion state matches the state at the time of the last session.
4. **Given** the user changes the expansion state during a session, **When** the user relaunches the TUI again, **Then** the most recently chosen state is reflected (not an older one).

---

### User Story 2 - Sensible behavior for tasks I haven't seen yet (Priority: P2)

A user relaunches the TUI after tasks have been added (by them on another client, or by themselves since the last session). These new tasks have no remembered expansion state. The TUI shows them using the standard default expansion behavior, without discarding or corrupting the remembered state of existing tasks.

**Why this priority**: The tree changes over time, so the feature must degrade gracefully when it encounters tasks it has no memory of. Important, but secondary to restoring known state.

**Independent Test**: With saved expansion state in place, add a new task with sub-tasks, relaunch, and confirm the new task uses the default state while previously remembered tasks keep their state.

**Acceptance Scenarios**:

1. **Given** saved tree state and a newly created task with sub-tasks that has never been seen, **When** the user launches the TUI, **Then** the new task is displayed with the default expansion state and all previously remembered tasks retain their saved state.
2. **Given** a remembered task that has since been deleted, **When** the user launches the TUI, **Then** the TUI displays the remaining tasks correctly and does not error because of the missing task.

---

### User Story 3 - State stays separate per account/profile (Priority: P3)

A user who uses more than one account (via the CLI's profile support) expects each profile to remember its own tree state. Switching profiles should not show another profile's expansion choices.

**Why this priority**: Keeps the feature correct for multi-profile users and avoids confusing cross-account bleed, but most users use a single account, so it is the lowest priority.

**Independent Test**: Curate distinct tree states under two profiles, relaunch each with `--profile`, and confirm each profile shows its own remembered state.

**Acceptance Scenarios**:

1. **Given** two profiles with different curated tree states, **When** the user launches the TUI under one profile, **Then** only that profile's remembered expansion state is applied.

---

### Edge Cases

- **Missing or unreadable state**: On first-ever launch, or when the stored state is missing, the TUI starts with default expansion behavior and creates state going forward — no error shown.
- **Corrupt or unparseable state**: If the stored state cannot be read/parsed, the TUI ignores it, falls back to default expansion, and recovers (overwrites with valid state) rather than crashing.
- **Deleted tasks**: Remembered entries for tasks that no longer exist are ignored and should not accumulate indefinitely.
- **Task gains sub-tasks**: A task that previously had no children but now does should honor any remembered expansion choice if present, otherwise use the default.
- **Task loses all sub-tasks**: A leaf task carries no meaningful expansion state; this should not cause errors.
- **Concurrent sessions**: If two TUI sessions run at once, the last one to save wins; this must not corrupt the stored state for future launches.
- **Abnormal exit**: State should reflect the user's most recent expansion choices even if the application is closed without a graceful quit, to the extent reasonably achievable.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The TUI MUST persist, across application restarts, which tasks in the tree are expanded and which are collapsed.
- **FR-002**: On launch, the TUI MUST restore each known task's expansion state to match the state recorded at the end of the previous session.
- **FR-003**: The TUI MUST persist expansion-state changes such that the most recent state from a session is the one restored on the next launch.
- **FR-004**: For any task with no recorded expansion state (e.g., newly created tasks), the TUI MUST apply the standard default expansion behavior.
- **FR-005**: The TUI MUST continue to function normally when the stored state is missing, empty, or unreadable, falling back to default expansion behavior without surfacing an error to the user.
- **FR-006**: The TUI MUST tolerate corrupt or invalid stored state by ignoring it and recovering on the next save, without crashing or blocking startup.
- **FR-007**: The TUI MUST ignore recorded state for tasks that no longer exist and MUST avoid unbounded growth of stored state over time (stale entries are pruned).
- **FR-008**: Restoring saved state MUST NOT alter the underlying tasks themselves (it changes only the local view, not task data on the server).
- **FR-009**: Persisted tree state MUST be scoped per account/profile so that distinct profiles do not share or overwrite each other's expansion state.
- **FR-010**: Persistence MUST be local to the user's environment and require no manual action by the user to save or restore state.

### Key Entities *(include if data involved)*

- **Tree View State**: The remembered set of expansion choices for the task tree. Conceptually a mapping from a task's identity to whether its sub-tasks are expanded or collapsed, scoped to a single account/profile. It is view-only metadata and is not part of the task data stored on the server.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After curating the tree and relaunching, 100% of tasks that existed in both sessions display the same expansion state as at the end of the previous session.
- **SC-002**: A returning user reaches their previous working view with zero manual expand/collapse actions when no tasks have changed.
- **SC-003**: The TUI launches successfully and shows the tree in 100% of cases where stored state is missing or corrupt, with no visible error.
- **SC-004**: Stored state does not grow without bound: entries for deleted tasks do not persist indefinitely across sessions.
- **SC-005**: Restoring state adds no perceptible delay to TUI startup (startup remains effectively instantaneous from the user's perspective).

## Assumptions

- **Local persistence**: "Across restarts" refers to restarts on the same environment, so expansion state is stored locally on the user's machine rather than synced through the server across devices. Expansion state is treated as client-side view preference, consistent with how the existing config is stored locally.
- **Per-profile scoping**: Because the CLI already supports multiple accounts via profiles, remembered tree state is keyed per profile/account so accounts don't interfere with one another.
- **Default expansion behavior**: The "default" applied to unknown tasks is whatever expansion state the TUI currently shows for tasks on a fresh launch today; this feature does not change that default, only remembers deviations from it.
- **Tasks identified by stable IDs**: Tasks have stable identifiers that can be used to associate remembered expansion state with the correct task across sessions.
- **Scope limited to the tasks tree**: Only the expand/collapse state of the tasks tree is remembered. Other UI state (selection/cursor position, active tab, scroll position, filters) is out of scope unless specified later.
