package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// renderDetails formats the details pane for the given task, wrapping text to
// the given width. When styled, the name is rendered as a bold accent header
// and labels are dim and column-aligned.
func renderDetails(task *taskv1.Task, width int, styled bool) string {
	if task == nil {
		return ""
	}

	if !styled {
		var sb strings.Builder
		fmt.Fprintf(&sb, "ID:   %d\n", task.Id)
		fmt.Fprintf(&sb, "Name: %s\n", task.Name)

		if due := cli.FormatDue(task.Due); due != "" {
			fmt.Fprintf(&sb, "Due:  %s\n", due)
		}

		if task.GetEstimate() > 0 {
			fmt.Fprintf(&sb, "Est:  %d pomodoros\n", task.GetEstimate())
		}

		if task.GetDescription() != "" {
			fmt.Fprintln(&sb)
			desc := wordWrap(task.GetDescription(), width)
			fmt.Fprintln(&sb, desc)
		}

		if cat := cli.FormatCompletedAt(task.CompletedAt); cat != "" {
			fmt.Fprintf(&sb, "\nCompleted: %s\n", cat)
		}

		return sb.String()
	}

	// Styled path: bold accent header + dim column-aligned labels.
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	labelStyle := lipgloss.NewStyle().Foreground(dim)
	completedStyle := lipgloss.NewStyle().Foreground(completed)

	var sb strings.Builder

	// Task name as bold accent header.
	fmt.Fprintln(&sb, headerStyle.Render(task.Name))

	// Column-aligned labels (4-char label + ": " = 6 chars).
	fmt.Fprintf(&sb, "%s %d\n", labelStyle.Render("ID:  "), task.Id)

	if due := cli.FormatDue(task.Due); due != "" {
		fmt.Fprintf(&sb, "%s %s\n", labelStyle.Render("Due: "), due)
	}

	if task.GetEstimate() > 0 {
		fmt.Fprintf(&sb, "%s %d pomodoros\n", labelStyle.Render("Est: "), task.GetEstimate())
	}

	if task.GetDescription() != "" {
		fmt.Fprintln(&sb)
		desc := wordWrap(task.GetDescription(), width)
		fmt.Fprintln(&sb, desc)
	}

	if cat := cli.FormatCompletedAt(task.CompletedAt); cat != "" {
		fmt.Fprintf(&sb, "\n%s %s\n", labelStyle.Render("Completed:"), completedStyle.Render(cat))
	}

	return sb.String()
}

// wordWrap wraps text at width characters on word boundaries.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var sb strings.Builder
	lineLen := 0
	for i, w := range words {
		wlen := len(w)
		if lineLen > 0 && lineLen+1+wlen > width {
			sb.WriteByte('\n')
			lineLen = 0
		} else if i > 0 {
			sb.WriteByte(' ')
			lineLen++
		}
		sb.WriteString(w)
		lineLen += wlen
	}
	return sb.String()
}
