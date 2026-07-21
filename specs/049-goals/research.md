# Research: Goals

**Feature**: 049-goals | **Date**: 2026-06-11

Decisions resolving every open technical choice in the plan. No NEEDS
CLARIFICATION markers remain.

## D1: New proto package `goal/v1` with its own `GoalService`

**Decision**: Define goals in a new proto package `api/proto/goal/v1/goal.proto`
with a `GoalService` (CreateGoal, GetGoal, ListGoals, UpdateGoal, DeleteGoal,
SetGoalState, ReorderGoal). Registered in `cmd/server/main.go` alongside the
existing services.

**Rationale**: Goals are a first-class domain object with their own lifecycle —
exactly the situation `plan.v1.PlanService` already models. Bolting seven RPCs
onto `task.v1.TaskService` (already 15 RPCs after the pomodoro and report
additions) would bloat the task contract with non-task semantics.

**Alternatives considered**: Extending `task.v1` — rejected: TaskService is
already the dumping ground for pomodoro RPCs; goals have a clean boundary and
their own message types, so a separate service is *less* total complexity.

## D2: Task association lives on the Task, via a dedicated `SetTaskGoal` RPC — not via UpdateTask

**Decision**: `task.v1.Task` gains a read-only `optional int64 goal_id` field
(populated on reads, ignored on Create/Update writes, like `position`).
Association changes go through one new RPC on TaskService:
`SetTaskGoal(task_id, optional goal_id)` — set to associate, omit to clear.

**Rationale**: `UpdateTask` has full-replace semantics, and the web app (out of
scope for this feature, per spec) builds its payload in
`src/lib/updatePayload.ts` without knowledge of goals. If `goal_id` were a
writable UpdateTask field, every task edit from the unmodified web app would
silently strip the task's goal. A dedicated RPC (precedent: `SetEstimate`)
keeps existing write paths — including the web app's — completely unaffected,
which is what FR-006/SC-006 ("tasks without a goal behave exactly as today")
demand.

**Alternatives considered**: Writable `goal_id` on Create/UpdateTask —
rejected for the web-app clobbering hazard above. Association RPCs on
GoalService — workable, but the mutation targets a task row; keeping task
mutations on TaskService matches the existing layout.

## D3: Storage — one migration: `goals` table + `tasks.goal_id` column

**Decision**: Migration `000010_goals`:

- `goals` table: `id` (identity PK), `user_id` (FK), `name` (VARCHAR(255),
  non-blank check), `description` (TEXT, default ''), `due` (TIMESTAMPTZ,
  nullable), `state` (TEXT, CHECK in incubating/in_progress/completed/archived,
  default 'incubating'), `position` (INTEGER) — plus an index on
  `(user_id, state, position)`.
- `tasks.goal_id BIGINT REFERENCES goals(id) ON DELETE SET NULL` — deleting a
  goal automatically clears associations without touching tasks (FR-007).

**Rationale**: TEXT + CHECK for state matches YAGNI (a PG enum type adds
migration friction for zero benefit at this scale). `ON DELETE SET NULL` makes
FR-007's delete semantics a database guarantee rather than handler code.

**Alternatives considered**: PG enum type — rejected (harder to evolve);
join table for task↔goal — rejected (cardinality is at-most-one goal per task;
a nullable FK is the simplest correct shape).

## D4: Subtree inheritance is enforced at write time, computed at read time

**Decision**: Only the association root stores `goal_id`; descendants inherit
implicitly. `SetTaskGoal` returns `FailedPrecondition` if (a) any ancestor of
the task already has a `goal_id`, or (b) the task has a descendant with its own
`goal_id` (when setting; clearing is always allowed). Clients compute a task's
effective goal as "nearest self-or-ancestor with goal_id" from the `ListTasks`
payload they already fetch.

**Rationale**: Storing the association once keeps moves/edits cheap and makes
the clarified rule ("descendants cannot belong to a different goal") structurally
impossible to violate. The client already builds the full tree from `ListTasks`
(`internal/cli.TreeNode`), so effective-goal computation is a small pure
function, testable in isolation.

**Alternatives considered**: Denormalizing goal_id onto every descendant —
rejected: write amplification on every re-parent, and two sources of truth.
Allowing nested differing goals — rejected by clarification session.

## D5: Ranking — `position` per (user, state) group; `ReorderGoal` mirrors `ReorderTask`

**Decision**: `goals.position` orders goals within their `(user_id, state)`
group. `ReorderGoal(goal_id, before_goal_id | after_goal_id)` uses the same
anchor oneof shape as `ReorderTask` and renumbers the group contiguously.
CreateGoal appends at `max(position)+1` of the user's Incubating group;
SetGoalState appends the goal to the bottom of its new group (FR-013a).

**Rationale**: Identical semantics to task ranking means the TUI `{`/`}`
handlers, help text, and tests follow the established pattern with minimal new
concepts (Principle III).

**Alternatives considered**: Global rank independent of state — rejected by
clarification session (bottom-of-group on state change was chosen).

## D6: State changes via a dedicated `SetGoalState` RPC; `UpdateGoal` is full-replace for name/description/due only

**Decision**: `UpdateGoal` replaces name, description, and due (same
full-replace contract as UpdateTask). State transitions go through
`SetGoalState(goal_id, state)`, which also performs the re-rank-to-bottom side
effect. All 4×3 transitions are accepted.

**Rationale**: State changes have a side effect (rank placement) and distinct
validation; mixing them into a full-replace update invites accidental
transitions. Precedent: Complete/UncompleteTask are separate from UpdateTask.

## D7: TUI surface — `tabGoals` is first in the tab bar; startup stays on Tasks

**Decision**: Add `tabGoals` as the first `tab` constant; tab bar order becomes
Goals · Tasks · Plan · Report. The TUI continues to open on the Tasks tab
(user decision, 2026-06-11). Two-pane layout
(list left, detail right) using the shared theme and existing pane-rendering
patterns from the Plan/Tasks tabs. List groups: Committed, then Incubating;
`c` (the Tasks tab's "toggle show all" binding) reveals Completed and Archived
groups. `{`/`}` reorder within the group; `n` new goal; `e` edit; `ctrl+d`
delete; a state-change key cycles/sets state (exact binding documented in
`contracts/goal-cli.md`). Goal detail pane shows fields plus associated task
subtrees (computed client-side per D4). From the detail pane: create a new task
attached to the goal, and link/unlink existing tasks via the existing task
picker component (`pickerState`, reused from the Plan tab's add-task flow).

**Rationale**: The spec makes Goals the first tab in the bar; day-to-day work
still lives in Tasks, so the app keeps opening there — goals are one
`shift+tab` away. Reusing the picker, theme, and key conventions is mandated
by Principle III.

**Alternatives considered**: Opening on Goals because it is first in the bar —
rejected by the user: it would put a long-horizon view in front of the daily
workflow every launch.

## D8: CLI surface — `twig goal` mirrors `twig task` conventions

**Decision**:

```
twig goal [--all]                          # list (default), grouped by state
twig goal add [--due <date>] [--desc <text>] <name>
twig goal show <id>                        # fields + associated task subtrees
twig goal mod <id> [<name>] [--due <date>] [--desc <text>]
twig goal state <id> <incubating|in-progress|completed|archived>
twig goal rm <id>
```

Task-side association reuses task commands: `twig task add --goal <id> …` and
`twig task mod <id> --goal <id>|none` (the mod path calls `SetTaskGoal`, not
UpdateTask, per D2). `twig task` detail output (`twig task show` equivalent
views) and the TUI task detail pane display the effective goal name (FR-017).

**Rationale**: Subcommand naming (`add`/`rm`/`mod`), flag style, and
"bare command lists" behavior copy `twig task` exactly — a user who knows one
predicts the other (Principle III). `--all` matches the task listing flag.

## D9: Goal data fetching — client composes goal detail from `ListGoals` + existing `ListTasks`

**Decision**: `ListGoals` returns goals only (no task data). Goal detail views
join against the already-fetched task list client-side to find associated
subtrees. `GetGoal` returns a single goal (used by `twig goal show`, which also
calls `ListTasks` for the association display).

**Rationale**: The app's established pattern is client-side composition over
the full task list (tree building, report grouping). Embedding task lists in
goal responses would duplicate task representation in a second proto package.

## D10: Testing approach

**Decision**: Follow repo conventions exactly: handler tests in
`services/twig/internal/handler/goal_test.go` are `DATABASE_URL`-gated;
CLI/TUI tests use `export_test.go` shims with string assertions; pure logic
(effective-goal computation, state grouping/ordering) gets table-driven unit
tests in the root module. No new test infrastructure.

**Rationale**: "No integration test infrastructure — tests do not require a
running database" (CLAUDE.md) except the existing gated handler pattern.
