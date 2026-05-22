package cli_test

import (
	"strings"
	"testing"

	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

func TestRenderGrid_Empty(t *testing.T) {
	out := cli.RenderGrid(nil)
	if out != "Plan is empty.\n" {
		t.Errorf("empty = %q, want %q", out, "Plan is empty.\n")
	}
	out = cli.RenderGrid([]*planv1.PlanEntry{})
	if out != "Plan is empty.\n" {
		t.Errorf("empty slice = %q, want %q", out, "Plan is empty.\n")
	}
}

func TestRenderGrid_SingleEntry_OnQuarterHour(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-21", Id: 1, Name: "Meeting", StartMinute: 540, DurationMinute: 60},
	}
	out := cli.RenderGrid(entries)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// 9:00–10:00 → floor=540, ceil=600 → 4 rows (540,555,570,585)
	if len(lines) != 4 {
		t.Errorf("expected 4 rows, got %d:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "09:00") {
		t.Errorf("first line should start with 09:00, got: %q", lines[0])
	}
	if !strings.Contains(lines[0], "┌ 1 Meeting") {
		t.Errorf("start row should contain ┌ marker and name, got: %q", lines[0])
	}
	if strings.Contains(lines[1], "Meeting") {
		t.Errorf("middle row should not repeat name, got: %q", lines[1])
	}
	if !strings.HasSuffix(lines[1], "│") {
		t.Errorf("middle row should have │ span marker, got: %q", lines[1])
	}
	if strings.Contains(lines[3], "Meeting") {
		t.Errorf("end row should not repeat name, got: %q", lines[3])
	}
	if !strings.HasSuffix(lines[3], "└") {
		t.Errorf("end row should have └ span marker, got: %q", lines[3])
	}
}

func TestRenderGrid_TwoEntries_WithGap(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-21", Id: 1, Name: "Focus", StartMinute: 480, DurationMinute: 60}, // 8:00–9:00
		{Day: "2026-05-21", Id: 2, Name: "Lunch", StartMinute: 720, DurationMinute: 60}, // 12:00–13:00
	}
	out := cli.RenderGrid(entries)
	// Floor: 480 → 480; Ceil: 780 → 780. Rows: 480..780 in steps of 15 = 20 rows.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 20 {
		t.Errorf("expected 20 rows, got %d", len(lines))
	}
	// Row 0 (08:00) should contain entry 1.
	if !strings.Contains(lines[0], "1 Focus") {
		t.Errorf("row 0: %q", lines[0])
	}
	// Row 4 (09:00) should be a gap.
	if strings.Contains(lines[4], "Focus") || strings.Contains(lines[4], "Lunch") {
		t.Errorf("row 4 (gap) should be empty: %q", lines[4])
	}
	// Row 16 (12:00) should contain entry 2.
	if !strings.Contains(lines[16], "2 Lunch") {
		t.Errorf("row 16: %q", lines[16])
	}
}

func TestRenderGrid_OffQuarterHour(t *testing.T) {
	// Entry starts at 10:05 (605 minutes).
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-21", Id: 1, Name: "Late start", StartMinute: 605, DurationMinute: 30},
	}
	out := cli.RenderGrid(entries)
	// Floor: 600 (10:00), Ceil: 645 (10:45). Rows: 600, 615, 630.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 rows, got %d:\n%s", len(lines), out)
	}
	// First row should have ~ prefix with actual start time.
	if !strings.HasPrefix(lines[0], "~10:05") {
		t.Errorf("expected ~10:05 prefix, got: %q", lines[0])
	}
}

func TestRenderGrid_SpanMarkers(t *testing.T) {
	cases := []struct {
		name    string
		entries []*planv1.PlanEntry
		want    []string
	}{
		{
			// T002: single-slot entry must use the dash marker, not ┌/└.
			name: "single-slot entry uses dash marker",
			entries: []*planv1.PlanEntry{
				{Id: 1, Name: "Sprint", StartMinute: 540, DurationMinute: 15},
			},
			want: []string{
				"09:00 │ ─ 1 Sprint",
			},
		},
		{
			// T003: two-slot entry must show ┌ on row 1 and └ on row 2 with no │ between.
			name: "two-slot entry uses start and end markers without middle row",
			entries: []*planv1.PlanEntry{
				{Id: 1, Name: "Focus", StartMinute: 540, DurationMinute: 30},
			},
			want: []string{
				"09:00 │ ┌ 1 Focus",
				"09:15 │ └",
			},
		},
		{
			// T004: adjacent entries — A's └ and B's ┌ must be on consecutive lines.
			name: "adjacent entries have no blank row between them",
			entries: []*planv1.PlanEntry{
				{Id: 1, Name: "A", StartMinute: 480, DurationMinute: 30},
				{Id: 2, Name: "B", StartMinute: 510, DurationMinute: 30},
			},
			want: []string{
				"08:00 │ ┌ 1 A",
				"08:15 │ └",
				"08:30 │ ┌ 2 B",
				"08:45 │ └",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := cli.RenderGrid(tc.entries)
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) != len(tc.want) {
				t.Fatalf("expected %d lines, got %d:\n%s", len(tc.want), len(lines), out)
			}
			for i, want := range tc.want {
				if lines[i] != want {
					t.Errorf("line %d: got %q, want %q", i, lines[i], want)
				}
			}
		})
	}
}

func TestRenderGrid_TaskLinkedEntry(t *testing.T) {
	// Server already substituted the task name into Name field.
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-21", Id: 3, Name: "Deep Work", TaskId: 42, StartMinute: 480, DurationMinute: 60},
	}
	out := cli.RenderGrid(entries)
	if !strings.Contains(out, "3 Deep Work") {
		t.Errorf("expected task name in grid, got:\n%s", out)
	}
}
