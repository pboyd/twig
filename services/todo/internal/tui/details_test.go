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
// and that the Completed: line is still present. (unstyled path)
func TestRenderDetails_CompletedTaskNoStrikethrough(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	task := &taskv1.Task{
		Id:          42,
		Name:        "finished work",
		CompletedAt: timestamppb.New(now),
	}

	out := renderDetails(task, 80, false)

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

// TestRenderDetails_StyledHeader (T017) asserts the styled path renders name as bold header
// and labels as dim column-aligned text (C5.1/C5.2).
func TestRenderDetails_StyledHeader(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	task := &taskv1.Task{
		Id:          7,
		Name:        "my task",
		Estimate:    3,
		CompletedAt: timestamppb.New(now),
	}

	out := renderDetails(task, 60, true)

	// Name must appear.
	if !strings.Contains(out, "my task") {
		t.Errorf("renderDetails styled: task name missing; got:\n%q", out)
	}
	// Labels must appear (dim column-aligned).
	for _, label := range []string{"ID", "Est", "Completed"} {
		if !strings.Contains(out, label) {
			t.Errorf("renderDetails styled: label %q missing; got:\n%q", label, out)
		}
	}
	// Unstyled path: no ANSI codes on name (this checks styled=false, not styled=true).
	plain := renderDetails(task, 60, false)
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("renderDetails unstyled: must not emit ANSI codes; got:\n%q", plain)
	}
}
