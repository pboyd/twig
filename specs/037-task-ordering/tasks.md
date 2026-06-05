---
description: "Task list for User-Configurable Task Order"
---

# Tasks: User-Configurable Task Order

**Input**: Design documents from `/specs/037-task-ordering/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/reorder-task.md

**Tests**: Included — this repo tests handlers (via `export_test.go` shims), the CLI/builder, the TUI, and the web SPA (vitest). Test tasks mirror those established conventions.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story the task belongs to (US1, US2, US3)
- File paths are repository-relative

## Path Conventions

Full-stack layout per plan.md: shared proto `api/proto/`, server `services/twig/`, root CLI/TUI `internal/`, web SPA `services/twig-web/`. Generated code (`api/gen/`, `services/twig/internal/db/`, `services/twig-web/src/gen/`) is regenerated, never hand-edited.

---

## Phase 1: Setup & Contract (API-First)

**Purpose**: Define the ordering contract before any implementation (Constitution Principle II).

- [X] T001 Edit `api/proto/task/v1/task.proto`: add `int64 position = 9;` to the `Task` message and add the `ReorderTask` rpc plus `ReorderTaskRequest` (with `oneof anchor { before_task_id, after_task_id }`) and `ReorderTaskResponse` (repeated `Task siblings`), exactly as specified in `specs/037-task-ordering/contracts/reorder-task.md`.
- [X] T002 Run `make proto` from repo root to regenerate Go stubs in `api/gen/task/v1/` (do not hand-edit generated output).

**Checkpoint**: Contract committed and Go stubs available for the server and CLI/TUI.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Persistence + server logic + shared sibling sorting that ALL user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T003 Create migration `services/twig/db/migrations/000008_task_position.up.sql` (add `position INTEGER NOT NULL DEFAULT 0`; backfill with `ROW_NUMBER() OVER (PARTITION BY user_id, parent_id ORDER BY id) - 1`; create `tasks_user_parent_position_idx ON tasks (user_id, parent_id, position)`) and `000008_task_position.down.sql` (drop index, drop column), per `data-model.md`.
- [X] T004 Update `services/twig/db/queries/task.sql`: set `position` to end-of-group on `CreateTask`; add a query for the max sibling position in a `(user_id, parent_id)` group; add an ordered sibling-list query (`ORDER BY position, id`, `FOR UPDATE`) and a per-task position-update query for renumbering; then run `sqlc generate` (from `services/twig/`) to refresh `internal/db/`.
- [X] T005 In `services/twig/internal/handler/task.go`, populate `position` in `dbTaskToProto` so every returned `Task` carries its position. (depends on T004)
- [X] T006 In `services/twig/internal/handler/task.go`, assign end-of-group `position` on `CreateTask`, and when `UpdateTask` changes `parent_id` assign the moved task the end position of the destination group (FR-012/FR-013). (same file as T005 — sequential)
- [X] T007 In `services/twig/internal/handler/task.go`, implement the `ReorderTask` handler: validate exactly one anchor set, anchor ≠ task, anchor is a sibling (same `parent_id`/user) → `InvalidArgument`; missing task/anchor → `NotFound`; then, in a transaction, read the sibling group, remove the task, reinsert before/after the anchor, renumber to `0..n-1`, and return the updated siblings (FR-004/FR-015). Add pgx pool/`WithTx` access to the handler if not already present. (same file as T005/T006 — sequential)
- [X] T008 [P] Change `internal/cli/render.go` `SortNodes` to order siblings by `position` ascending then `id` ascending (replaces id-only sort); this is inherited by both the CLI and the TUI via `cli.BuildTree`.
- [X] T009 Add server handler tests in `services/twig/internal/handler/task_test.go`: `ReorderTask` before/after moves, anchor validation errors, full-group renumber/no-loss, end-position on `CreateTask` and on `UpdateTask` reparent, and that completing/uncompleting a task leaves its `position` and the sibling group's relative order unchanged (FR-014). (depends on T005–T007)
- [X] T010 [P] Add `internal/cli/render_test.go` coverage asserting `SortNodes`/`BuildTree` orders siblings by `position` then `id`.

**Checkpoint**: Order persists and is served via the API; CLI/TUI builders sort by position. User stories can now proceed.

---

## Phase 3: User Story 1 - Reorder tasks in the TUI (Priority: P1) 🎯 MVP

**Goal**: `{` ranks the highlighted task higher and `}` ranks it lower, among siblings (root or sub-tasks), persisting across restarts.

**Independent Test**: In the TUI, press `{`/`}` on a highlighted task → it moves one visible position and stays highlighted; boundaries are silent no-ops; relaunch shows the new order.

- [X] T011 [US1] Add `RankUp` (`{`) and `RankDown` (`}`) bindings to `internal/tui/keymap.go` and surface them in `FullHelp` (Tasks-mode group), consistent with the shared keymap/help (Principle III).
- [X] T012 [P] [US1] Add a pure helper in `internal/tui/tree.go` that, given the current `tree`, `showCompleted`, and a task id, returns its visible siblings and the previous/next visible sibling id (or none at a boundary) — used to compute the reorder anchor (skips hidden tasks per FR-005).
- [X] T013 [US1] Add `reorderTaskCmd` in `internal/tui/update.go` that calls `ReorderTask` (before/after anchor) and uses the returned `ReorderTaskResponse.siblings` to patch the affected group's positions in the in-memory model and rebuild the tree locally — no follow-up `ListTasks` round-trip — preserving the moved task's highlight (FR-006). (depends on T002)
- [X] T014 [US1] Handle `RankUp`/`RankDown` in `handleListKey` (`internal/tui/update.go`): use the T012 helper to find the visible anchor, no-op at boundaries (FR-007), otherwise dispatch `reorderTaskCmd` (before previous-visible for `{`, after next-visible for `}`). (same file as T013 — sequential)
- [X] T015 [US1] Add TUI tests in `internal/tui/update_test.go` (and `tree_test.go` for the helper): `{`/`}` swap order, boundary no-ops, hidden-sibling skip with completed filter on, and highlight preserved after reorder.

**Checkpoint**: TUI ranking fully functional and independently testable — this is the MVP.

---

## Phase 4: User Story 2 - Reorder by drag-and-drop in the web app (Priority: P2)

**Goal**: Drag a task to a new position among its siblings (root or sub-tasks); order updates immediately and persists. Reorder-only (no re-parenting).

**Independent Test**: Drag a task above/below a sibling → order updates; reload → persists; cancelled drag/drop-on-self → unchanged; cannot drop under a different parent.

- [X] T016 [US2] Add `@dnd-kit/core` and `@dnd-kit/sortable` to `services/twig-web/package.json` and run `npm install` (from `services/twig-web/`).
- [X] T017 [P] [US2] In `services/twig-web/src/lib/tree.ts`, sort each sibling list by `task.position` (then `id`) when building the tree (web stops relying on flat-list order).
- [X] T018 [P] [US2] Create `services/twig-web/src/lib/reorderAnchor.ts`: a pure function mapping a dnd-kit drop (active id, over id, direction) to a `{ before_task_id }` or `{ after_task_id }` `ReorderTask` request, returning none for drop-on-self.
- [X] T019 [P] [US2] Add reorder-related copy (e.g., drag handle aria-label, reorder error/empty hints) to `services/twig-web/src/theme/messages.ts` in the warm/playful tone (Principle IV).
- [X] T020 [US2] Make sibling rows sortable in `services/twig-web/src/components/TreeRow.tsx` using `useSortable`, restricting drag/drop targets to the same sibling group (reorder-only, FR-008) and using existing `theme/tokens.ts` for drag affordances (Principle III). (depends on T016)
- [X] T021 [US2] Wire `DndContext` + `SortableContext` and an `onDragEnd` handler in `services/twig-web/src/pages/TaskTreePage.tsx` that calls the `ReorderTask` mutation (via connect-query) using `reorderAnchor.ts`, then updates the `listTasks` query cache from the returned `ReorderTaskResponse.siblings` (e.g., `queryClient.setQueryData` on `listTasksKey`) instead of invalidating — avoiding a refetch; cancelled/no-op drags make no call (FR-009). (depends on T016, T018, T020)
- [X] T022 [P] [US2] Add web tests: `services/twig-web/src/lib/tree.test.ts` for sibling sort-by-position and `services/twig-web/src/lib/reorderAnchor.test.ts` for the drop→before/after mapping (incl. drop-on-self → no request).

**Checkpoint**: Web drag-and-drop reordering works and persists, independently of the TUI.

---

## Phase 5: User Story 3 - Consistent order across all clients (Priority: P3)

**Goal**: The user-defined order shows identically in the TUI, web app, and the read-only CLI; the CLI gains no ranking controls but respects the order.

**Independent Test**: Reorder in one client, then `./twig task` (CLI) and the web app both show siblings in that order.

- [ ] T023 [US3] Add a CLI output test in `internal/cli/task_test.go` asserting the rendered task tree (root and sub-task groups) prints siblings in `position` order (confirms FR-010/FR-011 — CLI respects the order with no new commands). (foundational T008 provides the behavior)
- [ ] T024 [US3] Execute the cross-client consistency checks from `specs/037-task-ordering/quickstart.md` (reorder in TUI → verify via CLI list and web reload show the same order).

**Checkpoint**: All three surfaces present one consistent, persisted order.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T025 [P] Tone review of all new user-facing strings (TUI help/status, web messages) against Constitution Principle IV.
- [ ] T026 [P] Run full suites and the quickstart: `go test ./...` (root), `cd services/twig && go test ./...`, `cd services/twig-web && npm test`.
- [ ] T027 [P] Verify generated code is up to date and committed where required (`make proto`, `sqlc generate`, `npm run gen`) and that no generated files were hand-edited.

---

## Dependencies & Execution Order

- **Phase 1 (Setup/Contract)** → blocks everything (proto + stubs).
- **Phase 2 (Foundational)** → blocks all user stories. Within it: T003 → T004 → T005 → T006 → T007 (T005–T007 share `task.go`, sequential); T008 and T010 are independent `[P]`; T009 depends on T005–T007.
- **Phase 3 (US1)**: depends on Phase 2 (esp. T002 stubs, T008 sort). T011 and T012 are `[P]`; T013 → T014 (same file) → T015.
- **Phase 4 (US2)**: depends on Phase 2 (T002 stubs, T005 position). T016 first; T017/T018/T019 are `[P]`; T020 then T021; T022 `[P]`.
- **Phase 5 (US3)**: depends on Phase 2 (T008). T023 then T024.
- **Phase 6 (Polish)**: after the stories it reviews.

**User story independence**: US1 (TUI) and US2 (web) each work on top of the shared foundation and can be built/tested in either order or in parallel by different developers. US3 is largely satisfied by the foundational builder change and adds verification.

## Parallel Execution Examples

- **Foundation**: T008 and T010 (CLI builder + its test) can run alongside the server work (T003–T007/T009), as they touch different files.
- **US1**: T011 (keymap) and T012 (tree helper) in parallel before T013–T015.
- **US2**: after T016, run T017 (tree sort), T018 (reorderAnchor), T019 (messages), and T022 (tests) in parallel; T020/T021 are sequential UI wiring.

## Implementation Strategy

- **MVP = Phase 1 + Phase 2 + Phase 3 (US1)**: persisted ordering controllable from the primary TUI, respected by the CLI for free.
- **Increment 2 = Phase 4 (US2)**: web drag-and-drop.
- **Increment 3 = Phase 5 (US3) + Phase 6**: explicit consistency verification and polish.
