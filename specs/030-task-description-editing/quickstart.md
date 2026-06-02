# Quickstart: Multiline Task Descriptions & External Editor

How to build, exercise, and verify this feature.

## Build & test

```bash
cd services/twig
go test ./internal/tui/...      # unit tests for this feature
go test ./...                   # full suite
go build -o twig ./cmd/twig     # build the CLI/TUI
```

## Run the TUI

```bash
# Requires a provisioned user + running server (see CLAUDE.md "Dev Environment").
export TWIG_API_KEY=...           # from --provision-user
export TWIG_ADDR=http://localhost:8080
./twig                            # launches the interactive TUI on a TTY
```

## Manual verification

### Whitespace on display (User Story 1)
1. In the Tasks tab, select a task and press `enter` to edit (or `ctrl+n` for a
   new root task).
2. Tab to **Description**. Type several lines, including a blank line between two
   paragraphs and an indented bullet, e.g.:
   ```
   Plan:
     - step one
     - step two

   Done when all steps pass.
   ```
3. Save with `ctrl+s`.
4. Move the cursor to that task and confirm the details pane shows the line
   breaks, the blank line, and the indentation — not a single run-on line.
5. (Regression for existing data) Open a task whose description was created
   before this change and contained newlines; confirm they now display.

### External editor (User Story 2)
1. `export EDITOR=nvim` (or leave unset to test the `vim` fallback; try
   `export EDITOR="code --wait"` to test arguments).
2. Edit a task; Tab to **Description**; press `ctrl+g`.
3. The configured editor opens full-screen, pre-loaded with the current text.
4. Edit, then save & quit.
5. Confirm the edited text appears back in the Description field; `ctrl+s` to
   persist, then check the details pane.
6. Re-open the editor and quit **without** changes — the description is
   unchanged. Repeat a few times — no extra blank lines accumulate.
7. Failure path: `export EDITOR=definitely-not-a-real-editor`, press `ctrl+g`;
   confirm a friendly error appears and the description/form are intact.

## Automated coverage map

| Requirement | Test location |
|-------------|---------------|
| FR-001/002/003 (render preserves newlines, blank lines, indentation; wraps) | `details_test.go` (new cases via `ExportRenderDetails` / `wrapDescription`) |
| FR-004 (existing data) | `details_test.go` — render a task with stored `\n` |
| FR-005 (ctrl+g only when Description focused) | `edit_test.go` |
| FR-006 (`$EDITOR` split / vim fallback) | `editor_test.go` (`resolveEditor` table) |
| FR-007/008 (apply edited text / unchanged on no-op) | `update_test.go` (`editorFinishedMsg`) |
| FR-009/010 (failure preserves form, playful error) | `update_test.go` (error path) |
| SC-005 (trailing-newline normalization) | `editor_test.go` (normalize table) |

## Touch points (for reviewers)

- `internal/tui/details.go` — `wordWrap` → `wrapDescription`
- `internal/tui/editor.go` (new) — `openEditorCmd`, `resolveEditor`, normalize
- `internal/tui/edit.go` — `ctrl+g` handling + hint line
- `internal/tui/keymap.go` — `KeyMap.Editor`
- `internal/tui/update.go` — `editorFinishedMsg` case
- `internal/tui/export_test.go` — test exports
