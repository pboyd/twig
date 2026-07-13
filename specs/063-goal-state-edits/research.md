# Phase 0 Research: Goal State Edits

All items below were resolved from the existing codebase; no external research was required. There are no remaining `NEEDS CLARIFICATION` markers (the one open behavioral question was answered in the spec's Clarifications).

## Decision 1 — State changes from the edit form reuse `SetGoalState`

**Decision**: The new **State** field in the goal edit form applies changes via the existing `GoalService.SetGoalState` RPC. `UpdateGoal` is left unchanged (name/description/due, full-replace).

**Rationale**:
- `SetGoalState` already exists, is purpose-built ("any transition allowed", idempotent no-op on same state, re-ranks to the bottom of the destination group), and is already wired in the TUI as `setGoalStateCmd`.
- Adding `state` to `UpdateGoalRequest` would change the full-replace contract and duplicate re-ranking logic — a Principle I (Simplicity) and Principle II (API-First) regression for no benefit.
- On save, the goal-edit handler issues `updateGoalCmd` for the text fields and, **only when the state changed**, additionally issues `setGoalStateCmd`. The two commands are batched; both return `goalMutationMsg` and trigger the existing goals refresh.

**Alternatives considered**:
- *Extend `UpdateGoal` with a `state` field* — rejected: contract churn, breaks full-replace semantics, redundant with `SetGoalState`.
- *Sequence (update → then set state)* — rejected as unnecessary; ordering doesn't matter for correctness, so `tea.Batch` is simpler.

## Decision 2 — Hold placement in ordering

**Decision**: `hold` sorts **after `incubating`, before `completed`** everywhere it appears:
- Server `ListGoals` `ORDER BY CASE`: `committed`=1, `incubating`=2, `hold`=3, `completed`=4, `archived`=5.
- TUI `goalGroupHeaders` order (when "show all" is on): Committed, Incubating, Hold, Completed, Archived.

**Rationale**: Hold represents a still-active intent that's paused — conceptually between the actively-pursued states and the finished ones. The TUI regroups client-side, but keeping the server order consistent avoids surprises for any other consumer (e.g. the web app).

**Alternatives considered**: Placing Hold last (after Archived) — rejected as it visually equates "paused" with "done/abandoned", contradicting the spec's intent.

## Decision 3 — Space-to-complete toggle wiring

**Decision**: Remove the `GoalSetComplete` (`d`), `GoalSetIncubate` (`i`), `GoalSetCommit` (`o`), and `GoalSetArchive` (`v`) bindings and their handler cases. In the goal key handler, match the shared `Complete` binding (already `space`, help "toggle complete") and toggle:
- current state `COMPLETED` → `COMMITTED`
- any other state → `COMPLETED`

**Rationale**:
- Reusing the same `Complete` binding object as the Tasks tab is the most literal reading of "to match tasks" and satisfies Principle III (consistency).
- The toggle target (`COMMITTED` on un-complete) is fixed per the spec Clarification, so no prior-state tracking is needed (Principle I).
- Rank keys (`{`/`}`), "show all" (`c`), edit (`e`/`enter`), delete (`ctrl+d`), and add/link task (`n`/`L`) are untouched.

**Alternatives considered**: Rebinding `GoalSetComplete` to `space` and keeping a goal-specific binding — rejected: reusing the shared `Complete` binding is simpler and guarantees the keys can't drift apart.

## Decision 4 — Seeding the current state into the goal edit form

**Decision**: Add a `state goalv1.GoalState` field (plus `origState` for dirty/change detection) to `editFormModel`, shown/cycled only when `isGoal` is true. The goal-edit path in `update.go` sets `m.edit.state` from the selected goal right after `NewEditForm(...)` / `NewRootForm(...)`; `buildSaveMsg` copies it into `editSavedMsg` along with a `stateChanged` flag.

**Rationale**: Goals are edited by bridging through a fake `taskv1.Task` (`goalToFakeTask`), which carries no state. The form therefore needs the state supplied out-of-band. A single extra field mirrors how the existing Goal selector (`goalIdx`/`origGoalIdx`) already lives on the same struct — consistent and minimal.

**Alternatives considered**: A separate goal-only form model — rejected as duplication; the fake-task bridge already exists and only the State field is new.

## Decision 5 — State field interaction & rendering

**Decision**: The State field is a cycling selector (like the existing Goal selector's `cycleGoal`): left/right (or the field's cycle keys) move through the five states `Incubating · Committed · Hold · Completed · Archived`, starting on the goal's current state. It participates in the form's focus cycle (`cycleFocus`) only when `isGoal`. Rendering follows the existing label/value row style in `View`.

**Rationale**: Principle III — one established in-form control pattern, reused, so goal editing feels like the rest of the form. No free-text parsing or validation surface (the value set is closed).

**Alternatives considered**: A text field where the user types the state name — rejected: error-prone, needs validation, inconsistent with the Goal selector.

## Compatibility note — web app (out of scope)

`services/twig-web/` regenerates its TypeScript enum from the proto, so `GOAL_STATE_HOLD` will appear there after `npm run gen`. Per the spec, no web interaction changes are in scope for this feature. Risk is low: the web goals view already switches on known states and would fall through to a default for an unrecognized one. Any deliberate web treatment of Hold (hiding, labeling) is a separate follow-up and is **not** implemented here. The backend `CHECK` constraint and enum mapping fully support Hold regardless of client.

## Backend surface confirmed

- `goals.state` is `TEXT NOT NULL DEFAULT 'incubating' CHECK (state IN ('incubating','committed','completed','archived'))` — a migration must widen the `CHECK` to include `'hold'`.
- `handler/goal.go` maps enum↔string in two small functions (`goalStateToString`, `goalStateFromString`); both need a `hold`/`GOAL_STATE_HOLD` case. `goalStateToString` still rejects `UNSPECIFIED`.
- `SetGoalState` SQL re-ranks to the bottom of the destination group and no-ops when `state` is unchanged — works for `hold` with no query change beyond the `ListGoals` CASE.
