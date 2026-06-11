# Quickstart: Goals — manual verification

**Feature**: 049-goals

Walkthrough to verify the feature end-to-end against a local stack.

## Setup

```bash
make dev                                   # postgres + server (runs migration 000010)
go build -o twig ./cmd/twig
export TWIG_API_KEY=<key>                  # from --provision-user
export TWIG_ADDR=http://localhost:8080
```

## 1. CLI lifecycle (US1, US3)

```bash
./twig goal                                # playful empty state
./twig goal add "Buy a new car" --due 2027-03-01 --desc "EV, after the bonus lands"
./twig goal add "Learn woodworking"
./twig goal                                # both under Incubating
./twig goal state 1 committed
./twig goal                                # goal 1 now under Committed (listed first)
./twig goal mod 2 "Learn furniture making"
./twig goal state 2 completed
./twig goal                                # goal 2 gone from default view
./twig goal --all                          # goal 2 visible under Completed
./twig goal state 2 committed              # reversible; reappears at bottom of Committed
./twig goal show 1                         # fields + "No tasks attached yet…"
./twig goal state 1 bogus                  # error listing the four valid states
./twig goal show 99                        # playful not-found, exit 1
```

## 2. Task association (US2)

```bash
./twig task add "Compare insurance quotes" --goal 1
./twig task add "Test drive the EV" --goal 1
./twig task add --parent <test-drive-id> "Book dealership visit"
./twig goal show 1                         # both subtrees listed (child inherits)
./twig task mod <subtask-id> --goal 2      # FailedPrecondition — ancestor owns it
./twig task mod <test-drive-id> --goal none
./twig goal show 1                         # only the insurance task remains
./twig goal rm 1                           # delete goal
./twig task                                # tasks still present, unassociated
```

## 3. TUI (US1, US2, US4)

```bash
./twig                                     # opens on the Tasks tab, as today
```

Verify:

- Tab bar order: Goals · Tasks · Plan · Report; startup tab is Tasks; one
  `shift+tab` lands on Goals with a two-pane layout in the shared theme.
- `n` creates a goal → appears at bottom of Incubating.
- `o`/`d`/`v`/`i` move it between state groups; completed/archived vanish; `c` reveals them.
- Three goals in one group: `}`/`{` reorder; restart the TUI → order kept; `twig goal` shows same order.
- `a` on a goal creates an attached task; `L` links an existing task via the picker; `U` unlinks.
- Detail pane shows fields + associated subtrees.
- Tasks tab: detail pane of an associated task shows `Goal: <name>`; tree rows unchanged; edit form's Goal field cycles none/goals.
- Empty goals list shows the playful canvas message.

## 4. Tests

```bash
go test ./...                              # root module (cli, tui, config, goal logic)
cd services/twig && go test ./...          # handler tests skip without DATABASE_URL
DATABASE_URL=postgres://twig:twig@localhost:5432/twig?sslmode=disable \
  go test ./internal/handler/ -run Goal    # gated handler integration tests
cd ../.. && go vet ./...
```

## 5. Success-criteria spot checks

| SC | Check |
|---|---|
| SC-003 | Every operation in §1 has a TUI equivalent in §3 and changes cross-appear |
| SC-004 | §1: default list hides completed; `--all` shows it |
| SC-005 | §3: rank order survives restart, matches CLI |
| SC-006 | `./twig task` flows with no goals behave exactly as before (regression: existing tests pass) |
