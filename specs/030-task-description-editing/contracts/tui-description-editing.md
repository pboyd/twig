# Contract: TUI Description Editing & Display

This feature has no network/API contract changes. The relevant "interfaces" are
(1) the details-pane rendering guarantee and (2) the edit-form keyboard
interaction. Both are specified here with worked examples that the unit tests
assert against.

## C1. Details rendering preserves user whitespace

**Surface**: `renderDetails(task, width, styled)` → `wrapDescription(text, width)`
in `internal/tui/details.go`.

**Guarantees**:
1. Every `\n` the user entered appears as a line break in the rendered details.
2. Blank lines between paragraphs are preserved (one stored blank line → one
   rendered blank line).
3. A line's leading indentation (spaces/tabs) is preserved on its first rendered
   visual line.
4. A logical line longer than `width` is wrapped onto multiple visual lines;
   wrapping does not delete or move the user's own line breaks.
5. Runs of multiple spaces in the middle of a line MAY collapse to a single
   space.
6. `width <= 0` returns the text unchanged.

### Worked examples

Input description (stored):
```
Buy groceries:
  - milk
  - eggs

Then cook.
```

Rendered details body (width comfortably wide), exact line structure:
```
Buy groceries:
  - milk
  - eggs

Then cook.
```
(Note the preserved blank line before "Then cook." and the 2-space indentation
on the list items.)

Long-line wrap (width = 20):
```
Input : "the quick brown fox jumps over"
Output: "the quick brown fox\njumps over"
```

Mid-line space collapse (acceptable):
```
Input : "a     b"
Output: "a b"
```

## C2. `ctrl+g` opens the external editor (edit form)

**Surface**: `editFormModel.Update` (`internal/tui/edit.go`) + `KeyMap.Editor`
(`internal/tui/keymap.go`).

**Preconditions**: the edit/create form is open AND the Description field is the
focused field (`focusIndex == focusDescription`).

**Behavior**:

| Condition | Result |
|-----------|--------|
| `ctrl+g`, Description focused | Form returns the editor command (a `tea.Cmd` from `tea.ExecProcess`). |
| `ctrl+g`, any other field focused (Name/Due/Estimate/Save/Cancel) | No editor launch; key is forwarded/ignored as today. |
| `EDITOR="nvim"` | Launches `nvim <tmpfile>`. |
| `EDITOR="code --wait"` | Launches `code --wait <tmpfile>` (args preserved, file last). |
| `EDITOR` unset/empty | Launches `vim <tmpfile>`. |

**Round trip**:
1. Current `description.Value()` is written to a temp file before launch.
2. Editor runs full-screen (alt-screen released, then restored by `ExecProcess`).
3. On save+exit (process exit 0): the temp file is read, **one** trailing `\n`
   is stripped, and the resulting text replaces the Description field value;
   `Model.err` is cleared.
4. On exit-without-change / quit: the field equals its pre-launch value
   (idempotent — re-saving identical content does not alter the visible text).
5. On launch failure or non-zero exit: the Description field and the rest of the
   form are unchanged; a playful error notice is shown.
6. The editor action never persists the task by itself; the edited text is saved
   only through the normal Save (`ctrl+s`) flow.

## C3. Editor command resolution (pure)

**Surface**: `resolveEditor(editorEnv, path string) (name string, args []string)`
in `internal/tui/editor.go`.

| `EDITOR` value | `name` | `args` |
|----------------|--------|--------|
| `""` (unset)   | `vim`  | `[path]` |
| `"vim"`        | `vim`  | `[path]` |
| `"code --wait"`| `code` | `[--wait, path]` |
| `"  emacsclient  -nw "` (extra spaces) | `emacsclient` | `[-nw, path]` |

The temp-file path is always the final element of `args`.

## C4. Trailing-newline normalization (pure)

**Surface**: normalization applied to editor output before `SetValue`.

| Editor output bytes | Stored/applied value |
|---------------------|----------------------|
| `"hello\n"`         | `"hello"` |
| `"hello"`           | `"hello"` |
| `"a\n\nb\n"`        | `"a\n\nb"` (internal blank line kept; one trailing `\n` removed) |
| `"" ` (empty)       | `""` |

Exactly one trailing `\n` is stripped (not all), so repeated open/save cycles do
not grow or shrink the description (SC-005).

## Non-goals (explicit)

- No change to how descriptions are stored or transmitted.
- No change to the web client.
- No improvement to the inline `textarea` beyond adding the `ctrl+g` escape
  hatch.
