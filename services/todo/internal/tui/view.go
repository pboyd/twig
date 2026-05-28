package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

var (
	highlightStyle = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("4")).Foreground(lipgloss.Color("15"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
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
	moveWidth := m.width - listWidth - 1

	list := m.renderList(listWidth)
	moveView := m.move.View(moveWidth, m.height-2)

	maxLines := m.height - 2
	if maxLines < 1 {
		maxLines = 1
	}

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

	status := m.renderStatus()
	return strings.Join(rows, "\n") + "\n" + status
}

func (m Model) viewList() string {
	if m.width == 0 {
		return "loading..."
	}

	listWidth := m.width / 2
	detailWidth := m.width - listWidth - 1

	list := m.renderList(listWidth)
	details := m.renderDetailPane(detailWidth)

	maxLines := m.height - 2
	if maxLines < 1 {
		maxLines = 1
	}

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

	status := m.renderStatus()
	return strings.Join(rows, "\n") + "\n" + status
}

func (m Model) viewWithForm() string {
	if m.width == 0 {
		return "loading..."
	}

	listWidth := m.width / 2
	formWidth := m.width - listWidth - 1

	list := m.renderList(listWidth)
	form := m.edit.View(formWidth)

	maxLines := m.height - 2
	if maxLines < 1 {
		maxLines = 1
	}

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

	status := m.renderStatus()
	return strings.Join(rows, "\n") + "\n" + status
}

func (m Model) renderList(width int) string {
	if len(m.visible) == 0 {
		return "(no tasks)"
	}

	var sb strings.Builder
	for i, row := range m.visible {
		prefix := row.treePrefix + row.marker + " "
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

		if i == m.cursor {
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
	return renderDetails(task, width)
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
