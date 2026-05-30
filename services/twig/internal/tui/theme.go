package tui

import "github.com/charmbracelet/lipgloss"

// Semantic palette — adaptive so the app is legible on light and dark terminals.
var (
	accent       = lipgloss.AdaptiveColor{Light: "#005FD7", Dark: "#5F9FFF"}
	border       = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#444444"}
	borderActive = lipgloss.AdaptiveColor{Light: "#005FD7", Dark: "#5F9FFF"} // = accent
	dim          = lipgloss.AdaptiveColor{Light: "#909090", Dark: "#7A7A7A"}
	completed    = lipgloss.AdaptiveColor{Light: "#3A7A3A", Dark: "#5AA85A"}
	errorColor   = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF5555"}
	cursorBar    = lipgloss.AdaptiveColor{Light: "#005FD7", Dark: "#5F9FFF"} // = accent
	cursorBg     = lipgloss.AdaptiveColor{Light: "#DDEEFF", Dark: "#1A2A3A"}
)

var (
	highlightStyle = lipgloss.NewStyle().Bold(true).Background(accent).Foreground(lipgloss.Color("15"))
	errorStyle     = lipgloss.NewStyle().Foreground(errorColor)
)
