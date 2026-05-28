package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExecHook(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := cli.ExportExecHook("true")
		if err != nil {
			t.Errorf("expected nil error for 'true', got %v", err)
		}
	})

	t.Run("failure returns non-nil error", func(t *testing.T) {
		err := cli.ExportExecHook("false")
		if err == nil {
			t.Error("expected non-nil error for 'false'")
		}
	})
}

func TestRunEstimate_ArgParsing(t *testing.T) {
	code := cli.ExportRunEstimateArgs(nil, []string{})
	if code == 0 {
		t.Error("expected non-zero exit for missing args")
	}

	code = cli.ExportRunEstimateArgs(nil, []string{"notanumber", "3"})
	if code == 0 {
		t.Error("expected non-zero exit for non-integer task_id")
	}

	code = cli.ExportRunEstimateArgs(nil, []string{"1", "notanumber"})
	if code == 0 {
		t.Error("expected non-zero exit for non-integer estimate")
	}
}

func TestRunStatus_NoActive(t *testing.T) {
	var buf bytes.Buffer
	code := cli.ExportRunStatusWith(
		func(_ context.Context) (*taskv1.Pomodoro, error) { return nil, nil },
		nil,
		time.Now(),
		&buf,
	)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(buf.String(), "No active pomodoro") {
		t.Errorf("expected 'No active pomodoro', got %q", buf.String())
	}
}

func TestRunStatus_Active(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-5 * time.Minute)
	pom := &taskv1.Pomodoro{
		Id:      1,
		TaskId:  42,
		StartAt: timestamppb.New(startAt),
	}
	task := &taskv1.Task{Id: 42, Name: "my task"}

	var buf bytes.Buffer
	code := cli.ExportRunStatusWith(
		func(_ context.Context) (*taskv1.Pomodoro, error) { return pom, nil },
		func(_ context.Context, id int64) (*taskv1.Task, error) { return task, nil },
		now,
		&buf,
	)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "42") {
		t.Errorf("expected task id 42 in output, got %q", out)
	}
	if !strings.Contains(out, "my task") {
		t.Errorf("expected task name in output, got %q", out)
	}
	if !strings.Contains(out, "20:00") {
		t.Errorf("expected 20:00 remaining, got %q", out)
	}
}

func TestRunStatus_StaleActive(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-30 * time.Minute)
	pom := &taskv1.Pomodoro{
		Id:      1,
		TaskId:  42,
		StartAt: timestamppb.New(startAt),
	}
	task := &taskv1.Task{Id: 42, Name: "stale task"}

	var buf bytes.Buffer
	code := cli.ExportRunStatusWith(
		func(_ context.Context) (*taskv1.Pomodoro, error) { return pom, nil },
		func(_ context.Context, id int64) (*taskv1.Task, error) { return task, nil },
		now,
		&buf,
	)
	if code != 0 {
		t.Errorf("expected exit 0 for stale active, got %d", code)
	}
	if !strings.Contains(buf.String(), "0:00") {
		t.Errorf("expected 0:00 remaining for stale, got %q", buf.String())
	}
}

func TestRunCancel_NoActive(t *testing.T) {
	code := cli.ExportRunCancelWith(func(_ context.Context) error {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("There's no pomodoro ticking. Start one first."))
	})
	if code == 0 {
		t.Error("expected non-zero exit for no active pomodoro")
	}
}

func TestRunCancel_HappyPath(t *testing.T) {
	code := cli.ExportRunCancelWith(func(_ context.Context) error { return nil })
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRunResume_NoActive(t *testing.T) {
	code := cli.ExportRunResumeWith(
		func(_ context.Context) (*taskv1.Pomodoro, error) { return nil, nil },
		nil, nil, nil, time.Now(), cli.ExportPomodoroConfig{},
	)
	if code == 0 {
		t.Error("expected non-zero exit when no active pomodoro")
	}
}

func TestRunResume_StaleActive(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-30 * time.Minute)
	pom := &taskv1.Pomodoro{
		Id:      1,
		TaskId:  42,
		StartAt: timestamppb.New(startAt),
	}

	completeCalled := false
	code := cli.ExportRunResumeWith(
		func(_ context.Context) (*taskv1.Pomodoro, error) { return pom, nil },
		func(_ context.Context) error { completeCalled = true; return nil },
		nil, nil, now, cli.ExportPomodoroConfig{},
	)
	if code != 0 {
		t.Errorf("expected exit 0 for stale resume, got %d", code)
	}
	if !completeCalled {
		t.Error("expected CompletePomodoro to be called for stale active")
	}
}

// TestStaleResumeFiresOnComplete verifies that on_complete fires when a stale pomodoro is completed via resume.
func TestStaleResumeFiresOnComplete(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-30 * time.Minute)
	pom := &taskv1.Pomodoro{
		Id:      1,
		TaskId:  42,
		StartAt: timestamppb.New(startAt),
	}

	marker := t.TempDir() + "/complete-marker"
	hooks := cli.ExportPomodoroConfig{OnComplete: "touch " + marker}

	code := cli.ExportRunResumeWith(
		func(_ context.Context) (*taskv1.Pomodoro, error) { return pom, nil },
		func(_ context.Context) error { return nil },
		nil, nil, now, hooks,
	)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("expected on_complete marker to exist")
	}
}

// TestCancelFiresOnCancel verifies that on_cancel hook fires when explicit cancel succeeds.
func TestCancelFiresOnCancel(t *testing.T) {
	marker := t.TempDir() + "/cancel-marker"
	hooks := cli.ExportPomodoroConfig{OnCancel: "touch " + marker}

	code := cli.ExportRunCancelWithHooks(func(_ context.Context) error { return nil }, hooks)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("expected on_cancel marker to exist")
	}
}

// TestCancelHookNotFiredOnError verifies that on_cancel doesn't fire when cancel RPC fails.
func TestCancelHookNotFiredOnError(t *testing.T) {
	marker := t.TempDir() + "/cancel-marker"
	hooks := cli.ExportPomodoroConfig{OnCancel: "touch " + marker}

	code := cli.ExportRunCancelWithHooks(func(_ context.Context) error {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no active"))
	}, hooks)
	if code == 0 {
		t.Error("expected non-zero exit")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Error("on_cancel should not fire when cancel RPC fails")
	}
}

// TestHookNonZeroExitIsWarningOnly verifies a failing hook doesn't abort the command.
func TestHookNonZeroExitIsWarningOnly(t *testing.T) {
	hooks := cli.ExportPomodoroConfig{OnCancel: "false"}
	// Should still exit 0 (cancel succeeded) even though hook exits non-zero.
	code := cli.ExportRunCancelWithHooks(func(_ context.Context) error { return nil }, hooks)
	if code != 0 {
		t.Errorf("expected exit 0 even when hook fails, got %d", code)
	}
}
