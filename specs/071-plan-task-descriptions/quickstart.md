# Quickstart: Task Descriptions on the Planning Tab

**Feature**: 071-plan-task-descriptions | **Date**: 2026-07-31

## What changes

Selecting a plan entry linked to a task now shows that task's markdown description in the Planning tab's Details pane, below the pomodoro row.

## Files touched

| File | Change |
|---|---|
| `internal/tui/plan_view.go` | `renderPlanDetail` — append the description block to both the plain and styled paths |
| `internal/tui/view.go` | Clamp the styled Planning right-pane content to the pane's inner height |
| `internal/tui/plan_view_test.go` | New tests for presence, absence, fidelity, and containment |

No proto, no server, no migration, no `make proto`, no `sqlc generate`.

## The change in shape

In `renderPlanDetail`, after the pomodoro row in each path:

```go
if entry.TaskId != 0 && task != nil {
    if desc := task.GetDescription(); strings.TrimSpace(desc) != "" {
        fmt.Fprintln(&sb)
        if md != nil {
            fmt.Fprintln(&sb, md.Render(desc, markdown.Options{Width: width, Styled: styled}))
        } else {
            fmt.Fprintln(&sb, wrapDescription(desc, width))
        }
    }
}
```

Note that `plan_view.go:313` currently contains `_ = width` — a deliberate marker that `width` was unused. Delete that line; `width` is now genuinely used.

In `view.go`, the styled Planning path (around line 104-114), clamp before boxing:

```go
rightContent = clampLines(rightContent, innerH)
```

`clampLines` truncates to at most `n` lines without padding — distinct from the existing `splitLines`, which pads to exactly `n` and returns a slice.

## Verify

```bash
go build -o twig ./cmd/twig
go test ./internal/tui/
```

Then, with a running stack (`make dev`):

```bash
# 1. Give a task a multi-line markdown description
./twig task add "Write the quarterly report"
# then edit it in the TUI (Tasks tab, 'e') and paste:
#   ## Sections
#   - Revenue
#   - **Headcount**
#   See `notes.md` for last quarter.

# 2. Schedule it onto today's plan, then open the TUI
./twig
```

Walk the Planning tab and confirm:

| Check | Expect |
|---|---|
| Select the scheduled task | Description appears below the pomodoro row, formatted — heading, bullets, bold, inline code |
| Compare with the Tasks tab | Same description renders identically at the same pane width |
| Select an event (no linked task) | No description, no blank gap |
| Select a task with no description | Pane looks exactly as it did before this feature |
| Move the cursor between entries | Description tracks the selection with no lag |
| Paste a 200-line description | Pane clips at its bottom border; grid, borders, and status line stay aligned |
| Shrink the terminal to ~60 columns | Text wraps inside the pane; nothing bleeds into the grid |
| Run with styling off (pipe or non-TTY) | Plain wrapped text, no escape codes |

## Out of scope

- Scrolling the Details pane.
- Editing the description from the Planning tab.
- The identical pre-existing height-overflow bug on the **Tasks** tab detail pane (`view.go:245-248`). Real, but a separate change — see `research.md` R4.
