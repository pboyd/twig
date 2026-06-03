# Phase 1 Data Model: Plan Entry Form Parity

This feature introduces **no domain/persistence model changes** — no proto messages, DB tables, or stored fields are added or altered. The only "model" affected is the in-memory TUI form state. It is documented here for implementation clarity.

## Plan entry (existing, unchanged)

`planv1.PlanEntry` as already defined. Fields relevant to the form:

| Field | Type | Meaning in form |
|---|---|---|
| `Id` | `int32` | Identifies the entry being edited (`planFormState.entryID`). |
| `Name` | `string` | Pre-fills the Name field. |
| `StartMinute` | `*int32` (nullable) | `nil` → untimed (Start field empty). Non-nil → minutes since midnight; pre-fills Start as `HH:MM`. |
| `DurationMinute` | `int32` | When scheduled and `> 0`, pre-fills Duration as compact unit string (`30m`, `1h30m`). |

## Form state: `planFormState` (existing struct, extended behavior)

Location: `internal/tui/model.go`.

```go
type planFormState struct {
    fields  []textinput.Model // 2 fields (schedule) or 3 (add-event, edit)
    focus   int               // 0..len(fields)+1  (see focus model below)
    taskID  int64             // schedule-task mode
    entryID int32             // edit mode
}
```

**No struct field changes are required.** The change is to the **interpretation of `focus`**: it now ranges over `len(fields) + 2` positions, where the last two are the virtual Save and Cancel buttons.

### Focus model

For a form with `n = len(fields)` text inputs:

| `focus` value | Target | Has textinput focus? |
|---|---|---|
| `0 .. n-1` | Text field at that index | Yes (that field) |
| `n` | **Save** button | No |
| `n+1` | **Cancel** button | No |

- Cycling: `focus = (focus + delta + (n+2)) % (n+2)`.
- Predicates: `focus == n` → Save focused; `focus == n+1` → Cancel focused.
- Only one textinput is `Focus()`ed at a time; when a button slot is active, all fields are `Blur()`ed.

### Per-mode field layout

| Mode | `fields` | Pre-filled? |
|---|---|---|
| `planTaskTime` (Schedule task) | `[Start, Duration]` | No (new scheduling) |
| `planEventForm` (Add event) | `[Name, Start, Duration]` | No (new event) |
| `planEdit` (Edit entry) | `[Name, Start, Duration]` | Yes — from the selected entry |

## Validation rules (from spec)

| Rule | Source | Behavior on violation |
|---|---|---|
| Name (edit/add-event) must be non-empty | FR-011 | Validation error; form stays open |
| Start must parse (`HH:MM` / accepted formats) when present | FR-011 | Validation error; form stays open |
| Duration must parse (`30m`, `1h30m`, …) when present | FR-011 | Validation error; form stays open |
| Clearing Start unschedules; Duration ignored | FR-009 | Entry moved to untimed |
| Unchanged form save | FR-014 | No rename/move dispatched; form closes |

## State transitions (edit mode)

```
planList ──Enter on entry──▶ planEdit (fields pre-filled, focus=0 / Name)
planEdit ──Tab/Shift-Tab/Enter-on-field──▶ planEdit (focus cycles: fields → Save → Cancel → wrap)
planEdit ──Ctrl+S or Enter-on-Save──▶ submit:
            ├─ invalid          → planEdit (error shown)
            ├─ no change        → planList (no command)
            ├─ name changed     → rename dispatched → planList
            ├─ start cleared    → unschedule dispatched (duration ignored) → planList
            └─ start/dur changed→ move dispatched → planList
planEdit ──Esc or Enter-on-Cancel──▶ planList (no change)
```
