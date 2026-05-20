package cli

import (
	"context"
	"io"
	"time"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
)

// ---- countdown exports ----

type ExportCountdownState = countdownState

type ExportCountdownDeps struct {
	Now          func() time.Time
	Sleep        func(time.Duration)
	ReadKey      func() (byte, error)
	Render       func(ExportCountdownState)
	GetActivePom func(ctx context.Context, pomID int64) (*taskv1.Pomodoro, bool)
	Ctx          context.Context
}

const (
	ExportOutcomeCompleted = outcomeCompleted
	ExportOutcomeCanceled  = outcomeCanceled
	ExportOutcomeQuit      = outcomeQuit
	ExportOutcomeExternal  = outcomeExternal
)

func ExportRunCountdown(
	ctx context.Context,
	deps ExportCountdownDeps,
	activePom *taskv1.Pomodoro,
	task *taskv1.Task,
	completedCount int64,
) countdownOutcome {
	inner := countdownDeps{
		Now:          deps.Now,
		Sleep:        deps.Sleep,
		ReadKey:      deps.ReadKey,
		Render:       deps.Render,
		GetActivePom: deps.GetActivePom,
	}
	return runCountdown(ctx, inner, activePom, task, completedCount)
}

// ---- pom.go exports ----

func ExportExecHook(cmd string) error {
	return execHook(cmd)
}

func ExportRunEstimateArgs(client taskv1connect.TaskServiceClient, args []string) int {
	return runEstimate(client, args)
}

func ExportRunStatusWith(
	getActiveFn func(ctx context.Context) (*taskv1.Pomodoro, error),
	getTaskFn func(ctx context.Context, id int64) (*taskv1.Task, error),
	now time.Time,
	w io.Writer,
) int {
	return runStatusWith(getActiveFn, getTaskFn, now, w)
}

func ExportRunCancelWith(cancelFn func(ctx context.Context) error) int {
	return runCancelWith(cancelFn)
}

func ExportRunResumeWith(
	getActiveFn func(ctx context.Context) (*taskv1.Pomodoro, error),
	completeFn func(ctx context.Context) error,
	getTaskFn func(ctx context.Context, id int64) (*taskv1.GetTaskResponse, error),
	countdownFn func(pom *taskv1.Pomodoro, task *taskv1.Task, completedCount int64) int,
	now time.Time,
	execCmd string,
) int {
	return runResumeWith(getActiveFn, completeFn, getTaskFn, countdownFn, now, execCmd)
}
