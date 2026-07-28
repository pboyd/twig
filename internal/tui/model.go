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
	"github.com/pboyd/twig/internal/markdown"
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
	modeFilter     // Tasks-tab filter bar
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
	goalList goalViewMode = iota
	goalEdit
	goalNew
	goalConfirmDelete
	goalNewTask          // `a`: creating a new task to attach to the selected goal
	goalPickLink         // `L`: task picker open — pick a task to link to the selected goal
	goalPickUnlink       // `U`: picker open — pick an association root to unlink
	goalStatusHistory    // `s`: browsing the status-update history list
	goalStatusReader     // enter (from history): full-screen reader for a single update
	goalStatusConfirmDel // `d` (in history): confirm before deleting a status update
)

// statusCompose tracks an in-progress status-update compose/edit started from the editor.
type statusCompose struct {
	goalID    int64
	editingID int64 // 0 = add new; >0 = edit existing
	active    bool
}

type goalState struct {
	goals    []*goalv1.Goal
	cursor   int
	showAll  bool
	loaded   bool
	err      error
	mode     goalViewMode
	picker   pickerState // task picker used for L (link) and U (unlink)
	lastLoad time.Time   // when the Goals tab's data was last loaded or a refresh was dispatched
	// Status-update compose state (US1/US3).
	compose statusCompose
	// Status-update history state (US2/US3).
	statusUpdates []*goalv1.StatusUpdate
	statusCursor  int
	readerOffset  int
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
	fields     []textinput.Model
	focus      int
	taskID     int64
	entryID    int32
	origFields []string // snapshot of field values when form was opened
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
	pendingComplete *int32    // entry id of a just-completed untimed entry to retain while highlighted
	lastLoad        time.Time // when the Plan tab's data was last loaded or a refresh was dispatched
}

// reportState holds all state for the Report tab.
type reportState struct {
	presetIdx int           // index into report.PresetOrder()
	period    report.Period // resolved period for presetIdx
	dayGroups []report.DayGroup
	finished  []report.AccomplishmentGroup
	ongoing   []report.AccomplishmentGroup
	totals    report.Totals
	loaded    bool
	err       error
	scroll    int       // scroll offset in lines
	lastLoad  time.Time // when the Report tab's data was last loaded or a refresh was dispatched
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
	listScroll        int // top visible row index of the Tasks list viewport
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
	confirmingDiscard bool
	// date prompt state (modeDatePrompt): used when ctrl+p is pressed on Tasks tab
	datePromptInput    textinput.Model
	datePromptCalendar *calendarModel // nil when closed
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
	// tasksLastLoad records when the Tasks tab's data was last loaded or a refresh was dispatched.
	tasksLastLoad time.Time
	// md renders Markdown to terminal-styled text.
	md *markdown.Renderer
	// Filter state (Task Filters feature).
	filterInput   textinput.Model // filter bar text input
	filterExpr    string          // the last-submitted filter expression
	filterMatches []int64         // task IDs matching the active filter (nil = no filter)
	filterInvalid bool            // true when the current expression is invalid
	filterGen     int             // generation counter to discard stale FilterTasks responses
	filteredIDs   map[int64]bool  // set of task IDs in filterMatches for O(1) lookup
	filterFocus   filterFocus     // where to put the cursor when the pending result lands
	// planHooks holds the in-memory state for the plan entry boundary hooks
	// (see internal/tui/plan_hook.go). cfg is seeded from config; the
	// ticker is gated on cfg.enabled(). watermark is seeded to time.Now()
	// at init so nothing retroactive fires (FR-006).
	planHooks planHookState
}

func newModel(client taskv1connect.TaskServiceClient, planClient planv1connect.PlanServiceClient, addr string, pomConfig config.PomodoroConfig, planConfig config.PlanConfig, hasDarkBg bool, expanded map[int64]bool) Model {
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
		planHooks: planHookState{
			cfg:       planConfig,
			watermark: time.Now(),
		},
		md:          markdown.NewRenderer(buildMarkdownTheme()),
		filterInput: newPlanInput("search, or try completed=false AND ^goal_id=1"),
	}
}
