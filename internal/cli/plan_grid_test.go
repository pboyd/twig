package cli_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	"github.com/pboyd/twig/internal/cli"
)

func pint32(v int32) *int32 { return &v }

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
	out := cli.RenderGrid(nil, "2026-05-27", fixedTime(9, 0), 80, false, cli.GridOptions{})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	want := 37 // 9 hours × 4 rows + 1 closing hour line
	if len(lines) != want {
		t.Fatalf("expected %d rows for default window, got %d:\n%s", want, len(lines), out)
	}

	// First row must be the 08:00 hour divider.
	// Gutter is 7 chars (label + space + marker-col); inner region is 71 chars.
	wantFirst := "08:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[0] != wantFirst {
		t.Errorf("first row:\n  got  %q\n  want %q", lines[0], wantFirst)
	}

	// Row 1 (08:15) must be a quarter-hour text row.
	wantQH := "       │" + strings.Repeat(" ", 71) + "│"
	if lines[1] != wantQH {
		t.Errorf("row 1 (08:15):\n  got  %q\n  want %q", lines[1], wantQH)
	}

	// Last row must be the 17:00 closing hour divider.
	wantLast := "17:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[len(lines)-1] != wantLast {
		t.Errorf("last row:\n  got  %q\n  want %q", lines[len(lines)-1], wantLast)
	}
}

// TestRenderGrid_WindowExtensionEarly verifies that an entry starting before
// 08:00 causes the window to extend downward to the enclosing hour.
func TestRenderGrid_WindowExtensionEarly(t *testing.T) {
	// Entry at 07:30 (450 min) — snap start = 450, floor to hour = 07:00.
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Early start", StartMinute: pint32(450), DurationMinute: 60},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(9, 0), 80, false, cli.GridOptions{})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	wantFirst := "07:00  ├" + strings.Repeat("─", 71) + "┤"
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
		{Day: "2026-05-27", Id: 1, Name: "Late end", StartMinute: pint32(1035), DurationMinute: 30},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(9, 0), 80, false, cli.GridOptions{})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	wantLast := "18:00  ├" + strings.Repeat("─", 71) + "┤"
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
		{Id: 1, Name: "Focus", StartMinute: pint32(480), DurationMinute: 120},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// Row 0 = 08:00: top edge; box is inset one column from each rail.
	if !strings.HasPrefix(lines[0], "08:00  ├─┏") {
		t.Errorf("row 0 (top): got %q", lines[0])
	}
	if !strings.HasSuffix(lines[0], "┓─┤") {
		t.Errorf("row 0 (top) right edge: got %q", lines[0])
	}

	// Row 1 = 08:15: first interior, has label; rail + space-pad flank the box.
	if !strings.Contains(lines[1], "[1] 08:00-10:00 Focus") {
		t.Errorf("row 1 (first interior): got %q", lines[1])
	}
	if !strings.HasPrefix(lines[1], "       │ ┃") {
		t.Errorf("row 1 left edge: got %q", lines[1])
	}

	// Row 8 = 10:00: bottom edge; box inset from rails.
	if !strings.HasPrefix(lines[8], "10:00  ├─┗") {
		t.Errorf("row 8 (bottom): got %q", lines[8])
	}
	if !strings.HasSuffix(lines[8], "┛─┤") {
		t.Errorf("row 8 (bottom) right edge: got %q", lines[8])
	}
}

// TestRenderGrid_EntryBox_Truncation checks that a long label is truncated with "..."
// when the available interior space is only one row (10:00–10:30).
func TestRenderGrid_EntryBox_Truncation(t *testing.T) {
	longName := strings.Repeat("A", 80) // exceeds any reasonable field width
	entries := []*planv1.PlanEntry{
		{Id: 2, Name: longName, StartMinute: pint32(600), DurationMinute: 30},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// Row 9 = 10:15: only interior row; must end with "..." (after closing │/┃)
	// The line ends with ┃ and before that the last visible text chars are "..."
	interior := lines[9]
	if !strings.HasSuffix(interior, "...┃ │") {
		t.Errorf("truncated interior row should end ...┃ │, got %q", interior)
	}
}

// TestRenderGrid_EntryBox_Wrapping checks that a long label wraps across interior rows
// in an 11:15–12:00 entry (2 interior rows).
func TestRenderGrid_EntryBox_Wrapping(t *testing.T) {
	// fieldWidth=72; prefix="[1] 11:15-12:00 "=16 chars; need name > 56 chars to wrap.
	name := strings.Repeat("X", 60)
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: name, StartMinute: pint32(675), DurationMinute: 45},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// 11:15 = minute 675; winStart=480; topLine=(675-480)/15=13
	// Interior rows: 14, 15 (11:30, 11:45)
	// Row 14 must have the beginning of the label.
	if !strings.Contains(lines[14], "[1] 11:15-12:00") {
		t.Errorf("row 14 (first interior): got %q", lines[14])
	}
	// Row 15 must be non-blank (label wrapped onto it).
	interior15 := strings.TrimPrefix(lines[15], "       │ ┃")
	interior15 = strings.TrimSuffix(interior15, "┃ │")
	if strings.TrimSpace(interior15) == "" {
		t.Errorf("row 15 (second interior) should have wrapped label, got %q", lines[15])
	}
}

// TestRenderGrid_EntryBox_SingleRow checks a 13:00–13:15 entry renders as one row
// using junction characters.
func TestRenderGrid_EntryBox_SingleRow(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(780), DurationMinute: 15},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// 13:00 = minute 780; winStart=480; topLine=(780-480)/15=20
	row := lines[20]
	if !strings.HasPrefix(row, "13:00  ├─┣") {
		t.Errorf("single-row left junction: got %q", row)
	}
	if !strings.HasSuffix(row, "┫─┤") {
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
		{Id: 1, Name: "Morning", StartMinute: pint32(480), DurationMinute: 120}, // 08:00-10:00
		{Id: 2, Name: "Review", StartMinute: pint32(600), DurationMinute: 30},   // 10:00-10:30
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// Row 8 = 10:00: must be the shared ┣━━━┫ border (not ┗ or ┏).
	row8 := lines[8]
	if !strings.HasPrefix(row8, "10:00  ├─┣") {
		t.Errorf("shared border at 10:00: got %q", row8)
	}
	if !strings.HasSuffix(row8, "┫─┤") {
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
		{Id: 1, Name: "Standup", StartMinute: pint32(780), DurationMinute: 15}, // 13:00-13:15
		{Id: 2, Name: "Retro", StartMinute: pint32(795), DurationMinute: 15},   // 13:15-13:30
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// Row 20 = 13:00: first single-row entry; hour row so rail+─ flank the box.
	if !strings.HasPrefix(lines[20], "13:00  ├─┣") {
		t.Errorf("row 20 (13:00 entry): got %q", lines[20])
	}
	if !strings.Contains(lines[20], "Standup") {
		t.Errorf("row 20 missing label: got %q", lines[20])
	}

	// Row 21 = 13:15: second single-row entry; non-hour row so rail+space flank the box.
	if !strings.HasPrefix(lines[21], "       │ ┣") {
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
	out := cli.RenderGrid(nil, "2026-05-27", now, 80, false, cli.GridOptions{})
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
	out := cli.RenderGrid(nil, "2026-05-26", now, 80, false, cli.GridOptions{})
	if strings.Contains(out, "▶") {
		t.Errorf("expected no ▶ marker on non-today render, but found one:\n%s", out)
	}
}

// --- Phase 7: completed-task tests (T023) ---

// TestRenderGrid_CompletedTask_NoTTY verifies that a completed entry produces no
// ANSI escape codes when isTTY=false.
func TestRenderGrid_CompletedTask_NoTTY(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", StartMinute: pint32(480), DurationMinute: 120, Completed: true},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
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
		{Id: 1, Name: "Done task", StartMinute: pint32(480), DurationMinute: 120, Completed: true},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, cli.GridOptions{})
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

// --- Feature 014: Calendar Grid Padding & Gutter Refinement ---

// TestPlanGrid_MultiHourEntry_HourLinesFlankBox verifies that hour rows inside a
// multi-hour entry still show light hour lines flanking the heavy box (US1, T005).
func TestPlanGrid_MultiHourEntry_HourLinesFlankBox(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Focus", StartMinute: pint32(480), DurationMinute: 120}, // 08:00-10:00
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// Row 0 = 08:00: top edge row (hour) — light ─ between rail and heavy corner on both sides.
	if !strings.HasPrefix(lines[0], "08:00  ├─┏") {
		t.Errorf("row 0 top-edge left: got %q", lines[0])
	}
	if !strings.HasSuffix(lines[0], "┓─┤") {
		t.Errorf("row 0 top-edge right: got %q", lines[0])
	}

	// Row 4 = 09:00: interior hour row — light hour line flanks the heavy box sides.
	if !strings.HasPrefix(lines[4], "09:00  ├─┃") {
		t.Errorf("row 4 (09:00 interior) left: got %q", lines[4])
	}
	if !strings.HasSuffix(lines[4], "┃─┤") {
		t.Errorf("row 4 (09:00 interior) right: got %q", lines[4])
	}

	// Row 8 = 10:00: bottom edge row (hour) — same flanking rule.
	if !strings.HasPrefix(lines[8], "10:00  ├─┗") {
		t.Errorf("row 8 bottom-edge left: got %q", lines[8])
	}
	if !strings.HasSuffix(lines[8], "┛─┤") {
		t.Errorf("row 8 bottom-edge right: got %q", lines[8])
	}
}

// TestPlanGrid_OffHourEntry_PaddingPreserved verifies that an entry with an off-hour
// start/end has exactly one space of padding between each rail and the box (US1, T009).
func TestPlanGrid_OffHourEntry_PaddingPreserved(t *testing.T) {
	// 09:15–09:45: top at row 5 (09:15, non-hour), bottom at row 7 (09:45, non-hour).
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Quick sync", StartMinute: pint32(555), DurationMinute: 30},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// Row 5 = 09:15: top edge on a non-hour row — space (not ─) between rail and box.
	if !strings.HasPrefix(lines[5], "       │ ┏") {
		t.Errorf("row 5 (09:15 top) left padding: got %q", lines[5])
	}
	if !strings.HasSuffix(lines[5], "┓ │") {
		t.Errorf("row 5 (09:15 top) right padding: got %q", lines[5])
	}

	// Row 7 = 09:45: bottom edge on a non-hour row — same space padding.
	if !strings.HasPrefix(lines[7], "       │ ┗") {
		t.Errorf("row 7 (09:45 bottom) left padding: got %q", lines[7])
	}
	if !strings.HasSuffix(lines[7], "┛ │") {
		t.Errorf("row 7 (09:45 bottom) right padding: got %q", lines[7])
	}
}

// TestPlanGrid_SharedBorder_PaddingPreserved verifies that two back-to-back 15-min
// entries produce a single ┣━…━┫ at the :15 row with space padding (non-hour, US1, T010).
func TestPlanGrid_SharedBorder_PaddingPreserved(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(780), DurationMinute: 15}, // 13:00-13:15
		{Id: 2, Name: "Retro", StartMinute: pint32(795), DurationMinute: 15},   // 13:15-13:30
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// Row 21 = 13:15: second single-row entry on a non-hour row.
	// Padding must be a space (not ─) and there must be exactly one ┣…┫ on this line.
	row21 := lines[21]
	if !strings.HasPrefix(row21, "       │ ┣") {
		t.Errorf("row 21 (13:15) left: got %q", row21)
	}
	if !strings.HasSuffix(row21, "┫ │") {
		t.Errorf("row 21 (13:15) right: got %q", row21)
	}
}

// TestPlanGrid_EmptyHours_FullGrid verifies that empty hour rows render as a full
// light ├─…─┤ and non-hour rows render as │ … │ (US2, T011).
func TestPlanGrid_EmptyHours_FullGrid(t *testing.T) {
	// One entry at 09:00–10:00; rows 0–3 (08:xx) and 9+ (10:15+) are empty.
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Deep work", StartMinute: pint32(540), DurationMinute: 60},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	wantHour := "08:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[0] != wantHour {
		t.Errorf("row 0 (08:00 empty hour):\n  got  %q\n  want %q", lines[0], wantHour)
	}

	wantQH := "       │" + strings.Repeat(" ", 71) + "│"
	if lines[1] != wantQH {
		t.Errorf("row 1 (08:15 empty non-hour):\n  got  %q\n  want %q", lines[1], wantQH)
	}

	// Row 12 = 11:00 (empty hour after the entry's window).
	wantHour11 := "11:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[12] != wantHour11 {
		t.Errorf("row 12 (11:00 empty hour):\n  got  %q\n  want %q", lines[12], wantHour11)
	}
}

// TestPlanGrid_FirstAndLastHourRows_FullGrid verifies the first and last rows of the
// visible window are full light hour lines with no entries present (US2, T013).
func TestPlanGrid_FirstAndLastHourRows_FullGrid(t *testing.T) {
	out := cli.RenderGrid(nil, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	wantFirst := "08:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[0] != wantFirst {
		t.Errorf("first row:\n  got  %q\n  want %q", lines[0], wantFirst)
	}

	wantLast := "17:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[len(lines)-1] != wantLast {
		t.Errorf("last row:\n  got  %q\n  want %q", lines[len(lines)-1], wantLast)
	}
}

// TestPlanGrid_NowMarker_GutterSpacing verifies the gutter layout: label + space +
// marker-col + rail, with the rail at the same column index on every row (US3, T014).
func TestPlanGrid_NowMarker_GutterSpacing(t *testing.T) {
	// now=08:00 → nowLine=0 (the 08:00 row itself carries the marker).
	now := time.Date(2026, 5, 27, 8, 0, 0, 0, time.UTC)
	out := cli.RenderGrid(nil, "2026-05-27", now, 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// Row 0 (08:00, marker present): gutter = "08:00 ▶" (label + 1 space + marker).
	if !strings.HasPrefix(lines[0], "08:00 ▶├") {
		t.Errorf("row 0 (now row) gutter: got %q, want prefix %q", lines[0], "08:00 ▶├")
	}

	// Row 4 (09:00, no marker): gutter = "09:00  " (label + 2 spaces).
	if !strings.HasPrefix(lines[4], "09:00  ├") {
		t.Errorf("row 4 (09:00 non-marker hour) gutter: got %q, want prefix %q", lines[4], "09:00  ├")
	}

	// Rail must be at rune index 7 on every row.
	for i, row := range lines {
		runes := []rune(row)
		if len(runes) < 8 {
			t.Errorf("row %d too short: %q", i, row)
			continue
		}
		r := runes[7]
		if r != '├' && r != '│' {
			t.Errorf("row %d: rune at index 7 = %q, want ├ or │", i, r)
		}
	}
}

// TestPlanGrid_NonTodayRender_MarkerColumnBlank verifies that a non-today render has
// no ▶ marker but the rail stays at the same column position (US3, T016).
func TestPlanGrid_NonTodayRender_MarkerColumnBlank(t *testing.T) {
	now := time.Date(2026, 5, 27, 8, 0, 0, 0, time.UTC)
	// Render yesterday — no marker should appear.
	out := cli.RenderGrid(nil, "2026-05-26", now, 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	if strings.Contains(out, "▶") {
		t.Errorf("non-today render should have no ▶ marker:\n%s", out)
	}

	// Rail must still be at rune index 7 on every row.
	for i, row := range lines {
		runes := []rune(row)
		if len(runes) < 8 {
			t.Errorf("row %d too short: %q", i, row)
			continue
		}
		r := runes[7]
		if r != '├' && r != '│' {
			t.Errorf("row %d: rune at index 7 = %q, want ├ or │", i, r)
		}
	}
}

// TestRenderGrid_GridOptions_Default verifies that zero-value GridOptions reproduces
// the same output as the prior no-options call (id shown, no selection highlight).
func TestRenderGrid_GridOptions_Default(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 3, Name: "Standup", StartMinute: pint32(780), DurationMinute: 15},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{})
	if !strings.Contains(out, "[3]") {
		t.Errorf("default options: expected [3] id prefix in output:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("default options isTTY=false: unexpected escape codes:\n%s", out)
	}
}

// TestRenderGrid_GridOptions_HideID verifies that HideID:true removes the "[id] " prefix.
func TestRenderGrid_GridOptions_HideID(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 7, Name: "Standup", StartMinute: pint32(780), DurationMinute: 15},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{HideID: true})
	if strings.Contains(out, "[7]") {
		t.Errorf("HideID: unexpected [7] prefix in output:\n%s", out)
	}
	if !strings.Contains(out, "Standup") {
		t.Errorf("HideID: entry name missing from output:\n%s", out)
	}
	if !strings.Contains(out, "13:00-13:15") {
		t.Errorf("HideID: time prefix missing from output:\n%s", out)
	}
}

// TestRenderGrid_GridOptions_SelectedID_Styled verifies that SelectedID highlights
// only the target entry's rows with bold ANSI codes when isTTY=true.
func TestRenderGrid_GridOptions_SelectedID_Styled(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Focus", StartMinute: pint32(480), DurationMinute: 120},
		{Id: 2, Name: "Review", StartMinute: pint32(600), DurationMinute: 60},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, cli.GridOptions{SelectedID: 1})
	lines := rowsOf(out)

	// Interior rows of entry 1 (rows 1–7) should have bold codes.
	for _, i := range []int{1, 2, 3} {
		if !strings.Contains(lines[i], "\x1b[1m") {
			t.Errorf("row %d (entry 1 interior): expected bold code, got %q", i, lines[i])
		}
	}
	// Entry 2 interior row should NOT have bold codes.
	if strings.Contains(lines[9], "\x1b[1m") {
		t.Errorf("row 9 (entry 2 interior): unexpected bold code on non-selected entry, got %q", lines[9])
	}
}

// TestRenderGrid_GridOptions_SelectedID_NoStyle verifies that SelectedID produces
// no ANSI codes when isTTY=false.
func TestRenderGrid_GridOptions_SelectedID_NoStyle(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 5, Name: "Standup", StartMinute: pint32(780), DurationMinute: 15},
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{SelectedID: 5})
	if strings.Contains(out, "\x1b[") {
		t.Errorf("SelectedID with isTTY=false: unexpected escape codes:\n%s", out)
	}
}

// ── US4: GridOptions.Styled regression (T014) ──────────────────────────────

// TestRenderGrid_StyledFalseIsUnchanged verifies that Styled:false produces
// byte-for-byte identical output to the zero-value GridOptions (T014).
func TestRenderGrid_StyledFalseIsUnchanged(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Focus", StartMinute: pint32(480), DurationMinute: 120},
		{Id: 2, Name: "Standup", StartMinute: pint32(780), DurationMinute: 15},
	}
	baseline := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{HideID: true, SelectedID: 1})
	withStyled := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{HideID: true, SelectedID: 1, Styled: false})

	if baseline != withStyled {
		t.Errorf("Styled:false must produce identical output to zero-value Styled field")
	}
}

// TestRenderGrid_StyledTrueAccentHighlight verifies that Styled:true + isTTY=true
// highlights the selected entry with a different style than plain bold (T014).
func TestRenderGrid_StyledTrueAccentHighlight(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Focus", StartMinute: pint32(480), DurationMinute: 120},
	}
	boldOnly := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, cli.GridOptions{HideID: true, SelectedID: 1, Styled: false})
	accented := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, cli.GridOptions{HideID: true, SelectedID: 1, Styled: true})

	// The accent output should contain ANSI codes (styled=true, isTTY=true).
	if !strings.Contains(accented, "\x1b[") {
		t.Errorf("Styled:true + isTTY=true: expected ANSI codes in output")
	}
	// The two outputs should differ (accent ≠ plain bold).
	if boldOnly == accented {
		t.Errorf("Styled:true should produce different output from Styled:false when isTTY=true")
	}
}

// ── US4: SelectionStyle cell-only highlight (T015) ─────────────────────────

// styleMarker wraps content with sentinel markers so tests can identify styled content.
func styleMarker(s string) string { return "<<" + s + ">>" }

// TestSelectionStyle_SingleRowEntry checks that SelectionStyle is applied only to
// the content cell of a single-row entry, not to the ┣/┫ rails or gutter (T015).
func TestSelectionStyle_SingleRowEntry(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 15},
	}
	opts := cli.GridOptions{HideID: true, SelectedID: 1, Styled: true, SelectionStyle: styleMarker}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, opts)
	lines := rowsOf(out)

	// Find the single-row entry line (contains ┣ and ┫).
	var entryLine string
	for _, l := range lines {
		if strings.Contains(l, "┣") && strings.Contains(l, "┫") {
			entryLine = l
			break
		}
	}
	if entryLine == "" {
		t.Fatalf("could not find single-row entry line in:\n%s", out)
	}

	// Content between ┣ and ┫ should be wrapped with markers.
	if !strings.Contains(entryLine, "<<") || !strings.Contains(entryLine, ">>") {
		t.Errorf("SelectionStyle: content cell not styled; line: %q", entryLine)
	}
	// The ┣ and ┫ themselves must NOT be inside the markers.
	markerOpen := strings.Index(entryLine, "<<")
	markerClose := strings.Index(entryLine, ">>")
	leftRail := strings.Index(entryLine, "┣")
	rightRail := strings.LastIndex(entryLine, "┫")
	if leftRail >= markerOpen {
		t.Errorf("┣ rail must not be inside selection marker; line: %q", entryLine)
	}
	if rightRail <= markerClose {
		t.Errorf("┫ rail must not be inside selection marker; line: %q", entryLine)
	}
	// Gutter must not contain markers.
	// Gutter is the first 7 chars (before first rail char).
	gutter := entryLine[:7]
	if strings.Contains(gutter, "<<") {
		t.Errorf("gutter must not be styled; gutter: %q", gutter)
	}
}

// TestSelectionStyle_MultiRowEntry checks that SelectionStyle is applied to
// all rows of a selected multi-row entry: top border, interior content rows,
// and bottom border (T015 + standalone-border fix).
func TestSelectionStyle_MultiRowEntry(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Focus", StartMinute: pint32(480), DurationMinute: 120},
	}
	opts := cli.GridOptions{HideID: true, SelectedID: 1, Styled: true, SelectionStyle: styleMarker}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, opts)
	lines := rowsOf(out)

	for _, l := range lines {
		// Top border line (┏━┓) MUST have selection markers.
		if strings.Contains(l, "┏") && !strings.Contains(l, "<<") {
			t.Errorf("top border row must be styled; line: %q", l)
		}
		// Bottom border line (┗━┛) MUST have selection markers.
		if strings.Contains(l, "┗") && !strings.Contains(l, "<<") {
			t.Errorf("bottom border row must be styled; line: %q", l)
		}
		// Interior row (┃content┃) MUST have selection markers.
		if strings.Contains(l, "┃") && !strings.Contains(l, "<<") {
			t.Errorf("interior content row must be styled; line: %q", l)
		}
	}
}

// TestSelectionStyle_NoSelectionIDZeroUnchanged checks that SelectedID==0 output
// is byte-for-byte identical with and without SelectionStyle set (T015 golden).
func TestSelectionStyle_NoSelectionIDZeroUnchanged(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 15},
	}
	baseline := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, cli.GridOptions{HideID: true, SelectedID: 0})
	withStyle := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, cli.GridOptions{HideID: true, SelectedID: 0, SelectionStyle: styleMarker})

	if baseline != withStyle {
		t.Errorf("SelectedID==0: output should be identical with/without SelectionStyle\nbaseline: %q\nwithStyle: %q", baseline, withStyle)
	}
}

// TestSelectionStyle_RowWidthUnchanged checks that styled rows have the same
// printed width as unstyled rows (T015 width invariant).
func TestSelectionStyle_RowWidthUnchanged(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Stand", StartMinute: pint32(540), DurationMinute: 15},
	}
	noSelect := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{HideID: true})
	withSelect := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, cli.GridOptions{
		HideID: true, SelectedID: 1, Styled: true,
		SelectionStyle: func(s string) string { return "\x1b[1;44;37m" + s + "\x1b[0m" },
	})

	noLines := rowsOf(noSelect)
	withLines := rowsOf(withSelect)
	if len(noLines) != len(withLines) {
		t.Fatalf("row count differs: %d vs %d", len(noLines), len(withLines))
	}
	for i := range noLines {
		wNo := visWidth(noLines[i])
		wWith := visWidth(withLines[i])
		if wNo != wWith {
			t.Errorf("row %d: width differs (no-select=%d, with-select=%d)\n  no:   %q\n  with: %q",
				i, wNo, wWith, noLines[i], withLines[i])
		}
	}
}

// visWidth returns the visible (non-ANSI) width of a string.
func visWidth(s string) int {
	// Strip ANSI escape sequences by counting non-escape bytes.
	var w int
	inEsc := false
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			inEsc = true
			i++
			continue
		}
		if inEsc {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEsc = false
			}
			i++
			continue
		}
		// Count UTF-8 rune width.
		r, size := rune(s[i]), 1
		if s[i]&0x80 != 0 {
			if s[i]&0xe0 == 0xc0 && i+1 < len(s) {
				r, size = rune(s[i]&0x1f)<<6|rune(s[i+1]&0x3f), 2
			} else if s[i]&0xf0 == 0xe0 && i+2 < len(s) {
				r, size = rune(s[i]&0x0f)<<12|rune(s[i+1]&0x3f)<<6|rune(s[i+2]&0x3f), 3
			} else if i+3 < len(s) {
				r, size = 0, 4
			}
		}
		_ = r
		w++
		i += size
	}
	return w
}

// ---- T013: RenderUntimed tests ----

// ── Feature 035: WindowStartMin / WindowEndMin override tests (T004) ──────────

// TestRenderGrid_Override_StartOnly verifies that setting WindowStartMin fixes the
// start and disables backward expansion, while the end still expands to contain entries.
func TestRenderGrid_Override_StartOnly(t *testing.T) {
	// Entry entirely at 06:00–07:00 would normally push winStart to 06:00.
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Early", StartMinute: pint32(360), DurationMinute: 60},
	}
	startMin := 8 * 60 // fix start at 08:00
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{WindowStartMin: &startMin})
	lines := rowsOf(out)

	// Start must be 08:00 (not expanded backward to 06:00).
	wantFirst := "08:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[0] != wantFirst {
		t.Errorf("start-only override: expected 08:00 first row, got %q", lines[0])
	}
	// End stays at 17:00 (entry ends before 17:00).
	wantLast := "17:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[len(lines)-1] != wantLast {
		t.Errorf("start-only override: expected 17:00 last row, got %q", lines[len(lines)-1])
	}
}

// TestRenderGrid_Override_EndOnly verifies that setting WindowEndMin fixes the end
// and disables forward expansion, while the start still expands to contain entries.
func TestRenderGrid_Override_EndOnly(t *testing.T) {
	// Entry at 17:30 would normally push winEnd to 18:00.
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Late", StartMinute: pint32(1050), DurationMinute: 30},
	}
	end := 17 * 60 // fix end at 17:00
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(9, 0), 80, false, cli.GridOptions{WindowEndMin: &end})
	lines := rowsOf(out)

	// End must be 17:00 (not expanded forward to 18:00).
	wantLast := "17:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[len(lines)-1] != wantLast {
		t.Errorf("end-only override: expected 17:00 last row, got %q", lines[len(lines)-1])
	}
	// Start stays at 08:00 (entry at 17:30 doesn't trigger start expansion).
	wantFirst := "08:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[0] != wantFirst {
		t.Errorf("end-only override: expected 08:00 first row, got %q", lines[0])
	}
}

// TestRenderGrid_Override_BothSet verifies that when both overrides are set, the
// window is exactly as specified — no auto-expansion occurs on either side.
func TestRenderGrid_Override_BothSet(t *testing.T) {
	// Entries that would normally trigger both expansions.
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Early", StartMinute: pint32(360), DurationMinute: 60},
		{Day: "2026-05-27", Id: 2, Name: "Late", StartMinute: pint32(1050), DurationMinute: 30},
	}
	// Use fixedTime(6,0) so the now-marker is outside the 09:00–15:00 window.
	s, e := 9*60, 15*60 // fix window 09:00–15:00
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, false, cli.GridOptions{
		WindowStartMin: &s, WindowEndMin: &e,
	})
	lines := rowsOf(out)

	wantFirst := "09:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[0] != wantFirst {
		t.Errorf("both-set override: expected 09:00 first row, got %q", lines[0])
	}
	wantLast := "15:00  ├" + strings.Repeat("─", 71) + "┤"
	if lines[len(lines)-1] != wantLast {
		t.Errorf("both-set override: expected 15:00 last row, got %q", lines[len(lines)-1])
	}
	// Window is 09:00–15:00 = 6 hours × 4 rows + 1 = 25 rows.
	if len(lines) != 25 {
		t.Errorf("both-set override: expected 25 rows for 09:00–15:00, got %d", len(lines))
	}
}

// TestRenderGrid_Override_NilPreservesDefault verifies that GridOptions{} (nil overrides)
// leaves output byte-for-byte identical to the pre-feature baseline (CLI parity).
func TestRenderGrid_Override_NilPreservesDefault(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
	}
	baseline := cli.RenderGrid(entries, "2026-05-27", fixedTime(9, 0), 80, false, cli.GridOptions{})
	withNilOverrides := cli.RenderGrid(entries, "2026-05-27", fixedTime(9, 0), 80, false, cli.GridOptions{
		WindowStartMin: nil, WindowEndMin: nil,
	})
	if baseline != withNilOverrides {
		t.Error("nil overrides must produce identical output to zero-value GridOptions")
	}
}

// ── Feature 035: GridWindow FILL mode unit tests (T008) ───────────────────────

// TestGridWindow_Fill_ExtendsEnd checks that when availableRows > baseRows the
// end is extended to later hours and the start stays at 08:00.
func TestGridWindow_Fill_ExtendsEnd(t *testing.T) {
	// Empty day: baseStart=480, baseEnd=1020, baseRows=37.
	start, end := cli.GridWindow(nil, fixedTime(9, 0), "2026-05-27", 50)
	if start != 480 {
		t.Errorf("FILL start: want 480 (08:00), got %d", start)
	}
	// end = 480 + 49×15 = 480+735 = 1215
	wantEnd := 480 + 49*15
	if end != wantEnd {
		t.Errorf("FILL end: want %d, got %d", wantEnd, end)
	}
}

// TestGridWindow_Fill_EqualBaseRows checks that exactly baseRows returns the default window.
func TestGridWindow_Fill_EqualBaseRows(t *testing.T) {
	// baseRows for empty 08:00–17:00 window = 37.
	start, end := cli.GridWindow(nil, fixedTime(9, 0), "2026-05-27", 37)
	if start != 480 {
		t.Errorf("FILL equal start: want 480 (08:00), got %d", start)
	}
	if end != 1020 {
		t.Errorf("FILL equal end: want 1020 (17:00), got %d", end)
	}
}

// TestGridWindow_Fill_ClampAt24h checks that the end is clamped to 1440 (24:00).
func TestGridWindow_Fill_ClampAt24h(t *testing.T) {
	// Very tall terminal: 200 rows would push end way past 24:00.
	_, end := cli.GridWindow(nil, fixedTime(9, 0), "2026-05-27", 200)
	if end > 1440 {
		t.Errorf("FILL end must be clamped to 1440, got %d", end)
	}
	if end != 1440 {
		t.Errorf("FILL end (200 rows): expected 1440, got %d", end)
	}
}

// TestGridWindow_Fill_StartsAt08 checks that 08:00 start is preserved in FILL mode.
func TestGridWindow_Fill_StartsAt08(t *testing.T) {
	start, _ := cli.GridWindow(nil, fixedTime(9, 0), "2026-05-27", 60)
	if start != 480 {
		t.Errorf("FILL start must be 08:00 (480), got %d", start)
	}
}

// ── Feature 035: GridWindow anchor tests (T011) ────────────────────────────────

// TestGridWindow_Anchor_Today anchors to now-block on a short terminal viewing today.
func TestGridWindow_Anchor_Today(t *testing.T) {
	// baseRows=37; availableRows=20 (constrained); today at 14:00.
	today := "2026-05-27"
	now := time.Date(2026, 5, 27, 14, 7, 0, 0, time.UTC) // nowBlock=snapDown15(847)=840 (14:00)
	start, end := cli.GridWindow(nil, now, today, 20)
	if start != 840 {
		t.Errorf("ANCHOR today start: want 840 (14:00), got %d", start)
	}
	// end = 840 + 19×15 = 840+285 = 1125
	if end != 1125 {
		t.Errorf("ANCHOR today end: want 1125, got %d", end)
	}
}

// TestGridWindow_Anchor_NonToday uses TOP-TRUNCATE (baseStart) when the day is not today.
func TestGridWindow_Anchor_NonToday(t *testing.T) {
	today := "2026-05-27"
	now := time.Date(2026, 5, 27, 14, 0, 0, 0, time.UTC)
	// Render yesterday — should not anchor.
	start, _ := cli.GridWindow(nil, now, "2026-05-26", 20)
	if start != 480 {
		t.Errorf("TOP-TRUNCATE non-today: want 480 (08:00), got %d (today=%s)", start, today)
	}
}

// TestGridWindow_Anchor_NowBeforeWindow falls back to baseStart when now is before the window.
func TestGridWindow_Anchor_NowBeforeWindow(t *testing.T) {
	today := "2026-05-27"
	now := time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC) // 07:00 — before 08:00 default window
	start, _ := cli.GridWindow(nil, now, today, 20)
	if start != 480 {
		t.Errorf("early now: want 480 (08:00), got %d", start)
	}
}

// TestGridWindow_Anchor_LateNow anchors to a late block when now is well into the evening.
func TestGridWindow_Anchor_LateNow(t *testing.T) {
	today := "2026-05-27"
	now := time.Date(2026, 5, 27, 22, 0, 0, 0, time.UTC) // 22:00
	start, end := cli.GridWindow(nil, now, today, 20)
	if start != 1320 {
		t.Errorf("late ANCHOR start: want 1320 (22:00), got %d", start)
	}
	// end = 1320 + 19×15 = 1320+285 = 1605 → clamped to 1440
	if end != 1440 {
		t.Errorf("late ANCHOR end: want 1440 (clamped), got %d", end)
	}
}

// TestGridWindow_Anchor_EndClampedAt24h verifies the 24:00 clamp in ANCHOR mode.
func TestGridWindow_Anchor_EndClampedAt24h(t *testing.T) {
	today := "2026-05-27"
	now := time.Date(2026, 5, 27, 20, 0, 0, 0, time.UTC) // 20:00
	_, end := cli.GridWindow(nil, now, today, 100)
	if end > 1440 {
		t.Errorf("ANCHOR end must not exceed 1440, got %d", end)
	}
}

// ---- T013: RenderUntimed tests ----

// TestRenderUntimed_EmptyReturnsEmpty checks that an empty entry list produces no output.
func TestRenderUntimed_EmptyReturnsEmpty(t *testing.T) {
	out := cli.RenderUntimed(nil, 80, false, cli.GridOptions{})
	if out != "" {
		t.Errorf("empty entries: expected empty string, got %q", out)
	}
}

// TestRenderUntimed_EmptySliceReturnsEmpty checks that an empty slice (not nil) also returns "".
func TestRenderUntimed_EmptySliceReturnsEmpty(t *testing.T) {
	out := cli.RenderUntimed([]*planv1.PlanEntry{}, 80, false, cli.GridOptions{})
	if out != "" {
		t.Errorf("empty slice: expected empty string, got %q", out)
	}
}

// ── US1: New list row format tests (T003-T005) ────────────────────────────

// TestRenderUntimed_ListRowFormat_Simple checks that untimed entries render as
// "  ☐ name" / "  ☑ name" rows (C2/C3/C5), one per line, no [id] prefix.
func TestRenderUntimed_ListRowFormat_Simple(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Task A"},                  // incomplete
		{Id: 2, Name: "Task B", Completed: true}, // completed
	}
	out := cli.RenderUntimed(entries, 80, true, cli.GridOptions{})
	lines := rowsOf(out)

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), out)
	}

	// C3: "  ☐ name" for incomplete, "  ☑ name" for completed.
	want0 := "  ☐ Task A"
	if lines[0] != want0 {
		t.Errorf("row 0:\n  got  %q\n  want %q", lines[0], want0)
	}
	want1 := "  ☑ " + cli.DimStrike("Task B")
	if lines[1] != want1 {
		t.Errorf("row 1:\n  got  %q\n  want %q", lines[1], want1)
	}

	// C5: no [id] prefix.
	if strings.Contains(out, "[1]") || strings.Contains(out, "[2]") {
		t.Error("C5: [id] prefix must not appear in untimed rows")
	}
}

// TestRenderUntimed_ListRowFormat_SingleLine asserts each untimed entry is
// exactly one line regardless of DurationMinute (C2).
func TestRenderUntimed_ListRowFormat_SingleLine(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Quick task", DurationMinute: 15},
		{Id: 2, Name: "Long task", DurationMinute: 120},
	}
	out := cli.RenderUntimed(entries, 80, true, cli.GridOptions{})
	lines := rowsOf(out)
	if len(lines) != 2 {
		t.Errorf("C2: expected 2 lines (one per entry), got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "Quick task") {
		t.Errorf("entry 1 name missing: %q", lines[0])
	}
	if !strings.Contains(lines[1], "Long task") {
		t.Errorf("entry 2 name missing: %q", lines[1])
	}
}

// TestRenderUntimed_ListSelectionStyle verifies selection highlighting per C3.1:
//   - SelectionStyle wraps the matching row's content
//   - Non-selected rows are unstyled
//   - nil SelectionStyle falls back to applySelection
func TestRenderUntimed_ListSelectionStyle(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Alpha"},
		{Id: 2, Name: "Beta"},
	}

	// With SelectionStyle: selected row gets markers, unselected does not.
	opts := cli.GridOptions{SelectedID: 1, SelectionStyle: styleMarker}
	out := cli.RenderUntimed(entries, 80, true, opts)
	lines := rowsOf(out)
	if !strings.Contains(lines[0], "<<") {
		t.Errorf("C3.1: selected row should have SelectionStyle markers; got %q", lines[0])
	}
	if strings.Contains(lines[1], "<<") {
		t.Errorf("C3.1: non-selected row must not have markers; got %q", lines[1])
	}

	// With SelectionStyle: nil — applySelection fallback.
	optsNil := cli.GridOptions{SelectedID: 1}
	outNil := cli.RenderUntimed(entries, 80, true, optsNil)
	linesNil := rowsOf(outNil)
	if !strings.Contains(linesNil[0], "\x1b[") {
		t.Errorf("C3.1: selected row should have ANSI codes via applySelection; got %q", linesNil[0])
	}
	if strings.Contains(linesNil[1], "\x1b[") {
		t.Errorf("C3.1: non-selected row must not have ANSI codes; got %q", linesNil[1])
	}
}

// TestRenderUntimed_ListPlainMode checks plain/non-TTY output per C4:
// no checkbox glyphs, no selection styling, completed names unstyled.
func TestRenderUntimed_ListPlainMode(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Task A"},
		{Id: 2, Name: "Task B", Completed: true},
	}

	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	// C4: no checkbox glyphs.
	if strings.Contains(out, "☐") || strings.Contains(out, "☑") {
		t.Error("C4: no checkbox glyphs in plain mode")
	}
	// C4: name-only rows with leading indent.
	if !strings.Contains(lines[0], "Task A") {
		t.Errorf("C4: expected 'Task A' in row; got %q", lines[0])
	}
	// C4: no ANSI escape codes.
	if strings.Contains(out, "\x1b[") {
		t.Errorf("C4: expected no ANSI codes in plain mode; got %q", out)
	}
	// Completed name must still appear (Strike is no-op unstyled).
	if !strings.Contains(lines[1], "Task B") {
		t.Errorf("C4: completed name must appear; got %q", lines[1])
	}
}

// TestRenderUntimed_ListSelectionPlainMode checks no selection styling in plain mode.
func TestRenderUntimed_ListSelectionPlainMode(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Task A"},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{SelectedID: 1})
	if strings.Contains(out, "\x1b[") {
		t.Errorf("C4: no ANSI codes expected in plain mode even with SelectedID; got %q", out)
	}
}

// TestRenderUntimed_SingleEntry15min checks that any entry (regardless of
// DurationMinute) renders as exactly one line in the new list format.
func TestRenderUntimed_SingleEntry15min(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Quick task", DurationMinute: 15},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	lines := rowsOf(out)

	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "Quick task") {
		t.Errorf("expected name in output; got %q", lines[0])
	}
	// No box-drawing chars in list mode.
	if strings.ContainsAny(out, "┏┓┗┛┣┫┃━") {
		t.Error("list format must not contain box-drawing characters")
	}
}

// TestRenderUntimed_DurationIgnored checks that DurationMinute does not affect
// the number of lines (always 1 per entry, by contract C2).
func TestRenderUntimed_DurationIgnored(t *testing.T) {
	for _, dur := range []int32{15, 30, 60, 120} {
		entries := []*planv1.PlanEntry{
			{Id: 1, Name: "Task", DurationMinute: dur},
		}
		out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{})
		lines := rowsOf(out)
		if len(lines) != 1 {
			t.Errorf("DurationMinute=%d: expected 1 line, got %d", dur, len(lines))
		}
	}
}

// TestRenderUntimed_MultipleEntries_ListFormat checks that multiple entries stack
// vertically as single lines.
func TestRenderUntimed_MultipleEntries_ListFormat(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Task A", DurationMinute: 15},
		{Id: 2, Name: "Task B", DurationMinute: 30},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{})
	lines := rowsOf(out)

	if len(lines) != 2 {
		t.Errorf("2 entries: expected 2 lines, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "Task A") {
		t.Errorf("first entry: expected 'Task A'; got %q", lines[0])
	}
	if !strings.Contains(lines[1], "Task B") {
		t.Errorf("second entry: expected 'Task B'; got %q", lines[1])
	}
}

// TestRenderUntimed_SelectedIDHighlights checks that the selected entry is styled.
func TestRenderUntimed_SelectedIDHighlights(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Unselected", DurationMinute: 15},
		{Id: 2, Name: "Selected", DurationMinute: 15},
	}
	opts := cli.GridOptions{HideID: true, SelectedID: 2, SelectionStyle: styleMarker}
	out := cli.RenderUntimed(entries, 80, true, opts)
	lines := rowsOf(out)

	if strings.Contains(lines[0], "<<") {
		t.Errorf("unselected entry must not have selection markers; got %q", lines[0])
	}
	if !strings.Contains(lines[1], "<<") {
		t.Errorf("selected entry must have selection markers; got %q", lines[1])
	}
	// With list format, markers wrap checkbox + name, not the whole line.
	if !strings.Contains(lines[1], "<<☐ Selected>>") && !strings.Contains(lines[1], "<<☐ Selected") {
		t.Errorf("selected entry: expected markers around checkbox+name; got %q", lines[1])
	}
}

// TestRenderUntimed_ListTruncation checks that a long name is truncated to
// available width in both styled and plain modes.
func TestRenderUntimed_ListTruncation(t *testing.T) {
	longName := "AReallyReallyLongTaskNameThatExceedsAvailableWidth"
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: longName},
	}

	// Plain mode: width=20, prefixLen=2, avail=18.
	out := cli.RenderUntimed(entries, 20, false, cli.GridOptions{})
	lines := rowsOf(out)
	// Remove the leading "  " to get the visible content.
	content := strings.TrimPrefix(lines[0], "  ")
	// Content should be at most 18 runes, possibly truncated with "..."
	if utf8.RuneCountInString(content) > 18+3 {
		t.Errorf("plain truncation: content is %d runes, expected ≤21; got %q", utf8.RuneCountInString(content), content)
	}

	// Styled mode: width=20, prefixLen=4, avail=16.
	outSty := cli.RenderUntimed(entries, 20, true, cli.GridOptions{})
	styLines := rowsOf(outSty)
	// Remove the "  ☐ " prefix.
	if utf8.RuneCountInString(styLines[0]) > 20 {
		t.Errorf("styled truncation: line is %d runes, expected ≤20; got %q", utf8.RuneCountInString(styLines[0]), styLines[0])
	}
}

// TestRenderUntimed_ListWideRuneTruncation checks that names containing wide
// (double-column) runes are truncated by display width, not rune count, so the
// row's visible width never exceeds the available width. A rune-count-based
// truncation would let CJK/emoji names overflow onto a second terminal line.
// Uses lipgloss.Width (rather than the test package's rune-counting visWidth
// helper) because it correctly accounts for double-width runes.
func TestRenderUntimed_ListWideRuneTruncation(t *testing.T) {
	wideName := strings.Repeat("宽", 30) // each rune renders at 2 columns
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: wideName},
	}

	// Plain mode: width=20, prefixLen=2, avail=18 columns.
	out := cli.RenderUntimed(entries, 20, false, cli.GridOptions{})
	lines := rowsOf(out)
	content := strings.TrimPrefix(lines[0], "  ")
	if w := lipgloss.Width(content); w > 18 {
		t.Errorf("plain wide-rune truncation: content display width = %d, expected ≤18; got %q", w, content)
	}

	// Styled mode: width=20, prefixLen=4, avail=16 columns.
	outSty := cli.RenderUntimed(entries, 20, true, cli.GridOptions{})
	styLines := rowsOf(outSty)
	if w := lipgloss.Width(styLines[0]); w > 20 {
		t.Errorf("styled wide-rune truncation: line display width = %d, expected ≤20; got %q", w, styLines[0])
	}
}

// TestRenderUntimed_ListCompletedStyling checks that completed entries get ☑ + struck name.
func TestRenderUntimed_ListCompletedStyling(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", Completed: true},
	}
	out := cli.RenderUntimed(entries, 80, true, cli.GridOptions{})
	lines := rowsOf(out)
	if !strings.Contains(lines[0], "☑") {
		t.Errorf("completed: expected ☑ checkbox; got %q", lines[0])
	}
	if !strings.Contains(lines[0], cli.DimStrike("Done task")) {
		t.Errorf("completed: expected dim+strikethrough name; got %q", lines[0])
	}
}

// TestRenderUntimed_ListPlainCompleted checks that completed entries in plain mode
// show the name without glyphs or ANSI codes.
func TestRenderUntimed_ListPlainCompleted(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Done task", Completed: true},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{})
	if strings.Contains(out, "☐") || strings.Contains(out, "☑") {
		t.Error("plain mode: no checkbox glyphs expected")
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("plain mode: no ANSI codes expected; got %q", out)
	}
	if !strings.Contains(out, "Done task") {
		t.Errorf("plain mode: name must appear; got %q", out)
	}
}

// TestSelectionStyle_StandaloneBorders_GapBeforeAfter checks that when an entry
// has a gap before and after it (standalone top and bottom borders), those border
// lines carry SelectionStyle markers and the hour gutter is left unstyled.
func TestSelectionStyle_StandaloneBorders_GapBeforeAfter(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Early", StartMinute: pint32(480), DurationMinute: 60}, // 08:00–09:00
		{Id: 2, Name: "Later", StartMinute: pint32(660), DurationMinute: 60}, // 11:00–12:00
	}
	opts := cli.GridOptions{HideID: true, SelectedID: 2, Styled: true, SelectionStyle: styleMarker}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(6, 0), 80, true, opts)
	lines := rowsOf(out)

	for _, l := range lines {
		if strings.Contains(l, "┏") || strings.Contains(l, "┗") {
			// Only the selected entry's borders should carry the marker.
			if strings.Contains(l, "11:00") || strings.Contains(l, "12:00") ||
				(!strings.Contains(l, "08:00") && !strings.Contains(l, "09:00")) {
				// This is a border line for the selected entry (id 2) or an empty row.
				// Selected entry borders must have the marker.
				if strings.Contains(l, "<<") {
					// Gutter (first 7 chars) must not contain the marker.
					if len(l) >= 7 && strings.Contains(l[:7], "<<") {
						t.Errorf("hour gutter must not contain SelectionStyle marker; line: %q", l)
					}
				}
			}
		}
		// The selected entry's standalone top/bottom borders must be styled.
		if strings.Contains(l, "┏") && strings.Contains(l, "11") {
			if !strings.Contains(l, "<<") {
				t.Errorf("selected entry standalone top border must be styled; line: %q", l)
			}
		}
		if strings.Contains(l, "┗") && strings.Contains(l, "12") {
			if !strings.Contains(l, "<<") {
				t.Errorf("selected entry standalone bottom border must be styled; line: %q", l)
			}
		}
	}
}

// TestRenderUntimed_ListLineWidth checks that each line's visible width does not
// exceed the given width in both styled and plain modes.
func TestRenderUntimed_ListLineWidth(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Task", DurationMinute: 30},
	}
	// Plain mode: no ANSI, width should be exactly width.
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{})
	lines := rowsOf(out)
	for i, l := range lines {
		if w := visWidth(l); w > 80 {
			t.Errorf("RenderUntimed line %d (plain): width = %d, exceeds 80; line: %q", i, w, l)
		}
	}

	// Styled mode: ANSI codes make raw string longer, but visible width ≤ width.
	outSty := cli.RenderUntimed(entries, 80, true, cli.GridOptions{})
	styLines := rowsOf(outSty)
	for i, l := range styLines {
		if w := visWidth(l); w > 80 {
			t.Errorf("RenderUntimed line %d (styled): visible width = %d, exceeds 80; line: %q", i, w, l)
		}
	}
}

// ---- T015: US4 separator tests ----

// TestRenderUntimedSeparator_PresentWhenUntimedExists checks that RenderUntimedSeparator
// returns a non-empty separator line when untimed entries exist.
func TestRenderUntimedSeparator_PresentWhenUntimedExists(t *testing.T) {
	sep := cli.RenderUntimedSeparator([]int{1}, 80)
	if sep == "" {
		t.Error("separator: expected non-empty string when untimed entries exist")
	}
	// Must contain box-drawing divider chars.
	if !strings.Contains(sep, "─") {
		t.Errorf("separator: expected horizontal line chars; got %q", sep)
	}
}

// TestRenderUntimedSeparator_AbsentWhenEmpty checks that no separator is returned
// when no untimed entries exist (empty slice).
func TestRenderUntimedSeparator_AbsentWhenEmpty(t *testing.T) {
	sep := cli.RenderUntimedSeparator(nil, 80)
	if sep != "" {
		t.Errorf("separator with no untimed entries: expected empty string, got %q", sep)
	}
}

// TestRenderUntimed_LabelHasNoDurationSuffix verifies that untimed entry boxes
// show only the entry name — no "(30min)" or similar parenthetical duration.
func TestRenderUntimed_LabelHasNoDurationSuffix(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Build fence", DurationMinute: 30},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	if strings.Contains(out, "min)") {
		t.Errorf("RenderUntimed: box label should not contain a duration suffix like '(30min)'; got:\n%s", out)
	}
	if !strings.Contains(out, "Build fence") {
		t.Errorf("RenderUntimed: box label should contain the entry name 'Build fence'; got:\n%s", out)
	}
}

// TestRenderUntimedSeparator_WidthMatchesGrid checks the separator is the same
// visual width as a grid row.
func TestRenderUntimedSeparator_WidthMatchesGrid(t *testing.T) {
	sep := cli.RenderUntimedSeparator([]int{1}, 80)
	gridOut := cli.RenderGrid(nil, "2026-05-27", fixedTime(9, 0), 80, false, cli.GridOptions{})
	gridLines := rowsOf(gridOut)
	wantWidth := visWidth(gridLines[0])
	// Separator has a trailing newline; check the first line.
	sepLines := rowsOf(sep)
	if len(sepLines) == 0 {
		t.Fatal("separator produced no lines")
	}
	if w := visWidth(sepLines[0]); w != wantWidth {
		t.Errorf("separator width = %d, want %d (grid width); line: %q", w, wantWidth, sepLines[0])
	}
}

// ── T003: Backward-compatibility guard ────────────────────────────────────────

// TestRenderGrid_BackwardCompat_PreviewZero verifies that when PreviewID == 0
// and PreviewConflictSlots is empty/nil, RenderGrid output is byte-for-byte
// identical to a call with a zero-value GridOptions (contract §5).
func TestRenderGrid_BackwardCompat_PreviewZero(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Standup", StartMinute: pint32(540), DurationMinute: 30},
		{Day: "2026-05-27", Id: 2, Name: "Lunch", StartMinute: pint32(720), DurationMinute: 60},
	}
	day := "2026-05-27"
	now := fixedTime(9, 0)

	baseline := cli.RenderGrid(entries, day, now, 80, false, cli.GridOptions{})
	withPreviewFields := cli.RenderGrid(entries, day, now, 80, false, cli.GridOptions{
		PreviewID:            0,
		PreviewConflictSlots: nil,
	})

	if baseline != withPreviewFields {
		t.Errorf("RenderGrid: output changed when PreviewID==0 and PreviewConflictSlots is nil\nbaseline:\n%s\nwith zero preview fields:\n%s", baseline, withPreviewFields)
	}
}

// TestRenderUntimed_BackwardCompat_PreviewZero verifies RenderUntimed is unchanged
// with zero preview fields.
func TestRenderUntimed_BackwardCompat_PreviewZero(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Review PR", DurationMinute: 30},
	}
	baseline := cli.RenderUntimed(entries, 80, false, cli.GridOptions{})
	withPreviewFields := cli.RenderUntimed(entries, 80, false, cli.GridOptions{
		PreviewID:            0,
		PreviewConflictSlots: nil,
	})

	if baseline != withPreviewFields {
		t.Errorf("RenderUntimed: output changed when PreviewID==0 and PreviewConflictSlots is nil")
	}
}

// ── T005: Dashed-rune rendering tests ────────────────────────────────────────

// TestRenderGrid_PreviewEntry_DashedRunes verifies that an entry with Id == PreviewID
// renders with heavy double-dash runes (╍/╏) while saved entries keep solid runes (━/┃).
func TestRenderGrid_PreviewEntry_DashedRunes(t *testing.T) {
	const previewID int32 = -1
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Saved", StartMinute: pint32(540), DurationMinute: 60},
		{Day: "2026-05-27", Id: previewID, Name: "Preview", StartMinute: pint32(630), DurationMinute: 60},
	}
	opts := cli.GridOptions{
		HideID:    true,
		PreviewID: previewID,
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, false, opts)

	// Preview entry must use heavy double-dash horizontal rune ╍.
	if !strings.Contains(out, "╍") {
		t.Errorf("preview entry: expected heavy double-dash horizontal rune ╍ in output:\n%s", out)
	}
	// Preview entry must use heavy double-dash vertical rune ╏.
	if !strings.Contains(out, "╏") {
		t.Errorf("preview entry: expected heavy double-dash vertical rune ╏ in output:\n%s", out)
	}
	// Saved entry must still use solid horizontal rune ━.
	if !strings.Contains(out, "━") {
		t.Errorf("saved entry: expected solid horizontal rune ━ in output:\n%s", out)
	}
	// Saved entry must still use solid vertical rune ┃.
	if !strings.Contains(out, "┃") {
		t.Errorf("saved entry: expected solid vertical rune ┃ in output:\n%s", out)
	}
}

// TestRenderGrid_PreviewEntry_SingleRow verifies single-row preview uses heavy double-dash rune.
func TestRenderGrid_PreviewEntry_SingleRow(t *testing.T) {
	const previewID int32 = -1
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: previewID, Name: "Micro", StartMinute: pint32(540), DurationMinute: 15},
	}
	opts := cli.GridOptions{
		HideID:    true,
		PreviewID: previewID,
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, false, opts)

	if !strings.Contains(out, "╍") {
		t.Errorf("single-row preview: expected heavy double-dash rune ╍:\n%s", out)
	}
}

// ── T012: Conflict-rendering tests (US2) ─────────────────────────────────────

// TestRenderGrid_ConflictStyle_StyledMode verifies that preview rows whose slot is in
// PreviewConflictSlots are wrapped with ConflictStyle in styled mode (contract §1.4).
func TestRenderGrid_ConflictStyle_StyledMode(t *testing.T) {
	const previewID int32 = -1
	// Preview: 10:00–11:00 (600–660). Existing entry: 10:30–11:30 (630–690).
	// Overlapping slots: 630 (10:30) and 645 (10:45).
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Saved", StartMinute: pint32(630), DurationMinute: 60},
		{Day: "2026-05-27", Id: previewID, Name: "Preview", StartMinute: pint32(600), DurationMinute: 60},
	}
	opts := cli.GridOptions{
		HideID:               true,
		PreviewID:            previewID,
		PreviewConflictSlots: map[int]bool{630: true, 645: true},
		ConflictStyle:        styleMarker,
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, true, opts)

	// Some rows must have the conflict markers.
	if !strings.Contains(out, "<<") || !strings.Contains(out, ">>") {
		t.Errorf("conflict styled mode: expected ConflictStyle markers in output:\n%s", out)
	}
}

// TestRenderGrid_ConflictStyle_PlainFallback verifies that when ConflictStyle is nil,
// conflicting preview rows show a '!' gutter marker (contract §4).
func TestRenderGrid_ConflictStyle_PlainFallback(t *testing.T) {
	const previewID int32 = -1
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Saved", StartMinute: pint32(630), DurationMinute: 60},
		{Day: "2026-05-27", Id: previewID, Name: "Preview", StartMinute: pint32(600), DurationMinute: 60},
	}
	opts := cli.GridOptions{
		HideID:               true,
		PreviewID:            previewID,
		PreviewConflictSlots: map[int]bool{630: true, 645: true},
		ConflictStyle:        nil, // plain-mode fallback
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, false, opts)

	lines := rowsOf(out)
	// winStart=480; slot 630 → row (630-480)/15=10; slot 645 → row 11.
	// Gutter char is at index 6 in each line.
	foundBang := false
	for _, i := range []int{10, 11} {
		if i < len(lines) && strings.HasPrefix(lines[i][6:], "!") {
			foundBang = true
		}
	}
	if !foundBang {
		t.Errorf("conflict plain fallback: expected '!' gutter marker on conflicting preview row:\n%s", out)
	}
}

// TestRenderGrid_ConflictStyle_NonConflictingPreviewUnchanged verifies that preview rows
// NOT in PreviewConflictSlots do not receive the conflict marker/style (contract §1.3).
func TestRenderGrid_ConflictStyle_NonConflictingPreviewUnchanged(t *testing.T) {
	const previewID int32 = -1
	// Preview at 10:00–11:00; conflict only at slot 630 (10:30).
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: previewID, Name: "Preview", StartMinute: pint32(600), DurationMinute: 60},
	}
	opts := cli.GridOptions{
		HideID:               true,
		PreviewID:            previewID,
		PreviewConflictSlots: map[int]bool{630: true}, // only 10:30 conflicts
		ConflictStyle:        nil,
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, false, opts)

	lines := rowsOf(out)
	// winStart=480; slot 600 → row 8; slot 615 → row 9; slot 630 → row 10; slot 645 → row 11.
	// Rows 8, 9, 11 should NOT have '!' in the gutter (only row 10 should).
	for _, i := range []int{8, 9} {
		if i < len(lines) && len(lines[i]) > 6 && lines[i][6] == '!' {
			t.Errorf("non-conflicting row %d: unexpected '!' gutter marker; line: %q", i, lines[i])
		}
	}
}

// TestRenderGrid_ConflictStyle_PreviewTopAtInteriorRow tests the case where the
// preview starts at the same row as another entry's interior (their time windows
// overlap). The preview's top border at that row must be styled with ConflictStyle
// (not swallowed by the existing entry's interior case).
func TestRenderGrid_ConflictStyle_PreviewTopAtInteriorRow(t *testing.T) {
	const previewID int32 = -1
	// Coffee: 08:45–09:15 (525–555). Preview: 09:00–09:45 (540–585).
	// Coffee's interior is at row 4 (09:00). Preview's top is also at row 4.
	// Conflict slot: 540 (09:00).
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Coffee", StartMinute: pint32(525), DurationMinute: 30},
		{Day: "2026-05-27", Id: previewID, Name: "Collect eggs", StartMinute: pint32(540), DurationMinute: 45},
	}
	opts := cli.GridOptions{
		HideID:               true,
		PreviewID:            previewID,
		PreviewConflictSlots: map[int]bool{540: true},
		ConflictStyle:        styleMarker,
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, true, opts)
	lines := rowsOf(out)

	// winStart=480; row 4 = (540-480)/15 = 4. Preview's top must appear there with conflict marker.
	if len(lines) <= 4 {
		t.Fatalf("not enough rows: %d", len(lines))
	}
	row4 := lines[4]
	// Row 4 must have the conflict style markers (dashed preview top border styled red).
	if !strings.Contains(row4, "<<") || !strings.Contains(row4, ">>") {
		t.Errorf("row 4 (09:00, preview topLine = conflict slot): expected ConflictStyle markers; got: %q", row4)
	}
	// It must use dashed runes (preview top border) not solid (Coffee interior).
	if !strings.Contains(row4, "╍") {
		t.Errorf("row 4: expected heavy double-dash rune ╍ for preview top border; got: %q", row4)
	}
}

// TestRenderGrid_ConflictStyle_PreviewTopAtInteriorRow_PlainFallback tests that
// the plain-mode '!' gutter marker appears when ConflictStyle is nil and the
// preview's top row coincides with another entry's interior row.
func TestRenderGrid_ConflictStyle_PreviewTopAtInteriorRow_PlainFallback(t *testing.T) {
	const previewID int32 = -1
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Coffee", StartMinute: pint32(525), DurationMinute: 30},
		{Day: "2026-05-27", Id: previewID, Name: "Collect eggs", StartMinute: pint32(540), DurationMinute: 45},
	}
	opts := cli.GridOptions{
		HideID:               true,
		PreviewID:            previewID,
		PreviewConflictSlots: map[int]bool{540: true},
		ConflictStyle:        nil, // plain fallback
	}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, false, opts)
	lines := rowsOf(out)

	// winStart=480; row 4 = 09:00. Gutter char (index 6) should be '!'.
	if len(lines) <= 4 {
		t.Fatalf("not enough rows: %d", len(lines))
	}
	row4 := lines[4]
	if len(row4) < 7 || row4[6] != '!' {
		t.Errorf("row 4 plain fallback: expected '!' at gutter index 6; got: %q", row4)
	}
}

// ── T016: Gap-visibility test (US3) ──────────────────────────────────────────

// TestRenderGrid_GapVisibility_PreviewWithGap verifies that a preview starting at 10:30
// after an entry ending at 10:00 leaves visible empty grid rows for the 10:00–10:30 gap.
func TestRenderGrid_GapVisibility_PreviewWithGap(t *testing.T) {
	const previewID int32 = -1
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Morning", StartMinute: pint32(480), DurationMinute: 120},        // 08:00–10:00
		{Day: "2026-05-27", Id: previewID, Name: "Preview", StartMinute: pint32(630), DurationMinute: 30}, // 10:30–11:00
	}
	opts := cli.GridOptions{HideID: true, PreviewID: previewID}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, false, opts)
	lines := rowsOf(out)

	// winStart=480; 10:00=row 8 (bottom of morning), 10:15=row 9 (gap), 10:30=row 10 (top of preview).
	// Row 9 (10:15) must be an empty grid row, not an entry row.
	if len(lines) <= 9 {
		t.Fatalf("not enough rows in output: %d", len(lines))
	}
	gapRow := lines[9]
	if strings.ContainsAny(gapRow, "┏┓┗┛┣┫┃╍╏") {
		t.Errorf("gap row (10:15) should be an empty grid row, but has entry runes: %q", gapRow)
	}
	// Row 10 (10:30) must be the top of the preview box.
	previewRow := lines[10]
	if !strings.Contains(previewRow, "╍") && !strings.Contains(previewRow, "┏") {
		t.Errorf("preview top row (10:30): expected preview or box border rune: %q", previewRow)
	}
}

// ── NextGapFloor unit tests ────────────────────────────────────────────────

// TestNextGapFloor_SingleEntry verifies that a single entry starting at/after fromMin
// returns its end as the floor of the next gap.
func TestNextGapFloor_SingleEntry(t *testing.T) {
	// entry 540–600 (09:00–10:00); fromMin=480 → returns (600, true).
	timed := []*planv1.PlanEntry{makeTimedEntry(1, 540, 60)}
	floor, ok := cli.NextGapFloor(timed, 480, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if floor != 600 {
		t.Errorf("expected floor=600, got %d", floor)
	}
}

// TestNextGapFloor_TwoEntries verifies that only the FIRST entry whose start >= fromMin
// contributes (earliest one).
func TestNextGapFloor_TwoEntries(t *testing.T) {
	// entries 540–600 and 660–720; fromMin=480 → returns (600, true).
	timed := []*planv1.PlanEntry{
		makeTimedEntry(1, 540, 60),
		makeTimedEntry(2, 660, 60),
	}
	floor, ok := cli.NextGapFloor(timed, 480, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if floor != 600 {
		t.Errorf("expected floor=600 (first entry end), got %d", floor)
	}
}

// TestNextGapFloor_EntryBeforeFromMin verifies ok=false when the only entry ends before fromMin.
func TestNextGapFloor_EntryBeforeFromMin(t *testing.T) {
	// entry 420–480 (07:00–08:00); fromMin=480 → no entry starts at/after 480.
	timed := []*planv1.PlanEntry{makeTimedEntry(1, 420, 60)}
	_, ok := cli.NextGapFloor(timed, 480, 0)
	if ok {
		t.Error("expected ok=false when entry ends before fromMin")
	}
}

// TestNextGapFloor_NoEntries verifies ok=false with no entries.
func TestNextGapFloor_NoEntries(t *testing.T) {
	_, ok := cli.NextGapFloor(nil, 480, 0)
	if ok {
		t.Error("expected ok=false with no entries")
	}
}

// TestNextGapFloor_ExcludeID verifies that the excludeID entry is ignored.
func TestNextGapFloor_ExcludeID(t *testing.T) {
	// Excluded entry at 540–600; another at 660–720; fromMin=480.
	// Excluding id=1 means the first qualifying entry is id=2 (660–720) → returns (720, true).
	timed := []*planv1.PlanEntry{
		makeTimedEntry(1, 540, 60),
		makeTimedEntry(2, 660, 60),
	}
	floor, ok := cli.NextGapFloor(timed, 480, 1)
	if !ok {
		t.Fatal("expected ok=true (second entry is still qualifying)")
	}
	if floor != 720 {
		t.Errorf("expected floor=720 (second entry end, first excluded), got %d", floor)
	}
}

// TestNextGapFloor_ExcludeID_NoOtherEntries verifies ok=false when the only entry
// at/after fromMin is the excluded one.
func TestNextGapFloor_ExcludeID_NoOtherEntries(t *testing.T) {
	// Only entry at 540–600, and it is excluded.
	timed := []*planv1.PlanEntry{makeTimedEntry(1, 540, 60)}
	_, ok := cli.NextGapFloor(timed, 480, 1)
	if ok {
		t.Error("expected ok=false when only qualifying entry is excluded")
	}
}

// ── AutoScheduleSlot unit tests ────────────────────────────────────────────

// TestAutoScheduleSlot_BoundaryAligned_OffBoundaryObstacle verifies that when an
// obstacle ends off-boundary (e.g. 08:00–09:07 = 480–547), the returned start is
// rounded UP to the next 15-min boundary (09:15 = 555). Boundary-aligned cases
// (480, 540, 600) must still pass unchanged.
func TestAutoScheduleSlot_BoundaryAligned_OffBoundaryObstacle(t *testing.T) {
	// Obstacle ends at 547 (09:07); 30-min task must land at 555 (09:15), not 547.
	timed := []*planv1.PlanEntry{makeTimedEntry(1, 480, 67)} // 08:00–09:07
	start, ok := cli.AutoScheduleSlot(timed, 30, 480, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if start != 555 {
		t.Errorf("off-boundary obstacle: expected start=555 (09:15), got %d", start)
	}
	// start must be a 15-minute boundary.
	if start%15 != 0 {
		t.Errorf("start must be a 15-minute boundary, got %d", start)
	}
}

// TestAutoScheduleSlot_BoundaryAligned_OnBoundaryObstacle verifies that when an
// obstacle ends exactly on a boundary (08:00–09:00 = 480–540), the start stays at 540.
func TestAutoScheduleSlot_BoundaryAligned_OnBoundaryObstacle(t *testing.T) {
	timed := []*planv1.PlanEntry{makeTimedEntry(1, 480, 60)} // 08:00–09:00
	start, ok := cli.AutoScheduleSlot(timed, 30, 480, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if start != 540 {
		t.Errorf("on-boundary obstacle: expected start=540, got %d", start)
	}
}

// TestAutoScheduleSlot_BoundaryAligned_FloorAlreadyBoundary verifies that an
// already-boundary floor (480, 540, 600) on an empty day returns that floor unchanged.
func TestAutoScheduleSlot_BoundaryAligned_FloorAlreadyBoundary(t *testing.T) {
	for _, floor := range []int{480, 540, 600} {
		start, ok := cli.AutoScheduleSlot(nil, 30, floor, 0)
		if !ok {
			t.Fatalf("floor=%d: expected ok=true", floor)
		}
		if start != floor {
			t.Errorf("floor=%d: expected start=%d, got %d", floor, floor, start)
		}
	}
}

func makeTimedEntry(id int32, startMin, durMin int) *planv1.PlanEntry {
	sm := int32(startMin)
	return &planv1.PlanEntry{Id: id, StartMinute: &sm, DurationMinute: int32(durMin)}
}

func TestAutoScheduleSlot_EmptyDay(t *testing.T) {
	start, ok := cli.AutoScheduleSlot(nil, 30, 480, 0)
	if !ok {
		t.Fatal("expected ok=true on empty day")
	}
	if start != 480 {
		t.Errorf("empty day: expected start=480, got %d", start)
	}
}

func TestAutoScheduleSlot_GapAfterBlock(t *testing.T) {
	// 08:00–09:00 blocked; 30-min task should land at 09:00 (540).
	timed := []*planv1.PlanEntry{makeTimedEntry(1, 480, 60)}
	start, ok := cli.AutoScheduleSlot(timed, 30, 480, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if start != 540 {
		t.Errorf("gap after block: expected start=540, got %d", start)
	}
}

func TestAutoScheduleSlot_TooSmallGapSkipped(t *testing.T) {
	// 08:00–09:00 and 09:15–10:00 blocked; 30-min task skips 15-min gap, lands at 600.
	timed := []*planv1.PlanEntry{
		makeTimedEntry(1, 480, 60),
		makeTimedEntry(2, 555, 45),
	}
	start, ok := cli.AutoScheduleSlot(timed, 30, 480, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if start != 600 {
		t.Errorf("too-small gap: expected start=600, got %d", start)
	}
}

func TestAutoScheduleSlot_FloorRespected(t *testing.T) {
	// floor=600 (10:00) on an empty day; should not start before 600.
	start, ok := cli.AutoScheduleSlot(nil, 30, 600, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if start != 600 {
		t.Errorf("floor: expected start=600, got %d", start)
	}
}

func TestAutoScheduleSlot_ZeroDurationUsesThirtyMin(t *testing.T) {
	// 08:00–09:00 blocked; zero-duration task needs 30-min block, lands at 540.
	timed := []*planv1.PlanEntry{makeTimedEntry(1, 480, 60)}
	start, ok := cli.AutoScheduleSlot(timed, 0, 480, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if start != 540 {
		t.Errorf("zero-duration: expected start=540, got %d", start)
	}
}

func TestAutoScheduleSlot_BlockMustEndBy1440(t *testing.T) {
	// floor=1410 (23:30): only 30 min remain before midnight, not enough for 60 min.
	_, ok := cli.AutoScheduleSlot(nil, 60, 1410, 0)
	if ok {
		t.Error("expected ok=false when 60-min block cannot fit before midnight")
	}
}

func TestAutoScheduleSlot_BlockMustEndBy1440_Fits(t *testing.T) {
	// 23:00 (1380) + 30 min = 1410 ≤ 1440: should fit.
	start, ok := cli.AutoScheduleSlot(nil, 30, 1380, 0)
	if !ok {
		t.Fatal("expected ok=true at 23:00")
	}
	if start != 1380 {
		t.Errorf("end-of-day: expected start=1380, got %d", start)
	}
}

func TestAutoScheduleSlot_ExcludeID(t *testing.T) {
	// Entry 1 at 08:00–09:00; entry 2 at 09:00–10:00.
	// Asking to schedule entry 1 (excludeID=1) should ignore its own block,
	// placing it at 08:00 even though entry 2 starts at 09:00.
	timed := []*planv1.PlanEntry{
		makeTimedEntry(1, 480, 60),
		makeTimedEntry(2, 540, 60),
	}
	start, ok := cli.AutoScheduleSlot(timed, 60, 480, 1)
	if !ok {
		t.Fatal("expected ok=true with excludeID")
	}
	if start != 480 {
		t.Errorf("excludeID: expected start=480 (self excluded), got %d", start)
	}
}

func TestAutoScheduleSlot_OverlappingObstacles(t *testing.T) {
	// Two overlapping entries: 08:00–09:30 and 09:00–10:00 → union 08:00–10:00.
	// 30-min task should land at 600 (10:00).
	timed := []*planv1.PlanEntry{
		makeTimedEntry(1, 480, 90),
		makeTimedEntry(2, 540, 60),
	}
	start, ok := cli.AutoScheduleSlot(timed, 30, 480, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if start != 600 {
		t.Errorf("overlapping obstacles: expected start=600, got %d", start)
	}
}

func TestAutoScheduleSlot_NoFit(t *testing.T) {
	// Day is full from floor to midnight.
	timed := []*planv1.PlanEntry{makeTimedEntry(1, 480, 960)} // 08:00–24:00
	_, ok := cli.AutoScheduleSlot(timed, 30, 480, 0)
	if ok {
		t.Error("expected ok=false on a full day")
	}
}

// TestRenderGrid_GapVisibility_PreviewFlush verifies that a preview starting at 10:00
// is flush with an entry ending at 10:00 (shared border, no empty row between them).
func TestRenderGrid_GapVisibility_PreviewFlush(t *testing.T) {
	const previewID int32 = -1
	entries := []*planv1.PlanEntry{
		{Day: "2026-05-27", Id: 1, Name: "Morning", StartMinute: pint32(480), DurationMinute: 120},        // 08:00–10:00
		{Day: "2026-05-27", Id: previewID, Name: "Preview", StartMinute: pint32(600), DurationMinute: 30}, // 10:00–10:30
	}
	opts := cli.GridOptions{HideID: true, PreviewID: previewID}
	out := cli.RenderGrid(entries, "2026-05-27", fixedTime(7, 0), 80, false, opts)
	lines := rowsOf(out)

	// winStart=480; row 8 = 10:00. With flush entries the shared-border row at 10:00
	// must contain ┣ or ┗ (entry boundary, not empty grid).
	if len(lines) <= 8 {
		t.Fatalf("not enough rows in output: %d", len(lines))
	}
	sharedRow := lines[8]
	if !strings.ContainsAny(sharedRow, "┣┗╍┏├") {
		t.Errorf("flush preview: expected entry boundary at row 8 (10:00), got: %q", sharedRow)
	}
}
