package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// Run launches the interactive TUI. Returns an error if the program exits
// abnormally.
func Run(_ context.Context) error {
	client, addr := NewClient()
	m := newModel(client, addr)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
