package tui

import (
	"os"

	"github.com/charmbracelet/bubbles/help"
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
	modePomodoro
	modeMove
)

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	client          taskv1connect.TaskServiceClient
	addr            string
	pomConfig       config.PomodoroConfig
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
	styled          bool
}

func newModel(client taskv1connect.TaskServiceClient, addr string, pomConfig config.PomodoroConfig) Model {
	return Model{
		client:    client,
		addr:      addr,
		pomConfig: pomConfig,
		expanded:  make(map[int64]bool),
		keys:      DefaultKeyMap(),
		help:      newHelpModel(),
		styled:    cli.WantStyled(os.Stdout),
	}
}
