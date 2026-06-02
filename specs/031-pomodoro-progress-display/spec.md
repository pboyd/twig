# Feature Specification: Pomodoro Progress Display

**Feature Branch**: `031-pomodoro-progress-display`

**Created**: 2026-06-02

**Status**: Draft

**Input**: User description: "Pomodoro Enhancements — First, a simple UI tweak: the tomato glyph for an active pomodoro should be red and bold. The more important feature is displaying the estimated and completed pomodoros in a way that's easy to interpret at a glance: a row of tomato glyphs shown in the details pane for a task on the task tree and for a linked task on the planning tab. Estimated pomodoros appear dimmed; completed pomodoros up to the estimate replace dimmed glyphs with bold red ones; completed pomodoros beyond the estimate appear yellow and bold."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See pomodoro progress at a glance on a task (Priority: P1)

A user viewing the details pane for a task on the task tree wants to instantly understand how much focused work a task is expected to take and how much has already been done, without reading numbers.

**Why this priority**: This is the core of the feature — a visual, scannable representation of pomodoro estimate vs. completion is the primary value the user asked for. It stands alone and delivers immediate benefit.

**Independent Test**: Open the details pane for a task that has a pomodoro estimate and some completed pomodoros, and confirm a row of tomato glyphs renders with the correct count and coloring.

**Acceptance Scenarios**:

1. **Given** a task with an estimate of 5 pomodoros and 0 completed, **When** the user views the task details pane, **Then** five tomato glyphs are shown, all dimmed.
2. **Given** a task with an estimate of 5 pomodoros and 2 completed, **When** the user views the task details pane, **Then** five tomato glyphs are shown, the first two bold red and the remaining three dimmed.
3. **Given** a task with an estimate of 5 pomodoros and 5 completed, **When** the user views the task details pane, **Then** five tomato glyphs are shown, all bold red.
4. **Given** a task with an estimate of 5 pomodoros and 6 completed, **When** the user views the task details pane, **Then** six tomato glyphs are shown, the first five bold red and the sixth yellow and bold.

---

### User Story 2 - See pomodoro progress for a linked task on the planning tab (Priority: P2)

A user planning their day on the planning tab wants the same at-a-glance pomodoro progress for a task linked to a plan entry, so they can gauge remaining effort while planning.

**Why this priority**: Extends the same visual to the planning context. Valuable but secondary to having the representation working in the task tree first; reuses the same rendering logic.

**Independent Test**: On the planning tab, view a plan entry linked to a task with a pomodoro estimate and completed count, and confirm the same row-of-tomatoes representation appears.

**Acceptance Scenarios**:

1. **Given** a plan entry linked to a task with an estimate of 3 and 1 completed, **When** the user views that entry's details on the planning tab, **Then** three tomato glyphs are shown, the first bold red and the other two dimmed.
2. **Given** a plan entry linked to a task with completions exceeding its estimate, **When** the user views that entry's details, **Then** the over-estimate glyphs are shown yellow and bold, consistent with the task tree.

---

### User Story 3 - Distinguish an active pomodoro glyph (Priority: P3)

A user with a pomodoro currently running wants the active-pomodoro tomato glyph to clearly stand out as red and bold, so the running state is unmistakable.

**Why this priority**: A small, self-contained styling tweak. Independent of the progress-row feature and lower impact, but improves clarity of the active state.

**Independent Test**: Start a pomodoro and confirm the tomato glyph representing the active pomodoro renders red and bold.

**Acceptance Scenarios**:

1. **Given** an active (running) pomodoro, **When** the user views the interface where the active pomodoro glyph appears, **Then** that glyph is rendered red and bold.

---

### Edge Cases

- **No estimate and no completions**: The progress row is omitted entirely (nothing to show).
- **No estimate but one or more completions**: Every completed pomodoro counts as beyond the (zero) estimate, so all glyphs render yellow and bold, one per completed pomodoro.
- **Estimate but zero completions**: A full row of dimmed glyphs equal to the estimate.
- **Completions far exceeding estimate**: The row grows to the completed count; estimate-count glyphs are bold red and the remainder are yellow and bold.
- **Non-color terminal / plain output**: Without color/styling support, the row still shows the correct number of glyphs; emphasis distinctions degrade gracefully (no crash, no broken layout).
- **Very large counts**: A long row of glyphs renders without breaking the surrounding layout. (Wrapping/truncation behavior noted as an assumption.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST render a task's pomodoro information as a single row of tomato glyphs in the task details pane on the task tree.
- **FR-002**: The system MUST render the same row of tomato glyphs for a linked task shown on the planning tab.
- **FR-003**: The number of glyphs in the row MUST equal the greater of the estimated pomodoro count and the completed pomodoro count.
- **FR-004**: Glyphs representing completed pomodoros up to and including the estimate MUST be styled bold red.
- **FR-005**: Glyphs representing estimated-but-not-yet-completed pomodoros MUST be styled dimmed.
- **FR-006**: Glyphs representing completed pomodoros beyond the estimate MUST be styled yellow and bold.
- **FR-007**: When a task has neither an estimate nor any completed pomodoros, the system MUST omit the progress row.
- **FR-008**: The glyph used for an active (currently running) pomodoro MUST be styled red and bold.
- **FR-009**: The progress row MUST update to reflect the current estimate and completed counts whenever the details pane is shown for the task.
- **FR-010**: When styling is unavailable (e.g., a non-color output context), the system MUST still render the correct number of glyphs without error.

### Key Entities *(include if feature involves data)*

- **Task pomodoro state**: The estimated number of pomodoros for a task and the number of completed pomodoros for that task. These drive the count and coloring of the glyph row.
- **Active pomodoro**: The single in-progress pomodoro session, whose glyph receives the red-bold active styling.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can determine a task's pomodoro estimate and completion count from the glyph row alone, without reading any numeric value.
- **SC-002**: For any combination of estimate and completed counts, the rendered row matches the rules: count = max(estimate, completed); first min(completed, estimate) glyphs bold red; estimated remainder dimmed; completions beyond estimate yellow and bold.
- **SC-003**: The same representation appears consistently in both the task tree details pane and the planning tab for the same task.
- **SC-004**: An active pomodoro's glyph is visually distinguishable as red and bold whenever a pomodoro is running.
- **SC-005**: The example case (estimate 5, completed 2 → five glyphs with first two red; then completed 6 → six glyphs, five red and one yellow) renders exactly as described.

## Assumptions

- "Dimmed", "bold red", and "yellow and bold" refer to whatever emphasis/color mechanism the interface already uses; in contexts without color support the glyph count still conveys the information.
- The estimated and completed pomodoro counts for a task are already available to the interface; this feature concerns presentation only and does not introduce new ways to set or record those counts.
- The tomato glyph is the existing tomato emoji/character already used for pomodoros in the product.
- When the completed count exceeds the estimate, the row length follows the completed count (no upper cap is imposed by this feature). Layout wrapping or truncation of very long rows follows existing details-pane behavior and is out of scope here.
- "Linked task on the planning tab" refers to a plan entry that is associated with a task carrying pomodoro estimate/completion data; untimed or task-less plan entries show no progress row.
