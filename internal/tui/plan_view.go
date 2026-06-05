package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
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
	case planTaskTime, planEventForm, planEdit:
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

	untimed, timed := splitPlanEntries(m.plan.entries)
	opts := m.planGridOptions()
	untimedStr := cli.RenderUntimed(untimed, width, m.styled, opts)
	sep := cli.RenderUntimedSeparator(untimedIDs(untimed), width)
	untimedLines := strings.Count(untimedStr, "\n") + strings.Count(sep, "\n")
	availableRows := gridHeight - untimedLines
	if availableRows < 1 {
		availableRows = 1
	}
	start, end := cli.GridWindow(timed, now, m.plan.day, availableRows)
	opts.WindowStartMin = &start
	opts.WindowEndMin = &end
	grid := cli.RenderGrid(timed, m.plan.day, now, width, m.styled, opts)

	// Combine header + untimed pane + separator + grid, trimmed to gridHeight.
	combined := untimedStr + sep + grid
	lines := strings.Split(strings.TrimRight(combined, "\n"), "\n")
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
	_ = width

	for i, field := range m.plan.form.fields {
		label := planFieldLabel(m.plan.mode, i)
		if m.plan.form.focus == i {
			sb.WriteString(fmt.Sprintf("  > %s: %s\n", label, field.View()))
		} else {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", label, field.View()))
		}
		sb.WriteString("\n")
	}

	saveBtn := "[ Save ]"
	cancelBtn := "[ Cancel ]"
	if planFocusSave(m.plan.form) {
		saveBtn = "[>Save<]"
	}
	if planFocusCancel(m.plan.form) {
		cancelBtn = "[>Cancel<]"
	}
	sb.WriteString(fmt.Sprintf("  %s  %s\n", saveBtn, cancelBtn))
	sb.WriteString("\nCtrl+S: save  Esc: cancel  Tab: next field\n")
	return sb.String()
}

// renderPlanRightPane renders the active picker or form content for the right pane.
func (m Model) renderPlanRightPane(width int) string {
	if m.plan.mode == planPickTask {
		return m.renderPlanPickerView(width)
	}
	return m.renderPlanFormView(width)
}

// planFormPaneTitle returns the pane title for the active form mode.
func (m Model) planFormPaneTitle() string {
	switch m.plan.mode {
	case planPickTask:
		return "Add task"
	case planTaskTime:
		return "Schedule"
	case planEventForm:
		return "Add event"
	case planEdit:
		return "Edit entry"
	}
	return ""
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

// renderPlanGrid renders the untimed pane (when non-empty) followed by the
// calendar grid rows, trimmed to height lines. Used in the styled two-pane path.
func (m Model) renderPlanGrid(width, height int, now time.Time) string {
	untimed, timed := splitPlanEntries(m.plan.entries)
	timedOthers, renderTimed := m.buildTimedSliceWithPreview(timed)
	opts := m.planGridOptions()
	opts.PreviewConflictSlots = planPreviewConflicts(m.buildPlanPreview(), timedOthers)
	untimedStr := cli.RenderUntimed(untimed, width, m.styled, opts)
	sep := cli.RenderUntimedSeparator(untimedIDs(untimed), width)
	untimedLines := strings.Count(untimedStr, "\n") + strings.Count(sep, "\n")
	availableRows := height - untimedLines
	if availableRows < 1 {
		availableRows = 1
	}
	start, end := cli.GridWindow(renderTimed, now, m.plan.day, availableRows)
	opts.WindowStartMin = &start
	opts.WindowEndMin = &end
	combined := untimedStr + sep + cli.RenderGrid(renderTimed, m.plan.day, now, width, m.styled, opts)
	lines := strings.Split(strings.TrimRight(combined, "\n"), "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// untimedIDs returns a slice of int (len = len(entries)) used only to check
// whether any untimed entries exist — passed to RenderUntimedSeparator.
func untimedIDs(entries []*planv1.PlanEntry) []int {
	ids := make([]int, len(entries))
	for i, e := range entries {
		ids[i] = int(e.Id)
	}
	return ids
}

// renderPlanGridContent renders the day header + untimed pane + calendar grid for
// the non-styled fallback path where the header appears inline as the first row.
func (m Model) renderPlanGridContent(width, height int, now time.Time) string {
	header := m.planDayHeader(now)
	gridH := height - 1 // subtract the inline header line
	if gridH < 1 {
		gridH = 1
	}
	untimed, timed := splitPlanEntries(m.plan.entries)
	timedOthers, renderTimed := m.buildTimedSliceWithPreview(timed)
	opts := m.planGridOptions()
	opts.PreviewConflictSlots = planPreviewConflicts(m.buildPlanPreview(), timedOthers)
	untimedStr := cli.RenderUntimed(untimed, width, m.styled, opts)
	sep := cli.RenderUntimedSeparator(untimedIDs(untimed), width)
	untimedLines := strings.Count(untimedStr, "\n") + strings.Count(sep, "\n")
	availableRows := gridH - untimedLines
	if availableRows < 1 {
		availableRows = 1
	}
	start, end := cli.GridWindow(renderTimed, now, m.plan.day, availableRows)
	opts.WindowStartMin = &start
	opts.WindowEndMin = &end
	combined := untimedStr + sep + cli.RenderGrid(renderTimed, m.plan.day, now, width, m.styled, opts)
	lines := strings.Split(strings.TrimRight(combined, "\n"), "\n")
	if len(lines) > gridH {
		lines = lines[:gridH]
	}
	return header + "\n" + strings.Join(lines, "\n")
}

// renderPlanDetail renders a read-only details pane for the given PlanEntry.
// When entry is nil (empty day or no selection), a placeholder is returned.
// For event entries (TaskId == 0), task-specific fields are omitted.
// task is the linked Task looked up from the in-memory tree (nil when absent or TaskId==0).
func renderPlanDetail(entry *planv1.PlanEntry, task *taskv1.Task, width int, styled bool) string {
	if entry == nil {
		if styled {
			return lipgloss.NewStyle().Foreground(dim).Render("(no entry selected)") + "\n"
		}
		return "(no entry selected)\n"
	}

	dur := int(entry.DurationMinute)
	_ = width

	var window string
	if entry.StartMinute != nil {
		start := int(entry.GetStartMinute())
		end := start + dur
		window = fmt.Sprintf("%02d:%02d–%02d:%02d", start/60, start%60, end/60, end%60)
	} else {
		window = "untimed"
	}

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
		if task != nil {
			if row := renderPomodoroRow(int(task.GetEstimate()), int(task.GetCompletedPomodoroCount()), false); row != "" {
				fmt.Fprintln(&sb, row)
			}
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
	if task != nil {
		if row := renderPomodoroRow(int(task.GetEstimate()), int(task.GetCompletedPomodoroCount()), true); row != "" {
			fmt.Fprintln(&sb, row)
		}
	}
	return sb.String()
}

// buildTimedSliceWithPreview returns two slices derived from timedEntries:
//
//   - timedOthers: the timed entries excluding the current edit target (m.plan.form.entryID
//     when mode == planEdit). Used for conflict detection.
//   - renderTimed: timedOthers with the preview appended (when non-nil). Passed to
//     RenderGrid and GridWindow so the grid expands to include the preview window.
//
// m.plan.entries is never modified. When no form is active, both slices equal timedEntries.
func (m Model) buildTimedSliceWithPreview(timedEntries []*planv1.PlanEntry) (timedOthers, renderTimed []*planv1.PlanEntry) {
	// Exclude the edit target from timedOthers (self-exclusion for planEdit).
	editTarget := m.plan.form.entryID // 0 when no edit form open
	timedOthers = make([]*planv1.PlanEntry, 0, len(timedEntries))
	for _, e := range timedEntries {
		if editTarget != 0 && e.Id == editTarget {
			continue
		}
		timedOthers = append(timedOthers, e)
	}

	// Append the preview when one should be shown.
	preview := m.buildPlanPreview()
	if preview != nil {
		renderTimed = make([]*planv1.PlanEntry, len(timedOthers)+1)
		copy(renderTimed, timedOthers)
		renderTimed[len(timedOthers)] = preview
	} else {
		renderTimed = timedOthers
	}
	return timedOthers, renderTimed
}

// splitPlanEntries partitions entries into untimed (nil StartMinute) and timed slices,
// preserving the original order within each group.
func splitPlanEntries(entries []*planv1.PlanEntry) (untimed, timed []*planv1.PlanEntry) {
	for _, e := range entries {
		if e.StartMinute == nil {
			untimed = append(untimed, e)
		} else {
			timed = append(timed, e)
		}
	}
	return
}

// planFocusSave reports whether the form's focus is on the Save button slot.
func planFocusSave(form planFormState) bool {
	return form.focus == len(form.fields)
}

// planFocusCancel reports whether the form's focus is on the Cancel button slot.
func planFocusCancel(form planFormState) bool {
	return form.focus == len(form.fields)+1
}

// planFieldLabel returns a human-readable label for a planning form field.
func planFieldLabel(mode planMode, idx int) string {
	switch mode {
	case planTaskTime:
		labels := []string{"Start (optional)", "Duration (optional)"}
		if idx < len(labels) {
			return labels[idx]
		}
	case planEventForm:
		labels := []string{"Name", "Start", "Duration (optional)"}
		if idx < len(labels) {
			return labels[idx]
		}
	case planEdit:
		labels := []string{"Name", "Start", "Duration"}
		if idx < len(labels) {
			return labels[idx]
		}
	}
	return fmt.Sprintf("Field %d", idx)
}
