# Quickstart: Manual verification of the move-task TUI feature

This recipe verifies the feature against the spec's acceptance scenarios end-to-end. Run it after implementation against a fresh dev stack.

## Setup

1. Start the stack:

   ```
   make dev
   ```

2. Provision a user and capture the API key (one-time):

   ```
   podman-compose exec server /todo-server --provision-user qa:qa
   ```

3. Export the CLI env vars:

   ```
   export TODO_API_KEY=<from provisioning>
   export TODO_ADDR=http://localhost:8080
   ```

4. Build the CLI:

   ```
   cd services/todo && go build -o todo ./cmd/todo
   ```

5. Seed a small task tree (so the dialog is non-trivial):

   ```
   ./todo task new "Project A"
   ./todo task new "Project B"
   ./todo task new "Subtask A1" --parent <id of Project A>
   ./todo task new "Subtask A2" --parent <id of Project A>
   ./todo task new "Subtask A1a" --parent <id of Subtask A1>
   ```

6. Launch the TUI:

   ```
   ./todo
   ```

## Verification scenarios

### Scenario 1 — Reparent a child task (Story 1 / FR-001..FR-006, FR-010)

1. Move the cursor to `Subtask A1`.
2. Press `m`. ✅ A dialog opens listing the incomplete tasks as an indented tree, with a "(no parent)" row at the top, and `Project A` highlighted as the pre-selected current parent.
3. Navigate to `Project B`. Press Enter. ✅ Dialog closes; `Subtask A1` (and its child `Subtask A1a`) now appears under `Project B`.
4. Quit (`q`) and re-launch `./todo`. ✅ The new parentage persists.

### Scenario 2 — Promote to top-level (Story 2 / FR-002, FR-003)

1. Move the cursor to `Subtask A2`.
2. Press `m`. ✅ Dialog opens; `Project A` is pre-selected.
3. Navigate to the "(no parent)" entry at the top. Press Enter. ✅ Dialog closes; `Subtask A2` appears at the root of the list.
4. Re-open the dialog on `Subtask A2`. ✅ The "(no parent)" entry is now pre-selected.

### Scenario 3 — Cancel preserves state (FR-005)

1. Cursor on any task. Press `m`. Move the cursor to a different entry. Press Esc.
2. ✅ Dialog closes; task tree is unchanged.

### Scenario 4 — Cycle prevention (FR-007, FR-008, Edge Case)

1. Cursor on `Project A`. Press `m`. ✅ Neither `Project A` itself, nor `Subtask A1`, `Subtask A2`, `Subtask A1a` appears in the candidate list (they are descendants of `Project A`).
2. If a server-side cycle is somehow triggered (e.g., racing client), confirm the dialog shows an error message in its footer and the cursor position is preserved.

### Scenario 5 — Completed tasks excluded (FR-009)

1. Complete `Project B` (space on the task in the main list).
2. Cursor on `Subtask A1`. Press `m`. ✅ `Project B` does NOT appear in the candidate list.

### Scenario 6 — `m` with no selection (Edge Case)

1. Apply a filter or scroll such that no task is selected (if reproducible in your TUI state). Press `m`. ✅ Nothing happens; the TUI state is unchanged.

### Scenario 7 — Help mentions the binding

1. Press `?`. ✅ The help screen includes an `m` row labeled "move (change parent)".

## Automated test sanity check

```
cd services/todo && go test ./internal/tui/...
```

Expected: all existing tests still pass, plus the new `move_test.go` tests.
