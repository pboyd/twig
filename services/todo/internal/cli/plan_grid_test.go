package cli_test

import (
	"strings"
	"testing"
	"time"

	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// rowsOf splits a RenderGrid output into lines, trimming the trailing newline.
func rowsOf(out string) []string {
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

// fixedTime returns a time.Time set to the given hour:minute on 2026-05-27 in UTC.
func fixedTime(hour, min int) time.Time {
	return time.Date(2026, 5, 27, hour, min, 0, 0, time.UTC)
}

// TestRenderGrid_EmptyDay verifies that an empty day renders the default
// 08:00–17:00 window: 10 hour lines and 27 quarter-hour text rows (37 total).
func TestRenderGrid_EmptyDay(t *testing.T) {
	out := cli.RenderGrid(nil, "2026-05-27", fixedTime(9, 0), 80, false)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	want := 37 // 9 hours × 4 rows + 1 closing hour line
	if len(lines) != want {
		t.Fatalf("expected %d rows for default window, got %d:\n%s", want, len(lines), out)
	}

	// First row must be the 08:00 hour divider.
	wantFirst := "08:00 ├" + strings.Repeat("─", 72) + "┤"
	if lines[0] != wantFirst {
		t.Errorf("first row:\n  got  %q\n  want %q", lines[0], wantFirst)
	}

	// Row 1 (08:15) must be a quarter-hour text row.
	wantQH := "      │" + strings.Repeat(" ", 72) + "│"
	if lines[1] != wantQH {
		t.Errorf("row 1 (08:15):\n  got  %q\n  want %q", lines[1], wantQH)
	}

	// Last row must be the 17:00 closing hour divider.
	wantLast := "17:00 ├" + strings.Repeat("─", 72) + "┤"
	if lines[len(lines)-1] != wantLast {
		t.Errorf("last row:\n  got  %q\n  want %q", lines[len(lines)-1], wantLast)
	}
}

// TestRenderGrid_WindowExtensionEarly verifies that an entry starting before
// 08:00 causes the window to extend downward to the enclosing hour.
func TestRenderGrid_WindowExtensionEarly(t *testing.T) {
	// Entry at 07:30 (450 min) — snap start = 450, floor to hour = 07:00.
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Early start", StartMinute: 450, DurationMinute: 60},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(9, 0), 80, false)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	wantFirst := "07:00 ├" + strings.Repeat("─", 72) + "┤"
	if lines[0] != wantFirst {
		t.Errorf("first row with early entry:\n  got  %q\n  want %q", lines[0], wantFirst)
	}

	// Window should be 07:00–17:00 = 10 hours × 4 rows + 1 = 41 rows.
	want := 41
	if len(lines) != want {
		t.Errorf("expected %d rows for 07:00–17:00 window, got %d", want, len(lines))
	}
}

// TestRenderGrid_WindowExtensionLate verifies that an entry ending after 17:00
// causes the window to extend upward to the enclosing hour.
func TestRenderGrid_WindowExtensionLate(t *testing.T) {
	// Entry at 17:15 (1035 min) for 30 min → end = 1065, snap up = 1065, ceil hour = 18:00.
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Late end", StartMinute: 1035, DurationMinute: 30},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(9, 0), 80, false)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	wantLast := "18:00 ├" + strings.Repeat("─", 72) + "┤"
	if lines[len(lines)-1] != wantLast {
		t.Errorf("last row with late entry:\n  got  %q\n  want %q", lines[len(lines)-1], wantLast)
	}

	// Window should be 08:00–18:00 = 10 hours × 4 rows + 1 = 41 rows.
	want := 41
	if len(lines) != want {
		t.Errorf("expected %d rows for 08:00–18:00 window, got %d", want, len(lines))
	}
}

// --- Phase 4: US2 entry box tests (T016) ---

// TestRenderGrid_EntryBox_TwoHour checks that a 08:00–10:00 entry renders with
// the correct top/bottom heavy borders and its label on the first interior row.
func TestRenderGrid_EntryBox_TwoHour(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Focus", StartMinute: 480, DurationMinute: 120},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false)
	lines := rowsOf(out)

	// Row 0 = 08:00: top edge
	if !strings.HasPrefix(lines[0], "08:00 ┏") {
		t.Errorf("row 0 (top): got %q", lines[0])
	}
	if !strings.HasSuffix(lines[0], "┓") {
		t.Errorf("row 0 (top) right edge: got %q", lines[0])
	}

	// Row 1 = 08:15: first interior, has label
	if !strings.Contains(lines[1], "[1] 08:00-10:00 Focus") {
		t.Errorf("row 1 (first interior): got %q", lines[1])
	}
	if !strings.HasPrefix(lines[1], "      ┃") {
		t.Errorf("row 1 left edge: got %q", lines[1])
	}

	// Row 8 = 10:00: bottom edge
	if !strings.HasPrefix(lines[8], "10:00 ┗") {
		t.Errorf("row 8 (bottom): got %q", lines[8])
	}
	if !strings.HasSuffix(lines[8], "┛") {
		t.Errorf("row 8 (bottom) right edge: got %q", lines[8])
	}
}

// TestRenderGrid_EntryBox_Truncation checks that a long label is truncated with "..."
// when the available interior space is only one row (10:00–10:30).
func TestRenderGrid_EntryBox_Truncation(t *testing.T) {
	longName := strings.Repeat("A", 80) // exceeds any reasonable field width
	entries := []*planv1.PlanEntry{
		{Id: 2, Name: longName, StartMinute: 600, DurationMinute: 30},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false)
	lines := rowsOf(out)

	// Row 9 = 10:15: only interior row; must end with "..." (after closing │/┃)
	// The line ends with ┃ and before that the last visible text chars are "..."
	interior := lines[9]
	if !strings.HasSuffix(interior, "...┃") {
		t.Errorf("truncated interior row should end ...┃, got %q", interior)
	}
}

// TestRenderGrid_EntryBox_Wrapping checks that a long label wraps across interior rows
// in an 11:15–12:00 entry (2 interior rows).
func TestRenderGrid_EntryBox_Wrapping(t *testing.T) {
	// fieldWidth=72; prefix="[1] 11:15-12:00 "=16 chars; need name > 56 chars to wrap.
	name := strings.Repeat("X", 60)
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: name, StartMinute: 675, DurationMinute: 45},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false)
	lines := rowsOf(out)

	// 11:15 = minute 675; winStart=480; topLine=(675-480)/15=13
	// Interior rows: 14, 15 (11:30, 11:45)
	// Row 14 must have the beginning of the label.
	if !strings.Contains(lines[14], "[1] 11:15-12:00") {
		t.Errorf("row 14 (first interior): got %q", lines[14])
	}
	// Row 15 must be non-blank (label wrapped onto it).
	interior15 := strings.TrimPrefix(lines[15], "      ┃")
	interior15 = strings.TrimSuffix(interior15, "┃")
	if strings.TrimSpace(interior15) == "" {
		t.Errorf("row 15 (second interior) should have wrapped label, got %q", lines[15])
	}
}

// TestRenderGrid_EntryBox_SingleRow checks a 13:00–13:15 entry renders as one row
// using junction characters.
func TestRenderGrid_EntryBox_SingleRow(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: 780, DurationMinute: 15},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false)
	lines := rowsOf(out)

	// 13:00 = minute 780; winStart=480; topLine=(780-480)/15=20
	row := lines[20]
	if !strings.HasPrefix(row, "13:00 ┣") {
		t.Errorf("single-row left junction: got %q", row)
	}
	if !strings.HasSuffix(row, "┫") {
		t.Errorf("single-row right junction: got %q", row)
	}
	if !strings.Contains(row, "[1] 13:00-13:15 Standup") {
		t.Errorf("single-row label: got %q", row)
	}
	// The row immediately before (12:45) and after (13:15) must be normal grid rows.
	if strings.ContainsAny(lines[19], "┣┗┛┏┓") {
		t.Errorf("row before single-row should be plain grid, got %q", lines[19])
	}
	if strings.ContainsAny(lines[21], "┣┗┛┏┓") {
		t.Errorf("row after single-row should be plain grid, got %q", lines[21])
	}
}

// --- Phase 5: US3 adjacent shared border tests (T018) ---

// TestRenderGrid_AdjacentSharedBorder checks that a multi-row entry ending at 10:00
// and a second entry starting at 10:00 share a single ┣━━━┫ line at that row.
func TestRenderGrid_AdjacentSharedBorder(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Morning", StartMinute: 480, DurationMinute: 120},  // 08:00-10:00
		{Id: 2, Name: "Review", StartMinute: 600, DurationMinute: 30},    // 10:00-10:30
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false)
	lines := rowsOf(out)

	// Row 8 = 10:00: must be the shared ┣━━━┫ border (not ┗ or ┏).
	row8 := lines[8]
	if !strings.HasPrefix(row8, "10:00 ┣") {
		t.Errorf("shared border at 10:00: got %q", row8)
	}
	if !strings.HasSuffix(row8, "┫") {
		t.Errorf("shared border right end: got %q", row8)
	}
	// Must not contain ┗ or ┏.
	if strings.ContainsAny(row8, "┗┏") {
		t.Errorf("shared border must not have corner chars: got %q", row8)
	}
}

// TestRenderGrid_AdjacentSingleRows checks that two adjacent 15-min entries each
// render as their own ┣━━━┫ row on consecutive lines.
func TestRenderGrid_AdjacentSingleRows(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: 780, DurationMinute: 15},  // 13:00-13:15
		{Id: 2, Name: "Retro", StartMinute: 795, DurationMinute: 15},    // 13:15-13:30
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false)
	lines := rowsOf(out)

	// Row 20 = 13:00: first single-row entry.
	if !strings.HasPrefix(lines[20], "13:00 ┣") {
		t.Errorf("row 20 (13:00 entry): got %q", lines[20])
	}
	if !strings.Contains(lines[20], "Standup") {
		t.Errorf("row 20 missing label: got %q", lines[20])
	}

	// Row 21 = 13:15: second single-row entry.
	if !strings.HasPrefix(lines[21], "      ┣") {
		t.Errorf("row 21 (13:15 entry): got %q", lines[21])
	}
	if !strings.Contains(lines[21], "Retro") {
		t.Errorf("row 21 missing label: got %q", lines[21])
	}
}

// --- Phase 6: US4 now-marker tests (T021) ---

// TestRenderGrid_NowMarker_Today verifies that the ▶ marker appears in the left
// gutter on the 10:30 row when now=10:37 and day is today.
func TestRenderGrid_NowMarker_Today(t *testing.T) {
	now := time.Date(2026, 5, 27, 10, 37, 0, 0, time.UTC)
	out := cli.RenderGrid(nil, "2026-05-27", now, 80, false)
	lines := rowsOf(out)

	// 10:30 = minute 630; winStart=480; row=(630-480)/15=10
	row10 := lines[10]
	if !strings.Contains(row10, "▶") {
		t.Errorf("row 10 (10:30) should have ▶ marker: got %q", row10)
	}

	// No other row should have the marker.
	for i, l := range lines {
		if i == 10 {
			continue
		}
		if strings.Contains(l, "▶") {
			t.Errorf("unexpected ▶ on row %d: %q", i, l)
		}
	}
}

// TestRenderGrid_NowMarker_NotToday verifies that no ▶ marker appears when
// the rendered day is not today.
func TestRenderGrid_NowMarker_NotToday(t *testing.T) {
	now := time.Date(2026, 5, 27, 10, 37, 0, 0, time.UTC)
	// Render yesterday.
	out := cli.RenderGrid(nil, "2026-05-26", now, 80, false)
	if strings.Contains(out, "▶") {
		t.Errorf("expected no ▶ marker on non-today render, but found one:\n%s", out)
	}
}

// --- Phase 7: completed-task tests (T023) ---

// TestRenderGrid_CompletedTask_NoTTY verifies that a completed entry produces no
// ANSI escape codes when isTTY=false.
func TestRenderGrid_CompletedTask_NoTTY(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", StartMinute: 480, DurationMinute: 120, Completed: true},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false)
	if strings.Contains(out, "\x1b[") {
		t.Errorf("isTTY=false: unexpected escape codes in output:\n%s", out)
	}
	// Label must still be readable.
	if !strings.Contains(out, "Done task") {
		t.Errorf("isTTY=false: label missing from output:\n%s", out)
	}
}

// TestRenderGrid_CompletedTask_TTY verifies that a completed entry wraps its label
// in dim+strikethrough ANSI codes when isTTY=true.
func TestRenderGrid_CompletedTask_TTY(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", StartMinute: 480, DurationMinute: 120, Completed: true},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true)
	if !strings.Contains(out, "\x1b[2;9m") {
		t.Errorf("isTTY=true: expected dim+strikethrough code \\x1b[2;9m in output")
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Errorf("isTTY=true: expected reset code \\x1b[0m in output")
	}
	// Border chars must NOT be wrapped: top edge row contains ┏ but no escape code on same line.
	lines := rowsOf(out)
	topRow := lines[0]
	if strings.Contains(topRow, "\x1b[") {
		t.Errorf("top border row must not contain escape codes: %q", topRow)
	}
}
