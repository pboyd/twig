# Feature Specification: Daily Planning

**Feature Branch**: `006-daily-planning`

**Created**: 2026-05-21

**Status**: Draft

**Input**: User description: "Users need to make plans and store them in the database through the API. A plan is always for a single day, and contains entries which cover part of the day. CLI command `todo plan` with subcommands task/event/rm/rename/mv/clear."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build a daily plan from tasks and events (Priority: P1)

A user wants to lay out their day by scheduling existing tasks and blocking off time for meetings, breaks, or other events. They run `todo plan task <task_id> <start>` to schedule work on a task, and `todo plan event <name> <start>` to block off time for a meeting. Each addition is persisted to the day's plan and assigned a sequential id within that day.

**Why this priority**: Without the ability to add entries, the planning feature has no value. This is the foundation that every other capability builds on.

**Independent Test**: Run `todo plan task 5 9:00am 1h` and `todo plan event "Lunch" 12:00pm`, then run `todo plan` to verify both entries appear in the day-planner grid for today.

**Acceptance Scenarios**:

1. **Given** no plan exists for today, **When** the user runs `todo plan task 3 09:00 1h`, **Then** the system creates an entry with id `1` linked to task 3, starting at 09:00 local time with a 60 minute duration, and prints the entry id `1`.
2. **Given** a plan with one entry exists, **When** the user runs `todo plan event "Standup" 10:00am 15m`, **Then** a second entry is added with id `2`.
3. **Given** a task with a pomodoro estimate of 4 and 1 completed pomodoro, **When** the user runs `todo plan task <task_id> 09:00` with no duration, **Then** the entry duration is set to 90 minutes (3 remaining pomodoros × 30 minutes).
4. **Given** a task without a pomodoro estimate, **When** the user runs `todo plan task <task_id> 09:00` with no duration, **Then** the entry duration defaults to 30 minutes.
5. **Given** an event command without a duration, **When** the user runs `todo plan event "Coffee" 10:00am`, **Then** the entry duration defaults to 30 minutes.
6. **Given** an existing entry from 09:00 to 10:00, **When** the user runs `todo plan event "Sync" 09:30 30m`, **Then** the system returns an error indicating an overlap and does not save the new entry.

---

### User Story 2 - View the daily plan (Priority: P1)

A user wants to see their plan for the day rendered as an ASCII day-planner grid so they can quickly understand how their day is structured. Running `todo plan` with no subcommand shows today's plan; adding `--date YYYY-MM-DD` shows another day.

**Why this priority**: Adding entries is only useful if the user can see the resulting schedule. This is required on day one alongside entry creation.

**Independent Test**: Add a few entries spanning 09:00–14:00 with a gap mid-day, then run `todo plan` and verify the grid starts at 09:00, ends at the last entry's end time, includes the gap, and labels each entry with its id and name.

**Acceptance Scenarios**:

1. **Given** today's plan has entries from 09:00–10:00 and 13:00–14:00, **When** the user runs `todo plan`, **Then** the grid shows hours from 09:00 through 14:00, omitting hours before 09:00 and after 14:00, with a visible gap between 10:00 and 13:00.
2. **Given** today's plan is empty, **When** the user runs `todo plan`, **Then** the system displays a message indicating the plan is empty rather than an empty grid.
3. **Given** a plan exists for 2026-05-22, **When** the user runs `todo plan --date 2026-05-22`, **Then** the grid shows that day's entries.
4. **Given** an entry has a linked task and no explicit name, **When** the plan is displayed, **Then** the entry shows the task's name.

---

### User Story 3 - Modify entries on the plan (Priority: P2)

After building a plan, the user often needs to reshape it: rename an entry, push it later, shorten it, or remove it. The `rm`, `rename`, and `mv` subcommands operate on entry ids and let the user iterate on the day.

**Why this priority**: Useful refinements but the plan is still functional without them — a user could delete and re-add. Ship after the core add/view flow works.

**Independent Test**: Create three entries, then `mv 2 14:00 45m`, `rename 1 "Deep work"`, and `rm 3`; verify each change is reflected on the next `todo plan` display.

**Acceptance Scenarios**:

1. **Given** an entry with id 2 exists, **When** the user runs `todo plan rm 2`, **Then** the entry is removed from the day and subsequent entries keep their existing ids (ids are not renumbered).
2. **Given** an entry with id 1 named "Email" exists, **When** the user runs `todo plan rename 1 "Inbox triage"`, **Then** the entry's name becomes "Inbox triage".
3. **Given** an entry with id 1 starting at 09:00 for 60m, **When** the user runs `todo plan mv 1 10:00 45m`, **Then** the entry moves to start at 10:00 with a 45-minute duration.
4. **Given** an entry with id 1 starting at 09:00 for 60m, **When** the user runs `todo plan mv 1 10:00` with no duration, **Then** only the start time changes; the duration remains 60 minutes.
5. **Given** a move would cause the entry to overlap another entry, **When** `mv` is executed, **Then** the system returns an error and leaves the entry unchanged.
6. **Given** the user references a non-existent entry id, **When** any modify subcommand runs, **Then** the system returns a clear "not found" error.

---

### User Story 4 - Clear the rest of the day (Priority: P3)

When a user's day goes off-script, they want to wipe the remaining schedule and re-plan. `todo plan clear` removes entries from a given point forward, trimming any entry that straddles the cutoff.

**Why this priority**: A convenience built on top of `rm`. Ship after the modify capabilities are in place.

**Independent Test**: Create entries at 09:00, 11:00, and 13:00 with the current time near 12:30; run `todo plan clear` and verify entries starting at or after 12:30 are removed, and any entry overlapping 12:30 is shortened to end at 12:30.

**Acceptance Scenarios**:

1. **Given** entries at 09:00–10:00, 11:00–12:00, 13:00–14:00 and the current time is 12:30, **When** the user runs `todo plan clear`, **Then** the 13:00 entry is removed, the 11:00 entry is unaffected (already ended), and the 09:00 entry is unaffected.
2. **Given** an entry from 11:00–12:00 and `clear 11:30` is run, **When** the command executes, **Then** the entry's duration is shortened to end at 11:30 and any later entries are removed.
3. **Given** `clear` is run with a start time before all entries on the day, **When** the command executes, **Then** all entries for that day are removed.

---

### Edge Cases

- The user provides a start time in mixed formats (e.g., `13:15`, `1:15pm`, `01:15 PM`) — all must be accepted and parsed to the same moment in local time.
- The user provides a duration as `90m`, `1h30m`, or `2h` — all must be accepted.
- An entry's start + duration crosses midnight — out of scope; the system rejects such entries with a clear error since a plan is for a single day.
- The user runs `todo plan` for a date with no plan — show an empty-plan message rather than failing.
- The user adds an entry to a date in the past or far future — allowed; no restriction on which day a plan covers.
- The user runs `task <task_id>` for a task that does not exist — return a not-found error.
- The user runs `clear` on a day with no entries — succeed silently (no-op).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST persist plan entries through the API so they survive across CLI invocations.
- **FR-002**: Each entry MUST be uniquely identified by the combination of `day` and `id`, where `id` is a sequential integer starting at 1 for the first entry added to that day.
- **FR-003**: Entry ids MUST NOT be renumbered when an entry is removed; subsequently added entries continue from the highest used id + 1 on that day.
- **FR-004**: An entry MAY link to a task via `task_id`; the link is optional so entries can represent meetings, breaks, or other blocked time.
- **FR-005**: An entry's `name` MAY be omitted when a `task_id` is provided; in that case the displayed name falls back to the linked task's name.
- **FR-006**: An entry's `name` MUST be provided for entries without a linked task.
- **FR-007**: Entry start times MUST be stored and interpreted in the user's local timezone.
- **FR-008**: System MUST reject any add or move operation that would cause two entries on the same day to overlap, and MUST return a clear error identifying the conflict.
- **FR-009**: CLI command `todo plan` MUST accept an optional `--date YYYY-MM-DD` flag; when omitted, operations target the current local day.
- **FR-010**: With no subcommand, `todo plan` MUST render the day's entries as an ASCII day-planner grid that begins at the hour of the first entry, ends at the end of the last entry, includes intermediate hours (showing gaps as empty), and labels each entry with its id and display name.
- **FR-011**: `todo plan task <task_id> <start> [duration]` MUST add an entry linked to the given task and print the new entry id. If `duration` is omitted: when the task has a pomodoro estimate, duration is `(estimate - completed_pomodoros) * 30 minutes`; otherwise duration defaults to 30 minutes.
- **FR-012**: `todo plan event <name> <start> [duration]` MUST add an entry with no linked task; if `duration` is omitted it defaults to 30 minutes.
- **FR-013**: `todo plan rm <n>` MUST remove the entry with id `n` from the target day.
- **FR-014**: `todo plan rename <n> <name>` MUST change the `name` field of entry `n` on the target day.
- **FR-015**: `todo plan mv <n> <start> [duration]` MUST update the start time of entry `n`, and MUST update its duration if provided; when duration is omitted the previous duration is kept.
- **FR-016**: `todo plan clear [start]` MUST remove every entry starting at or after `start` on the target day; if an entry straddles `start`, its duration MUST be shortened so it ends exactly at `start`. When `start` is omitted, the current local time is used.
- **FR-017**: Start time parsing MUST accept 24-hour (`13:15`), 12-hour without space (`1:15pm`), and 12-hour with space and uppercase meridiem (`01:15 PM`), all producing the same instant on the target day.
- **FR-018**: Duration parsing MUST accept minute-only (`90m`), hour+minute (`1h30m`), and hour-only (`2h`) forms.
- **FR-019**: All subcommands that target an entry by id MUST return a clear "not found" error when the id does not exist on the target day.
- **FR-020**: Deleting a task referenced by a plan entry is out of scope for this feature; entries reference tasks as a foreign key but task-deletion semantics are not changed here.

### Key Entities

- **Plan Entry**: A scheduled segment of a single day. Attributes: `day` (the date), `id` (sequential per-day integer), optional `task_id` (link to a Task), optional `name` (overrides the task's name for display), `start` (local time-of-day on `day`), `duration` (length of segment). Identity is `(day, id)`. Two entries on the same `day` MUST NOT overlap on the time axis.
- **Plan** (implicit): The collection of all Plan Entries for a single `day` for a given user. Not a separately persisted entity — derived by querying entries for a date.
- **Task** (existing): Referenced by `task_id`. Provides the default display name and, when present, a pomodoro estimate used to compute a default duration.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can build a full day's plan (5+ entries mixing tasks and events) using the CLI in under 2 minutes.
- **SC-002**: 100% of overlapping-entry attempts are rejected with an error that names the conflicting entry id.
- **SC-003**: All documented start-time formats (`13:15`, `1:15pm`, `01:15 PM`) and duration formats (`90m`, `1h30m`, `2h`) parse to the expected value in automated tests, with zero parsing failures across the documented variants.
- **SC-004**: After running any modify command (`rm`, `rename`, `mv`, `clear`), running `todo plan` reflects the change with no additional refresh step.
- **SC-005**: When a task with a pomodoro estimate is scheduled with no explicit duration, the computed duration matches `(estimate - completed) * 30 minutes` in 100% of cases.

## Assumptions

- The CLI already has a way to identify the active user (or operates as a single-user local tool), so plans are scoped implicitly; multi-user scoping at the API level follows whatever pattern the existing task API uses.
- The existing Task entity exposes `name`, a pomodoro estimate, and a count of completed pomodoros that the planner can read (introduced by the 005-pomodoro-tracking feature).
- "Current day" and "local timezone" are determined from the machine running the CLI; the API stores times in a way that round-trips this local interpretation.
- A plan is strictly bounded to a single calendar day; entries that would cross midnight are rejected rather than split.
- The ASCII grid renders at a fixed granularity (e.g., one row per 15 or 30 minutes) chosen by the implementation; the spec only requires that the grid spans first-entry to last-entry and shows gaps.
- Display formatting (column widths, exact characters) is at implementation discretion as long as the entry id and name are legible and ordering follows clock time.
