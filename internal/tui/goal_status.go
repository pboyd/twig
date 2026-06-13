package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	"github.com/pboyd/twig/internal/markdown"
)

// renderStatusHistory renders the status-update history list as full-view content.
func (m Model) renderStatusHistory(width, height int) string {
	updates := m.goal.statusUpdates

	if len(updates) == 0 {
		empty := "No status updates yet — press S to jot one down!"
		if m.styled {
			return dimStyle.Render(empty) + "\n"
		}
		return empty + "\n"
	}

	var sb strings.Builder
	for i, su := range updates {
		timeStr := relativeTime(su.GetCreatedAt())
		// First line of body (up to width).
		body := su.GetBody()
		firstLine := body
		if idx := strings.IndexByte(body, '\n'); idx >= 0 {
			firstLine = body[:idx]
		}
		maxBodyW := width - len(timeStr) - 2
		if maxBodyW < 0 {
			maxBodyW = 0
		}
		// Inline-render to apply markdown (bold/italic) without block wrapping.
		inlineBody := m.md.RenderInline(firstLine, markdown.Options{Width: maxBodyW, Styled: m.styled})
		if lipgloss.Width(inlineBody) > maxBodyW && maxBodyW > 3 {
			inlineBody = ansi.Truncate(inlineBody, maxBodyW, "…")
		}

		line := fmt.Sprintf("%s  %s", timeStr, inlineBody)

		if i == m.goal.statusCursor {
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

// renderStatusReader renders the full text of the selected status update with scroll.
func (m Model) renderStatusReader(width, height int) string {
	updates := m.goal.statusUpdates
	if len(updates) == 0 || m.goal.statusCursor >= len(updates) {
		return ""
	}
	su := updates[m.goal.statusCursor]

	var sb strings.Builder

	// Header: timestamp.
	header := relativeTime(su.GetCreatedAt())
	if m.styled {
		sb.WriteString(dimStyle.Render(header))
	} else {
		sb.WriteString(header)
	}
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Body rendered as markdown.
	rendered := m.md.Render(su.GetBody(), markdown.Options{Width: width, Styled: m.styled})
	lines := strings.Split(rendered, "\n")

	offset := m.goal.readerOffset
	if offset > len(lines) {
		offset = len(lines)
	}
	visible := lines[offset:]

	// Trim to fit height (leave 2 lines for header + blank).
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}
	if len(visible) > contentHeight {
		visible = visible[:contentHeight]
	}

	for _, l := range visible {
		sb.WriteString(l)
		sb.WriteByte('\n')
	}

	// Scroll indicator.
	if offset > 0 || len(lines)-offset > contentHeight {
		indicator := fmt.Sprintf("── line %d/%d ──", offset+1, len(lines))
		if m.styled {
			sb.WriteString(dimStyle.Render(indicator))
		} else {
			sb.WriteString(indicator)
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// renderStatusDeleteConfirm renders the delete-confirmation prompt.
func (m Model) renderStatusDeleteConfirm(width, height int) string {
	var sb strings.Builder
	// Show the history list dimmed behind the prompt.
	sb.WriteString(m.renderStatusHistory(width, height-2))
	prompt := "Delete this status update? [y]es  [n]o"
	if m.styled {
		sb.WriteString(accentStyle.Render(prompt))
	} else {
		sb.WriteString(prompt)
	}
	sb.WriteByte('\n')
	return sb.String()
}

// statusHistoryHeader renders the pane title for the status history/reader views.
func statusHistoryHeader(mode goalViewMode) string {
	switch mode {
	case goalStatusReader:
		return "Status Update"
	default:
		return "Status History"
	}
}

// goalStatusHelp returns contextual help text for the current status mode.
func goalStatusHelp(mode goalViewMode) string {
	switch mode {
	case goalStatusReader:
		return "↑/↓ scroll  esc back"
	case goalStatusHistory:
		return "↑/↓ nav  enter read  S add  e edit  d delete  esc back"
	case goalStatusConfirmDel:
		return "[y]es  [n]o"
	default:
		return ""
	}
}

// formatGoalStatusUpdatesCount returns a pluralised count string.
func formatGoalStatusUpdatesCount(updates []*goalv1.StatusUpdate) string {
	n := len(updates)
	if n == 1 {
		return "1 update"
	}
	return fmt.Sprintf("%d updates", n)
}

// statusReaderLines returns the rendered body lines for the currently selected
// status update, using the model's current width and style settings. Returns
// nil when there is no selection.
func (m Model) statusReaderLines() []string {
	updates := m.goal.statusUpdates
	if len(updates) == 0 || m.goal.statusCursor >= len(updates) {
		return nil
	}
	su := updates[m.goal.statusCursor]
	rendered := m.md.Render(su.GetBody(), markdown.Options{Width: m.width - 2, Styled: m.styled})
	return strings.Split(rendered, "\n")
}

// statusReaderContentHeight returns the number of body lines visible in the
// reader pane given the current terminal height.
func (m Model) statusReaderContentHeight() int {
	// innerH = m.height - 2 - statusHeight - tabBarHeight  (pane border/title)
	// contentHeight = innerH - 2  (header line + blank line before body)
	h := m.height - 4 - m.statusHeight() - tabBarHeight
	if h < 1 {
		h = 1
	}
	return h
}
