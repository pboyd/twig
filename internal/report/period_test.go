package report

import (
	"testing"
	"time"
)

// ref is a fixed reference time: Wednesday 2025-06-11 14:00:00 local (mid-week, Q2).
var ref = time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)

func TestParsePreset_Recent(t *testing.T) {
	p, err := ParsePeriod("recent", "", "", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantFrom := time.Date(2025, 6, 10, 0, 0, 0, 0, time.Local)
	wantTo := time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", p.From, wantFrom)
	}
	if !p.To.Equal(wantTo) {
		t.Errorf("To = %v, want %v", p.To, wantTo)
	}
	if p.Label != "Yesterday & Today" {
		t.Errorf("Label = %q", p.Label)
	}
}

func TestParsePreset_Today(t *testing.T) {
	p, err := ParsePeriod("today", "", "", ref)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(want) || !p.To.Equal(want) {
		t.Errorf("From=%v To=%v, both want %v", p.From, p.To, want)
	}
}

func TestParsePreset_Yesterday(t *testing.T) {
	p, err := ParsePeriod("yesterday", "", "", ref)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2025, 6, 10, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(want) || !p.To.Equal(want) {
		t.Errorf("From=%v To=%v, both want %v", p.From, p.To, want)
	}
}

func TestParsePreset_Week(t *testing.T) {
	// ref is Wednesday 2025-06-11; Monday of this week is 2025-06-09.
	p, err := ParsePeriod("week", "", "", ref)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 6, 9, 0, 0, 0, 0, time.Local)
	wantTo := time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", p.From, wantFrom)
	}
	if !p.To.Equal(wantTo) {
		t.Errorf("To = %v, want %v", p.To, wantTo)
	}
}

func TestParsePreset_Week_OnMonday(t *testing.T) {
	// A Monday: should return just that day.
	monday := time.Date(2025, 6, 9, 10, 0, 0, 0, time.Local)
	p, err := ParsePeriod("week", "", "", monday)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 6, 9, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", p.From, wantFrom)
	}
}

func TestParsePreset_Week_OnSunday(t *testing.T) {
	// Sunday 2025-06-15 — Monday is 2025-06-09.
	sunday := time.Date(2025, 6, 15, 10, 0, 0, 0, time.Local)
	p, err := ParsePeriod("week", "", "", sunday)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 6, 9, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v (Sunday should use Mon of same week)", p.From, wantFrom)
	}
}

func TestParsePreset_LastWeek(t *testing.T) {
	// ref is Wed 2025-06-11; last week Mon = 2025-06-02, Sun = 2025-06-08.
	p, err := ParsePeriod("last-week", "", "", ref)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 6, 2, 0, 0, 0, 0, time.Local)
	wantTo := time.Date(2025, 6, 8, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", p.From, wantFrom)
	}
	if !p.To.Equal(wantTo) {
		t.Errorf("To = %v, want %v", p.To, wantTo)
	}
}

func TestParsePreset_Month(t *testing.T) {
	p, err := ParsePeriod("month", "", "", ref)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", p.From, wantFrom)
	}
}

func TestParsePreset_Quarter(t *testing.T) {
	// ref is 2025-06-11 → Q2 starts 2025-04-01.
	p, err := ParsePeriod("quarter", "", "", ref)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 4, 1, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", p.From, wantFrom)
	}
	wantTo := time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local)
	if !p.To.Equal(wantTo) {
		t.Errorf("To = %v, want %v", p.To, wantTo)
	}
}

func TestParsePreset_Quarter_Q1(t *testing.T) {
	q1 := time.Date(2025, 2, 15, 10, 0, 0, 0, time.Local)
	p, err := ParsePeriod("quarter", "", "", q1)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v (Q1)", p.From, wantFrom)
	}
}

func TestParsePreset_Quarter_Q3(t *testing.T) {
	q3 := time.Date(2025, 8, 1, 0, 0, 0, 0, time.Local)
	p, err := ParsePeriod("quarter", "", "", q3)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 7, 1, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v (Q3)", p.From, wantFrom)
	}
}

func TestParsePreset_Year(t *testing.T) {
	p, err := ParsePeriod("year", "", "", ref)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", p.From, wantFrom)
	}
}

func TestParseExplicit_Valid(t *testing.T) {
	p, err := ParsePeriod("", "2025-03-01", "2025-03-31", ref)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local)
	wantTo := time.Date(2025, 3, 31, 0, 0, 0, 0, time.Local)
	if !p.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", p.From, wantFrom)
	}
	if !p.To.Equal(wantTo) {
		t.Errorf("To = %v, want %v", p.To, wantTo)
	}
}

func TestParseExplicit_SameDay(t *testing.T) {
	p, err := ParsePeriod("", "2025-03-15", "2025-03-15", ref)
	if err != nil {
		t.Fatal(err)
	}
	if !p.From.Equal(p.To) {
		t.Errorf("From != To for single-day range")
	}
}

func TestParseExplicit_InvertedRange(t *testing.T) {
	_, err := ParsePeriod("", "2025-03-31", "2025-03-01", ref)
	if err == nil {
		t.Fatal("expected error for inverted range")
	}
}

func TestParseExplicit_BadFromDate(t *testing.T) {
	_, err := ParsePeriod("", "not-a-date", "2025-03-31", ref)
	if err == nil {
		t.Fatal("expected error for bad from date")
	}
}

func TestParseExplicit_BadToDate(t *testing.T) {
	_, err := ParsePeriod("", "2025-03-01", "99-99", ref)
	if err == nil {
		t.Fatal("expected error for bad to date")
	}
}

func TestParseExplicit_OnlyFrom(t *testing.T) {
	_, err := ParsePeriod("", "2025-03-01", "", ref)
	if err == nil {
		t.Fatal("expected error when only --from provided")
	}
}

func TestParseExplicit_OnlyTo(t *testing.T) {
	_, err := ParsePeriod("", "", "2025-03-31", ref)
	if err == nil {
		t.Fatal("expected error when only --to provided")
	}
}

func TestParsePreset_AndFromTo_Exclusive(t *testing.T) {
	_, err := ParsePeriod("week", "2025-03-01", "2025-03-07", ref)
	if err == nil {
		t.Fatal("expected error: preset and --from/--to are mutually exclusive")
	}
}

func TestParseUnknownPreset(t *testing.T) {
	_, err := ParsePeriod("fortnight", "", "", ref)
	if err == nil {
		t.Fatal("expected error for unknown preset")
	}
}

func TestPeriod_StartUTC_EndUTC(t *testing.T) {
	p, _ := ParsePeriod("today", "", "", ref)
	start := p.StartUTC()
	end := p.EndUTC()

	// start should be midnight local as UTC
	localMidnight := time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local).UTC()
	if !start.Equal(localMidnight) {
		t.Errorf("StartUTC = %v, want %v", start, localMidnight)
	}
	// end should be next midnight local as UTC (exclusive upper bound)
	nextMidnight := time.Date(2025, 6, 12, 0, 0, 0, 0, time.Local).UTC()
	if !end.Equal(nextMidnight) {
		t.Errorf("EndUTC = %v, want %v", end, nextMidnight)
	}
	// start < end
	if !start.Before(end) {
		t.Errorf("StartUTC should be before EndUTC")
	}
}

func TestPeriod_Days(t *testing.T) {
	tests := []struct {
		from, to time.Time
		want     int
	}{
		{time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local), time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local), 1},
		{time.Date(2025, 6, 10, 0, 0, 0, 0, time.Local), time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local), 2},
		{time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local), time.Date(2025, 6, 30, 0, 0, 0, 0, time.Local), 30},
	}
	for _, tc := range tests {
		p := Period{From: tc.from, To: tc.to}
		if got := p.Days(); got != tc.want {
			t.Errorf("Days() = %d, want %d (from=%v to=%v)", got, tc.want, tc.from, tc.to)
		}
	}
}

func TestPeriod_Layout_Threshold(t *testing.T) {
	today := time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local)

	// 14-day period → DayGrouped
	p14 := Period{From: today.AddDate(0, 0, -13), To: today}
	if p14.Days() != 14 {
		t.Fatalf("expected 14 days, got %d", p14.Days())
	}
	if p14.ReportLayout() != DayGrouped {
		t.Errorf("14-day period should use DayGrouped")
	}

	// 15-day period → AccomplishmentGrouped
	p15 := Period{From: today.AddDate(0, 0, -14), To: today}
	if p15.Days() != 15 {
		t.Fatalf("expected 15 days, got %d", p15.Days())
	}
	if p15.ReportLayout() != AccomplishmentGrouped {
		t.Errorf("15-day period should use AccomplishmentGrouped")
	}
}

func TestPeriod_NearMidnight(t *testing.T) {
	// A task completed at 23:59:59 local should fall in today's window.
	p, _ := ParsePeriod("today", "", "", ref)
	nearMidnight := time.Date(2025, 6, 11, 23, 59, 59, 0, time.Local).UTC()
	if !nearMidnight.Before(p.EndUTC()) || nearMidnight.Before(p.StartUTC()) {
		t.Errorf("23:59:59 local should be inside today's [StartUTC, EndUTC)")
	}
	// A task completed exactly at midnight next day should NOT be in today's window.
	nextDay := time.Date(2025, 6, 12, 0, 0, 0, 0, time.Local).UTC()
	if nextDay.Before(p.EndUTC()) {
		t.Errorf("midnight of next day should be >= EndUTC (exclusive upper bound)")
	}
}

// TestDays_DSTSpringForward verifies calendar-day counting across the
// spring-forward transition (DST starts 2025-03-09 in America/New_York):
// the lost hour must not drop a day from the span.
func TestDays_DSTSpringForward(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	p := Period{
		From: time.Date(2025, 3, 1, 0, 0, 0, 0, ny),
		To:   time.Date(2025, 3, 15, 0, 0, 0, 0, ny),
	}
	if got := p.Days(); got != 15 {
		t.Errorf("Days() across spring-forward = %d, want 15", got)
	}
	if got := p.ReportLayout(); got != AccomplishmentGrouped {
		t.Errorf("ReportLayout() for a 15-day DST span = %v, want AccomplishmentGrouped", got)
	}
}

// TestDays_DSTFallBack verifies calendar-day counting across the fall-back
// transition (DST ends 2025-11-02 in America/New_York): the extra hour must
// not add a day to the span.
func TestDays_DSTFallBack(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	p := Period{
		From: time.Date(2025, 11, 1, 0, 0, 0, 0, ny),
		To:   time.Date(2025, 11, 14, 0, 0, 0, 0, ny),
	}
	if got := p.Days(); got != 14 {
		t.Errorf("Days() across fall-back = %d, want 14", got)
	}
	if got := p.ReportLayout(); got != DayGrouped {
		t.Errorf("ReportLayout() for a 14-day DST span = %v, want DayGrouped", got)
	}
}
