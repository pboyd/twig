# Phase 1 Data Model: TUI Theme Pass

This feature introduces **no persisted data and no API/proto entities**. It is a
presentation-only change. What follows documents the in-memory render-time
structures the implementation touches, so the renderer and its tests stay
aligned. These are internal Go types in `internal/tui`, not data contracts.

## Render-time structures

### `visibleRow` (modified — `tree.go`)

One rendered row in the task list pane. Today it carries a pre-computed `marker`
string conflating fold state. It is changed to expose **fold state**, letting the
renderer (not the tree builder) decide glyphs and gate them on `styled`.

| Field | Type | Meaning | Change |
|-------|------|---------|--------|
| `node` | `*cli.TreeNode` | The task node (source of name + completion). | unchanged |
| `depth` | `int` | Nesting depth. | unchanged |
| `treePrefix` | `string` | Tree connector prefix (`├── `, `└── `, `│   `). | unchanged |
| `marker` | `string` | Pre-baked `[-]`/`[+]` fold marker. | **removed** |
| `expandable` | `bool` | Row has collapsible children currently hidden (the old `hasExpandableChildren`). Drives whether a chevron is shown. | **added** |
| `expanded` | `bool` | Row is currently expanded. Drives chevron direction. | **added** |

**Derived at render time (not stored)**:
- *Completion* — read from `node.Task.GetCompletedAt() != nil`.
- *Chevron glyph* — `expandable ? (expanded ? "▾" : "▸") : " "` (styled path only).
- *Checkbox glyph* — `completed ? "☑" : "☐"` (styled path only).

> Note: in the *unstyled* path, no chevron/checkbox glyphs are emitted; output
> preserves today's plain form (see research Decision 3).

### `Model` (unchanged shape)

No fields added. The existing `styled bool` gates all new styling; `width` /
`height` drive box sizing; `cursor` selects the highlighted row. No new state is
required to satisfy any requirement.

## Validation / invariants (render contract, enforced by tests)

These are the testable invariants that replace "validation rules" for a UI
feature; the authoritative form lives in
[`contracts/ui-rendering-contract.md`](./contracts/ui-rendering-contract.md).

1. **Chevron ⇔ expandable**: a chevron is present on a row **iff** `expandable`
   is true (FR-004).
2. **Checkbox on every row**: a completion glyph is present on every styled row
   and matches completion state (FR-005).
3. **Completed name distinguished**: completed rows render the name with
   strikethrough/dim-green in the styled path (FR-006); unstyled path: no codes.
4. **Inner width fidelity**: `lipgloss.Width` of each rendered pane equals its
   allotted outer width; names truncate / descriptions wrap to inner width
   (FR-007, SC-004) — content never overflows the frame.
5. **Focus emphasis**: the active pane's border uses the accent color; the
   inactive pane uses dim (FR-002).
6. **Plain when unstyled**: `styled == false` ⇒ no ANSI escape codes and no
   decorative unicode glyphs anywhere in output (FR-012, SC-003).
7. **No panic at degenerate sizes**: inner width clamps ≥ 0, inner height ≥ 1
   (FR-014, SC-006).

## State transitions

None. No new modes, no new keybindings, no lifecycle. Mode set
(`modeList`, `modeEdit`, `modeNewSubtask`, `modeNewRoot`, `modeHelp`,
`modePomodoro`, `modeMove`) is unchanged; only how each list-mode view *renders*
changes.
