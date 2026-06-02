# Research: Pomodoro Progress Display

## Question 1 — How does the TUI obtain the completed-pomodoro count for the selected task (and for a linked task on the planning tab)?

**Context**: The TUI tree is built from `ListTasks`, which returns bare `Task` rows. `Task` carries `estimate` (0..10) but **not** a completed-pomodoro count. The count exists server-side via `CountCompletedPomodorosForTask` and is exposed only through `GetTaskResponse.completed_pomodoro_count` (a single-task RPC). The planning tab's `renderPlanDetail` receives only a `PlanEntry` (`task_id`, no estimate, no count).

**Decision**: Add an additive `completed_pomodoro_count` field to the `Task` message and populate it in the `ListTasks` handler via a new per-user, grouped aggregate query (`CountCompletedPomodorosByTask`). The TUI tree then carries both estimate and completed count for every task; both details panes render the glyph row synchronously, and the planning tab looks up the linked task in the in-memory tree by `task_id`.

**Rationale**:
- **Single source of truth**: estimate and completed count travel together on the same `Task`, already threaded through `cli.TreeNode` everywhere the TUI renders.
- **Synchronous rendering**: no per-selection RPC, no loading flicker, no stale-then-refresh in the details pane.
- **No new abstraction** (Principle I): one field, one query, one handler edit. Refresh after a completed pomodoro is a one-line addition — reload the tree with the existing `listTasksCmd` after `pomCompletedMsg`.
- **Additive & backward-compatible** (Principle II): a new protobuf field does not break the existing web client, which simply ignores it.

**Alternatives considered**:
- **Lazy `GetTask` per selection + client-side cache**: avoids any backend change, but introduces async fetch on cursor movement, a cache keyed by task id, cache invalidation on pomodoro completion, and stale-render flicker. More moving parts and a cache-invalidation concern — rejected as more complex than the data-plumbing change (violates the spirit of Principle I).
- **New bulk RPC returning counts only**: a parallel `task_id → count` map fetched alongside `ListTasks`. Avoids touching the `Task` message but adds a second call to keep in sync with the tree and a join at the render layer. More coordination for no real benefit over carrying the count on `Task` — rejected.
- **Reusing `GetTaskResponse` shape for the list**: over-fetching (it also returns the full `pomodoros` slice) and a larger contract change — rejected.

## Question 2 — Where exactly does the "active pomodoro glyph" live, and what does "red and bold" mean there?

**Context**: US3 asks that the tomato glyph for an *active* (running) pomodoro be red and bold. The running timer renders in the status bar (`view.go`) as `🍅 %02d:%02d · <task>  [x] cancel` and, while confirming quit, `🍅 %02d:%02d still running…`.

**Decision**: Style only the leading `🍅` glyph on the active-pomodoro status line(s) as red + bold, using the same shared red token introduced for completed pomodoros (`pomodoroDone`). The surrounding timer text keeps its current styling. This is independent of the glyph-row feature.

**Rationale**: Keeps the active-state cue consistent with the "completed = red, bold" language of the progress row, reinforcing that red+bold tomatoes mean "real pomodoro effort." Reuses a palette token (Principle III); no new color needed for the active glyph.

**Alternatives considered**: Styling the entire status line red (too loud, hurts readability); a distinct active-only color (adds a palette entry for no clear benefit) — both rejected.

## Question 3 — Should the glyph row replace or accompany the existing numeric "Est: N pomodoros" line?

**Context**: The task-tree details pane currently prints `Est: 5 pomodoros` when `estimate > 0`. The planning detail shows no pomodoro info today.

**Decision**: Replace the numeric `Est:` line in the task-tree details pane with the glyph row when there is anything to show (estimate > 0 or completed > 0); omit entirely otherwise. The planning-tab detail gains the same glyph row for linked tasks. The glyphs convey the estimate (row length) and progress (coloring) at a glance, making the numeric line redundant.

**Rationale**: Matches the spec's intent ("easy to interpret at a glance"), avoids duplicated/contradictory information, and keeps both surfaces consistent (Principle III).

**Alternatives considered**: Keeping both the numeric line and the glyph row (redundant, noisier) — rejected. A numeric `2/5` badge instead of glyphs (contradicts the explicit glyph-row request) — rejected.

## Plain (no-color) mode

Without styling, `renderPomodoroRow` still emits the correct number of `🍅` glyphs (count = max(estimate, completed)) with no ANSI styling — satisfying FR-010. The glyph count alone conveys estimate and (via the spec's rules) cannot distinguish completed vs. remaining by color, which is the accepted graceful degradation; no crash, no layout break.
