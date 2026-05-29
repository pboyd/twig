package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// renderTabBar renders the one-line tab bar: " Tasks │ Planning "
// with the active tab highlighted in accent color (styled) or plain text (unstyled).
func (m Model) renderTabBar(width int) string {
	tasks := " Tasks "
	planning := " Planning "

	if m.styled {
		// Active tab: bold blue accent. Inactive: dim.
		const (
			accentOpen = "\x1b[1;34m" // bold + blue
			dimOpen    = "\x1b[2m"    // dim
			reset      = "\x1b[0m"
			sepDim     = "\x1b[2m│\x1b[0m"
		)
		var taskStr, planStr string
		if m.activeTab == tabTasks {
			taskStr = accentOpen + tasks + reset
			planStr = dimOpen + planning + reset
		} else {
			taskStr = dimOpen + tasks + reset
			planStr = accentOpen + planning + reset
		}
		bar := taskStr + sepDim + planStr
		visW := lipgloss.Width(bar)
		if width > visW {
			bar += strings.Repeat(" ", width-visW)
		}
		return bar
	}

	sep := "|"
	bar := tasks + sep + planning
	if width > len(bar) {
		bar += strings.Repeat(" ", width-len(bar))
	}
	return bar
}

// renderPlanningView renders the full planning tab content (not including the
// tab bar or status bar).
func (m Model) renderPlanningView(width, height int, now time.Time) string {
	// Modals overlay the grid.
	switch m.plan.mode {
	case planPickTask:
		return m.renderPlanPickerView(width)
	case planTaskTime, planEventForm, planRename, planMove, planClear:
		header := m.planDayHeader(now)
		return header + "\n" + m.renderPlanFormView(width)
	}

	if !m.plan.loaded {
		return "Loading..."
	}

	// Build the day header.
	header := m.planDayHeader(now)

	// Leave one line for the header.
	gridHeight := height - 1
	if gridHeight < 1 {
		gridHeight = 1
	}

	grid := cli.RenderGrid(m.plan.entries, m.plan.day, now, width, m.styled, m.planGridOptions())

	// Combine header + grid, trimmed to gridHeight.
	lines := strings.Split(strings.TrimRight(grid, "\n"), "\n")
	if len(lines) > gridHeight {
		lines = lines[:gridHeight]
	}
	gridStr := strings.Join(lines, "\n")
	if gridStr != "" {
		gridStr += "\n"
	}

	return header + "\n" + gridStr
}

// planDayHeader returns the single-line day header for the in-view day.
func (m Model) planDayHeader(now time.Time) string {
	t, err := time.Parse("2006-01-02", m.plan.day)
	if err != nil {
		return m.plan.day
	}
	label := t.Format("Mon Jan 2, 2006")
	today := now.Format("2006-01-02")
	if m.plan.day == today {
		if m.styled {
			label = "\x1b[1m" + label + "\x1b[0m" + " (today)"
		} else {
			label += " (today)"
		}
	}
	return fmt.Sprintf("  %s", label)
}

// renderPlanPickerView renders the task picker overlay.
func (m Model) renderPlanPickerView(width int) string {
	var sb strings.Builder
	sb.WriteString("  Select a task to schedule:\n")

	if len(m.plan.picker.visible) == 0 {
		sb.WriteString("  (no incomplete tasks)\n")
		sb.WriteString("  [esc] cancel\n")
		return sb.String()
	}

	for i, row := range m.plan.picker.visible {
		name := row.node.Task.Name
		prefix := row.treePrefix + "  "
		if m.styled && i == m.plan.picker.cursor {
			line := prefix + name
			sb.WriteString("\x1b[1m" + padRightAnsi(line, width) + "\x1b[0m\n")
		} else if !m.styled && i == m.plan.picker.cursor {
			sb.WriteString("> " + name + "\n")
		} else {
			sb.WriteString(prefix + name + "\n")
		}
	}
	sb.WriteString("  [↑/↓] navigate  [enter] select  [esc] cancel\n")
	return sb.String()
}

// renderPlanFormView renders the current planning prompt form.
func (m Model) renderPlanFormView(width int) string {
	var sb strings.Builder

	var title string
	switch m.plan.mode {
	case planTaskTime:
		sb.WriteString("  Schedule task:\n")
		title = "Schedule task"
	case planEventForm:
		title = "Add event"
	case planRename:
		title = "Rename entry"
	case planMove:
		title = "Move entry"
	case planClear:
		title = "Clear from time"
	}
	_ = title
	_ = width

	for i, field := range m.plan.form.fields {
		label := planFieldLabel(m.plan.mode, i)
		if m.plan.form.focus == i {
			sb.WriteString(fmt.Sprintf("  > %s: %s\n", label, field.View()))
		} else {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", label, field.View()))
		}
	}
	sb.WriteString("  [tab] next field  [ctrl+s/enter] save  [esc] cancel\n")
	return sb.String()
}

// planDayTitle returns the plain-text day title for use in the pane border.
// No ANSI codes — the border renderer applies its own styling.
func (m Model) planDayTitle(now time.Time) string {
	t, err := time.Parse("2006-01-02", m.plan.day)
	if err != nil {
		return m.plan.day
	}
	label := t.Format("Mon Jan 2, 2006")
	if m.plan.day == now.Format("2006-01-02") {
		label += " (today)"
	}
	return label
}

// renderPlanGrid renders only the calendar grid rows (no day header) trimmed to
// height lines. Used for the styled two-pane path where the date lives in the pane title.
func (m Model) renderPlanGrid(width, height int, now time.Time) string {
	grid := cli.RenderGrid(m.plan.entries, m.plan.day, now, width, m.styled, m.planGridOptions())
	lines := strings.Split(strings.TrimRight(grid, "\n"), "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// renderPlanGridContent renders the day header + calendar grid for the non-styled
// fallback path where the header appears inline as the first row.
func (m Model) renderPlanGridContent(width, height int, now time.Time) string {
	header := m.planDayHeader(now)
	gridH := height - 1
	if gridH < 1 {
		gridH = 1
	}
	grid := cli.RenderGrid(m.plan.entries, m.plan.day, now, width, m.styled, m.planGridOptions())
	lines := strings.Split(strings.TrimRight(grid, "\n"), "\n")
	if len(lines) > gridH {
		lines = lines[:gridH]
	}
	return header + "\n" + strings.Join(lines, "\n")
}

// renderPlanDetail renders a read-only details pane for the given PlanEntry.
// When entry is nil (empty day or no selection), a placeholder is returned.
// For event entries (TaskId == 0), task-specific fields are omitted.
func renderPlanDetail(entry *planv1.PlanEntry, width int, styled bool) string {
	if entry == nil {
		if styled {
			return lipgloss.NewStyle().Foreground(dim).Render("(no entry selected)") + "\n"
		}
		return "(no entry selected)\n"
	}

	start := int(entry.StartMinute)
	dur := int(entry.DurationMinute)
	end := start + dur
	window := fmt.Sprintf("%02d:%02d–%02d:%02d", start/60, start%60, end/60, end%60)
	_ = width

	if !styled {
		var sb strings.Builder
		fmt.Fprintln(&sb, entry.Name)
		fmt.Fprintf(&sb, "Window:   %s\n", window)
		fmt.Fprintf(&sb, "Duration: %d min\n", dur)
		if entry.TaskId != 0 {
			status := "in progress"
			if entry.Completed {
				status = "completed"
			}
			fmt.Fprintf(&sb, "Task:     #%d  %s\n", entry.TaskId, status)
		}
		return sb.String()
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	labelStyle := lipgloss.NewStyle().Foreground(dim)
	completedStyle := lipgloss.NewStyle().Foreground(completed)

	var sb strings.Builder
	fmt.Fprintln(&sb, headerStyle.Render(entry.Name))
	fmt.Fprintf(&sb, "%s %s\n", labelStyle.Render("Window:  "), window)
	fmt.Fprintf(&sb, "%s %d min\n", labelStyle.Render("Duration:"), dur)
	if entry.TaskId != 0 {
		taskStr := fmt.Sprintf("#%d", entry.TaskId)
		if entry.Completed {
			taskStr += "  " + completedStyle.Render("completed")
		} else {
			taskStr += "  in progress"
		}
		fmt.Fprintf(&sb, "%s %s\n", labelStyle.Render("Task:    "), taskStr)
	}
	return sb.String()
}

// planFieldLabel returns a human-readable label for a planning form field.
func planFieldLabel(mode planMode, idx int) string {
	switch mode {
	case planTaskTime:
		labels := []string{"Start", "Duration (optional)"}
		if idx < len(labels) {
			return labels[idx]
		}
	case planEventForm:
		labels := []string{"Name", "Start", "Duration (optional)"}
		if idx < len(labels) {
			return labels[idx]
		}
	case planRename:
		return "Name"
	case planMove:
		labels := []string{"Start", "Duration (blank=keep)"}
		if idx < len(labels) {
			return labels[idx]
		}
	case planClear:
		return "Clear from"
	}
	return fmt.Sprintf("Field %d", idx)
}
