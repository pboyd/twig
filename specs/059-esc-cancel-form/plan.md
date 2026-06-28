# Implementation Plan: Esc Cancels TUI Forms

**Branch**: `059-esc-cancel-form` | **Date**: 2026-06-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/059-esc-cancel-form/spec.md`

## Summary

Make `Esc` a uniform cancel action across every TUI form. When the form has no
unsaved input it cancels immediately; when canceling would discard typed/edited
content, a playful discard-confirmation prompt appears first. The work is
entirely in the CLI/TUI module (`internal/tui`): add an "opened-state" snapshot
and dirty check to the two form models (`editFormModel`, `planFormState`), wire
`Esc` to a cancel path in each form, and add a single modal discard-confirmation
overlay reusing the existing `confirmingQuit` pattern. No server, API, proto, or
database changes.

## Technical Context

**Language/Version**: Go 1.24 (root CLI/TUI module `github.com/pboyd/twig`)

**Primary Dependencies**: Bubble Tea v2 (`charm.land/bubbletea/v2`), Bubbles v2
(`textinput`, `textarea`, `key`), Lip Gloss v2 — all already in use by the TUI.

**Storage**: N/A — purely in-memory TUI interaction state.

**Testing**: `go test ./...` (root module). TUI logic is unit-tested via
`Model.Update`/form `Update` with synthesized `tea.KeyPressMsg`, plus
`export_test.go` shims (see `update_test.go`, `edit_test.go`, `plan_update_test.go`).

**Target Platform**: Terminal (TTY) on Linux/macOS; behavior gated by TTY styling.

**Project Type**: Single-module CLI/TUI client (no frontend/backend split for this feature).

**Performance Goals**: Per-keystroke handling stays effectively instant; the dirty
check compares a handful of short strings — negligible cost.

**Constraints**: No new dependencies. Preserve the existing layered `Esc` behavior
(calendar overlay consumes the first `Esc`). No server/API/proto/db changes.

**Scale/Scope**: Five form surfaces sharing two form models, plus one new modal
overlay. Estimated ~150–200 LOC across ~4 files in `internal/tui`, with tests.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses the existing `confirmingQuit` overlay pattern and existing `editCancelledMsg` cancel path. Adds a snapshot + `isDirty()` per form model and one boolean overlay state — no new abstraction layers. No new dependencies. |
| II. API-First Design | ✅ | No external/RPC interface changes. The only "contract" is the keyboard-interaction contract, documented in `contracts/keyboard-interaction.md` before implementation. |
| III. UI/UX Consistency | ✅ | `Esc` already means "cancel" (`keys.Cancel`); this extends it consistently to forms that lacked it. The discard prompt mirrors the existing quit-confirmation layout/styling and `[y]es [n]o` convention from the shared theme. |
| IV. Playful User Messages | ✅ | The discard prompt copy is warm-but-measured (destructive action), drafted in `contracts/keyboard-interaction.md` and `quickstart.md`. |

**Post-Design Re-check**: Still ✅ on all four. Design introduces no abstractions
requiring Complexity Tracking; no API surface; reuses shared theme and existing
overlay/cancel patterns; one new user-facing string reviewed for tone.

## Project Structure

### Documentation (this feature)

```text
specs/059-esc-cancel-form/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (TUI state shape)
├── quickstart.md        # Phase 1 output (manual + test walkthrough)
├── contracts/
│   └── keyboard-interaction.md   # Esc-cancel interaction contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
internal/tui/
├── edit.go        # editFormModel: add opened-state snapshot + isDirty(); emit
│                  #   editCancelledMsg (clean) or editDiscardRequestedMsg (dirty) on Esc
├── edit_test.go   # unit tests for snapshot/isDirty + Esc cancel/confirm routing
├── update.go      # Model: add confirmingDiscard handling (intercept in handleKey),
│                  #   handle editDiscardRequestedMsg, dirty-aware Esc in handlePlanFormKey
├── update_test.go # unit tests for the discard overlay across tabs/forms
├── plan_update.go # planFormState: opened-state snapshot + dirty check (or in update.go)
├── plan_update_test.go
├── view.go        # render the discard-confirmation overlay (mirrors confirmingQuit)
├── model.go       # add confirmingDiscard bool (+ snapshot fields on form structs)
└── keymap.go      # (no new binding required; reuses keys.Cancel = "esc")
```

**Structure Decision**: Single-module change confined to `internal/tui`. No new
packages. The two existing form models (`editFormModel` in `edit.go`, used by both
the Tasks and Goals tabs; `planFormState` in `model.go`/`plan_update.go`) each gain
a lightweight opened-state snapshot and dirty check. A single model-level
`confirmingDiscard` boolean drives one modal overlay rendered in `view.go` and
intercepted at the top of `handleKey` so it works uniformly regardless of active tab.

## Complexity Tracking

> No Constitution Check violations. No entries required.
