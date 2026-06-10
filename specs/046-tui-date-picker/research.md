# Research: TUI Calendar Date Picker

**Feature**: 046-tui-date-picker | **Date**: 2026-06-10

All Technical Context unknowns are resolved below. No NEEDS CLARIFICATION markers remain.

## R1. Calendar widget: build vs. dependency

**Decision**: Build a small custom `calendarModel` component inside `internal/tui` (stdlib `time` + existing lipgloss theme).

**Rationale**:
- The project is on the **bubbletea v2 module line** (`charm.land/bubbletea/v2`, `charm.land/bubbles/v2`). `charm.land/bubbles/v2` v2.1.0 ships no datepicker component.
- The known third-party option, `ethanefung/bubble-datepicker`, targets bubbletea **v1** (`github.com/charmbracelet/bubbletea` import path, v1 message types). It cannot be used against v2's `tea.KeyPressMsg`/`charm.land` module graph without forking and porting — more work than writing the widget, plus a permanent dependency to maintain.
- A month grid is small, pure logic: stdlib `time.Date` normalization gives day/week/month/year arithmetic and clamping for free. Estimated ~200 LOC including rendering.
- A custom widget can use the shared palette in `theme.go` directly, satisfying Principle III (UI/UX Consistency) without adapter code; a third-party widget would need restyling anyway.
- Principle I (Simplicity/YAGNI) favors the dependency-free option when effort is comparable.

**Alternatives considered**:
- `ethanefung/bubble-datepicker` — rejected: bubbletea v1 only; would require fork + port + restyle.
- `charmbracelet/huh` form library — rejected: no calendar/date picker field; would also replace the whole hand-rolled form, far beyond this feature's scope.
- Pure text improvements (date validation hints, auto-complete) — rejected: doesn't deliver the requested calendar UX.

## R2. How the calendar attaches to the form

**Decision**: The calendar is an **inline overlay state of the edit form**, not a new top-level TUI mode. `editFormModel` gains an optional `calendar *calendarModel` (open/closed) plus the index of the field it serves. While open, the form routes all key messages to the calendar first.

**Rationale**:
- The form already owns modal-ish behavior (focus cycling, `enter` semantics per field). Keeping the calendar inside the form avoids touching the top-level `mode` switch in `model.go`/`update.go` and keeps `esc`/`enter` scoping local: when the calendar is open, `esc` closes the calendar (field unchanged), not the form — exactly FR-006 and the edge case in the spec.
- Mirrors the existing pattern of `editorModel` (ctrl+g external editor for the description field): a per-field helper invoked from the form.

**Alternatives considered**:
- New top-level `modeCalendar` — rejected: leaks form-internal concerns into the global update loop; more wiring for no benefit.
- Replace the textinput with a composite date widget — rejected: violates FR-007 (text entry must remain exactly as-is) and is a bigger change.

## R3. Key bindings

**Decision**:

| Key | Action | Consistency anchor |
|---|---|---|
| `ctrl+g` (on Due/Snooze field) | Open calendar | Same "summon the field's helper" key as the description field's $EDITOR binding |
| `↑/↓/←/→` and `k/j/h/l` | Move selection by week (up/down) / day (left/right) | Tree navigation uses the same arrows + vim keys |
| `[` / `]` | Previous / next month | Plan tab uses `[`/`]` for previous/next day |
| `{` / `}` | Previous / next year | Shift-variant of `[`/`]` |
| `t` | Jump to today | Plan tab uses `.` for today; `t` chosen because `.` is harder to discover and `t` is free in this context |
| `enter` | Confirm selection, close calendar, write date to field | Universal confirm |
| `esc` | Close calendar, leave field unchanged | Universal cancel (scoped: does not cancel the form while calendar is open) |

A new `Calendar` binding is added to `KeyMap` (`ctrl+g`, help text "pick a date"). The in-calendar keys are handled directly by `calendarModel` (they are modal and conflict-free because the calendar swallows all keys while open). `tab`/`shift+tab` while the calendar is open close it (unchanged field) and cycle focus as usual, so the form never feels trapped.

**Rationale**: Every binding reuses an existing muscle-memory pattern from the app (Principle III). `ctrl+g` is already "open the helper for this field" — overloading it per-field keeps the form's footer help short.

**Alternatives considered**:
- `pgup`/`pgdn` for months — kept as undocumented aliases? Rejected for v1: `[`/`]` already match the plan tab; fewer bindings to document (YAGNI).
- Auto-opening the calendar on field focus — rejected: violates FR-007/US3 scenario 3 (calendar must not open uninvited).

## R4. Date semantics: parsing, time-of-day preservation, output format

**Decision**:
- **Opening**: parse the field's current value with `cli.ParseDue` (the existing single source of truth: RFC3339 or `YYYY-MM-DD`). Valid → calendar opens on that date (UTC). Invalid/empty → opens on today.
- **Confirming on Snooze**: write `YYYY-MM-DD`.
- **Confirming on Due**: if the prior field value parsed as RFC3339 **with a non-midnight time or explicit RFC3339 form**, write RFC3339 with the prior clock time (UTC) and the newly picked date (FR-008). Otherwise write `YYYY-MM-DD` (midnight UTC default — identical to current typed behavior).
- **Clearing**: text deletion in the field remains the way to unset a date (FR-009); the calendar adds no separate clear action in v1 (YAGNI — revisit if users ask).

**Rationale**: Reuses `cli.ParseDue` so the calendar can never produce a value the save path rejects (SC-003). Time preservation is a pure string/`time.Time` recombination, no new parsing rules.

**Alternatives considered**:
- Time-of-day selection in the widget — rejected: spec assumption explicitly keeps time a typed concern.
- A dedicated clear key in the calendar — deferred: FR-009 is already satisfied by text deletion.

## R5. Rendering & small terminals

**Decision**: Render the calendar directly beneath its field inside the existing form panel: a header line (`◀ June 2026 ▶` style), a weekday row (`Su Mo Tu We Th Fr Sa`, Sunday-first per Go's `time.Weekday` ordering), and 4–6 week rows. Width is fixed at ~22 columns (7 × 3 − 1), well under the form's minimum usable width (`fieldWidth` floor is 20, panel adds margins), so no special small-terminal handling is needed beyond what the form already does. Styling: today underlined/dim-accented, selected day rendered with the shared `highlightStyle`, adjacent-month padding days in `dim`.

**Rationale**: Inline (vs. floating overlay) needs no z-ordering or absolute positioning — lipgloss composition stays simple. The fixed 22-column footprint fits every layout the form already supports.

**Alternatives considered**:
- Floating overlay centered on screen — rejected: requires layering/compositing machinery the TUI doesn't have; inline is simpler and keeps spatial association with the field.

## R6. Testing approach

**Decision**: Table-driven unit tests in `calendar_test.go` for: month navigation incl. clamping (Jan 31 → Feb 28/29, leap-day year jumps), week/day movement across month boundaries, open-from-value (valid date / RFC3339 / garbage / empty), confirm output strings for both field kinds (incl. time preservation), and a render smoke test (selected day highlighted, correct weekday alignment). Form-level tests in `edit_test.go` drive `tea.KeyPressMsg` sequences: open → navigate → confirm writes the field; open → esc leaves it untouched; calendar swallows form keys while open. Follows the existing no-database, `export_test.go`-shim conventions.

**Rationale**: Matches the package's established test style; calendar logic is pure functions of (date, key) → date, ideal for tables.
