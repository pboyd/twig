# Feature Specification: Reorder Untimed Plan Entries in the TUI

**Feature Branch**: `056-reorder-untimed-tasks`

**Created**: 2026-06-15

**Status**: Draft

**Input**: User description: "reorder untimed tasks in the TUI planning tab. Untimed plan entries (a misnomer, they have duration, but not a start time) have no user-specified order today. Users would like to reorder these tasks so they can work through them in a logical order. The `{` and `}` keys are used elsewhere in the TUI to change the order of tasks and goals, so those keys should be used for untimed tasks as well. For now, this ability should only be added to the TUI (not the CLI or web app)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reorder untimed entries in the TUI planning tab (Priority: P1)

A user reviewing a day's plan in the TUI has several untimed entries in the top-left untimed pane (tasks that must get done that day but have no committed start time). They want those entries arranged in the order they intend to tackle them. They highlight an untimed entry and press `{` to move it earlier (higher) in the untimed list, or `}` to move it later (lower). The list re-renders immediately with the entry in its new position, and the entry stays highlighted so they can keep nudging it.

**Why this priority**: This is the entire feature. Untimed entries currently appear in a fixed, system-determined order the user cannot influence, which makes the untimed pane an unordered pile rather than a worklist. Giving users direct keyboard control over that order, with the same `{`/`}` keys they already use to rank tasks and goals, delivers the complete value on its own.

**Independent Test**: Open the Planning view on a day with three or more untimed entries, highlight one, press `{` and `}`, and confirm it visibly changes position within the untimed pane relative to its neighbors while remaining highlighted.

**Acceptance Scenarios**:

1. **Given** untimed entries in order A, B, C with B highlighted, **When** the user presses `{`, **Then** the untimed order becomes B, A, C and B remains highlighted.
2. **Given** untimed entries in order A, B, C with B highlighted, **When** the user presses `}`, **Then** the untimed order becomes A, C, B and B remains highlighted.
3. **Given** the first untimed entry is highlighted, **When** the user presses `{`, **Then** the order is unchanged (it is already first), no error is shown, and the entry stays highlighted.
4. **Given** the last untimed entry is highlighted, **When** the user presses `}`, **Then** the order is unchanged (it is already last), no error is shown, and the entry stays highlighted.

---

### User Story 2 - Reordered untimed entries stay put (Priority: P2)

After arranging the day's untimed entries into a logical working order, the user expects that order to stick — when they refresh the Planning view, switch days and come back, or close and reopen the TUI, the untimed entries are still in the order they set, not reverted to the default.

**Why this priority**: A reorder that does not survive a refresh or restart is little better than no reorder at all; persistence is what turns the gesture into a durable worklist. It depends on US1 existing first, so it ranks just below it.

**Independent Test**: Reorder the untimed entries on a day, then refresh the view / navigate away and back / restart the TUI, and confirm the untimed entries appear in the user-defined order.

**Acceptance Scenarios**:

1. **Given** the user has reordered a day's untimed entries, **When** the Planning view re-renders or the day is re-opened, **Then** the untimed entries appear in the user-defined order.
2. **Given** the user has reordered a day's untimed entries, **When** the TUI is closed and reopened, **Then** the untimed entries appear in the user-defined order.

---

### Edge Cases

- **Already at boundary**: Ranking the first untimed entry higher or the last untimed entry lower is a no-op — the action is accepted without error and the highlight is preserved.
- **Single or empty untimed pane**: With zero or one untimed entry, `{`/`}` are no-ops (there is nothing to reorder, and the untimed pane is hidden entirely when empty).
- **Highlight on a timed/grid entry**: When the highlight is on a timed entry in the day grid (not an untimed entry), `{`/`}` do not reorder the untimed entries; ordering applies only to the untimed group. (Timed entries already have an implicit order via their start times.)
- **Untimed entry gains a start time**: When an untimed entry is given a start time (scheduled onto the grid), it leaves the untimed group; the remaining untimed entries keep their relative order.
- **Timed entry loses its start time**: When a timed entry's start time is cleared and it returns to the untimed pane, it joins the untimed group at a well-defined position (see Assumptions) without disturbing the relative order of the entries already there.
- **Newly added untimed entry**: A newly created untimed entry appears at a well-defined default position in the untimed group (see Assumptions); existing untimed entries keep their relative order.
- **Completed untimed entries**: Reordering operates over the untimed entries currently shown in the pane; if completed entries are hidden, `{`/`}` moves the highlighted entry relative to the next visible untimed entry so a keypress always produces a visible change unless the entry is at a visible boundary.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The TUI Planning view MUST allow the user to change the relative order of the untimed entries shown in the untimed pane for the displayed day.
- **FR-002**: Pressing `{` with an untimed entry highlighted MUST move that entry one position earlier (higher) within the untimed group; pressing `}` MUST move it one position later (lower).
- **FR-003**: After a reorder, the moved untimed entry MUST remain highlighted so the user can perform consecutive moves.
- **FR-004**: Reordering MUST be confined to the untimed group — `{`/`}` MUST NOT change the order or grid placement of timed entries, and an untimed entry MUST NOT be merged into the timed schedule by reordering.
- **FR-005**: Moving the first untimed entry higher or the last untimed entry lower MUST be a no-op that is accepted without error and preserves the current highlight.
- **FR-006**: The user-defined untimed order MUST persist so that re-rendering the view, leaving and returning to the day, and restarting the TUI all show the untimed entries in the order the user set.
- **FR-007**: When the highlight is not on an untimed entry, `{`/`}` MUST NOT reorder untimed entries.
- **FR-008**: The untimed pane MUST re-render to reflect the new order immediately after each `{`/`}` press.
- **FR-009**: The reordering capability MUST be added only to the TUI; the CLI and web app gain no new controls for reordering untimed entries as part of this feature.

### Key Entities *(include if feature involves data)*

- **Untimed plan entry**: A plan entry belonging to a specific day that has a duration but no start time. This feature gives such entries, within a single day, a user-defined relative position among the other untimed entries on that day. Entries that have a start time (timed entries / events) are not part of the untimed ordering.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can move an untimed entry to any position within the day's untimed list using only the `{` and `}` keys, with no other keystrokes required.
- **SC-002**: After a single `{` or `}` press on a non-boundary untimed entry, the entry's new position is visible in the untimed pane within one render cycle (no perceptible delay).
- **SC-003**: 100% of reorder operations preserve the full set of untimed entries — no untimed entry is lost, duplicated, or converted to a timed entry by reordering.
- **SC-004**: A user-defined untimed order is retained across a TUI restart in 100% of cases.
- **SC-005**: The `{`/`}` ranking gesture for untimed entries matches the gesture users already use for tasks and goals, so no new key needs to be learned.

## Assumptions

- **Persistence model**: The untimed order is persisted as a property of the user's data (consistent with the existing user-configurable task/goal ordering in feature 037), so it survives restarts. Because order is stored centrally, other clients that display untimed entries will naturally reflect the stored order even though only the TUI gains controls to change it; adding reorder controls to the CLI or web app is out of scope.
- **Default order before manual reorder**: Until a day's untimed entries are manually reordered, they appear in the current default order (creation order). Manual ordering overrides that default from then on.
- **New / re-entering untimed entries**: A newly created untimed entry, or a timed entry whose start time is cleared so it re-enters the untimed pane, is placed at the end of the day's untimed group by default; existing untimed entries keep their relative order.
- **Scope of ordering**: Untimed order is per-day; reordering affects only the untimed entries of the displayed day and never reorders or reschedules timed entries.
- **Unified highlight**: The existing single unified Up/Down highlight cycle through the untimed pane and the grid (established in feature 027) is unchanged; `{`/`}` act on the highlighted entry only when that entry is in the untimed pane.
- **Help/keymap surfacing**: The `{`/`}` "rank higher / rank lower" help already shown for tasks and goals is surfaced for untimed entries in the Planning view in the same style.

## Dependencies

- Builds on **027-untimed-plan-entries** (the untimed pane and untimed-entry concept) and **028-untimed-entries-polish**.
- Follows the ordering model and `{`/`}` interaction established by **037-task-ordering**.
