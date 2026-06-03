package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderPomodoroRow renders a row of tomato glyphs representing pomodoro progress.
// estimate == 0 && completed == 0 returns "" (caller omits the line entirely).
// When styled, segments are colored: bold-red (done), dim (remaining), bold-yellow (over).
// When not styled, emits n plain glyphs with no ANSI codes.
func renderPomodoroRow(estimate, completed int, styled bool) string {
	n := estimate
	if completed > n {
		n = completed
	}
	if n == 0 {
		return ""
	}

	done := completed
	if done > estimate {
		done = estimate
	}
	remain := estimate - done
	over := completed - estimate
	if over < 0 {
		over = 0
	}

	if !styled {
		return strings.Repeat("🍅", n)
	}

	doneStyle := lipgloss.NewStyle().Bold(true).Foreground(pomodoroDone)
	dimStyle := lipgloss.NewStyle().Foreground(dim)
	overStyle := lipgloss.NewStyle().Bold(true).Foreground(pomodoroOver)

	var sb strings.Builder
	for i := 0; i < done; i++ {
		sb.WriteString(doneStyle.Render("🍅"))
	}
	for i := 0; i < remain; i++ {
		sb.WriteString(dimStyle.Render("🍅"))
	}
	for i := 0; i < over; i++ {
		sb.WriteString(overStyle.Render("🍅"))
	}
	return sb.String()
}
