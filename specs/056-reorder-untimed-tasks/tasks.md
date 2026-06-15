---
description: "Task list for reorder-untimed-tasks feature implementation"
---

# Tasks: Reorder Untimed Plan Entries in the TUI

**Input**: Design documents from `/specs/056-reorder-untimed-tasks/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/reorder-plan-entry.md, quickstart.md

**Tests**: Included. The spec is not explicitly TDD, but this repo's conventions (handler/db tests via `export_test.go` shims, TUI tests) and the constitution's Quality Gates (implementation must match the committed contract) make targeted tests part of "done" here.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files / modules, no dependency on an incomplete task)
- **[Story]**: US1, US2 — maps to the user stories in spec.md
- Exact file paths are included in each task

## Path Conventions

Multi-module Go monorepo (per plan.md):
- Shared proto: `api/proto/`, generated `api/gen/`
- Server: `services/twig/` (migrations, queries, `internal/db`, `internal/handler`)
- CLI/TUI client: repo-root module, `internal/tui/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the regeneration toolchain is available before touching generated code.

- [X] T001 Verify `buf` (for `make proto`) and `sqlc` CLIs are installed and on PATH; run `go build -o twig ./cmd/twig` and `cd services/twig && go build ./...` to confirm both modules compile on a clean checkout of branch `056-reorder-untimed-tasks`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Define and generate the API contract first (Constitution Principle II — API-First). Every other task depends on the generated stubs.

**⚠️ CRITICAL**: No server or TUI work can begin until the contract is committed and stubs are regenerated.

- [X] T002 Add the `ReorderPlanEntry` rpc plus `ReorderPlanEntryRequest` and `ReorderPlanEntryResponse` messages to `api/proto/plan/v1/plan.proto`, exactly as specified in `specs/056-reorder-untimed-tasks/contracts/reorder-plan-entry.md` (request: `day`, `id`, `oneof anchor { int32 before_id = 3; int32 after_id = 4; }`; response: `repeated PlanEntry untimed = 1`).
- [X] T003 Regenerate ConnectRPC stubs by running `make proto`; confirm `api/gen/plan/v1/` now exposes `ReorderPlanEntry` and the new request/response types (do not hand-edit generated files).

**Checkpoint**: Contract is the source of truth; server and client implementations can now conform to it.

---

## Phase 3: User Story 1 - Reorder untimed entries in the TUI planning tab (Priority: P1) 🎯 MVP

**Goal**: Pressing `{` / `}` on a highlighted untimed entry in the Planning tab moves it earlier / later within the untimed pane, the pane re-renders immediately, and the moved entry stays highlighted. Timed entries are untouched; boundary presses are no-ops.

**Independent Test**: Open the Planning view on a day with three or more untimed entries (A, B, C), highlight B, press `{` → order becomes B, A, C with B highlighted; press `}` twice → A, C, B. Highlight the first entry and press `{` → no change, no error. Highlight a timed grid entry and press `{`/`}` → nothing reorders.

### Server (persistence + reorder RPC)

- [X] T004 [US1] Create migration `services/twig/db/migrations/000011_plan_entry_position.up.sql` and `000011_plan_entry_position.down.sql`: up adds `position SMALLINT NOT NULL DEFAULT 0` to `plan_entries`, backfills via `ROW_NUMBER() OVER (PARTITION BY user_id, day ORDER BY id) - 1`, and creates `plan_entries_user_day_position_idx ON plan_entries (user_id, day, position)`; down drops the index then the column (see `data-model.md`).
- [X] T005 [US1] Update `services/twig/db/queries/plan.sql`: (a) add `position` to `InsertPlanEntry` using `COALESCE((SELECT MAX(position)+1 FROM plan_entries WHERE user_id = $1 AND day = $2), 0)` so new entries land at the end of the day group; (b) change `ListPlanEntriesForDay` ORDER BY to `start_minute ASC NULLS FIRST, position ASC, id ASC`; (c) add `LockUntimedPlanEntriesForDay` (`SELECT * FROM plan_entries WHERE user_id = $1 AND day = $2 AND start_minute IS NULL ORDER BY position, id FOR UPDATE`); (d) add `UpdatePlanEntryPosition` (`UPDATE plan_entries SET position = $4 WHERE user_id = $1 AND day = $2 AND id = $3`).
- [X] T006 [US1] Regenerate the query layer: `cd services/twig && sqlc generate`; confirm `internal/db` has the new `LockUntimedPlanEntriesForDay`/`UpdatePlanEntryPosition` methods and updated `InsertPlanEntry` params (do not hand-edit generated files).
- [X] T007 [US1] Implement `ReorderPlanEntry` in `services/twig/internal/handler/plan.go`, mirroring `ReorderTask` (`services/twig/internal/handler/task.go:448`): begin tx; resolve `before_id`/`after_id` anchor (reject nil/both → `InvalidArgument`, reject `anchor == id`); `GetPlanEntry` for the moved entry and anchor (→ `NotFound` if missing); validate both share `day` and both have nil `start_minute` (→ `InvalidArgument` if timed or cross-day); `LockUntimedPlanEntriesForDay`; splice the moved entry to the anchor position; renumber `0..n-1` via `UpdatePlanEntryPosition`; commit; return the reordered untimed slice as `ReorderPlanEntryResponse.untimed`.
- [X] T008 [US1] Add handler tests in `services/twig/internal/handler/plan_test.go` (using the existing `export_test.go` shims): before-move and after-move reorder within the untimed group; boundary no-op (first-higher / last-lower) returns OK with unchanged order; timed entry as mover and as anchor each rejected with `InvalidArgument`; missing entry/anchor → `NotFound`; no-loss invariant (same set of ids before/after, contiguous positions).

### TUI (key wiring)

- [X] T009 [P] [US1] Add `reorderPlanEntryCmd(client, day, id, anchorID, insertBefore)` to `internal/tui/plan_update.go`: build a `planv1.ReorderPlanEntryRequest` with `before_id` or `after_id` set, call `ReorderPlanEntry`, and return `planMutatedMsg{highlightID: id}` on success (error → `planMutatedMsg{err}`), reusing the existing reload/highlight path.
- [X] T010 [US1] In `handlePlanningKey` (`internal/tui/update.go:1552`), add `RankUp` and `RankDown` cases: act only when `m.plan.entries[m.plan.cursor]` is untimed (`StartMinute == nil`); add a small helper that, given the cursor entry, returns the previous / next *visible untimed* neighbor (skipping hidden completed entries, per the spec edge case); on `{` dispatch `reorderPlanEntryCmd(..., prevID, insertBefore=true)`, on `}` dispatch with `nextID, insertBefore=false`; no-op (no request, no error) when on a timed entry or at the visible boundary.
- [X] T011 [P] [US1] Add a TUI test (e.g. `internal/tui/plan_update_test.go` or the existing plan test file) asserting that `{`/`}` on an untimed entry issue a `ReorderPlanEntry` with the correct anchor and that a timed-entry highlight / boundary produces no command.

**Checkpoint**: User Story 1 is fully functional end-to-end — `{`/`}` visibly reorder untimed entries in the TUI and the change is stored server-side.

---

## Phase 4: User Story 2 - Reordered untimed entries stay put (Priority: P2)

**Goal**: The user-defined untimed order survives view refresh, day navigation, and a full TUI restart; newly added or newly-unscheduled untimed entries appear at the end of the group without disturbing existing order.

**Independent Test**: Reorder a day's untimed entries, quit and relaunch `./twig`, reopen the same day → entries appear in the user-defined order. Add a new untimed entry → it appears last; existing order unchanged.

> Persistence is delivered structurally by US1 (the `position` column + the `ListPlanEntriesForDay` ordering). This phase verifies that guarantee and locks in the end-of-group placement for new/re-entering entries.

- [X] T012 [US2] Add a db/handler regression test (in `services/twig/internal/handler/plan_test.go`) that reorders untimed entries, then calls `ListPlanEntries` for the day and asserts the returned untimed entries come back in the user-defined `position` order (proves reload/restart reflect the stored order, since the TUI re-lists on launch).
- [X] T013 [US2] Add a test asserting new-entry placement: after `AddPlanTask`/`AddPlanEvent` of an untimed entry on a day that already has untimed entries, the new entry receives the highest `position` (lands at the end of the group) and existing entries keep their relative order — and that an entry whose start time is cleared back to untimed (`MovePlanEntry` to untimed) likewise lands at the end.
- [X] T014 [US2] Manual restart verification per `quickstart.md`: with the full stack up (`make dev`), reorder untimed entries in `./twig`, quit, relaunch, and confirm the order persists; confirm a freshly added untimed entry appears last.

**Checkpoint**: Reordering is durable across restarts and stable as the day's entry set changes.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Constitution compliance and final verification.

- [X] T015 [P] Ensure the only new user-facing string — the reorder-failure message surfaced in the plan status bar (`m.plan.err`) — carries the established warm/playful tone (Constitution Principle IV); reorder succeeds silently with no notice, matching task reordering.
- [X] T016 [P] Confirm the Planning-tab help row still surfaces `{`/`}` "rank higher / rank lower" (`internal/tui/keymap.go:361`) so the gesture is discoverable and consistent with the Tasks/Goals tabs (Constitution Principle III); no new key binding is introduced.
- [X] T017 Run the full suites and a manual smoke test: `cd /home/user/dev/twig && go test ./...` and `cd services/twig && go test ./...` both green; then execute the acceptance smoke tests in `quickstart.md` against `make dev`.
- [X] T018 Quality-gate sweep: confirm no `[ALL_CAPS_PLACEHOLDER]` tokens remain in the feature's spec/plan/contracts, and that the committed contract in `contracts/reorder-plan-entry.md` matches the implemented proto and handler behavior.

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (Phase 1)**: no dependencies.
- **Foundational (Phase 2)**: depends on Setup. **Blocks everything** — the proto stubs (T003) are imported by both the handler and the TUI.
- **User Story 1 (Phase 3)**: depends on Foundational. This is the MVP.
- **User Story 2 (Phase 4)**: depends on US1 (it verifies and extends US1's persistence). Not independently shippable before US1, by design — persistence has no meaning without the reorder action.
- **Polish (Phase 5)**: depends on US1 (and US2 for full verification).

### Task-level dependencies (within US1)

- T004 (migration) → T005 (queries) → T006 (sqlc gen) → T007 (handler) → T008 (handler tests).
- T003 (stubs) → T009 (TUI command) → T010 (TUI keys) → T011 (TUI test).
- T009/T010 (TUI, root module) are independent of T004–T008 (server module) once T003 is done — they can proceed in parallel.

### Parallel opportunities

- **Cross-module after T003**: the server chain (T004→T008) and the TUI chain (T009→T011) run concurrently — different modules, no shared files.
- Tasks marked **[P]**: T009 (different module from the server chain), T011 (test file, after T010), T015 and T016 (independent polish in different files).
- T008 and T009 can start together once their respective prerequisites (T007 for T008; T003 for T009) are met.

### Parallel execution example

```
After T003 (stubs regenerated):
  ├─ Track A (server):  T004 → T005 → T006 → T007 → T008
  └─ Track B (TUI):     T009 → T010 → T011        [P with Track A]
Then converge on Phase 4 (T012, T013, T014) and Phase 5.
```

---

## Implementation Strategy

### MVP first

Complete **Phase 1 → Phase 2 → Phase 3 (US1)**. That delivers the entire user-visible feature: `{`/`}` reorder untimed entries in the TUI and the order is stored server-side. Ship/demo here.

### Incremental delivery

1. Foundational contract (T002–T003) — API-first gate.
2. US1 (T004–T011) — working, persisted reorder. **MVP.**
3. US2 (T012–T014) — verify durability + new-entry placement.
4. Polish (T015–T018) — tone, consistency, full verification.

---

## Summary

- **Total tasks**: 18
- **Setup**: 1 (T001)
- **Foundational**: 2 (T002–T003)
- **User Story 1 (P1, MVP)**: 8 (T004–T011) — server schema/queries/handler + TUI keys, with tests
- **User Story 2 (P2)**: 3 (T012–T014) — persistence verification + new-entry placement
- **Polish**: 4 (T015–T018)
- **Parallel opportunities**: server chain ∥ TUI chain after T003; plus T015/T016 polish in parallel
- **Suggested MVP scope**: Phases 1–3 (through T011)
