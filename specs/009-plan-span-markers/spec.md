# Feature Specification: Plan Span Markers

**Feature Branch**: `009-plan-span-markers`

**Created**: 2026-05-22

**Status**: Draft

**Input**: User description: "Replace the per-slot repeated task name in the `todo plan` CLI output with span markers using box-drawing characters."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read the plan at a glance without name repetition (Priority: P1)

A user runs `todo plan` to view today's schedule. Each task occupies one or more contiguous 15-minute slots. Instead of seeing the task name printed on every single slot in the block (which is noisy when a task spans many slots), the user sees the task name only once — on the first slot of the block — followed by visual markers indicating where the task continues and ends.

**Why this priority**: This is the entire feature. Improving scan-ability of the daily plan is the user-facing payoff; without it there is nothing to ship.

**Independent Test**: Run `todo plan` against a day containing at least one multi-slot task, one single-slot task, and a gap of free slots. Confirm the task name appears exactly once per contiguous block, that the start/middle/end of each block is visually distinct, and that free slots remain blank.

**Acceptance Scenarios**:

1. **Given** a task assigned to a contiguous block of 4+ slots, **When** the user runs `todo plan`, **Then** the first slot shows the task number and name with a "start" marker, the intermediate slots show only a "middle" marker (no name repeated), and the last slot shows an "end" marker (no name repeated).
2. **Given** a task occupying exactly one slot, **When** the user runs `todo plan`, **Then** that slot shows the task number, name, and a "single-slot" marker — not the start/end markers used for multi-slot blocks.
3. **Given** two different tasks assigned to adjacent slots with no gap between them, **When** the user runs `todo plan`, **Then** each task is rendered as its own block (end marker for the first, start marker for the second on the next row); the names are not merged or shared.
4. **Given** free (unassigned) slots in the plan, **When** the user runs `todo plan`, **Then** those slots are rendered blank, exactly as they are today.

---

### Edge Cases

- A task that spans exactly two slots: the first slot is the start marker (with name), the second is the end marker (no name). There is no middle marker.
- The same task ID appears in two non-contiguous blocks (split by a gap or another task): each block is rendered independently with its own start/end markers and the name printed at the start of each block.
- A task name long enough that, combined with the start marker and number prefix, exceeds the available row width: the name uses the same truncation/wrapping behavior the current `todo plan` output uses — this feature does not change name-length handling.
- A terminal that does not render the chosen box-drawing characters (e.g., a non-UTF-8 locale or a font without the glyphs): see Assumptions.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The plan view MUST group consecutive slots assigned to the same task into a single visual "block."
- **FR-002**: For a block of two or more slots, the plan view MUST render the first slot with a "start" marker, intermediate slots (if any) with a "middle" marker, and the last slot with an "end" marker.
- **FR-003**: For a block consisting of a single slot, the plan view MUST render that slot with a distinct "single-slot" marker — different from the start, middle, and end markers.
- **FR-004**: The task's display name and number prefix MUST appear exactly once per block, on the start (or single-slot) row.
- **FR-005**: Intermediate and end rows within a multi-slot block MUST NOT display the task name or number.
- **FR-006**: Free (unassigned) slots MUST continue to render as blank rows, with no marker glyph.
- **FR-007**: The leading time column and the column separator (`│`) that frames the plan view today MUST be preserved unchanged.
- **FR-008**: Two adjacent blocks belonging to different tasks MUST each have their own start and end markers; markers from different blocks MUST NOT be merged or shared.

### Key Entities

- **Plan slot**: A 15-minute row in the daily plan. May be free or assigned to exactly one task. Already exists in the system; this feature only changes how a sequence of slots is rendered.
- **Block**: A maximal run of consecutive slots assigned to the same task. Derived at render time from the existing slot data; not stored.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For any block of N slots assigned to one task, the task's display name appears exactly 1 time in the rendered output (down from N times today).
- **SC-002**: A user looking at the plan can identify the start and end rows of every multi-slot block without reading any text — purely from the marker glyphs.
- **SC-003**: The plan view's overall row count and time-column alignment are unchanged from today; only the right-hand content of each row differs.

## Assumptions

- The user's terminal renders Unicode box-drawing characters (`┌`, `│`, `└`, `─`). The existing plan view already uses the box-drawing `│` as its column separator, so a terminal that runs the current `todo plan` will render these markers too. No ASCII-only fallback is in scope for this feature.
- The 15-minute slot granularity, the time column format, and the data model for plan slots are unchanged by this feature. The change is purely a rendering change inside the CLI's plan view.
- The numbering prefix (1, 2, 3…) is the task's existing position-in-plan number, rendered the same way as today — just only on the start row of each block.
- TTY/non-TTY behavior follows the existing pattern in `internal/cli` (ANSI styling is gated on TTY detection); the span markers themselves are plain Unicode characters and render in both modes.
