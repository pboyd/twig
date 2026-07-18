# Feature Specification: Add New Tasks to a Plan

**Feature Branch**: `066-add-task-to-plan`

**Created**: 2026-07-17

**Status**: Draft

**Input**: User description: "add new tasks to a plan. Often when adding a task it's something needs to be done today. Let's save the user a step and give them the option to add it to the day's plan. If it can done unobtrusively, let's also give them the option to add it to any other day's plan. Let's look at a few options for how the UI will work. This should apply to the TUI and the web app."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plan a new task for today as you create it (Priority: P1)

A user capturing a task they intend to do today marks it for today's plan from within the create form itself. When they save, the task is created and appears on today's plan, without them having to find the task again afterward and schedule it as a second action.

**Why this priority**: This is the case the feature exists for and the one the user identified as most common. It delivers the entire "save the user a step" value on its own, and is the smallest slice that is worth shipping.

**Independent Test**: Open the create-task form, choose "Today" on the plan control, save, and confirm the task exists in the task tree *and* appears as an untimed entry on today's plan. Fully testable without the other-day or tomorrow options existing.

**Acceptance Scenarios**:

1. **Given** the create-task form is open with the plan control on its default of "No plan", **When** the user selects "Today" and saves, **Then** the task is created and an untimed entry linking it appears on today's plan.
2. **Given** the create-task form is open, **When** the user saves without touching the plan control, **Then** the task is created and no plan entry is created on any day.
3. **Given** the user selected "Today" on the plan control, **When** saving fails to create the task, **Then** no plan entry is created and the user is told the task was not created.
4. **Given** the user selected "Today" and the task is created successfully, **When** adding it to the plan fails, **Then** the task remains created, the user is told it was created but not planned, and the failure does not discard their work.

---

### User Story 2 - Plan a new task for tomorrow or another day (Priority: P2)

The same control that offers "Today" also offers "Tomorrow" directly, and lets the user reach any other date without leaving the create form. A user capturing something they know is for next week schedules it in the same pass.

**Why this priority**: Extends the same saved step to days other than today. Valuable, but the user framed it as conditional ("if it can be done unobtrusively"), so it must not complicate the P1 path. Ships only after P1 works.

**Independent Test**: Open the create-task form, choose "Tomorrow", save, and confirm the task appears on tomorrow's plan and not today's. Then repeat via the arbitrary-date affordance with a date several days out.

**Acceptance Scenarios**:

1. **Given** the create-task form is open, **When** the user selects "Tomorrow" and saves, **Then** an untimed entry linking the task appears on tomorrow's plan and today's plan is unchanged.
2. **Given** the create-task form is open, **When** the user reaches the arbitrary-date affordance, picks a date, and saves, **Then** an untimed entry linking the task appears on that day's plan.
3. **Given** the user has picked an arbitrary date on the plan control, **When** they return the control to "No plan" and save, **Then** the task is created with no plan entry on any day.
4. **Given** the user picks a date in the past, **When** they save, **Then** the task is added to that past day's plan, consistent with how the system already treats scheduling on past days.

---

### User Story 3 - Consistent plan control across TUI and web (Priority: P3)

A user who works in both the terminal and the browser meets the same choices — no plan, today, tomorrow, or a specific date — in both create forms, with the same default and the same resulting behavior.

**Why this priority**: Parity is what makes the control learnable rather than surface-specific trivia, but each surface delivers value independently, so this is a consistency constraint rather than a prerequisite.

**Independent Test**: Create a task planned for today in the TUI and another in the web app; confirm both produce the same kind of plan entry on the same day, and that both forms default to not planning.

**Acceptance Scenarios**:

1. **Given** the same choice ("Today", "Tomorrow", or a given date) is made on either surface, **When** the task is saved, **Then** both surfaces produce an equivalent plan entry on the same day.
2. **Given** a freshly opened create-task form on either surface, **When** the user inspects the plan control, **Then** it reads "No plan".

---

### Edge Cases

- **Creating a goal, not a task**: Goals are not plannable, so the plan control does not appear on the goal create form.
- **Creating a subtask**: A subtask is planned exactly like a top-level task; the parent's plan state has no bearing on the child's.
- **"Today" spans midnight**: The day the user sees on the control is their local day. A form left open across midnight resolves "Today" and "Tomorrow" against the local day at the moment of saving, not at the moment the form was opened.
- **Task is also snoozed**: A task can be snoozed and planned at once. The plan entry is created for the chosen day regardless of the snooze date; the two settings are independent and the user is not blocked or warned.
- **Task created but planning fails**: Treated as a partial success, never a silent one — see User Story 1, scenario 4.
- **Editing an existing task**: The plan control is absent from the edit form. Existing tasks are already plannable from the task list, and duplicating that in the edit form would create two competing paths to the same outcome.
- **Unparseable or unreachable date**: If the arbitrary-date affordance yields no valid date, the control stays on its previous value rather than guessing a day.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The create-task form MUST offer a plan control that lets the user choose, before saving, whether the new task joins a day's plan and which day.
- **FR-002**: The plan control MUST offer "No plan", "Today", and "Tomorrow" as directly selectable choices, and MUST provide a way to reach any other calendar date.
- **FR-003**: The plan control MUST default to "No plan" on every newly opened create-task form, so that saving a task without engaging the control never schedules it.
- **FR-004**: On save with a day selected, the system MUST create the task and then add it to that day's plan as an untimed entry, so that it lands in the day's untimed group rather than claiming a time slot the user never chose.
- **FR-005**: "Today" and "Tomorrow" MUST resolve against the user's local date at the time of saving.
- **FR-006**: If task creation fails, the system MUST NOT add anything to any plan.
- **FR-007**: If task creation succeeds but adding to the plan fails, the system MUST keep the created task, report that the task was created but not planned, and MUST NOT discard the user's input.
- **FR-008**: After a successful save with a day selected, the affected day's plan MUST reflect the new entry without the user manually refreshing or re-navigating.
- **FR-009**: The plan control MUST be offered on both the TUI and the web create-task forms, with the same choices, the same default, and equivalent results.
- **FR-010**: The plan control MUST NOT appear when the form is creating a goal.
- **FR-011**: The plan control MUST NOT appear when the form is editing an existing task, leaving the existing task-list scheduling path as the single way to plan an existing task.
- **FR-012**: The TUI plan control MUST be reachable and operable by keyboard alone, consistent with the form's existing field navigation.
- **FR-013**: The web plan control MUST be operable by keyboard and expose its selected state to assistive technology, consistent with the app's existing form controls.
- **FR-014**: Choosing a plan day MUST NOT alter any other task attribute, including due date, snooze, estimate, and goal association.

### Chosen UI Approach

Three options were weighed per surface; the following were selected. Both surfaces express one model — *a single control whose value is either "no plan" or a specific day* — so the concept transfers between them.

**TUI — cycling selector plus calendar.** A "Plan" field takes its place in the create form's existing field order and cycles `No plan → Today → Tomorrow → No plan` with the left/right arrows, mirroring the Goal selector the form already has. The form's existing calendar key (`ctrl+g`, already bound on the Due and Snooze fields) opens the date picker to set an explicit date, which the field then displays in place of the named choices.

```
  Name          Write the spec
  Description
  Due
  Estimate      2
  Snooze until
▸ Plan         ‹ Today ›     ←/→ cycle · ctrl+g: summon the calendar
  Goal          ‹ Ship v2 ›

       [ Save ]   [ Cancel ]
```

*Why*: reuses two idioms the form already teaches — the cycling selector and the calendar key on date fields — so the control costs the user no new concepts. Today is one keystroke; any other day stays available without occupying form space.

**Web — segmented buttons plus revealed date input.** A labelled row of mutually exclusive choices sits below Description. A calendar affordance reveals a date input for any other day.

```
  Add to plan
  ┌─────────┬───────┬──────────┬────┐
  │ No plan │ Today │ Tomorrow │ 📅 │
  └─────────┴───────┴──────────┴────┘
              ^^^^^ selected
```

*Why*: shows the same choices as the TUI in the same order, keeps all of them one click away, and makes the current selection visible at a glance rather than hidden behind a popover.

**Rejected**: a post-save prompt on both surfaces (interrupts rapid capture and adds the step this feature set out to remove); a plain date text field in the TUI (makes the common "today" case cost typing); a checkbox plus disclosure on the web (splits "today" and "another day" across two dissimilar controls); reusing the task row's existing popover control inside the form (built as a row icon action, and reads oddly among labelled fields).

### Key Entities

- **Task**: The item being created. Gains no new stored attribute from this feature — the plan choice is a transient property of the create form, not of the task.
- **Plan entry**: An existing per-day item that may link a task. This feature creates one, untimed, on the chosen day, identical to what the existing scheduling path produces.
- **Plan day**: The local calendar day a plan entry belongs to. The user's choice resolves to exactly one, or to none.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can create a task and have it on today's plan in a single save, with no navigation away from the create form and no follow-up action.
- **SC-002**: Planning a new task for today costs at most one additional input beyond the keystrokes needed to create an unplanned task; planning it for tomorrow costs at most two.
- **SC-003**: Users who ignore the new control create tasks in exactly the same number of inputs, with exactly the same result, as before this feature.
- **SC-004**: 100% of saves with a day selected either place the task on that day's plan or tell the user plainly that it was not placed; no save silently drops the plan choice.
- **SC-005**: 0% of saves with the control left at its default create a plan entry on any day.
- **SC-006**: Both surfaces present the same four choices in the same order and the same default, such that a user who learns the control on one surface can operate it on the other without instruction.
- **SC-007**: The whole control is operable by keyboard alone on both surfaces.

## Assumptions

- **Untimed entries are the right landing spot.** The user chose a day, not a time, so the entry is created untimed rather than given a guessed start time. Existing auto-scheduling behavior is unchanged and out of scope.
- **Duration is left to the server's existing default.** The create form does not ask for one; the system's established default for task plan entries applies.
- **Creation only, not editing.** Existing tasks already have a scheduling path from the task list. Adding the control to the edit form would create a second, competing route to the same outcome, so it is excluded (FR-011).
- **"Today" and "Tomorrow" are the only named relative days.** Other relative shorthands ("next Monday", "this weekend") add cycling steps for diminishing return; arbitrary dates cover them.
- **No new stored state.** The plan choice does not persist between form openings and is not saved on the task; the default is always "No plan" (FR-003). This differs deliberately from the TUI's persisted task filters, because a filter describes a view while this choice performs a one-time write.
- **The existing scheduling capability is reused.** Adding a task to a day's plan is an existing, working operation; this feature changes where the user can invoke it, not what it does.
- **Goals are not plannable**, consistent with the current system, so the control is absent from the goal form (FR-010).
- **The create form's existing behaviors carry over unchanged**, including cancel/discard confirmation on a dirty form; a touched plan control counts as making the form dirty.
