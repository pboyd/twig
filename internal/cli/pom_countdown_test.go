package cli_test

import (
	"context"
	"testing"
	"time"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeActivePom(id, taskID int64, startAt time.Time) *taskv1.Pomodoro {
	return &taskv1.Pomodoro{
		Id:      id,
		TaskId:  taskID,
		StartAt: timestamppb.New(startAt),
	}
}

func makeTask(id int64, name string) *taskv1.Task {
	return &taskv1.Task{Id: id, Name: name}
}

// noopGetActivePom returns the same pom every time (no external cancel).
func noopGetActivePom(pom *taskv1.Pomodoro) func(ctx context.Context, pomID int64) (*taskv1.Pomodoro, bool) {
	return func(_ context.Context, pomID int64) (*taskv1.Pomodoro, bool) {
		return pom, true
	}
}

func TestCountdown_Completes(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-24 * time.Minute) // 1 minute remaining

	ticks := 0
	clock := now

	pom := makeActivePom(1, 42, startAt)
	deps := cli.ExportCountdownDeps{
		Now: func() time.Time {
			c := clock
			clock = clock.Add(time.Second)
			return c
		},
		Sleep:        func(d time.Duration) { ticks++ },
		ReadKey:      func() (byte, error) { return 0, nil },
		Render:       func(s cli.ExportCountdownState) {},
		GetActivePom: noopGetActivePom(pom),
		Ctx:          context.Background(),
	}

	outcome := cli.ExportRunCountdown(context.Background(), deps, pom, makeTask(42, "test task"), 0)

	if outcome != cli.ExportOutcomeCompleted {
		t.Errorf("expected Completed, got %v", outcome)
	}
	if ticks < 55 || ticks > 65 {
		t.Errorf("expected ~60 ticks, got %d", ticks)
	}
}

func TestCountdown_AlreadyExpired(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-30 * time.Minute) // already expired

	pom := makeActivePom(1, 42, startAt)
	deps := cli.ExportCountdownDeps{
		Now:          func() time.Time { return now },
		Sleep:        func(d time.Duration) {},
		ReadKey:      func() (byte, error) { return 0, nil },
		Render:       func(s cli.ExportCountdownState) {},
		GetActivePom: noopGetActivePom(pom),
		Ctx:          context.Background(),
	}

	outcome := cli.ExportRunCountdown(context.Background(), deps, pom, makeTask(42, "stale task"), 0)

	if outcome != cli.ExportOutcomeCompleted {
		t.Errorf("expected Completed for stale start, got %v", outcome)
	}
}

func TestCountdown_CancelKey(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-5 * time.Minute) // 20 minutes remaining

	keySent := false
	pom := makeActivePom(1, 42, startAt)
	deps := cli.ExportCountdownDeps{
		Now:   func() time.Time { return now },
		Sleep: func(d time.Duration) {},
		ReadKey: func() (byte, error) {
			if !keySent {
				keySent = true
				return 'c', nil
			}
			return 0, nil
		},
		Render:       func(s cli.ExportCountdownState) {},
		GetActivePom: noopGetActivePom(pom),
		Ctx:          context.Background(),
	}

	outcome := cli.ExportRunCountdown(context.Background(), deps, pom, makeTask(42, "cancel task"), 0)

	if outcome != cli.ExportOutcomeCanceled {
		t.Errorf("expected Canceled, got %v", outcome)
	}
}

func TestCountdown_QuitKey(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-5 * time.Minute)

	keySent := false
	pom := makeActivePom(1, 42, startAt)
	deps := cli.ExportCountdownDeps{
		Now:   func() time.Time { return now },
		Sleep: func(d time.Duration) {},
		ReadKey: func() (byte, error) {
			if !keySent {
				keySent = true
				return 'q', nil
			}
			return 0, nil
		},
		Render:       func(s cli.ExportCountdownState) {},
		GetActivePom: noopGetActivePom(pom),
		Ctx:          context.Background(),
	}

	outcome := cli.ExportRunCountdown(context.Background(), deps, pom, makeTask(42, "quit task"), 0)

	if outcome != cli.ExportOutcomeQuit {
		t.Errorf("expected Quit, got %v", outcome)
	}
}

func TestCountdown_ExternalCancel(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-5 * time.Minute)

	ticks := 0
	pom := makeActivePom(1, 42, startAt)
	deps := cli.ExportCountdownDeps{
		Now:     func() time.Time { return now },
		Sleep:   func(d time.Duration) { ticks++ },
		ReadKey: func() (byte, error) { return 0, nil },
		Render:  func(s cli.ExportCountdownState) {},
		GetActivePom: func(_ context.Context, pomID int64) (*taskv1.Pomodoro, bool) {
			if ticks >= 2 {
				return nil, true // nil = no active pomodoro
			}
			return pom, true
		},
		Ctx: context.Background(),
	}

	outcome := cli.ExportRunCountdown(context.Background(), deps, pom, makeTask(42, "external task"), 0)

	if outcome != cli.ExportOutcomeExternal {
		t.Errorf("expected External, got %v", outcome)
	}
}
