package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Semantic palette — adaptive so the app is legible on light and dark terminals.
// initPalette must be called once at startup with the detected hasDark value.
var (
	accent       color.Color
	border       color.Color
	borderActive color.Color
	dim          color.Color
	completed    color.Color
	errorColor   color.Color
	cursorBar    color.Color
	cursorBg     color.Color
	pomodoroDone color.Color
	pomodoroOver color.Color
)

var (
	highlightStyle lipgloss.Style
	errorStyle     lipgloss.Style
)

// initPalette resolves all theme colors for the detected terminal background.
// Must be called before creating any Model (i.e. before tea.NewProgram).
func initPalette(hasDark bool) {
	ld := lipgloss.LightDark(hasDark)
	accent = ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F9FFF"))
	border = ld(lipgloss.Color("#999999"), lipgloss.Color("#444444"))
	borderActive = ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F9FFF")) // = accent
	dim = ld(lipgloss.Color("#909090"), lipgloss.Color("#7A7A7A"))
	completed = ld(lipgloss.Color("#3A7A3A"), lipgloss.Color("#5AA85A"))
	errorColor = ld(lipgloss.Color("#CC0000"), lipgloss.Color("#FF5555"))
	cursorBar = ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F9FFF")) // = accent
	cursorBg = ld(lipgloss.Color("#DDEEFF"), lipgloss.Color("#1A2A3A"))
	pomodoroDone = ld(lipgloss.Color("#CC0000"), lipgloss.Color("#FF5555"))
	pomodoroOver = ld(lipgloss.Color("#B58900"), lipgloss.Color("#FFD75F"))

	highlightStyle = lipgloss.NewStyle().Bold(true).Background(accent).Foreground(lipgloss.Color("15"))
	errorStyle = lipgloss.NewStyle().Foreground(errorColor)
}
