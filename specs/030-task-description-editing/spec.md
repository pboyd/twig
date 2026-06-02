# Feature Specification: Multiline Task Descriptions & External Editor

**Feature Branch**: `030-task-description-editing`

**Created**: 2026-06-01

**Status**: Draft

**Input**: User description: "users sometimes want longer task descriptions, but there are two problems: whitespace is not preserved (it's not clear to me if it's not stored or not rendered, but either way the effect is the same) and the editor is minimal. We need whitespace that the user enters to be shown when the task is displayed (particularly newlines). And ctrl+g when the description is being edited should launch the user's editor (use $EDITOR or fallback to vim) and allow them to edit the text there."

## Clarifications

### Session 2026-06-01

- Q: How faithfully should whitespace inside a description be preserved when displayed? → A: Preserve newlines, blank lines, and per-line leading indentation; mid-line runs of multiple spaces may collapse; long lines still wrap to the available width.
- Q: How should the `EDITOR` value be interpreted when it contains arguments (e.g. "code --wait")? → A: Support arguments — split `EDITOR` on whitespace so values like "code --wait" or "emacsclient -nw" work, passing the temporary file as the final argument.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Whitespace and newlines preserved on display (Priority: P1)

A user writes a task description that spans multiple lines — for example a short
checklist, a paragraph break, or an indented note. When they later view that
task's details, the description appears the way they typed it, with their line
breaks and blank lines intact, rather than collapsed into a single run-on
paragraph.

**Why this priority**: This is the core complaint. Today multiline content the
user already typed is silently flattened when displayed, making longer
descriptions effectively unusable. Fixing display delivers immediate value even
without any editor changes, and it works for descriptions that already exist.

**Independent Test**: Create or edit a task whose description contains multiple
lines and a blank line between paragraphs, save it, then open the task's
details view and confirm the line breaks and blank line are shown exactly as
entered.

**Acceptance Scenarios**:

1. **Given** a task whose description is "Line one\nLine two", **When** the user
   views the task details, **Then** "Line one" and "Line two" appear on separate
   lines.
2. **Given** a task whose description contains a blank line between two
   paragraphs, **When** the user views the task details, **Then** the blank line
   separating the paragraphs is preserved.
3. **Given** a description line that is longer than the available display width,
   **When** the task details are shown, **Then** that line is wrapped to fit the
   width while the user's own line breaks remain as separate lines.
4. **Given** a description with a line indented by leading spaces, **When** the
   task details are shown, **Then** that line's leading indentation is preserved.
5. **Given** an existing task created before this change whose description
   contains newlines, **When** the user views it, **Then** the stored newlines
   are now displayed (no re-entry required).

---

### User Story 2 - Edit a description in the external editor (Priority: P2)

While editing a task's description in the interactive interface, the user
presses `ctrl+g`. Their preferred terminal editor opens with the current
description text loaded. They edit freely using the full power of that editor —
including easy multiline editing — then save and quit. The interface resumes
with the edited text in place in the description field, ready to be saved with
the rest of the task.

**Why this priority**: This removes the second pain point (the minimal built-in
editor) and makes authoring long descriptions comfortable. It depends on
nothing from Story 1 to function, but Story 1 is what makes the resulting
multiline text actually display correctly, so it is sequenced second.

**Independent Test**: While editing a task description, press `ctrl+g`, confirm
the configured editor opens pre-loaded with the current text, make a change,
save and quit the editor, and confirm the changed text appears in the
description field back in the interface.

**Acceptance Scenarios**:

1. **Given** the user is editing a description field, **When** they press
   `ctrl+g`, **Then** an external editor launches containing the current
   description text.
2. **Given** the `EDITOR` environment variable names an editor, **When** the
   user triggers the external editor, **Then** that editor is launched.
3. **Given** the `EDITOR` environment variable is unset or empty, **When** the
   user triggers the external editor, **Then** `vim` is launched as the
   fallback.
4. **Given** the user edits the text in the external editor and saves and quits,
   **When** control returns to the interface, **Then** the description field
   contains the edited text.
5. **Given** the user opens the external editor and quits without saving any
   changes, **When** control returns to the interface, **Then** the description
   field is unchanged from before.
6. **Given** the user has finished using the external editor, **When** control
   returns, **Then** the interface is restored to a correct visual state and the
   user can continue editing or save the task normally.

---

### Edge Cases

- **Empty description**: Pressing `ctrl+g` on an empty description opens the
  editor with empty content; quitting without typing leaves the description
  empty.
- **Editor exits non-zero / is interrupted**: If the external editor exits with
  an error or is killed, the description field retains its prior value and the
  user is returned to the interface without losing the rest of the form.
- **Configured editor cannot be launched**: If neither the configured editor nor
  the `vim` fallback can be started, the user is informed and returned to the
  interface with the description unchanged.
- **Trailing newline from editors**: Many editors append a trailing newline on
  save; the displayed/stored description should not accumulate spurious blank
  lines from repeated edit cycles. *(See Assumptions.)*
- **Very long descriptions**: A description far taller than the viewport should
  display without breaking layout (scroll/truncation behavior follows existing
  details-pane behavior).
- **Width changes / narrow terminals**: Line wrapping adapts to the available
  width while user-entered line breaks are still honored.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST preserve user-entered newlines when displaying a
  task description, rendering each entered line break as a line break in the
  details view.
- **FR-002**: The system MUST preserve, when displaying a description, the
  user's newlines, blank lines between paragraphs, and the leading indentation
  of each line. Runs of multiple spaces in the middle of a line MAY be collapsed
  to a single space.
- **FR-003**: The system MUST wrap individual description lines that exceed the
  available display width, without discarding the user's own line breaks; a
  wrapped continuation does not introduce an additional user-visible line break
  in the stored text.
- **FR-004**: The display behavior MUST apply to descriptions already stored
  before this change (no data migration or re-entry required).
- **FR-005**: While a task description is being edited, the system MUST provide a
  `ctrl+g` action that launches an external text editor pre-loaded with the
  current description text.
- **FR-006**: The system MUST select the external editor from the `EDITOR`
  environment variable, falling back to `vim` when `EDITOR` is unset or empty.
  An `EDITOR` value that includes arguments (e.g. "code --wait") MUST be honored
  by splitting it on whitespace and passing the temporary file as the final
  argument.
- **FR-007**: When the external editor exits after a save, the system MUST
  replace the in-progress description field value with the text the user saved.
- **FR-008**: When the external editor exits without changes (or the user
  abandons the edit), the system MUST leave the description field value
  unchanged.
- **FR-009**: After the external editor exits (success or failure), the system
  MUST restore the interface to a usable state and allow the user to continue
  editing other fields and save the task.
- **FR-010**: If the external editor fails to launch, the system MUST inform the
  user and preserve the existing description and the rest of the in-progress
  form.
- **FR-011**: The external-editor action MUST NOT, by itself, persist the task;
  the edited description becomes part of the normal task save flow.

### Key Entities *(include if feature involves data)*

- **Task Description**: The free-text notes field of a task. May contain
  newlines and intra-line whitespace that carry meaning for the reader. Stored
  as part of the task and displayed in the task details view.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A description entered with multiple lines and a blank-line
  paragraph break displays with 100% of those line breaks intact in the details
  view.
- **SC-002**: Descriptions created before this change that contain newlines
  display those newlines without any user action or data migration.
- **SC-003**: A user can open the external editor, edit a multiline description,
  and return to the interface with their changes applied, in a single `ctrl+g`
  round trip.
- **SC-004**: The external editor honors the user's `EDITOR` preference in 100%
  of cases where it is set — including values that carry arguments such as
  "code --wait" — and uses `vim` in 100% of cases where it is not.
- **SC-005**: Repeatedly opening and saving a description in the external editor
  without intentional changes does not alter the visible description (no
  accumulation of blank lines or whitespace drift).
- **SC-006**: No existing task-editing or task-display workflow regresses; users
  who never press `ctrl+g` and never use multiline text see unchanged behavior
  apart from whitespace now being honored.

## Assumptions

- This feature targets the interactive terminal interface (TUI), where task
  descriptions are edited in a multiline field and shown in a details pane;
  that is where both the "whitespace not shown" and "minimal editor" problems
  occur. The web client is out of scope for this feature.
- The description is already stored faithfully (including newlines); the
  observed loss of whitespace is a display-time behavior. The fix therefore
  centers on rendering and does not require a storage/schema change. *(Verified
  during analysis: the details renderer collapses whitespace; the edit field
  already accepts multiline input.)*
- "Preserve whitespace" is scoped (per clarification) to newlines, blank lines,
  and per-line leading indentation; mid-line runs of multiple spaces may collapse
  and long lines are still wrapped to the viewport width.
- A single trailing newline added by an editor on save is normalized so that
  repeated edit cycles do not accumulate blank lines; the user's intentional
  internal blank lines are kept.
- `ctrl+g` is available (not already bound to a conflicting action) within the
  description-editing context; it activates only while the description field is
  focused/being edited.
- The external editor runs in the same terminal, taking over the screen while
  active; the interface redraws correctly when the editor exits. This is
  standard "shell out to $EDITOR" behavior.
- When no TTY/terminal is available (e.g., non-interactive context), the
  external-editor action is simply unavailable rather than an error.
