package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// View renders the current model state to a string.
func (m Model) View() string {
	switch m.mode {
	case modeHelp:
		return m.viewHelp()
	case modeEdit, modeNewSubtask, modeNewRoot:
		return m.viewWithForm()
	case modeMove:
		return m.viewWithMove()
	default:
		return m.viewList()
	}
}

func (m Model) viewWithMove() string {
	if m.width == 0 || m.move == nil {
		return "loading..."
	}

	listWidth := m.width / 2
	moveWidth := m.width - listWidth

	if m.styled {
		innerH := m.height - 3
		if innerH < 1 {
			innerH = 1
		}
		innerListW := listWidth - 2
		if innerListW < 0 {
			innerListW = 0
		}
		innerMoveW := moveWidth - 2
		if innerMoveW < 0 {
			innerMoveW = 0
		}

		listContent := m.renderList(innerListW)
		moveContent := m.move.View(innerMoveW, innerH)

		listPane := paneBox(listContent, listWidth, innerH, "Tasks", false)
		movePane := paneBox(moveContent, moveWidth, innerH, "Move", true)

		joined := lipgloss.JoinHorizontal(lipgloss.Top, listPane, movePane)
		return joined + "\n" + m.renderStatus()
	}

	moveWidth = m.width - listWidth - 1
	maxLines := m.height - 2
	if maxLines < 1 {
		maxLines = 1
	}

	list := m.renderList(listWidth)
	moveView := m.move.View(moveWidth, m.height-2)

	listLines := splitLines(list, maxLines)
	moveLines := splitLines(moveView, maxLines)

	var rows []string
	for i := 0; i < maxLines; i++ {
		l := padRightAnsi(listLines[i], listWidth)
		mv := ""
		if i < len(moveLines) {
			mv = moveLines[i]
		}
		rows = append(rows, fmt.Sprintf("%s %s", l, mv))
	}

	return strings.Join(rows, "\n") + "\n" + m.renderStatus()
}

func (m Model) viewList() string {
	if m.width == 0 {
		return "loading..."
	}

	listWidth := m.width / 2
	detailWidth := m.width - listWidth

	if m.styled {
		innerH := m.height - 3
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

		listContent := m.renderList(innerListW)
		detailContent := m.renderDetailPane(innerDetailW)

		listPane := paneBox(listContent, listWidth, innerH, "Tasks", true)
		detailPane := paneBox(detailContent, detailWidth, innerH, "Details", false)

		joined := lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)
		return joined + "\n" + m.renderStatus()
	}

	detailWidth = m.width - listWidth - 1
	maxLines := m.height - 2
	if maxLines < 1 {
		maxLines = 1
	}

	list := m.renderList(listWidth)
	details := m.renderDetailPane(detailWidth)

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

	return strings.Join(rows, "\n") + "\n" + m.renderStatus()
}

func (m Model) viewWithForm() string {
	if m.width == 0 {
		return "loading..."
	}

	listWidth := m.width / 2
	formWidth := m.width - listWidth

	if m.styled {
		innerH := m.height - 3
		if innerH < 1 {
			innerH = 1
		}
		innerListW := listWidth - 2
		if innerListW < 0 {
			innerListW = 0
		}
		innerFormW := formWidth - 2
		if innerFormW < 0 {
			innerFormW = 0
		}

		listContent := m.renderList(innerListW)
		formContent := m.edit.View(innerFormW)

		listPane := paneBox(listContent, listWidth, innerH, "Tasks", false)
		formPane := paneBox(formContent, formWidth, innerH, "", true)

		joined := lipgloss.JoinHorizontal(lipgloss.Top, listPane, formPane)
		return joined + "\n" + m.renderStatus()
	}

	formWidth = m.width - listWidth - 1
	maxLines := m.height - 2
	if maxLines < 1 {
		maxLines = 1
	}

	list := m.renderList(listWidth)
	form := m.edit.View(formWidth)

	listLines := splitLines(list, maxLines)
	formLines := splitLines(form, maxLines)

	var rows []string
	for i := 0; i < maxLines; i++ {
		l := padRightAnsi(listLines[i], listWidth)
		f := ""
		if i < len(formLines) {
			f = formLines[i]
		}
		rows = append(rows, fmt.Sprintf("%s %s", l, f))
	}

	return strings.Join(rows, "\n") + "\n" + m.renderStatus()
}

func (m Model) renderList(width int) string {
	if len(m.visible) == 0 {
		return "(no tasks)"
	}

	var sb strings.Builder
	for i, row := range m.visible {
		var prefix string
		if m.styled {
			var chevron string
			switch {
			case row.expandable && row.expanded:
				chevron = "▾"
			case row.expandable && !row.expanded:
				chevron = "▸"
			default:
				chevron = " "
			}
			var checkbox string
			if row.node.Task.GetCompletedAt() != nil {
				checkbox = "☑"
			} else {
				checkbox = "☐"
			}
			prefix = row.treePrefix + chevron + " " + checkbox + " "
		} else {
			prefix = row.treePrefix + " "
		}

		prefixW := lipgloss.Width(prefix)
		maxNameW := width - prefixW
		if maxNameW < 0 {
			maxNameW = 0
		}

		name := row.node.Task.Name
		if len(name) > maxNameW {
			name = name[:maxNameW]
		}
		if row.node.Task.GetCompletedAt() != nil {
			name = cli.Strike(name, m.styled)
		}

		line := prefix + name

		if i == m.cursor && m.styled {
			line = lipgloss.NewStyle().Bold(true).Background(cursorBg).Render(padRightAnsi(line, width))
		} else if i == m.cursor {
			line = highlightStyle.Render(padRightAnsi(line, width))
		} else {
			line = padRightAnsi(line, width)
		}

		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (m Model) renderDetailPane(width int) string {
	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		return ""
	}
	task := m.visible[m.cursor].node.Task
	return renderDetails(task, width, m.styled)
}

func (m Model) renderStatus() string {
	if m.err != nil {
		return errorStyle.Render("error: " + m.err.Error())
	}
	return m.help.View(m.keys)
}

func (m Model) viewHelp() string {
	return m.help.FullHelpView(m.keys.FullHelp())
}

// paneBox wraps content in a rounded-border box with an optional title.
// outerWidth is the total visual width including borders.
// innerHeight is the content height (not including border lines).
// focused selects the accent border color; inactive uses the dim border color.
func paneBox(content string, outerWidth, innerHeight int, title string, focused bool) string {
	innerW := outerWidth - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	borderColor := border
	if focused {
		borderColor = borderActive
	}

	rendered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(innerW).
		Height(innerHeight).
		Render(content)

	if title == "" {
		return rendered
	}

	// Embed title in the top border line: ╭─ Title ──...──╮
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}

	titleFill := "─ " + title + " "
	titleW := lipgloss.Width(titleFill)
	dashCount := innerW - titleW
	if dashCount < 0 {
		dashCount = 0
	}
	topBorder := "╭" + titleFill + strings.Repeat("─", dashCount) + "╮"
	lines[0] = lipgloss.NewStyle().Foreground(borderColor).Render(topBorder)

	return strings.Join(lines, "\n")
}

// splitLines splits s by newlines and returns exactly n lines (padding with empty
// strings if needed, truncating if too many).
func splitLines(s string, n int) []string {
	lines := strings.Split(s, "\n")
	if len(lines) >= n {
		return lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines
}

// padRightAnsi pads s to at least width visible columns, using lipgloss.Width
// to measure the visible width (ignoring ANSI escape codes). Lines that already
// meet or exceed width are returned unchanged so ANSI codes are never truncated.
func padRightAnsi(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
