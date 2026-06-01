# Phase 0 Research: Plan Task Actions

## Context

The planning tab (`internal/tui`, `planList` mode in `handlePlanKey`) can already navigate, add, edit, move, and remove plan entries, and it already mirrors a linked task's completed state on each entry (`PlanEntry.Completed`, computed server-side by `ListPlan`). It cannot yet act on the task behind an entry. The Tasks tab already implements both target actions; this feature extends them to the planning tab. No new server capability is required.

## Decision 1 — Reuse Tasks-tab keys and operations verbatim

**Decision**: Bind `space` (`keys.Complete`) to toggle-complete and `s` (`keys.PomStart`) to start-pomodoro inside `planList` mode, calling the same `task.v1` operations the Tasks tab calls.

**Rationale**: Principle III (UI/UX Consistency) and FR-005/FR-006 require that a user who learns one surface can predict the other. Using the same `key.Binding` values and the same RPCs guarantees identical results, not just similar ones. Both keys are currently **unbound** in `planList` mode (verified in `handlePlanKey`), so there is no collision with existing planning controls (`t`, `e`, `enter`, `ctrl+d`, `[`/`]`/`.`, `ctrl+r`, `x`).

**Alternatives considered**:
- *New, planning-specific keys* — rejected: introduces a second vocabulary for the same operation, violating Principle III and adding learning cost for zero benefit.

## Decision 2 — Pomodoro reuses `startPomCmd` with no plan reload

**Decision**: `case keys.PomStart` calls `startPomCmd(m.client, entry.TaskId, entry.Name)` directly.

**Rationale**: `startPomCmd` already returns `pomStartedMsg`, which the root `Update` handles tab-agnostically (sets `m.pom`, kicks `pomTickCmd`, runs the `on_start` hook). The running-timer indicator already renders regardless of active tab, so switching to the Tasks tab shows the same running pomodoro (FR-005, acceptance US2-3). No day reload is needed because starting a pomodoro does not change any plan entry's displayed fields. Single-active-pomodoro semantics, lifecycle hooks, and the completion banner are all inherited unchanged.

**Alternatives considered**:
- *A planning-specific pomodoro command* — rejected as redundant; nothing about the plan context changes the pomodoro behavior.

## Decision 3 — Complete funnels into the existing `planMutatedMsg` reload path

**Decision**: Add one helper, `completePlanTaskCmd(taskClient, planClient, day, taskID, complete, entryID, notice)`. It calls `CompleteTask` or `UncompleteTask` (per `complete`), then returns `planMutatedMsg{notice, highlightID: entryID}` on success or `planMutatedMsg{err}` on failure.

**Rationale**: `planMutatedMsg` already does exactly the three things needed (added in feature 028): it triggers `listPlanHighlightCmd(day, highlightID)` to re-fetch the day, it shows a `notice`, and it can route an error (`tabAgnosticErr` defaults false → `m.plan.err`, which is correct since the user is on the planning tab). Re-fetching the whole day is precisely why completing one entry updates **all** entries linked to the same task (FR-004): every entry's `Completed` is recomputed by `ListPlan`. Reusing this path means no new message type and no bespoke state juggling.

**Why not reuse `completeTaskCmd`/`uncompleteTaskCmd` directly?** Those return `refreshedMsg`, which rebuilds the **task tree** but never reloads plan entries — so a completion made on the planning tab would not redraw the entry. The thin wrapper redirects the result into the plan reload path instead.

**Alternatives considered**:
- *Optimistically flip `entry.Completed` in the model, then reload* — rejected: the existing `planMutatedMsg` reload is fast and authoritative; an optimistic flip adds a rollback-on-error branch for no perceptible latency win (Principle I).

## Decision 4 — Toggle direction from `entry.Completed`

**Decision**: Choose complete vs. uncomplete from the selected entry's current `Completed` flag (incomplete → complete; completed → uncomplete).

**Rationale**: The entry already carries the authoritative mirrored state, so no extra fetch is needed to decide direction. Mirrors the Tasks-tab branch on `node.Task.GetCompletedAt()`. Satisfies FR-002 (toggle, optimized for completing).

## Decision 5 — Guards for events and empty selection

**Decision**: Before either action, guard with `len(m.plan.entries) > 0` (no-op when empty, FR-008) and `entry.TaskId == 0` (event → playful `notice`, no state change, FR-007).

**Rationale**: `TaskId == 0` is the established "this is an event, not a task" sentinel in `PlanEntry`. Guarding mirrors the existing edit/remove guards in `handlePlanKey`. Events have nothing to complete or focus on, so the actions must be inert with feedback explaining why.

## Decision 6 — Notice persistence

**Decision**: No auto-dismiss; the `notice` persists until the next action replaces or clears it.

**Rationale**: Consistency with the feature-028 notice behavior already shipped; introducing a timer here would diverge for no reason (Principle I).

## Proposed user-facing copy (Principle IV tone review)

Authored warm/playful, accurate, actionable. Final wording reviewed at implementation:

- **Complete an event attempt**: `That's an event, not a task — nothing to check off here.`
- **Pomodoro on an event attempt**: `Events don't run on tomatoes — pick a task to focus on.`
- **Completion confirmation (optional)**: `Nice — '<task>' is done and dusted.`

These are inert-action and success notices; existing error surfaces (network/permission) keep their current playful error copy.
