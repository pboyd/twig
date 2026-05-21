# Quickstart: Verifying CLI UX Improvements

Manual smoke test for feature 007. Assumes the backend is running (`TODO_ADDR`, `TODO_API_KEY` set).

From the repo root:

```sh
# Build
cd services/todo && go build -o todo ./cmd/todo && cd -

TODO=./services/todo/todo
```

## 1. `todo task` lists tasks by default

```sh
$TODO task              # should print task tree (formerly `todo task list`)
$TODO task --all        # all tasks, including completed
$TODO task --completed  # only completed
$TODO task list         # should fail: unknown subcommand
```

## 2. Completed tasks render gray + strikethrough

```sh
$TODO task add "demo-incomplete"
$TODO task add "demo-complete"
# Note the IDs printed; complete the second:
$TODO task complete <id-of-demo-complete>
$TODO task --all
# Expect: "demo-incomplete" in default style; "demo-complete" in dim + strikethrough.
# Expect: no "[ ]" or "[x]" markers anywhere in the output.

# Verify graceful degradation:
$TODO task --all | cat
# Expect: no ANSI escape codes in the captured output.
```

## 3. `todo task mod` flag-after-id and optional name

```sh
$TODO task add "keepme"            # note ID, call it $A
$TODO task add "parent-target"     # note ID, call it $B

# Re-parent without renaming:
$TODO task mod $A --parent $B
$TODO task --all
# Expect: task $A still named "keepme", now a child of $B.

# Rename only:
$TODO task mod $A "renamed"
$TODO task --all
# Expect: task $A renamed; parent unchanged.

# Both:
$TODO task mod $A "renamed-again" --parent $B
# Expect: success.

# Empty mod — should error:
$TODO task mod $A
# Expect: "nothing to update" (or similar) error message.
```

## 4. Pomodoro estimate displayed in parentheses

```sh
# Set an estimate via the existing API (no CLI subcommand is added by this feature;
# this assumes a separate code path or direct gRPC call sets task.estimate).
# Once a task has estimate=3:
$TODO task --all
# Expect: that task's line reads "[<id>] <name> (3) ..." — parens with the estimate
# immediately after the name, before any (due …) / (completed …) suffix.
```

## 5. Automated checks

```sh
cd services/todo && go test ./internal/cli/...
```

All updated tests should pass; new tests cover:
- `task` dispatch with no subcommand calls list.
- `task list` is unknown.
- `runMod` correctly handles `mod <id> --parent <p>` (name preserved).
- `renderRoots` emits dim+strikethrough only when styling is enabled.
- `renderRoots` includes ` (N)` only when `estimate > 0`.
