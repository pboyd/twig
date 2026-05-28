# Feature Specification: TUI Theme Pass

**Feature Branch**: `018-tui-theme-pass`

**Created**: 2026-05-28

**Status**: Draft

**Input**: User description: "for the plan in `docs/plans/2026-05-28-tui-theme-pass-design.md`"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Clearly framed, focus-aware panels (Priority: P1)

A person working in the interactive TUI sees the task list and the task
details as two distinctly framed regions rather than two blocks of text
separated by whitespace. Each region carries a visible label, and the region
they are currently acting on is visually distinguished from the inactive one,
so at a glance they always know where their input will land.

**Why this priority**: The lack of framing is the most disorienting part of the
current presentation. Distinct, labeled, focus-aware panels deliver the single
biggest legibility gain and are independently valuable even if no other change
ships.

**Independent Test**: Open the TUI, observe that the task list and details are
each enclosed in a labeled frame, move focus between panes, and confirm the
active pane is visibly distinguished while behavior (selection, navigation) is
unchanged.

**Acceptance Scenarios**:

1. **Given** the TUI is open on a terminal that supports styling, **When** the
   user views the screen, **Then** the task list and the details appear as two
   separately framed regions, each with a title identifying it.
2. **Given** the task list pane has focus, **When** the user views the screen,
   **Then** the task list frame is visually emphasized and the details frame is
   de-emphasized; **When** focus moves to the details pane, the emphasis
   follows the focus.
3. **Given** the edit/new-task form view or the move-task view is showing,
   **When** the user views the screen, **Then** the same framed treatment is
   applied so all list-mode layouts look consistent.

---

### User Story 2 - Honest fold and completion indicators (Priority: P2)

A person scanning the task tree can tell, for each row, two independent things:
whether the row has hidden children that can be unfolded, and whether the task
is complete. These two states are shown by distinct, intuitive indicators
rather than overloaded onto one marker.

**Why this priority**: The current `[-]`/`[+]` marker conflates "this row folds"
with nothing about completion, and completion is conveyed only by strikethrough.
Separating fold and completion into recognizable indicators removes ambiguity,
but it depends on the row-rendering changes and is less foundational than
framing.

**Independent Test**: Build a tree with collapsible parents, leaf tasks, and a
mix of complete and incomplete tasks; confirm a fold indicator appears only on
rows that have hidden/expandable children, and a completion indicator on every
row reflects that task's done state.

**Acceptance Scenarios**:

1. **Given** a row that has children which can be expanded or collapsed,
   **When** the tree is rendered, **Then** that row shows a fold indicator whose
   direction reflects whether it is currently expanded or collapsed.
2. **Given** a leaf row or a row with no collapsible children, **When** the tree
   is rendered, **Then** that row shows no fold indicator.
3. **Given** any task row, **When** the tree is rendered, **Then** the row shows
   a completion indicator that reflects whether the task is done; completed rows
   are additionally distinguished (e.g., de-emphasized) on the task name.

---

### User Story 3 - Polished details, footer, and consistent palette (Priority: P3)

A person using the TUI reads the detail pane and the help/status line as
deliberately styled, readable elements: the task name stands out as a header,
the metadata labels are aligned and visually secondary, the description wraps to
fit, and the help line reads as a footer. The whole interface draws from one
coherent set of colors that stays legible on both light and dark terminals.

**Why this priority**: These refinements raise overall polish and consistency
but are the least essential to core comprehension; they build on the framing and
indicator work.

**Independent Test**: View a task with a description, due date, estimate, and
completion timestamp; confirm the name reads as a header, labels are aligned and
secondary, the description wraps to the pane width, the help line reads as a
footer, and colors remain legible when the terminal uses a light background.

**Acceptance Scenarios**:

1. **Given** a selected task with metadata, **When** the detail pane renders,
   **Then** the task name is presented as a prominent header and the
   `ID / Due / Est / Completed` labels are de-emphasized and column-aligned.
2. **Given** a task with a long description, **When** the detail pane renders,
   **Then** the description wraps within the pane's inner width without
   overflowing the frame.
3. **Given** the TUI is open, **When** the user views the bottom of the screen,
   **Then** the help/shortcut line reads as a distinct full-width footer, and
   error messages remain clearly distinguished.
4. **Given** a terminal with a light background, **When** any styled element
   renders, **Then** text and accents remain legible (colors adapt rather than
   assuming a dark background).

---

### Edge Cases

- **Output is not a terminal** (piped/redirected): all styling is suppressed and
  output stays plain, exactly as today. Glyphs and framing must not appear in
  unstyled output.
- **Very small terminal**: at narrow widths or short heights the frames and
  content must clamp gracefully and never crash; inner dimensions collapse to a
  minimum rather than going negative.
- **Light vs. dark terminal background**: accent and secondary colors must
  remain legible in both; no element may become invisible against the
  background.
- **Mixed tree shapes**: deep nesting, fully expanded subtrees, and collapsed
  subtrees must each render the correct fold/completion indicators and tree
  connectors.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The task list and the detail view MUST each be presented as a
  distinct framed region with a title that identifies it.
- **FR-002**: The framed region the user is currently focused on MUST be
  visually distinguished from the inactive region, and this distinction MUST
  follow focus as it moves between panes.
- **FR-003**: The framed treatment MUST be applied consistently across all
  list-mode layouts (default list view, edit/new-task form view, and move-task
  view).
- **FR-004**: Each task row MUST show a fold indicator only when the row has
  children that can be expanded or collapsed; the indicator MUST convey whether
  the row is currently expanded or collapsed. Rows without collapsible children
  MUST show no fold indicator.
- **FR-005**: Each task row MUST show a completion indicator reflecting whether
  the task is done, on every row regardless of fold state.
- **FR-006**: Completed tasks MUST remain visually distinguished on the task
  name (in addition to the completion indicator).
- **FR-007**: The detail pane MUST present the task name as a prominent header,
  with metadata labels (`ID`, `Due`, `Est`, `Completed`) de-emphasized and
  column-aligned, and the description wrapped to the pane's inner width.
- **FR-008**: The help/shortcut line MUST be presented as a distinct full-width
  footer; error messaging MUST remain clearly distinguished from normal text.
- **FR-009**: All colors used by the interface MUST adapt so that text and
  accents remain legible on both light and dark terminal backgrounds.
- **FR-010**: The interface MUST use a single coherent accent treatment reused
  across focus emphasis, headers, and the selection cursor, rather than ad hoc
  one-off colors.
- **FR-011**: The selection cursor MUST be presented in a softened style (an
  accent emphasis plus a gentle row distinction) rather than a heavy full-width
  solid fill.
- **FR-012**: When output is not a terminal, all styling, framing, and decorative
  glyphs MUST be suppressed and output MUST remain plain.
- **FR-013**: All existing TUI behavior (navigation, selection, editing, moving,
  completion toggling, folding) MUST be unchanged; this feature changes
  presentation only.
- **FR-014**: At degenerate terminal sizes, the interface MUST render without
  crashing and MUST clamp inner content dimensions to a non-negative minimum.

### Key Entities

*Not applicable — this feature changes presentation only and introduces no new
data entities.*

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can identify which of the two panes is focused on first
  glance, without interacting, in 100% of focus states.
- **SC-002**: For any task row, a user can correctly state both whether the row
  can be unfolded and whether the task is complete, from the indicators alone,
  without expanding or selecting the row.
- **SC-003**: Every existing automated test that asserts plain/unstyled output
  continues to pass unchanged, confirming non-terminal output is unaffected.
- **SC-004**: The visible width of each rendered pane matches its allotted width
  (framed content does not overflow or leave ragged columns) across the tested
  terminal sizes.
- **SC-005**: Styled output remains legible on both a light-background and a
  dark-background terminal, with no element rendering invisibly against the
  background.
- **SC-006**: The TUI renders without panicking at terminal sizes from the
  smallest supported dimensions upward.

## Assumptions

- The redesign targets the interactive TUI only (`internal/tui`); other CLI
  output paths are out of scope.
- The existing terminal-capability gate (styling enabled only on an interactive
  terminal) is reused as-is; no new configuration toggles are introduced.
- No new color/theme customization is exposed to the user in this pass; the
  palette is fixed (but adaptive to light/dark backgrounds).
- The specific glyphs and color values described in the design plan are
  reasonable defaults; exact characters and shades may be adjusted during
  implementation as long as the stated outcomes hold.
- Behavior, keybindings, and data are unchanged; only rendering is affected.
