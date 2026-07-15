package tui

import "charm.land/bubbles/v2/key"

// KeyMap holds all keybindings for the TUI.
// PlanningMode, ReportMode, and GoalMode control which bindings appear in ShortHelp/FullHelp.
type KeyMap struct {
	GoalMode     bool
	PlanningMode bool
	ReportMode   bool
	Up           key.Binding
	Down         key.Binding
	First        key.Binding
	Last         key.Binding
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
	ToggleAll    key.Binding
	Refresh      key.Binding
	Help         key.Binding
	Quit         key.Binding
	// Tab navigation
	NextTab key.Binding
	PrevTab key.Binding
	// Planning actions
	PlanAddTask      key.Binding
	PlanAddEvent     key.Binding
	PlanEdit         key.Binding
	PlanRemove       key.Binding
	PlanGoToTask     key.Binding
	PlanPrevDay      key.Binding
	PlanNextDay      key.Binding
	PlanToday        key.Binding
	PlanAutoSchedule key.Binding
	PlanUnschedule   key.Binding
	// Rank ordering (Tasks tab)
	RankUp   key.Binding
	RankDown key.Binding
	// Send-to-plan shortcuts (Tasks tab)
	PlanSendToday   key.Binding
	PlanSendPickDay key.Binding
	// Edit-form keys
	Save     key.Binding
	Cancel   key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Editor   key.Binding
	Calendar key.Binding
	// Report-tab keys
	ReportPrevPreset key.Binding
	ReportNextPreset key.Binding
	// Page navigation (Tasks tab)
	PageUp   key.Binding
	PageDown key.Binding
	// Goals-tab keys
	GoalNew           key.Binding
	GoalEdit          key.Binding
	GoalDelete        key.Binding
	GoalToggleAll     key.Binding
	GoalRankUp        key.Binding
	GoalRankDown      key.Binding
	GoalAddTask       key.Binding
	GoalLinkTask      key.Binding
	GoalUnlinkTask    key.Binding
	GoalStatusHistory key.Binding // `s`: open status history
	GoalAddStatus     key.Binding // `S`: quick-add status update
	GoalStatusEdit    key.Binding // `e`: edit selected update (history view)
	GoalStatusDelete  key.Binding // `d`: delete selected update (history view)
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
		First: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "first"),
		),
		Last: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "last"),
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
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit task"),
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
			key.WithKeys("space"),
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
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "toggle show all"),
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
			key.WithKeys("t"),
			key.WithHelp("t", "add task"),
		),
		PlanAddEvent: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "add event"),
		),
		PlanEdit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit entry"),
		),
		PlanRemove: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "remove entry"),
		),
		PlanGoToTask: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "go to task"),
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
			key.WithKeys("."),
			key.WithHelp(".", "today"),
		),
		PlanAutoSchedule: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "auto-schedule"),
		),
		PlanUnschedule: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "unschedule"),
		),
		RankUp: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "rank higher"),
		),
		RankDown: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "rank lower"),
		),
		PlanSendToday: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "send to today's plan"),
		),
		PlanSendPickDay: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "send to a day"),
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
		Editor: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "edit in $EDITOR"),
		),
		Calendar: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "pick a date"),
		),
		ReportPrevPreset: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev period"),
		),
		ReportNextPreset: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next period"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
		GoalNew: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new goal"),
		),
		GoalEdit: key.NewBinding(
			key.WithKeys("e", "enter"),
			key.WithHelp("e/enter", "edit goal"),
		),
		GoalDelete: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "delete goal"),
		),
		GoalToggleAll: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "toggle show all"),
		),
		GoalRankUp: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "rank higher"),
		),
		GoalRankDown: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "rank lower"),
		),
		GoalAddTask: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "add task"),
		),
		GoalLinkTask: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "link task"),
		),
		GoalUnlinkTask: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "unlink task"),
		),
		GoalStatusHistory: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "status history"),
		),
		GoalAddStatus: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "add status"),
		),
		GoalStatusEdit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit update"),
		),
		GoalStatusDelete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete update"),
		),
	}
}

// ShortHelp returns the short help for the key map (used by the bubbles help component).
func (k KeyMap) ShortHelp() []key.Binding {
	if k.GoalMode {
		return []key.Binding{k.Up, k.Down, k.GoalNew, k.GoalAddTask, k.GoalEdit, k.Complete, k.GoalStatusHistory, k.GoalAddStatus, k.Help, k.Quit}
	}
	if k.ReportMode {
		return []key.Binding{k.ReportPrevPreset, k.ReportNextPreset, k.Up, k.Down, k.Refresh, k.Help, k.Quit}
	}
	if k.PlanningMode {
		return []key.Binding{k.Up, k.Down, k.Complete, k.PomStart, k.PlanAddTask, k.PlanAutoSchedule, k.PlanUnschedule, k.Help, k.Quit}
	}
	return []key.Binding{k.Up, k.Down, k.Edit, k.Complete, k.Help, k.Quit}
}

// FullHelp returns the full help for the key map grouped by category.
func (k KeyMap) FullHelp() [][]key.Binding {
	if k.GoalMode {
		return [][]key.Binding{
			{k.Up, k.Down, k.GoalNew, k.GoalEdit, k.Complete},
			{k.GoalDelete, k.GoalRankUp, k.GoalRankDown},
			{k.GoalAddTask, k.GoalLinkTask, k.GoalUnlinkTask, k.GoalToggleAll},
			{k.GoalStatusHistory, k.GoalAddStatus, k.GoalStatusEdit, k.GoalStatusDelete},
			{k.Help, k.Quit},
		}
	}
	if k.ReportMode {
		return [][]key.Binding{
			{k.ReportPrevPreset, k.ReportNextPreset},
			{k.Up, k.Down, k.Refresh},
			{k.NextTab, k.Help, k.Quit},
		}
	}
	if k.PlanningMode {
		return [][]key.Binding{
			{k.Up, k.Down, k.PlanPrevDay, k.PlanNextDay},
			{k.PlanAddTask, k.PlanAddEvent, k.PlanEdit, k.PlanRemove, k.PlanGoToTask},
			{k.PlanAutoSchedule, k.PlanUnschedule, k.Complete, k.PomStart, k.PomCancel, k.NextTab},
			{k.PlanToday, k.Refresh, k.Help, k.Quit},
		}
	}
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Collapse, k.Expand},
		{k.Edit, k.NewSub, k.NewRoot, k.Delete},
		{k.Complete, k.PomStart, k.PomCancel, k.Move},
		{k.RankUp, k.RankDown, k.PlanSendToday, k.PlanSendPickDay},
		{k.Filter, k.Refresh, k.Help, k.Quit},
	}
}
