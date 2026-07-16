# Implementation Plan: Goal Task Filter Shortcut

**Branch**: `065-goal-task-filter` | **Date**: 2026-07-16 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/065-goal-task-filter/spec.md`

## Summary

Add a `ctrl+t` binding to the Goals tab that jumps to the Tasks tab with the filter `^goal_id=<id>`
applied for the goal under the cursor, so users can reach a goal's tasks without knowing its ID.

The filter engine, the `^goal_id` expression form, the transitive matching, and the filter bar all
already exist and ship untouched. This feature is a *composer*: it writes the expression the user could
have typed and pushes it through the exact path the `/` key already uses. The entire change lives in
`internal/tui` across three files — no proto, handler, SQL, or web change.

The one non-obvious constraint, from research R3: `handleFilterResult` records the active filter by
reading `m.filterInput.Value()`, not the expression handed to `filterCmd`. The shortcut must therefore
set the input's value before dispatching, or the filter would apply while `filterExpr` silently
desynchronized — breaking filter editing, filter replacement, and every mutation re-fire path.

## Technical Context

**Language/Version**: Go 1.25 (root module `github.com/pboyd/twig`)

**Primary Dependencies**: `charm.land/bubbletea/v2` (TUI runtime), `charm.land/bubbles/v2` (textinput,
key), `charm.land/lipgloss/v2` (styling), `connectrpc.com/connect` (existing `FilterTasks` client)

**Storage**: N/A — no persistence change. The feature composes a query against the existing server-side
filter engine; no schema, migration, or sqlc regeneration.

**Testing**: `go test ./...` from repo root. Table-driven unit tests against `Model.Update` using
synthetic `tea.KeyPressMsg` values, following `internal/tui/update_test.go` and `goal_view_test.go`.
Unexported access via the existing `internal/tui/export_test.go` shim. No running database required.

**Target Platform**: Linux/macOS terminal (TTY). TUI only — the web app is out of scope.

**Project Type**: Multi-module Go monorepo (CLI/TUI + api + server). This feature touches the root
CLI/TUI module exclusively.

**Performance Goals**: Keystroke-to-tab-switch is synchronous and instant; the task list populates on the
existing `FilterTasks` round-trip, which is unchanged. No new N+1 or extra round-trip is introduced — one
RPC per jump, identical to a typed filter.

**Constraints**: Must not alter `ctrl+t` on the Plan tab (FR-008). Must not regress the `c` show-all
toggle for filtered views (FR-009). Must remain inert in Goals sub-modes (FR-007).

**Scale/Scope**: ~3 source files touched, ~25 lines of production code, plus tests. One new key binding.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses the existing filter engine and `PlanGoToTask` jump pattern verbatim. No new abstraction, no helper layer, no new RPC. Research R1/R3 explicitly rejected both a dedicated `ListTasksByGoal` endpoint and a `filterResultMsg` refactor as speculative. Complexity Tracking table is empty. |
| II. API-First Design | ✅ | No API surface changes — `^goal_id` and `FilterTasks` already exist (R1). The contract this feature *does* introduce is a UI interaction contract, committed to `contracts/keybinding.md` before implementation per Quality Gate 1. |
| III. UI/UX Consistency | ✅ | Reuses the shared `KeyMap`, the existing filter bar, and the established `PlanGoToTask` jump ordering (R5). `ctrl+t` keeps its existing "take me to my tasks" meaning, now on a second tab. No new colors, styles, or widgets. Documented in Goals `FullHelp` per FR-010. |
| IV. Playful User Messages | ✅ | The feature introduces no new user-facing strings. The empty-goal case (FR-004) routes to the existing "no tasks match" state, whose tone was already reviewed. Should implementation find that state needs goal-specific wording, it must be composed in the established warm register — see `contracts/keybinding.md` §5. |

**Post-Phase-1 re-evaluation**: Still passing on all four. The Phase 1 design added no entities, no
endpoints, and no strings; `data-model.md` records state *transitions* over existing fields rather than
any new structure.

## Project Structure

### Documentation (this feature)

```text
specs/065-goal-task-filter/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output — R1–R5 verification of spec assumptions
├── data-model.md        # Phase 1 output — TUI state transitions (no persisted entities)
├── quickstart.md        # Phase 1 output — manual verification walkthrough
├── checklists/
│   └── requirements.md  # Spec quality checklist (passing)
├── contracts/
│   └── keybinding.md    # Phase 1 output — TUI interaction contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/tui/
├── keymap.go        # MODIFY: add GoalGoToTasks binding (ctrl+t); add to Goals FullHelp (FR-010)
├── update.go        # MODIFY: add case in handleGoalsKey main switch (~line 1439, near GoalAddTask)
├── model.go         # READ-ONLY: goalState.cursor / goalState.goals are the selection source
├── goal_view.go     # READ-ONLY: visibleGoals() resolves cursor → goal, matching detail-pane logic
└── update_test.go   # ADD: shortcut behavior tests
                     # (or a new goal_filter_test.go, following the per-feature test-file convention)

services/twig/internal/filter/    # UNCHANGED — ^goal_id parse + transitive eval already ship (R1, R2)
api/proto/                        # UNCHANGED — no new messages or RPCs
services/twig-web/                # UNCHANGED — TUI-only scope
```

**Structure Decision**: Root CLI/TUI module only (`github.com/pboyd/twig`). The `api/` and
`services/twig/` modules are untouched, which is what keeps this feature small: the backend capability
already exists and is reached through the current `filterCmd` → `FilterTasks` path. Test files follow the
existing `internal/tui/*_test.go` convention with `export_test.go` for unexported access.

## Implementation Approach

### Change 1 — `keymap.go`: declare the binding

Add to the Goals-tab key group in `KeyMap`, alongside `GoalAddTask` / `GoalLinkTask`:

```go
GoalGoToTasks key.Binding // ctrl+t: jump to Tasks filtered to this goal
```

Bound as `key.WithKeys("ctrl+t")`, `key.WithHelp("ctrl+t", "show goal's tasks")`. Add it to the
`GoalMode` branch of `FullHelp` (FR-010). It is deliberately **not** added to the `GoalMode` `ShortHelp`
row, which is already near capacity.

**Do not touch** the existing `PlanGoToTask` binding, which independently declares `ctrl+t` and is matched
only inside `handlePlanningKey` (FR-008). Two bindings may share a key; they are dispatched by tab.

### Change 2 — `update.go`: handle the key

Add a case to the main switch in `handleGoalsKey`, placed near `GoalAddTask` (~line 1439). It reaches the
main switch only after the existing early returns for form, picker, status-history, confirm-delete, and
help modes — which is what satisfies FR-007 with no new checks (R4).

Ordering is load-bearing (R3):

```go
case key.Matches(msg, m.keys.GoalGoToTasks):
    // ctrl+t: jump to Tasks filtered to this goal's whole tree.
    if len(visible) == 0 || m.goal.cursor >= len(visible) {
        return m, nil                    // FR-006: no goal under cursor, do nothing
    }
    g := visible[m.goal.cursor]
    expr := fmt.Sprintf("^goal_id=%d", g.Id)

    m.activeTab = tabTasks               // FR-001
    m.keys.GoalMode = false              // help flags follow the tab (R5)
    m.goal.err = nil
    m.err = nil

    m.filterInput.SetValue(expr)         // FR-003/FR-004 — MUST precede dispatch (R3)
    m.filterInput.Blur()                 // focus belongs to the list, not the input
    m.mode = modeList
    m.filterInvalid = false
    m.filterGen++                        // invalidate any in-flight filter response
    return m, filterCmd(m.client, expr, m.showAll, m.nowOrDefault(), m.filterGen)
```

Notes on the details:

- `visible` is the already-computed `visibleGoals(m.goal.goals, m.goal.showAll)` in scope in that switch —
  the same list the cursor and detail pane index into, so the jump always targets the goal the user sees.
- `m.showAll` is passed through unmodified, preserving the completion/snooze default (FR-009, R2).
- `filterGen++` before dispatch is the existing staleness guard; `handleFilterResult` drops any response
  whose `gen` no longer matches.
- Assigning `m.filterExpr` here is unnecessary and would be misleading — `handleFilterResult` sets it from
  the input on success, which is precisely why `SetValue` comes first.
- No cursor positioning: unlike `PlanGoToTask` there is no single target task, so the cursor settles via
  `clampCursor` in `handleFilterResult` (R5).

### Change 3 — tests

Table-driven tests over `Model.Update` with a fake `TaskServiceClient`, covering each requirement:

| Test | Asserts | Requirement |
|------|---------|-------------|
| Jump applies goal filter | `activeTab == tabTasks`; `filterInput.Value() == "^goal_id=<id>"`; `FilterTasks` called with that expression | FR-001, FR-002 |
| Expression is recorded after result | after `handleFilterResult`, `filterExpr == "^goal_id=<id>"` | FR-003 |
| Replaces an existing filter | pre-set a different `filterExpr`/input; assert both are overwritten | FR-004 |
| Empty goals list | `ctrl+t` with no goals: `activeTab` unchanged, no command returned | FR-006 |
| Inert in sub-modes | `ctrl+t` while `goalEdit`, `goalPickLink`, `goalStatusHistory`: no tab switch | FR-007 |
| Plan tab unaffected | `ctrl+t` on Plan still jumps to the entry's task | FR-008 |
| show-all passthrough | `showAll` true/false reaches `FilterTasks` unchanged | FR-009 |
| Help lists the shortcut | Goals `FullHelp` contains the binding | FR-010 |

The FR-008 test is the regression guard for the shared key and should be treated as non-negotiable.

## Complexity Tracking

> No Constitution Check violations. No entries.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| *(none)* | — | — |
