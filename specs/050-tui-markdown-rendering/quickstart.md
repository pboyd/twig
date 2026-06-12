# Quickstart: Markdown Rendering in TUI Text Fields

**Feature**: 050-tui-markdown-rendering

## What this delivers

Text fields in the TUI render GitHub-flavored markdown instead of showing raw markup:

- **Long fields** (task/goal descriptions, plan notes) — full rendering: headings, lists, tables, blockquotes, code, rules, links.
- **Short fields** (task/goal names) — inline emphasis only: **bold**, *italic*, ~~strikethrough~~, `code`, links.
- **Editing** still shows raw markdown; stored text is never altered.

## Add the dependency

```bash
# from repo root
go get github.com/yuin/goldmark@v1.7.13
go mod tidy
```

## Using the renderer (consumer view)

```go
// internal/tui/model.go — construct once per session
m.md = markdown.NewRenderer(buildMarkdownTheme()) // buildMarkdownTheme() lives in theme.go

// Long field (e.g. task description in details.go)
desc := m.md.Render(task.GetDescription(), markdown.Options{Width: width, Styled: m.styled})

// Short field (e.g. task name in a tree row in view.go)
name := m.md.RenderInline(task.Name, markdown.Options{Width: width, Styled: m.styled})
```

`buildMarkdownTheme()` maps the existing palette to the renderer's `Theme` (no new colors):

```go
func buildMarkdownTheme() markdown.Theme {
    return markdown.Theme{Accent: accent, Dim: dim, CodeBg: cursorBg /* or derived */}
}
```

## Verify it works

### Automated

```bash
# from repo root
go test ./internal/markdown/...   # renderer unit tests (styled + plain, edge cases)
go test ./internal/tui/...        # integration at the render sites
go test ./...                     # full suite
```

Key assertions the tests must cover:

- **Styled** output for each construct contains the expected lipgloss styling; **plain** output (`Styled:false`) contains **zero** escape sequences (`\x1b`).
- Inline render of `**Ship** the *report*` is single-line, styled, with no `*` characters.
- A description with a heading, nested list, table, blockquote, code block, and link renders each recognizably and within `Width`.
- Malformed input (unclosed `**`, half-written table) renders without panic.
- Non-ASCII/emoji content wraps on display width, not byte length.
- Links: styled → OSC 8 sequence present; plain → `text (url)`.

### Manual (in the real TUI)

```bash
go build -o twig ./cmd/twig
./twig            # launch the TUI
```

1. Create/edit a task; set the description to a markdown sample (heading, `- list`, `**bold**`, a table, a `> quote`, a fenced code block, and a `[link](https://example.com)`).
2. View the task — confirm each construct renders; no stray `*`/`` ` ``/`#`.
3. Give a task a name like `**Ship** the report` — confirm bold renders inline in the tree row and detail header.
4. Re-open the field for editing — confirm you see the **raw** markdown, and saving leaves it unchanged.
5. Resize the terminal narrow — confirm wrapping/table reflow stays inside the pane (code blocks/long URLs may clip — expected).
6. `NO_COLOR=1 ./twig` (or pipe to a non-TTY) — confirm plain, escape-free output.

## Scope reminder

TUI only. No changes to the plain CLI output, the server, the database, or the React web app.

## Known terminal limitations (by design — see research.md)

- **Italic** display depends on the terminal (some ignore SGR 3).
- **Headings** are styled text, not real headings (terminals have no heading concept).
- **Links**: OSC 8 support can't be reliably detected at runtime; styled mode emits OSC 8 regardless, plain mode shows `text (url)`.
- **Wide tables / long code lines / long URLs**: best-effort; may clip at the pane edge rather than wrap. They never corrupt surrounding UI.
