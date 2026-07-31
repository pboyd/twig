package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/markdown"
)

// renderTaskDescription renders a task description for a detail pane: markdown
// when md is non-nil, plain wrapped text otherwise.
func renderTaskDescription(md *markdown.Renderer, text string, width int, styled bool) string {
	if md != nil {
		return md.Render(text, markdown.Options{Width: width, Styled: styled})
	}
	return wrapDescription(text, width)
}

// renderDetails formats the details pane for the given task, wrapping text to
// the given width. When styled, the name is rendered as a bold accent header
// and labels are dim and column-aligned. scheduledDays is the task's scheduled
// days slice (ascending YYYY-MM-DD, today-or-future); nil or empty omits the line.
// effectiveGoalName, when non-empty, adds a "Goal:" line (from own or inherited goal).
func renderDetails(task *taskv1.Task, md *markdown.Renderer, width int, styled bool, scheduledDays []string, effectiveGoalName string, now time.Time) string {
	if task == nil {
		return ""
	}

	renderDesc := func() string {
		return renderTaskDescription(md, task.GetDescription(), width, styled)
	}

	// Compute the display name with markup stripped for both paths.
	// Plain: RenderInline(Styled:false) strips ** * ~~ but keeps `backticks`.
	// Styled: strip ANSI from the styled render to get clean text for the header.
	displayName := task.Name
	if md != nil {
		displayName = md.RenderInline(task.Name, markdown.Options{Width: width, Styled: false})
	}
	styledDisplayName := task.Name
	if md != nil {
		styledDisplayName = ansi.Strip(md.RenderInline(task.Name, markdown.Options{Width: width, Styled: true}))
	}

	if !styled {
		var sb strings.Builder
		fmt.Fprintf(&sb, "ID:   %d\n", task.Id)
		fmt.Fprintf(&sb, "Name: %s\n", displayName)

		if due := cli.FormatDue(task.Due); due != "" {
			fmt.Fprintf(&sb, "Due:  %s\n", due)
		}

		if effectiveGoalName != "" {
			fmt.Fprintf(&sb, "Goal: %s\n", effectiveGoalName)
		}

		if len(scheduledDays) > 0 {
			fmt.Fprintf(&sb, "Scheduled for: %s\n", strings.Join(scheduledDays, ", "))
		}

		if row := renderPomodoroRow(int(task.GetEstimate()), int(task.GetCompletedPomodoroCount()), false); row != "" {
			fmt.Fprintln(&sb, row)
		}

		if strings.TrimSpace(task.GetDescription()) != "" {
			fmt.Fprintln(&sb)
			fmt.Fprintln(&sb, renderDesc())
		}

		if cat := cli.FormatCompletedAt(task.CompletedAt); cat != "" {
			fmt.Fprintf(&sb, "\nCompleted: %s\n", cat)
		}

		if taskIsSnoozed(task, now) {
			snoozeDay := task.SnoozeUntil.AsTime().UTC().Format("2006-01-02")
			fmt.Fprintf(&sb, "Snooze:   %s 💤\n", snoozeDay)
		}

		return sb.String()
	}

	// Styled path: bold accent header + dim column-aligned labels.
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	labelStyle := lipgloss.NewStyle().Foreground(dim)
	completedStyle := lipgloss.NewStyle().Foreground(completed)

	var sb strings.Builder

	// Task name as bold accent header.
	fmt.Fprintln(&sb, headerStyle.Render(styledDisplayName))

	// Column-aligned labels (4-char label + ": " = 6 chars).
	fmt.Fprintf(&sb, "%s %d\n", labelStyle.Render("ID:  "), task.Id)

	if due := cli.FormatDue(task.Due); due != "" {
		fmt.Fprintf(&sb, "%s %s\n", labelStyle.Render("Due: "), due)
	}

	if effectiveGoalName != "" {
		fmt.Fprintf(&sb, "%s %s\n", labelStyle.Render("Goal:"), effectiveGoalName)
	}

	if len(scheduledDays) > 0 {
		fmt.Fprintf(&sb, "%s %s\n", labelStyle.Render("Scheduled for:"), strings.Join(scheduledDays, ", "))
	}

	if row := renderPomodoroRow(int(task.GetEstimate()), int(task.GetCompletedPomodoroCount()), true); row != "" {
		fmt.Fprintln(&sb, row)
	}

	if task.GetDescription() != "" {
		fmt.Fprintln(&sb)
		fmt.Fprintln(&sb, renderDesc())
	}

	if cat := cli.FormatCompletedAt(task.CompletedAt); cat != "" {
		fmt.Fprintf(&sb, "\n%s %s\n", labelStyle.Render("Completed:"), completedStyle.Render(cat))
	}

	if taskIsSnoozed(task, now) {
		snoozeDay := task.SnoozeUntil.AsTime().UTC().Format("2006-01-02")
		fmt.Fprintf(&sb, "%s %s 💤\n", labelStyle.Render("Snooze:  "), snoozeDay)
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
