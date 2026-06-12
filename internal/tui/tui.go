package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
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
	hasDarkBg := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	initPalette(hasDarkBg)

	statePath, err := treeStatePath()
	if err != nil {
		statePath = "" // non-fatal; persistence disabled
	}
	pKey := profileKey(profile)
	tsf := loadTreeState(statePath)
	expanded := tsf.expandedSet(pKey)

	taskClient, planClient, goalClient, addr := NewClient(cfg)
	m := newModel(taskClient, planClient, addr, cfg.Pomodoro, hasDarkBg, expanded)
	m.goalClient = goalClient
	m.statePath = statePath
	m.activeProfile = pKey
	// colorprofile (charmbracelet/colorprofile) ignores COLORTERM when TERM
	// starts with "tmux", probing `tmux info` for RGB/Tc instead. When that
	// capability is absent it falls back to ANSI256, downsampling every
	// truecolor escape — turning the soft #1A2A3A cursor highlight into a
	// garish #00005F. Restore termenv's behavior: if COLORTERM says truecolor,
	// force the profile explicitly so bubbletea skips detection entirely.
	var opts []tea.ProgramOption
	if ct := strings.ToLower(os.Getenv("COLORTERM")); ct == "truecolor" || ct == "24bit" {
		opts = append(opts, tea.WithColorProfile(colorprofile.TrueColor))
	}
	p := tea.NewProgram(m, opts...)
	_, err = p.Run()
	return err
}
