# Implementation Plan: Plan Entry Form Parity with Task Form

**Branch**: `034-plan-form-parity` | **Date**: 2026-06-02 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/034-plan-form-parity/spec.md`

## Summary

The three planning forms (schedule task, add event, edit entry) are rendered by a single shared surface (`renderPlanFormView`) that looks and behaves differently from the task edit form, with no justification. This feature brings them to parity with the task form: a per-mode title line, blank-line separation between fields, focusable `[ Save ]`/`[ Cancel ]` buttons, removal of Enter-to-save, and an identical help line. The edit form additionally pre-fills the entry's current Name/Start/Duration (replacing the old `blank=keep` / `null=unschedule` sentinel convention) and unschedules an entry when the Start field is cleared. The change is confined to the TUI (`internal/tui`); no backend, proto, or DB changes.

## Technical Context

**Language/Version**: Go 1.x (single module at `services/twig/`)

**Primary Dependencies**: Bubble Tea (`charmbracelet/bubbletea`), Bubbles (`textinput`), existing `internal/cli/timeparse` (start/duration parsing)

**Storage**: N/A — no persistence changes; existing Plan ConnectRPC operations (rename/move) are reused unchanged

**Testing**: `go test ./...`; TUI table/string tests in `internal/tui/*_test.go` (no DB required)

**Target Platform**: Interactive TUI on a TTY (Linux/macOS terminals)

**Project Type**: Single Go module, two binaries (CLI/TUI + server). This feature touches only the CLI/TUI binary.

**Performance Goals**: N/A — synchronous local rendering; no new network calls

**Constraints**: Pre-filled values MUST re-parse without modification (Start `HH:MM`, Duration compact unit form); saving an unchanged form MUST be a true no-op (no rename/move dispatched)

**Scale/Scope**: ~3 source files (`plan_view.go`, `plan_update.go`, `update.go`) plus model focus changes (`model.go`) and test updates; no new packages

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses the existing `planFormState` (slice + focus int) and existing rename/move commands. Adds two virtual focus slots for Save/Cancel rather than introducing a new form abstraction. No new packages or layers. |
| II. API-First Design | ✅ | No API/proto/DB change. The relevant contract is the TUI form's input→operation mapping, documented in `contracts/edit-form-ui.md` before implementation. |
| III. UI/UX Consistency | ✅ | This feature **is** a consistency fix — it aligns the planning forms with the task form, the established reference design (same title treatment, spacing, buttons, focus model, key bindings, help line). Directly serves Principle III. |
| IV. Playful User Messages | ✅ | No new prose copy is introduced; titles ("Edit entry", "Schedule task", "Add event") and the help line mirror the existing task-form wording. Existing playful validation messages (e.g. "name cannot be empty") are preserved and re-checked for tone. |

## Project Structure

### Documentation (this feature)

```text
specs/034-plan-form-parity/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (form state / focus model)
├── quickstart.md        # Phase 1 output (manual verification steps)
├── contracts/
│   └── edit-form-ui.md  # Phase 1 output (TUI form contract)
└── checklists/
    └── requirements.md  # Spec quality checklist (already present)
```

### Source Code (repository root)

```text
services/twig/internal/tui/
├── model.go          # planFormState — focus model now spans fields + Save/Cancel
├── plan_view.go      # renderPlanFormView (title, blank lines, buttons, help line),
│                     # planFieldLabel (drop blank=keep/null=unschedule hints)
├── plan_update.go    # initEditForm (pre-fill Start/Duration, neutral placeholders),
│                     # initAddEventForm / initTaskTimeForm (placeholders),
│                     # cyclePlanFormFocus (wrap over fields + buttons),
│                     # submitEditForm (clear-Start = unschedule; change detection)
├── update.go         # handlePlanFormKey (remove Enter-to-save; Enter advances field
│                     # or activates focused Save/Cancel button)
├── plan_update_test.go   # update edit-form expectations (pre-fill, unschedule, no-op)
├── plan_us2_test.go      # schedule-form label/behavior unaffected; verify still green
└── plan_view_test.go     # add/adjust render assertions (title, buttons, help line)
```

**Structure Decision**: Single Go module, existing `internal/tui` package. No new source files; changes are localized edits to the five files above plus their tests. The planning forms keep sharing `renderPlanFormView` and `planFormState`; parity is achieved by extending those shared pieces, not by forking per-mode renderers.

## Complexity Tracking

> No Constitution Check violations. No complexity to justify.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
