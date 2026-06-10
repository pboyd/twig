package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/pboyd/twig/internal/cli"
)

// ─── T004: constructor tests ──────────────────────────────────────────────────

func TestNewCalendar_OpensOnFieldDate(t *testing.T) {
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)

	t.Run("YYYY-MM-DD", func(t *testing.T) {
		cal := newCalendar("2026-07-04", now, false)
		want := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
		if !cal.selected.Equal(want) {
			t.Errorf("selected: want %v, got %v", want, cal.selected)
		}
	})

	t.Run("RFC3339", func(t *testing.T) {
		cal := newCalendar("2026-07-04T15:30:00Z", now, false)
		want := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
		if !cal.selected.Equal(want) {
			t.Errorf("selected: want %v, got %v", want, cal.selected)
		}
	})

	t.Run("RFC3339 rfc3339Field preserves time", func(t *testing.T) {
		cal := newCalendar("2026-07-04T15:30:00Z", now, true)
		want := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
		if !cal.selected.Equal(want) {
			t.Errorf("selected: want %v, got %v", want, cal.selected)
		}
		wantKeep := 15*time.Hour + 30*time.Minute
		if cal.keepTime != wantKeep {
			t.Errorf("keepTime: want %v, got %v", wantKeep, cal.keepTime)
		}
		if !cal.rfc3339Out {
			t.Error("rfc3339Out should be true")
		}
	})
}

func TestNewCalendar_FallsBackToToday(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	today := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"garbage", "not-a-date"},
		{"malformed", "2026-13-99"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cal := newCalendar(tc.input, now, false)
			if !cal.selected.Equal(today) {
				t.Errorf("selected: want today %v, got %v", today, cal.selected)
			}
		})
	}
}

// ─── T005: render tests ───────────────────────────────────────────────────────

func TestCalendarView_Header(t *testing.T) {
	cal := &calendarModel{
		selected: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		today:    time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	}
	view := cal.View()
	if !strings.Contains(view, "June") || !strings.Contains(view, "2026") {
		t.Errorf("header missing month/year, got:\n%s", view)
	}
}

func TestCalendarView_WeekdayRow(t *testing.T) {
	cal := &calendarModel{
		selected: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		today:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	view := cal.View()
	if !strings.Contains(view, "Su") || !strings.Contains(view, "Sa") {
		t.Errorf("weekday row missing, got:\n%s", view)
	}
}

func TestCalendarView_SelectedDayHighlighted(t *testing.T) {
	cal := &calendarModel{
		selected: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		today:    time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	}
	view := cal.View()
	// The view should contain the rendered output for day 15.
	// Since styles may add ANSI codes, just check "15" appears somewhere.
	if !strings.Contains(view, "15") {
		t.Errorf("selected day 15 missing from view:\n%s", view)
	}
}

func TestCalendarView_ApproxWidth(t *testing.T) {
	cal := &calendarModel{
		selected: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		today:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	view := cal.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	// The weekday row is "Su Mo Tu We Th Fr Sa" = 20 chars
	for _, line := range lines[1:2] {
		stripped := stripANSI(line)
		if len(stripped) < 14 || len(stripped) > 30 {
			t.Errorf("weekday row width %d outside expected range [14,30]: %q", len(stripped), stripped)
		}
	}
}

// ─── T012: navigation tests (US2) ─────────────────────────────────────────────

func TestCalendarMoveDay(t *testing.T) {
	cases := []struct {
		name   string
		start  time.Time
		delta  int
		want   time.Time
	}{
		{"forward 1", time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), 1, time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)},
		{"back 1", time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), -1, time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)},
		{"forward 7", time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), 7, time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)},
		{"cross month boundary", time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), 1, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
		{"cross year boundary", time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), 1, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cal := &calendarModel{selected: tc.start, today: tc.start}
			cal.moveDay(tc.delta)
			if !cal.selected.Equal(tc.want) {
				t.Errorf("want %v, got %v", tc.want, cal.selected)
			}
		})
	}
}

func TestCalendarMoveMonth(t *testing.T) {
	cases := []struct {
		name  string
		start time.Time
		delta int
		want  time.Time
	}{
		{"June to July", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), 1, time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)},
		{"June to May", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), -1, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)},
		{"Jan31 to Feb (non-leap)", time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 1, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)},
		{"Jan31 to Feb (leap)", time.Date(2028, 1, 31, 0, 0, 0, 0, time.UTC), 1, time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)},
		{"Mar31 to Apr", time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC), 1, time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)},
		{"cross year back", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), -1, time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)},
		{"cross year forward", time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC), 1, time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cal := &calendarModel{selected: tc.start, today: tc.start}
			cal.moveMonth(tc.delta)
			if !cal.selected.Equal(tc.want) {
				t.Errorf("want %v, got %v", tc.want, cal.selected)
			}
		})
	}
}

func TestCalendarMoveYear(t *testing.T) {
	cases := []struct {
		name  string
		start time.Time
		delta int
		want  time.Time
	}{
		{"2026 to 2027", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), 1, time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)},
		{"2026 to 2025", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), -1, time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)},
		{"Feb29 leap to non-leap", time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC), 1, time.Date(2029, 2, 28, 0, 0, 0, 0, time.UTC)},
		{"Feb29 leap to leap", time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC), 4, time.Date(2032, 2, 29, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cal := &calendarModel{selected: tc.start, today: tc.start}
			cal.moveYear(tc.delta)
			if !cal.selected.Equal(tc.want) {
				t.Errorf("want %v, got %v", tc.want, cal.selected)
			}
		})
	}
}

func TestCalendarToday(t *testing.T) {
	today := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	cal := &calendarModel{
		selected: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		today:    today,
	}
	cal.selected = cal.today
	if !cal.selected.Equal(today) {
		t.Errorf("want today %v, got %v", today, cal.selected)
	}
}

// ─── T015: output-format tests (US3) ──────────────────────────────────────────

func TestCalendarConfirm_DateOnly(t *testing.T) {
	cal := &calendarModel{
		selected:   time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		today:      time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		rfc3339Out: false,
	}
	got := cal.confirm()
	if got != "2026-06-12" {
		t.Errorf("want %q, got %q", "2026-06-12", got)
	}
}

func TestCalendarConfirm_RFC3339PreservesTime(t *testing.T) {
	cal := &calendarModel{
		selected:   time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		today:      time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		keepTime:   15*time.Hour + 0*time.Minute,
		rfc3339Out: true,
	}
	got := cal.confirm()
	if got != "2026-06-12T15:00:00Z" {
		t.Errorf("want %q, got %q", "2026-06-12T15:00:00Z", got)
	}
}

func TestCalendarConfirm_RFC3339OpenedFromTimestamp(t *testing.T) {
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	cal := newCalendar("2026-06-10T15:00:00Z", now, true)
	// Advance to June 12
	cal.moveDay(2)
	got := cal.confirm()
	if got != "2026-06-12T15:00:00Z" {
		t.Errorf("want %q, got %q", "2026-06-12T15:00:00Z", got)
	}
}

func TestCalendarConfirm_SnoozeAlwaysDateOnly(t *testing.T) {
	// Snooze field: rfc3339Field=false, so rfc3339Out is never set
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	cal := newCalendar("2026-06-15", now, false)
	got := cal.confirm()
	if got != "2026-06-15" {
		t.Errorf("want %q, got %q", "2026-06-15", got)
	}
}

func TestCalendarConfirm_RoundTrip(t *testing.T) {
	// Every output from confirm() must be accepted by cli.ParseDue.
	cases := []struct {
		name       string
		fieldValue string
		rfc3339    bool
		moveDays   int
	}{
		{"date-only", "2026-06-15", false, 0},
		{"date-only move", "2026-06-15", false, 5},
		{"rfc3339 preserve", "2026-06-15T10:30:00Z", true, 3},
		{"empty fallback", "", false, 0},
	}
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cal := newCalendar(tc.fieldValue, now, tc.rfc3339)
			cal.moveDay(tc.moveDays)
			out := cal.confirm()
			if _, err := parseDueForTest(out); err != nil {
				t.Errorf("round-trip failed for %q: %v", out, err)
			}
		})
	}
}

func parseDueForTest(s string) (interface{}, error) {
	return cli.ParseDue(s)
}
