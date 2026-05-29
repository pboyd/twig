package tui

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pboyd/todo/services/todo/internal/config"
)

// Run launches the interactive TUI. Returns an error if the program exits
// abnormally.
func Run(_ context.Context) error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	cfg = cfg.Resolve()
	if cfg.APIKey == "" {
		fmt.Fprintf(os.Stderr, "error: API key not set; set TODO_API_KEY env var or api_key in %s\n", path)
		os.Exit(1)
	}

	taskClient, planClient, addr := NewClient(cfg)
	m := newModel(taskClient, planClient, addr, cfg.Pomodoro)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
