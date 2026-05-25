package tui

import "github.com/charmbracelet/bubbles/help"

// newHelpModel returns a bubbles help.Model configured for full-screen display.
func newHelpModel() help.Model {
	m := help.New()
	m.ShowAll = true
	return m
}
