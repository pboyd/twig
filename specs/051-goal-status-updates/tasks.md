# Tasks: Goal Status Updates

**Input**: Design documents from `/specs/051-goal-status-updates/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/status-update-rpc.md, quickstart.md

**Tests**: Included — the repo has established test conventions (plan §Testing); every prior feature ships tests alongside implementation.

**Organization**: Tasks are grouped by user story. All server-side work is foundational: the ConnectRPC handler must implement the full generated `GoalService` interface (including all four new status-update RPCs) to compile, so proto + migration + queries + handler land in Phases 1–2. Every story phase is then purely client-side TUI. US1 (record + read latest) is the MVP; US2 (browse history) and US3 (edit/delete) are independent increments on the same foundation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 (record + latest), US2 (browse history), US3 (edit/delete)

## Phase 1: Setup (API contract)

**Purpose**: Commit the API contract change and regenerate stubs — required by Principle II before any implementation.

- [x] T001 Edit `api/proto/goal/v1/goal.proto` per `contracts/status-update-rpc.md`: add the `StatusUpdate` message, the read-only `StatusUpdate latest_status_update = 7` field on `Goal`, the four RPCs (`ListGoalStatusUpdates`, `AddGoalStatusUpdate`, `UpdateGoalStatusUpdate`, `DeleteGoalStatusUpdate`) and their request/response messages; run `make proto` to regenerate `api/gen/` (the server will not compile until T004 implements the new methods — expected mid-phase state)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema, queries, and the complete server-side implementation of the four RPCs plus latest-update population. The TUI stories all consume this.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T002 [P] Create migration `services/twig/db/migrations/000011_goal_status_updates.up.sql` / `.down.sql` per `data-model.md` — `goal_status_updates` table (identity PK, `goal_id BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE`, `body TEXT NOT NULL CHECK (btrim(body) <> '')`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`) with index `goal_status_updates_goal_created_idx ON goal_status_updates (goal_id, created_at DESC, id DESC)`; the `down` drops the table
- [x] T003 Add queries to new `services/twig/db/queries/goal_status_update.sql` exactly per `contracts/status-update-rpc.md` (`ListGoalStatusUpdates`, `AddGoalStatusUpdate`, `UpdateGoalStatusUpdate`, `DeleteGoalStatusUpdate`, `GetLatestGoalStatusUpdate`, plus a batched latest-per-goal query for list population), each scoped via a join to `goals` on `user_id`; run `sqlc generate` in `services/twig/` (depends on T002)
- [x] T004 Implement the four status-update RPCs on the existing `Goal` handler in `services/twig/internal/handler/goal.go` per `contracts/status-update-rpc.md` — trim+validate `body` (empty → `InvalidArgument` with playful copy) before the DB call; map empty `:one` results (goal/update not owned) to `NotFound`; preserve `created_at` on update; and populate `Goal.latest_status_update` in `ListGoals` (batched, single round trip — no N+1) and `GetGoal`; user scoping from auth context throughout (depends on T001, T003)
- [x] T005 [P] Add handler tests in `services/twig/internal/handler/goal_test.go` (via `export_test.go` shims, no DB) — add/list/update/delete happy paths, newest-first ordering, empty/whitespace body → `InvalidArgument`, cross-user goal/update → `NotFound`, `created_at` unchanged by update, and `latest_status_update` populated by `ListGoals`/`GetGoal` (unset when none) (depends on T004)

**Checkpoint**: Foundation ready — `cd services/twig && go test ./...` passes; quickstart §"Manual verification" server behavior is exercisable. TUI stories can proceed (in parallel if desired).

---

## Phase 3: User Story 1 - Record and read the latest status (Priority: P1) 🎯 MVP

**Goal**: From the Goals tab, press `S` on a goal to compose a status update in `$EDITOR`; on save it is recorded and the goal's detail pane shows it as "Latest status" (markdown-rendered, with a relative timestamp). Empty input records nothing.

**Independent Test**: Without the history view: select a goal, press `S`, save markdown text → detail pane shows it as the latest with a timestamp and rendered formatting; add a newer one → it replaces the shown latest; save empty → nothing recorded, notice shown; a goal with no updates shows the friendly empty state.

### Implementation for User Story 1

- [x] T006 [US1] In `internal/tui/model.go`, add a status-compose context to `goalState` — a `statusCompose` struct holding the target `goalID` (and an `editingID int64` defaulting to 0 for add, reused by US3) and a flag indicating an in-flight compose; no new mode needed for quick-add
- [x] T007 [US1] In `internal/tui/update.go`, add the `addGoalStatusUpdateCmd(goalClient, goalID, body)` command factory and a `goalStatusMutationMsg` (carrying the goal id / err), plus the `S` key handler in the goals-list handler: set `statusCompose{goalID}` and return `openEditorCmd("")`; route `editorFinishedMsg` while a status compose is active to validate (discard empty/whitespace with a playful notice) and otherwise call `addGoalStatusUpdateCmd`; on success refresh via `listGoalsCmd` so the embedded latest updates (depends on T006)
- [x] T008 [P] [US1] In `internal/tui/keymap.go`, add the `GoalAddStatus` binding (`S`, help "add status") to the goals keymap and include it in the goals-tab help (FullHelp/ShortHelp as appropriate)
- [x] T009 [US1] In `internal/tui/goal_view.go` `renderGoalDetail`, add a "Latest status" section after the description: when `g.GetLatestStatusUpdate()` is set, render a relative timestamp (e.g. "2d ago") plus the body via `m.md.Render` and a dim `[s] status history` hint; when unset, render a playful empty state (e.g. "No status yet — how's it going?"); all styles from `theme.go` (depends on T007)
- [x] T010 [US1] Add tests in `internal/tui/goal_view_test.go` and `internal/tui/update_test.go` — latest-status section renders body + timestamp (markdown applied, not raw), empty-state copy when none, `S` opens the editor, empty editor result is discarded with a notice and sends no RPC, non-empty result triggers add + refresh (depends on T009)

**Checkpoint**: User Story 1 fully functional — a user can record and re-read the latest status entirely from the goal detail pane, no history view required.

---

## Phase 4: User Story 2 - Browse the history of status updates (Priority: P2)

**Goal**: Press `s` on a goal to open a dedicated status-history view: a newest-first master list (timestamp + first line) with a cursor; `enter` opens a scrollable full reader for the selected update; `esc` steps back.

**Independent Test**: On a goal with several updates, press `s` → all updates listed newest-first with timestamps; ↑/↓ moves the cursor; `enter` opens the reader; for a 5000+ char entry the reader scrolls through all of it with no truncation; `esc` returns to the list, `esc` again to the goal detail.

### Implementation for User Story 2

- [x] T011 [US2] In `internal/tui/model.go`, extend `goalState` with history view state — modes `goalStatusHistory` and `goalStatusReader`, a `statusUpdates []*goalv1.StatusUpdate` slice, a `statusCursor int`, and a `readerOffset int` for reader scrolling (depends on T006)
- [x] T012 [US2] In `internal/tui/update.go`, add `listGoalStatusUpdatesCmd(goalClient, goalID)` and a `goalStatusListMsg`; the `s` key handler in the goals-list handler enters `goalStatusHistory` and fires the list command; add a `handleGoalStatusKey` handler for the history list (↑/↓ move `statusCursor`, `enter` → `goalStatusReader` resetting `readerOffset`, `esc` → back to goals list) and reader scrolling (↑/↓/pgup/pgdn adjust `readerOffset`, `esc` → back to list); dispatch to these handlers from the central key router when in the new modes (depends on T011)
- [x] T013 [US2] Create `internal/tui/goal_status.go` — `renderStatusHistory(width, height)` (newest-first list: relative/absolute timestamp + markdown-inline first line, cursor highlight via `theme.go`, playful empty state) and `renderStatusReader(width, height)` (selected update's timestamp header + `m.md.Render` body, vertically scrolled by `readerOffset` with a scroll indicator); both as full-view content replacing the two panes (depends on T012)
- [x] T014 [US2] Wire the new modes into the goals view dispatch in `internal/tui/goal_view.go` (`viewGoals`) / `internal/tui/view.go` so `goalStatusHistory`/`goalStatusReader` render `goal_status.go` output, and add `GoalStatusHistory` (`s`) binding + history/reader help entries to `internal/tui/keymap.go` (depends on T013)
- [x] T015 [US2] Add tests in `internal/tui/goal_status_test.go` — `s` loads and enters history, list ordering newest-first with timestamps, cursor nav bounds, `enter` opens reader, reader scrolls a long body (offset changes visible window, no truncation), `esc` transitions back through reader → list → detail, empty-history copy (depends on T014)

**Checkpoint**: User Story 2 functional on top of US1 — full timestamped history is browsable and long entries are fully readable.

---

## Phase 5: User Story 3 - Correct or remove a status update (Priority: P3)

**Goal**: From the history view, `n` composes a new update, `e` edits the selected update's text (in `$EDITOR`, prefilled), and `d` deletes the selected update after a confirmation. Edits keep the original timestamp.

**Independent Test**: In the history view, `e` opens `$EDITOR` prefilled with the selected text → save shows updated text (markdown), timestamp unchanged; `d` then `y` removes the update, siblings remain; deleting the last update returns the detail pane to the empty state; `n` records a new update without leaving history.

### Implementation for User Story 3

- [x] T016 [US3] In `internal/tui/update.go`, add `updateGoalStatusUpdateCmd(goalClient, id, body)` and `deleteGoalStatusUpdateCmd(goalClient, id)` factories; extend `handleGoalStatusKey` with `n` (set `statusCompose{goalID, editingID:0}`, `openEditorCmd("")`), `e` (set `statusCompose{goalID, editingID:selected.Id}`, `openEditorCmd(selected.Body)`), and `d` (enter a `goalStatusConfirmDelete` mode with playful confirm copy; `y` → delete, `n`/`esc` → cancel); route `editorFinishedMsg` to update vs add based on `editingID`; after any mutation refresh both `listGoalStatusUpdatesCmd` (history) and `listGoalsCmd` (embedded latest) (depends on T012)
- [x] T017 [US3] In `internal/tui/keymap.go` add `GoalStatusNew`/`GoalStatusEdit`/`GoalStatusDelete` bindings (`n`/`e`/`d`) scoped to the history view with help entries, and render the delete confirmation prompt in `internal/tui/goal_status.go` (depends on T016)
- [x] T018 [US3] Add tests in `internal/tui/goal_status_test.go` — `e` opens editor prefilled and saves via update (timestamp preserved), `n` adds from history, `d`+`y` deletes selected (siblings unaffected) and `d`+`n`/`esc` cancels, deleting the only update returns the detail to empty state, empty editor result discarded for both add and edit (depends on T017)

**Checkpoint**: All user stories complete — record, browse, and manage status updates end-to-end.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T019 [P] Review all new user-facing copy for the playful-but-actionable tone (Principle IV): empty-history state, no-status detail state, empty-body discard notice, delete confirmation, and server `InvalidArgument`/`NotFound` messages across `goal_view.go`, `goal_status.go`, `update.go`, and `handler/goal.go`
- [x] T020 [P] Update the Goals-tab help/legend so `s` and `S` (and history-mode `n`/`e`/`d`/`enter`/`esc`) appear correctly in `internal/tui/help.go`/`keymap.go`, consistent with existing per-mode help
- [x] T021 Run the full quickstart manual verification (`specs/051-goal-status-updates/quickstart.md`, US1–US3 + edge/scoping incl. goal-delete cascade and cross-user isolation) and both test suites (`go test ./...` and `cd services/twig && go test ./...`), plus a migration up/down round trip; fix any gaps

---

## Dependencies & completion order

```
Phase 1 (T001 proto)
   └─> Phase 2 (T002 → T003 → T004 → T005)   ← foundation, blocks all stories
          ├─> Phase 3 US1 (T006 → T007 → {T008 [P]} → T009 → T010)   🎯 MVP
          ├─> Phase 4 US2 (T011 → T012 → T013 → T014 → T015)
          │        └─> Phase 5 US3 (T016 → T017 → T018)   (builds on US2's history view)
          └─> Phase 6 Polish (T019 [P], T020 [P], T021)   ← after stories land
```

- **US1** depends only on the foundation (uses the embedded `latest_status_update` + `AddGoalStatusUpdate`). Independently shippable as the MVP.
- **US2** depends on the foundation (uses `ListGoalStatusUpdates`); independent of US1.
- **US3** depends on **US2** (its `n`/`e`/`d` actions operate on US2's history list/selection).

## Parallel opportunities

- Phase 2: **T002** (migration) runs in parallel with starting on the proof of T003 once the schema shape is fixed; **T005** (tests) parallels other modules once T004 lands.
- Phase 3: **T008** (keymap) is `[P]` — different file from the model/update/view work.
- Stories: once Phase 2 is done, **US1** and **US2** can be built in parallel by different developers (US3 waits on US2).
- Phase 6: **T019** and **T020** are `[P]`.

## Implementation strategy

- **MVP = Phase 1 + Phase 2 + Phase 3 (US1)**: a user can record and re-read the latest status of a goal — the core value of the feature.
- Then layer **US2** (browse history) and **US3** (edit/delete) as independent increments, each independently testable per its checkpoint.
