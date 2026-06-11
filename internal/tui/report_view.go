package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/pboyd/twig/internal/report"
)

// viewReport renders the full Report-tab content (tab bar + body + status).
func (m Model) viewReport() string {
	if m.width == 0 {
		return "loading..."
	}
	body := m.renderReportBody(m.width, m.reportBodyHeight())
	return m.renderTabBar(m.width) + "\n" + body + "\n" + m.renderStatus()
}

// reportBodyHeight returns the visible height of the Report tab's body area.
func (m Model) reportBodyHeight() int {
	h := m.height - tabBarHeight - m.statusHeight() - 1
	if h < 1 {
		h = 1
	}
	return h
}

// renderReportBody renders the scrollable content area of the Report tab.
func (m Model) renderReportBody(width, height int) string {
	if height < 1 {
		height = 1
	}

	if !m.reportData.loaded {
		return "Loading..."
	}
	if m.reportData.err != nil {
		return ""
	}

	lines := m.renderReportLines(width)
	// Apply scroll offset, clamped so an out-of-range value never wraps the
	// view back to the top (the key handler also clamps; this is a backstop).
	offset := m.reportData.scroll
	if offset > len(lines) {
		offset = len(lines)
	}
	if offset > 0 {
		lines = lines[offset:]
	}
	// Truncate to visible height.
	if len(lines) > height {
		lines = lines[:height]
	}
	// Pad to fill height so the status bar doesn't jump.
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// renderReportLines generates the full list of content lines for the Report tab.
func (m Model) renderReportLines(width int) []string {
	p := m.reportData.period
	totals := m.reportData.totals

	var lines []string

	// Header.
	header := fmt.Sprintf("What you got done — %s", p.Label)
	if m.styled {
		header = accentStyle.Render(header)
	}
	lines = append(lines, header, "")

	if p.ReportLayout() == report.AccomplishmentGrouped {
		// Totals above sections for long-range layout.
		lines = append(lines, renderTotalsLine(totals, m.styled), "")
		lines = append(lines, m.renderAccomplishmentSections(width)...)
	} else {
		// Day-grouped layout.
		groups := m.reportData.dayGroups
		if len(groups) == 0 {
			lines = append(lines, renderEmptyState(p, m.styled))
		} else {
			lines = append(lines, renderDayGroups(groups, m.styled)...)
			lines = append(lines, "")
			lines = append(lines, renderTotalsLine(totals, m.styled))
		}
	}

	return lines
}

// renderDayGroups renders all DayGroups for the day-grouped layout.
func renderDayGroups(groups []report.DayGroup, styled bool) []string {
	var lines []string
	for _, g := range groups {
		lines = append(lines, renderDayHeader(g.Day, styled))
		for _, e := range g.Entries {
			lines = append(lines, renderEntry(e, 2, styled))
		}
	}
	return lines
}

// renderDayHeader renders a day header line.
func renderDayHeader(day time.Time, styled bool) string {
	label := day.Format("Mon, Jan 2")
	if styled {
		return accentStyle.Render(label)
	}
	return label
}

// renderEntry renders a single completed entry with optional parent context.
func renderEntry(e report.Entry, indent int, styled bool) string {
	check := "✓"
	name := e.Name
	if styled {
		check = accentStyle.Render(check)
	}
	line := strings.Repeat(" ", indent) + check + " " + name
	if e.ParentName != "" {
		ctx := fmt.Sprintf("(under: %s)", e.ParentName)
		if styled {
			ctx = dimStyle.Render(ctx)
		}
		line += "  " + ctx
	}
	return line
}

// renderTotalsLine renders the summary totals line.
func renderTotalsLine(t report.Totals, styled bool) string {
	tasks := fmt.Sprintf("%d task", t.TasksCompleted)
	if t.TasksCompleted != 1 {
		tasks += "s"
	}
	poms := fmt.Sprintf("%d pomodoro", t.PomodorosCompleted)
	if t.PomodorosCompleted != 1 {
		poms += "s"
	}
	line := fmt.Sprintf("%s done · %s burned", tasks, poms)
	if styled {
		return dimStyle.Render(line)
	}
	return line
}

// renderEmptyState renders the playful empty state message.
func renderEmptyState(p report.Period, styled bool) string {
	from := p.From.Format("Jan 2")
	to := p.To.Format("Jan 2")
	var msg string
	if from == to {
		msg = fmt.Sprintf("Nothing checked off on %s — a blank page, full of potential.", from)
	} else {
		msg = fmt.Sprintf("Nothing checked off between %s and %s — a blank page, full of potential.", from, to)
	}
	if styled {
		return dimStyle.Render(msg)
	}
	return msg
}

// renderAccomplishmentSections renders the Finished and ongoing sections.
func (m Model) renderAccomplishmentSections(width int) []string {
	var lines []string

	finished := m.reportData.finished
	ongoing := m.reportData.ongoing

	if len(finished) == 0 && len(ongoing) == 0 {
		lines = append(lines, renderEmptyState(m.reportData.period, m.styled))
		return lines
	}

	if len(finished) > 0 {
		sectionHeader := "Finished"
		if m.styled {
			sectionHeader = accentStyle.Render(sectionHeader)
		}
		lines = append(lines, sectionHeader)
		for _, g := range finished {
			lines = append(lines, renderAccomplishmentGroup(g, m.styled)...)
		}
		lines = append(lines, "")
	}

	if len(ongoing) > 0 {
		sectionHeader := "Progress on ongoing work"
		if m.styled {
			sectionHeader = accentStyle.Render(sectionHeader)
		}
		lines = append(lines, sectionHeader)
		for _, g := range ongoing {
			lines = append(lines, renderAccomplishmentGroupOngoing(g, m.styled)...)
		}
	}

	return lines
}

// renderAccomplishmentGroup renders a finished AccomplishmentGroup.
func renderAccomplishmentGroup(g report.AccomplishmentGroup, styled bool) []string {
	var lines []string

	check := "✓"
	if styled {
		check = accentStyle.Render(check)
	}
	done := fmt.Sprintf("(done %s)", g.TopLevelCompletedAt.Format("Jan 2"))
	if styled {
		done = dimStyle.Render(done)
	}
	topLine := "  " + check + " " + g.TopLevelName + "  " + done
	lines = append(lines, topLine)

	for _, e := range g.Entries {
		if e.TopLevelID == g.TopLevelID && e.Depth == 0 {
			continue // skip the top-level entry itself if it appears
		}
		indent := 6 + (e.Depth-1)*4
		if indent < 6 {
			indent = 6
		}
		entryLine := strings.Repeat(" ", indent) + check + " " + e.Name
		done := fmt.Sprintf("(done %s)", e.CompletedAt.Format("Jan 2"))
		if styled {
			done = dimStyle.Render(done)
		}
		entryLine += "  " + done
		lines = append(lines, entryLine)
	}
	return lines
}

// renderAccomplishmentGroupOngoing renders an ongoing AccomplishmentGroup.
func renderAccomplishmentGroupOngoing(g report.AccomplishmentGroup, styled bool) []string {
	var lines []string

	ellipsis := "…"
	topLine := "  " + ellipsis + " " + g.TopLevelName
	if styled {
		topLine = "  " + dimStyle.Render(ellipsis+" "+g.TopLevelName)
	}
	lines = append(lines, topLine)

	check := "✓"
	if styled {
		check = accentStyle.Render(check)
	}
	for _, e := range g.Entries {
		indent := 6 + (e.Depth-1)*4
		if indent < 6 {
			indent = 6
		}
		entryLine := strings.Repeat(" ", indent) + check + " " + e.Name
		done := fmt.Sprintf("(done %s)", e.CompletedAt.Format("Jan 2"))
		if styled {
			done = dimStyle.Render(done)
		}
		entryLine += "  " + done
		lines = append(lines, entryLine)
	}
	return lines
}
