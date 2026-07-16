# Phase 0 Research: Goal Task Filter Shortcut

**Feature**: 065-goal-task-filter | **Date**: 2026-07-16

No `NEEDS CLARIFICATION` markers were carried into Technical Context — the feature sits entirely inside
an existing, well-established TUI subsystem. Research therefore focused on verifying the four assumptions
the spec rests on, so the plan can commit to a code path rather than guess at one.

## R1: Does the server already support `^goal_id=<id>`?

**Decision**: Yes — reuse it untouched. No proto, handler, or SQL change is in scope.

**Evidence**:
- `services/twig/internal/filter/parser.go:186` parses `^goal_id` into a `RelCondition`.
- `services/twig/internal/filter/ast.go:73` defines `RelCondition` for relationship fields.
- `services/twig/internal/filter/eval.go:23` implements transitive relation matching for `^goal_id`.
- The TUI already dispatches expressions verbatim via `filterCmd` → `FilterTasks`
  (`internal/tui/update.go:662`).

**Rationale**: FR-002 (transitive goal scoping) is already implemented and tested server-side. Shipping
this feature means *composing* an expression the user could have typed, not extending the filter engine.

**Alternatives considered**: A dedicated `ListTasksByGoal` RPC — rejected. It would duplicate working
filter logic, violate Principle I, and break FR-003 (the applied filter must be an ordinary, editable
filter expression) by creating a second code path with its own behavior.

## R2: Does an expression naming only `^goal_id` preserve the show-all visibility default?

**Decision**: Yes — confirmed, so FR-009 needs no extra code.

**Evidence**: `services/twig/internal/filter/eval.go:56` carries the explicit comment: *"RelCondition on
parent_id/goal_id does not count as mentioning completed/snoozed."* The per-attribute override semantics
established in spec 062 mean `^goal_id=<id>` leaves both completion and snooze defaults in force, and
`showAll` continues to be passed independently through `FilterTasks`.

**Rationale**: This validates the spec's central assumption. Emitting the bare expression the user asked
for (`^goal_id=<id>`) yields exactly the desired behavior — open tasks by default, completed ones when
`c` is toggled — with no conditional expression-building.

**Alternatives considered**: Emitting `completed=false AND ^goal_id=<id>` — rejected. It would hard-code
the completion default into the expression and permanently break the `c` (show all) toggle for anyone
arriving via the shortcut.

## R3: How is the applied filter expression recorded, and what does that force?

**Decision**: The shortcut MUST populate `m.filterInput` before dispatching `filterCmd`.

**Evidence**: `handleFilterResult` (`internal/tui/update.go:199-200`) assigns:

```go
m.filterExpr = m.filterInput.Value()
```

The persisted expression is read back from the **input widget**, not from the expression passed to
`filterCmd`. A shortcut that dispatched `filterCmd(..., "^goal_id=7", ...)` without also calling
`m.filterInput.SetValue("^goal_id=7")` would filter the list correctly but then overwrite `filterExpr`
with the input's stale or empty contents.

**Consequences if missed**: FR-003 breaks (reopening `/` shows the wrong text), FR-004 breaks (the old
filter's text lingers), and the refresh/mutation re-fire paths at `update.go:804`, `update.go:2360`, and
`update.go:2546` — all of which re-dispatch `m.filterExpr` — would re-fire the wrong expression or drop
the filter entirely on the next task edit.

**Rationale**: This is the single highest-risk detail in the feature and the main reason the plan pins an
explicit ordering: `SetValue` → `filterGen++` → `filterCmd`.

**Alternatives considered**: Refactoring `handleFilterResult` to carry the expression on
`filterResultMsg` — rejected for this feature. It is a cleaner design, but it rewrites a working path
shared by every existing filter entry point for no user-visible gain. Setting the input is what the
existing `/` path already does; the shortcut should look identical to it.

## R4: Is `ctrl+t` free on the Goals tab, and how is sub-mode gating handled?

**Decision**: `ctrl+t` is free on Goals; add a `GoalGoToTasks` binding and a case in the main
`handleGoalsKey` switch.

**Evidence**:
- `ctrl+t` is bound only as `PlanGoToTask` (`internal/tui/keymap.go`), consumed exclusively inside
  `handlePlanningKey` (`internal/tui/update.go:1876`). No Goals-tab binding uses it.
- `handleGoalsKey` (`internal/tui/update.go:1282-1316`) guards every sub-mode — `goalEdit`/`goalNew`/
  `goalNewTask`, `goalStatusHistory`/`goalStatusReader`/`goalStatusConfirmDel`, `goalPickLink`/
  `goalPickUnlink`, `goalConfirmDelete`, and `modeHelp` — with early returns *before* reaching the main
  action switch.

**Rationale**: FR-007 (shortcut inert while a form/picker/history/help view is open) is satisfied
structurally by placing the new case in the main switch — the existing early returns do the gating. No
new mode checks are needed, which keeps the change aligned with Principle I.

**Alternatives considered**: A separate top-level `ctrl+t` intercept in `Update` — rejected. It would
bypass the sub-mode guards and require re-implementing them, directly violating FR-007 and FR-008.

## R5: What is the established pattern for a cross-tab jump?

**Decision**: Mirror `PlanGoToTask` (`internal/tui/update.go:1876-1891`).

**Evidence**: That handler is the existing precedent for this exact interaction and performs, in order:
guard on empty selection → resolve the target ID → set `m.activeTab` → update the `m.keys.*Mode` help
flags → clear the departing tab's transient state (`err`, `pendingComplete`) → position the cursor.

**Rationale**: Principle III (UI/UX consistency) is best served by the new shortcut being recognizably
the same interaction as the one it shares a key with. The one deliberate divergence: `PlanGoToTask`
positions the cursor on a known task, whereas the goal jump has no single target task — the cursor lands
via the normal `clampCursor` path in `handleFilterResult` once results return.

**Alternatives considered**: Inventing a new jump idiom — rejected under Principle III.

## Summary of decisions

| # | Question | Decision |
|---|----------|----------|
| R1 | Server support for `^goal_id` | Exists; reuse verbatim, zero backend change |
| R2 | Show-all interaction | Bare `^goal_id=<id>` preserves defaults; no conditional building |
| R3 | Recording the expression | Must `SetValue` the input before dispatching `filterCmd` |
| R4 | Key availability + gating | `ctrl+t` free on Goals; main-switch case inherits sub-mode guards |
| R5 | Jump pattern | Mirror `PlanGoToTask` ordering |
