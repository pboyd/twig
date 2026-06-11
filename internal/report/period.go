package report

import (
	"fmt"
	"time"
)

// Layout determines which grouping is used when rendering a report.
type Layout int

const (
	DayGrouped            Layout = iota // periods ≤ 14 days
	AccomplishmentGrouped               // periods > 14 days
)

// Period is a resolved reporting window.
type Period struct {
	From  time.Time // first local calendar day (midnight local)
	To    time.Time // last local calendar day (midnight local); To >= From
	Label string    // human-readable label
}

// StartUTC returns the UTC instant of the start of the first day (inclusive).
func (p Period) StartUTC() time.Time {
	return p.From.UTC()
}

// EndUTC returns the UTC instant of the start of the day after the last day (exclusive).
func (p Period) EndUTC() time.Time {
	return p.To.AddDate(0, 0, 1).UTC()
}

// Days returns the number of calendar days in the period (inclusive).
// The span is computed from calendar dates, not wall-clock durations, so a
// DST transition inside the period cannot add or drop a day.
func (p Period) Days() int {
	from := time.Date(p.From.Year(), p.From.Month(), p.From.Day(), 0, 0, 0, 0, time.UTC)
	to := time.Date(p.To.Year(), p.To.Month(), p.To.Day(), 0, 0, 0, 0, time.UTC)
	return int(to.Sub(from)/(24*time.Hour)) + 1
}

// ReportLayout returns the layout to use for this period.
func (p Period) ReportLayout() Layout {
	if p.Days() <= 14 {
		return DayGrouped
	}
	return AccomplishmentGrouped
}

// presetOrder is the cycle order for TUI preset switching.
var presetOrder = []string{
	"recent",
	"today",
	"yesterday",
	"week",
	"last-week",
	"month",
	"quarter",
	"year",
}

// PresetOrder returns the ordered list of preset names for TUI cycling.
func PresetOrder() []string {
	return presetOrder
}

// ParsePeriod resolves a period from a preset name or explicit date range.
// If preset is non-empty, from/to must be empty and vice versa.
// now is used as the reference time for relative presets; it should be time.Now().
func ParsePeriod(preset, fromStr, toStr string, now time.Time) (Period, error) {
	if preset != "" && (fromStr != "" || toStr != "") {
		return Period{}, fmt.Errorf("a preset and --from/--to are mutually exclusive — pick one or the other")
	}
	if (fromStr != "") != (toStr != "") {
		return Period{}, fmt.Errorf("--from and --to must both be provided together (got only one)")
	}
	if preset == "" && fromStr == "" {
		preset = "recent"
	}

	if fromStr != "" {
		return parseExplicit(fromStr, toStr)
	}
	return resolvePreset(preset, now)
}

func parseExplicit(fromStr, toStr string) (Period, error) {
	from, err := time.ParseInLocation("2006-01-02", fromStr, time.Local)
	if err != nil {
		return Period{}, fmt.Errorf("can't parse --from %q — use YYYY-MM-DD (e.g. 2025-01-15)", fromStr)
	}
	to, err := time.ParseInLocation("2006-01-02", toStr, time.Local)
	if err != nil {
		return Period{}, fmt.Errorf("can't parse --to %q — use YYYY-MM-DD (e.g. 2025-01-15)", toStr)
	}
	if to.Before(from) {
		return Period{}, fmt.Errorf("--to (%s) must not be before --from (%s)", toStr, fromStr)
	}
	label := fmt.Sprintf("%s – %s", from.Format("Jan 2"), to.Format("Jan 2, 2006"))
	if from.Year() != to.Year() {
		label = fmt.Sprintf("%s – %s", from.Format("Jan 2, 2006"), to.Format("Jan 2, 2006"))
	} else if from.Month() == to.Month() && from.Day() == to.Day() {
		label = from.Format("Jan 2, 2006")
	}
	return Period{From: from, To: to, Label: label}, nil
}

func resolvePreset(preset string, now time.Time) (Period, error) {
	today := truncateDay(now)
	yesterday := today.AddDate(0, 0, -1)

	switch preset {
	case "recent":
		return Period{From: yesterday, To: today, Label: "Yesterday & Today"}, nil
	case "today":
		return Period{From: today, To: today, Label: "Today"}, nil
	case "yesterday":
		return Period{From: yesterday, To: yesterday, Label: "Yesterday"}, nil
	case "week":
		mon := weekMonday(today)
		return Period{From: mon, To: today, Label: "This Week"}, nil
	case "last-week":
		mon := weekMonday(today).AddDate(0, 0, -7)
		sun := mon.AddDate(0, 0, 6)
		label := fmt.Sprintf("%s – %s", mon.Format("Jan 2"), sun.Format("Jan 2, 2006"))
		if mon.Year() != sun.Year() {
			label = fmt.Sprintf("%s – %s", mon.Format("Jan 2, 2006"), sun.Format("Jan 2, 2006"))
		}
		return Period{From: mon, To: sun, Label: label}, nil
	case "month":
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		label := fmt.Sprintf("%s – %s", first.Format("Jan 2"), today.Format("Jan 2, 2006"))
		if first.Day() == today.Day() && first.Month() == today.Month() {
			label = today.Format("Jan 2, 2006")
		}
		return Period{From: first, To: today, Label: label}, nil
	case "quarter":
		first := quarterStart(today)
		label := fmt.Sprintf("%s – %s", first.Format("Jan 2"), today.Format("Jan 2, 2006"))
		if first.Equal(today) {
			label = today.Format("Jan 2, 2006")
		}
		return Period{From: first, To: today, Label: label}, nil
	case "year":
		first := time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, today.Location())
		label := fmt.Sprintf("Jan 1 – %s", today.Format("Jan 2, 2006"))
		if first.Equal(today) {
			label = today.Format("Jan 2, 2006")
		}
		return Period{From: first, To: today, Label: label}, nil
	default:
		return Period{}, fmt.Errorf("unknown preset %q — try: today, yesterday, week, last-week, month, quarter, year", preset)
	}
}

// truncateDay returns midnight of the given time in its local timezone.
func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// weekMonday returns midnight on the ISO Monday of the week containing t.
func weekMonday(t time.Time) time.Time {
	wd := t.Weekday()
	if wd == time.Sunday {
		wd = 7
	}
	// ISO: Mon=1 … Sun=7
	offset := int(wd) - 1
	return t.AddDate(0, 0, -offset)
}

// quarterStart returns the first day of the calendar quarter containing t.
func quarterStart(t time.Time) time.Time {
	month := t.Month()
	var qMonth time.Month
	switch {
	case month <= 3:
		qMonth = time.January
	case month <= 6:
		qMonth = time.April
	case month <= 9:
		qMonth = time.July
	default:
		qMonth = time.October
	}
	return time.Date(t.Year(), qMonth, 1, 0, 0, 0, 0, t.Location())
}
