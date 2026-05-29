# Phase 0 Research: Planning Tab Polish

Each decision below was reached by reading the current `internal/tui` and `internal/cli` code. The theme is "reuse what the Tasks tab already does."

## 1. Pomodoro cancel on the Planning tab

- **Decision**: Add a `key.Matches(msg, m.keys.PomCancel)` case in `handlePlanningKey` (planList branch) that runs the existing `cancelPomCmd(m.client)` when `m.pom != nil && !m.pom.completed`. Also handle the running quit-confirmation (`y`/`n`/`esc`) at the top of `handlePlanningKey`, mirroring `handleListKey`.
- **Rationale**: `handleKey` routes to `handlePlanningKey` and returns before reaching the Tasks-only `PomCancel` case (update.go:665) and the `confirmingQuit` handler (update.go:538). The cancel command itself (`cancelPomCmd`) is shared and tab-independent; only the key routing is missing. The quit-confirm gap is the same class of bug — `handlePlanningKey` sets `confirmingQuit = true` (update.go:390) but never reads `y`/`n`, so a running-pomodoro quit is currently stuck on the Planning tab.
- **Alternatives considered**: Hoisting pom/quit handling above the tab switch in `handleKey` so both tabs share it. Rejected for now — it would entangle the two reducers and risks regressing Tasks-tab behavior; the targeted cases are smaller and testable in isolation (Principle I).

## 2. Planning help screen

- **Decision**: Reuse the existing help machinery. On the Planning tab, route `m.keys.Help` to set `m.mode = modeHelp` (or an equivalent planning-help flag) and let `View()` fall through to `viewHelp()`; dismiss returns to the grid with selection intact. No new help content is written — `KeyMap.FullHelp()` already branches on `PlanningMode` (keymap.go:185) and `PlanningMode` is already toggled on tab switch (update.go:404/563).
- **Rationale**: The help *content* for Planning already exists and is correct; only the *entry/exit wiring* on the Planning tab is missing (`viewPlanning` never renders help; `handlePlanningKey` never enters help). Reusing `viewHelp()` guarantees identical visual style and dismissal (FR-004).
- **Implementation note**: `View()` checks `m.activeTab == tabPlanning` first and returns `viewPlanning()` before the `modeHelp` switch (view.go:18). The cleanest fix is to check the help condition inside `viewPlanning()` (e.g. a `planHelp` planMode, or honor `m.mode == modeHelp`) and render `viewHelp()` there, so help is reachable from Planning without reordering the top-level `View()` dispatch.
- **Alternatives considered**: A separate Planning-only help renderer. Rejected — duplicates the `help.Model` styling and violates the consistency goal.

## 3. Sub-tasks in the add-task picker

- **Decision**: Build the picker's visible rows with **all nodes expanded** by passing a fully-populated `expanded` map to the existing `buildVisible`, instead of the empty map currently used (update.go:342). The picker has no expand/collapse keys, so an empty map hides every sub-task; a full map reveals the entire incomplete-task tree with the same `├──`/`└──` connectors the Tasks list uses.
- **Rationale**: This reuses the Tasks-tab flattener (`buildVisible`) verbatim — the most direct interpretation of the user's "selecting a task from a list seems like something that could be re-used." It yields a richer, connector-style tree than the move dialog's 2-space indentation while still satisfying "sub-tasks visible, consistent presentation." A small helper (e.g. `allTaskIDs(tree) map[int64]bool`) produces the expand-all map.
- **Alternatives considered**:
  - *Reuse the move dialog's `buildCandidates`/indented list directly*: viable and explicitly sanctioned by FR-006, but `buildCandidates` carries move-specific concerns (the `(no parent)` sentinel, excluding the moving task) and uses flat indentation rather than the tree connectors already present in the picker. Expand-all `buildVisible` is the smaller, cleaner reuse.
  - *Add expand/collapse keys to the picker*: more UI surface than the gap requires (YAGNI); deferred unless desired later.
- **Consistency check**: Both the move dialog and the picker will show the full incomplete-task hierarchy with depth; acceptance scenario US3-#3 ("present the hierarchy the same way — no omitted levels") is satisfied by neither omitting levels.

## 4. Two-pane Planning layout

- **Decision**: Restructure `viewPlanning()` to mirror `viewList()`: a left pane containing the day header + calendar grid and a right details pane, composed with `paneBox` + `lipgloss.JoinHorizontal` in styled mode, and the `splitLines`/`padRightAnsi` row-join fallback in non-styled mode. Add `renderPlanDetail(width)` that renders the selected entry (read-only, per the spec clarification).
- **Rationale**: `viewList` already solves the exact layout problem (split panes, height math accounting for `tabBarHeight` and `statusHeight()`, narrow-terminal fallback). Reusing its structure gives pixel-consistent panes and, as a side effect, pins the status line to the bottom (see §5). The details pane is read-only (clarification): existing rename/move prompts remain the edit path.
- **Details content**: name, `HH:MM–HH:MM` window, and duration; for a task-linked entry, the linked task and its completion state; events omit task-only fields. Empty day → placeholder, matching `renderDetailPane` returning empty for no selection.
- **Alternatives considered**: A standalone planning split helper. Rejected — `paneBox`/`splitLines` already generalize across `viewList`, `viewWithForm`, and `viewWithMove`.

## 5. Status line pinned to bottom

- **Decision**: Resolved as a consequence of §4. The current `viewPlanning` renders the grid at its natural height and appends the status, so the status floats directly under the grid. Building full-height panes with `paneBox(..., innerH, ...)` (where `innerH = height - 2 - statusHeight() - tabBarHeight`) fills the vertical space, so `renderStatus()` lands on the bottom row exactly as on the Tasks tab.
- **Rationale**: No new layout logic; the fix falls out of adopting the `viewList` height arithmetic.

## 6. Redundant panel titles

- **Decision**: Pass an empty title to the left list/grid pane on both tabs (drop `"Tasks"` in `viewList`/`viewWithForm`/`viewWithMove` and the equivalent `"Planning"`/`"Tasks"` panel label on the grid pane). Keep the right pane's `"Details"` title (it labels a distinct pane, not the view). Keep the Planning day-header date line (distinct information).
- **Rationale**: The tab bar already names the active view, so a `"Tasks"`/`"Planning"` pane title is the duplicated word the user flagged. The clarification confirmed removing it on **both** tabs. `"Details"` and the date header are not redundant and stay.

## 7. Grid theming (monotone → blue accent)

- **Decision**: Extend the shared `cli.GridOptions` with an **optional, additive** styling hook so the TUI can render selection/borders with the theme palette while the plain `todo plan` CLI keeps its current output. Concretely, add an optional accent/selection style (e.g. a `Styled bool` or function/color fields) that, when set, makes `applySelection` use the accent highlight (`cursorBg` background like the Tasks list cursor) and tints box borders with `accent`; when unset (the CLI default), behavior is byte-for-byte unchanged.
- **Rationale**: The grid is drawn by `cli.RenderGrid`, shared with the non-interactive CLI. `applySelection` currently emits raw `\x1b[1m` bold (plan_grid.go:242) — that is the "monotone" look. Threading an optional style through `GridOptions` (already the TUI's injection point via `planGridOptions`) is additive and keeps the CLI contract stable, satisfying Principle II.
- **Alternatives considered**:
  - *Post-process the grid string in the TUI to recolor*: brittle (must parse box-drawing/ANSI), rejected.
  - *Fork a TUI-only grid renderer*: large duplication, violates Principle I.
- **Contract impact**: Documented in `contracts/grid-options.md`. The new field is optional with a zero value equal to today's behavior.

## 8. Testing approach

- **Decision**: Extend existing string-view tests and reducer tests. Add `plan_view_test.go` cases asserting: themed selection style present when `styled`; two-pane layout (grid + details) with bottom status; no redundant pane title; details content for task vs. event entries. Add `update_test.go`/`plan_update_test.go` cases for: `PomCancel` on Planning issues `cancelPomCmd`; quit-confirm `y`/`n` on Planning; `?` opens/closes help on Planning; picker visible rows include sub-tasks (expand-all). Keep a `!styled` assertion for each rendering change to protect the non-ANSI fallback.
- **Rationale**: Matches the established convention (no DB, fixed clock, stub clients, `export_test.go` shims).
