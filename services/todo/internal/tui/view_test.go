package tui

import (
	"strings"
	"testing"
	"time"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestRenderList_StrikethroughOnCompletedRow (T-F) asserts that renderList emits
// ANSI strikethrough around the name of a completed row and not on an incomplete row.
func TestRenderList_StrikethroughOnCompletedRow(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{
		{Id: 1, Name: "done task", CompletedAt: now},
		{Id: 2, Name: "todo task"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	// Show completed tasks so the completed row is in m.visible.
	m.showCompleted = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(80)

	if !strings.Contains(out, "\x1b[9m") {
		t.Errorf("renderList: expected ANSI strikethrough open code for completed row; got:\n%q", out)
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Errorf("renderList: expected ANSI reset code for completed row; got:\n%q", out)
	}

	// Verify the incomplete row does NOT contain strikethrough.
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "todo task") && strings.Contains(line, "\x1b[9m") {
			t.Errorf("incomplete row must not contain strikethrough; got:\n%q", line)
		}
	}
}

// TestRenderList_NoStrikethroughWhenUnstyled asserts that renderList produces no
// ANSI strikethrough codes when styled=false (non-TTY mode).
func TestRenderList_NoStrikethroughWhenUnstyled(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	tasks := []*taskv1.Task{
		{Id: 1, Name: "done task", CompletedAt: now},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, false)
	m.showCompleted = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(80)

	if strings.Contains(out, "\x1b[9m") {
		t.Errorf("renderList with styled=false must not emit strikethrough; got:\n%q", out)
	}
}

// TestRenderList_StrikethroughAtMultipleDepths (T013 / FR-003) asserts that
// strikethrough is present on every completed row regardless of nesting depth.
func TestRenderList_StrikethroughAtMultipleDepths(t *testing.T) {
	now := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	p1ID := int64(1)
	p2ID := int64(2)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent done", CompletedAt: now},
		{Id: 2, Name: "child done", ParentId: &p1ID, CompletedAt: now},
		{Id: 3, Name: "grandchild done", ParentId: &p2ID, CompletedAt: now},
		{Id: 4, Name: "todo root"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewStyledModel(nil, tree, true)
	// Show completed so all rows are visible.
	m.showCompleted = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
	// Expand all to see nested rows.
	m.expanded[1] = true
	m.expanded[2] = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)

	out := m.renderList(120)

	// Every completed task name must be preceded by \x1b[9m somewhere on its line.
	lines := strings.Split(out, "\n")
	completedNames := []string{"parent done", "child done", "grandchild done"}
	for _, name := range completedNames {
		found := false
		for _, line := range lines {
			if strings.Contains(line, name) {
				if strings.Contains(line, "\x1b[9m") {
					found = true
				}
				break
			}
		}
		if !found {
			t.Errorf("renderList: expected strikethrough on completed row %q", name)
		}
	}

	// Incomplete row must NOT have strikethrough.
	for _, line := range lines {
		if strings.Contains(line, "todo root") && strings.Contains(line, "\x1b[9m") {
			t.Errorf("incomplete row 'todo root' must not have strikethrough; got:\n%q", line)
		}
	}
}
