# Phase 0 Research: Plan Entry Form Parity

All spec-level ambiguities were resolved during `/speckit-clarify` (see spec `## Clarifications`). This document records the **implementation** design decisions needed to realize that spec against the existing TUI code.

## Reference design: the task edit form

**Decision**: Mirror `editFormModel` in `internal/tui/edit.go` as the parity target.

Observed conventions in the task form (lines referenced from `edit.go`):
- Title line + blank line: `title + "\n\n"` (e.g. `Edit Task #5`).
- One blank line between fields: each field block ends with `"\n\n"`.
- Buttons: `[ Save ]  [ Cancel ]`, with focused button shown as `[>Save<]` / `[>Cancel<]`.
- Help line, preceded by a blank line: `"\nCtrl+S: save  Esc: cancel  Tab: next field"`.
- Focus model: integer `focusIndex` over constants `focusName…focusCancel`, `focusCount` total; `cycleFocus(delta)` wraps with `% focusCount`; only text fields call `Focus()`, buttons are "focused" purely by index.
- Key handling (`edit.go` Update): `keys.Save` → save; `keys.Cancel` → cancel; `keys.Tab`/`keys.ShiftTab` → cycle; `Enter` → activate Save/Cancel when a button is focused, else advance focus on single-line fields. **Enter never saves from a text field.**

**Rationale**: The spec explicitly names the task form as the superior reference ("far and away better"); copying its exact conventions guarantees parity and satisfies Constitution Principle III.

## Adding Save/Cancel to the slice-based plan form

**Decision**: Keep `planFormState{ fields []textinput.Model; focus int; … }` and treat focus indices `len(fields)` and `len(fields)+1` as the virtual **Save** and **Cancel** slots. Total focus positions = `len(fields) + 2`.

- `cyclePlanFormFocus(delta)` wraps over `len(fields)+2`. When the new focus is a text-field index, call `Focus()` on it and `Blur()` the others; when it is a button slot, `Blur()` all fields (no textinput focused).
- Helper predicates (e.g. `planFocusSave(form)` / `planFocusCancel(form)`) compare `form.focus` against `len(fields)` / `len(fields)+1` to keep `update.go` and `plan_view.go` readable.

**Alternatives considered**:
- *Replace `planFormState` with a struct mirroring `editFormModel` (named fields + focus constants).* Rejected (Principle I / YAGNI): the plan forms have a **variable** field count (2 for schedule, 3 for add-event/edit), so fixed focus constants don't fit. The slice + offset approach is the minimal change.
- *Reuse `editFormModel` directly for plan forms.* Rejected: it is wired to task semantics (Description textarea, Due/Estimate, `editSavedMsg` carrying task fields) and would require heavy adaptation for little gain.

## Edit-form pre-fill and value formats

**Decision**:
- Name: `SetValue(entry.Name)` (already done).
- Start: when `entry.StartMinute != nil`, `SetValue(fmt.Sprintf("%02d:%02d", min/60, min%60))` → `HH:MM`. When nil (untimed), leave empty.
- Duration: when scheduled with a positive `DurationMinute`, `SetValue` a compact unit string (e.g. `30m`, `1h30m`). When nil/zero, leave empty.
- Placeholders become neutral hints with no sentinel text: Start `"e.g. 09:00"`, Duration `"e.g. 30m"`. `planFieldLabel(planEdit, …)` drops `blank=keep, null=unschedule` → plain `Start` / `Duration`.

**Round-trip requirement**: `HH:MM` is accepted by `timeparse.ParseStart`; compact `30m`/`1h30m` is accepted by `timeparse.ParseDuration`. A compact formatter (`minutes → "30m"/"1h30m"`) is needed for Duration. Check for an existing helper in `internal/cli/timeparse` before adding one; if absent, add a small `FormatDuration(minutes int) string` next to the parser and unit-test it (parse∘format == identity for representative values).

**Rationale**: Satisfies FR-014 and SC-002 (unchanged save is a no-op, values display in the same notation the user would type).

## Submit semantics with pre-filled values (the critical change)

**Decision**: Rewrite `submitEditForm` around **change detection against the original entry**, replacing the old blank/`null` sentinel logic:

1. Trim Name; empty → validation error "name cannot be empty" (unchanged).
2. If `nameStr != originalName` → dispatch `renamePlanCmd`.
3. Start/Duration handling:
   - **Start cleared** (`startStr == ""`) **and entry was scheduled** → dispatch unschedule (`movePlanCmd(..., 0, 0, scheduled=false)`); **ignore Duration** (per clarification).
   - **Start cleared and entry already untimed** → no move.
   - **Start present** → parse start; parse duration if present; if the resulting (start, duration) differ from the entry's current (start, duration), dispatch `movePlanCmd(..., start, dur, scheduled=true)`; otherwise no move.
   - Invalid start or duration → validation error, form stays open.
4. If no commands were produced → close the form immediately (existing no-op path).

**Rationale**: With values pre-filled, an unchanged save would otherwise re-issue a redundant move every time. Change detection makes unchanged saves true no-ops (FR-014, no-op edge case) and gives clearing Start a single, clear meaning: unschedule (FR-009). The `null` keyword sentinel is removed entirely — clearing the field replaces it.

## Removing Enter-to-save

**Decision**: In `handlePlanFormKey` (`update.go`), drop `|| msg.Type == tea.KeyEnter` from the Save case. Add explicit Enter handling that mirrors the task form:
- Focus on Save slot → `submitPlanForm()`.
- Focus on Cancel slot → cancel (return to `planList`).
- Focus on a text field → `cyclePlanFormFocus(1)` (advance), **not** save.

`keys.Save` (Ctrl+S) continues to submit from any focus.

**Rationale**: FR-006/FR-007. Mirrors `edit.go` Enter handling exactly.

## Title rendering

**Decision**: In `renderPlanFormView`, stop discarding the already-computed `title` (`_ = title`) and emit `title + "\n\n"` ahead of the fields, exactly like the task form. Titles: "Edit entry" / "Schedule task" / "Add event" (already produced by the existing switch).

**Rationale**: FR-013; the value is already computed, so this is a one-line un-suppression.

## Test impact

**Decision**: Update TUI tests rather than add a parallel suite.
- `plan_update_test.go`: edit-form tests that assumed "Start/Duration blank = no move" must account for pre-filled Start; assert unschedule-on-clear and no-op-on-unchanged; remove any `null`-keyword expectations.
- `plan_view_test.go`: add render assertions for title line, `[ Save ]`/`[ Cancel ]`, and the exact help line `Ctrl+S: save  Esc: cancel  Tab: next field`; update any assertion of the old `[tab] next field  [ctrl+s/enter] save  [esc] cancel` line.
- `plan_us2_test.go`: schedule-form (`planTaskTime`) label/optional-start behavior is unchanged; confirm still green, adjust only if focus-count assumptions break.

**Rationale**: Tests are the executable spec for this behavior; TDD per the project's testing conventions (string/table tests, no DB).
