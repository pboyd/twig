# UI Contract: Calendar Date Picker Widget

**Feature**: 046-tui-date-picker | **Date**: 2026-06-10

This feature has no API/protobuf surface. The external contract is the widget's user-facing
behavior: key bindings, visual states, and the strings it writes into the form's date fields.
Implementation MUST conform to this contract (Constitution Principle II applied to a UI feature).

## Scope

Applies to the task edit form (`internal/tui/edit.go`) fields:

| Field | Accepts (unchanged) | Calendar writes |
|---|---|---|
| Due | `YYYY-MM-DD` or RFC3339 | `YYYY-MM-DD`; or RFC3339 preserving the field's prior clock time when the prior value was RFC3339 |
| Snooze until | `YYYY-MM-DD` | `YYYY-MM-DD` |

Out of scope: CLI flags, web frontend, plan-tab date prompt (`ctrl+p`), time-of-day selection.

## Key bindings

### Form level (calendar closed)

| Key | Context | Action |
|---|---|---|
| `ctrl+g` | Due or Snooze field focused | Open calendar on the field's parsed date, else today |
| any text key | date field focused | Edits field text exactly as before this feature (no calendar) |

### Calendar level (calendar open — swallows all keys)

| Key | Action |
|---|---|
| `←`/`h`, `→`/`l` | Selection −1 / +1 day (flows across month boundaries) |
| `↑`/`k`, `↓`/`j` | Selection −7 / +7 days |
| `[` / `]` | Previous / next month (day clamped to month length) |
| `{` / `}` | Previous / next year (Feb 29 clamps to Feb 28) |
| `t` | Jump selection to today |
| `enter` | Confirm: close calendar, write formatted date to the field |
| `esc` | Cancel: close calendar, field unchanged (does NOT cancel the form) |
| `tab` / `shift+tab` | Close calendar (field unchanged), then cycle field focus as usual |
| all other keys | Ignored (not forwarded to the field or form) |

## Visual contract

- Rendered inline, directly beneath the owning field, inside the existing form panel; fixed grid width ~22 columns.
- Header: month + year with prev/next affordances (e.g. `◀ June 2026 ▶`).
- Weekday row `Su Mo Tu We Th Fr Sa` (Sunday-first), then 4–6 week rows.
- Styling MUST use only the shared palette (`theme.go`): selected day uses `highlightStyle`; today is marked distinctly (accent); out-of-month padding cells use `dim`. No ad-hoc colors (Principle III).
- A one-line hint appears under a focused date field when the calendar is closed (mirrors the description field's `ctrl+g: open editor` hint) and a key-hint line while open.

## Copy (Principle IV — playful tone)

| Surface | Text |
|---|---|
| Hint under focused date field (closed) | `ctrl+g: summon the calendar` |
| Hint line while open | `enter: pick  esc: never mind  t: today  [/]: month  {/}: year` |

Final wording may be tuned during implementation but MUST keep the warm, lightly playful register and MUST stay accurate.

## Behavioral guarantees

1. **Round-trip safety**: every string the calendar writes MUST be accepted by `cli.ParseDue` (verified by test). No calendar selection can produce a save-time format error.
2. **No uninvited opening**: the calendar opens only on `ctrl+g`; typing into date fields never triggers it.
3. **Cancellation is lossless**: `esc`/`tab`/`shift+tab` from an open calendar leave the field byte-for-byte unchanged.
4. **Time preservation**: confirming on a Due field whose prior value was RFC3339 keeps the prior clock time (UTC), changing only the date.
5. **Always-valid selection**: navigation can never land on an invalid date; month/year jumps clamp.
6. **Past dates allowed**: no lower bound on selectable dates.
7. **Text path untouched**: with the calendar closed, all key handling on date fields is byte-identical to pre-feature behavior.
