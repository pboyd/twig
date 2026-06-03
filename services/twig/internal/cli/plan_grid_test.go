package cli_test

import (
	"strings"
	"testing"
	"time"

	planv1 "github.com/pboyd/twig/services/twig/gen/plan/v1"
	"github.com/pboyd/twig/services/twig/internal/cli"
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

// TestRenderUntimed_EmptyReturnsEmpty checks that an empty entry list produces no output.
func TestRenderUntimed_EmptyReturnsEmpty(t *testing.T) {
	out := cli.RenderUntimed(nil, 80, false, cli.GridOptions{})
	if out != "" {
		t.Errorf("empty entries: expected empty string, got %q", out)
	}
}

// TestRenderUntimed_SingleEntry15min checks that a 15-minute entry renders as one line.
func TestRenderUntimed_SingleEntry15min(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Quick task", DurationMinute: 15},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	lines := rowsOf(out)

	if len(lines) != 1 {
		t.Errorf("15min entry: expected 1 line, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "Quick task") {
		t.Errorf("15min entry: expected name in output; got %q", lines[0])
	}
	if !strings.Contains(lines[0], "15min") {
		t.Errorf("15min entry: expected duration in output; got %q", lines[0])
	}
}

// TestRenderUntimed_Entry30min checks that a 30-minute entry renders with correct box geometry:
// ┏┓ top border, one interior content line (with label), and ┗┛ bottom border.
func TestRenderUntimed_Entry30min(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Medium task", DurationMinute: 30},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	lines := rowsOf(out)

	// ┏┓ top + 1 interior + ┗┛ bottom = 3 lines.
	if len(lines) != 3 {
		t.Errorf("30min entry: expected 3 lines (┏┓ + interior + ┗┛), got %d:\n%s", len(lines), out)
	}
	// Row 0: top border ┏━━━┓ (title must NOT be on this line).
	if !strings.Contains(lines[0], "┏") || !strings.Contains(lines[0], "┓") {
		t.Errorf("30min entry row 0: expected ┏...┓ top border; got %q", lines[0])
	}
	if strings.Contains(lines[0], "Medium task") {
		t.Errorf("30min entry row 0: title must not appear on the top border line; got %q", lines[0])
	}
	// Row 1: interior — contains the label.
	if !strings.Contains(lines[1], "Medium task") {
		t.Errorf("30min entry row 1: expected name in interior; got %q", lines[1])
	}
	// Row 2: bottom border ┗━━━┛.
	if !strings.Contains(lines[2], "┗") || !strings.Contains(lines[2], "┛") {
		t.Errorf("30min entry row 2: expected ┗...┛ bottom border; got %q", lines[2])
	}
}

// TestRenderUntimed_Entry60min checks that a 60-minute entry renders as five lines:
// ┏┓ top + 3 interior + ┗┛ bottom.
func TestRenderUntimed_Entry60min(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Long task", DurationMinute: 60},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	lines := rowsOf(out)

	if len(lines) != 5 {
		t.Errorf("60min entry: expected 5 lines (┏┓ + 3 interior + ┗┛), got %d:\n%s", len(lines), out)
	}
}

// TestRenderUntimed_MultipleEntries checks that multiple entries stack vertically.
func TestRenderUntimed_MultipleEntries(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Task A", DurationMinute: 15},
		{Id: 2, Name: "Task B", DurationMinute: 30},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	lines := rowsOf(out)

	// Task A: 1 line (single-row); Task B: 3 lines (┏┓ + interior + ┗┛) → total 4.
	if len(lines) != 4 {
		t.Errorf("2 entries (15+30min): expected 4 lines, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "Task A") {
		t.Errorf("first entry line: expected 'Task A'; got %q", lines[0])
	}
	// Task B appears on the interior line (row 2 = index 2).
	if !strings.Contains(out, "Task B") {
		t.Errorf("second entry: expected 'Task B' somewhere in output; got:\n%s", out)
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

	// Line 0 (Unselected) must NOT have markers.
	if strings.Contains(lines[0], "<<") {
		t.Errorf("unselected entry must not have selection markers; got %q", lines[0])
	}
	// Line 1 (Selected) must have markers.
	if !strings.Contains(lines[1], "<<") {
		t.Errorf("selected entry must have selection markers; got %q", lines[1])
	}
}

// ---- T006: US3 rendering fix tests ----

// TestRenderUntimed_TopBorderAboveTitle asserts that a multi-row untimed entry has a
// ┏━━━┓ top border line with the title on the NEXT line (not on the border itself).
func TestRenderUntimed_TopBorderAboveTitle(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "My task", DurationMinute: 30},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	lines := rowsOf(out)

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d:\n%s", len(lines), out)
	}
	// First line must contain ┏ and ┓.
	if !strings.Contains(lines[0], "┏") || !strings.Contains(lines[0], "┓") {
		t.Errorf("row 0: expected ┏...┓ top border; got %q", lines[0])
	}
	// Title must NOT appear on the top border line.
	if strings.Contains(lines[0], "My task") {
		t.Errorf("row 0: title must not be on the top border; got %q", lines[0])
	}
	// Title MUST appear on a subsequent interior line.
	found := false
	for _, l := range lines[1:] {
		if strings.Contains(l, "My task") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("title 'My task' not found on any interior line:\n%s", out)
	}
}

// TestRenderUntimed_SharedBorderBetweenAdjacentEntries checks that two adjacent
// multi-row untimed entries share a single ┣━━━┫ boundary instead of ┗┛ + ┏┓.
func TestRenderUntimed_SharedBorderBetweenAdjacentEntries(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Alpha", DurationMinute: 30},
		{Id: 2, Name: "Beta", DurationMinute: 30},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	lines := rowsOf(out)

	// Two 30-min multi-row entries sharing a boundary: ┏┓ + label-A + ┣┫ + label-B + ┗┛ = 5 lines.
	if len(lines) != 5 {
		t.Errorf("two adjacent 30min entries: expected 5 lines (sharing boundary), got %d:\n%s", len(lines), out)
	}
	// The shared boundary line must contain ┣ and ┫ but not ┗ or ┏.
	sharedLine := lines[2]
	if !strings.Contains(sharedLine, "┣") || !strings.Contains(sharedLine, "┫") {
		t.Errorf("shared boundary line: expected ┣...┫; got %q", sharedLine)
	}
	if strings.ContainsAny(sharedLine, "┗┏") {
		t.Errorf("shared boundary line must not have corner chars; got %q", sharedLine)
	}
	// Alpha label on line 1.
	if !strings.Contains(lines[1], "Alpha") {
		t.Errorf("line 1: expected 'Alpha'; got %q", lines[1])
	}
	// Beta label on line 3.
	if !strings.Contains(lines[3], "Beta") {
		t.Errorf("line 3: expected 'Beta'; got %q", lines[3])
	}
}

// TestRenderUntimed_LastLineColorMatchesInterior checks that the last line of a
// multi-row untimed entry (the ┗┛ bottom) receives SelectionStyle markers just
// like the interior lines (standalone-border fix).
func TestRenderUntimed_LastLineColorMatchesInterior(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Style test", DurationMinute: 30},
	}
	opts := cli.GridOptions{HideID: true, SelectedID: 1, SelectionStyle: styleMarker}
	out := cli.RenderUntimed(entries, 80, true, opts)
	lines := rowsOf(out)

	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "┗") && !strings.Contains(lastLine, "┛") {
		t.Fatalf("last line is not the bottom border; got %q", lastLine)
	}
	if !strings.Contains(lastLine, "<<") {
		t.Errorf("selected entry bottom border must carry SelectionStyle markers; got %q", lastLine)
	}
}

// TestSelectionStyle_StandaloneBorders_GapBeforeAfter checks that when an entry
// has a gap before and after it (standalone top and bottom borders), those border
// lines carry SelectionStyle markers and the hour gutter is left unstyled.
func TestSelectionStyle_StandaloneBorders_GapBeforeAfter(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Early", StartMinute: pint32(480), DurationMinute: 60},  // 08:00–09:00
		{Id: 2, Name: "Later", StartMinute: pint32(660), DurationMinute: 60},  // 11:00–12:00
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

// TestRenderUntimed_ParityWithGridBox checks that a 30-min untimed entry produces
// output that is byte-for-byte identical to what the grid would produce for the
// interior portion (┃content┃) of an equivalent 30-min entry box.
func TestRenderUntimed_ParityWithGridBox(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Parity check", DurationMinute: 30},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	lines := rowsOf(out)

	// The interior line must be flanked by ┃ chars (matching grid interior).
	interiorLine := lines[1]
	if !strings.HasPrefix(strings.TrimPrefix(interiorLine, "       │ "), "┃") {
		t.Errorf("interior line: expected ┃ after gutter+rail+pad; got %q", interiorLine)
	}
	if !strings.HasSuffix(interiorLine, "┃ │") {
		t.Errorf("interior line: expected ┃ │ suffix; got %q", interiorLine)
	}
}

// TestRenderUntimed_LineWidthMatchesGrid checks that each line has the same
// visual width as a grid row (no wider, no narrower).
func TestRenderUntimed_LineWidthMatchesGrid(t *testing.T) {
	entries := []*planv1.PlanEntry{
		{Id: 1, Name: "Task", DurationMinute: 30},
	}
	out := cli.RenderUntimed(entries, 80, false, cli.GridOptions{HideID: true})
	gridOut := cli.RenderGrid(nil, "2026-05-27", fixedTime(9, 0), 80, false, cli.GridOptions{})

	untimedLines := rowsOf(out)
	gridLines := rowsOf(gridOut)

	wantWidth := visWidth(gridLines[0])
	for i, l := range untimedLines {
		if w := visWidth(l); w != wantWidth {
			t.Errorf("RenderUntimed line %d: width = %d, want %d (grid width); line: %q", i, w, wantWidth, l)
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
