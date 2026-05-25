package tui

import (
	"fmt"
	"strings"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// renderDetails formats the details pane for the given task, wrapping text to
// the given width.
func renderDetails(task *taskv1.Task, width int) string {
	if task == nil {
		return ""
	}

	name := task.Name
	if task.GetCompletedAt() != nil {
		name = cli.DimStrike(name)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "ID:   %d\n", task.Id)
	fmt.Fprintf(&sb, "Name: %s\n", name)

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
