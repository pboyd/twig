# Phase 1 Data Model: Planning Tab Polish

This feature is UI-only. There are **no persisted entities, no DB tables, and no proto/message changes**. The relevant "data" is the in-memory view-state (Bubble Tea `Model`) and the cross-package render options. This document records the deltas to those structures.

## Existing entities (unchanged, for reference)

- **PlanEntry** (`gen/plan/v1`): `Id int32`, `Name string`, `StartMinute int32`, `DurationMinute int32`, `Completed bool`, plus task linkage. Rendered on the grid and now also in the details pane. No changes.
- **Task** (`gen/task/v1`): used by the add-task picker tree. No changes.

## View-state deltas (`internal/tui`)

### `Model` / `planState` (model.go)

- **No new persisted fields required for the core fixes.** Pomodoro cancel, help, and the picker fix reuse existing fields (`m.pom`, `m.mode`, `m.confirmingQuit`, `m.plan.picker`).
- **Help on Planning**: reuse the existing `m.mode == modeHelp` flag rather than adding a planning-specific mode, OR add a `planHelp` value to the `planMode` enum. Decision deferred to implementation; either is a single enum/flag, not a new struct. The renderer (`viewPlanning`) must consult whichever flag is chosen.
- **`pickerState.expanded`**: already exists (`map[int64]bool`). It will be populated with **all** incomplete task IDs (expand-all) when the picker opens, instead of an empty map. No shape change — only the value passed to `buildVisible`.

### Helper (new, small)

- **`allTaskIDs(tree []*cli.TreeNode) map[int64]bool`** (or equivalent): returns every task ID in the tree so the picker can be built fully expanded. Pure function, table-testable. This is the only genuinely new code unit, and it is a leaf helper, not an abstraction layer.

## Cross-package render contract delta (`internal/cli`)

### `GridOptions` (plan_grid.go) — additive

Current:

```go
type GridOptions struct {
    HideID     bool
    SelectedID int32
}
```

Planned (additive, backward-compatible — zero value preserves today's CLI output):

```go
type GridOptions struct {
    HideID     bool
    SelectedID int32
    // Styled, when true, renders the selected entry with the TUI accent
    // highlight and tints box borders with the accent color instead of the
    // plain bold used by the non-interactive CLI. Zero value (false) keeps
    // the existing CLI rendering byte-for-byte.
    Styled bool
    // (Exact field shape — bool vs. color/func hooks — finalized in
    //  contracts/grid-options.md and during implementation.)
}
```

- **Validation/Invariants**:
  - When `Styled == false`, `RenderGrid` output is identical to the pre-feature output (regression-guarded by existing CLI grid tests).
  - Styling is applied only when `isTTY` is also true (the TUI always passes `m.styled`, which already gates on `WantStyled`).

## Details-pane projection (presentation only)

`renderPlanDetail(entry *planv1.PlanEntry, width int, styled bool) string` projects a `PlanEntry` to read-only text:

| Field shown | Source | Notes |
|---|---|---|
| Name | `entry.Name` | bold accent header when styled (mirrors `renderDetails`) |
| Time window | `StartMinute`, `DurationMinute` | formatted `HH:MM–HH:MM` |
| Duration | `DurationMinute` | e.g. `30m` |
| Linked task / completion | task linkage, `entry.Completed` | shown only for task entries; events omit these |

Empty/placeholder state when no entry is selected (no entries on the day), consistent with `renderDetailPane` returning empty.
