package tui

import (
	"os"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	goalv1connect "github.com/pboyd/twig/api/gen/goal/v1/goalv1connect"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/config"
	"github.com/pboyd/twig/internal/report"
)

type viewMode int

const (
	modeList viewMode = iota
	modeEdit
	modeNewSubtask
	modeNewRoot
	modeHelp
	modeMove
	modeDatePrompt // Tasks-tab date picker for ctrl+p send-to-plan
)

type tab int

const (
	tabGoals tab = iota
	tabTasks
	tabPlanning
	tabReport
)

type goalViewMode int

const (
	goalList          goalViewMode = iota
	goalEdit
	goalNew
	goalConfirmDelete
	goalNewTask  // `a`: creating a new task to attach to the selected goal
	goalPickLink // `L`: task picker open — pick a task to link to the selected goal
	goalPickUnlink // `U`: picker open — pick an association root to unlink
)

type goalState struct {
	goals   []*goalv1.Goal
	cursor  int
	showAll bool
	loaded  bool
	err     error
	mode    goalViewMode
	picker  pickerState // task picker used for L (link) and U (unlink)
}

type planMode int

const (
	planList      planMode = iota // grid shown; selection nav + action keys
	planPickTask                  // task tree picker open (step 1 of add-task)
	planTaskTime                  // start/duration form (step 2 of add-task)
	planEventForm                 // name + start + duration form (add-event)
	planEdit                      // name + start + duration form (edit selected entry)
)

// activePom holds the TUI's in-memory view of the one running pomodoro.
type activePom struct {
	taskID    int64
	taskName  string
	startAt   time.Time
	completed bool
	banner    string
}

type pickerState struct {
	tree     []*cli.TreeNode
	visible  []*visibleRow
	cursor   int
	expanded map[int64]bool
}

type planFormState struct {
	fields  []textinput.Model
	focus   int
	taskID  int64
	entryID int32
}

type planState struct {
	day             string
	entries         []*planv1.PlanEntry
	cursor          int
	loaded          bool
	mode            planMode
	picker          pickerState
	form            planFormState
	err             error
	pendingComplete *int32 // entry id of a just-completed untimed entry to retain while highlighted
}

// reportState holds all state for the Report tab.
type reportState struct {
	presetIdx int             // index into report.PresetOrder()
	period    report.Period   // resolved period for presetIdx
	dayGroups []report.DayGroup
	finished  []report.AccomplishmentGroup
	ongoing   []report.AccomplishmentGroup
	totals    report.Totals
	loaded    bool
	err       error
	scroll    int // scroll offset in lines
}

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	client            taskv1connect.TaskServiceClient
	planClient        planv1connect.PlanServiceClient
	goalClient        goalv1connect.GoalServiceClient
	addr              string
	pomConfig         config.PomodoroConfig
	activeTab         tab
	plan              planState
	tree              []*cli.TreeNode
	visible           []*visibleRow
	cursor            int
	expanded          map[int64]bool
	statePath         string
	activeProfile     string
	showAll           bool
	pendingComplete   *int64
	mode              viewMode
	edit              editFormModel
	move              *moveState
	originalCursor    int
	help              help.Model // styled help (used when m.styled == true)
	plainHelp         help.Model // unstyled help (used when m.styled == false)
	keys              KeyMap
	err               error
	width             int
	height            int
	styled            bool
	hasDarkBackground bool
	pom               *activePom
	confirmingQuit    bool
	// date prompt state (modeDatePrompt): used when ctrl+p is pressed on Tasks tab
	datePromptInput    textinput.Model
	datePromptTaskID   int64
	datePromptTaskName string
	// scheduledDays maps task id → ascending YYYY-MM-DD days (today-or-future).
	// Populated by listScheduledDaysCmd; read synchronously by renderDetails.
	scheduledDays map[int64][]string
	// notice is a transient info message shown in the status bar (distinct from err).
	// It is cleared on the next user action.
	notice string
	// nowFunc, when non-nil, overrides time.Now().Local() for the auto-schedule floor.
	// Set only in tests via export_test.go shim.
	nowFunc func() time.Time
	// report holds all state for the Report tab.
	reportData reportState
	// goal holds all state for the Goals tab.
	goal goalState
}

func newModel(client taskv1connect.TaskServiceClient, planClient planv1connect.PlanServiceClient, addr string, pomConfig config.PomodoroConfig, hasDarkBg bool, expanded map[int64]bool) Model {
	if expanded == nil {
		expanded = make(map[int64]bool)
	}
	return Model{
		client:            client,
		planClient:        planClient,
		addr:              addr,
		pomConfig:         pomConfig,
		activeTab:         tabTasks,
		expanded:          expanded,
		keys:              DefaultKeyMap(),
		help:              newHelpModel(hasDarkBg),
		plainHelp:         newPlainHelpModel(),
		styled:            cli.WantStyled(os.Stdout),
		hasDarkBackground: hasDarkBg,
		scheduledDays:     make(map[int64][]string),
		plan: planState{
			day: time.Now().Format("2006-01-02"),
		},
	}
}
