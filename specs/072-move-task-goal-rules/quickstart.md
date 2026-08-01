# Quickstart: Verifying Goal Handling on Task Moves

Manual verification for the three reported issues, plus the regressions most likely to be introduced.
Run after implementation; each scenario maps to acceptance scenarios in [spec.md](./spec.md).

## Setup

```bash
make dev                    # postgres + server on :8080
export TWIG_ADDR=http://localhost:8080
export TWIG_API_KEY=...     # from --provision-user
go build -o twig ./cmd/twig
```

Create two goals and a starting tree:

```bash
./twig goal add "Fitness"
./twig goal add "Health"
./twig task add "Run a 5k"          # → A
./twig task add "Sleep better"      # → B
./twig task add "Buy shoes"         # → C, will become a sub-task
```

Link goals to the two top-level tasks, then note the ids — the scenarios below use `A`, `B`, `C` for
readability.

```bash
./twig task goal <A> "Fitness"
./twig task goal <B> "Health"
./twig task edit <C> --parent <A>   # C becomes a sub-task of A
./twig task list
```

Expect: `A` under Fitness, `C` nested beneath it showing Fitness, `B` under Health.

## Scenario 1 — The move that used to fail (US1, FR-001, FR-002)

```bash
./twig task edit <A> --parent <B>
```

- **Expect**: succeeds. No `failed_precondition` and no mention of clearing a goal link.
- **Before this change**: `failed_precondition: moving this task would nest goal associations — clear the goal link first`

```bash
./twig task list
```

- **Expect**: `A` nested under `B`, displaying **Health**. `A` holds no goal link of its own.
- **Expect**: `C` moved with `A` and also displays Health.

Confirm in the database that the invariant holds:

```bash
psql postgres://twig:twig@localhost:5432/twig \
  -c "SELECT id, name, parent_id, goal_id FROM tasks WHERE parent_id IS NOT NULL AND goal_id IS NOT NULL;"
```

- **Expect**: zero rows.

## Scenario 2 — Same goal on both sides (US1 scenario 2)

Reset so `A` and `B` are both top-level and both linked to Fitness, then:

```bash
./twig task edit <A> --parent <B>
```

- **Expect**: succeeds silently. `A` displays Fitness through `B`.
- **Before this change**: refused, even though the goals matched.

## Scenario 3 — Descending under a goal-free task (US1 scenario 3, FR-002)

With `A` top-level and linked to Fitness, and `D` a top-level task with no goal:

```bash
./twig task edit <A> --parent <D>
./twig task list
```

- **Expect**: `A` nested under `D` with **no goal shown**, and no goal link stored on `A`.
- **Before this change**: the move produced a nested task still holding a Fitness link.

## Scenario 4 — Promotion preserves the inherited goal (US2, FR-004)

With `C` nested under `A`, and `A` linked to Fitness:

```bash
./twig task edit <C> --parent 0     # or the TUI move dialog → "(no parent)"
./twig task list
```

- **Expect**: `C` is top-level and linked to **Fitness** in its own right.
- **Before this change**: `C` arrived at the top level with no goal at all.

Verify the link is stored, not merely displayed:

```bash
psql postgres://twig:twig@localhost:5432/twig \
  -c "SELECT id, name, parent_id, goal_id FROM tasks WHERE id = <C>;"
```

- **Expect**: `parent_id` NULL, `goal_id` set to the Fitness goal.

## Scenario 5 — Promotion from depth ≥ 2 (US2 scenario 2)

Build `A (Fitness) → B → C`, then promote `C`:

```bash
./twig task edit <C> --parent 0
```

- **Expect**: `C` linked to Fitness — the **nearest** goal-bearing ancestor, found two levels up.

## Scenario 6 — Promotion with no goal anywhere (US2 scenario 3, FR-005)

With `D` a goal-free top-level task and `E` nested under it:

```bash
./twig task edit <E> --parent 0
```

- **Expect**: succeeds. `E` is top-level with no goal. No error.

## Scenario 7 — Renaming must not clear a goal (FR-008)

This is the regression the full-replace semantics make easy to introduce. With `A` top-level and
linked to Fitness:

```bash
./twig task edit <A> --name "Run a 10k"
./twig task list
```

- **Expect**: still linked to Fitness.

Then the nested case — with `C` nested under `A`:

```bash
./twig task edit <C> --name "Buy better shoes"
```

- **Expect**: no goal links anywhere in the subtree are touched. Nothing is cleared.

Also verify via the web UI, which sends a full-replace payload on every edit:

```bash
cd services/twig-web && npm run dev   # → http://localhost:5173
```

Rename a goal-linked task inline. **Expect**: the goal badge does not disappear.

## Scenario 8 — Reordering must not clear a goal (FR-008)

With two goal-linked top-level tasks, reorder them in the TUI (`twig`, Tasks tab) or via the reorder
command.

- **Expect**: both keep their goals. Reordering is not a parent change.

## Scenario 9 — Non-goal rejections still reject (FR-009)

```bash
./twig task edit <A> --parent <C>    # C is a descendant of A
```

- **Expect**: `parent_id would create a cycle`. Nothing moved, no goal link changed.

```bash
./twig task done <B>
./twig task edit <A> --parent <B>
```

- **Expect**: `Task <B> is already crossed off — nothing moves under a finished job.` No goal link
  changed.

## Scenario 10 — Closed goal is carried on promotion (FR-007, G4)

With `C` nested under `A`, `A` linked to Fitness:

```bash
./twig goal complete "Fitness"
./twig task edit <C> --parent 0
```

- **Expect**: `C` is top-level and **still linked to Fitness**, despite the goal being completed.
  This is deliberate — see research.md decision 5. It is the one case where a move creates a link
  that `./twig task goal` would refuse to create directly.

Confirm the direct path still refuses it:

```bash
./twig task goal <D> "Fitness"
```

- **Expect**: `cannot link a task to a completed or archived goal`.

## Scenario 11 — Legacy rows are repaired on move (US3, FR-003)

Hand-craft a violating row, then move its ancestor:

```bash
psql postgres://twig:twig@localhost:5432/twig \
  -c "UPDATE tasks SET goal_id = <fitness_id> WHERE id = <C>;"   # C is nested — illegal state
./twig task edit <A> --parent <D>
psql postgres://twig:twig@localhost:5432/twig \
  -c "SELECT id, parent_id, goal_id FROM tasks WHERE id = <C>;"
```

- **Expect**: `C.goal_id` is NULL. The subtree clear reaches descendants, not just direct children.

## Scenario 12 — Position on promotion (G7)

Promote a sub-task to the top level while several top-level tasks exist.

- **Expect**: it lands at the **end** of the top-level list, not at an arbitrary position inherited
  from the group it left.
- **Before this change**: the repositioning step was skipped for promotions.

## Automated checks

```bash
go test ./...                        # root module: CLI/TUI
cd services/twig && go test ./...    # server: handler tests need DATABASE_URL
```

Handler tests requiring a database read `DATABASE_URL`; see `testPool` in
`services/twig/internal/handler/pomodoro_test.go`.

```bash
export DATABASE_URL="postgres://twig:twig@localhost:5432/twig?sslmode=disable"
```
