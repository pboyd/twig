---

description: "Task list for Interactive TUI implementation"
---

# Tasks: Interactive TUI

**Input**: Design documents from `/specs/010-interactive-tui/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/keymap.md, contracts/view-model.md, quickstart.md

**Tests**: This project's existing convention is table-driven unit tests next to the code (`*_test.go`). Tests for pure-logic units (visible-row flattening, marker computation, highlight rules, edit-form lifecycle) are included as standard practice. Wire-level/server tests are NOT generated — the TUI calls existing ConnectRPC handlers that are already covered.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- All file paths are absolute or repo-relative; the Go module is at `services/todo/`.

## Path Conventions

- New TUI code: `services/todo/internal/tui/`
- Existing CLI helpers (modified to export): `services/todo/internal/cli/`
- Entry point wiring: `services/todo/internal/cli/cli.go`
- Module manifest: `services/todo/go.mod`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Pull in TUI dependencies and create the new package scaffold.

- [ ] T001 Add TUI dependencies to `services/todo/go.mod`: `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/lipgloss`. Run `go mod tidy` from `services/todo/`.
- [ ] T002 [P] Create the empty package skeleton at `services/todo/internal/tui/` with a placeholder `tui.go` declaring `package tui` and an empty exported `Run(ctx context.Context) error` function returning `nil`. This unblocks parallel work on sibling files.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Promote shared CLI helpers to exported names and add the TTY-gated entry point. Every user story depends on these.

- [ ] T003 In `services/todo/internal/cli/render.go`, export the helpers reused by the TUI: rename `buildTree` → `BuildTree`, `sortNodes` → `SortNodes` (called from within `BuildTree`; export only what the TUI calls directly), `formatDue` → `FormatDue`, `formatCompletedAt` → `FormatCompletedAt`, `dimStrike` → `DimStrike`, `wantStyled` → `WantStyled`, `buildPostID` → `BuildPostID`. Update all internal call sites in `services/todo/internal/cli/*.go` accordingly.
- [ ] T004 In `services/todo/internal/cli/task.go`, export `parseDue` → `ParseDue` and update all internal call sites. The TUI's edit form uses this for the due-date field per research R7.
- [ ] T005 Run `go build ./...` and `go test ./internal/cli/...` from `services/todo/` to verify the renames in T003–T004 didn't break the existing CLI.
- [ ] T006 [P] Create `services/todo/internal/tui/client.go` exposing `NewClients(addr, apiKey string)` that returns both a `TaskServiceClient` and a `PomodoroServiceClient` constructed the same way `runTask`/`runPomTop` already construct theirs (same `bearerInterceptor`, same `connect.WithSendGzip`). Read env vars `TODO_API_KEY` / `TODO_ADDR` with the same defaults as the existing CLI.
- [ ] T007 [P] Create `services/todo/internal/tui/keymap.go` with a `KeyMap` struct of `key.Binding` values for every entry in `specs/010-interactive-tui/contracts/keymap.md`. Provide a `DefaultKeyMap()` constructor. Implement `ShortHelp()` and `FullHelp()` so the Bubbles `help` component can render the help overlay automatically (research R6).
- [ ] T008 In `services/todo/internal/cli/cli.go`, modify `Run(args)`: when `len(args) == 0`, check `golang.org/x/term.IsTerminal(int(os.Stdout.Fd()))`. If true, delegate to `internal/tui.Run(context.Background())` and return its exit code (0 on success, 1 on error). If false, keep the existing `printRootUsage` + exit-1 behavior. Add an import for the `tui` package.

**Checkpoint**: The TUI launches (showing nothing yet) when `todo` is invoked with no args on a TTY; piped invocations still print the old usage banner.

---

## Phase 3: User Story 1 — Browse and view tasks (Priority: P1) 🎯 MVP

**Goal**: Launch the TUI, render the task tree with the same order/styling as `todo task list`, and let the user navigate / expand / collapse with a details pane that updates live.

**Independent Test**: Per spec User Story 1 acceptance scenarios — launch against a DB with hierarchical tasks; arrow keys move the highlight, `H`/`L` collapse/expand, details pane mirrors the highlighted task, completed tasks (when shown) appear dimmed with strikethrough.

### Tests for User Story 1

- [ ] T009 [P] [US1] In `services/todo/internal/tui/tree_test.go`, table-driven tests for visible-row flattening: given a tree + `expanded` map + `showCompleted` flag + `pendingComplete` pointer, assert the produced `[]*visibleRow` (length, task IDs in order, `depth`, `marker`, `treePrefix`). Cover collapsed subtree skipping, completed-task filtering, and the `[+]`/`[-]` marker rules from `contracts/view-model.md`.
- [ ] T010 [P] [US1] In `services/todo/internal/tui/update_test.go`, tests for cursor navigation messages: `Up`/`K` and `Down`/`J` clamp at boundaries and skip rows that are not present in `visible`. Use a fake `Model` populated via `export_test.go`.
- [ ] T011 [P] [US1] In `services/todo/internal/tui/tree_test.go`, tests for expansion: `H`/`Left` on a row sets `expanded[id]=false`; `L`/`Right` sets `expanded[id]=true`; collapsing the parent of the highlighted row does NOT change the highlight (the parent remains visible).

### Implementation for User Story 1

- [ ] T012 [US1] Create `services/todo/internal/tui/model.go` defining `Model`, `viewMode` constants (`modeList`, `modeEdit`, `modeNewSubtask`, `modeNewRoot`, `modeHelp`, `modePomodoro`), and the `Init() tea.Cmd` that fires an initial `ListTasks` command. Fields match `data-model.md`.
- [ ] T013 [P] [US1] Create `services/todo/internal/tui/tree.go`: define `treeNode`/`visibleRow` (or alias to `cli`'s `treeNode`), implement `buildVisible(tree, expanded, showCompleted, pendingComplete) []*visibleRow` per `contracts/view-model.md` "Visible-row flattening" section. Implement marker (`[+]`/`[-]`) and tree-prefix computation matching `internal/cli.renderTree`'s `├──`/`└──`/`│  `/`   ` rules.
- [ ] T014 [P] [US1] Create `services/todo/internal/tui/details.go` with `renderDetails(task *taskv1.Task, width int) string` that formats id, name (using `cli.DimStrike` when completed), due (via `cli.FormatDue`), pomodoro estimate, description (wrapped to `width`), and `cli.FormatCompletedAt` when present.
- [ ] T015 [US1] Create `services/todo/internal/tui/update.go` with `Model.Update(msg tea.Msg) (tea.Model, tea.Cmd)` handling at minimum: `tea.WindowSizeMsg` (store `width`/`height`), `tea.KeyMsg` dispatched through `KeyMap` for `Up/K/Down/J/Left/H/Right/L`, and `listTasksResultMsg` to populate `Model.tree`. Recompute `Model.visible` after any state change that affects it.
- [ ] T016 [US1] Create `services/todo/internal/tui/view.go` (or extend `model.go`) with `Model.View() string` rendering a two-pane layout using Lipgloss: left pane shows `visible` with the highlighted row inverted; right pane shows `renderDetails(visible[cursor].node.task, rightWidth)`. Handle empty `visible` (placeholder text).
- [ ] T017 [US1] Replace the placeholder `tui.Run` in `services/todo/internal/tui/tui.go` with a real entry point: construct clients via `NewClients`, build an initial `Model`, run `tea.NewProgram(model, tea.WithAltScreen()).Run()`, return any error.
- [ ] T018 [US1] Create `services/todo/internal/tui/export_test.go` exposing constructors for `Model` and `visibleRow` and a way to inject a fake `tree` for tests (mirrors the project's `internal/cli/export_test.go` pattern).
- [ ] T019 [US1] In `services/todo/internal/tui/update.go`, implement initial `ListTasks` flow: `Init()` returns a `tea.Cmd` that calls `TaskServiceClient.ListTasks`, builds the tree via `cli.BuildTree`, and dispatches a `listTasksResultMsg`. On result, set `cursor=0`, all `expanded` empty, `showCompleted=false`. On error, set `Model.err`.

**Checkpoint**: User Story 1 is fully functional — running `todo` shows the tree, the user can navigate, expand, collapse, and see details. The MVP slice is complete.

---

## Phase 4: User Story 2 — Edit, create, delete, complete tasks (Priority: P1)

**Goal**: Allow all mutations from inside the TUI: edit (`E`), new subtask (`N`), new root (`Ctrl-N`), delete (`Ctrl-D`), toggle completion (`Space`), and set pomodoro estimate via digit keys.

**Independent Test**: Per spec User Story 2 — exercise each mutation and confirm: (a) it persists to the server (visible after `Ctrl-R` or relaunch), (b) the highlight lands on the task specified in `contracts/keymap.md` "Highlight rules after mutations".

### Tests for User Story 2

- [ ] T020 [P] [US2] In `services/todo/internal/tui/edit_test.go`, tests for `editFormModel` lifecycle: opening with an existing task pre-fills all fields; opening "new subtask" leaves fields blank but sets `parentID`; `Tab`/`Shift-Tab` cycles focus including Save/Cancel buttons; `Esc` and `Ctrl-S` global shortcuts behave correctly regardless of focused field.
- [ ] T021 [P] [US2] In `services/todo/internal/tui/update_test.go`, tests for highlight rules from `contracts/keymap.md`: after `E` Save → same task; after `N` Save → new subtask, parent expanded; after `Ctrl-N` Save → new root highlighted; after `Ctrl-D` → next visible (or previous if last); after `Space` with completed-only-filter → `pendingComplete` set and cleared on next cursor move; after `0`-`9` → same task.

### Implementation for User Story 2

- [ ] T022 [P] [US2] Create `services/todo/internal/tui/edit.go` defining `editFormModel` per `data-model.md`. Use `textinput.Model` for name, due, pomodoro-estimate; `textarea.Model` for description. Implement `Init`, `Update`, `View`, plus helpers `NewEditForm(task *taskv1.Task)`, `NewSubtaskForm(parentID int64)`, `NewRootForm()`. Validate due date via `cli.ParseDue` on save; on parse failure return a validation error to be surfaced in `Model.err`.
- [ ] T023 [US2] In `services/todo/internal/tui/update.go`, handle `E`, `N`, `Ctrl-N` keys: transition to the matching `viewMode`, initialize `Model.edit`, save `originalCursor` so cancel can restore it for new-form flows.
- [ ] T024 [US2] In `services/todo/internal/tui/update.go`, handle edit-form completion messages: `editSavedMsg{form, result}` and `editCancelledMsg{}`. On save for existing task → call `UpdateTask`, on save for new subtask/root → call `CreateTask`, then re-fetch via `ListTasks` and apply highlight per the contract (same task, or new task id, or original cursor on cancel). On error, keep mode but populate `Model.err`.
- [ ] T025 [US2] In `services/todo/internal/tui/update.go`, handle `Ctrl-D`: call `DeleteTask` on the highlighted task; on response, re-fetch `ListTasks`; recompute `visible`; move cursor to next visible (or previous if at end; or 0 if list empty).
- [ ] T026 [US2] In `services/todo/internal/tui/update.go`, handle `Space`: call the existing completion-toggle RPC on the highlighted task. After re-fetch, if `showCompleted == false` and the task is now complete, set `pendingComplete = &taskID` so the task stays visible until cursor moves; clear `pendingComplete` on the next `Up`/`Down` message.
- [ ] T027 [US2] In `services/todo/internal/tui/update.go`, handle digit keys `0`-`9`: call `UpdateTask` with `pomodoroEstimate` set to the digit value; re-fetch and preserve the highlight.
- [ ] T028 [US2] In `services/todo/internal/tui/view.go`, extend `View()` to render the edit form (when `mode` is `modeEdit`/`modeNewSubtask`/`modeNewRoot`) in the right pane, replacing the details view. Show validation errors from `Model.err` in a status line below the form.

**Checkpoint**: User Story 2 fully functional — all CRUD and completion operations work from the TUI with correct post-action highlight behavior.

---

## Phase 5: User Story 3 — Pomodoro integration (Priority: P2)

**Goal**: `S` starts a pomodoro on the highlighted task using existing CLI flow and returns to the list on completion; `0`-`9` already covered in US2; `R` resumes a backgrounded pomodoro.

**Independent Test**: Per spec User Story 3 — press `S`, run the pomodoro to completion, confirm the TUI returns to the list (not the shell). Background a pomodoro from outside, press `R`, confirm it resumes inside the TUI.

### Tests for User Story 3

- [ ] T029 [P] [US3] In `services/todo/internal/tui/update_test.go`, tests asserting that `S` dispatches the correct command (verify the `tea.Cmd` returns a `pomodoroRequestMsg` carrying the highlighted task id) and that `pomodoroDoneMsg` restores `mode = modeList` with the same cursor.

### Implementation for User Story 3

- [ ] T030 [US3] In `services/todo/internal/cli/pom.go`, refactor `runPomStart` and `runPomResume` per research R4: extract the core (no `os.Exit`, no `fmt.Println` directly to stderr for fatal-only errors) into `RunPomodoroForTask(ctx, taskID int64) error` and `ResumeBackgroundedPomodoro(ctx) error`. The existing CLI wrappers translate the returned `error` to an exit code.
- [ ] T031 [US3] In `services/todo/internal/cli/pom.go`, run `go test ./internal/cli/...` from `services/todo/` to confirm the refactor preserves CLI behavior.
- [ ] T032 [US3] Create `services/todo/internal/tui/pomodoro.go` with two `tea.Cmd` factories: `startPomodoroCmd(taskID int64)` and `resumePomodoroCmd()`. Both use `tea.ExecProcess`-equivalent (or `tea.Exec` with a custom `tea.ExecCommand`) to yield the terminal, call `cli.RunPomodoroForTask` / `cli.ResumeBackgroundedPomodoro`, then dispatch `pomodoroDoneMsg{err}`. The TUI re-enters the alt-screen automatically when the command returns.
- [ ] T033 [US3] In `services/todo/internal/tui/update.go`, wire `S` and `R` keys to dispatch the commands from T032 and set `mode = modePomodoro`. On `pomodoroDoneMsg`, set `mode = modeList`, surface error if any, and trigger a `ListTasks` refresh (pomodoro may have updated counters).

**Checkpoint**: User Story 3 fully functional — pomodoros run within the TUI session.

---

## Phase 6: User Story 4 — Filter, help, exit, manual refresh (Priority: P2)

**Goal**: `C` toggles completed-task visibility, `?` opens/dismisses help, `Q` exits, `Ctrl-R` performs a full reload.

**Independent Test**: Per spec User Story 4 — press `C` and watch completed tasks appear with dim/strike styling and disappear when toggled off; press `?` and confirm every keybinding is listed; press `Q` and confirm clean exit.

### Tests for User Story 4

- [ ] T034 [P] [US4] In `services/todo/internal/tui/update_test.go`, tests for `C` filter toggle: highlight preserved when still visible, falls back to first visible task when not (per `contracts/keymap.md`).
- [ ] T035 [P] [US4] In `services/todo/internal/tui/update_test.go`, tests for `Ctrl-R`: triggers a `ListTasks` command; on result, cursor lands on the same task id when it still exists; otherwise on first visible row.

### Implementation for User Story 4

- [ ] T036 [P] [US4] Create `services/todo/internal/tui/help.go` with a `helpModel` wrapping `bubbles/help.New()` and the `KeyMap` from T007. Provide `View(width int) string` rendering the full keybinding list. Show on `?`, dismiss on `?` or `Esc`.
- [ ] T037 [US4] In `services/todo/internal/tui/update.go`, handle `C`: toggle `showCompleted`, recompute `visible`, preserve highlight by task id when possible (otherwise fall back to first visible).
- [ ] T038 [US4] In `services/todo/internal/tui/update.go`, handle `Q`: only when `mode == modeList`, return `tea.Quit`.
- [ ] T039 [US4] In `services/todo/internal/tui/update.go`, handle `Ctrl-R`: dispatch a `ListTasks` command; in the result handler, attempt to restore cursor by `task.id`; on miss, set cursor to 0.
- [ ] T040 [US4] In `services/todo/internal/tui/update.go`, handle `?` (toggle help): set `mode = modeHelp`; the existing `Esc`/`?` handling for `modeHelp` returns to `modeList` with the original cursor.
- [ ] T041 [US4] In `services/todo/internal/tui/view.go`, render the help overlay when `mode == modeHelp` (full-screen, replaces the two-pane layout).

**Checkpoint**: All four user stories complete. The TUI fully matches the spec and the keymap/view-model contracts.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T042 Manual smoke test: follow every step of `specs/010-interactive-tui/quickstart.md` against a real `make dev` environment with a provisioned user. Document any deviations and fix them.
- [ ] T043 [P] Run `go vet ./...` and `gofmt -l` from `services/todo/`; fix any issues.
- [ ] T044 [P] Run the full test suite `go test ./...` from `services/todo/`; all packages green.
- [ ] T045 Update `services/todo/CLAUDE.md` (the project-root CLAUDE.md) with a one-line note that running `todo` with no arguments launches the TUI on a TTY. Only one line — no further docs.

---

## Dependencies

```text
Phase 1 (T001–T002)
  └── Phase 2 (T003–T008)
        ├── Phase 3 / US1 (T009–T019)   ← MVP
        ├── Phase 4 / US2 (T020–T028)   ← depends on US1's tree/Model
        ├── Phase 5 / US3 (T029–T033)   ← depends on US1
        ├── Phase 6 / US4 (T034–T041)   ← depends on US1
        └── Phase 7 polish (T042–T045)  ← after all stories
```

**Story dependencies**:
- US1 must complete before US2/US3/US4 (they all extend the same `Update`/`View` dispatch).
- US2, US3, US4 are independent of each other and can be implemented in parallel by different developers.

## Parallel execution opportunities

Within Phase 2 (foundational): T006 and T007 can be done in parallel after T003–T005 land.

Within US1: T009, T010, T011 (tests) are parallel; T013 and T014 (tree.go and details.go) are parallel; T012 (model.go) gates T015/T017.

Within US2: T020 and T021 (tests) parallel; T022 (edit.go) can start in parallel with US2 update-handlers once T012 is in.

Across stories: after the Phase 2 checkpoint, US2/US3/US4 can all proceed in parallel (different files: `edit.go`, `pomodoro.go`, `help.go` respectively; they share `update.go` so serialize the `update.go` edits or coordinate via small commits).

Polish phase: T043 and T044 parallel.

## Independent test criteria per story

- **US1**: `todo` (with no args, on a TTY, against a populated DB) launches the TUI; arrow keys + H/L navigate and expand/collapse; details pane updates; completed tasks (when shown) appear dimmed/struck-through. Verified via the unit tests in T009–T011 plus a manual quickstart walk.
- **US2**: From the TUI, edit a task and save → change persists. Create a subtask, root task, delete a task, toggle completion, set estimate via digit. Highlight lands per `contracts/keymap.md`. Verified via T020–T021 plus manual quickstart steps.
- **US3**: `S` starts a pomodoro for the highlighted task; on completion the TUI is still running. `R` resumes a backgrounded pomodoro. Verified via T029 plus manual interaction.
- **US4**: `C` toggles filter; `?`/`Esc` open and dismiss help; `Q` exits cleanly; `Ctrl-R` reloads from server. Verified via T034–T035 plus manual quickstart.

## Implementation strategy

1. **Land the foundation (T001–T008)** in one branch / one batch — the entry-point wiring and renamed helpers are the only changes that touch the existing CLI; landing them together minimizes the time the existing CLI is in a half-renamed state.
2. **Ship US1 alone as the MVP** — `todo` with no args launches a navigable, view-only TUI. This is releasable on its own and demonstrates the UX commitment.
3. **Add US2** next — it's the second P1 and turns the TUI into a daily-driver replacement.
4. **Add US3 and US4** in either order (or in parallel) — both are P2 and unblocked once US1 is in.
5. **Polish (T042–T045)** before merge.
