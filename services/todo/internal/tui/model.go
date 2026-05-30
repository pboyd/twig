package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	planv1connect "github.com/pboyd/todo/services/todo/gen/plan/v1/planv1connect"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli"
	"github.com/pboyd/todo/services/todo/internal/config"
)

type viewMode int

const (
	modeList viewMode = iota
	modeEdit
	modeNewSubtask
	modeNewRoot
	modeHelp
	modeMove
)

type tab int

const (
	tabTasks tab = iota
	tabPlanning
)

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
	day     string
	entries []*planv1.PlanEntry
	cursor  int
	loaded  bool
	mode    planMode
	picker  pickerState
	form    planFormState
	err     error
}

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	client          taskv1connect.TaskServiceClient
	planClient      planv1connect.PlanServiceClient
	addr            string
	pomConfig       config.PomodoroConfig
	activeTab       tab
	plan            planState
	tree            []*cli.TreeNode
	visible         []*visibleRow
	cursor          int
	expanded        map[int64]bool
	showCompleted   bool
	pendingComplete *int64
	mode            viewMode
	edit            editFormModel
	move            *moveState
	originalCursor  int
	help            help.Model
	keys            KeyMap
	err             error
	width           int
	height          int
	styled           bool
	hasDarkBackground bool
	pom             *activePom
	confirmingQuit  bool
}

func newModel(client taskv1connect.TaskServiceClient, planClient planv1connect.PlanServiceClient, addr string, pomConfig config.PomodoroConfig, hasDarkBg bool) Model {
	return Model{
		client:            client,
		planClient:        planClient,
		addr:              addr,
		pomConfig:         pomConfig,
		expanded:          make(map[int64]bool),
		keys:              DefaultKeyMap(),
		help:              newHelpModel(),
		styled:            cli.WantStyled(os.Stdout),
		hasDarkBackground: hasDarkBg,
		plan: planState{
			day: time.Now().Format("2006-01-02"),
		},
	}
}
