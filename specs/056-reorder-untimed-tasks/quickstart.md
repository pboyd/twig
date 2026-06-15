# Quickstart: Reorder Untimed Plan Entries in the TUI

## What changes for the user

In the TUI **Planning** tab, the untimed pane (top-left) now respects a user-defined order. Highlight an untimed entry and press:

- `{` — move it **higher** (earlier) among the untimed entries
- `}` — move it **lower** (later)

The entry stays highlighted so you can keep nudging it. The order persists across refreshes, day navigation, and TUI restarts. Timed entries on the grid are unaffected. These are the same keys already used to rank tasks (Tasks tab) and goals (Goals tab).

## Implementation order (for the implementer)

Follow the data flow `proto → db → handler → TUI`, contract first (Principle II).

1. **Contract** — add `ReorderPlanEntry` rpc + `ReorderPlanEntryRequest`/`ReorderPlanEntryResponse` to `api/proto/plan/v1/plan.proto` (see `contracts/reorder-plan-entry.md`), then `make proto`.
2. **Schema** — add migration `000011_plan_entry_position.{up,down}.sql`: `ADD COLUMN position SMALLINT NOT NULL DEFAULT 0`, backfill by `ROW_NUMBER() OVER (PARTITION BY user_id, day ORDER BY id)`, add `plan_entries_user_day_position_idx`. Down drops the index and column.
3. **Queries** — in `services/twig/db/queries/plan.sql`:
   - add `position` to `InsertPlanEntry` using `COALESCE((SELECT MAX(position)+1 FROM plan_entries WHERE user_id=$ AND day=$), 0)` (end-of-group default);
   - change `ListPlanEntriesForDay` ORDER BY to `start_minute ASC NULLS FIRST, position ASC, id ASC`;
   - add `LockUntimedPlanEntriesForDay` (`... WHERE user_id=$ AND day=$ AND start_minute IS NULL ORDER BY position, id FOR UPDATE`) and `UpdatePlanEntryPosition` (`UPDATE ... SET position=$ WHERE user_id=$ AND day=$ AND id=$`). Then `cd services/twig && sqlc generate`.
4. **Handler** — implement `ReorderPlanEntry` in `services/twig/internal/handler/plan.go`, mirroring `ReorderTask` (`task.go:448`): tx + validate (exists, same day, both untimed, distinct anchor, exactly one anchor side), read untimed group, splice, renumber `0..n-1`, return ordered untimed slice. Register is automatic (method on the existing `PlanService` handler).
5. **TUI** — add `reorderPlanEntryCmd` to `internal/tui/plan_update.go` (build the `before_id`/`after_id` request, return a `planMutatedMsg{highlightID: movedID}` on success). In `handlePlanningKey` (`internal/tui/update.go:1552`), add `RankUp`/`RankDown` cases: only act when `m.plan.entries[m.plan.cursor]` is untimed; find the previous/next *visible* untimed neighbor (skip hidden completed entries) and dispatch the command; otherwise no-op.

## Verify

```bash
# server-side
cd services/twig && go test ./...        # handler + db query tests

# client-side
cd /home/user/dev/twig && go test ./...  # TUI tests

# manual (full stack)
make dev                                  # postgres + server (auto-migrates)
./twig                                    # open TUI → Plan tab → untimed pane → press { / }
```

### Acceptance smoke test (maps to spec scenarios)

1. With untimed entries A, B, C and B highlighted, press `{` → order B, A, C, B still highlighted. (US1 #1)
2. Press `}` from A, B, C with B highlighted → A, C, B. (US1 #2)
3. Highlight the first untimed entry, press `{` → no change, no error. (US1 #3 / FR-005)
4. Reorder, then quit and relaunch `./twig` → order preserved. (US2 #2 / FR-006)
5. Highlight a timed grid entry, press `{`/`}` → untimed order and grid both unchanged. (FR-007 / edge case)
6. (Optional) `twig plan` CLI listing of the same day shows untimed entries in the new order — confirms the shared query reflects order without new CLI controls. (FR-009 / Assumption)
