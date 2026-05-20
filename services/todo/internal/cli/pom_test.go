package cli_test

import (
	"bytes"
	"context"
	"fmt"
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
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no active pomodoro"))
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
		nil, nil, nil, time.Now(), "",
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
		nil, nil, now, "",
	)
	if code != 0 {
		t.Errorf("expected exit 0 for stale resume, got %d", code)
	}
	if !completeCalled {
		t.Error("expected CompletePomodoro to be called for stale active")
	}
}
