package tui

import (
	"context"
	"os/exec"
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	taskv1 "github.com/pboyd/twig/services/twig/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/services/twig/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/services/twig/internal/cli"
)

// ── message types ──────────────────────────────────────────────────────────

type pomTickMsg struct{}

type pomStartedMsg struct {
	taskID   int64
	taskName string
	startAt  time.Time
	err      error
}

type pomActiveMsg struct {
	pom *activePom
	err error
}

type pomCancelledMsg struct {
	err error
}

type pomCompletedMsg struct {
	taskName string
	err      error
}

type pomHookErrMsg struct {
	err error
}

type pomBannerExpireMsg struct{}

// ── tick ───────────────────────────────────────────────────────────────────

func pomTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return pomTickMsg{}
	})
}

// ── hook runner ────────────────────────────────────────────────────────────

// runPomHook runs sh -c <cmd> with stdio detached from the TUI terminal so it
// cannot corrupt the alt-screen. Returns nil for an empty command string.
func runPomHook(cmd, name string) tea.Cmd {
	if cmd == "" {
		return nil
	}
	return func() tea.Msg {
		c := exec.Command("sh", "-c", cmd)
		// Detach stdio: do NOT assign os.Stdin/Stdout/Stderr.
		if err := c.Run(); err != nil {
			return pomHookErrMsg{err: err}
		}
		return nil
	}
}

// ── helper ─────────────────────────────────────────────────────────────────

// activeTaskIDFromErr extracts the task id from an AlreadyExists error detail.
func activeTaskIDFromErr(ce *connect.Error) int64 {
	for _, d := range ce.Details() {
		v, err := d.Value()
		if err != nil {
			continue
		}
		if req, ok := v.(*taskv1.StartPomodoroRequest); ok {
			return req.TaskId
		}
	}
	return 0
}

// ── command factories ──────────────────────────────────────────────────────

func startPomCmd(client taskv1connect.TaskServiceClient, taskID int64, taskName string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.StartPomodoro(context.Background(), connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			ce, ok := err.(*connect.Error)
			if !ok || ce.Code() != connect.CodeAlreadyExists {
				return pomStartedMsg{err: err}
			}
			activeID := activeTaskIDFromErr(ce)
			if activeID == taskID {
				// Same task: attach to the running pomodoro.
				activeResp, aerr := client.GetActivePomodoro(context.Background(), connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
				if aerr != nil {
					return pomStartedMsg{err: aerr}
				}
				if activeResp.Msg.Pomodoro == nil {
					return pomStartedMsg{err: err}
				}
				return pomStartedMsg{
					taskID:   taskID,
					taskName: taskName,
					startAt:  activeResp.Msg.Pomodoro.StartAt.AsTime(),
				}
			}
			// Different task: cancel then start.
			if _, cerr := client.CancelPomodoro(context.Background(), connect.NewRequest(&taskv1.CancelPomodoroRequest{})); cerr != nil {
				return pomStartedMsg{err: cerr}
			}
			resp, err = client.StartPomodoro(context.Background(), connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
			if err != nil {
				return pomStartedMsg{err: err}
			}
			return pomStartedMsg{
				taskID:   taskID,
				taskName: taskName,
				startAt:  resp.Msg.Pomodoro.StartAt.AsTime(),
			}
		}
		return pomStartedMsg{
			taskID:   taskID,
			taskName: taskName,
			startAt:  resp.Msg.Pomodoro.StartAt.AsTime(),
		}
	}
}

func cancelPomCmd(client taskv1connect.TaskServiceClient) tea.Cmd {
	return func() tea.Msg {
		_, err := client.CancelPomodoro(context.Background(), connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		return pomCancelledMsg{err: err}
	}
}

func completePomCmd(client taskv1connect.TaskServiceClient, taskName string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.CompletePomodoro(context.Background(), connect.NewRequest(&taskv1.CompletePomodoroRequest{}))
		return pomCompletedMsg{taskName: taskName, err: err}
	}
}

func pomBannerExpireCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
		return pomBannerExpireMsg{}
	})
}

func getActivePomCmd(client taskv1connect.TaskServiceClient, tree []*cli.TreeNode) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetActivePomodoro(context.Background(), connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
		if err != nil {
			return pomActiveMsg{err: err}
		}
		if resp.Msg.Pomodoro == nil {
			return pomActiveMsg{}
		}
		p := resp.Msg.Pomodoro
		taskName := resolveTaskName(client, tree, p.TaskId)
		return pomActiveMsg{
			pom: &activePom{
				taskID:   p.TaskId,
				taskName: taskName,
				startAt:  p.StartAt.AsTime(),
			},
		}
	}
}

// resolveTaskName looks up the task name from the in-memory tree, falling back
// to a GetTask RPC if not found.
func resolveTaskName(client taskv1connect.TaskServiceClient, tree []*cli.TreeNode, taskID int64) string {
	name := findTaskName(tree, taskID)
	if name != "" {
		return name
	}
	if client == nil {
		return ""
	}
	resp, err := client.GetTask(context.Background(), connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
	if err != nil {
		return ""
	}
	return resp.Msg.Task.GetName()
}

func findTaskName(tree []*cli.TreeNode, taskID int64) string {
	for _, n := range tree {
		if n.Task.Id == taskID {
			return n.Task.Name
		}
		if name := findTaskName(n.Children, taskID); name != "" {
			return name
		}
	}
	return ""
}

// findTask looks up a task by ID in the in-memory tree. Returns nil when absent.
func findTask(tree []*cli.TreeNode, taskID int64) *taskv1.Task {
	for _, n := range tree {
		if n.Task.Id == taskID {
			return n.Task
		}
		if t := findTask(n.Children, taskID); t != nil {
			return t
		}
	}
	return nil
}
