package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Glyphs used in the pomodoro progress row. Text-presentation characters from
// the Geometric Shapes block are used instead of emoji so that all terminal
// emulators draw them from the text font and respect ANSI color/bold styling.
const (
	glyphDone   = "●" // completed within estimate   (bold red)
	glyphRemain = "○" // estimated but not yet done   (dim)
	glyphOver   = "◆" // completed beyond estimate    (bold yellow)
)

// renderPomodoroRow renders a row of progress glyphs representing pomodoro
// progress: ● done, ○ remaining, ◆ over-estimate.
// estimate == 0 && completed == 0 returns "" (caller omits the line entirely).
// When styled, segments are colored: bold-red (done), dim (remaining), bold-yellow (over).
// When not styled, emits plain glyphs with no ANSI codes; distinct shapes make
// state readable even without color.
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
		var sb strings.Builder
		for i := 0; i < done; i++ {
			sb.WriteString(glyphDone)
		}
		for i := 0; i < remain; i++ {
			sb.WriteString(glyphRemain)
		}
		for i := 0; i < over; i++ {
			sb.WriteString(glyphOver)
		}
		return sb.String()
	}

	doneStyle := lipgloss.NewStyle().Bold(true).Foreground(pomodoroDone)
	dimStyle := lipgloss.NewStyle().Foreground(dim)
	overStyle := lipgloss.NewStyle().Bold(true).Foreground(pomodoroOver)

	var sb strings.Builder
	for i := 0; i < done; i++ {
		sb.WriteString(doneStyle.Render(glyphDone))
	}
	for i := 0; i < remain; i++ {
		sb.WriteString(dimStyle.Render(glyphRemain))
	}
	for i := 0; i < over; i++ {
		sb.WriteString(overStyle.Render(glyphOver))
	}
	return sb.String()
}
