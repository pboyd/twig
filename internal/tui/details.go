package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
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

		if row := renderPomodoroRow(int(task.GetEstimate()), int(task.GetCompletedPomodoroCount()), false); row != "" {
			fmt.Fprintln(&sb, row)
		}

		if task.GetDescription() != "" {
			fmt.Fprintln(&sb)
			desc := wrapDescription(task.GetDescription(), width)
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

	if row := renderPomodoroRow(int(task.GetEstimate()), int(task.GetCompletedPomodoroCount()), true); row != "" {
		fmt.Fprintln(&sb, row)
	}

	if task.GetDescription() != "" {
		fmt.Fprintln(&sb)
		desc := wrapDescription(task.GetDescription(), width)
		fmt.Fprintln(&sb, desc)
	}

	if cat := cli.FormatCompletedAt(task.CompletedAt); cat != "" {
		fmt.Fprintf(&sb, "\n%s %s\n", labelStyle.Render("Completed:"), completedStyle.Render(cat))
	}

	return sb.String()
}

// wrapDescription wraps text preserving the user's newlines, blank lines, and
// per-line leading indentation. Each logical line's words are word-wrapped to
// width; mid-line multi-space runs may collapse. width<=0 returns text unchanged.
func wrapDescription(text string, width int) string {
	if width <= 0 {
		return text
	}
	logicalLines := strings.Split(text, "\n")
	out := make([]string, 0, len(logicalLines))
	for _, line := range logicalLines {
		trimmed := strings.TrimLeft(line, " \t")
		indent := line[:len(line)-len(trimmed)]

		if trimmed == "" {
			out = append(out, indent)
			continue
		}

		words := strings.Fields(trimmed)
		avail := width - len(indent)
		if avail <= 0 {
			out = append(out, line)
			continue
		}

		var sb strings.Builder
		lineLen := 0
		prefix := indent
		for i, w := range words {
			wlen := len(w)
			if i > 0 && lineLen+1+wlen > avail {
				out = append(out, prefix+sb.String())
				sb.Reset()
				lineLen = 0
				prefix = ""
				avail = width
			} else if i > 0 {
				sb.WriteByte(' ')
				lineLen++
			}
			sb.WriteString(w)
			lineLen += wlen
		}
		if sb.Len() > 0 {
			out = append(out, prefix+sb.String())
		}
	}
	return strings.Join(out, "\n")
}
