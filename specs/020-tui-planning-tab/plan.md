# Implementation Plan: TUI Planning Tab

**Branch**: `020-tui-planning-tab` | **Date**: 2026-05-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/020-tui-planning-tab/spec.md`

## Summary

Bring the CLI day-planner into the interactive TUI as a second tab. Today the TUI (`internal/tui`) is a single-screen task tree that talks only to `TaskService`; the day-planner lives entirely in the CLI (`internal/cli/plan.go` + `plan_grid.go`) and talks to `PlanService`.

The design (Approach A) extends the existing Bubble Tea `Model` with an `activeTab` field (`tabTasks` / `tabPlanning`), a `planState` struct holding the planning view's own state (in-view day, fetched entries, selection cursor, sub-mode, and prompt inputs), and a `PlanServiceClient`. A one-line tab bar renders above whichever tab's content fills the screen; the shared status bar (pomodoro, help, errors) and quit-guard already render below both and are reused unchanged. `Update`/`View` branch on `m.activeTab` first, then delegate to the existing task logic or to new planning logic in new files (`plan_update.go`, `plan_view.go`, `plan_grid.go`).

The grid is rendered by **parametrizing the existing `cli.RenderGrid`** with an options struct (hide the `[id]` prefix; mark one entry as selected for highlight) rather than duplicating ~270 lines of box-drawing — keeping a single source of truth so the TUI grid stays visually identical to the CLI (SC-003). Planning actions map 1:1 to the existing `PlanService` RPCs (`ListPlanEntries`, `AddPlanTask`, `AddPlanEvent`, `RenamePlanEntry`, `MovePlanEntry`, `RemovePlanEntry`, `ClearPlan`). Time/duration parsing reuses `internal/cli/timeparse`; the task picker reuses `cli.BuildTree`. A planning-scoped `tea.Tick` (live only while the Planning tab is active on today) advances the now-marker without user input (FR-008a). This is a purely client-side change confined to `services/todo/internal/tui/` plus one parametrization of `internal/cli/plan_grid.go` — no proto, handler, db, or migration work.

## Technical Context

**Language/Version**: Go 1.22+ (existing module at `services/todo/`).

**Primary Dependencies**: Bubble Tea (`github.com/charmbracelet/bubbletea`) for the event loop and `tea.Tick`; Bubbles `textinput` (for prompts, as in the existing edit form) and `help`; Lipgloss for the tab bar and selection highlight; existing ConnectRPC clients (`TaskServiceClient`, plus a new `PlanServiceClient` built the same way). No new third-party dependencies.

**Storage**: N/A — plan data is server-authoritative via existing `PlanService` RPCs; the TUI holds only a transient in-memory copy of the fetched entries for the in-view day.

**Testing**: `go test ./...` from `services/todo/`. Reducer/transition unit tests in `internal/tui/` following the existing `update_test.go` / `view_test.go` style: drive `Model.Update` with synthetic messages, inject `now` for the now-marker, use fake `PlanServiceClient` + `TaskServiceClient`, and the `export_test.go` shim. Grid-rendering tests for the parametrized `cli.RenderGrid` extend `internal/cli/plan_grid_test.go`. No DB or running-server dependency.

**Target Platform**: TTY (Linux/macOS terminals); alt-screen Bubble Tea program.

**Project Type**: CLI / TUI client (single Go binary at `cmd/todo`).

**Performance Goals**: Tab switch and grid re-render are instantaneous (in-memory). Entry list reloads on tab activation and after each mutation cost one `ListPlanEntries` call (FR-024). The now-marker stays within one tick of wall-clock while the Planning tab is open (FR-008a).

**Constraints**: Keyboard-only; no new dependencies; no server changes; must render legibly in unstyled (non-TTY) output as the existing TUI does; the planning tick must not run when the Planning tab is inactive or when viewing a day other than today.

**Scale/Scope**: One day in view at a time; a day holds a handful to a few dozen entries. Change is confined to ~7 files under `internal/tui/` plus one parametrized function in `internal/cli/plan_grid.go`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Simplicity / YAGNI**: PASS. Reuses existing building blocks instead of adding abstractions: `cli.RenderGrid` (parametrized, not duplicated), `cli.BuildTree` for the picker, `internal/cli/timeparse` for inputs, Bubbles `textinput` for prompts, and the already-shared status-bar/quit-guard rendering. The one structural addition (an `activeTab` enum + `planState` + tab-aware `Update`/`View` dispatch) is the irreducible core of the feature, not speculative generalization. Tabs are modeled as a two-value enum, not a general tab framework. Out-of-scope ideas (week view, drag-to-reschedule, complete-toggle, scheduling-conflict logic) are explicitly excluded in the spec's Assumptions.
- **II. API-First Design**: PASS. No contract changes and no `make proto`. The feature consumes existing, already-implemented `PlanService` RPCs defined in `services/todo/proto/plan/v1/plan.proto`; `contracts/plan-rpcs.md` documents the reused contract and the server-side defaults the TUI relies on (e.g. `duration_minute = 0` ⇒ server-chosen default). The new `PlanServiceClient` is constructed exactly like the existing `TaskServiceClient` in `internal/tui/client.go`.

No violations. Complexity Tracking table is unused.

## Project Structure

### Documentation (this feature)

```text
specs/020-tui-planning-tab/
├── spec.md                  # Feature spec (already created + clarified)
├── plan.md                  # This file
├── research.md              # Phase 0 output (grid reuse, tick strategy, key map, deferred-item decisions)
├── data-model.md            # Phase 1 output (Model extensions: activeTab, planState, planMode + transitions)
├── quickstart.md            # Phase 1 output (manual verification recipe)
├── contracts/
│   └── plan-rpcs.md         # Phase 1 output (pointer to existing PlanService RPC contract + defaults)
└── checklists/
    └── requirements.md      # From /speckit-specify
```

### Source Code (repository root)

```text
services/todo/internal/cli/
├── plan_grid.go     # MODIFY: parametrize RenderGrid via a GridOptions struct
│                    #         (HideID bool, SelectedID int32 → highlight that entry's box/label).
│                    #         CLI callers pass defaults (id shown, no selection); behavior unchanged.
└── plan_grid_test.go# MODIFY: add cases for HideID and SelectedID highlighting.

services/todo/internal/tui/
├── model.go         # MODIFY: add `activeTab tab`, `planClient PlanServiceClient`, `plan planState`;
│                    #         define `tab` enum (tabTasks, tabPlanning) and planState/planMode types.
├── client.go        # MODIFY: also build and return a PlanServiceClient (same bearer interceptor).
├── tui.go           # MODIFY: pass the plan client into newModel.
├── keymap.go        # MODIFY: add NextTab/PrevTab and planning action bindings
│                    #         (add task, add event, rename, move, remove, clear, prev/next day, today);
│                    #         extend FullHelp groupings (context-aware per active tab).
├── update.go        # MODIFY: branch Update on m.activeTab; handle tab-switch (blocked while a modal is open);
│                    #         start/stop the planning tick; route non-planning msgs as today.
├── view.go          # MODIFY: render the top tab bar; branch View on m.activeTab; reserve a line for the
│                    #         tab bar in the height math; keep shared status bar/quit-guard.
├── plan_update.go   # NEW: planState reducer — msg types (planEntriesMsg, planMutatedMsg, planTickMsg),
│                    #      command factories (listPlanCmd, addPlanTaskCmd, addPlanEventCmd, renamePlanCmd,
│                    #      movePlanCmd, removePlanCmd, clearPlanCmd, planTickCmd, listTasksForPickerCmd),
│                    #      key handling for planList + sub-modes, day navigation, selection clamping.
├── plan_view.go     # NEW: render the planning tab — day header, grid (via cli.RenderGrid with options),
│                    #      task picker, and prompt forms; unstyled + styled paths.
├── plan_grid.go     # NEW (thin): TUI-side helpers that adapt entries → cli.RenderGrid options and map the
│                    #      selection cursor ↔ entry id; (kept separate from cli to avoid import cycles).
├── export_test.go   # MODIFY: export planState accessors/constructors for tests.
└── *_test.go        # NEW/MODIFY: plan_update_test.go (add/rename/move/remove/clear w/ fake client,
                     #             day nav, selection, refresh, error surfacing, tab-switch-blocked-while-modal,
                     #             now-marker tick), plan_view_test.go (grid render, picker, prompts, tab bar),
                     #             update_test.go/view_test.go (tab dispatch + height math).
```

No changes outside `services/todo/internal/tui/` except the one `RenderGrid` parametrization in `internal/cli/plan_grid.go`. No proto, handler, db, or migrations. The CLI `todo plan` subcommands keep working against the same (now-parametrized) renderer with unchanged output.

**Structure Decision**: Single Go module, client-side change. Follows the existing TUI pattern where a feature's concerns live in dedicated `plan_*.go` files with a small surface integrated into `update.go` / `view.go` / `keymap.go` — mirroring how feature 017 (move task) and feature 019 (pomodoro) were structured. Planning logic is isolated so the Tasks-tab code is touched only for tab dispatch and the tab-bar height adjustment.

## Complexity Tracking

Not applicable — no Constitution violations.
