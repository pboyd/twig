---
description: "Tasks for the TUI Pomodoro Integration feature"
---

# Tasks: TUI Pomodoro Integration

**Input**: Design documents in `/specs/019-tui-pomodoro-integration/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/pomodoro-rpcs.md`, `quickstart.md`

**Tests**: Included. The TUI in `services/todo/internal/tui/` is exercised via Go unit tests (see `update_test.go`, `view_test.go`), driving `Model.Update` directly with synthetic messages and an injected clock plus a fake `TaskServiceClient`. This feature follows the same convention.

**Organization**: Tasks are grouped by user story so each story is independently testable.

## Format

`- [ ] [TaskID] [P?] [Story?] Description`

- **[P]**: parallelizable (different file, no dependency on incomplete tasks)
- **[US1]–[US4]**: user-story tag (see `spec.md`)
- Setup, Foundational, and Polish tasks have no story tag

## Path Conventions

All code paths are under `services/todo/internal/tui/`. No proto, handler, db, or server changes. The CLI `internal/cli/pom.go` is left untouched (FR-016).

---

## Phase 1: Setup (Shared Infrastructure)

No new dependencies, build steps, or scaffolding are required. Skipping.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Remove the old `tea.Exec` takeover, add the background `activePom` state, the per-second tick infrastructure, and the terminal-safe hook runner that every story depends on. This phase leaves the package compiling and `go test ./...` green, with no user-visible behavior yet.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T001 In `services/todo/internal/tui/model.go`, add the `activePom` struct (`taskID int64`, `taskName string`, `startAt time.Time`, `completed bool`, `banner string`) per `data-model.md`; add `pom *activePom` and `confirmingQuit bool` fields to `Model`; remove the `modePomodoro` value from the `viewMode` enum.
- [ ] T002 Rewrite `services/todo/internal/tui/pomodoro.go`: delete the `tea.Exec` machinery (`funcExecCommand`, `execPomodoroStart`, `execPomodoroResume`, `startPomodoroCmd`, `resumePomodoroCmd`, `pomodoroRequestMsg`, `pomodoroDoneMsg`). Define the new message types (`pomTickMsg`, `pomStartedMsg`, `pomActiveMsg`, `pomCancelledMsg`, `pomCompletedMsg`, `pomHookErrMsg`, `pomBannerExpireMsg`) per `data-model.md`, and add `pomTickCmd() tea.Cmd` returning a one-second `tea.Tick` that emits `pomTickMsg`.
- [ ] T003 In `services/todo/internal/tui/pomodoro.go`, implement the terminal-safe hook runner `runPomHook(cmd, name string) tea.Cmd`: returns nil for an empty command; otherwise runs `sh -c <cmd>` with stdio **detached** (not `os.Stdin/Stdout/Stderr`) inside the `tea.Cmd` (off the UI goroutine), returning `pomHookErrMsg` on non-zero exit and nil on success (R6 / FR-013).
- [ ] T004 In `services/todo/internal/tui/pomodoro.go`, add a small unexported `activeTaskIDFromErr(*connect.Error) int64` helper that reads the active task id from the `AlreadyExists` error detail (mirroring the logic of `cli.extractActiveTaskID`, which is unexported in another package) for use by the start-conflict policy.
- [ ] T005 In `services/todo/internal/tui/keymap.go`, replace the `PomResume` binding (`r`) with a `PomCancel` binding (`x`, help "cancel pomodoro"); keep `PomStart` (`s`); update `ShortHelp`/`FullHelp` groupings to drop resume and include cancel.
- [ ] T006 In `services/todo/internal/tui/update.go`, remove the `pomodoroRequestMsg` and `pomodoroDoneMsg` cases and the `PomResume` key branch; add a `pomTickMsg` handler that recomputes remaining via `pomodoro.Remaining(m.pom.startAt, now)`, reschedules the next tick **only while** `m.pom != nil && !m.pom.completed`, and ignores the tick otherwise (R3 / FR-014). Temporarily route `s`/`x` to no-ops if needed so the package compiles (wired in US1).
- [ ] T007 Update `services/todo/internal/tui/update_test.go` to remove the obsolete tests that reference the deleted messages/modes (`pomodoroRequestMsg`, `pomodoroDoneMsg`, `modePomodoro` — currently around lines 603–674), so the package builds and existing tests pass.

**Checkpoint**: `cd services/todo && go test ./... && go build -o todo ./cmd/todo` pass; tick infra, hook runner, and `activePom` state exist with no visible timer yet.

---

## Phase 3: User Story 1 — Run a pomodoro without leaving the task view (Priority: P1) 🎯 MVP

**Goal**: Press `s` on a selected task to start a pomodoro; a live `🍅 mm:ss · <task>` timer appears in the status bar and counts down while the task list stays fully interactive; press `x` to cancel and clear it.

**Independent Test**: Quickstart S1 + S2 — start a pomodoro, confirm the two-line status bar with a ticking timer while navigating/editing, exercise same-task/different-task start, then cancel and confirm the timer clears.

### Implementation for User Story 1

- [ ] T008 [US1] In `services/todo/internal/tui/pomodoro.go`, implement `startPomCmd(client, taskID) tea.Cmd`: call `StartPomodoro`; on `CodeAlreadyExists` use `activeTaskIDFromErr` to apply the non-interactive conflict policy — same task ⇒ attach to the running pomodoro (fetch its `start_at` via `GetActivePomodoro`), different task ⇒ `CancelPomodoro` then `StartPomodoro`; return `pomStartedMsg{taskID, startAt, err}` (R4 / FR-010).
- [ ] T009 [US1] In `services/todo/internal/tui/pomodoro.go`, implement `cancelPomCmd(client) tea.Cmd`: call `CancelPomodoro` and return `pomCancelledMsg{err}` (the `on_cancel` hook is fired from the Update handler).
- [ ] T010 [US1] In `services/todo/internal/tui/update.go`, wire `case key.Matches(msg, m.keys.PomStart)` (guarded by a non-empty selection) to `startPomCmd`, and `case key.Matches(msg, m.keys.PomCancel)` to `cancelPomCmd`. Handle `pomStartedMsg` (on success set `m.pom` with the selected task's cached name and `startAt`, fire `on_start` via `runPomHook`, start `pomTickCmd`; on error set `m.err`) and `pomCancelledMsg` (clear `m.pom`, fire `on_cancel`; on error set `m.err`).
- [ ] T011 [US1] In `services/todo/internal/tui/view.go`, add `statusHeight() int` (returns 2 while `m.pom != nil`, else 1) and use it in the pane height math of `viewList`, `viewWithForm`, and `viewWithMove` so panes shrink by one row instead of overflowing. Extend `renderStatus` to render a pomodoro line (`🍅 mm:ss · <task>  [x] cancel`) above the existing help/error line while `m.pom != nil && !m.pom.completed`, gating emoji/color on `m.styled` (R7 / FR-002, FR-004). The full-screen help view (`viewHelp`) is intentionally left untouched — per FR-002 the timer is exempt there.
- [ ] T012 [P] [US1] Create `services/todo/internal/tui/pomodoro_test.go` with tests using a fake `TaskServiceClient` and injected clock: `startPomCmd` success returns `pomStartedMsg` with the server `startAt`; same-task `AlreadyExists` attaches (no restart); different-task `AlreadyExists` issues cancel+start; `cancelPomCmd` returns `pomCancelledMsg`; a `pomTickMsg` recomputes remaining and reschedules only while active; a stale `pomTickMsg` with `m.pom == nil` is ignored. **Error paths (FR-015)**: a `pomStartedMsg`/`pomCancelledMsg` carrying an RPC error sets `m.err`, leaves the model usable (mode unchanged), and does not leave a half-set `m.pom`.
- [ ] T013 [P] [US1] Extend `services/todo/internal/tui/view_test.go`: while a pomodoro is active the status area is two lines and shows the correct `mm:ss` and task name; idle status area is one line; the timer line is present across `modeList`/`modeEdit`/`modeMove`; in unstyled mode the timer renders as plain text (no ANSI escapes).

**Checkpoint**: A user can start a pomodoro, watch it tick in the status bar while using the TUI normally, and cancel it. MVP is shippable.

---

## Phase 4: User Story 2 — Automatic completion with notification (Priority: P2)

**Goal**: When the timer reaches zero while the TUI is open, the pomodoro is recorded complete, the `on_complete` hook fires once, and a brief non-blocking "🍅 Pomodoro complete! · <task>" banner appears, clearing on the next keypress or after ~5s.

**Independent Test**: Quickstart S3 + S6 — advance the injected clock past the pomodoro length, confirm exactly-once completion, the banner appears and clears, and a failing hook surfaces as a status error without aborting completion.

### Implementation for User Story 2

- [ ] T014 [US2] In `services/todo/internal/tui/pomodoro.go`, implement `completePomCmd(client) tea.Cmd` (call `CompletePomodoro`, return `pomCompletedMsg{taskName, err}`) and `pomBannerExpireCmd() tea.Cmd` (a ~5s `tea.Tick` emitting `pomBannerExpireMsg`).
- [ ] T015 [US2] In `services/todo/internal/tui/update.go`, in the `pomTickMsg` handler, when `remaining == 0 && !m.pom.completed`, set `m.pom.completed = true` (stops further ticks per T006) and dispatch `completePomCmd`. Handle `pomCompletedMsg` (fire `on_complete` via `runPomHook`, set `m.pom.banner`, schedule `pomBannerExpireCmd`; on error set `m.err` but keep the pomodoro completed), `pomBannerExpireMsg` (clear `m.pom`), and `pomHookErrMsg` (set `m.err`). On the next key press, clear the banner / `m.pom` **as a side effect while still performing that key's normal action** — the keystroke is not swallowed (FR-007, F2). Hooks fire off the UI goroutine (FR-013, R2).
- [ ] T016 [US2] In `services/todo/internal/tui/view.go`, render `m.pom.banner` in the status-area pomodoro line while `m.pom.completed`, returning to a one-line status area once the banner is cleared.
- [ ] T017 [P] [US2] Extend `services/todo/internal/tui/pomodoro_test.go` / `update_test.go`: a tick crossing zero triggers completion exactly once (a subsequent tick does not re-fire); `pomCompletedMsg` sets the banner; the banner clears on keypress (and that keypress still performs its normal action, per F2) and on `pomBannerExpireMsg`; a hook failure (`pomHookErrMsg`) sets `m.err` while the pomodoro stays completed. **Error path (FR-015)**: a `pomCompletedMsg` carrying a `CompletePomodoro` RPC error sets `m.err` while still clearing the running timer and leaving the TUI usable.

**Checkpoint**: Pomodoros complete automatically with the hook firing and a transient banner; Stories 1 and 2 both work.

---

## Phase 5: User Story 3 — Pick up an already-running pomodoro at launch (Priority: P2)

**Goal**: On launch, the TUI detects an active pomodoro (e.g. started from the CLI) and immediately shows it ticking with the correct remaining time and task name.

**Independent Test**: Quickstart S4 — with the fake client reporting an active pomodoro, confirm the timer appears immediately with the right derived remaining time and task; with none, the status bar starts idle.

### Implementation for User Story 3

- [ ] T018 [US3] In `services/todo/internal/tui/pomodoro.go`, implement `getActivePomCmd(client) tea.Cmd`: call `GetActivePomodoro`; if a pomodoro is returned, resolve its task name (from the loaded tree if present, else `GetTask`) and return `pomActiveMsg{pom: &activePom{...}}`; otherwise return `pomActiveMsg{pom: nil}`.
- [ ] T019 [US3] In `services/todo/internal/tui/update.go`, change `Init` to `tea.Batch(listTasksCmd(m.client), getActivePomCmd(m.client))`; handle `pomActiveMsg` by seeding `m.pom` and starting `pomTickCmd` when present, leaving the status bar idle when nil (R5 / FR-009).
- [ ] T020 [P] [US3] Extend `services/todo/internal/tui/update_test.go`: launching with an active pomodoro seeds `m.pom` with the correct `startAt`/derived remaining/`taskName` and starts ticking; launching with no active pomodoro leaves `m.pom == nil`.

**Checkpoint**: A pre-existing pomodoro is shown automatically at launch; Stories 1–3 work.

---

## Phase 6: User Story 4 — Quit safely with a pomodoro running (Priority: P3)

**Goal**: Pressing `q` while a pomodoro is actively counting down asks for confirmation; confirming exits (pomodoro keeps running server-side), declining returns to the list; quitting with no/finished pomodoro exits immediately.

**Independent Test**: Quickstart S5 — `q` with an active pomodoro shows the confirm overlay; `y` quits, `n`/`esc` dismisses; `q` with no active or an already-completed pomodoro quits immediately.

### Implementation for User Story 4

- [ ] T021 [US4] In `services/todo/internal/tui/update.go`, intercept the `Quit` key in `handleListKey`: if `m.pom != nil && !m.pom.completed`, set `m.confirmingQuit = true` instead of returning `tea.Quit`; otherwise quit immediately (including when a completion banner is still showing — `q` quits and the banner clearing is irrelevant, per F2). While `m.confirmingQuit`, handle `y` → `tea.Quit`, `n`/`esc` → clear `confirmingQuit` (R8 / FR-011, FR-012).
- [ ] T022 [US4] In `services/todo/internal/tui/view.go`, render the one-key quit-confirm overlay (`🍅 mm:ss still running. Quit anyway? [y]es [n]o`) while `m.confirmingQuit`.
- [ ] T023 [P] [US4] Extend `services/todo/internal/tui/update_test.go`: `q` with an active pomodoro sets `confirmingQuit` and does not quit; `y` returns `tea.Quit`; `n`/`esc` clears `confirmingQuit`; `q` with no pomodoro and `q` after completion both return `tea.Quit` immediately.

**Checkpoint**: All four user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T024 Update `services/todo/internal/tui/help.go` so the `?` help screen reflects the new keys (`s` start, `x` cancel) and no longer mentions resume.
- [ ] T025 Run `cd services/todo && go test ./...` and confirm all tests pass.
- [ ] T026 Run `cd services/todo && go build -o todo ./cmd/todo` and execute Scenarios S1–S7 in `specs/019-tui-pomodoro-integration/quickstart.md` against `make dev`, including the terminal-safe-hook and plain-terminal checks; confirm `todo pom start|resume|cancel|status` still behave as before (FR-016).

---

## Dependencies

```
Phase 2 Foundational (T001 → T002 → T003, T004 [P], T005 [P]; then T006 → T007)
   │
   ├── Phase 3 US1 (T008 [P], T009 [P] → T010 → T011; T012, T013 [P])      🎯 MVP
   │      │
   │      ├── Phase 4 US2 (T014 → T015 → T016; T017 [P])
   │      ├── Phase 5 US3 (T018 → T019; T020 [P])
   │      └── Phase 6 US4 (T021 → T022; T023 [P])
   │
   └── Phase 7 Polish (after desired stories): T024 → T025 → T026
```

- **US1** depends only on Foundational and is the MVP.
- **US2, US3, US4** each depend on Foundational and build on the US1 status-bar/tick plumbing; they are independent of one another and can proceed in any order (or in parallel) after US1.

## Parallel Opportunities

- Foundational: T004 and T005 touch different files and can run in parallel after T002; T003 is independent of T004/T005.
- Within each story, the test task ([P]) can run alongside the remaining implementation once the code under test exists (separate `*_test.go` files).
- After US1 lands, US2 / US3 / US4 can be developed in parallel by different people (they touch overlapping files — `update.go`, `view.go` — so coordinate edits, but the logic is independent).

## MVP Scope

**User Story 1 (Phase 2 + Phase 3)** is the MVP: a pomodoro runs in the background with a visible ticking timer while the TUI stays fully usable, and can be cancelled — the headline capability of this feature.

## Task Count Summary

- Phase 2 (Foundational): 7 tasks (T001–T007)
- Phase 3 (US1, P1): 6 tasks (T008–T013)
- Phase 4 (US2, P2): 4 tasks (T014–T017)
- Phase 5 (US3, P2): 3 tasks (T018–T020)
- Phase 6 (US4, P3): 3 tasks (T021–T023)
- Phase 7 (Polish): 3 tasks (T024–T026)
- **Total: 26 tasks**
