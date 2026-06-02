# Implementation Plan: Multiline Task Descriptions & External Editor

**Branch**: `030-task-description-editing` | **Date**: 2026-06-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/030-task-description-editing/spec.md`

## Summary

Task descriptions already store newlines (the edit form uses a `textarea`), but
the TUI details renderer flattens them: `renderDetails` calls `wordWrap`, which
uses `strings.Fields()` and collapses every run of whitespace — including
newlines — into single spaces. The fix is a display-time change: replace
`wordWrap` with a line-aware wrapper that preserves the user's newlines, blank
lines, and per-line leading indentation while still wrapping long lines to the
pane width.

Second, the inline `textarea` is a cramped place to write longer text. We add a
`ctrl+g` action that, while the Description field is focused in the edit form,
shells out to the user's `$EDITOR` (splitting the value so `code --wait` etc.
work; falling back to `vim`) pre-loaded with the current description, then
folds the saved text back into the form field. Bubble Tea's `tea.ExecProcess`
handles suspending/restoring the alt-screen terminal around the editor.

Both changes are confined to the Go TUI (`internal/tui/`). No proto, no
handler, no DB, and no web-client changes.

## Technical Context

**Language/Version**: Go (module at `services/twig/`, toolchain per existing `go.mod`)

**Primary Dependencies**: `github.com/charmbracelet/bubbletea v1.3.10`
(provides `tea.ExecProcess`), `github.com/charmbracelet/bubbles v1.0.0`
(`textarea`, `key`), `github.com/charmbracelet/lipgloss`. Standard library
`os`, `os/exec`, `strings`.

**Storage**: N/A for this feature — descriptions are persisted unchanged via the
existing `task.v1` ConnectRPC path; the field already round-trips newlines.

**Testing**: `go test ./...`; table-driven unit tests in `internal/tui/`
(`details_test.go`, `edit_test.go`, `update_test.go`) using `export_test.go`
shims. No DB required.

**Target Platform**: Interactive terminal (TTY), alt-screen Bubble Tea program
launched from `cmd/twig`. Linux/macOS terminals.

**Project Type**: CLI/TUI client within a single Go module (plus an unrelated
React web client that is out of scope here).

**Performance Goals**: Imperceptible. Rendering is per-frame string work on a
single description; editor launch is a one-shot subprocess.

**Constraints**: Must work under `tea.WithAltScreen()` (the program is started
with it in `tui.go`). Editor action must be a no-op when no TTY is available.
Repeated editor round-trips must not accumulate trailing blank lines.

**Scale/Scope**: ~3 source files touched plus tests. Single-user local client.

## Constitution Check

*No `.specify/memory/constitution.md` exists in this repo; evaluated against the
four principles embedded in the plan template.*

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses `tea.ExecProcess`; one new key binding; one rewritten render helper. No new packages, deps, or abstractions. |
| II. API-First Design | ✅ | No API surface changes. The CLI "contract" affected is the edit-form key set + details rendering, documented in `contracts/`. |
| III. UI/UX Consistency | ✅ | New `ctrl+g` binding follows existing `key.Binding` + `KeyMap` conventions and the form's hint-line/help patterns. Rendering reuses the details pane. |
| IV. Playful User Messages | ✅ | New user-facing strings (editor-failed notice, hint text) authored in the warm/playful tone used elsewhere in the TUI (e.g. plan notices). |

Post-Phase-1 re-check: still ✅ — design introduces no new violations (see end of Phase 1).

## Project Structure

### Documentation (this feature)

```text
specs/030-task-description-editing/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── tui-description-editing.md   # Phase 1 output (CLI/TUI interaction contract)
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/twig/internal/tui/
├── details.go        # CHANGE: replace wordWrap → line-aware wrapDescription (FR-001/002/003)
├── details_test.go   # CHANGE: add multiline/indentation/blank-line render tests
├── edit.go           # CHANGE: handle ctrl+g in description focus; hint line; (FR-005/007/008/009/011)
├── edit_test.go      # CHANGE: assert ctrl+g emits editor command only when description focused
├── editor.go         # NEW: openEditorCmd + editor resolution ($EDITOR split / vim) (FR-006/010)
├── editor_test.go    # NEW: editor argv resolution + trailing-newline normalization tests
├── keymap.go         # CHANGE: add KeyMap.Editor binding (ctrl+g)
├── update.go         # CHANGE: route editorFinishedMsg → apply text to m.edit / set error
├── update_test.go    # CHANGE: editorFinishedMsg updates description; error path preserves it
└── export_test.go    # CHANGE: export wrapDescription + editor helpers for tests
```

**Structure Decision**: Single Go module, TUI-only change. All edits live under
`services/twig/internal/tui/`. The editor-launch logic is isolated in a new
`editor.go` so the OS/exec concern (and its `$EDITOR` parsing) is unit-testable
in isolation from Bubble Tea wiring, mirroring how `pomodoro`/`plan` logic is
kept separable from rendering.

## Phase 0: Research

See [research.md](./research.md). Key resolved decisions:

- **Whitespace preservation** → line-aware wrap that splits on `\n`, preserves
  blank lines and per-line leading indentation, and wraps each line's words to
  width (mid-line multi-space runs may collapse). Resolves the "stored vs.
  rendered" question: stored fine, rendered wrong.
- **Editor launch under alt-screen** → `tea.ExecProcess`, which releases and
  restores the terminal correctly; no manual `tea.Suspend`/`tput` handling.
- **`$EDITOR` with arguments** → `strings.Fields($EDITOR)`, exec `parts[0]` with
  `parts[1:]` plus the temp-file path appended last; fall back to `vim`.
- **Round-trip transport** → a temp file written before launch and read in the
  `ExecProcess` callback; strip exactly one trailing `\n` to avoid blank-line
  accumulation across edits.
- **No-TTY / launch failure** → editor unavailable is a graceful no-op /
  playful error; description and the rest of the form are preserved.

## Phase 1: Design & Contracts

### Data model

See [data-model.md](./data-model.md). No persistent schema change. The only new
in-memory artifacts are a `KeyMap.Editor` binding and an `editorFinishedMsg`
Bubble Tea message carrying the edited text (or an error) back into `Model.Update`.

### Contracts

See [contracts/tui-description-editing.md](./contracts/tui-description-editing.md)
for the CLI/TUI interaction contract: the `ctrl+g` trigger conditions, editor
resolution rules, the message round-trip, and the details-render guarantees with
worked input/output examples.

### Agent context update

`CLAUDE.md`'s SPECKIT block is repointed from the 029 plan to this plan
(`specs/030-task-description-editing/plan.md`).

### Post-design Constitution re-check

✅ No new violations. The design adds one binding, one message type, one new
small file, and rewrites one helper — all within established TUI patterns.

## Complexity Tracking

No constitution violations; table intentionally empty.
