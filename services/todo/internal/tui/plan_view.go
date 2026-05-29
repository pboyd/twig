package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// renderTabBar renders the one-line tab bar: " Tasks | Planning "
// with the active tab highlighted in styled mode.
func (m Model) renderTabBar(width int) string {
	tasks := " Tasks "
	planning := " Planning "

	if m.styled {
		sep := "│"
		var taskStr, planStr string
		if m.activeTab == tabTasks {
			taskStr = "\x1b[1m" + tasks + "\x1b[0m"
			planStr = planning
		} else {
			taskStr = tasks
			planStr = "\x1b[1m" + planning + "\x1b[0m"
		}
		bar := taskStr + sep + planStr
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
