package tui

import (
	"context"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// pomodoroRequestMsg is dispatched by startPomodoroCmd / resumePomodoroCmd so
// the Update handler can switch mode before launching the blocking exec.
type pomodoroRequestMsg struct {
	taskID int64
	resume bool
}

// pomodoroDoneMsg is sent when the tea.Exec pomodoro command finishes.
type pomodoroDoneMsg struct {
	err error
}

// funcExecCommand implements tea.ExecCommand for a plain Go function.
// SetStdin/SetStdout/SetStderr are no-ops because the countdown code uses
// os.Stdin/os.Stdout directly (same fds that bubbletea would pass).
type funcExecCommand struct {
	fn func() error
}

func (c *funcExecCommand) Run() error            { return c.fn() }
func (c *funcExecCommand) SetStdin(_ io.Reader)  {}
func (c *funcExecCommand) SetStdout(_ io.Writer) {}
func (c *funcExecCommand) SetStderr(_ io.Writer) {}

// startPomodoroCmd returns a Cmd that emits pomodoroRequestMsg for the given task.
// The Update handler processes pomodoroRequestMsg and dispatches execPomodoroStart.
func startPomodoroCmd(taskID int64) tea.Cmd {
	return func() tea.Msg {
		return pomodoroRequestMsg{taskID: taskID}
	}
}

// resumePomodoroCmd returns a Cmd that emits pomodoroRequestMsg requesting a resume.
func resumePomodoroCmd() tea.Cmd {
	return func() tea.Msg {
		return pomodoroRequestMsg{resume: true}
	}
}

// execPomodoroStart yields the terminal to RunPomodoroForTask and dispatches
// pomodoroDoneMsg when the countdown finishes.
func execPomodoroStart(client taskv1connect.TaskServiceClient, taskID int64) tea.Cmd {
	return tea.Exec(&funcExecCommand{fn: func() error {
		return cli.RunPomodoroForTask(context.Background(), client, taskID)
	}}, func(err error) tea.Msg {
		return pomodoroDoneMsg{err: err}
	})
}

// execPomodoroResume yields the terminal to ResumeBackgroundedPomodoro and
// dispatches pomodoroDoneMsg when the countdown finishes.
func execPomodoroResume(client taskv1connect.TaskServiceClient) tea.Cmd {
	return tea.Exec(&funcExecCommand{fn: func() error {
		return cli.ResumeBackgroundedPomodoro(context.Background(), client)
	}}, func(err error) tea.Msg {
		return pomodoroDoneMsg{err: err}
	})
}
