# Interaction Contract: Esc Cancels TUI Forms

This feature exposes no RPC/API surface. Its external contract is the keyboard
interaction in the TUI. This document is the source of truth for that contract and
MUST match the implementation (Constitution Quality Gate 1, adapted for a TUI
interaction contract).

## Scope: which surfaces are "forms"

Covered by the `Esc`-cancel + dirty-guard contract:

| Surface | Model | Modes |
|---|---|---|
| Task create / edit | `editFormModel` | `modeEdit`, `modeNewSubtask`, `modeNewRoot` |
| Goal create / edit | `editFormModel` | `goalNew`, `goalEdit` |
| Goal's new-task | `editFormModel` | `goalNewTask` |
| Plan timed task | `planFormState` | `planTaskTime` |
| Plan event | `planFormState` | `planEventForm` |
| Plan entry edit | `planFormState` | `planEdit` |

Explicitly **not** covered (cancel immediately on `Esc`, no confirmation — they
have no free-text input to lose, FR-011):

- Move-target picker (`modeMove`), task picker (`planPickTask`), goal link/unlink
  pickers (`goalPickLink`, `goalPickUnlink`).
- Date prompt (`modeDatePrompt`) — single trivial field; keeps current immediate
  cancel.
- Existing `[y]es/[n]o` confirmations (quit, goal delete, status delete).

**Status-update compose** opens the external `$EDITOR`; cancel/discard is the
editor's own (`:cq` / exit without save). No in-TUI `Esc` guard applies.

## Contract

### C1 — Esc on a clean form
**Given** a covered form whose current field values equal their values at open
(after trimming whitespace),
**When** `Esc` is pressed,
**Then** the form closes immediately, no create/update is performed, and focus
returns to the prior view with the previous cursor/selection restored.

### C2 — Esc on a dirty form
**Given** a covered form where at least one field differs (after trimming) from its
value at open,
**When** `Esc` is pressed,
**Then** a modal discard-confirmation prompt is shown and no input is discarded yet.

### C3 — Confirm discard
**Given** the discard prompt is shown,
**When** `y` is pressed,
**Then** the form closes, all unsaved input is discarded, nothing is saved, and
focus returns to the prior view with the previous cursor/selection restored.

### C4 — Decline discard
**Given** the discard prompt is shown,
**When** `n` or `Esc` is pressed,
**Then** the prompt closes and the form is shown again with every previously
entered value intact.

### C5 — Revert clears dirtiness
**Given** a pre-filled form whose changed field has been edited back to its
original value,
**When** `Esc` is pressed,
**Then** behavior follows C1 (immediate cancel, no prompt).

### C6 — Nested overlay layering
**Given** a covered form with the calendar date picker open,
**When** `Esc` is pressed,
**Then** the calendar closes and the form stays open; a subsequent `Esc` follows
C1/C2.

### C7 — Whitespace-only is clean
**Given** the only difference from the opened state is leading/trailing/again-only
whitespace,
**Then** the form is treated as clean (C1 applies).

## Discard prompt copy (Principle IV — playful, measured)

Primary string:

```
Toss out your unsaved edits? [y]es  [n]o — keep editing
```

Requirements for the copy:
- Accurate and unambiguous about the destructive outcome (Principle IV).
- Exposes both choices (`y` to discard, `n`/`esc` to keep editing).
- Tone is warm but measured, consistent with the existing quit/delete prompts.
