# Phase 0 Research: Multiline Task Descriptions & External Editor

All Technical Context unknowns are resolved below. No outstanding NEEDS CLARIFICATION.

## R1. Where is whitespace actually lost — storage or rendering?

**Decision**: It is lost at **render** time, not storage.

**Evidence**:
- `internal/tui/edit.go` builds the description field as a `textarea.Model`
  (`desc := textarea.New()`), which accepts and returns multiline text via
  `description.Value()`. Newlines survive into `editSavedMsg.description` and are
  sent verbatim in `UpdateTaskRequest.Description` / `CreateTaskRequest.Description`
  (`internal/tui/update.go`).
- `internal/tui/details.go:renderDetails` displays the description through
  `wordWrap`, which does `words := strings.Fields(text)` and rejoins with single
  spaces. `strings.Fields` splits on **all** Unicode whitespace, so every newline
  and run of spaces collapses to one space.

**Rationale**: This confirms the spec's "stored vs. rendered" uncertainty in
favor of rendering, so no data migration is needed and existing tasks are fixed
automatically (FR-004, SC-002).

**Alternatives considered**: Storing a normalized form — rejected; storage is
already correct and rewriting stored data is riskier and unnecessary.

## R2. How should the line-aware wrapper behave? (FR-001/002/003)

**Decision**: New `wrapDescription(text string, width int) string`:
1. If `width <= 0`, return `text` unchanged (matches current `wordWrap` guard).
2. Split on `"\n"` into logical lines.
3. For each logical line:
   - If it is empty / all-whitespace → emit an empty line (preserves blank-line
     paragraph breaks).
   - Else capture leading indentation (`" "`/`"\t"` prefix), then wrap the
     remaining words to `width`, emitting the indentation on the first visual
     line. Continuation lines from wrapping are emitted without re-introducing a
     logical break in stored text (display-only).
4. Join with `"\n"`.

This honors newlines + blank lines + leading indentation, wraps long lines, and
allows mid-line multi-space runs to collapse — exactly the clarified scope.

**Rationale**: Smallest change that satisfies the clarification; keeps the
existing `width <= 0` contract so callers/tests are unaffected for short text.

**Alternatives considered**:
- Fully literal preservation (no re-wrap) — rejected per clarification; risks
  overflow in the details pane.
- A third-party reflow/wrap library (e.g. `muesli/reflow`, already transitively
  present via lipgloss) — rejected to avoid coupling render semantics to a
  dependency for a ~20-line helper; revisit only if width-aware ANSI wrapping is
  later needed.

## R3. Launching an external editor from a Bubble Tea alt-screen program

**Decision**: Use `tea.ExecProcess(cmd, callback)`.

**Evidence**: `bubbletea@v1.3.10/exec.go` exposes
`func ExecProcess(c *exec.Cmd, fn ExecCallback) Cmd` with
`type ExecCallback func(error) Msg`. The program is started with
`tea.WithAltScreen()` in `internal/tui/tui.go`; `ExecProcess` releases the
terminal (leaving alt-screen), runs the command attached to the real stdio, and
restores Bubble Tea control afterward, then dispatches the callback's `Msg`.

**Rationale**: Purpose-built for "shell out to $EDITOR"; avoids manual terminal
teardown/restore and avoids racing the renderer.

**Alternatives considered**:
- `tea.Suspend` + manual `exec.Command(...).Run()` — rejected; more error-prone
  terminal-state management, no benefit here.
- Running the editor in a goroutine — rejected; corrupts the shared TTY.

## R4. Resolving the editor command, including arguments (FR-006)

**Decision**:
- Read `EDITOR`. If non-empty after trimming, `parts := strings.Fields(EDITOR)`;
  command is `parts[0]`, args are `parts[1:]` followed by the temp-file path.
- If `EDITOR` is unset/empty, command is `vim` with the temp-file path.

**Rationale**: Per clarification, supports `code --wait`, `emacsclient -nw`,
`vim -u NONE`, etc. — matching the convention used by `git`, `less`, and most
CLIs. Temp file is always the final argument.

**Alternatives considered**:
- Treat `EDITOR` as a bare binary — rejected by clarification (breaks flagged
  editors).
- Run via `sh -c "$EDITOR <file>"` — rejected; brings quoting/shell-injection
  surface for no real gain over `strings.Fields`.

## R5. Transporting text in and out of the editor

**Decision**: Temp-file round trip.
- Before launch: `os.CreateTemp("", "twig-desc-*.md")`, write current
  description, close. (`.md` gives editors a sensible filetype for soft-wrap.)
- In the `ExecProcess` callback: if the process error is nil, read the file,
  normalize, and return `editorFinishedMsg{content, nil}`; on error return
  `editorFinishedMsg{err: ...}` without reading. Remove the temp file in both
  paths (deferred unlink).

**Normalization**: Strip exactly one trailing `"\n"` if present (many editors
append one on save). This satisfies SC-005 (no blank-line accumulation across
repeated edits) while preserving the user's intentional internal blank lines.

**Rationale**: Files are the universal editor interface; stdin/stdout piping
does not work for interactive editors.

**Alternatives considered**: Pipe-based capture — rejected (interactive editors
need a real TTY, not pipes).

## R6. Threading the result back into the form

**Decision**: `editFormModel.Update` returns the `ExecProcess` command when
`ctrl+g` is pressed with the Description field focused. The resulting
`editorFinishedMsg` surfaces at the top-level `Model.Update`, which is the only
place non-key messages are dispatched. Add a `case editorFinishedMsg` there
that, when still in an edit mode (`modeEdit`/`modeNewSubtask`/`modeNewRoot`):
- on success → `m.edit.description.SetValue(content)` and `m.err = nil`;
- on failure → set a playful `m.err`, leaving the field untouched.

**Rationale**: Mirrors the existing message-routing pattern (e.g.
`editSavedMsg`, `moveTaskResultMsg` are handled at `Model.Update`). The form's
own `Update` only receives `KeyMsg` (via `handleEditKey`) and field-forwarded
messages, so the async editor result must be handled at the model level.

**Alternatives considered**: Forwarding all messages into the form — rejected;
the model does not currently fan messages into `m.edit`, and doing so broadly
would be a larger, riskier change.

## R7. Key binding choice and conflicts (FR-005)

**Decision**: Add `KeyMap.Editor = ctrl+g`. Handle it in `editFormModel.Update`
**before** field-forwarding, and **only** when `f.focusIndex == focusDescription`.

**Evidence**: `ctrl+g` is not bound anywhere in `DefaultKeyMap()`
(`internal/tui/keymap.go`); the edit form's global shortcuts are Save (`ctrl+s`),
Cancel (`esc`), Tab, ShiftTab. No conflict.

**Rationale**: User explicitly requested `ctrl+g`. Gating on description focus
keeps it from interfering when other fields are active and matches the spec.

## R8. No-TTY / launch-failure behavior

**Decision**: If the editor process cannot start or exits non-zero,
`ExecProcess` delivers a non-nil error to the callback → `editorFinishedMsg`
carries it → `Model.Update` shows a playful error and preserves the description
and the rest of the in-progress form (FR-010, edge cases). In a non-interactive
context the TUI is not running at all, so the action is simply never reachable;
no special-casing required.
