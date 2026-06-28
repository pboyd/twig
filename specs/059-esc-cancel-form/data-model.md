# Phase 1 Data Model: Esc Cancels TUI Forms

This feature has no persistent data and no API schema. The "data model" here is the
in-memory TUI state added or touched in `internal/tui`. All types already exist;
the changes are additive fields/methods plus one new message type.

## editFormModel (`internal/tui/edit.go`) — modified

Add an opened-state snapshot captured when the form is built (in `newBlankForm`,
`NewEditForm`, `NewRootForm`, `NewSubtaskForm`).

| Field (new) | Type | Purpose |
|---|---|---|
| `origName` | `string` | Value of `name` at open |
| `origDescription` | `string` | Value of `description` at open |
| `origDue` | `string` | Value of `due` at open |
| `origEstimate` | `string` | Value of `pomodoroEstimate` at open |
| `origSnooze` | `string` | Value of `snooze` at open |
| `origGoalIdx` | `int` | Value of `goalIdx` at open (when `showGoalField`) |

> Implementation note: these may instead be stored as a single `opened` snapshot
> struct; the field-by-field list above is the logical content either way.

**New method**: `isDirty() bool` — returns true if any current field value, after
`strings.TrimSpace`, differs from its snapshot (and, when `showGoalField`, if
`goalIdx != origGoalIdx`). Whitespace-only differences are not dirty (FR-010).

**Behavior change in `editFormModel.Update`** (calendar closed only):
- `Esc` and `!isDirty()` → command emitting `editCancelledMsg{originalCursor}`.
- `Esc` and `isDirty()` → command emitting `editDiscardRequestedMsg{originalCursor}`.
- `Esc` while `calendar != nil` → unchanged (closes the calendar).

## editDiscardRequestedMsg (`internal/tui/edit.go` or `update.go`) — new

```go
// editDiscardRequestedMsg is emitted when Esc is pressed on a dirty edit form,
// asking the model to raise the discard-confirmation overlay.
type editDiscardRequestedMsg struct {
    originalCursor int
}
```

Handled by `Model.Update`: sets `confirmingDiscard = true` (form state is left
intact so declining restores it).

## planFormState (`internal/tui/model.go`) — modified

| Field (new) | Type | Purpose |
|---|---|---|
| `origValues` | `[]string` | Snapshot of each `fields[i].Value()` at form open |

Captured wherever the plan form is initialized (`initTaskTimeForm`,
`initAddEventForm`, `initEditForm`, or equivalent). A helper
`planFormDirty(form planFormState) bool` compares trimmed current field values to
`origValues`.

**Behavior change in `handlePlanFormKey`**:
- `Esc`/`keys.Cancel` and not dirty → `plan.mode = planList` (immediate cancel).
- `Esc`/`keys.Cancel` and dirty → `confirmingDiscard = true`.

## Model (`internal/tui/model.go`) — modified

| Field (new) | Type | Purpose |
|---|---|---|
| `confirmingDiscard` | `bool` | When true, the modal discard prompt is shown and intercepts keys |

**Key interception** (top of `handleKey`, before tab routing):
- `y` → perform cancel: if `activeTab == tabPlanning` and `plan.mode` is a form mode,
  set `plan.mode = planList`; else emit `editCancelledMsg{originalCursor}` (its
  handler restores the prior cursor and resets the right mode for Tasks/Goals).
  Then clear `confirmingDiscard`.
- `n` or `esc` → clear `confirmingDiscard` only (return to the form, input intact).

## State transitions

```
Form open (clean snapshot taken)
   │
   ├─ user edits fields ──────────────► form dirty
   │
   ▼ Esc
 isDirty? ── no ──► cancel immediately ──► list/grid view (nothing saved)
   │
   yes
   ▼
 confirmingDiscard = true  (form preserved underneath)
   │
   ├─ y ──► cancel ──► list/grid view (input discarded, nothing saved)
   └─ n / esc ──► confirmingDiscard = false ──► back in form (input intact)
```

## Invariants

- Canceling (clean or via confirm) never issues a create/update RPC.
- Declining the discard prompt mutates no form field — all typed input survives.
- A closed nested calendar overlay consumes `Esc` before any form-cancel logic runs.
- `confirmingDiscard` is only ever set from a dirty form; a clean form skips it.
