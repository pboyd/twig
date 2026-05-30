package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"connectrpc.com/connect"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/config"
	"github.com/pboyd/todo/services/todo/internal/pomodoro"
)

func runPomTop(args []string) int {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if cfg.APIKey == "" {
		path, _ := config.DefaultPath()
		fmt.Fprintf(os.Stderr, "no API key found — set TWIG_API_KEY or add api_key to %s\n", path)
		return 1
	}
	client := newTaskClient(cfg.APIURL, cfg.APIKey)
	return runPom(client, cfg.Pomodoro, args)
}

func newTaskClient(addr, apiKey string) taskv1connect.TaskServiceClient {
	return taskv1connect.NewTaskServiceClient(
		&http.Client{},
		addr,
		connect.WithSendGzip(),
		connect.WithInterceptors(BearerInterceptor(apiKey)),
	)
}

func runPom(client taskv1connect.TaskServiceClient, hooks config.PomodoroConfig, args []string) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		printPomUsage(os.Stdout)
		return 0
	}
	if len(args) == 0 {
		printPomUsage(os.Stderr)
		return 1
	}
	switch args[0] {
	case "estimate":
		return runEstimate(client, args[1:])
	case "start":
		return runStart(client, hooks, args[1:])
	case "resume":
		return runResume(client, hooks, nil)
	case "cancel":
		return runCancel(client, hooks, args[1:])
	case "status":
		return runStatus(client, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown pom subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'twig help pom' for usage.")
		return 1
	}
}

func printPomUsage(w io.Writer) {
	path, _ := config.DefaultPath()
	fmt.Fprintln(w, "Usage: twig pom <subcommand> [arguments]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  estimate <task_id> <n>   Set estimated pomodoros (0–10)")
	fmt.Fprintln(w, "  start <task_id>          Start a 25-minute pomodoro")
	fmt.Fprintln(w, "  resume                   Re-attach to the active pomodoro")
	fmt.Fprintln(w, "  cancel                   Cancel the active pomodoro")
	fmt.Fprintln(w, "  status                   Show active pomodoro status")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Lifecycle hooks: configured via [pomodoro].on_start / on_cancel / on_complete in %s\n", path)
}

func runEstimate(client taskv1connect.TaskServiceClient, args []string) int {
	if len(args) < 2 {
		printPomUsage(os.Stderr)
		return 1
	}
	taskID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid task_id: %v\n", args[0])
		printPomUsage(os.Stderr)
		return 1
	}
	n, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid estimate: %v\n", args[1])
		printPomUsage(os.Stderr)
		return 1
	}

	resp, err := client.SetEstimate(context.Background(), connect.NewRequest(&taskv1.SetEstimateRequest{
		TaskId:   taskID,
		Estimate: int32(n),
	}))
	if err != nil {
		ce, ok := err.(*connect.Error)
		if ok && ce.Code() == connect.CodeNotFound {
			fmt.Fprintln(os.Stderr, "task not found")
		} else if ok && ce.Code() == connect.CodeInvalidArgument {
			fmt.Fprintln(os.Stderr, ce.Message())
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		return 1
	}
	fmt.Printf("set estimate for task %d to %d\n", resp.Msg.Task.Id, resp.Msg.Task.Estimate)
	return 0
}

// RunPomodoroForTask starts a pomodoro for the given task and blocks until the
// countdown finishes. If another pomodoro is already active on a different task,
// it is cancelled first; if the same task is already active, it is resumed.
// Intended for TUI use where interactive prompts are not available.
func RunPomodoroForTask(ctx context.Context, client taskv1connect.TaskServiceClient, taskID int64, hooks config.PomodoroConfig) error {
	taskResp, err := client.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
	if err != nil {
		return err
	}
	task := taskResp.Msg.Task
	completedCount := taskResp.Msg.CompletedPomodoroCount

	startResp, err := client.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
	if err != nil {
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeAlreadyExists {
			return err
		}
		activeTaskID := extractActiveTaskID(ce)
		if activeTaskID != taskID {
			if _, cerr := client.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{})); cerr != nil {
				return cerr
			}
			if startResp, err = client.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID})); err != nil {
				return err
			}
		} else {
			return ResumeBackgroundedPomodoro(ctx, client, hooks)
		}
	}

	fireHook(hooks.OnStart, "on_start")
	if code := runCountdownAndComplete(client, ctx, startResp.Msg.Pomodoro, task, completedCount, hooks); code != 0 {
		return fmt.Errorf("pomodoro failed")
	}
	return nil
}

// ResumeBackgroundedPomodoro re-attaches to an active pomodoro and blocks until done.
func ResumeBackgroundedPomodoro(ctx context.Context, client taskv1connect.TaskServiceClient, hooks config.PomodoroConfig) error {
	if code := runResume(client, hooks, nil); code != 0 {
		return fmt.Errorf("resume pomodoro failed")
	}
	return nil
}

func runStart(client taskv1connect.TaskServiceClient, hooks config.PomodoroConfig, args []string) int {
	taskID, ok := parseStartArgs(args)
	if !ok {
		printPomUsage(os.Stderr)
		return 1
	}

	ctx := context.Background()

	taskResp, err := client.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	task := taskResp.Msg.Task
	completedCount := taskResp.Msg.CompletedPomodoroCount

	startResp, err := client.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
	if err != nil {
		ce, ok := err.(*connect.Error)
		if ok && ce.Code() == connect.CodeAlreadyExists {
			activeTaskID := extractActiveTaskID(ce)
			if activeTaskID == taskID {
				// Same task: prompt restart or resume.
				choice := promptRestartResume(taskID)
				switch choice {
				case "restart":
					_, cerr := client.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
					if cerr != nil {
						fmt.Fprintf(os.Stderr, "error canceling: %v\n", cerr)
						return 1
					}
					startResp2, serr := client.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
					if serr != nil {
						fmt.Fprintf(os.Stderr, "error: %v\n", serr)
						return 1
					}
					fireHook(hooks.OnStart, "on_start")
					return runCountdownAndComplete(client, ctx, startResp2.Msg.Pomodoro, task, completedCount, hooks)
				case "resume":
					return runResume(client, hooks, nil)
				default:
					return 1
				}
			} else {
				// Different task: prompt cancel-and-start or abort.
				choice := promptCancelAndStart(activeTaskID)
				switch choice {
				case "cancel":
					_, cerr := client.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
					if cerr != nil {
						fmt.Fprintf(os.Stderr, "error canceling: %v\n", cerr)
						return 1
					}
					startResp2, serr := client.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
					if serr != nil {
						fmt.Fprintf(os.Stderr, "error: %v\n", serr)
						return 1
					}
					fireHook(hooks.OnStart, "on_start")
					return runCountdownAndComplete(client, ctx, startResp2.Msg.Pomodoro, task, completedCount, hooks)
				default:
					fmt.Fprintf(os.Stderr, "you have an active pomodoro on task %d\n", activeTaskID)
					return 1
				}
			}
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	fireHook(hooks.OnStart, "on_start")
	return runCountdownAndComplete(client, ctx, startResp.Msg.Pomodoro, task, completedCount, hooks)
}

func runCountdownAndComplete(
	client taskv1connect.TaskServiceClient,
	ctx context.Context,
	activePom *taskv1.Pomodoro,
	task *taskv1.Task,
	completedCount int64,
	hooks config.PomodoroConfig,
) int {
	deps := realCountdownDeps(client, ctx)
	outcome := runCountdown(ctx, deps, activePom, task, completedCount)

	switch outcome {
	case outcomeCompleted:
		_, err := client.CompletePomodoro(ctx, connect.NewRequest(&taskv1.CompletePomodoroRequest{}))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error completing pomodoro: %v\n", err)
			return 1
		}
		fireHook(hooks.OnComplete, "on_complete")
	case outcomeCanceled:
		_, err := client.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error canceling pomodoro: %v\n", err)
			return 1
		}
		fireHook(hooks.OnCancel, "on_cancel")
	case outcomeQuit:
		// No API call — timer keeps running on the server. No hook fires.
	case outcomeExternal:
		// Already handled by countdown renderer.
	}
	return 0
}

func runResume(client taskv1connect.TaskServiceClient, hooks config.PomodoroConfig, args []string) int {
	ctx := context.Background()
	return runResumeWith(
		func(_ context.Context) (*taskv1.Pomodoro, error) {
			resp, err := client.GetActivePomodoro(ctx, connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
			if err != nil {
				return nil, err
			}
			return resp.Msg.Pomodoro, nil
		},
		func(_ context.Context) error {
			_, err := client.CompletePomodoro(ctx, connect.NewRequest(&taskv1.CompletePomodoroRequest{}))
			return err
		},
		func(_ context.Context, id int64) (*taskv1.GetTaskResponse, error) {
			resp, err := client.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: id}))
			if err != nil {
				return nil, err
			}
			return resp.Msg, nil
		},
		func(pom *taskv1.Pomodoro, task *taskv1.Task, completedCount int64) int {
			return runCountdownAndComplete(client, ctx, pom, task, completedCount, hooks)
		},
		time.Now(),
		hooks,
	)
}

func runResumeWith(
	getActiveFn func(ctx context.Context) (*taskv1.Pomodoro, error),
	completeFn func(ctx context.Context) error,
	getTaskFn func(ctx context.Context, id int64) (*taskv1.GetTaskResponse, error),
	countdownFn func(pom *taskv1.Pomodoro, task *taskv1.Task, completedCount int64) int,
	now time.Time,
	hooks config.PomodoroConfig,
) int {
	ctx := context.Background()
	activePom, err := getActiveFn(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if activePom == nil {
		fmt.Fprintln(os.Stderr, "no active pomodoro to resume")
		return 1
	}
	startAt := activePom.StartAt.AsTime()

	if pomodoro.Remaining(startAt, now) == 0 {
		if err := completeFn(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "error completing pomodoro: %v\n", err)
			return 1
		}
		fireHook(hooks.OnComplete, "on_complete")
		return 0
	}

	if getTaskFn == nil || countdownFn == nil {
		return 0
	}
	taskResp, err := getTaskFn(ctx, activePom.TaskId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return countdownFn(activePom, taskResp.Task, taskResp.CompletedPomodoroCount)
}

func runCancel(client taskv1connect.TaskServiceClient, hooks config.PomodoroConfig, args []string) int {
	return runCancelWith(func(ctx context.Context) error {
		_, err := client.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		return err
	}, hooks)
}

func runCancelWith(cancelFn func(ctx context.Context) error, hooks config.PomodoroConfig) int {
	err := cancelFn(context.Background())
	if err != nil {
		ce, ok := err.(*connect.Error)
		if ok && ce.Code() == connect.CodeFailedPrecondition {
			fmt.Fprintln(os.Stderr, "no active pomodoro")
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		return 1
	}
	fmt.Println("canceled active pomodoro")
	fireHook(hooks.OnCancel, "on_cancel")
	return 0
}

func runStatus(client taskv1connect.TaskServiceClient, args []string) int {
	return runStatusWith(
		func(ctx context.Context) (*taskv1.Pomodoro, error) {
			resp, err := client.GetActivePomodoro(ctx, connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
			if err != nil {
				return nil, err
			}
			return resp.Msg.Pomodoro, nil
		},
		func(ctx context.Context, id int64) (*taskv1.Task, error) {
			resp, err := client.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: id}))
			if err != nil {
				return nil, err
			}
			return resp.Msg.Task, nil
		},
		time.Now(),
		os.Stdout,
	)
}

func runStatusWith(
	getActiveFn func(ctx context.Context) (*taskv1.Pomodoro, error),
	getTaskFn func(ctx context.Context, id int64) (*taskv1.Task, error),
	now time.Time,
	w io.Writer,
) int {
	ctx := context.Background()
	p, err := getActiveFn(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if p == nil {
		fmt.Fprintln(w, "No active pomodoro.")
		return 0
	}
	startAt := p.StartAt.AsTime()
	remaining := pomodoro.Remaining(startAt, now)
	mins := int(remaining.Minutes())
	secs := int(remaining.Seconds()) % 60

	task, err := getTaskFn(ctx, p.TaskId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(w, "Active pomodoro: task %d %q\n", task.Id, task.Name)
	fmt.Fprintf(w, "started: %s\n", startAt.Format(time.RFC3339))
	fmt.Fprintf(w, "remaining: %d:%02d\n", mins, secs)
	return 0
}

func execHook(cmd string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// fireHook runs cmd if non-empty. On failure it prints a warning and continues.
func fireHook(cmd, name string) {
	if cmd == "" {
		return
	}
	if err := execHook(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s hook exited with status %v: %s\n", name, err, cmd)
	}
}

// parseStartArgs parses `pom start <task_id>`. Any extra arguments are rejected.
func parseStartArgs(args []string) (taskID int64, ok bool) {
	if len(args) == 0 {
		return 0, false
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid task_id: %v\n", args[0])
		return 0, false
	}
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[1])
		return 0, false
	}
	return id, true
}

func extractActiveTaskID(ce *connect.Error) int64 {
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

func promptRestartResume(taskID int64) string {
	fmt.Fprintf(os.Stderr, "you have an active pomodoro on task %d. [r]estart or [s]resume? ", taskID)
	var input string
	fmt.Fscan(os.Stdin, &input)
	switch input {
	case "r", "restart":
		return "restart"
	case "s", "resume":
		return "resume"
	}
	return ""
}

func promptCancelAndStart(activeTaskID int64) string {
	fmt.Fprintf(os.Stderr, "you have an active pomodoro on task %d. [c]ancel it and start, or [a]bort? ", activeTaskID)
	var input string
	fmt.Fscan(os.Stdin, &input)
	switch input {
	case "c", "cancel":
		return "cancel"
	}
	return ""
}
