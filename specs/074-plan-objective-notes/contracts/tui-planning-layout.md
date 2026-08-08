# Contract: TUI planning tab layout and keys

**Feature**: 074-plan-objective-notes | **Date**: 2026-08-06

The interaction contract for the planning tab after this feature. It governs
`internal/tui/view.go`, `internal/tui/plan_view.go`, `internal/tui/plan_update.go`,
`internal/tui/keymap.go`, and `internal/tui/model.go`.

## Layout

Today the planning tab is a tab bar, one two-column row (grid | details-or-form), and a
status line. This feature adds a full-width band above the row and splits the right column.

```
┌──────────────────────────────────────────────────────────────┐
│ Goals │ Tasks │ Planning │ Report                            │  tab bar
├──────────────────────────────────────────────────────────────┤
│ Objective ─────────────────────────────────────────────────  │  full width,
│   Ship the plan-day migration                                │  OMITTED when unset
├───────────────────────────────┬──────────────────────────────┤
│ Thu Aug 6, 2026 (today) ───── │ Details ──────────────────── │
│   09:00  Ship migration       │   Ship migration             │
│   10:30  Standup              │   09:00–10:00 · 60m          │
│                               ├──────────────────────────────┤
│                               │ Notes ─────────────────────  │  always present
│                               │   Blocked on review          │
├───────────────────────────────┴──────────────────────────────┤
│ status                                                       │
└──────────────────────────────────────────────────────────────┘
```

### Objective band

| Rule | |
|---|---|
| Shown when | The displayed day's objective is non-empty, **or** `mode == planObjectiveEdit` |
| Omitted when | Objective is empty and no objective edit is in progress — zero height, no border, no reserved space (FR-007) |
| Width | Full terminal width |
| Position | Between the tab bar and the grid/right row |
| Inner height | Rendered content height, capped at 3 lines |
| Content | `m.md.Render(objective, markdown.Options{Width: inner, Styled: m.styled})` |
| Title (styled) | `Objective` |
| Effect on the row below | Its total height is subtracted from `innerH` before the grid and right panes are sized |

When omitted, the tab renders byte-identically to today (SC-003).

### Right column split

| Rule | |
|---|---|
| Top | Details pane — unchanged content, reduced height |
| Bottom | Notes pane |
| Split | `detailsInner = ceil(h/2)`, `notesInner = h - detailsInner - borders`; details floored at 3 inner lines |
| Notes when empty | Pane still rendered, content empty (FR-013) |
| Content | `m.md.Render(notes, …)`, wrapped to the pane's inner width |
| Overflow | Clipped to the pane; never spills (FR-020, FR-021) |
| While `mode == planNotesEdit` | The **whole** right column becomes the notes editor; the details pane is not shown |
| While a picker or entry form is open | Unchanged from today — the right column is the picker/form, no notes pane |

Both the styled and unstyled (`m.styled == false`) branches of `viewPlanning` implement
these rules; the unstyled branch uses its existing row-join instead of `paneBox`.

## Modes

`planMode` (`internal/tui/model.go:81`) gains:

| Mode | Entered by | Leaves via |
|---|---|---|
| `planObjectiveEdit` | `o` while `planList` | Enter (save) or Esc (cancel) |
| `planNotesEdit` | `n` while `planList` | `ctrl+s` (save) or Esc (cancel) |

## Keys

| Key | Context | Action |
|---|---|---|
| `o` | Planning tab, `mode == planList` | Open the objective editor, pre-filled with the day's objective |
| `n` | Planning tab, `mode == planList` | Open the notes editor, pre-filled with the day's notes |
| `enter` | `planObjectiveEdit` | Save and close |
| `esc` | `planObjectiveEdit` | Discard and close |
| `ctrl+s` | `planNotesEdit` | Save and close |
| `enter` | `planNotesEdit` | Insert a newline — does **not** save (FR-015) |
| `esc` | `planNotesEdit` | Discard and close |
| `ctrl+g` | `planNotesEdit` | Hand the draft to `$EDITOR` via the existing `openEditorCmd`; on exit the text returns to the editor, still unsaved |

Neither editor may be opened while the other, a picker, or an entry form is open — the
`planList` guard is the whole mechanism.

`o` and `n` are added to the planning tab's help under the existing `k.PlanningMode` gate.
`n` on the Tasks tab still creates a subtask; `o` remains unbound elsewhere (FR-023).

## Data flow

| Event | Behaviour |
|---|---|
| Day loads or changes | `ListPlanEntries` returns entries plus `PlanDay`; both panes show the loaded day's values (FR-022) |
| Background auto-refresh (`bg`) | Applies new values only when the response's day matches the displayed day **and** no objective/notes editor is open — an open draft is never overwritten |
| Save succeeds | `plan.objective` / `plan.notes` updated from the response, mode returns to `planList`, pane reflects it immediately (FR-011, FR-017) |
| Save fails | Error shown via the planning tab's existing error presentation; the editor stays open with the text intact |
| Editor cancelled | Stored value is re-displayed unchanged |
| `ctrl+g` fails or `$EDITOR` exits non-zero | Warm, actionable error; the draft is preserved and the editor stays open |

## Tone

All new user-facing text follows Principle IV: warm, accurate, actionable. Examples of the
register (final wording is an implementation choice):

- Objective saved: `Objective set — that's the one that matters today.`
- Notes saved: `Notes tucked away.`
- Editor failure: `Couldn't open the editor — your notes are safe, though!` (mirrors the
  existing message at `internal/tui/update.go:1045`)

## Non-goals

- No scrolling inside either pane; both show what fits.
- No editing of either field from the Tasks, Goals, or Report tabs.
- No web app surface.
