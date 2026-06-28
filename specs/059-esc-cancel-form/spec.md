# Feature Specification: Esc Cancels TUI Forms

**Feature Branch**: `059-esc-cancel-form`

**Created**: 2026-06-28

**Status**: Draft

**Input**: User description: "In the TUI, `Esc` should cancel any form. If canceling the form would cause the user to lose any text they have typed, then it should ask for confirmation."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Quickly dismiss a form I didn't change (Priority: P1)

A user opens a form in the TUI (for example, to create or edit a task, goal, plan entry, or status update), then decides not to proceed. Because they have not entered or altered anything, pressing `Esc` immediately closes the form and returns them to where they were, with nothing saved.

**Why this priority**: This is the most common cancel case and the core of the request — `Esc` must reliably back out of a form. Without it, users are stuck reaching for the explicit Cancel control on every form. It also delivers a complete, demonstrable slice on its own.

**Independent Test**: Open any TUI form without changing any field, press `Esc`, and confirm the form closes with no changes persisted and focus returns to the prior view.

**Acceptance Scenarios**:

1. **Given** a newly opened blank form with no text entered, **When** the user presses `Esc`, **Then** the form closes immediately and the user returns to the previous screen with nothing saved.
2. **Given** a pre-filled edit form whose fields the user has not modified, **When** the user presses `Esc`, **Then** the form closes immediately without a confirmation prompt and no changes are saved.
3. **Given** a form was just dismissed with `Esc`, **When** the user looks at the underlying view, **Then** their previous selection/cursor position is preserved.

---

### User Story 2 - Protect work I've typed from accidental loss (Priority: P1)

A user opens a form and types or edits content, then presses `Esc`. Because abandoning the form would discard their unsaved input, the system first asks them to confirm. They can confirm to discard and close, or decline to stay in the form with their input intact.

**Why this priority**: Equally critical to the request — the confirmation guard is the explicit second half of the requirement and prevents data loss from a single accidental keypress. Together with Story 1 it forms the complete intended behavior.

**Independent Test**: Open a form, type into at least one field, press `Esc`, verify a confirmation prompt appears; confirm and verify the form closes discarding input; repeat and decline, verifying input is retained.

**Acceptance Scenarios**:

1. **Given** a form where the user has entered text that is not yet saved, **When** the user presses `Esc`, **Then** a confirmation prompt appears asking whether to discard the unsaved input.
2. **Given** the discard confirmation prompt is showing, **When** the user confirms discarding, **Then** the form closes, the typed input is discarded, and nothing is saved.
3. **Given** the discard confirmation prompt is showing, **When** the user declines, **Then** the prompt closes and the user is returned to the form with all previously entered input still present.
4. **Given** a pre-filled edit form, **When** the user changes a field's value and presses `Esc`, **Then** the confirmation prompt appears because the form now differs from its opened state.
5. **Given** a pre-filled edit form whose field the user changed and then manually restored to its original value, **When** the user presses `Esc`, **Then** the form closes without a confirmation prompt because no effective change remains.

---

### User Story 3 - Consistent cancel across every form (Priority: P2)

A user who has learned that `Esc` cancels one form expects the same key to cancel every other form in the TUI the same way, including forms that have nested overlays (such as the calendar date picker). `Esc` first closes an open nested overlay; a subsequent `Esc` then applies the cancel-the-form behavior.

**Why this priority**: Consistency is what makes the behavior trustworthy, but the per-form mechanics in Stories 1 and 2 already deliver the essential value. This story ensures uniform coverage and well-defined layering with existing `Esc`-driven overlays.

**Independent Test**: Visit each form type in the TUI and confirm `Esc` cancels per Stories 1 and 2; for a form with the calendar open, confirm the first `Esc` closes the calendar and the next `Esc` cancels the form.

**Acceptance Scenarios**:

1. **Given** any form type in the TUI (task create/edit, subtask, goal create/edit, goal's new-task, plan entry forms, and status-update compose), **When** the user presses `Esc`, **Then** the form cancels following the same rules defined in Stories 1 and 2.
2. **Given** a form with a nested overlay open (e.g., the calendar date picker), **When** the user presses `Esc`, **Then** the nested overlay closes and the form remains open.
3. **Given** the user has dismissed a nested overlay with `Esc`, **When** the user presses `Esc` again, **Then** the cancel-the-form behavior applies (immediate close or confirmation depending on unsaved input).

---

### Edge Cases

- **Whitespace-only input**: Input that consists solely of whitespace is treated as no meaningful change, so `Esc` cancels without a confirmation prompt.
- **Discard prompt dismissal**: Pressing `Esc` while the discard confirmation prompt is showing declines the discard (keeps the form open with input intact), mirroring "no".
- **Selection-only forms**: Forms or overlays that involve no free-text entry (for example, a move-target picker or a goal link/unlink picker) cancel immediately on `Esc` with no confirmation, since there is no typed text to lose.
- **Pre-filled then cleared**: If the user clears a value that was pre-filled when the form opened, that counts as a change and triggers the confirmation prompt on `Esc`.
- **External editor content**: Content brought back into a description field from the external editor counts toward "unsaved input" when determining whether to confirm on `Esc`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Pressing `Esc` while any TUI form is open MUST initiate canceling that form.
- **FR-002**: When the form has no unsaved input (no field differs from its state when the form was opened), `Esc` MUST close the form immediately without a confirmation prompt and without saving any changes.
- **FR-003**: When canceling would discard unsaved input (one or more fields differ from their state when the form was opened), `Esc` MUST present a confirmation prompt before closing.
- **FR-004**: From the confirmation prompt, the user MUST be able to confirm discarding, which closes the form and discards all unsaved input without saving.
- **FR-005**: From the confirmation prompt, the user MUST be able to decline, which returns them to the form with all previously entered input preserved.
- **FR-006**: Canceling a form (with or without confirmation) MUST NOT persist any changes and MUST return the user to the view they were in before opening the form, preserving their prior cursor/selection.
- **FR-007**: The cancel-on-`Esc` behavior MUST apply uniformly to every TUI form, including task create/edit, subtask create, goal create/edit, the goal's new-task form, plan entry forms (timed task, untimed entry, event, and edit), and the status-update compose form.
- **FR-008**: When a form has a nested overlay open (such as the calendar date picker), `Esc` MUST close the nested overlay first and leave the form open; a subsequent `Esc` MUST apply the form cancel behavior.
- **FR-009**: The "unsaved input" determination MUST compare current field values to the values present when the form opened, so that reverting a field to its original value results in no confirmation prompt.
- **FR-010**: Whitespace-only differences MUST be treated as no meaningful change for the purpose of the confirmation decision.
- **FR-011**: Forms or overlays that accept no free-text entry MUST cancel immediately on `Esc` without a confirmation prompt.
- **FR-012**: The confirmation prompt MUST clearly communicate that proceeding discards unsaved input, and MUST make the discard and keep-editing choices discoverable.

### Key Entities

- **Form**: Any TUI interface that collects user input before an explicit save action. Includes the task create/edit form (and subtask), the goal create/edit form, the goal's new-task form, the plan entry forms, and the status-update compose form. Each form has an "opened state" snapshot of its initial field values.
- **Unsaved input**: The condition where a form's current field values differ meaningfully (ignoring whitespace-only differences) from its opened state.
- **Discard confirmation**: A transient prompt shown only when canceling a form with unsaved input, offering to discard-and-close or keep-editing.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From every TUI form, pressing `Esc` cancels and returns to the prior view in a single keypress when there is no unsaved input (zero forms require the explicit Cancel control to back out).
- **SC-002**: 100% of cancel attempts on a form with unsaved input present a confirmation before any input is discarded — no single `Esc` keypress silently destroys typed content.
- **SC-003**: When a user declines the discard confirmation, 100% of their previously entered input is still present in the form.
- **SC-004**: Across all form types covered by this feature, `Esc` behavior is identical (immediate cancel when clean, confirm when dirty), verified for each form type.
- **SC-005**: For forms with a nested overlay, the first `Esc` closes the overlay and the second `Esc` reaches the form-cancel behavior in 100% of cases.

## Assumptions

- "Lose any text they have typed" is interpreted as having unsaved input — current field values differing from the values present when the form opened — covering both brand-new entries and edits to pre-filled fields.
- A pre-filled edit form that the user opens and closes without changing anything is considered clean and cancels without a confirmation prompt.
- Reverting a changed field back to its original value before pressing `Esc` makes the form clean again (no confirmation), since the decision compares against the opened-state snapshot.
- Whitespace-only changes do not count as meaningful unsaved input.
- The existing layered `Esc` behavior is preserved: an open nested overlay (e.g., the calendar date picker) consumes the first `Esc`, and form-cancel applies on the next `Esc`.
- Non-text-entry overlays (move-target picker, goal link/unlink pickers, existing delete/quit confirmations) are out of scope for the confirmation guard; they continue to cancel immediately on `Esc`.
- This feature changes only TUI client behavior; no server, API, or data-model changes are required.
- The explicit on-screen Cancel control on forms continues to work as before; `Esc` is an additional, consistent path to the same outcome.
