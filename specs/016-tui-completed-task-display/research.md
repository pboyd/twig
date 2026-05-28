# Phase 0 Research — TUI Completed Task Display

No `NEEDS CLARIFICATION` markers remained from the plan. The Technical Context resolved cleanly from the existing codebase. The few decisions worth recording are listed below.

## Decision 1: Reuse the existing `pendingComplete` field

- **Decision**: Use `Model.pendingComplete *int64` (already declared in `internal/tui/model.go` and threaded through `buildVisible` in `internal/tui/tree.go`) to represent the "lingering just-completed task." Set it in the `Complete` key handler and clear it on selection-changing actions.
- **Rationale**: The field exists, is plumbed through the visibility flattener, and is already cleared on `Up`/`Down` (`internal/tui/update.go:249, 255`). Wiring its setter is the smallest possible change that produces the desired behavior.
- **Alternatives considered**:
  - *Add a new field* (e.g. `lingeringCompleted`) — rejected: duplicates an existing concept and violates Principle I (Simplicity).
  - *Mark the task as "soft completed" in the tree model* — rejected: would require rebuilding the tree, more invasive, and conflates a view-only concept with the data model.

## Decision 2: Add a strikethrough-only helper alongside `DimStrike`

- **Decision**: Add `cli.Strike(s string) string` to `services/todo/internal/cli/render.go` that wraps `s` with the strikethrough ANSI code only (no dim), gated on `isTTY` the same way `DimStrike` is. Use this in the tree row renderer.
- **Rationale**: Spec clarification Q3 explicitly selected "strikethrough on the task name, no color change." `DimStrike` couples dim + strikethrough; using it would change color. A sibling helper preserves the existing pattern and TTY-gating logic.
- **Alternatives considered**:
  - *Use `DimStrike` as-is* — rejected: violates the explicit Q3 answer.
  - *Inline the ANSI codes in `view.go`* — rejected: bypasses the project's centralized TTY-gating in `internal/cli/render.go` and creates duplicate ANSI knowledge.

## Decision 3: Remove `DimStrike` from `renderDetails`'s name field

- **Decision**: In `internal/tui/details.go`, render `task.Name` without `cli.DimStrike` wrapping, even when `task.CompletedAt != nil`. The existing `Completed: <timestamp>` line continues to render and conveys the same information.
- **Rationale**: Spec clarification Q2 explicitly chose "remove completion styling from the detail-pane name."
- **Alternatives considered**:
  - *Leave detail pane unchanged* — rejected: directly contradicts Q2.
  - *Conditionally render based on a config flag* — rejected: speculative, no requirement asked for it.

## Decision 4: Lingering removal is tied to selection-changing actions, not arbitrary keystrokes

- **Decision**: Clear `m.pendingComplete = nil` only when the action actually changes the selected row index. Today this means: `Up`, `Down`, and any handler that explicitly sets `m.cursor` to a different visible row. Same-row actions (`Expand`, `Collapse` on a leaf, `Edit`, `Help`, `Filter`, digit-key estimate, pomodoro toggle) leave it alone.
- **Rationale**: Spec clarification Q1 said the lingering row should survive same-row actions and only be cleared by selection change or by a tree refresh.
- **Alternatives considered**:
  - *Clear on any keypress* — rejected: contradicts Q1 (option B was explicitly not chosen).
  - *Add a separate "linger timer"* — rejected: introduces async complexity that has no requirement.

## Decision 5: Refresh-clears-linger is already a side-effect of `refreshedMsg` handling

- **Decision**: No extra code needed for FR-006 ("tree refresh removes the lingering task"). A refresh fetches fresh tasks from the server; the just-completed task now has `CompletedAt != nil` in the new tree. After `buildVisible` runs against the new tree with `pendingComplete` still set to that id, the row stays visible — *which is the desired Story 1 behavior* (the row stays as the highlighted lingering row after the post-complete refresh). However, FR-006 says a *later* refresh (e.g. user-triggered `Refresh` or a sync-driven reload that is not the post-complete one) should remove it. The simplest correct rule: clear `m.pendingComplete` at the start of `listTasksCmd`-triggered refreshes (the Refresh keypath) but *not* on the post-complete `fetchAfterMutation` refresh (which carries the just-completed id as `highlightID`).
- **Rationale**: Preserves the Story 1 post-complete linger while honoring FR-006's "linger doesn't survive a reload" for explicit refreshes.
- **Alternatives considered**:
  - *Always clear on any `refreshedMsg`* — rejected: would defeat Story 1 because the post-complete redraw is itself a refresh.
  - *Track a separate "post-complete pending" timestamp* — rejected: needless complexity.

## Open questions

None. Proceed to Phase 1.
