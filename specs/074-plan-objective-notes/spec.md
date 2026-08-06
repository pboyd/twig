# Feature Specification: Plan Objectives and Notes

**Feature Branch**: `074-plan-objective-notes`

**Created**: 2026-08-06

**Status**: Draft

**Input**: User description: "Daily plans need a couple user-configurable fields: objective and notes. Both fields are optional. They are text fields and should be rendered as markdown when displayed. Objective is a short text field that the user can provide to give themselves a reminder of the main thing they need to get done that day. Notes is a longer text field that the user can use to write any details about the day they'd like to remember, or work through planning the day in writing. Objective, if it's provided, should be displayed in the TUI's planning tab in a full-width pane above the current day planner and details panes. If the day does not have an objective, then this pane should be omitted. `o` should allow the user to set the objective. Ideally, the edit form would be in the same location as the objective is displayed (unlike other forms, since this is only a single field, `Enter` should save it, `Esc` should cancel, and we don't need any buttons). Notes should be displayed in a new pane on the right, below the details pane. `n` should allow a user to edit the day's notes. When editing the notes, `ctrl+g` should open an external editor (like elsewhere in the program). The day's objective should be usable from the CLI. For example, `twig plan --date 2026-08-06 objective` should print the day's objective, and `twig plan --date 2026-08-06 objective 'my new objective'` should set the objective. As with other long-form text fields, notes do not need to be part of the CLI. Objectives and notes do not need to be in the web app right now."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Set and see the day's objective (Priority: P1)

A person planning their day wants one line at the top of the plan that answers "what is the main thing today?". While on the planning tab they press `o`, type the objective, and press Enter. The objective immediately appears in a full-width pane above the day's planner and detail panes, where it stays visible for the rest of the day's work. Days that have no objective show no such pane, so the planner keeps its full height.

**Why this priority**: This is the smallest slice that delivers the feature's core value — a persistent, visible reminder of the day's intent. It works with nothing else built.

**Independent Test**: On a day with no objective, confirm no objective pane is shown; press `o`, type text, press Enter, and confirm the objective pane appears with that text and persists after leaving and returning to the day.

**Acceptance Scenarios**:

1. **Given** a day with no objective, **When** the planning tab is viewed, **Then** no objective pane is rendered and no blank space is reserved for it.
2. **Given** a day with an objective, **When** the planning tab is viewed, **Then** the objective is shown in a full-width pane positioned above the day planner and detail panes.
3. **Given** the planning tab is showing a day, **When** `o` is pressed, **Then** a single-field editor opens in the same position the objective pane occupies (or would occupy), pre-filled with the day's current objective, with no buttons.
4. **Given** the objective editor is open with text entered, **When** Enter is pressed, **Then** the objective is saved for that day, the editor closes, and the objective pane shows the new text.
5. **Given** the objective editor is open with text changed, **When** Esc is pressed, **Then** nothing is saved, the editor closes, and the previously displayed objective (or absence of one) is restored.
6. **Given** a day with an objective, **When** the objective editor is opened, the text is cleared, and Enter is pressed, **Then** the objective is removed and the objective pane disappears.
7. **Given** an objective was set, **When** the user navigates to another day and back, **Then** the objective shown belongs to the day currently displayed, and each day carries its own independent objective.

---

### User Story 2 - Write and read the day's notes (Priority: P1)

The same person wants somewhere to think in writing: meeting details, decisions, a rough plan of attack. On the planning tab they press `n`, which opens a multi-line editor for the day's notes. They can type directly, or press `ctrl+g` to compose in their external editor as they do elsewhere in the program. Saved notes appear in a pane on the right, below the entry detail pane, where they can be read while working through the day.

**Why this priority**: Notes are the second half of what the user asked for and are independently valuable — a person could use notes without ever setting an objective.

**Independent Test**: Press `n` on the planning tab, type multi-line text, save, and confirm the text appears in the notes pane on the right below the details pane and persists across day navigation and restarts.

**Acceptance Scenarios**:

1. **Given** the planning tab is showing a day, **When** the tab is viewed, **Then** a notes pane occupies the region on the right below the entry detail pane.
2. **Given** a day with notes, **When** the planning tab is viewed, **Then** the notes content is shown in that pane.
3. **Given** a day with no notes, **When** the planning tab is viewed, **Then** the notes pane is shown empty, indicating notes can be added.
4. **Given** the planning tab is showing a day, **When** `n` is pressed, **Then** a multi-line notes editor opens pre-filled with the day's current notes.
5. **Given** the notes editor is open, **When** Enter is pressed, **Then** a new line is inserted rather than saving, because notes are multi-line.
6. **Given** the notes editor is open with edits, **When** the save action used by the program's other forms is performed, **Then** the notes are saved for that day and the notes pane shows the new content.
7. **Given** the notes editor is open with edits, **When** Esc is pressed, **Then** the edits are discarded and the previously saved notes are shown.
8. **Given** the notes editor is open, **When** `ctrl+g` is pressed, **Then** the current draft is handed to the user's external editor, and on exit the edited text returns to the notes editor — matching the external-editor behaviour used elsewhere in the program.
9. **Given** notes were saved for a day, **When** the user navigates to another day and back, **Then** the notes shown belong to the day currently displayed, and each day carries its own independent notes.

---

### User Story 3 - Objective and notes render as formatted markdown (Priority: P2)

Both fields are authored in markdown. When displayed, they should look the way markdown looks everywhere else in the program — emphasis, links, lists and inline code rendered, not shown as raw markup.

**Why this priority**: The fields are usable as plain text without this, so it sits below stories 1 and 2 — but the user asked for it explicitly and it is what makes longer notes readable.

**Independent Test**: Save an objective and notes containing emphasis, a link, a bullet list and inline code, then confirm both panes render them with the same formatting used for task descriptions.

**Acceptance Scenarios**:

1. **Given** an objective containing markdown constructs, **When** the objective pane is displayed, **Then** it is rendered using the same markdown presentation applied to descriptions elsewhere in the program.
2. **Given** notes containing headings, lists, emphasis, links and code, **When** the notes pane is displayed, **Then** they are rendered with that same markdown presentation.
3. **Given** the program is producing plain, unstyled output, **When** either field is displayed, **Then** it is presented as readable wrapped plain text, consistent with the plain-text fallback used elsewhere.
4. **Given** either field is being edited, **When** the editor is open, **Then** the editor shows the raw markdown source, not rendered output.

---

### User Story 4 - Read and set the objective from the command line (Priority: P2)

A person working in a terminal — or from a script or shell prompt — wants the day's objective without opening the interactive interface. `twig plan --date 2026-08-06 objective` prints it; `twig plan --date 2026-08-06 objective 'my new objective'` sets it.

**Why this priority**: This extends an already-delivered capability to a second surface. Valuable, but the interactive path in stories 1–2 carries the feature.

**Independent Test**: Set an objective through the command line, print it back through the command line, and confirm the interactive planning tab shows the same value for that day.

**Acceptance Scenarios**:

1. **Given** a day with an objective, **When** the objective command is run with no text argument for that day, **Then** the objective is printed and the command succeeds.
2. **Given** a day with no objective, **When** the objective command is run with no text argument, **Then** the command succeeds and prints nothing (no error, no placeholder).
3. **Given** any day, **When** the objective command is run with a text argument, **Then** that day's objective is set to the given text and the command succeeds.
4. **Given** a day with an objective, **When** the objective command is run with an empty-string argument, **Then** the day's objective is cleared.
5. **Given** no date option is supplied, **When** the objective command is run, **Then** it applies to the same default day the plan command already uses when no date is given.
6. **Given** an objective is set from the command line, **When** the planning tab for that day is opened, **Then** the objective pane shows that text.
7. **Given** an objective is set in the interactive interface, **When** the objective command is run for that day, **Then** it prints that text.

---

### Edge Cases

- **Day with no plan entries at all**: an objective and notes can still be set and are retained; they do not depend on the day having any scheduled work.
- **Objective or notes that are only whitespace**: treated as empty — the objective pane is omitted, the notes pane renders as empty, and the command line prints nothing.
- **Very long objective**: the objective pane does not grow without bound or break the layout; the planner and detail panes keep their positions.
- **Objective or notes longer than the available pane height**: as much as fits is shown; surrounding panes, the tab bar and the status line stay intact.
- **Very narrow terminal**: both panes remain readable and within their bounds; the planning tab layout is not damaged.
- **Notes long enough to squeeze the detail pane**: the detail pane and the notes pane divide the right-hand column so both remain usable.
- **Pressing `o` or `n` while another form, picker or editor is open on the planning tab**: does not open a second editor or discard the in-progress form.
- **Keys `o` and `n` on other tabs**: keep their existing meanings; the new bindings apply on the planning tab only.
- **External editor unavailable or exits non-zero while editing notes**: the notes editor is restored with the draft intact and the user is told; no notes are lost.
- **Save fails (e.g. the server is unreachable)**: the user is informed with the same error presentation used for other failed plan actions; the entered text is not silently discarded.
- **Day changing while the interface is open** (crossing midnight, or navigating days): the objective and notes displayed always match the day currently shown.
- **Markdown that is malformed or uses unsupported constructs**: displayed as text rather than causing an error or a blank pane.

## Requirements *(mandatory)*

### Functional Requirements

#### Data

- **FR-001**: Each day MUST be able to carry an objective (short text) and notes (long text), independently of one another and independently of whether the day has any plan entries.
- **FR-002**: Both fields MUST be optional; a day with neither MUST behave exactly as days do today.
- **FR-003**: Both fields MUST be scoped to a single user and a single day, and MUST persist across sessions until changed.
- **FR-004**: Both fields MUST be treated as markdown source when stored, and MUST be stored verbatim without reformatting.
- **FR-005**: Setting either field to empty or whitespace-only text MUST clear it, making the day equivalent to one that never had the field set.

#### Planning tab — objective

- **FR-006**: When the displayed day has an objective, the planning tab MUST show it in a full-width pane positioned above the day planner and detail panes.
- **FR-007**: When the displayed day has no objective, that pane MUST be omitted entirely, with no border, label or reserved blank space, and the remaining panes MUST use the space.
- **FR-008**: Pressing `o` on the planning tab MUST open a single-field objective editor, pre-filled with the displayed day's current objective.
- **FR-009**: The objective editor MUST appear in the same position as the objective pane — where the pane is shown, or where it would be shown if the day had an objective — rather than in the location used by other planning-tab forms.
- **FR-010**: The objective editor MUST have no buttons; Enter MUST save and close it, and Esc MUST cancel and close it, discarding changes.
- **FR-011**: Saving the objective MUST update the displayed objective pane immediately, including making it appear when an objective is first set and disappear when one is cleared.

#### Planning tab — notes

- **FR-012**: The planning tab MUST show a notes pane in the right-hand column, below the entry detail pane.
- **FR-013**: The notes pane MUST display the displayed day's notes, and MUST render as an empty pane when the day has none.
- **FR-014**: Pressing `n` on the planning tab MUST open a multi-line notes editor pre-filled with the displayed day's current notes.
- **FR-015**: In the notes editor, Enter MUST insert a newline; saving MUST use the same save action the program's other multi-line forms use, and Esc MUST cancel and discard changes.
- **FR-016**: In the notes editor, `ctrl+g` MUST hand the current draft to the user's external editor and return the edited text to the editor on exit, matching the external-editor behaviour used elsewhere in the program.
- **FR-017**: Saving notes MUST update the notes pane immediately.

#### Presentation

- **FR-018**: Both fields MUST be rendered as markdown when displayed, using the same rendering behaviour applied to task descriptions elsewhere in the interface, including the plain-text fallback when visual styling is unavailable.
- **FR-019**: Both editors MUST present the raw markdown source for editing, not rendered output.
- **FR-020**: Both panes MUST wrap their content to the width available and MUST NOT render outside their own bounds, at any supported terminal size.
- **FR-021**: When content exceeds the vertical space available, the pane MUST show as much as fits without distorting the calendar grid, pane borders, tab bar or status line.
- **FR-022**: Both panes MUST always show the values belonging to the day currently displayed, updating when the displayed day changes.
- **FR-023**: The planning tab's existing behaviour — entry selection, scheduling, editing, event creation, day navigation, the task picker and the calendar grid — MUST be unchanged, and the new `o` and `n` bindings MUST NOT alter the meaning of those keys on other tabs.
- **FR-024**: The new key bindings MUST appear in the planning tab's help alongside the existing ones.

#### Command line

- **FR-025**: An `objective` subcommand of the plan command MUST print the objective of the day it targets when invoked with no text argument, printing nothing when the day has no objective, and succeeding in both cases.
- **FR-026**: The same subcommand invoked with a text argument MUST set that day's objective to the given text, and MUST clear it when the argument is an empty string.
- **FR-027**: The subcommand MUST target the day given by the plan command's existing date option, and the plan command's existing default day when no date is given.
- **FR-028**: Values written through either surface MUST be immediately visible to the other; there is one objective per day, not one per interface.

#### Scope

- **FR-029**: Notes MUST NOT be exposed as a command-line subcommand in this feature.
- **FR-030**: Neither field needs to appear in the web application in this feature, and the web application's existing behaviour MUST be unaffected.

### Key Entities

- **Plan day**: a single user's single calendar day of planning. Today it is defined only by the plan entries that reference it; this feature gives the day itself two attributes of its own — an objective and notes — that exist whether or not the day has entries.
- **Objective**: optional short markdown text naming the main thing to get done that day. One per day per user.
- **Notes**: optional long-form markdown text about the day. One per day per user.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A person can set the day's objective from the planning tab in two actions — press `o`, type, press Enter — without leaving the tab or opening any other view.
- **SC-002**: Once set, the objective is visible on the planning tab at all times while that day is displayed, with no navigation or key press required to reveal it.
- **SC-003**: On days with no objective, the planning tab's usable planner height is identical to what it is today — no space is consumed by the new pane in 100% of such cases.
- **SC-004**: Objectives and notes survive closing and reopening the program, and are correct for the day displayed in 100% of day-navigation cases.
- **SC-005**: Objective and notes content containing emphasis, links, lists and inline code renders with the same formatting as task descriptions at equal width in 100% of tested cases.
- **SC-006**: The planning tab layout — grid alignment, pane boundaries, tab bar, status line — remains intact for content of any length or width, including at the narrowest supported terminal size, in 100% of tested cases.
- **SC-007**: A person can read or set any day's objective from the command line in a single command, and the value round-trips between the command line and the interactive interface with no discrepancy.
- **SC-008**: Adding both panes introduces no perceptible delay when opening the planning tab or moving between days.

## Assumptions

- The objective and notes attach to the day, not to any plan entry, and are per-user like all other plan data. Two users' objectives for the same date are independent.
- The notes pane is always present on the planning tab, even when empty, rather than appearing only once notes exist. A stable layout and a visible affordance for `n` outweigh reclaiming the space, and unlike the objective pane the notes pane sits at the bottom of a column where an empty region is not disruptive. This deliberately differs from the objective pane, whose omission when empty the user asked for explicitly.
- "Save" in the notes editor means the save action the program's existing multi-line forms already use, so the notes editor behaves like every other multi-line form. Only the objective editor gets the special single-key Enter-to-save treatment the user asked for, because it is a single-line field.
- Clearing a field is expressed by saving it empty; no separate delete or unset action is introduced on either surface.
- The objective is a single-line field. It is markdown, so inline constructs render, but it is not intended to hold paragraphs — that is what notes are for.
- Markdown rendering reuses the interface's existing presentation of markdown rather than defining new formatting, so both panes inherit any future changes to it.
- The right-hand column is shared between the existing entry detail pane and the new notes pane; how the available height is divided between them is a layout detail left to implementation, provided both stay usable at supported terminal sizes.
- `o` and `n` are currently unbound on the planning tab, so no existing planning-tab shortcut is displaced. `n` retains its existing meaning on the tasks tab.
- Reading the objective from the command line prints only the objective text, with no label or decoration, so it can be used directly in scripts and shell prompts.
- The web application is explicitly out of scope for this feature; the underlying day attributes may later be surfaced there without changing what is specified here.
