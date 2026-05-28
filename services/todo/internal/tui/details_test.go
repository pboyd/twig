package tui

import (
	"strings"
	"testing"
	"time"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestRenderDetails_CompletedTaskNoStrikethrough (T-G) asserts that renderDetails
// does not apply strikethrough or dim to the task name even when CompletedAt is set,
// and that the Completed: line is still present.
func TestRenderDetails_CompletedTaskNoStrikethrough(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	task := &taskv1.Task{
		Id:          42,
		Name:        "finished work",
		CompletedAt: timestamppb.New(now),
	}

	out := renderDetails(task, 80)

	if strings.Contains(out, "\x1b[9m") {
		t.Errorf("renderDetails: name must not have strikethrough (\\x1b[9m); got:\n%q", out)
	}
	if strings.Contains(out, "\x1b[2m") {
		t.Errorf("renderDetails: name must not have dim (\\x1b[2m); got:\n%q", out)
	}
	if strings.Contains(out, "\x1b[2;9m") {
		t.Errorf("renderDetails: name must not have dim+strikethrough (\\x1b[2;9m); got:\n%q", out)
	}
	if !strings.Contains(out, "finished work") {
		t.Errorf("renderDetails: task name missing from output; got:\n%q", out)
	}
	if !strings.Contains(out, "Completed:") {
		t.Errorf("renderDetails: Completed: line missing from output; got:\n%q", out)
	}
}
