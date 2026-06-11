# Feature Specification: Activity Report

**Feature Branch**: `048-activity-report`

**Created**: 2026-06-11

**Status**: Draft

**Input**: User description: "I need a reporting feature in this app so that I can see what I've done. Typical use-cases: preparing to give a status update in a meeting; looking at significant accomplishments to use for quarterly/annual reviews; quick reminder of what I did yesterday. This is really important to have in the TUI, and the CLI would be nice too. I don't see much value right now including it in the web app."

## Clarifications

### Session 2026-06-11

- Q: How should the report choose between the day-grouped layout and the accomplishment-grouped layout? → A: Automatically by period length — 14 days or fewer uses day-grouped; longer uses accomplishment-grouped. No user control.
- Q: Which completed pomodoros should the period's summary total count? → A: Every pomodoro completed during the period, regardless of whether its task is complete (measures effort spent).
- Q: When do the "this week" / "last week" presets start the week? → A: Monday (ISO 8601 work week, Monday through Sunday).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review recent activity in the TUI (Priority: P1)

A user opens the interactive TUI and switches to a report view to see what they completed recently. The view defaults to a short, recent period (today and yesterday) so the most common question — "what did I just do?" — is answered immediately, with no typing. The user can step to other common periods (yesterday, this week, last week) with single keystrokes.

**Why this priority**: This is the user's stated primary surface ("really important to have in the TUI") and covers the two most frequent use-cases: the quick "what did I do yesterday?" reminder and pre-meeting status-update prep. Without it the feature delivers no value.

**Independent Test**: Complete several tasks on known dates, open the TUI report view, and confirm the completed items appear under the correct period with their completion dates. Switching periods updates the list. This alone is a usable MVP.

**Acceptance Scenarios**:

1. **Given** a user with tasks completed today and yesterday, **When** they open the report view in the TUI, **Then** they see those tasks grouped by day with the most recent day first.
2. **Given** the report view is open, **When** the user selects the "yesterday" period, **Then** only tasks completed yesterday (in the user's local timezone) are shown.
3. **Given** a completed subtask whose parent task is still open, **When** the report covers the subtask's completion day, **Then** the subtask appears in the report with enough context (e.g., its parent's name) to identify it.
4. **Given** a period in which the user completed nothing, **When** they view the report for that period, **Then** they see a friendly empty state rather than a blank screen or error.

---

### User Story 2 - Generate a report from the CLI (Priority: P2)

A user runs a report command from the shell, optionally specifying a period (e.g., yesterday, last week, or an explicit start/end date range), and receives a readable text summary of completed work. The plain-text output can be copied into a status-update message or piped to other tools.

**Why this priority**: The user explicitly wants the CLI as a secondary surface ("the CLI would be nice too"). It also enables copy/paste into meeting notes and scripting, which the TUI cannot do as easily.

**Independent Test**: Run the report command with various period arguments and verify the output lists exactly the tasks completed within each period, formatted readably both in a terminal and when redirected to a file.

**Acceptance Scenarios**:

1. **Given** completed tasks across several days, **When** the user runs the report command with no arguments, **Then** they get a report for a sensible default period (today and yesterday).
2. **Given** the same data, **When** the user runs the report command with an explicit date range, **Then** only tasks completed within that range (inclusive, local-timezone day boundaries) are listed.
3. **Given** output is redirected to a file or pipe, **When** the report runs, **Then** the output contains no terminal styling artifacts and remains readable.
4. **Given** an invalid date range (e.g., end before start, unparseable date), **When** the user runs the command, **Then** they receive a clear error message explaining the accepted formats.

---

### User Story 3 - Long-range accomplishment review (Priority: P3)

A user preparing for a quarterly or annual review requests a report over a long period (e.g., three months or a year). Because such a period can contain hundreds of small items, the report helps surface the significant ones: completed top-level tasks are presented as the headline accomplishments, with their completed subtasks summarized beneath them rather than listed as a flat undifferentiated stream.

**Why this priority**: Quarterly/annual reviews happen a few times a year, so this is less frequent than the daily/weekly use-cases, but it is the use-case where memory fails most and the tool adds the most unique value.

**Independent Test**: Seed a multi-month history of completed parent and child tasks, request a quarter-long report, and verify top-level completions are shown prominently with subtask counts/summaries, and that the report remains readable (not an unstructured wall of items).

**Acceptance Scenarios**:

1. **Given** a quarter's worth of completed work, **When** the user requests a report for that quarter, **Then** completed top-level tasks appear as headline items with their completed subtasks grouped under them.
2. **Given** a long-period report, **When** it is displayed, **Then** the report includes summary totals (e.g., number of tasks completed, pomodoros completed) for the period.
3. **Given** a period of a year, **When** the report is requested, **Then** it is produced without error and within a few seconds.

---

### Edge Cases

- A task completed within the period and later marked incomplete again no longer appears in the report (only currently-complete tasks count).
- A subtask is complete but its parent is not: the subtask appears, attributed to its (incomplete) parent for context.
- Day-boundary handling: a task completed at 23:55 local time belongs to that local calendar day, even if it is the next day in UTC.
- Period spans a daylight-saving transition: days still group correctly by local calendar date.
- The user has never completed anything: every surface shows a friendly empty state.
- Very long task names or descriptions: report layout truncates or wraps without breaking alignment.
- The server is unreachable: TUI and CLI show the same clear connection error used elsewhere in the app.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST let a user view a report of their completed tasks for a chosen time period, in both the TUI and the CLI.
- **FR-002**: The report MUST include, for each completed task: its name, its completion date, and its parent-task context when it is a subtask.
- **FR-003**: For periods of 14 days or fewer, the report MUST group completed tasks by the local calendar day of completion, most recent day first.
- **FR-004**: For periods longer than 14 days, the report MUST organize work around completed top-level tasks, grouping completed subtasks beneath their parents, and MUST include period summary totals (tasks completed; pomodoros completed). The layout is selected automatically from the period length; there is no manual layout override.
- **FR-005**: The system MUST provide named period presets — at minimum: today, yesterday, this week, last week, this month, this quarter, this year — and accept an explicit start/end date range. Weeks run Monday through Sunday (ISO 8601).
- **FR-006**: Period boundaries MUST be interpreted in the user's local timezone, with both endpoints inclusive of the full calendar day.
- **FR-007**: The TUI report view MUST be reachable from the existing TUI navigation and allow switching periods without leaving the view.
- **FR-008**: The CLI report command MUST default to a recent period (today and yesterday) when invoked with no arguments.
- **FR-009**: CLI report output MUST be plain, unstyled text when not attached to a terminal, consistent with the rest of the CLI's TTY-detection behavior.
- **FR-010**: The report MUST show only the calling user's own activity.
- **FR-011**: The report MUST reflect current task state: tasks un-completed after being completed do not appear.
- **FR-012**: When a period contains no completed work, the system MUST show an explicit, friendly empty-state message.
- **FR-013**: Invalid period input (unparseable dates, end before start) MUST produce a clear error naming the accepted formats, without producing a partial report.
- **FR-014**: The report MUST include, in the summary totals, the count of all pomodoros completed during the period — including pomodoros on tasks that are still incomplete. The pomodoro total measures effort spent, independent of task completion.

### Key Entities

- **Activity Report**: A read-only view of the user's completed work over a period; composed of report entries plus summary totals. Not stored — generated on demand from existing data.
- **Report Period**: A start and end calendar day (inclusive) in the user's local timezone; either chosen from named presets or given explicitly.
- **Report Entry**: One completed task occurrence: task name, completion date, and parent context if it is a subtask.
- **Summary Totals**: Aggregate counts for the period: tasks completed and pomodoros completed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can answer "what did I do yesterday?" from a cold start of the TUI in under 10 seconds, using only keyboard navigation.
- **SC-002**: A user preparing a weekly status update can produce a copy-pasteable list of the week's completed work with a single CLI command.
- **SC-003**: A report covering a full year of activity is produced and displayed in under 3 seconds.
- **SC-004**: 100% of tasks completed within a requested period (and still complete) appear in that period's report; no tasks completed outside it appear.
- **SC-005**: Completion dates shown in reports match the user's local calendar day of completion in 100% of cases, including near-midnight completions.

## Assumptions

- "What I've done" means completed tasks (the app records a permanent completion moment for each task) supplemented by completed-pomodoro counts; the report does not attempt to track other activity (task edits, plans made, time-in-app).
- Pomodoro counts appear as summary totals; the report does not list individual pomodoro sessions.
- Reports are generated on demand from existing data; no new history or audit log is recorded, so activity that predates this feature is already reportable.
- "Significant accomplishments" for long-range reviews is approximated structurally (completed top-level tasks are the headlines); there is no manual flagging/starring of accomplishments in this version.
- The web app is explicitly out of scope for this feature, per the user's request.
- The existing authentication and single-user-scoped data access are reused; reports never aggregate across users.
- Reports are display-only; no export formats (CSV/PDF/markdown files) beyond plain CLI text output are in scope for this version.
