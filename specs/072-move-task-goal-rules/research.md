# Phase 0 Research: Goal Handling When Moving Tasks

All unknowns from Technical Context are resolved. Nothing remains marked NEEDS CLARIFICATION.

---

## Decision 1 — Detect a real parent change by comparing against the stored row

**Decision**: Load the task inside the transaction and compare its stored `parent_id` against the
requested one. Apply goal rules only when they differ.

**Rationale**: This is the trap in the whole feature. `UpdateTaskRequest.parent_id` is documented as
*"Unset clears the parent, making the task top-level"* (`api/proto/task/v1/task.proto:201`) and the
SQL is unconditional full-replace:

```sql
-- name: UpdateTask :one
UPDATE tasks
SET name = $2, description = $3, due = $4, parent_id = $5, snooze_until = $7
WHERE id = $1 AND user_id = $6
```

Every caller therefore sends `parent_id` on every edit — `buildUpdatePayload` in
`services/twig-web/src/lib/updatePayload.ts` passes `task.parentId` straight through on a plain
rename. If the goal rules keyed off `req.Msg.ParentId != nil` (as today's validation does), then
**renaming a top-level goal-linked task would be treated as a move and would not clear anything, but
renaming a nested task would clear goal links across its whole subtree** for no reason. Comparison
against the stored row is what makes FR-008 true.

**Alternatives considered**:
- *Key off `req.Msg.ParentId != nil`* — rejected: as above, conflates "field present" with "value
  changed", and the field is always present.
- *Add a dedicated `MoveTask` RPC* — rejected under Principle I. It would duplicate the cycle,
  existence, and completion validations that already live in `UpdateTask`, and force changes in all
  three clients. The comparison is three lines.
- *Add a `FieldMask`* — rejected: a large contract change to solve a problem one comparison solves.

---

## Decision 2 — Wrap the parent branch in the existing transaction pattern

**Decision**: Use `t.Pool.Begin(ctx)` / `defer tx.Rollback(ctx)` / `t.Queries.WithTx(tx)`, then
`tx.Commit(ctx)`, exactly as `ReorderTask` does at `services/twig/internal/handler/task.go:478-486`.

**Rationale**: FR-011 requires the move and its goal changes to be observable only together. Today
`UpdateTask` performs up to eight sequential unwrapped calls (`TaskExists`, `parentChainContains`,
`GetParentCompletion`, `GetTask` ×2, `UpdateTask`, `GetMaxSiblingPosition`, `UpdateTaskPosition`); a
failure partway leaves the task moved with a stale goal link or a stale position. The `Task` handler
already holds `Pool *pgxpool.Pool` (`task.go:26`) for precisely this, so no wiring is needed —
`ReorderTask` proves the pattern works here.

**Alternatives considered**:
- *Leave it unwrapped* — rejected: directly contradicts FR-011, and the window widens with the new
  subtree update.
- *A single mega-statement doing move plus goal fixes* — rejected under Principle I as unreadable
  and untestable relative to a transaction around clear steps.

**Note**: `ReorderTask` guards `if t.Pool == nil` and returns an internal error. `UpdateTask` must
adopt the same guard, since existing handler tests construct `Task` without a `Pool`.

---

## Decision 3 — Two new recursive-CTE queries, keep the two existing ones

**Decision**: Add `NearestAncestorGoal` (returns the first non-NULL `goal_id` walking upward) and
`ClearSubtreeGoals` (recursive `UPDATE ... SET goal_id = NULL` over descendants). Leave
`AncestorHasGoal` and `DescendantHasGoal` in place.

**Rationale**: `AncestorHasGoal` already walks exactly the right chain but collapses to
`EXISTS (...)`, discarding the id that FR-004 needs. Rather than change its shape and disturb
`SetTaskGoal`, which still needs the boolean for FR-010, a sibling query returns the id. Both new
queries follow the recursive-CTE idiom already used four times in
`services/twig/db/queries/task.sql`, so nothing novel is introduced.

`NearestAncestorGoal` needs a depth column and `ORDER BY depth LIMIT 1` — "nearest" matters when
legacy data has goals at two levels of one chain, which is precisely the inconsistent state User
Story 3 exists to clean up.

**Alternatives considered**:
- *Reuse `internal/goal.EffectiveGoalID`* — rejected: it lives in the **root CLI module**
  (`internal/goal/goal.go:111`) and operates on `[]*taskv1.Task` already fetched by a client. The
  server module cannot import it, and pulling the whole task list into the handler to compute one id
  would be strictly worse than a query.
- *Generalize `AncestorHasGoal` to return a nullable id and derive the boolean at call sites* —
  rejected as a wider blast radius for no gain; the boolean call sites are correct today.

---

## Decision 4 — Clear the whole subtree, not just direct children

**Decision**: `ClearSubtreeGoals` recurses to arbitrary depth over the moved task's descendants.

**Rationale**: FR-003 says "at any depth", and the invariant is about the whole tree, not one level.
Today's precondition already checks descendants recursively via `DescendantHasGoal`, so the code
being replaced was aware of arbitrary depth; the replacement must be too. This is also the mechanism
by which User Story 3 repairs legacy rows without a migration — the spec explicitly chose
opportunistic cleanup over a backfill.

**Alternatives considered**:
- *Clear only the moved task's own link* — rejected: leaves the invariant violated whenever the
  moved task carries a subtree that has legacy goal links, which is the case Story 3 names.
- *A one-time migration clearing all non-root `goal_id`s* — rejected: explicitly ruled out of scope
  in spec Assumptions.

---

## Decision 5 — Promotion may link to a closed goal; direct linking still may not

**Decision**: Promotion writes `goal_id` as part of the update, bypassing `SetTaskGoal`'s
completed/archived check. `SetTaskGoal` itself is unchanged.

**Rationale**: These two operations answer different questions. `SetTaskGoal` answers *"may the user
newly attach this goal?"* — and attaching work to a finished goal is meaningless, so
`task.go:604-608` refuses it. Promotion answers *"what was this task already showing?"* The
association already exists and is already visible; the move is a reorganization, not a new
commitment. Refusing it would mean promoting a sub-task under a just-completed goal silently drops
the goal — reintroducing the exact data loss User Story 2 exists to fix, in the case where the
history matters most.

FR-007 states this outcome explicitly, and the spec's Edge Cases section calls it out, so it is a
specified behavior rather than an oversight.

**Alternatives considered**:
- *Skip the goal when it is closed* — rejected: silent loss, contradicts FR-007.
- *Refuse the move when the inherited goal is closed* — rejected: contradicts FR-001, which forbids
  refusing a move for any goal-related reason.
- *Relax `SetTaskGoal` to permit closed goals* — rejected: FR-010 explicitly preserves current
  direct-linking rules, and the restriction is correct for its own use case.

---

## Decision 6 — Fix repositioning on promotion at the same time

**Decision**: Reposition the task at the end of its destination sibling group whenever the parent
actually changed, including a change to root.

**Rationale**: The repositioning block at `task.go:325-341` is guarded by
`if req.Msg.ParentId != nil`, so a promotion to root — expressed as an *absent* `parent_id` — moves
the task into the root group while keeping its old position value from the group it left. That is a
pre-existing bug on the exact code path being rewritten, and the correct guard is the same
parent-changed comparison from Decision 1. Leaving it would mean promoted tasks land at an arbitrary
spot in the root list, which users would read as part of this feature's behavior.

**Alternatives considered**:
- *Leave the guard alone and fix positions separately* — rejected: the guard is being replaced
  regardless, and shipping a known-wrong ordering on the feature's headline path invites a bug report
  against this change.

---

## Decision 7 — No proto change, no client change

**Decision**: Ship server-only. Amend only the comment on `Task.goal_id` in
`api/proto/task/v1/task.proto` to note that a move may change it.

**Rationale**: `goal_id` is already `optional int64` and already documented read-only
(`task.proto:135-138`), so the wire shape is correct as-is. All three clients — TUI `moveTaskCmd`
(`internal/tui/update.go:539`), CLI, and the SPA — refetch the task list after a mutation, so the
corrected `goal_id` propagates with no client edit. `make proto` still needs running for the comment
change, but `api/gen/` output is otherwise identical.

**Alternatives considered**:
- *Return the affected descendant ids in `UpdateTaskResponse`* — rejected under Principle I. Every
  client already refetches; nothing would consume the field.
