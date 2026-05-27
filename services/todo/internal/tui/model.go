package tui

import (
	"github.com/charmbracelet/bubbles/help"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

type viewMode int

const (
	modeList viewMode = iota
	modeEdit
	modeNewSubtask
	modeNewRoot
	modeHelp
	modePomodoro
)

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	client          taskv1connect.TaskServiceClient
	addr            string
	tree            []*cli.TreeNode
	visible         []*visibleRow
	cursor          int
	expanded        map[int64]bool
	showCompleted   bool
	pendingComplete *int64
	mode            viewMode
	edit            editFormModel
	originalCursor  int
	help            help.Model
	keys            KeyMap
	err             error
	width           int
	height          int
}

func newModel(client taskv1connect.TaskServiceClient, addr string) Model {
	return Model{
		client:   client,
		addr:     addr,
		expanded: make(map[int64]bool),
		keys:     DefaultKeyMap(),
		help:     newHelpModel(),
	}
}
