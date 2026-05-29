# Phase 0 Research: TUI Planning Tab

This feature is mostly composition of existing parts; research focuses on the few decisions that shape the implementation and on resolving the two items deferred from `/speckit-clarify`.

## 1. Grid rendering: reuse vs. duplicate

**Decision**: Parametrize the existing `cli.RenderGrid` with a small `GridOptions` struct rather than writing a second renderer.

```go
// internal/cli/plan_grid.go
type GridOptions struct {
    HideID     bool   // omit the "[id] " prefix from entry labels
    SelectedID int32  // entry id to highlight; 0 = none
}
// RenderGrid keeps its signature for CLI callers via a default-options wrapper,
// or gains an options parameter with CLI passing the zero value (id shown, no selection).
```

The label builder drops the `[%d] ` prefix when `HideID` is set; the per-line writers apply a highlight style (reverse/bold background, matching the Tasks-tab cursor) to every line belonging to the entry whose id equals `SelectedID`, but only when styled output is active.

**Rationale**: The grid is ~270 lines of careful box-drawing and window math. Duplicating it would fork the visual logic and risk the TUI grid drifting from the CLI's, directly threatening SC-003 ("visually recognizable as the same day-planner"). One parametrized renderer keeps a single source of truth and satisfies Constitution Principle I (no duplicate logic) while keeping CLI output byte-identical (CLI passes defaults).

**Alternatives considered**:
- *New TUI-only renderer in `internal/tui`*: rejected — duplicates complex logic, invites visual drift, more test surface.
- *Move the renderer into a shared `internal/plan` package*: rejected as premature; `cli.RenderGrid` is already importable from `internal/tui` and the CLI is its only other caller. Revisit only if a third caller appears (YAGNI).

## 2. Now-marker auto-advance (FR-008a)

**Decision**: A **planning-scoped tick** — a `tea.Tick` that is armed when the Planning tab becomes active *and* the in-view day is today, re-arms itself each tick while that remains true, and is not re-armed once the user leaves the tab or navigates off today. On each `planTickMsg` the model simply triggers a re-render; `View` recomputes the now-marker from `time.Now()` (the grid already takes `now` as a parameter).

**Tick interval**: 1 second, reusing the existing pomodoro tick pattern (`tea.Tick(time.Second, …)`). The marker only has 15-minute resolution, so a coarser interval would suffice, but 1s matches the established pattern, keeps the marker feeling live, and is negligible cost (the same rate the pomodoro tick already runs at).

**Rationale**: The TUI currently re-renders only on input or while a pomodoro runs (`pomTickCmd`), so without a dedicated tick the marker would freeze on a long-open tab — exactly the scenario the user raised. Scoping the tick to "Planning tab active on today" avoids needless wakeups on the Tasks tab or when viewing other days, and keeps the pomodoro tick untouched (lower risk against the just-landed feature 019).

**Alternatives considered**:
- *Consolidate pomodoro + clock into one always-on tick*: cleaner in theory but edits feature 019's freshly-merged tick logic and runs even when nothing is animating; rejected for risk and YAGNI.
- *Two independent always-on 1s ticks*: wasteful and slightly redundant; rejected in favor of the scoped tick.

## 3. Key bindings

**Decision** (within the Planning tab's list mode; final keys may be tuned during review):

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | switch to the other tab (blocked while a modal/prompt is open, per FR-023) |
| `↑`/`k`, `↓`/`j` | move selection between entries (chronological order) |
| `a` | add task (opens task picker, then start/duration prompt) |
| `e` | add event (name, then start/duration prompt) |
| `r` | rename selected entry |
| `m` | move selected entry (start and/or duration) |
| `d` | remove selected entry |
| `c` | clear from a start time (defaults to now) |
| `[` / `]` | previous / next day |
| `t` | jump to today |
| `ctrl+r` | refresh the in-view day |
| `?` | help · `q` | quit · `esc` | cancel a prompt |

**Rationale**: Mirrors the Tasks-tab muscle memory where it makes sense (`m` move, `ctrl+r` refresh, `?`/`q`, `esc` cancel, hjkl/arrows) and uses mnemonic letters for planning-only verbs (`a`dd, `e`vent, `r`ename, `c`lear, `t`oday). Because each tab routes keys independently, reusing letters that mean something different on the Tasks tab (e.g. `c` = filter-completed there, clear here; `e` = edit there, add-event here) is safe and not surprising within a tab's own context.

**Note on `d` vs `ctrl+d`**: the Tasks tab uses `ctrl+d` to delete (guarding against accidents on a destructive, cascading task delete). Removing a plan entry is low-stakes and reversible by re-adding, so `d` is used for ergonomics, consistent with the brainstormed design. Open to switching to `ctrl+d` in review if parity is preferred.

## 4. Deferred item — task picker contents

**Decision**: The picker reuses `cli.BuildTree` over a `ListTasks` result and shows the same set the Tasks tab shows by default — incomplete tasks (the tree's normal view). Completed tasks are not offered, since scheduling an already-done task has no practical use.

**Rationale**: Minimal code (reuse the existing tree builder and node rendering), and it matches the default Tasks-tab view so the picker looks familiar. The CLI's `plan task <id>` does not forbid scheduling a completed task, but omitting them from an interactive picker is a usability improvement, not a behavior the spec requires either way (Assumptions allow choosing any task the CLI allows). No server change.

**Alternatives considered**: showing all tasks including completed (with strikethrough) — rejected as clutter for no clear benefit.

## 5. Deferred item — selection after load and after mutations

**Decision**:
- On loading/refreshing a day: highlight the **first entry chronologically** (cursor 0); if the day has no entries, there is no selection and edit/move/remove/rename are no-ops (per the spec edge case).
- After **add**: highlight the newly created entry (its id is returned by `AddPlanTaskResponse`/`AddPlanEventResponse`).
- After **move/rename**: keep the selection on that same entry (match by id after refetch).
- After **remove/clear**: clamp the cursor to the nearest remaining entry (same clamp approach as the Tasks tab's `clampCursor`).

**Rationale**: Predictable, least-surprise behavior; reuses the existing clamp pattern; keeps the user's focus on the entry they just acted on.

## 6. Prompt and form mechanics

**Decision**: Reuse Bubbles `textinput` (already used by the Tasks edit form) for all planning prompts. Multi-field prompts (event = name+start+duration; task = start+duration; move = start+duration) are small fixed-field forms following the existing `editFormModel` pattern (Tab/Shift-Tab between fields, Ctrl+S/Enter to submit, Esc to cancel). Time and duration fields are validated on submit with `timeparse.ParseStart` and `timeparse.ParseDurationOrEnd`; a parse error is surfaced in the status bar (FR-021) and the prompt stays open.

**Rationale**: Consistent input UX across tabs, no new widgets, reuses validated parsers that already back the CLI (FR-016 parity).

## 7. Plan service client wiring

**Decision**: `internal/tui/client.go` builds a `planv1connect.PlanServiceClient` alongside the existing `TaskServiceClient`, sharing the same HTTP client, address, gzip, and bearer interceptor. `newModel`/`tui.Run` thread it into the `Model`.

**Rationale**: Identical construction to the existing task client; no new auth or transport concerns. Confirms Principle II (reuse the established contract path).

## Summary of resolved unknowns

| Unknown | Resolution |
|---------|-----------|
| Reuse or duplicate the grid renderer | Parametrize `cli.RenderGrid` (GridOptions: HideID, SelectedID) |
| How the now-marker advances unattended | Planning-scoped 1s `tea.Tick`, active only on today |
| Planning key bindings | Mnemonic per-tab map (see §3) |
| Picker contents (deferred) | Incomplete tasks via `cli.BuildTree` |
| Selection after load/mutation (deferred) | First on load; follow added/edited entry; clamp after delete |
| Prompt mechanics | Bubbles `textinput` fixed-field forms; validate with `timeparse` |
| Plan client | Built in `client.go` like the task client |
