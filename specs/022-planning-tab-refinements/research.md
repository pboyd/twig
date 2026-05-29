# Phase 0 Research: Planning Tab Refinements

All decisions below are design choices within the existing `internal/tui` + `internal/cli` code; there are no external unknowns. Each entry records the decision, rationale, and rejected alternatives. Spec clarifications (jump-to-today key, highlight text color) are already resolved in `spec.md` and are not re-litigated here.

## R1 — Merged Edit form: compose existing RPCs vs. new combined RPC

**Decision**: Add a single `planEdit` form mode with fields **Name + Start + Duration (blank = keep)**, prefilled from the selected entry. On submit, run one `tea.Cmd` (`editPlanCmd`) that calls `RenamePlanEntry` then `MovePlanEntry` sequentially against the existing Plan service, then reloads the day with the entry highlighted.

**Rationale**: The two RPCs already exist (`RenamePlanEntry`, `MovePlanEntry`) and the previous rename/move forms already call them. Composing them in one command keeps all work client-side, requires no proto/server/sqlc changes, and honors Principle II (no contract change) and Principle I (no new abstraction). Sequencing (rename → move → list) avoids the race of two independent commands each issuing their own `ListPlanEntries`.

**Alternatives considered**:
- *New `UpdatePlanEntry` RPC* combining name + time — rejected: requires proto + server + sqlc work for zero behavioral gain; violates YAGNI.
- *Batch the two existing commands with `tea.Batch`* — rejected: each command reloads the plan independently, producing two list round-trips and a possible stale-overwrite race; a single sequential command is simpler and deterministic.

**Optimization**: Only call `RenamePlanEntry` when the name actually changed, and only call `MovePlanEntry` when a start/duration was entered, so a no-op field issues no RPC. Both-unchanged save is a no-op that just closes the form.

## R2 — Where the merged Edit applies to task-linked vs. event entries

**Decision**: The Edit form is identical for both entry kinds. The Name field edits the entry's own `Name` (the same field the old `RenamePlanEntry`/rename form edited); it does **not** rename the underlying task for a task-linked entry. Start/Duration edit the entry's time window.

**Rationale**: `PlanEntry.Name` is the entry's display label for both task-linked and event entries (the detail pane already shows `Name` and the linked `Task #id` separately). Inheriting the existing rename semantics avoids surprising side effects (editing a scheduled block must not silently rename a task) and needs no new server behavior.

**Alternatives considered**:
- *Editing the name of a task-linked entry renames the task* — rejected: surprising, cross-cutting, and outside the existing RPC's contract.

## R3 — Rendering Planning forms/picker in the right pane

**Decision**: When `m.plan.mode != planList`, `viewPlanning` renders a two-pane layout: the grid on the left and the active picker/form on the right, exactly mirroring the Tasks tab's `viewWithForm` (styled: `paneBox` + `lipgloss.JoinHorizontal`; non-styled: `splitLines` + `padRightAnsi` row-join). The full-width modal branch is removed.

**Rationale**: Feature 021 already established the Planning two-pane layout and the right "Details" pane. Hosting the form in that same right pane keeps the grid visible (the spec's headline UX goal) and reuses the exact join logic the Tasks tab uses, so narrow-terminal and non-styled behavior are inherited rather than re-derived.

**Alternatives considered**:
- *Keep the full-width modal* — rejected: it is the specific inconsistency the spec removes.
- *Overlay the form on top of the grid* — rejected: no such overlay primitive exists; the side-by-side pane is the established pattern.

**Pane focus**: while a form/picker is open, the right pane is the focused (accent-bordered) pane and the grid pane is unfocused, mirroring `viewWithForm` where the form pane carries focus.

## R4 — Cell-only selection highlight reusing the Tasks `highlightStyle`

**Decision**: Rework `internal/cli/plan_grid.go` so the selection styling is applied to **only the entry's text content cell**, not the whole rendered line. The styling reuses the Tasks tab's selection look — **bold + blue accent background + white foreground** — passed in from the TUI via an additive `GridOptions` field (a `func(string) string` styler), so the `cli` package does not depend on the `tui` theme. Border-drawing characters (`┏┓┗┛┣┫┃`), the heavy fillers context, the hour gutter (`07:00`), and the `│`/`├┤` rails are left unstyled.

**Rationale**: Today `applySelection` wraps the entire line (gutter + rails + border + label) in a blue **foreground** code (`\x1b[94m`), which is exactly the "border and hour marker also highlighted" behavior the spec removes. Scoping the style to the content cell and reusing the literal `highlightStyle` makes the Planning selection indistinguishable from a selected Tasks row (FR-006/SC-003) while satisfying "only the cell" (FR-007). Passing the styler as a function keeps the shared `cli` package free of a `tui` dependency and reuses the *exact* style object rather than re-encoding its ANSI.

**Mechanics**:
- For **single-row** entries (`┣ label━━━ ┫`), style the inner field (label + heavy filler) as the cell; leave `┣`/`┫` plain.
- For **multi-row** entries, style only the **interior label rows** (`┃ content ┃` → style `content`); leave the `┏━┓` top, `┗━┛` bottom, shared `┣━┫` borders, and the `┃` walls plain.
- Selection no longer alters the now-marker gutter or rails on any row.

**Alternatives considered**:
- *Hardcode equivalent background ANSI in `cli`* — rejected: duplicates the accent color (`tui/theme.go` vs `cli`), risking drift; "reuse the exact same style" is better served by handing the `tui` style in.
- *Keep whole-line styling but switch fg→bg* — rejected: still colors the border and gutter, violating FR-007.
- *Move the grid renderer into `tui`* — rejected: the grid is shared with the plain `todo plan` CLI; relocating it is a large, unnecessary change.

**Backward compatibility**: the new `GridOptions` field is optional. When nil (the plain `todo plan` CLI path, where `SelectedID == 0` anyway), selection rendering falls back to the existing plain behavior, so non-TUI output is unchanged.

## R5 — Key rebindings and help/status consistency

**Decision**:
- Tasks tab: `Edit` binding keys `e` → `enter`. `handleListKey`'s existing `key.Matches(msg, m.keys.Edit)` case is unchanged; only the bound key string changes. The Tasks `FullHelp`/`ShortHelp` then advertise `enter` automatically via `key.WithHelp("enter", "edit task")`.
- Planning tab: `PlanAddTask` `a` → `t`; `PlanToday` `t` → `.`; add a `PlanEdit` binding (`enter`, "edit entry"); remove `PlanRename`, `PlanMove`, `PlanClear` bindings.
- `handlePlanningKey`: add an `Enter`→`initEditForm` case (guarded on a non-empty selection); the `t` and `.` cases follow from the rebound `PlanAddTask`/`PlanToday`; delete the rename/move/clear cases.
- Update `ShortHelp`/`FullHelp` planning groups to drop clear and the separate rename/move, add Edit (Enter), and reflect `t`=add task and `.`=today.

**Rationale**: `Enter` is currently unbound on both lists, so it is free to take over editing on each tab with no collision. After the rebinds every advertised key maps to exactly one action (FR-011): planning uses `t` add-task, `e` add-event, `enter` edit, `ctrl+d` remove, `[`/`]`/`.` day-nav, `x` pom-cancel, `?` help, `q` quit, `tab` switch — all distinct.

**Alternatives considered**:
- *Keep `a` as an alias for add-task alongside `t`* — rejected: the spec says `t` is the add-task key; a hidden alias adds surface for no stated need.
- *Bind Edit to a multi-key set including both `e` and `enter`* — rejected: the spec explicitly removes `e` for editing on the Tasks tab.

## R6 — Help screen derivation

**Decision**: No bespoke help content is authored. The Planning and Tasks help screens already render from `KeyMap.FullHelp()` (which branches on `PlanningMode`); updating the binding groups in `keymap.go` automatically updates both help screens and the status line.

**Rationale**: Feature 021 made `FullHelp` planning-aware; reusing it keeps help text and behavior in lockstep and prevents the "advertised-but-broken key" class of bug the prior feature fixed.
