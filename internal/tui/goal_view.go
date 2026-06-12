package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/goal"
	"github.com/pboyd/twig/internal/markdown"
)

// goalGroup holds a state-grouped set of goals for display.
type goalGroup struct {
	state goalv1.GoalState
	goals []*goalv1.Goal
}

// visibleGoals filters the goals list by visibility rules:
//   - Committed and Incubating are always shown.
//   - Completed and Archived are only shown when showAll is true.
func visibleGoals(goals []*goalv1.Goal, showAll bool) []*goalv1.Goal {
	var out []*goalv1.Goal
	for _, g := range goals {
		switch g.GetState() {
		case goalv1.GoalState_GOAL_STATE_COMPLETED, goalv1.GoalState_GOAL_STATE_ARCHIVED:
			if showAll {
				out = append(out, g)
			}
		default:
			out = append(out, g)
		}
	}
	return out
}

// goalGroupHeaders returns goals organized into display groups in order:
// Committed, Incubating (always); Completed, Archived (when showAll).
func goalGroupHeaders(goals []*goalv1.Goal, showAll bool) []goalGroup {
	groupMap := make(map[goalv1.GoalState][]*goalv1.Goal)
	for _, g := range goals {
		groupMap[g.GetState()] = append(groupMap[g.GetState()], g)
	}

	var groups []goalGroup
	orderedStates := []goalv1.GoalState{
		goalv1.GoalState_GOAL_STATE_COMMITTED,
		goalv1.GoalState_GOAL_STATE_INCUBATING,
	}
	if showAll {
		orderedStates = append(orderedStates,
			goalv1.GoalState_GOAL_STATE_COMPLETED,
			goalv1.GoalState_GOAL_STATE_ARCHIVED,
		)
	}

	for _, state := range orderedStates {
		if gs, ok := groupMap[state]; ok && len(gs) > 0 {
			groups = append(groups, goalGroup{state: state, goals: gs})
		}
	}
	return groups
}

// goalStateName returns the display name for a GoalState.
func goalStateName(state goalv1.GoalState) string {
	switch state {
	case goalv1.GoalState_GOAL_STATE_COMMITTED:
		return "Committed"
	case goalv1.GoalState_GOAL_STATE_INCUBATING:
		return "Incubating"
	case goalv1.GoalState_GOAL_STATE_COMPLETED:
		return "Completed"
	case goalv1.GoalState_GOAL_STATE_ARCHIVED:
		return "Archived"
	default:
		return "Unknown"
	}
}

// viewGoals renders the Goals tab content including tab bar and status bar.
func (m Model) viewGoals() string {
	if m.width == 0 {
		return "loading..."
	}

	if !m.goal.loaded {
		return m.renderTabBar(m.width) + "\nLoading...\n" + m.renderStatus()
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
		empty := "A blank canvas! Press 'n' to plant your first goal."
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
		header := goalStateName(grp.state)
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
			renderedName := m.md.RenderInline(g.GetName(), markdown.Options{Width: maxW, Styled: m.styled})
			if lipgloss.Width(renderedName) > maxW {
				renderedName = ansi.Truncate(renderedName, maxW, "")
			}
			line := "  " + renderedName
			if due := cli.FormatDue(g.GetDue()); due != "" {
				line += "  " + due
			}

			if cursorPos == m.goal.cursor && m.styled {
				line = lipgloss.NewStyle().Bold(true).Background(cursorBg).Render(padRightAnsi(line, width))
			} else if cursorPos == m.goal.cursor {
				line = highlightStyle.Render(padRightAnsi(line, width))
			} else {
				line = padRightAnsi(line, width)
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
	renderedName := m.md.RenderInline(g.GetName(), markdown.Options{Width: width, Styled: m.styled})
	if m.styled {
		sb.WriteString(accentStyle.Render(lipgloss.NewStyle().Bold(true).Render(renderedName)))
	} else {
		sb.WriteString(renderedName)
	}
	sb.WriteByte('\n')

	// State.
	stateLine := "State: " + goalStateName(g.GetState())
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

	// Associated task subtrees.
	allTasks := flattenTree(m.tree)
	subtree := goal.SubtreeForGoal(allTasks, g.GetId())
	if len(subtree) > 0 {
		sb.WriteByte('\n')
		taskLabel := "Tasks:"
		if m.styled {
			sb.WriteString(dimStyle.Render(taskLabel))
		} else {
			sb.WriteString(taskLabel)
		}
		sb.WriteByte('\n')
		// Build a mini-tree from the subtree and render it.
		roots := cli.BuildTree(subtree)
		rendered := renderGoalTaskTree(roots, m.md, m.styled, width)
		sb.WriteString(rendered)
	} else {
		sb.WriteByte('\n')
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
		taskName := m.md.RenderInline(row.node.Task.Name, markdown.Options{Width: maxW, Styled: m.styled})
		if lipgloss.Width(taskName) > maxW {
			taskName = ansi.Truncate(taskName, maxW, "")
		}
		line := indent + taskName
		if i == m.goal.picker.cursor {
			if m.styled {
				line = lipgloss.NewStyle().Bold(true).Background(cursorBg).Render(padRightAnsi(line, width))
			} else {
				line = highlightStyle.Render(padRightAnsi(line, width))
			}
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// renderGoalTaskTree renders a simple flat list of task names in the goal detail pane.
func renderGoalTaskTree(roots []*cli.TreeNode, md *markdown.Renderer, styled bool, width int) string {
	var sb strings.Builder
	var walk func(nodes []*cli.TreeNode, depth int)
	walk = func(nodes []*cli.TreeNode, depth int) {
		for _, n := range nodes {
			indent := strings.Repeat("  ", depth+1)
			maxW := width - lipgloss.Width(indent)
			if maxW < 0 {
				maxW = 0
			}
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
			walk(n.Children, depth+1)
		}
	}
	walk(roots, 0)
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
