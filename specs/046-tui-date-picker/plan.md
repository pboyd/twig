# Implementation Plan: TUI Calendar Date Picker

**Branch**: `046-tui-date-picker` | **Date**: 2026-06-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/046-tui-date-picker/spec.md`

## Summary

Add an inline month-view calendar widget to the two date fields in the TUI task edit form (Due and Snooze until). The calendar opens on demand from a focused date field, is fully keyboard-driven, and writes the chosen date back into the field in a format the form already accepts (`YYYY-MM-DD`, or RFC3339 with the existing time-of-day preserved on the Due field). Text entry continues to work unchanged. Implemented as a small self-contained `calendarModel` component in `internal/tui` using only stdlib `time` plus the existing lipgloss theme — no new dependencies, no API or server changes.

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: `charm.land/bubbletea/v2` v2.0.7, `charm.land/bubbles/v2` v2.1.0 (key, textinput), `charm.land/lipgloss/v2` v2.0.3 — all already in use; no new dependencies

**Storage**: N/A (no persistence changes; dates flow through the existing `editSavedMsg` → `cli.ParseDue` → ConnectRPC path)

**Testing**: `go test ./...` — table-driven unit tests in `internal/tui` following the existing `edit_test.go` / `export_test.go` shim conventions

**Target Platform**: Terminal (TTY) on Linux/macOS — wherever the existing TUI runs

**Project Type**: CLI/TUI client feature (root module only)

**Performance Goals**: Imperceptible — calendar render is a 7×~6 text grid; no I/O involved

**Constraints**: Keyboard-only interaction; must respect the adaptive light/dark palette in `internal/tui/theme.go`; must not conflict with existing form key bindings (`tab`, `shift+tab`, `ctrl+s`, `esc`, `enter`); must degrade gracefully at small terminal widths

**Scale/Scope**: One new component file (~200 LOC) + wiring in `edit.go` + keymap additions + tests. Two fields affected (Due, Snooze until).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Single new component built on stdlib `time`; no third-party datepicker dependency; no abstraction beyond what `edit.go` needs. Reuse by other surfaces (plan date prompt) is deliberately deferred. |
| II. API-First Design | ✅ | No API change. The UI contract (widget behavior + key bindings + output formats) is committed in `contracts/calendar-widget.md` before implementation. |
| III. UI/UX Consistency | ✅ | Calendar colors come exclusively from the shared palette (`accent`, `dim`, `border`, `highlightStyle`); navigation keys mirror existing conventions (`↑/↓/←/→` + `h/j/k/l` movement, `[`/`]` month jumps matching the plan tab's day jumps, `esc` cancel, `enter` confirm). |
| IV. Playful User Messages | ✅ | New user-facing text (calendar hint line, footer help) reviewed for tone in the contract; e.g. hint "ctrl+g: pick from calendar". Tone obligations noted in contracts. |

**Post-Phase-1 re-check**: ✅ All four principles still hold. No Complexity Tracking entries required.

## Project Structure

### Documentation (this feature)

```text
specs/046-tui-date-picker/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/
│   └── calendar-widget.md   # UI contract: states, key bindings, output formats
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/tui/
├── calendar.go          # NEW: calendarModel — month grid state, navigation, rendering
├── calendar_test.go     # NEW: unit tests (navigation, clamping, output format, render)
├── edit.go              # MODIFIED: open/route/apply calendar on Due & Snooze fields
├── edit_test.go         # MODIFIED: form-level integration of the calendar
├── keymap.go            # MODIFIED: add Calendar key binding (ctrl+g on date fields)
└── export_test.go       # MODIFIED (if needed): test shims for new helpers

internal/cli/
└── render.go            # UNCHANGED: cli.ParseDue remains the single date parser
```

**Structure Decision**: All changes live in the root module's `internal/tui` package, alongside the form they extend. The calendar is a sibling component to `editFormModel` (same pattern as `editorModel` in `editor.go`). No server, API, or web changes.

## Complexity Tracking

> No Constitution Check violations — table intentionally empty.
