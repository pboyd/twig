package cli

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/term"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/pomodoro"
)

type countdownOutcome int

const (
	outcomeCompleted countdownOutcome = iota
	outcomeCanceled
	outcomeQuit
	outcomeExternal
)

type countdownState struct {
	task           *taskv1.Task
	completedCount int64
	remaining      time.Duration
	activePomID    int64
}

// getActivePomFn returns the active pomodoro for the given pomID owner, or nil if gone/changed.
// The bool return is true if the call succeeded (false means skip external-cancel check).
type getActivePomFn func(ctx context.Context, pomID int64) (*taskv1.Pomodoro, bool)

type countdownDeps struct {
	Now          func() time.Time
	Sleep        func(time.Duration)
	ReadKey      func() (byte, error)
	Render       func(countdownState)
	GetActivePom getActivePomFn
}

func realGetActivePom(client taskv1connect.TaskServiceClient, ctx context.Context) getActivePomFn {
	return func(_ context.Context, pomID int64) (*taskv1.Pomodoro, bool) {
		resp, err := client.GetActivePomodoro(ctx, connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
		if err != nil {
			return nil, false
		}
		return resp.Msg.Pomodoro, true
	}
}

func realCountdownDeps(client taskv1connect.TaskServiceClient, ctx context.Context) countdownDeps {
	return countdownDeps{
		Now:          time.Now,
		Sleep:        time.Sleep,
		GetActivePom: realGetActivePom(client, ctx),
		ReadKey: func() (byte, error) {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return 0, nil
			}
			buf := make([]byte, 1)
			n, _ := os.Stdin.Read(buf)
			if n > 0 {
				return buf[0], nil
			}
			return 0, nil
		},
		Render: func(s countdownState) {
			mins := int(s.remaining.Minutes())
			secs := int(s.remaining.Seconds()) % 60
			est := ""
			if s.task.Estimate > 0 {
				est = fmt.Sprintf(" (est %d, done %d)", s.task.Estimate, s.completedCount)
			}
			fmt.Printf("\r\033[K[%d:%02d] task %d %q%s  [c]ancel [q]uit",
				mins, secs, s.task.Id, s.task.Name, est)
		},
	}
}

func runCountdown(
	ctx context.Context,
	deps countdownDeps,
	activePom *taskv1.Pomodoro,
	task *taskv1.Task,
	completedCount int64,
) countdownOutcome {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		if oldState, err := term.MakeRaw(fd); err == nil {
			_ = syscall.SetNonblock(fd, true)
			defer func() {
				_ = syscall.SetNonblock(fd, false)
				_ = term.Restore(fd, oldState)
				fmt.Println()
			}()
		}
	}

	startAt := activePom.StartAt.AsTime()
	pomID := activePom.Id

	for {
		now := deps.Now()
		remaining := pomodoro.Remaining(startAt, now)

		deps.Render(countdownState{
			task:           task,
			completedCount: completedCount,
			remaining:      remaining,
			activePomID:    pomID,
		})

		if deps.GetActivePom != nil {
			active, ok := deps.GetActivePom(ctx, pomID)
			if ok && (active == nil || active.Id != pomID) {
				fmt.Fprintln(os.Stderr, "\npomodoro was stopped externally")
				return outcomeExternal
			}
		}

		if remaining == 0 {
			return outcomeCompleted
		}

		key, _ := deps.ReadKey()
		switch key {
		case 'c', 'C':
			return outcomeCanceled
		case 'q', 'Q':
			return outcomeQuit
		}

		deps.Sleep(time.Second)
	}
}
