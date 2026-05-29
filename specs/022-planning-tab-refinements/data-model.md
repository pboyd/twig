# Phase 1 Data Model: Planning Tab Refinements

This feature is UI-only. There are **no persistent data model changes** — no new tables, no proto message changes, no sqlc queries. The "entities" here are the TUI view-state structures and the shared grid-rendering options that change. The domain `PlanEntry` (`gen/plan/v1`) is used unchanged.

## Domain entity (unchanged, for reference)

### PlanEntry (`gen/plan/v1`)

| Field | Type | Notes |
|---|---|---|
| `Id` | int32 | Entry id within the day. |
| `Name` | string | Display label. Edited by the merged Edit form (via `RenamePlanEntry`). |
| `StartMinute` | int32 | Minutes from midnight. Edited via `MovePlanEntry`. |
| `DurationMinute` | int32 | Length in minutes. Edited via `MovePlanEntry`. |
| `TaskId` | int64 | 0 for events; non-zero links a task. Read-only in the Edit form. |
| `Completed` | bool | Shown in the detail pane; not edited here. |

No fields are added or changed.

## TUI view-state deltas (`internal/tui`)

### `planMode` (in `model.go`)

Planning modal state enum. **Remove** three modes, **add** one:

| Mode | Before | After |
|---|---|---|
| `planList` | kept | kept |
| `planPickTask` | kept | kept |
| `planTaskTime` | kept | kept (add-task time entry) |
| `planEventForm` | kept | kept (add event) |
| `planRename` | present | **removed** |
| `planMove` | present | **removed** |
| `planClear` | present | **removed** |
| `planEdit` | — | **added** — merged Name+Start+Duration form for the selected entry |

### `planFormState` (in `model.go`, unchanged shape)

Reused as-is for `planEdit`:

| Field | Type | Use in `planEdit` |
|---|---|---|
| `fields` | `[]textinput.Model` | `[Name, Start, Duration]` (3 fields). |
| `focus` | int | focused field index. |
| `entryID` | int32 | id of the entry being edited (prefilled). |
| `taskID` | int64 | unused for edit (0). |

State transition for `planEdit`:

```
planList --Enter(entry selected)--> planEdit
planEdit --tab/shift+tab--> cycle focus (Name → Start → Duration → Name)
planEdit --ctrl+s / enter--> submitEditForm → (compose rename?/move?) → planList + reload
planEdit --esc--> planList (no change), right pane returns to details
```

Validation rules (inherited from the prior rename/move forms):
- Name: trimmed; **must not be empty** (else `plan.err = "name cannot be empty"`, form stays open).
- Start: parsed by `timeparse.ParseStart`; invalid → `plan.err`, form stays open. May be left blank to keep the current time (see R1: a blank/unchanged time issues no `MovePlanEntry`).
- Duration: optional; parsed by `timeparse.ParseDurationOrEnd`; blank = keep existing.

### `KeyMap` (in `keymap.go`)

| Binding | Before | After |
|---|---|---|
| `Edit` | keys `e`, help "e / edit task" | keys `enter`, help "enter / edit task" |
| `PlanAddTask` | keys `a`, help "a / add task" | keys `t`, help "t / add task" |
| `PlanToday` | keys `t`, help "t / today" | keys `.`, help ". / today" |
| `PlanEdit` | — | **added** — keys `enter`, help "enter / edit entry" |
| `PlanRename` | present | **removed** |
| `PlanMove` | present | **removed** |
| `PlanClear` | present | **removed** |

`ShortHelp()`/`FullHelp()` planning groups updated accordingly (drop clear + separate rename/move; add Edit; reflect new add-task/today keys). Tasks `ShortHelp`/`FullHelp` reflect `enter` for edit automatically through the binding's help text.

## Shared renderer delta (`internal/cli/plan_grid.go`)

### `GridOptions` (additive)

| Field | Before | After |
|---|---|---|
| `HideID` | bool | unchanged |
| `SelectedID` | int32 | unchanged |
| `Styled` | bool | unchanged |
| `SelectionStyle` | — | **added** — `func(string) string` optional; styles the selected entry's content cell. When nil, falls back to current plain behavior. |

The TUI supplies `SelectionStyle = highlightStyle.Render` (bold + accent background + white foreground) via `planGridOptions()`. The plain `todo plan` CLI leaves it nil. `applySelection` is reworked to apply this styler to the entry's content cell only (not the gutter, rails, or box border); see `contracts/grid-selection.md`.
