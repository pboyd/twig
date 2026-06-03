# Contract: Grid Window Rendering Interface

This feature exposes no network/API surface. The relevant contract is the
**internal Go rendering interface** between `internal/tui` and `internal/cli`.
Per the constitution's API-First principle, that contract is fixed here before
implementation. The overriding invariant is **CLI byte-for-byte parity**.

## 1. `GridOptions` extension (`internal/cli/plan_grid.go`)

Two optional fields are added to the existing struct:

```go
type GridOptions struct {
    // ... existing fields unchanged ...

    // WindowStartMin, when non-nil, fixes the grid's first minute (minutes since
    // midnight) and disables automatic backward expansion of the window start.
    WindowStartMin *int
    // WindowEndMin, when non-nil, fixes the grid's last minute and disables
    // automatic forward expansion of the window end.
    WindowEndMin *int
}
```

### Semantics

| `WindowStartMin` | `WindowEndMin` | Behavior |
|------------------|----------------|----------|
| `nil` | `nil` | **CLI default** — identical to current behavior: window = 08:00–17:00 expanded outward to contain every entry. |
| set | `nil` | Start fixed at the given minute (no backward expansion); end still expands to contain entries. |
| `nil` | set | End fixed at the given minute (no forward expansion); start still expands to contain entries. |
| set | set | Window fully fixed; no auto-expansion on either side. |

### Invariant (CLI parity)

> When `WindowStartMin == nil && WindowEndMin == nil`, `RenderGrid` output MUST be
> byte-for-byte identical to the pre-feature implementation.

Guarded by existing tests: `TestRenderGrid_EmptyDay`,
`TestRenderGrid_WindowExtensionEarly`, `TestRenderGrid_WindowExtensionLate`
(must remain unchanged and green).

## 2. `GridWindow` helper (`internal/cli/plan_grid.go`)

```go
// GridWindow returns the [startMin, endMin] window (minutes since midnight,
// 15-minute aligned) the TUI should pass to RenderGrid so the timed grid fills
// availableRows of vertical space.
//
//   - entries:       the day's timed plan entries
//   - now:           current time (for the anchor block and today detection)
//   - day:           the in-view day, "YYYY-MM-DD"
//   - availableRows: rows the timed grid may occupy (>= 1)
//
// Modes (see data-model.md):
//   FILL          availableRows >= baseRows      → start=baseStart, fill later
//   ANCHOR        constrained & today & now>=base → start=now-block, fill later
//   TOP-TRUNCATE  constrained & (not today | early) → start=baseStart
//
// end is always clamped to 24:00 (1440). The CLI does not call this helper.
func GridWindow(entries []*planv1.PlanEntry, now time.Time, day string, availableRows int) (startMin, endMin int)
```

### Guarantees

- `1 <= availableRows` assumed (caller clamps); result satisfies
  `0 <= startMin < endMin <= 1440`.
- Pure: no I/O, deterministic given inputs.
- In FILL/TOP-TRUNCATE, `startMin == baseStart`; in ANCHOR,
  `startMin == snapDown15(now block)`.

## 3. TUI call-site contract (`internal/tui/plan_view.go`)

Each grid render path MUST:
1. Render the untimed pane + separator and count their lines `u`.
2. Compute `availableRows = max(1, innerGridHeight - u)`.
3. `start, end := cli.GridWindow(timed, now, day, availableRows)`.
4. Pass `WindowStartMin: &start, WindowEndMin: &end` in `GridOptions`.
5. Keep the existing `lines[:height]` trim as a defensive backstop.

Applies to: `renderPlanGrid` (styled, live), `renderPlanGridContent`
(unstyled, live), and `renderPlanningView` (test-only helper) for consistency.

## Out of scope (explicitly unchanged)

- `proto/`, `gen/`, `internal/handler`, `internal/db` — untouched.
- `RenderUntimed`, `RenderUntimedSeparator` signatures — untouched.
- CLI command `plan` output (`internal/cli/plan.go:122`) — untouched.
