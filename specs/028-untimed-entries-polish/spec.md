# Feature Specification: Untimed Plan Entries — Polish & Bugfixes

**Feature Branch**: `028-untimed-entries-polish`

**Created**: 2026-06-01

**Status**: Draft

**Input**: User description: "After testing, there are a few refinements and bugfixes to make to the untimed plan entries: (1) When a task is added to a plan from the tasks tab (with `p` or `ctrl+p`), there's no indication that anything happened. There should be a message added to the status bar. (Auto-dismiss after a few seconds would be nice, but status-bar messages don't currently dismiss, so that is out of scope.) (2) Currently, I can add multiple copies of a task without a time. It is allowed that a task is added to a plan multiple times, but before now they've had different times. It should not be allowed to add multiple copies of a task without a time. (3) Visually, the entries without a time look different than normal entries — the title overlaps the top border, the boxes don't join together, and the text color / background color is wrong on the last line. (4) It would also be nice if there were some visual separation between the entries without a time and the day planner."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Confirmation when sending a task to a plan (Priority: P2)

A user on the Tasks tab presses `p` (or `ctrl+p` and picks a day) to send a highlighted task to a plan. Today, nothing visibly changes on the Tasks tab, so the user cannot tell whether the action succeeded. After this change, a confirmation message appears in the status bar telling the user the task was added (and, for `ctrl+p`, to which day).

**Why this priority**: Without feedback, users repeat the action or assume it failed — which both confuses them and contributes directly to the duplicate-entry problem in User Story 2. It is a small, high-value UX fix, but the feature still functions without it, so it ranks below the correctness and rendering fixes.

**Independent Test**: From the Tasks tab, highlight a task and press `p`; confirm a success message naming the task appears in the status bar. Repeat with `ctrl+p`, choose a day, and confirm the message reflects that the task was added to that day.

**Acceptance Scenarios**:

1. **Given** a highlighted task on the Tasks tab, **When** the user presses `p` and the task is added to today's plan, **Then** a confirmation message appears in the status bar indicating the task was added to today's plan.
2. **Given** a highlighted task on the Tasks tab, **When** the user completes the `ctrl+p` flow for a chosen day and the task is added, **Then** a confirmation message appears in the status bar indicating the task was added to that day's plan.
3. **Given** the send action does not succeed (e.g. it is rejected as a duplicate untimed entry per User Story 2), **When** the user presses `p`/`ctrl+p`, **Then** the status bar shows a message explaining why no entry was added, rather than a success message.

---

### User Story 2 - Prevent duplicate untimed entries for the same task (Priority: P1)

A user adds a task to a day's plan without a time. Later they try to add the same task to the same day without a time again. The system prevents the second untimed copy: a task may appear on a day's plan multiple times, but at most once *without* a time. The user is told the task is already on that day untimed, so no duplicate is created.

**Why this priority**: This is a correctness rule for the data. Untimed entries have no start time to distinguish them, so multiple untimed copies of one task on one day are indistinguishable duplicates with no way to tell apart or manage meaningfully. Preventing them keeps the plan coherent.

**Independent Test**: Add a task to a day's plan with no time; confirm one untimed entry exists. Attempt to add the same task to the same day with no time again; confirm no second untimed entry is created and the user is informed.

**Acceptance Scenarios**:

1. **Given** a day with one untimed entry for task X, **When** the user adds task X to that day again without a time, **Then** no second untimed entry is created and the user is informed the task is already on that day's plan untimed.
2. **Given** a day with one untimed entry for task X, **When** the user adds task X to that day *with* a time, **Then** a timed entry is created (timed copies remain allowed alongside the single untimed entry).
3. **Given** a day with one *timed* entry for task X, **When** the user adds task X to that day without a time, **Then** an untimed entry is created (the existing timed entry does not block the first untimed entry).
4. **Given** a day with one untimed entry for task X, **When** the user converts a *different* timed entry for task X to untimed (clearing its start time), **Then** the conversion is rejected because it would create a second untimed entry for task X on that day, and the entry keeps its time.

---

### User Story 3 - Untimed entries render correctly (Priority: P1)

A user viewing the Planning view sees untimed entries in the top-left pane rendered exactly like gridded entries: the title sits inside the box (not overlapping the top border), adjacent entry boxes join together the way gridded entries do, and the colors (text and background) on every line — including the last line — match the rest of the entry.

**Why this priority**: The current rendering looks broken and inconsistent with the rest of the plan, undermining confidence in the feature. It is the most visible defect and affects every user who has an untimed entry.

**Independent Test**: Create two adjacent untimed entries with multi-line durations. Open the Planning view and confirm: each title is fully inside its box, the two boxes join cleanly, and the colors on the last line of each entry match the other lines.

**Acceptance Scenarios**:

1. **Given** an untimed entry, **When** it is rendered in the untimed pane, **Then** its title is contained within the box and does not overlap or break the top border.
2. **Given** two or more untimed entries listed adjacently, **When** they are rendered, **Then** their boxes join together with the same border treatment used for adjacent gridded entries.
3. **Given** an untimed entry spanning multiple lines, **When** it is rendered, **Then** the text and background colors are consistent across all lines, including the last line.
4. **Given** an untimed entry and a gridded entry of the same duration and state (highlighted/normal, complete/incomplete), **When** both are rendered, **Then** they are visually indistinguishable in style.

---

### User Story 4 - Visual separation between untimed entries and the day planner (Priority: P3)

A user looking at the Planning view can clearly tell where the untimed pane ends and the day planner grid begins. A visual separation distinguishes the two areas so untimed entries do not blur into the timed schedule.

**Why this priority**: A clarity improvement explicitly described as "nice to have." The view is usable without it, so it is the lowest-priority slice.

**Independent Test**: Open the Planning view on a day that has at least one untimed entry and a populated grid; confirm there is a clear visual boundary between the untimed pane and the day planner.

**Acceptance Scenarios**:

1. **Given** a day with at least one untimed entry, **When** the Planning view renders, **Then** there is a visible separation between the untimed pane and the day planner grid below it.
2. **Given** a day with no untimed entries, **When** the Planning view renders, **Then** the untimed pane (and its separation) is absent and the day planner uses the full area, unchanged from current behavior.

---

### Edge Cases

- **Duplicate attempt via the CLI**: Adding a task without a time via `twig plan task` when that task already has an untimed entry on the target day is rejected the same way as the TUI path, with an informative message; no entry is created.
- **Clearing a time onto an existing untimed entry**: Using `twig plan mv` (or the TUI edit flow) to clear an entry's start time is rejected if the same task already has an untimed entry on that day; the entry keeps its time.
- **Same task, multiple timed copies**: Multiple *timed* copies of the same task on one day remain allowed; only the untimed copy is limited to one per task per day.
- **Status message persistence**: The confirmation message remains in the status bar until replaced (it does not auto-dismiss); auto-dismissal of status-bar messages is out of scope for this feature.
- **Send failure feedback**: When a send is rejected (duplicate untimed entry), the status bar reflects the failure reason rather than a success message.
- **Highlight after rendering fix**: A highlighted untimed entry continues to render its highlight consistently across all of its lines after the color fix.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: When a task is successfully added to a plan from the Tasks tab via `p`, the TUI MUST display a confirmation message in the status bar indicating the task was added to today's plan.
- **FR-002**: When a task is successfully added to a plan from the Tasks tab via the `ctrl+p` flow, the TUI MUST display a confirmation message in the status bar indicating the task was added to the chosen day's plan.
- **FR-003**: A task MUST NOT have more than one untimed entry on the same day's plan; the system MUST reject any attempt to create a second untimed entry for a task that already has one on that day.
- **FR-004**: Timed entries MUST remain unconstrained by FR-003 — a task may have multiple timed entries on a day, and may have timed entries alongside its single untimed entry.
- **FR-005**: When an attempt to add an untimed entry is rejected as a duplicate (FR-003), the user MUST be informed (via the status bar in the TUI, and via an error message in the CLI) and no entry MUST be created.
- **FR-006**: The duplicate-untimed prevention (FR-003) MUST apply across all paths that create or convert to an untimed entry: Tasks-tab `p`/`ctrl+p`, `twig plan task` without a time, and clearing a start time via `twig plan mv` or the TUI edit flow.
- **FR-007**: Untimed entries MUST render their title fully within the entry box, without overlapping or breaking the top border.
- **FR-008**: Adjacent untimed entries MUST join their boxes together using the same border treatment applied to adjacent gridded entries.
- **FR-009**: An untimed entry's text and background colors MUST be consistent across all of its display lines, including the last line, matching the colors used for gridded entries in the same state.
- **FR-010**: Untimed entries MUST be visually indistinguishable in style from gridded entries of the same duration and state (highlighted/normal, complete/incomplete).
- **FR-011**: The Planning view MUST provide a visible visual separation between the untimed pane and the day planner grid when at least one untimed entry exists.
- **FR-012**: When a day has no untimed entries, the untimed pane and its separation MUST be absent and the day planner MUST use the full area, unchanged from prior behavior.
- **FR-013**: The confirmation and rejection messages introduced here MAY persist in the status bar until replaced; automatic dismissal of status-bar messages is explicitly out of scope.

### Key Entities *(include if feature involves data)*

- **Plan Entry**: A segment of one day's plan, linked to a task (for untimed entries). Newly constrained so that for a given (day, task) pair at most one entry may be untimed; timed entries remain unconstrained in count.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After pressing `p` or completing `ctrl+p`, 100% of successful sends produce a visible status-bar confirmation that names the action's outcome (task added, and the day for `ctrl+p`).
- **SC-002**: It is impossible to create a second untimed entry for the same task on the same day through any supported path (Tasks tab, CLI add, or clearing a time); every such attempt is rejected with an informative message.
- **SC-003**: A task can still have multiple timed entries on one day, and a timed entry plus one untimed entry on one day, with no false rejections.
- **SC-004**: An untimed entry and a gridded entry of the same duration and state are visually indistinguishable in style — title placement, joined borders, and per-line colors all match.
- **SC-005**: On a day with untimed entries, users can identify the boundary between the untimed pane and the day planner without prior instruction.
- **SC-006**: On a day with no untimed entries, the Planning view layout is unchanged from current behavior.

## Assumptions

- **Scope is the 027 feature**: This work refines the untimed plan entries delivered in feature 027; existing timed-entry behavior and the overall Planning view layout are otherwise unchanged.
- **Duplicate rule is per (day, task), untimed only**: "At most one untimed entry" is scoped to a single day and a single task; it does not limit timed copies and does not de-duplicate across days.
- **Rejection over silent no-op**: A blocked duplicate-untimed attempt informs the user (status bar / CLI error) rather than failing silently, reinforcing the User Story 1 feedback theme.
- **Message wording**: Exact confirmation and rejection wording follows the app's existing warm/playful copy conventions; the precise strings are an implementation detail.
- **No auto-dismiss**: Per the user, auto-dismissing status-bar messages after a few seconds is out of scope and deferred; messages persist until replaced, matching how existing error messages behave.
- **Rendering parity is the target**: The fix makes untimed entries render via the same visual treatment as gridded entries rather than introducing a new style.
- **Separation style is an implementation detail**: The exact form of the visual separation (e.g. a divider line, spacing, or label) is left to design, provided it clearly delimits the two areas.
