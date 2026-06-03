# Feature Specification: Plan Entry Form Parity with Task Form

**Feature Branch**: `034-plan-form-parity`

**Created**: 2026-06-02

**Status**: Draft

**Input**: User description: "the task form and plan entry forms look very different without justification. These forms should be very similar, and the task form is far and away better, so let's fix the plan form. Preserve the original value in the edit form and remove the `blank=keep` text; add blank lines between the fields; add Save and Cancel buttons; remove `enter` to save the form; change the help text line to match the tasks form: `Ctrl+S: save  Esc: cancel  Tab: next field`."

## Clarifications

### Session 2026-06-02

- Q: How should a user unschedule a previously-scheduled entry from the new form, now that the `null`/`blank=keep` sentinels are removed? → A: Clear the pre-filled Start field and save (mirrors the task form unsetting an optional field).
- Q: Should the parity changes apply only to the edit-entry form or to all three shared planning forms? → A: All three forms (schedule task, add event, edit entry); pre-filling applies to the edit-entry form.
- Q: Should each plan form render a title header matching the task form's `Edit Task #N`? → A: Yes — render a per-mode title ("Edit entry" / "Schedule task" / "Add event") above the fields, followed by a blank line.
- Q: What format should the pre-filled Start and Duration values use? → A: Start as zero-padded 24-hour `HH:MM` (e.g. `09:00`); Duration as compact unit form (e.g. `30m`, `1h30m`).
- Q: When the user clears the Start field to unschedule, how is the Duration field treated? → A: Ignore Duration on save (untimed entries show no time window).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Edit a plan entry with a familiar, polished form (Priority: P1)

A user viewing the Planning tab opens the edit form for an existing plan entry. The form looks and behaves the same as the task edit form they already know: the current values are pre-filled, fields are visually separated, there are clear Save and Cancel buttons, and the help line at the bottom describes the same keys in the same order.

**Why this priority**: This is the entire purpose of the feature. The plan edit form is the primary surface being changed, and bringing it to parity with the task form is what delivers the value (consistency, discoverability, and reduced confusion).

**Independent Test**: Open the Planning tab, select an existing entry, open the edit form, and confirm the form shows the entry's current name/start/duration pre-filled, blank lines between fields, Save and Cancel buttons, the new help line, and that Enter does not submit the form. Fully testable on its own.

**Acceptance Scenarios**:

1. **Given** a scheduled plan entry named "Farm pigs" starting at 09:00 for 30m, **When** the user opens its edit form, **Then** the form shows the title "Edit entry", the Name field shows "Farm pigs", the Start field shows `09:00`, and the Duration field shows `30m` — none of them show `blank=keep` text.
2. **Given** the edit form is open, **When** the user reads the form, **Then** there is a blank line separating each field from the next (Name, Start, Duration), matching the task form's spacing.
3. **Given** the edit form is open, **When** the user looks below the fields, **Then** a `[ Save ]` and a `[ Cancel ]` button are displayed, and the currently focused button is visually indicated, matching the task form's button styling.
4. **Given** the edit form is open with focus in any text field, **When** the user presses Enter, **Then** the form is NOT submitted.
5. **Given** the edit form is open, **When** the user reads the bottom help line, **Then** it reads exactly `Ctrl+S: save  Esc: cancel  Tab: next field`.
6. **Given** the edit form is open, **When** the user presses Ctrl+S (or focuses the Save button and activates it), **Then** the entry's changes are saved.
7. **Given** the edit form is open, **When** the user presses Tab repeatedly, **Then** focus cycles through the text fields and the Save and Cancel buttons.
8. **Given** a scheduled entry's edit form is open with a pre-filled Start time, **When** the user clears the Start field and saves, **Then** the entry is unscheduled (moved to untimed) and any value in the Duration field is ignored.

---

### User Story 2 - Consistent presentation across all planning forms (Priority: P2)

A user works with all three planning forms — scheduling a task, adding an event, and editing an entry. Because they share a rendering surface, all three present the same conventions: blank lines between fields, Save and Cancel buttons, no Enter-to-save, and the same bottom help line.

**Why this priority**: The three forms share one rendering path, so leaving the schedule/add-event forms in the old style would create a new inconsistency. Extending parity to all three avoids an odd-one-out and keeps the change cohesive. It is P2 because the edit form (P1) is the explicitly requested surface; the other two ride along on the same change.

**Independent Test**: Open the Schedule-task form and the Add-event form, and confirm each shows blank-line field separation, Save/Cancel buttons, the new help line, and that Enter does not submit. Testable independently of the edit form.

**Acceptance Scenarios**:

1. **Given** the Schedule-task form is open, **When** the user reads it, **Then** it shows the title "Schedule task", blank lines between fields, Save/Cancel buttons, and the help line `Ctrl+S: save  Esc: cancel  Tab: next field`.
2. **Given** the Add-event form is open, **When** the user reads it, **Then** it shows the title "Add event", blank lines between fields, Save/Cancel buttons, and the help line `Ctrl+S: save  Esc: cancel  Tab: next field`.
3. **Given** any of the three planning forms is open with focus in a text field, **When** the user presses Enter, **Then** the form is NOT submitted.

---

### Edge Cases

- **Untimed entry**: When editing an entry that has no scheduled start (an untimed/backlog entry), the Start and Duration fields have no current value to pre-fill, so they appear empty. Leaving them empty MUST keep the entry untimed (no accidental scheduling).
- **Unscheduling a scheduled entry**: With pre-filled values replacing the old `blank=keep` / `null=unschedule` placeholder convention, the user unschedules a scheduled entry by clearing the (pre-filled) Start field and saving. This mirrors the task form, where clearing an optional field unsets that attribute. Any value remaining in the Duration field is ignored when the entry is unscheduled. See FR-009.
- **Empty name**: Saving with an empty Name field MUST be rejected with a validation error (unchanged from current behavior).
- **Invalid start/duration**: Saving with an unparseable start time or duration MUST be rejected with a validation error, and the form remains open.
- **No-op save**: Saving without changing any field MUST close the form without error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The plan entry edit form MUST pre-fill each editable field (Name, Start, Duration) with the entry's current value, rather than leaving Start and Duration blank.
- **FR-002**: The plan entry edit form MUST NOT display `blank=keep`, `null=unschedule`, or similar sentinel hint text in field labels or placeholders.
- **FR-003**: The plan entry edit form MUST render a blank line between adjacent fields, matching the vertical spacing of the task form.
- **FR-004**: The plan entry edit form MUST display a Save button and a Cancel button, styled and labeled consistently with the task form (`[ Save ]` / `[ Cancel ]`), including a visual focus indicator on whichever button is focused.
- **FR-005**: Keyboard focus cycling (Tab / Shift-Tab) MUST include the Save and Cancel buttons in the focus order, matching the task form.
- **FR-006**: Pressing Enter while a text field is focused MUST NOT submit the form.
- **FR-007**: The form MUST be savable via Ctrl+S and via activating the focused Save button.
- **FR-008**: The form's bottom help line MUST read exactly `Ctrl+S: save  Esc: cancel  Tab: next field`.
- **FR-009**: The form MUST continue to support all editing outcomes currently available: renaming the entry, changing its start time, changing its duration, and unscheduling it. Unscheduling MUST be achievable by clearing the pre-filled Start field and saving, which moves the entry to untimed; any value in the Duration field MUST be ignored when unscheduling.
- **FR-012**: The parity changes (blank-line field separation, Save/Cancel buttons, focus order including buttons, removal of Enter-to-save, and the updated help line) MUST apply to all three planning forms that share the rendering surface: schedule task, add event, and edit entry. Value pre-filling (FR-001) applies wherever existing values are present (i.e., the edit-entry form).
- **FR-013**: Each planning form MUST display a title line above its fields, followed by a blank line: "Edit entry" (edit), "Schedule task" (schedule), or "Add event" (add event), matching the task form's title treatment.
- **FR-014**: Pre-filled values MUST use formats that re-parse without modification: Start as zero-padded 24-hour `HH:MM` (e.g. `09:00`), and Duration in compact unit form (e.g. `30m`, `1h30m`). Saving an unchanged pre-filled form MUST produce no validation errors and no unintended changes.
- **FR-010**: Cancelling the form (Esc or activating the Cancel button) MUST discard changes and return to the plan view without modifying the entry.
- **FR-011**: Existing validation behavior MUST be preserved: empty name is rejected, and unparseable start/duration values are rejected with the form remaining open.

### Scope

The plan form rendering surface is shared by three planning modes — scheduling a task, adding an event, and editing an existing entry. The user's request and screenshots concern the **edit entry** form, but because all three modes share the rendering surface, the parity changes apply to all three (see FR-012). Value pre-filling specifically applies to the edit-entry form, which is the only mode with existing values to pre-fill.

### Key Entities

- **Plan entry**: A scheduled or untimed item on a day's plan. Relevant attributes for this feature: name, start time (optional), and duration (optional). May be a scheduled task or a standalone event.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A side-by-side comparison of the task edit form and the plan edit form shows identical structural conventions: a title line, pre-filled values, blank-line field separation, Save/Cancel buttons with focus indication, and the same bottom help line text.
- **SC-002**: 100% of fields in the plan edit form display the entry's current value (Start as `HH:MM`, Duration as compact unit form) when opened on an existing scheduled entry, and saving the unchanged form produces no validation error and no changes (no `blank=keep`-style placeholders shown).
- **SC-003**: Pressing Enter in a text field of the plan edit form never submits the form (0% submission rate from Enter).
- **SC-004**: All editing outcomes available before the change (rename, reschedule, change duration, unschedule) remain achievable after the change.
- **SC-005**: The bottom help line of the plan edit form is character-for-character identical to the task form's help line.

## Assumptions

- The task edit form (`Edit Task #N`) is the reference design; "parity" means matching its field pre-fill behavior, spacing, button presentation, focus order, save keys, and help-line wording, not necessarily its exact field set (the plan form has Name/Start/Duration rather than Name/Description/Due/Estimate).
- The visual focus indicator for buttons follows the task form's convention (e.g., `[>Save<]` when focused).
- The change is confined to the interactive TUI Planning tab; the non-interactive CLI subcommands for plan editing are out of scope.
- Underlying save/move/rename/unschedule operations against the backend are unchanged; only the form's presentation and the mapping of field input to those operations change.
