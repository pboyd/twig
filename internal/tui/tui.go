package tui

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pboyd/twig/internal/config"
)

// Run launches the interactive TUI. Returns an error if the program exits
// abnormally.
func Run(_ context.Context, profile string) error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	selected, ok := cfg.Profile(profile)
	if !ok {
		return fmt.Errorf("twig: hmm, I couldn't find a profile named %q — check the spelling, or add a [profile.%s] section to %s", profile, profile, path)
	}
	cfg = selected.Resolve()
	if cfg.APIKey == "" {
		fmt.Fprintf(os.Stderr, "no API key found — set TWIG_API_KEY or add api_key to %s\n", path)
		os.Exit(1)
	}

	// Detect terminal background before entering raw mode — OSC 11 queries
	// don't work once bubbletea owns stdin.
	hasDarkBg := lipgloss.NewRenderer(os.Stdout).HasDarkBackground()

	statePath, err := treeStatePath()
	if err != nil {
		statePath = "" // non-fatal; persistence disabled
	}
	pKey := profileKey(profile)
	tsf := loadTreeState(statePath)
	expanded := tsf.expandedSet(pKey)

	taskClient, planClient, addr := NewClient(cfg)
	m := newModel(taskClient, planClient, addr, cfg.Pomodoro, hasDarkBg, expanded)
	m.statePath = statePath
	m.activeProfile = pKey
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
