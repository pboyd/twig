# UI Contract: Planning Forms (parity with Task form)

This is the interface contract for the TUI planning forms after this feature. It is the source of truth that the implementation and tests must conform to. No network/proto contract changes are involved; the "interface" here is the rendered form and its key bindings.

## Scope

Applies to all three planning-form modes rendered by `renderPlanFormView`:

| Mode | Title | Fields |
|---|---|---|
| `planTaskTime` | `Schedule task` | Start, Duration |
| `planEventForm` | `Add event` | Name, Start, Duration |
| `planEdit` | `Edit entry` | Name, Start, Duration (pre-filled) |

## Rendered layout (contract)

```
<Title>
<blank line>
<label>: <field 0 view>
<blank line>
<label>: <field 1 view>
<blank line>
[<label>: <field 2 view>]      # when present
<blank line>
[ Save ]  [ Cancel ]
<blank line>
Ctrl+S: save  Esc: cancel  Tab: next field
```

Requirements:
- **R1 (Title)** — The first line is the mode title; a blank line follows it. (FR-013)
- **R2 (Spacing)** — Exactly one blank line separates adjacent fields and separates the last field from the button row. (FR-003)
- **R3 (Buttons)** — A `[ Save ]` and a `[ Cancel ]` button render below the fields. The focused button renders as `[>Save<]` / `[>Cancel<]`; unfocused as `[ Save ]` / `[ Cancel ]`. (FR-004)
- **R4 (Help line)** — The final line is exactly `Ctrl+S: save  Esc: cancel  Tab: next field` (two spaces between groups), preceded by a blank line. Character-for-character identical to the task form. (FR-008, SC-005)
- **R5 (No sentinel hints)** — No field label or placeholder contains `blank=keep`, `null=unschedule`, or similar. (FR-002)
- **R6 (Focused-field marker)** — The focused text field uses the same focus marker convention as the rest of the planning UI (leading `> ` / indentation), consistent with the existing renderer.

## Key bindings (contract)

| Key | Focus context | Action |
|---|---|---|
| `Ctrl+S` (`keys.Save`) | any | Submit the form |
| `Esc` (`keys.Cancel`) | any | Cancel; return to plan list, discard changes (FR-010) |
| `Tab` / `Shift+Tab` | any | Move focus forward/back across fields **and** Save/Cancel buttons, wrapping (FR-005) |
| `Enter` | text field | Advance focus to next field — **does NOT submit** (FR-006) |
| `Enter` | Save button | Submit the form (FR-007) |
| `Enter` | Cancel button | Cancel (FR-010) |
| other | text field | Forwarded to the focused textinput |

## Edit-form pre-fill (contract — `planEdit` only)

| Field | Source | Format | Empty when |
|---|---|---|---|
| Name | `entry.Name` | verbatim | never (entries always have a name) |
| Start | `entry.StartMinute` | `HH:MM`, zero-padded 24h (e.g. `09:00`) | entry is untimed (`StartMinute == nil`) |
| Duration | `entry.DurationMinute` | compact unit (e.g. `30m`, `1h30m`) | untimed or zero duration |

- **C1** — Pre-filled values MUST re-parse without modification via `timeparse.ParseStart` / `timeparse.ParseDuration`. (FR-014)

## Submit semantics (contract — `planEdit`)

Given trimmed field values `name`, `start`, `dur` and the original `entry`:

| Condition | Result |
|---|---|
| `name == ""` | Validation error `name cannot be empty`; form stays open |
| `name != entry.Name` | Dispatch rename |
| `start == ""` and entry was scheduled | Dispatch **unschedule** (move to untimed); `dur` ignored |
| `start == ""` and entry already untimed | No move |
| `start` present, unparseable | Validation error; form stays open |
| `dur` present, unparseable | Validation error; form stays open |
| `start`/`dur` parse and differ from entry's current values | Dispatch move (scheduled) |
| `start`/`dur` unchanged from entry | No move |
| No rename and no move produced | Close form (true no-op) (FR-014) |

Cancelling at any point dispatches nothing and returns to the plan list. (FR-010)

## Out of scope

- Schedule-task (`planTaskTime`) and add-event (`planEventForm`) **submit** semantics are unchanged by this feature; only their **presentation** (R1–R4) and Enter-handling change.
- No backend, proto, or DB changes.
