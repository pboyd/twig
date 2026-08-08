package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/markdown"
)

// renderPlanNotesPane renders the Notes pane for the planning tab. Returns
// either the notes content (when set) or an empty placeholder pane. The pane is
// always rendered when notes editing is not active and the right column is
// showing details. Width is the pane's inner content width; height bounds the
// rendered content.
func (m Model) renderPlanNotesPane(innerWidth, innerHeight int) string {
	if m.md != nil {
		rendered := m.md.Render(m.plan.notes, markdown.Options{Width: innerWidth, Styled: m.styled})
		return clampLines(rendered, innerHeight)
	}
	return clampLines(m.plan.notes, innerHeight)
}

// renderPlanNotesEditor renders the full-right-column notes editor for use
// while mode == planNotesEdit. The textarea fills the available height.
func (m Model) renderPlanNotesEditor(width, height int) string {
	ta := m.plan.notesInput
	ta.SetWidth(width)
	ta.SetHeight(height)
	return ta.View()
}

// renderPlanObjectiveBand renders the day's objective band above the planning
// tab. Returns "" when not shown (objective empty AND no editor open).
// innerWidth is the content width available inside the surrounding border.
func (m Model) renderPlanObjectiveBand(innerWidth int) string {
	editing := m.plan.mode == planObjectiveEdit
	if !editing && m.plan.objective == "" {
		return ""
	}

	var inner string
	if editing {
		m.plan.objectiveInput.SetWidth(innerWidth)
		inner = m.plan.objectiveInput.View()
	} else if m.md != nil {
		inner = m.md.Render(m.plan.objective, markdown.Options{Width: innerWidth, Styled: m.styled})
	} else {
		inner = m.plan.objective
	}

	// 2 lines of border (top + bottom); clamped to 3 inner.
	inner = clampLines(inner, 3)

	if !m.styled {
		// Unstyled: render as a plain block — a heading line plus body, no border.
		header := "Objective:"
		if editing {
			header = "Objective (Enter to save, Esc to cancel):"
		}
		body := clampLines(inner, 3)
		return header + "\n" + body + "\n"
	}

	renderedH := strings.Count(inner, "\n") + 1
	rendered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(innerWidth + 2).
		Height(renderedH + 2).
		Render(inner)

	// Embed title in top border.
	title := " Objective "
	if editing {
		title = " Objective (Enter to save, Esc to cancel) "
	}
	innerW := innerWidth
	titleW := lipgloss.Width(title)
	dashCount := innerW - titleW
	if dashCount < 0 {
		dashCount = 0
	}
	topBorder := "╭" + title + strings.Repeat("─", dashCount) + "╮"
	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		lines[0] = lipgloss.NewStyle().Foreground(border).Render(topBorder)
	}
	return strings.Join(lines, "\n")
}

// renderTabBar renders the one-line tab bar: " Goals │ Tasks │ Planning │ Report "
// with the active tab highlighted in accent color (styled) or plain text (unstyled).
func (m Model) renderTabBar(width int) string {
	goals := " Goals "
	tasks := " Tasks "
	planning := " Planning "
	reportLabel := " Report "

	if m.styled {
		// Active tab: bold blue accent. Inactive: dim.
		const (
			accentOpen = "\x1b[1;34m" // bold + blue
			dimOpen    = "\x1b[2m"    // dim
			reset      = "\x1b[0m"
			sepDim     = "\x1b[2m│\x1b[0m"
		)
		renderTab := func(label string, active bool) string {
			if active {
				return accentOpen + label + reset
			}
			return dimOpen + label + reset
		}
		bar := renderTab(goals, m.activeTab == tabGoals) +
			sepDim +
			renderTab(tasks, m.activeTab == tabTasks) +
			sepDim +
			renderTab(planning, m.activeTab == tabPlanning) +
			sepDim +
			renderTab(reportLabel, m.activeTab == tabReport)
		visW := lipgloss.Width(bar)
		if width > visW {
			bar += strings.Repeat(" ", width-visW)
		}
		return bar
	}

	sep := "|"
	bar := goals + sep + tasks + sep + planning + sep + reportLabel
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
		prefix := row.treePrefix + "  "
		maxW := width - lipgloss.Width(prefix)
		if maxW < 0 {
			maxW = 0
		}
		name := m.md.RenderInline(row.node.Task.Name, markdown.Options{Width: maxW, Styled: m.styled})
		if lipgloss.Width(name) > maxW {
			name = ansi.Truncate(name, maxW, "")
		}
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

	fieldWidth := width - 25
	if fieldWidth < 20 {
		fieldWidth = 20
	}

	for i, field := range m.plan.form.fields {
		field.SetWidth(fieldWidth)
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
	sb.WriteString("\nCtrl+S: save  Tab: next field  Esc: cancel\n")
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
	return clampLines(strings.TrimRight(combined, "\n"), height)
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
	return header + "\n" + clampLines(strings.TrimRight(combined, "\n"), gridH)
}

// renderPlanDetail renders a read-only details pane for the given PlanEntry.
// When entry is nil (empty day or no selection), a placeholder is returned.
// For event entries (TaskId == 0), task-specific fields are omitted.
// task is the linked Task looked up from the in-memory tree (nil when absent or TaskId==0).
//
// When task has a non-empty description, it is rendered below the pomodoro row
// using the shared markdown renderer (md.Render with block-level options), or
// wrapDescription when md is nil. Both paths emit one blank separator line above
// the description and never synthesize an empty-state placeholder.
func renderPlanDetail(entry *planv1.PlanEntry, task *taskv1.Task, width int, styled bool, md *markdown.Renderer) string {
	if entry == nil {
		if styled {
			return lipgloss.NewStyle().Foreground(dim).Render("(no entry selected)") + "\n"
		}
		return "(no entry selected)\n"
	}

	dur := int(entry.DurationMinute)

	var window string
	if entry.StartMinute != nil {
		start := int(entry.GetStartMinute())
		end := start + dur
		window = fmt.Sprintf("%02d:%02d–%02d:%02d", start/60, start%60, end/60, end%60)
	} else {
		window = "untimed"
	}

	entryNamePlain := entry.Name
	entryNameStyled := entry.Name
	if md != nil {
		entryNamePlain = md.RenderInline(entry.Name, markdown.Options{Width: width, Styled: false})
		entryNameStyled = md.RenderInline(entry.Name, markdown.Options{Width: width, Styled: true})
	}

	if !styled {
		var sb strings.Builder
		fmt.Fprintln(&sb, entryNamePlain)
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
		if entry.TaskId != 0 && task != nil && strings.TrimSpace(task.GetDescription()) != "" {
			fmt.Fprintln(&sb)
			fmt.Fprintln(&sb, renderTaskDescription(md, task.GetDescription(), width, false))
		}
		return sb.String()
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	labelStyle := lipgloss.NewStyle().Foreground(dim)
	completedStyle := lipgloss.NewStyle().Foreground(completed)

	var sb strings.Builder
	fmt.Fprintln(&sb, headerStyle.Render(entryNameStyled))
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
	if entry.TaskId != 0 && task != nil && strings.TrimSpace(task.GetDescription()) != "" {
		fmt.Fprintln(&sb)
		fmt.Fprintln(&sb, renderTaskDescription(md, task.GetDescription(), width, true))
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
	// Exclude the edit target from timedOthers only while the edit form is open.
	// form.entryID is never cleared when the form closes, so reading it
	// unconditionally would permanently hide the entry after editing.
	editTarget := int32(0)
	if m.plan.mode == planEdit {
		editTarget = m.plan.form.entryID
	}
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
