# Data Model: TUI Calendar Date Picker

**Feature**: 046-tui-date-picker | **Date**: 2026-06-10

No persisted data changes. All entities below are in-memory TUI state in `internal/tui`.

## calendarModel (new)

The transient state of one open calendar widget.

| Field | Type | Meaning |
|---|---|---|
| `selected` | `time.Time` | Currently highlighted day (always a valid date, normalized to midnight UTC). Doubles as the viewed month: the grid always shows `selected`'s month. |
| `today` | `time.Time` | Cached "today" (midnight UTC) captured at open time, used for the today marker. |
| `keepTime` | `time.Duration` | Clock-time offset to re-apply on confirm (Due field with an existing RFC3339 time). Zero for date-only values and the Snooze field. |
| `rfc3339Out` | `bool` | Whether confirm emits RFC3339 (`true` when the field's prior value was RFC3339 with a time) or `YYYY-MM-DD`. |

### Invariants

- `selected` is always valid: every mutation goes through `time.Date(...)` normalization with explicit clamping for month/year jumps (no Feb 30, no Dec 32).
- The widget never holds an "empty" state — if the field is empty or unparseable, `selected` initializes to `today`.
- Month/year jumps clamp the day-of-month to the target month's length (Jan 31 `]`→ Feb 28/29; Feb 29 `}`→ Feb 28 on non-leap years).

### State transitions

```
closed ──(ctrl+g on Due/Snooze field)──▶ open(selected = parsed field value | today)
open ──(arrows/hjkl)──▶ open(selected ± 1 day | ± 7 days)
open ──([ / ] / { / })──▶ open(selected ± 1 month | ± 1 year, day clamped)
open ──(t)──▶ open(selected = today)
open ──(enter)──▶ closed; field.SetValue(formatted selected)
open ──(esc | tab | shift+tab)──▶ closed; field unchanged
```

## editFormModel (modified)

| Field | Type | Meaning |
|---|---|---|
| `calendar` | `*calendarModel` | Nil when closed. Non-nil means the calendar is open and intercepts all key messages. |
| `calendarFor` | `int` | Focus index the calendar serves (`focusDue` or `focusSnooze`); determines output format rules and which field receives the confirmed value. |

All existing fields are unchanged. `editSavedMsg` / `editCancelledMsg` are unchanged — the calendar only mutates the textinput values before save, so the save path (`cli.ParseDue` validation in `update.go`) is untouched.

## KeyMap (modified)

| Field | Keys | Help |
|---|---|---|
| `Calendar` | `ctrl+g` | "pick a date" (shown when a date field is focused, mirroring the description field's "open editor" hint) |

In-calendar movement keys are internal to `calendarModel` (modal; not part of the global `KeyMap`).

## Relationships

```
editFormModel 1 ──── 0..1 calendarModel     (owns; created on open, discarded on close)
calendarModel ──── reads ──── cli.ParseDue  (single source of date-format truth)
calendarModel ──── styles ──── theme.go palette (accent, dim, highlightStyle)
```

## Validation rules (from spec FRs)

- FR-003: open-from-field parsing falls back to today on any parse failure — never an error state.
- FR-005/SC-003: confirm output is restricted to the two formats `cli.ParseDue` accepts; a unit test round-trips every output through `cli.ParseDue`.
- FR-008: `keepTime`/`rfc3339Out` preserve an existing Due time-of-day across date changes.
- FR-010: all month/year arithmetic clamps; property covered by table-driven tests.
