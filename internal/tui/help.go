package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

// newHelpModel returns a bubbles help.Model with a slightly lighter dark-mode
// palette than the default, so the help bar is easier to read on dark terminals.
func newHelpModel(hasDark bool) help.Model {
	m := help.New()

	ld := lipgloss.LightDark(hasDark)
	keyStyle := lipgloss.NewStyle().Foreground(ld(lipgloss.Color("#909090"), lipgloss.Color("#9A9A9A")))
	descStyle := lipgloss.NewStyle().Foreground(ld(lipgloss.Color("#B2B2B2"), lipgloss.Color("#7A7A7A")))
	sepStyle := lipgloss.NewStyle().Foreground(ld(lipgloss.Color("#DDDADA"), lipgloss.Color("#5A5A5A")))

	m.Styles = help.Styles{
		ShortKey:       keyStyle,
		ShortDesc:      descStyle,
		ShortSeparator: sepStyle,
		Ellipsis:       sepStyle,
		FullKey:        keyStyle,
		FullDesc:       descStyle,
		FullSeparator:  sepStyle,
	}

	return m
}

// newPlainHelpModel returns a help.Model with empty styles so View() produces
// plain text without ANSI escape codes — used when m.styled == false.
func newPlainHelpModel() help.Model {
	m := help.New()
	m.Styles = help.Styles{} // empty styles → no ANSI codes
	return m
}
