# Phase 0 Research: Esc Cancels TUI Forms

This feature is a localized TUI interaction change. There were no open
`NEEDS CLARIFICATION` markers in the spec; this document records the design
decisions reached by reading the current `internal/tui` code so implementation
can proceed without surprises.

## Current behavior (as built today)

- **Task / Goal edit form** (`editFormModel`, `internal/tui/edit.go`): used for
  `modeEdit`/`modeNewSubtask`/`modeNewRoot` (Tasks tab) and `goalEdit`/`goalNew`/
  `goalNewTask` (Goals tab). `Esc` is **not** handled when the calendar is closed —
  it falls through to the focused text input and does nothing. Cancel is only
  reachable via the on-screen `[ Cancel ]` button (Enter on it emits
  `editCancelledMsg`). When the calendar date picker is open, `Esc` closes the
  calendar (existing nested-overlay behavior in `editFormModel.Update`).
- **Plan forms** (`planFormState`, `planTaskTime`/`planEventForm`/`planEdit`):
  handled by `handlePlanFormKey`. `Esc` is **not** handled — it falls through to the
  focused input. Cancel is only reachable via the on-screen Cancel button.
- **Selection-only overlays** already cancel immediately on `Esc` via
  `keys.Cancel`: move-target picker (`handleMoveKey`), task picker
  (`handlePickerKey`), goal link/unlink pickers, and the date prompt
  (`handleDatePromptKey`).
- **Existing confirmation overlays**: quit-while-pom-running (`confirmingQuit`)
  and several `[y]es [n]o` prompts (goal delete, status-update delete) follow a
  consistent `y` / `n`-or-`esc` pattern, rendered in `view.go`/`renderStatus`.

**Conclusion**: the real gaps are the two multi-field text forms
(`editFormModel`, `planFormState`), which today have no `Esc` cancel at all.

## Decision 1 — Definition of "unsaved input" (dirty check)

**Decision**: Snapshot each form's field values at the moment it opens, then on
`Esc` compare the current values to that snapshot using whitespace-trimmed string
equality. The form is "dirty" if any field differs.

- `editFormModel`: snapshot `name`, `description`, `due`, `pomodoroEstimate`,
  `snooze`, and the goal selector index (`goalIdx`). Add an `isDirty()` method.
- `planFormState`: snapshot the `[]textinput.Model` field values; compare current
  field values.

**Rationale**: Matches the spec's interpretation ("differs from opened state"),
so a pre-filled edit form opened-and-closed unchanged cancels with no prompt, and
reverting a field to its original value clears the dirty state (FR-009). Trimming
satisfies the whitespace-only rule (FR-010). Snapshotting at open is trivial and
robust versus trying to track keystrokes.

**Alternatives considered**:
- *Track a "touched" flag on any keypress*: rejected — would prompt even after the
  user reverts a change, violating FR-009.
- *Diff against the persisted record*: rejected — the edit form already loads from
  the record at open, so the open-time snapshot is equivalent and simpler, and it
  also covers brand-new (blank) forms uniformly (empty snapshot).

## Decision 2 — Confirmation overlay mechanism

**Decision**: Add a single model-level `confirmingDiscard bool`. When a dirty form
receives `Esc`, set `confirmingDiscard = true` instead of canceling. Render a modal
discard prompt (mirroring the `confirmingQuit` rendering) over the current form.
Intercept keys for it at the **top of `handleKey`** (before tab routing) so it is
modal regardless of active tab: `y` performs the cancel, `n`/`esc` dismisses the
prompt and returns to the form unchanged.

**Cancel routing at confirm time** (no extra state needed): if the Planning tab is
active and `plan.mode` is a form mode, reset `plan.mode = planList`; otherwise the
edit form is active, so reuse the existing `editCancelledMsg` path (its handler
already distinguishes Tasks vs Goals tab and restores the prior cursor).

**Rationale**: Reuses the established `confirmingQuit` pattern and the existing
`editCancelledMsg` cancel logic (Principle I/III). One boolean + one render block +
one interception point keeps it minimal and uniform.

**Alternatives considered**:
- *Per-form confirmation flags*: rejected as redundant duplication.
- *Handle the prompt inside each tab's key handler (like `confirmingQuit` is)*:
  rejected — centralizing in `handleKey` avoids duplicating the y/n/esc block
  across `handleListKey`, `handleGoalsKey`, and `handlePlanningKey`.

## Decision 3 — Where `Esc` is detected per form

**Decision**:
- `editFormModel.Update`: when the calendar is **closed**, add an `Esc` case that
  emits `editCancelledMsg` if `!isDirty()`, or a new `editDiscardRequestedMsg`
  (carrying `originalCursor`) if dirty. The existing calendar-open `Esc` branch is
  untouched, preserving nested-overlay layering (FR-008). The Model turns
  `editDiscardRequestedMsg` into `confirmingDiscard = true`.
- `handlePlanFormKey`: add an `Esc`/`keys.Cancel` case — if the plan form is dirty,
  set `confirmingDiscard = true`; otherwise reset to `planList` immediately.

**Rationale**: Keeping the calendar-vs-form `Esc` decision inside
`editFormModel.Update` preserves the existing layered behavior in one place. Plan
forms have no nested overlay, so handling `Esc` directly in `handlePlanFormKey` is
simplest.

## Decision 4 — Status-update compose is the external `$EDITOR` (scope clarification)

**Decision**: The status-update compose flow (`s` on a goal; `e` on a history item)
opens the user's external `$EDITOR` via `openEditorCmd` (`internal/tui/editor.go`),
not an in-TUI form. Canceling/discarding there is the editor's own responsibility
(e.g. `:cq` / exiting without saving), and the result is already handled by
`handleGoalStatusEditorFinished`. Therefore there is **no in-TUI `Esc` form** to add
a guard to for status compose; it is out of scope for this change while still
honoring the spec's intent (the editor provides cancel/discard).

**Rationale**: The spec listed "status-update compose form," but in implementation
it is delegated to `$EDITOR`. Adding TUI-level interception is impossible while the
editor is foregrounded and would contradict the existing design. This is recorded
here as the reconciliation between spec wording and implementation reality.

## Decision 5 — Date prompt (`modeDatePrompt`) stays as-is

**Decision**: The single-field date prompt already cancels immediately on `Esc`.
Leave it unchanged (no dirty-guard). It is not in the spec's enumerated form list,
and its single trivial field does not warrant a confirmation step.

**Rationale**: YAGNI (Principle I) and matches the spec's explicit scope. Pickers
and the move overlay likewise keep their immediate-cancel behavior (FR-011).

## Decision 6 — Playful discard copy (Principle IV)

**Decision**: Use a warm-but-measured prompt, e.g.
`Toss out your unsaved edits? [y]es  [n]o — keep editing`. Final wording lives in
`contracts/keyboard-interaction.md`; it must stay accurate and actionable.

**Rationale**: Destructive-action confirmations should be warm but not flippant
(Principle IV guidance), consistent with the existing quit/delete prompts.

## Testing approach

Follow existing TUI test conventions (table-driven `Update` tests with synthesized
`tea.KeyPressMsg`, `export_test.go` shims):
- `edit_test.go`: snapshot/`isDirty` for blank vs pre-filled forms; revert-to-original
  clears dirty; whitespace-only is clean; `Esc` on clean form emits `editCancelledMsg`;
  `Esc` on dirty form emits `editDiscardRequestedMsg`; `Esc` with calendar open closes
  the calendar (regression).
- `update_test.go`: `confirmingDiscard` overlay — `y` cancels (returns to list,
  cursor restored, nothing saved), `n`/`esc` returns to form with input intact;
  verified for Tasks edit, Goals edit, and a plan form.
- `plan_update_test.go`: plan form clean `Esc` → `planList`; dirty `Esc` →
  `confirmingDiscard`.
