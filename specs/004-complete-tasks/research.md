# Research: Complete Tasks

This document records the implementation-level decisions that needed an explicit
choice and were not already settled by the spec, the constitution, or the
existing codebase.

## 1. Representing completion state in the database

**Decision**: Add a single nullable column `tasks.completed_at TIMESTAMPTZ` —
`NULL` means incomplete, a non-null value means complete at that moment.

**Rationale**: The spec requires exactly two pieces of information per task:
whether it is complete, and when it was completed (when complete). A nullable
timestamp encodes both in one column with no redundancy; "is complete" is just
`completed_at IS NOT NULL`. The column is indexable if we ever need to filter
on it server-side, and the migration is a one-line `ALTER TABLE`.

**Alternatives considered**:

- Two columns (`is_complete BOOLEAN`, `completed_at TIMESTAMPTZ`). Adds a
  consistency invariant the database does not enforce (a `true` flag with no
  timestamp, or vice versa) — pure redundancy.
- A `task_status` enum column with values `INCOMPLETE`, `COMPLETE`, and a
  separate `completed_at` column. Speculative — the spec excludes reopening and
  has no third state. Violates Principle I.
- A separate `task_completions` table. Solves a problem we do not have (no
  history of completion events is required); requires a join on every read.

## 2. Checking that all descendants are complete

**Decision**: A single recursive CTE — `HasIncompleteDescendants(parent_id,
user_id)` — that walks the `tasks` rows transitively via `parent_id` and
returns `true` if any descendant has `completed_at IS NULL`.

**Rationale**: The check is bounded by the depth of one user's task tree
(small, per the project's stated scale), runs entirely server-side in one
round-trip, and reuses the existing `tasks_parent_id_idx` index. The handler
calls it once before allowing `CompleteTask` to mutate state. The same query
shape (lookup by `user_id`) preserves the per-user isolation established in
feature 003.

**Alternatives considered**:

- Fetch all of a user's tasks into Go and walk in memory. Works at this scale
  but pulls every row even when the caller only needs a yes/no answer for one
  subtree. Strictly more data movement, no clearer code.
- Add a denormalized `incomplete_descendant_count` column maintained by
  triggers. Speculative complexity — three triggers (insert, update, delete)
  and a column to keep correct under concurrent edits, all to optimize a check
  that is already cheap.

## 3. Where list filtering lives

**Decision**: The `ListTasks` RPC keeps its existing wire contract — it returns
every task for the user, each carrying its `completed_at` value. The CLI
applies the default / `--completed` / `--all` filter while rendering.

**Rationale**: The project's stated scale (one to two users per instance, small
task lists) makes the server filter premature optimization. Keeping the API
surface unchanged means feature 004 ships without a new request field and
without altering any other client that may call `ListTasks`. The default-mode
tree-pruning rule from spec Q1 — "hide a branch only when the task and *all*
its descendants are complete" — also requires the renderer to look at the full
tree to make the decision, so the data is needed at the client anyway.

**Alternatives considered**:

- Add a `filter` enum to `ListTasksRequest` (`INCOMPLETE_ONLY`,
  `COMPLETED_ONLY`, `ALL`). Server returns a filtered set; client renders. The
  default-tree-pruning rule (Q1 of the clarify session) would still need the
  full tree on the client, so this adds API surface without simplifying the
  client. Rejected on YAGNI grounds.
- Two separate RPCs (`ListIncompleteTasks`, `ListCompletedTasks`). More RPCs,
  same data; rejected.

## 4. The default-mode tree-pruning rule

**Decision**: After building the full tree, the renderer recursively marks each
node "keep" if the node itself is incomplete OR any of its descendants is
incomplete; nodes that fail both tests are dropped along with their entire
(fully-complete) subtree. In `--completed` mode the rule inverts: keep a node
if it is complete; collapse incomplete intermediaries by promoting their kept
descendants up to the nearest kept ancestor's level. In `--all` mode no
filtering is applied. Within each parent, kept children remain in id order
(spec Q4).

**Rationale**: Directly implements spec Q1's clarification and preserves the
existing id-order invariant from Q4. Doing the prune in one pass after the
tree is built keeps the existing `buildTree` / `renderTree` helpers in
`render.go` untouched except for an extra filter step.

**Alternatives considered**:

- Build a separate tree per mode. Three nearly-identical builders; rejected as
  duplication.
- Drop the tree and render a flat list in `--completed` mode. Loses contextual
  parentage that the user reasonably expects to see. Rejected.

## 5. Idempotent re-completion

**Decision**: `CompleteTask` on an already-complete task is a successful no-op
that returns the existing row (with its original `completed_at`). The CLI
exits with status `0` and prints a message naming the task and its original
completion time.

**Rationale**: Matches FR-010 verbatim, matches the assumption recorded in the
spec for exit-status semantics, and matches the existing CLI convention of
emitting "verb task N" confirmation lines for successful mutations. SQL: the
update statement uses `SET completed_at = COALESCE(completed_at, NOW())` so
the column never moves once set.

**Alternatives considered**:

- Return an error (`CodeAlreadyExists` / `CodeFailedPrecondition`) on
  re-complete. Rejected — the spec explicitly calls re-complete a no-op, and
  treating it as an error makes scripts harder to write.
- Update the timestamp to the new "now". Rejected — would lose the original
  completion moment, which is the only piece of historical data this feature
  records.

## 6. Reopen / un-complete

**Decision**: Out of scope, no implementation. Confirmed by the spec.

**Rationale**: The spec excludes reopening. Combined with the FR-013 / FR-014
rules — a complete task cannot gain new (incomplete) descendants — completion
is therefore terminal for the duration of this feature. The schema does not
preclude a future "reopen" feature (setting `completed_at` back to `NULL` is
trivial), but no code or contract for it ships in 004.

## 7. Conflicting CLI filter flags

**Decision**: `--completed` and `--all` are mutually exclusive. If both are
supplied, the CLI prints an error to stderr and exits `1` without contacting
the server (FR-012). The `list` subcommand parses the flags with `flag.NewFlagSet`
and validates the pair before any RPC call.

**Rationale**: Cheapest possible enforcement; matches existing CLI validation
patterns (see `runAdd` / `runMod` flag handling).

**Alternatives considered**:

- Last-flag-wins. Silently honors one and discards the other; surprising and
  bug-prone. Rejected.
