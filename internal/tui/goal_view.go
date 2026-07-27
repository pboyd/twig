package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/goal"
	"github.com/pboyd/twig/internal/markdown"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// relativeTime formats a proto timestamp as a human-readable relative string
// (e.g. "just now", "5m ago", "2h ago", "3d ago").
func relativeTime(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	d := time.Since(ts.AsTime())
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// goalGroup holds a state-grouped set of goals for display.
type goalGroup struct {
	state goalv1.GoalState
	goals []*goalv1.Goal
}

// visibleGoals filters the goals list by visibility rules:
//   - In Progress, Incubating, and Hold are always shown.
//   - Completed and Archived are only shown when showAll is true.
func visibleGoals(goals []*goalv1.Goal, showAll bool) []*goalv1.Goal {
	var out []*goalv1.Goal
	for _, g := range goals {
		if showAll || goal.DefaultVisible(g.GetState()) {
			out = append(out, g)
		}
	}
	return out
}

// goalGroupHeaders returns goals organized into display groups in order:
// In Progress, Incubating, Hold (always); Completed, Archived (when showAll).
func goalGroupHeaders(goals []*goalv1.Goal, showAll bool) []goalGroup {
	groupMap := make(map[goalv1.GoalState][]*goalv1.Goal)
	for _, g := range goals {
		groupMap[g.GetState()] = append(groupMap[g.GetState()], g)
	}

	var groups []goalGroup
	for _, state := range goal.StateDisplayOrder() {
		if !showAll && !goal.DefaultVisible(state) {
			continue
		}
		if gs, ok := groupMap[state]; ok && len(gs) > 0 {
			groups = append(groups, goalGroup{state: state, goals: gs})
		}
	}
	return groups
}

// viewGoals renders the Goals tab content including tab bar and status bar.
func (m Model) viewGoals() string {
	if m.width == 0 {
		return "loading..."
	}

	if !m.goal.loaded {
		return m.renderTabBar(m.width) + "\nLoading...\n" + m.renderStatus()
	}

	// Status history / reader / confirm-delete: full-width single pane.
	if m.goal.mode == goalStatusHistory || m.goal.mode == goalStatusReader || m.goal.mode == goalStatusConfirmDel {
		innerH := m.height - 2 - m.statusHeight() - tabBarHeight
		if innerH < 1 {
			innerH = 1
		}
		innerW := m.width - 2
		if innerW < 0 {
			innerW = 0
		}

		var content string
		switch m.goal.mode {
		case goalStatusReader:
			content = m.renderStatusReader(innerW, innerH)
		case goalStatusConfirmDel:
			content = m.renderStatusDeleteConfirm(innerW, innerH)
		default:
			content = m.renderStatusHistory(innerW, innerH)
		}

		title := statusHistoryHeader(m.goal.mode)
		if m.styled {
			pane := paneBox(content, m.width, innerH, title, true)
			return m.renderTabBar(m.width) + "\n" + pane + "\n" + m.renderStatus()
		}

		help := goalStatusHelp(m.goal.mode)
		return m.renderTabBar(m.width) + "\n" + help + "\n" + content + "\n" + m.renderStatus()
	}

	listWidth := m.width / 2
	detailWidth := m.width - listWidth

	if m.goal.mode == goalEdit || m.goal.mode == goalNew || m.goal.mode == goalNewTask {
		// Show form in right pane.
		if m.styled {
			innerH := m.height - 2 - m.statusHeight() - tabBarHeight
			if innerH < 1 {
				innerH = 1
			}
			innerListW := listWidth - 2
			if innerListW < 0 {
				innerListW = 0
			}
			innerFormW := detailWidth - 2
			if innerFormW < 0 {
				innerFormW = 0
			}

			listContent := m.renderGoalList(innerListW)
			formContent := m.edit.View(innerFormW)
			if m.confirmingDiscard {
				overlay := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					Padding(1, 2).
					Width(innerFormW - 6).
					Align(lipgloss.Center).
					Render("Toss your edits?\n\n [y]es  [n]o — keep editing")
				formContent = lipgloss.Place(innerFormW, innerH, lipgloss.Center, lipgloss.Center, overlay)
			}

			listPane := paneBox(listContent, listWidth, innerH, "Goals", false)
			formPane := paneBox(formContent, detailWidth, innerH, "", true)

			joined := lipgloss.JoinHorizontal(lipgloss.Top, listPane, formPane)
			return m.renderTabBar(m.width) + "\n" + joined + "\n" + m.renderStatus()
		}

		detailWidth = m.width - listWidth - 1
		maxLines := m.height - 1 - m.statusHeight() - tabBarHeight
		if maxLines < 1 {
			maxLines = 1
		}
		list := m.renderGoalList(listWidth)
		form := m.edit.View(detailWidth)
		if m.confirmingDiscard {
			form = "Toss your edits? [y]es  [n]o — keep editing"
		}
		listLines := splitLines(list, maxLines)
		formLines := splitLines(form, maxLines)
		var rows []string
		for i := 0; i < maxLines; i++ {
			l := padRightAnsi(listLines[i], listWidth)
			r := ""
			if i < len(formLines) {
				r = formLines[i]
			}
			rows = append(rows, fmt.Sprintf("%s %s", l, r))
		}
		return m.renderTabBar(m.width) + "\n" + strings.Join(rows, "\n") + "\n" + m.renderStatus()
	}

	if m.styled {
		innerH := m.height - 2 - m.statusHeight() - tabBarHeight
		if innerH < 1 {
			innerH = 1
		}
		innerListW := listWidth - 2
		if innerListW < 0 {
			innerListW = 0
		}
		innerDetailW := detailWidth - 2
		if innerDetailW < 0 {
			innerDetailW = 0
		}

		listContent := m.renderGoalList(innerListW)
		detailContent := m.renderGoalDetail(innerDetailW)

		listPane := paneBox(listContent, listWidth, innerH, "Goals", true)
		detailPane := paneBox(detailContent, detailWidth, innerH, "Details", false)

		joined := lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)
		return m.renderTabBar(m.width) + "\n" + joined + "\n" + m.renderStatus()
	}

	detailWidth = m.width - listWidth - 1
	maxLines := m.height - 1 - m.statusHeight() - tabBarHeight
	if maxLines < 1 {
		maxLines = 1
	}

	list := m.renderGoalList(listWidth)
	details := m.renderGoalDetail(detailWidth)

	listLines := splitLines(list, maxLines)
	detailLines := splitLines(details, maxLines)

	var rows []string
	for i := 0; i < maxLines; i++ {
		l := padRightAnsi(listLines[i], listWidth)
		d := ""
		if i < len(detailLines) {
			d = detailLines[i]
		}
		rows = append(rows, fmt.Sprintf("%s %s", l, d))
	}

	return m.renderTabBar(m.width) + "\n" + strings.Join(rows, "\n") + "\n" + m.renderStatus()
}

// renderGoalList renders the left pane of the Goals tab.
func (m Model) renderGoalList(width int) string {
	visible := visibleGoals(m.goal.goals, m.goal.showAll)

	if len(visible) == 0 {
		empty := "A blank canvas! Press 'ctrl+n' to plant your first goal."
		if m.styled {
			return dimStyle.Render(empty)
		}
		return empty
	}

	groups := goalGroupHeaders(m.goal.goals, m.goal.showAll)

	// Build a flat index from group entries to global cursor position.
	// cursor indexes into visible[].
	var sb strings.Builder
	cursorPos := 0 // position within visible slice

	for _, grp := range groups {
		// Section header.
		header := goal.DisplayLabel(grp.state)
		if m.styled {
			sb.WriteString(dimStyle.Render(header))
		} else {
			sb.WriteString(header)
		}
		sb.WriteByte('\n')

		for _, g := range grp.goals {
			maxW := width - 2
			if maxW < 0 {
				maxW = 0
			}
			var line string
			if cursorPos == m.goal.cursor && m.styled {
				// Cursor row: render name in plain mode so there are no inner ANSI
				// reset codes that would clear the cursor background mid-line.
				plainName := m.md.RenderInline(g.GetName(), markdown.Options{Width: maxW, Styled: false})
				if lipgloss.Width(plainName) > maxW {
					plainName = ansi.Truncate(plainName, maxW, "")
				}
				line = lipgloss.NewStyle().Bold(true).Background(cursorBg).Render(padRightAnsi("  "+plainName, width))
			} else {
				renderedName := m.md.RenderInline(g.GetName(), markdown.Options{Width: maxW, Styled: m.styled})
				if lipgloss.Width(renderedName) > maxW {
					renderedName = ansi.Truncate(renderedName, maxW, "")
				}
				if cursorPos == m.goal.cursor {
					line = highlightStyle.Render(padRightAnsi("  "+renderedName, width))
				} else {
					line = padRightAnsi("  "+renderedName, width)
				}
			}

			sb.WriteString(line)
			sb.WriteByte('\n')
			cursorPos++
		}
	}

	return sb.String()
}

// renderGoalDetail renders the right pane of the Goals tab.
func (m Model) renderGoalDetail(width int) string {
	// Picker overlay (L and U modes).
	if m.goal.mode == goalPickLink || m.goal.mode == goalPickUnlink {
		return m.renderGoalPicker(width)
	}

	visible := visibleGoals(m.goal.goals, m.goal.showAll)
	if len(visible) == 0 || m.goal.cursor >= len(visible) {
		return ""
	}
	g := visible[m.goal.cursor]

	var sb strings.Builder

	// Name (bold/accent).
	// Compute two forms: plain (for unstyled output) and ANSI-stripped styled (for
	// the header, so accentStyle/Bold control all formatting without inner resets).
	displayName := m.md.RenderInline(g.GetName(), markdown.Options{Width: width, Styled: false})
	styledDisplayName := ansi.Strip(m.md.RenderInline(g.GetName(), markdown.Options{Width: width, Styled: true}))
	if m.styled {
		sb.WriteString(accentStyle.Render(lipgloss.NewStyle().Bold(true).Render(styledDisplayName)))
	} else {
		sb.WriteString(displayName)
	}
	sb.WriteByte('\n')

	// State.
	stateLine := "State: " + goal.DisplayLabel(g.GetState())
	if m.styled {
		sb.WriteString(dimStyle.Render(stateLine))
	} else {
		sb.WriteString(stateLine)
	}
	sb.WriteByte('\n')

	// Due date.
	if due := cli.FormatDue(g.GetDue()); due != "" {
		dueLine := "Due: " + due
		if m.styled {
			sb.WriteString(dimStyle.Render(dueLine))
		} else {
			sb.WriteString(dueLine)
		}
		sb.WriteByte('\n')
	}

	// Description.
	if desc := g.GetDescription(); desc != "" {
		sb.WriteByte('\n')
		sb.WriteString(m.md.Render(desc, markdown.Options{Width: width, Styled: m.styled}))
		sb.WriteByte('\n')
	}

	// Latest status update.
	sb.WriteByte('\n')
	if su := g.GetLatestStatusUpdate(); su != nil {
		statusLabel := "Latest status:"
		if m.styled {
			sb.WriteString(dimStyle.Render(statusLabel))
		} else {
			sb.WriteString(statusLabel)
		}
		sb.WriteByte('\n')
		timeStr := relativeTime(su.GetCreatedAt())
		if m.styled {
			sb.WriteString(dimStyle.Render(timeStr))
		} else {
			sb.WriteString(timeStr)
		}
		sb.WriteByte('\n')
		sb.WriteString(m.md.Render(su.GetBody(), markdown.Options{Width: width, Styled: m.styled}))
		sb.WriteByte('\n')
	} else {
		noStatus := "No status yet — how's it going?"
		if m.styled {
			sb.WriteString(dimStyle.Render(noStatus))
		} else {
			sb.WriteString(noStatus)
		}
		sb.WriteByte('\n')
		hint := "[S] add a status update  [s] history"
		if m.styled {
			sb.WriteString(dimStyle.Render(hint))
		} else {
			sb.WriteString(hint)
		}
		sb.WriteByte('\n')
	}

	// Associated task subtrees — show only incomplete root-level tasks.
	allTasks := flattenTree(m.tree)
	subtree := goal.SubtreeForGoal(allTasks, g.GetId())
	roots := cli.BuildTree(subtree)
	// Filter to incomplete roots only.
	var openRoots []*cli.TreeNode
	for _, n := range roots {
		if n.Task.GetCompletedAt() == nil {
			openRoots = append(openRoots, n)
		}
	}
	sb.WriteByte('\n')
	switch {
	case len(openRoots) > 0:
		taskLabel := "Tasks:"
		if m.styled {
			sb.WriteString(dimStyle.Render(taskLabel))
		} else {
			sb.WriteString(taskLabel)
		}
		sb.WriteByte('\n')
		sb.WriteString(renderGoalTaskTree(openRoots, m.md, m.styled, width))
	case len(subtree) > 0:
		// Tasks exist but none are incomplete root-level tasks.
		msg := "No open tasks right now — time to plan your next step?"
		if m.styled {
			sb.WriteString(dimStyle.Render(msg))
		} else {
			sb.WriteString(msg)
		}
		sb.WriteByte('\n')
	default:
		noTasks := "No tasks attached yet — every great goal starts as a wish."
		if m.styled {
			sb.WriteString(dimStyle.Render(noTasks))
		} else {
			sb.WriteString(noTasks)
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// renderGoalPicker renders the task picker overlay for L (link) and U (unlink) modes.
func (m Model) renderGoalPicker(width int) string {
	var sb strings.Builder
	label := "Link task (↑/↓: nav  enter: pick  esc: cancel)"
	if m.goal.mode == goalPickUnlink {
		label = "Unlink task (↑/↓: nav  enter: pick  esc: cancel)"
	}
	if m.styled {
		sb.WriteString(dimStyle.Render(label))
	} else {
		sb.WriteString(label)
	}
	sb.WriteByte('\n')

	if len(m.goal.picker.visible) == 0 {
		sb.WriteString("(no tasks)\n")
		return sb.String()
	}
	for i, row := range m.goal.picker.visible {
		indent := strings.Repeat("  ", row.depth)
		maxW := width - lipgloss.Width(indent)
		if maxW < 0 {
			maxW = 0
		}
		var line string
		if i == m.goal.picker.cursor && m.styled {
			// Cursor row: render name in plain mode so there are no inner ANSI
			// reset codes that would clear the cursor background mid-line.
			plainName := m.md.RenderInline(row.node.Task.Name, markdown.Options{Width: maxW, Styled: false})
			if lipgloss.Width(plainName) > maxW {
				plainName = ansi.Truncate(plainName, maxW, "")
			}
			line = lipgloss.NewStyle().Bold(true).Background(cursorBg).Render(padRightAnsi(indent+plainName, width))
		} else {
			taskName := m.md.RenderInline(row.node.Task.Name, markdown.Options{Width: maxW, Styled: m.styled})
			if lipgloss.Width(taskName) > maxW {
				taskName = ansi.Truncate(taskName, maxW, "")
			}
			line = indent + taskName
			if i == m.goal.picker.cursor {
				line = highlightStyle.Render(padRightAnsi(line, width))
			}
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// renderGoalTaskTree renders a flat list of root-level task names in the goal
// detail pane. Children are intentionally not rendered — callers should pass
// only the root nodes they want displayed.
func renderGoalTaskTree(roots []*cli.TreeNode, md *markdown.Renderer, styled bool, width int) string {
	const indent = "  "
	maxW := width - lipgloss.Width(indent)
	if maxW < 0 {
		maxW = 0
	}
	var sb strings.Builder
	for _, n := range roots {
		taskName := md.RenderInline(n.Task.Name, markdown.Options{Width: maxW, Styled: styled})
		if lipgloss.Width(taskName) > maxW {
			taskName = ansi.Truncate(taskName, maxW, "")
		}
		line := indent + taskName
		if due := cli.FormatDue(n.Task.Due); due != "" {
			line += "  " + due
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// goalToFakeTask converts a Goal into a fake taskv1.Task so we can reuse
// the editFormModel for goal editing (name, description, due only).
func goalToFakeTask(g *goalv1.Goal) *taskv1.Task {
	return &taskv1.Task{
		Id:          g.GetId(),
		Name:        g.GetName(),
		Description: g.GetDescription(),
		Due:         g.GetDue(),
	}
}
