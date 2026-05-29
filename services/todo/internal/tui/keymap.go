package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all keybindings for the TUI.
// PlanningMode controls which bindings appear in ShortHelp/FullHelp.
type KeyMap struct {
	PlanningMode bool
	Up           key.Binding
	Down         key.Binding
	Collapse     key.Binding
	Expand       key.Binding
	Edit         key.Binding
	NewSub       key.Binding
	NewRoot      key.Binding
	Delete       key.Binding
	Complete     key.Binding
	PomStart     key.Binding
	PomCancel    key.Binding
	Move         key.Binding
	Filter       key.Binding
	Refresh      key.Binding
	Help         key.Binding
	Quit         key.Binding
	// Tab navigation
	NextTab key.Binding
	PrevTab key.Binding
	// Planning actions
	PlanAddTask  key.Binding
	PlanAddEvent key.Binding
	PlanRename   key.Binding
	PlanMove     key.Binding
	PlanRemove   key.Binding
	PlanClear    key.Binding
	PlanPrevDay  key.Binding
	PlanNextDay  key.Binding
	PlanToday    key.Binding
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
		PomCancel: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "cancel pomodoro"),
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
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),
		PlanAddTask: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add task"),
		),
		PlanAddEvent: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "add event"),
		),
		PlanRename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename"),
		),
		PlanMove: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "move entry"),
		),
		PlanRemove: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "remove entry"),
		),
		PlanClear: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear from time"),
		),
		PlanPrevDay: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev day"),
		),
		PlanNextDay: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next day"),
		),
		PlanToday: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "today"),
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
	if k.PlanningMode {
		return []key.Binding{k.Up, k.Down, k.PlanAddTask, k.PlanAddEvent, k.Help, k.Quit}
	}
	return []key.Binding{k.Up, k.Down, k.Edit, k.Complete, k.Help, k.Quit}
}

// FullHelp returns the full help for the key map grouped by category.
func (k KeyMap) FullHelp() [][]key.Binding {
	if k.PlanningMode {
		return [][]key.Binding{
			{k.Up, k.Down, k.PlanPrevDay, k.PlanNextDay},
			{k.PlanAddTask, k.PlanAddEvent, k.PlanRename, k.PlanMove},
			{k.PlanRemove, k.PlanClear, k.PlanToday, k.Refresh},
			{k.PomCancel, k.NextTab, k.Help, k.Quit},
		}
	}
	return [][]key.Binding{
		{k.Up, k.Down, k.Collapse, k.Expand},
		{k.Edit, k.NewSub, k.NewRoot, k.Delete},
		{k.Complete, k.PomStart, k.PomCancel, k.Move},
		{k.Filter, k.Refresh, k.Help, k.Quit},
	}
}
