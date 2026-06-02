# Phase 1 Data Model: Multiline Task Descriptions & External Editor

No persistent/storage schema changes. The task `description` field already
stores arbitrary text including newlines. This document captures the **in-memory
/ runtime** artifacts the feature introduces or touches in `internal/tui/`.

## Existing entities (unchanged shape, behavior touched)

### Task.Description (string)
- The free-text notes of a task, persisted via `task.v1` ConnectRPC.
- May contain `\n`, blank lines, and leading indentation that carry reader
  meaning.
- **Validation**: none added. Optional; empty allowed.
- **Behavior change**: now *rendered* with whitespace preserved (see
  `wrapDescription`), not flattened.

### editFormModel (internal/tui/edit.go)
- Holds the in-progress edit/create form, including `description textarea.Model`.
- **Behavior change**: a `ctrl+g` press while `focusIndex == focusDescription`
  emits an editor command; on return, `description.SetValue(...)` is updated with
  the edited text.

## New runtime artifacts

### KeyMap.Editor (key.Binding)
- New field on `KeyMap` (`internal/tui/keymap.go`).
- Bound to `ctrl+g`, help text e.g. `"ctrl+g", "edit in $EDITOR"`.
- Active only within the edit form, gated on Description focus.

### editorFinishedMsg (tea.Msg)
- New message type (in `editor.go` or `update.go`).
- Fields:
  - `content string` — the edited description text, already normalized (one
    trailing newline stripped). Valid only when `err == nil`.
  - `err error` — non-nil if the editor failed to launch or exited abnormally,
    or the temp file could not be read/written.
- **Lifecycle**:
  1. `ctrl+g` (Description focused) → `openEditorCmd(currentText)` returned as a
     `tea.Cmd`.
  2. Command writes a temp file, runs `tea.ExecProcess(editorCmd, callback)`.
  3. Callback reads/normalizes the temp file (on success), removes it, returns
     `editorFinishedMsg`.
  4. `Model.Update` consumes it: success → update `m.edit.description`,
     `m.err = nil`; failure → set playful `m.err`, leave field unchanged.
- **State guard**: applied only when `m.mode` is one of `modeEdit`,
  `modeNewSubtask`, `modeNewRoot`; otherwise ignored (defensive).

### editorCommand resolution (pure function, editor.go)
- Input: environment (`EDITOR`) + temp-file path.
- Output: `(name string, args []string)` such that `exec.Command(name, args...)`
  launches the editor with the temp file as the **final** argument.
- Rules:
  - `EDITOR` non-empty → `fields := strings.Fields(EDITOR)`; `name = fields[0]`,
    `args = append(fields[1:], path)`.
  - `EDITOR` empty/unset → `name = "vim"`, `args = []string{path}`.
- Pure and table-testable without launching a process.

## Rendering helper (replaces wordWrap)

### wrapDescription(text string, width int) string
- Replaces `wordWrap` in `internal/tui/details.go`.
- Preserves `\n`, blank lines, and per-line leading indentation; wraps each line
  to `width`; collapses mid-line multi-space runs.
- `width <= 0` → returns `text` unchanged (preserves current contract).
- Pure; table-testable.

## Relationships / flow

```
editFormModel (Description focused)
   └─ ctrl+g ─▶ openEditorCmd(text)
                   ├─ write temp file
                   └─ tea.ExecProcess(resolve(EDITOR|vim, path), cb)
                          └─ cb ─▶ editorFinishedMsg{content|err}
                                      └─▶ Model.Update ─▶ editFormModel.description.SetValue / Model.err

Task.Description ──▶ renderDetails ──▶ wrapDescription(width) ──▶ details pane
```
