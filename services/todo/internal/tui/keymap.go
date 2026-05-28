package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all keybindings for the TUI.
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Collapse  key.Binding
	Expand    key.Binding
	Edit      key.Binding
	NewSub    key.Binding
	NewRoot   key.Binding
	Delete    key.Binding
	Complete  key.Binding
	PomStart  key.Binding
	PomResume key.Binding
	Move      key.Binding
	Filter    key.Binding
	Refresh   key.Binding
	Help      key.Binding
	Quit      key.Binding
	// Edit-form keys
	Save     key.Binding
	Cancel   key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Collapse: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "collapse"),
		),
		Expand: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "expand"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit task"),
		),
		NewSub: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new subtask"),
		),
		NewRoot: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new root task"),
		),
		Delete: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "delete task"),
		),
		Complete: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle complete"),
		),
		PomStart: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start pomodoro"),
		),
		PomResume: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "resume pomodoro"),
		),
		Move: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "move (change parent)"),
		),
		Filter: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "toggle completed"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev field"),
		),
	}
}

// ShortHelp returns the short help for the key map (used by the bubbles help component).
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Edit, k.Complete, k.Help, k.Quit}
}

// FullHelp returns the full help for the key map grouped by category.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Collapse, k.Expand},
		{k.Edit, k.NewSub, k.NewRoot, k.Delete},
		{k.Complete, k.PomStart, k.PomResume, k.Move},
		{k.Filter, k.Refresh, k.Help, k.Quit},
	}
}
