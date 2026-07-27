# Quickstart: Goal State Edits

## Regenerate & build

```bash
# 1. Proto: after adding GOAL_STATE_HOLD to api/proto/goal/v1/goal.proto
make proto

# 2. DB query codegen: after editing ListGoals ORDER BY in services/twig/db/queries/goal.sql
cd services/twig && sqlc generate && cd -

# 3. Build the CLI/TUI
go build -o twig ./cmd/twig
```

## Test

```bash
# Root module (TUI/CLI) — grouping, visibility, edit-form State field, Space toggle
go test ./...

# Server module — enum↔string mapping round-trip, SetGoalState(hold)
cd services/twig && go test ./... && cd -
```

## Apply the migration (dev stack)

```bash
make dev            # server auto-runs migrations on startup (includes 000012)
# or, manually, with DATABASE_URL set:
make migrate-up
```

## Manual verification (TUI)

Launch the TUI (`./twig`) and open the **Goals** tab.

1. **Hold shows by default**
   - Edit a goal, set **State** → **Hold**, save.
   - Confirm the goal stays visible in the default list, under a **Hold** group (positioned after Incubating, before Completed).
   - Press `c` ("show all") and back off: the Hold group stays visible throughout — "show all" only affects Completed/Archived.

2. **State only via the edit form**
   - Press the old keys `i`, `o`, `v`, `d` on a selected goal → nothing changes state.
   - Open the edit form; the **State** field shows the current state and cycles through Incubating · Committed · Hold · Completed · Archived. Change it and save → the goal moves groups immediately.

3. **Space completes (and un-completes)**
   - Select an active goal, press **Space** → it becomes **Completed** (with the celebratory notice).
   - With "show all" on, select the Completed goal, press **Space** → it returns to **Committed**.
   - Confirm the keybinding hints show Space for complete and no longer advertise `d`/`i`/`o`/`v`.

4. **Existing goals untouched**
   - Confirm no pre-existing goal was moved to Hold by the upgrade.

## Success check

Maps to spec Success Criteria SC-001…SC-005: Hold hides/reveals via "show all"; every state reachable through the form with no per-state hotkey; exactly one state hotkey remains (Space); Space toggles complete/uncomplete like a task; no existing goal silently reassigned.
