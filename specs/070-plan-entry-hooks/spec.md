# Feature Specification: External Commands for Plan Entry Boundaries

**Feature Branch**: `070-plan-entry-hooks`

**Created**: 2026-07-28

**Status**: Draft

**Input**: User description: "external commands for plan events/tasks — It's easy to lose track of time when working on a task and miss the fact that the plan says it's time to switch tasks. So, similar to the external commands for pomodoros, we need configurable commands that run when plan entries begin and end. In the config file, these should be `[plan] on_event_start / on_event_end / on_task_start / on_task_end`. We need to support placeholders in the command for the entry name and the time: `%s` (name), `%q` (name in double-quotes), `%t` (formatted time, hours and minutes). The task name may need to have characters escaped in order to make a valid command. The commands should run at the start/end time (not before)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Be told when it is time to start and stop a planned task (Priority: P1)

A user plans their day with timed task entries, then works heads-down. When the clock reaches the start of a planned task entry, their configured command runs and surfaces a notification. When the clock reaches the end of that entry, a second configured command runs. The user no longer has to watch the plan to know it is time to switch.

**Why this priority**: This is the core problem stated — losing track of time and missing the moment to switch tasks. Task entries are the dominant kind of plan entry, so this alone delivers the feature's value.

**Independent Test**: Configure `[plan] on_task_start` and `on_task_end`, put a timed task entry on today's plan a minute or two out, leave the interactive app running, and confirm each command fires once, at (not before) the entry's start minute and end minute respectively.

**Acceptance Scenarios**:

1. **Given** `on_task_start` is configured and today's plan has a task entry starting at 14:00, **When** the clock reaches 14:00 while the interactive app is running, **Then** the configured command runs exactly once and not before 14:00.
2. **Given** `on_task_end` is configured and a task entry runs 14:00–14:30, **When** the clock reaches 14:30, **Then** the end command runs exactly once.
3. **Given** both hooks are configured and two task entries are back-to-back (one ends at 14:30, the next starts at 14:30), **When** the clock reaches 14:30, **Then** the end command for the first entry runs and the start command for the second entry runs, with the end command running first.
4. **Given** no `[plan]` hooks are configured, **When** plan entry boundaries pass, **Then** nothing is executed and behaviour is unchanged from today.
5. **Given** the app is started at 14:20 and today's plan has a task entry that started at 14:00, **When** the app loads the plan, **Then** the already-passed start boundary does **not** fire.

---

### User Story 2 - Be told when a planned event begins and ends (Priority: P2)

A user's plan also contains events — entries not linked to a task, such as a meeting or a break. These need their own commands, separately configurable from task commands, so the user can distinguish "your meeting starts now" from "switch tasks now".

**Why this priority**: Same mechanism as US1 and clearly requested, but events are a smaller share of most plans, so US1 alone is a viable slice.

**Independent Test**: Configure only `on_event_start` / `on_event_end`, add a timed event to today's plan, and confirm those commands fire at its boundaries while the task commands do not.

**Acceptance Scenarios**:

1. **Given** `on_event_start` is configured and today's plan has an event starting at 09:00, **When** the clock reaches 09:00, **Then** the event start command runs.
2. **Given** both `on_task_start` and `on_event_start` are configured, **When** an event boundary is reached, **Then** only the event command runs.
3. **Given** only `on_event_end` is configured, **When** a task entry's end boundary is reached, **Then** nothing runs.

---

### User Story 3 - Put the entry name and time into the command (Priority: P3)

A user writes `on_task_start = 'notify-send "%t: start %q"'` and gets a notification reading `14:00: start "Write the quarterly report"`. Entry names containing quotes, dollar signs, backslashes, or backticks still produce a correct notification rather than a broken or mangled command.

**Why this priority**: Without substitution the hooks still fire and can still alert (e.g. a fixed sound or static notification), so this layers value on top of US1/US2 rather than gating it. It is, however, what makes the alert actually useful.

**Independent Test**: Configure a command containing `%s`, `%q`, and `%t`, use an entry whose name contains a double quote and a dollar sign, and confirm the resulting output contains the entry's literal name and its boundary time in `HH:MM` form.

**Acceptance Scenarios**:

1. **Given** `on_task_start = 'notify-send "%t: start %q"'` and a task entry named `Write report` starting at 14:00, **When** the start boundary fires, **Then** the notification text is `14:00: start "Write report"`.
2. **Given** an entry named `Fix "auth" $bug`, **When** a hook using `%q` fires, **Then** the command receives that name verbatim, with the embedded quote and dollar sign neutralised so the shell does not reinterpret them.
3. **Given** a task entry with no name override, **When** a hook using `%s` or `%q` fires, **Then** the linked task's name is substituted.
4. **Given** a command containing `%%`, **When** the hook fires, **Then** a single literal `%` appears in its place.
5. **Given** a command containing an unrecognised sequence such as `%z`, **When** the hook fires, **Then** the sequence is passed through unchanged.

---

### Edge Cases

- **Untimed entries**: task entries may have no start time. These have no boundaries and MUST never fire a hook.
- **App not running**: boundaries that pass while no interactive session is open do not fire, and are not replayed when a session later opens.
- **Boundary already past at load**: covered in US1 scenario 5 — no retroactive firing, including for entries added to the plan with a start time already in the past.
- **Plan edited mid-session**: moving, renaming, adding, or deleting an entry re-derives the pending boundaries. An entry moved from a past time to a future time fires at its new time; an entry deleted before its boundary never fires.
- **An entry moved after it already fired**: the boundary is identified by entry and boundary time, so an entry rescheduled to a new future time fires again at that new time. This is intended.
- **Hook command fails**: a non-zero exit, a missing binary, or a hook that hangs must not disturb the plan, the display, or other hooks.
- **Slow/long-running hooks**: a hook that takes a long time must not delay a later boundary or block the interface.
- **Midnight rollover / day change**: only entries on the current local day are watched; when the local date changes, the watched set switches to the new day's plan.
- **Machine sleep or clock jump**: if the clock jumps forward past one or more boundaries, those boundaries are skipped rather than fired late in a burst.
- **Same command in multiple slots**: configuring the same string for several hook keys is allowed; each fires independently.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The config file MUST support a `[plan]` section with four optional string keys: `on_event_start`, `on_event_end`, `on_task_start`, `on_task_end`. An absent or empty value means "do nothing".
- **FR-002**: The system MUST run `on_task_start` at the start time of each timed plan entry that is linked to a task, and `on_task_end` at that entry's end time (start plus duration).
- **FR-003**: The system MUST run `on_event_start` and `on_event_end` at the corresponding boundaries of each timed plan entry that is **not** linked to a task.
- **FR-004**: A hook MUST NOT run before its boundary time, and MUST run within 60 seconds after it.
- **FR-005**: Each boundary MUST fire at most once per session, identified by the entry and its boundary time.
- **FR-006**: The system MUST NOT fire boundaries that were already in the past when the plan was loaded or when the entry was created/rescheduled.
- **FR-007**: Untimed plan entries MUST NOT fire any hook.
- **FR-008**: Only entries on the current local day MUST be watched; the watched set MUST follow the local date across midnight.
- **FR-009**: The system MUST substitute `%s` in the command with the entry's display name — the entry's own name when set, otherwise the linked task's name.
- **FR-010**: The system MUST substitute `%q` with the entry's display name wrapped in double quotes, with any characters that would otherwise be reinterpreted by the shell (double quote, backslash, dollar sign, backtick) neutralised so the name reaches the command verbatim.
- **FR-011**: The system MUST substitute `%t` with the boundary's time formatted as zero-padded 24-hour `HH:MM` (e.g. `09:05`, `14:00`) — the entry's scheduled start time for start hooks and its scheduled end time for end hooks.
- **FR-012**: The system MUST substitute `%%` with a single literal `%`, and MUST pass through any other `%`-prefixed sequence unchanged.
- **FR-013**: Substitution MUST NOT be recursive: text introduced by one substitution MUST NOT itself be scanned for placeholders.
- **FR-014**: A hook that exits non-zero, cannot be launched, or writes output MUST NOT abort the session, corrupt the display, or prevent other hooks from running. The failure MUST be surfaced to the user as a non-fatal warning identifying which hook key failed.
- **FR-015**: Hook execution MUST NOT block the interface or delay evaluation of subsequent boundaries.
- **FR-016**: When multiple boundaries fall on the same minute, end hooks MUST run before start hooks, so an "end X / start Y" pair reads in the right order.
- **FR-017**: Editing the plan during a session (add, remove, rename, move, or change duration) MUST re-derive pending boundaries from the updated plan.
- **FR-018**: Unknown keys inside `[plan]` MUST be ignored, consistent with the existing config-file forward-compatibility rule.
- **FR-019**: The documented config schema MUST be updated to describe the `[plan]` section, its four keys, the placeholder set, and the execution semantics.

### Key Entities

- **Plan hook configuration**: the four optional command strings under `[plan]`, read from the same config file as the existing pomodoro hooks; no environment-variable override.
- **Boundary**: a scheduled moment derived from a timed plan entry — its start or its end — carrying the entry's kind (task or event), display name, and boundary time. This is the unit that is watched, fired, and deduplicated.
- **Placeholder set**: `%s`, `%q`, `%t`, `%%` — the substitutions applied to a hook command string just before execution.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With hooks configured and a session open, 100% of a day's timed plan-entry boundaries produce exactly one command run each, no earlier than the scheduled minute and no more than 60 seconds after it.
- **SC-002**: A user who configures a notification command is alerted at every planned switch point without looking at the plan, eliminating missed switches caused by not watching the clock.
- **SC-003**: Entry names containing quotes, backslashes, dollar signs, and backticks are delivered to the command verbatim in 100% of cases, with no command breakage or altered text.
- **SC-004**: A failing or missing hook command produces a visible non-fatal warning and zero disruption — the session continues and every later boundary still fires.
- **SC-005**: Configuring the feature takes one edit to the existing config file and no restart of anything other than the app itself.
- **SC-006**: With no `[plan]` section configured, behaviour is byte-for-byte identical to today (no processes launched, no warnings).

## Assumptions

- **Hooks fire only while the interactive session is running.** The product has no background daemon, and adding one is well beyond this request. Boundaries that pass with no session open are silently missed. This mirrors how pomodoro hooks already work (they fire from the running CLI/TUI process).
- **Execution semantics mirror the existing pomodoro hooks**: the string is run through the system shell so pipes, `&&`, redirects, and environment expansion all work, and a failure is a non-fatal warning. This reuses a mechanism users already understand.
- **`%s` substitutes the name raw and unescaped**; making it shell-safe is the user's responsibility. `%q` is the safe form. Both are offered because `%s` is the useful one inside a command the user has already quoted their own way, and `%q` is the one that is correct by default.
- **`%t` is the scheduled boundary time, not the wall-clock time at which the hook happened to run.** A hook that fires a few seconds late still reports the planned minute, which is what the user's plan says.
- **The end of an entry is its start time plus its duration**, matching how the plan already renders entries; there is no separate stored end time.
- **The entry's name override wins over the linked task's name** when both are present, matching existing plan display behaviour.
- **No new server-side data or API is required.** Boundaries are derived entirely from the plan entries the client already fetches.
- **No environment-variable override** is provided for these keys, matching the existing pomodoro hooks.
