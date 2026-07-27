---
description: "Task list for 069-tui-auto-refresh"
---

# Tasks: TUI Auto-Refresh

**Input**: Design documents from `/specs/069-tui-auto-refresh/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/refresh-contract.md](./contracts/refresh-contract.md)

**Tests**: Included. The spec asks for them directly — SC-003 requires a suite covering every interactive mode on every tab, and the contract's §C5 table enumerates thirteen acceptance rows.

**Organization**: Grouped by user story so each ships independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Exact file paths are included in every task

## Path Conventions

Single Go package: `internal/tui/` in the root module (`github.com/pboyd/twig`). No changes in `api/`, `services/twig/`, or `services/twig-web/`.

## Phase ordering note

Phases run **US1 → US3 → US2**, not in priority-number order. US3 (P3, non-disruption) is sequenced before US2 (P2, the heartbeat) deliberately: US3's tasks make the four load handlers safe to invoke unprompted, and releasing the heartbeat first would fire unprompted loads into handlers that still clamp the cursor by index and still surface transient network errors. That is worse than no auto-refresh at all. The priority labels reflect user value as specified; the ordering reflects that one is a prerequisite for the other's safety. This mirrors Phases B and C in `plan.md`.

---

## Phase 1: Setup

**Purpose**: Establish a green baseline. There is no project to initialize — this feature adds no dependency, file, or tool.

- [X] T001 Run `go build -o twig ./cmd/twig` and `go test ./internal/tui/...` from the repo root and confirm both pass before any change

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The per-tab load timestamps and their test shims. Required by US2 and US3.

**⚠️ Note**: US1 does **not** depend on this phase and may be implemented in parallel with it.

- [X] T002 Add `tasksLastLoad time.Time` to the `Model` struct and `lastLoad time.Time` to `goalState`, `planState`, and `reportState` in `internal/tui/model.go`, per the field table in `data-model.md`
- [X] T003 Add test shims to `internal/tui/export_test.go` for reading and writing the four `lastLoad` fields and for setting `m.nowFunc`, following the existing `ExportSetPlanEntries` style

**Checkpoint**: The timestamp fields exist and are reachable from tests. They are not yet written or read.

---

## Phase 3: User Story 1 — Switching tabs shows current data (Priority: P1) 🎯 MVP

**Goal**: Entering any tab reloads it, so a change made on one tab is visible on another without a keystroke.

**Independent test**: Complete a task on the Plan tab, press `shift+tab` to the Tasks tab, and see it completed. Separately, visit the Goals tab twice with a goal change in between and see the change.

**Ships alone**: yes — no timer, no new state, no dependency on Phase 2.

### Implementation

- [X] T004 [US1] Add `listTasksCmd(m.client)` and `listScheduledDaysCmd(m.planClient)` to the Goals→Tasks `NextTab` path at `internal/tui/update.go:1430`, batching them with `tea.Batch`
- [X] T005 [US1] Add `listTasksCmd(m.client)` to the Plan→Tasks `PrevTab` path at `internal/tui/update.go:1890`, batching it with the existing `listScheduledDaysCmd` call
- [X] T006 [US1] Remove the `if !m.goal.loaded` guard from the Report→Goals `NextTab` path at `internal/tui/update.go:1310` so `listGoalsCmd` is always dispatched
- [X] T007 [US1] Remove the `if !m.goal.loaded` guard from the Tasks→Goals `PrevTab` path at `internal/tui/update.go:2256` so `listGoalsCmd` is always dispatched
- [X] T008 [US1] Read the Plan-entry and Report-entry tab paths in `internal/tui/update.go` and confirm both already dispatch unconditionally; record the finding in the task list and make no change if so

### Tests

- [X] T009 [P] [US1] Create `internal/tui/tab_refresh_test.go` with a table-driven test asserting that every one of the eight tab-entry transitions returns a non-nil command that dispatches that tab's fetch, using `fakeTaskClient` and `fakePlanClient` to record calls (contract A1, A2, C1.1)
- [X] T010 [P] [US1] Add a test to `internal/tui/tab_refresh_test.go` asserting that entering the Goals tab a second time still dispatches `listGoalsCmd` when `m.goal.loaded` is already true

**Checkpoint**: US1 is complete and independently shippable. The completed-task-on-Plan bug is fixed.

---

## Phase 4: User Story 3 — Automatic refreshes never disturb work in progress (Priority: P3)

**Goal**: Make the four load handlers safe to invoke unprompted — cursor anchored by identity, open forms untouched, errors and scroll left alone.

**Independent test**: Drive each load handler directly with a result that inserts an item above the cursor and confirm the cursor stays on the same item; drive each with an error and confirm nothing visible changes when the load was background-flavored.

**Depends on**: Phase 2 (the `lastLoad` fields).

**Why before US2**: see the phase ordering note above.

### Message plumbing

- [X] T011 [US3] Add a `bg bool` field to `listTasksResultMsg`, `listGoalsResultMsg`, and `reportResultMsg` in `internal/tui/update.go`, with a comment noting it marks a clock-initiated load
- [X] T012 [US3] Add `bg bool` and `bgDay string` fields to `planEntriesMsg` in `internal/tui/plan_update.go`, with `bgDay` recording the day the background load was dispatched for

### Cursor anchoring (all triggers, per research Decision 5)

- [X] T013 [P] [US3] Change the `listGoalsResultMsg` handler at `internal/tui/update.go:1084` to capture the currently selected goal ID before replacing `m.goal.goals` and re-anchor `m.goal.cursor` to that ID afterwards, falling back to `clampCursor` when the goal is gone — mirroring the Tasks handler at `internal/tui/update.go:848`
- [X] T014 [P] [US3] Change `handlePlanEntriesMsg` at `internal/tui/plan_update.go:531` to capture the selected entry ID before replacing `m.plan.entries` and re-anchor `m.plan.cursor` to that ID afterwards, falling back to `clampCursor`, while leaving the existing `msg.highlightID` path ahead of it unchanged

### Background apply semantics

- [X] T015 [US3] Add an early return `if msg.bg && msg.err != nil { return m, nil }` to the `listTasksResultMsg` and `reportResultMsg` handlers in `internal/tui/update.go` (FR-020, FR-021)
- [X] T016 [US3] Add the same early return to the `listGoalsResultMsg` handler in `internal/tui/update.go` and to `handlePlanEntriesMsg` in `internal/tui/plan_update.go`
- [X] T017 [US3] Stop clearing an existing error on a successful background load: in `listGoalsResultMsg` change the unconditional `m.goal.err = msg.err` to leave `m.goal.err` untouched when `msg.bg`, and apply the equivalent change to `m.plan.err` in `handlePlanEntriesMsg` and `m.reportData.err` in `reportResultMsg` (FR-018)
- [X] T018 [US3] Guard the Report scroll reset in `internal/tui/update.go:1064` so `m.reportData.scroll = 0` runs only when `!msg.bg` (FR-017, contract C3.8)
- [X] T019 [US3] Add an active-tab guard to each of the four handlers of the form `if msg.bg && m.activeTab != <that tab> { return m, nil }`, and extend the Plan guard with `msg.bgDay != m.plan.day` (FR-019, research Decision 4)
- [X] T020 [US3] Stamp the tab's `lastLoad` field with `m.nowOrDefault()` on successful load in all four handlers, regardless of trigger (FR-006)

### Tests

- [X] T021 [P] [US3] Create `internal/tui/autorefresh_apply_test.go` with tests asserting cursor identity is preserved across a load that inserts an item above the cursor, for all four tabs (contract A9)
- [X] T022 [P] [US3] Add tests asserting the cursor lands on a valid index without panic when the selected item is absent from the reloaded data, for all four tabs (contract A10)
- [X] T023 [P] [US3] Add tests asserting that a `bg: true` result carrying an error leaves the tab's data and `err` field untouched, and that a `bg: false` result carrying the same error sets `err` (contract A7, C4.1)
- [X] T024 [P] [US3] Add a test asserting a successful `bg: true` result does not clear a pre-existing error (contract A11)
- [X] T025 [P] [US3] Add a test asserting a successful `bg: true` `reportResultMsg` preserves `m.reportData.scroll` while a `bg: false` one resets it to 0 (contract A12)
- [X] T026 [P] [US3] Add a test asserting a `bg: true` `planEntriesMsg` whose `bgDay` differs from `m.plan.day` is discarded (contract A13)
- [X] T027 [P] [US3] Add a test asserting no load handler modifies `m.mode`, `m.goal.mode`, or `m.plan.mode`: set each to a non-list value, feed each result message, and assert the mode is unchanged (contract C3.1, FR-014)
- [X] T028 [P] [US3] Add a test asserting a load preserves `m.expanded`, `m.filterExpr`, and `m.listScroll` (FR-017)

**Checkpoint**: The load handlers are safe to call unprompted. Nothing calls them unprompted yet.

---

## Phase 5: User Story 2 — Changes made elsewhere appear without asking (Priority: P2)

**Goal**: A once-a-minute heartbeat reloads the active tab when its data is over ten minutes old.

**Independent test**: With injected time, feed `autoRefreshTickMsg` to a model whose active tab loaded 11 minutes ago and assert exactly one fetch is dispatched; repeat at 30 seconds and assert none.

**Depends on**: Phase 2 (the `lastLoad` fields) and Phase 4 (safe handlers).

### Implementation

- [X] T029 [US2] Add `autoRefreshInterval = 10 * time.Minute` and `autoRefreshTickRate = time.Minute` constants and the `autoRefreshTickMsg struct{}` type to `internal/tui/update.go`, with a comment recording that the interval is fixed by FR-009 and not configurable
- [X] T030 [US2] Add `autoRefreshTickCmd()` to `internal/tui/update.go` returning `tea.Tick(autoRefreshTickRate, …)`, following the shape of `pomTickCmd` in `internal/tui/pomodoro.go:48`
- [X] T031 [US2] Add `func (m Model) autoRefreshEligible() bool` to `internal/tui/update.go` returning true only when `m.mode == modeList`, `!m.confirmingQuit`, `!m.confirmingDiscard`, and the active tab's own mode is at rest (`m.goal.mode == goalList` for Goals, `m.plan.mode == planList` for Plan), with a comment explaining that keying off `m.mode` rather than enumerating modes means future modes suppress automatically
- [X] T032 [US2] Add `func (m Model) activeTabLastLoad() time.Time` to `internal/tui/update.go` returning the `lastLoad` field for `m.activeTab`
- [X] T033 [US2] Add the `autoRefreshTickMsg` case to `Update` in `internal/tui/update.go` implementing the state machine in `data-model.md`: reschedule unconditionally; return early when not eligible or not stale; otherwise stamp the active tab's `lastLoad` and dispatch that tab's fetch with `bg` set
- [X] T034 [US2] Add background-flavored dispatch for each tab's fetch in `internal/tui/update.go` and `internal/tui/plan_update.go` — a `bg bool` parameter on `listTasksCmd`, `listGoalsCmd`, and `fetchReportCmd`, and `bg`/`bgDay` on the plan fetch — updating the existing call sites to pass `false`
- [X] T035 [US2] Start the tick chain in `Init` at `internal/tui/update.go:816` by appending `autoRefreshTickCmd()` to the batch — the only start point besides the handler's own reschedule (FR-010)

### Tests

- [X] T036 [P] [US2] Create `internal/tui/autorefresh_test.go` with a test asserting a tick against a tab whose `lastLoad` is 11 minutes old dispatches exactly one fetch for that tab and none for the others (contract A4, FR-008)
- [X] T037 [P] [US2] Add a test asserting a tick against a tab whose `lastLoad` is 30 seconds old dispatches no fetch (contract A3)
- [X] T038 [P] [US2] Add a table-driven test asserting no fetch is dispatched when the active tab is in any non-list mode — `modeEdit`, `modeNewSubtask`, `modeNewRoot`, `modeHelp`, `modeMove`, `modeDatePrompt`, `modeFilter`, `goalPickLink`, `goalConfirmDelete`, `planPickTask`, `planEventForm`, `confirmingQuit`, `confirmingDiscard` — even when stale (contract A5, FR-012, SC-003)
- [X] T039 [P] [US2] Add a test asserting the same model dispatches a fetch on the next tick once the mode returns to list, confirming suppression is not sticky and nothing was queued (contract A6, FR-013)
- [X] T040 [P] [US2] Add a test asserting the tick handler returns a rescheduling command in every branch — eligible, ineligible, and not-stale (FR-010)
- [X] T041 [P] [US2] Add a test asserting `lastLoad` is stamped at dispatch, so a second tick immediately after a dispatched-but-unanswered refresh dispatches nothing (research Decision 3, contract C2.4)
- [X] T042 [P] [US2] Add a test simulating an hour of ticks against an unreachable server and asserting at most 6 fetches are dispatched (contract A8, SC-005)

**Checkpoint**: All three user stories are complete.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T043 Add a source-level guard test to `internal/tui/autorefresh_test.go` that reads the non-test `.go` files in `internal/tui` and asserts `autoRefreshTickCmd(` appears exactly twice, since a third call site permanently doubles the heartbeat rate and is invisible at runtime (`data-model.md` invariant, `quickstart.md` review checklist)
- [X] T044 Run `go test ./...` from the repo root and confirm the whole repo is green, including the pre-existing TUI suite
- [X] T045 Walk the hand-verification steps in `quickstart.md` against a live stack started with `make dev`, including the server-stopped case that confirms background failures are silent while `ctrl+r` still reports them
- [X] T046 Confirm no new user-facing string was introduced anywhere in the diff, satisfying FR-024 and leaving Principle IV with no new surface to review

---

## Dependencies

```text
T001 (baseline)
  │
  ├──────────────────────────────┐
  │                              │
  ▼                              ▼
Phase 2: T002 → T003        Phase 3 (US1): T004..T008 → T009, T010
  │                              │
  ▼                              │
Phase 4 (US3): T011, T012        │
  → T013, T014 [P]               │
  → T015..T020                   │
  → T021..T028 [P]               │
  │                              │
  ▼                              │
Phase 5 (US2): T029..T035        │
  → T036..T042 [P]               │
  │                              │
  └──────────────┬───────────────┘
                 ▼
Phase 6: T043 → T044 → T045 → T046
```

**Story dependencies**:

- **US1** is fully independent. It shares no code path with the other two and can be built, reviewed, and merged on its own.
- **US3** depends only on Phase 2.
- **US2** depends on Phase 2 and on US3. This is a release constraint, not just a build one: shipping the heartbeat without US3 exposes users to a moving cursor and spurious error banners.

**Within-phase ordering**: T011 and T012 add the message fields that T015–T020 branch on, so they come first. T013 and T014 touch different files and are independent of the message fields.

---

## Parallel Execution Opportunities

| Group | Tasks | Why parallel |
|---|---|---|
| US1 vs. Foundational | T004–T008 alongside T002–T003 | US1 touches only tab-entry paths; the timestamp fields are untouched by it |
| Cursor anchoring | T013, T014 | Different files (`update.go` vs. `plan_update.go`), no shared state |
| US3 tests | T021–T028 | All new assertions in one new test file, each independent of the others |
| US2 tests | T036–T042 | Same |

The implementation tasks T015–T020 and T029–T035 are **not** parallelizable — they repeatedly touch the same four handlers and the same `Update` switch in `update.go`.

---

## Implementation Strategy

**MVP**: Phase 1 + Phase 3 (US1). Eight tasks, two of them tests, no new state and no timer. This alone fixes the bug that motivated the feature — completing a task on the Plan tab and finding it still pending on the Tasks tab — and is safe to merge on its own.

**Increment 2**: Phase 2 + Phase 4 (US3). Invisible to users on its own, but it corrects real latent inconsistencies: the Goals and Plan tabs stop moving the cursor onto a different item when their lists change under a manual refresh.

**Increment 3**: Phase 5 (US2) + Phase 6. The heartbeat, landing on handlers that are already safe.

**Total**: 46 tasks — 25 implementation, 21 test and verification.
