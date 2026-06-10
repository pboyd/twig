package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pboyd/twig/internal/cli"
)

// calendarModel is the transient state of an open calendar widget.
// selected always holds a valid date normalized to midnight UTC.
// The widget is nil when closed; non-nil when open.
type calendarModel struct {
	selected  time.Time
	today     time.Time
	keepTime  time.Duration // clock-time offset to re-apply on confirm (Due RFC3339 only)
	rfc3339Out bool          // emit RFC3339 on confirm when true
}

// newCalendar constructs a calendarModel from the current field value.
// fieldValue is parsed with cli.ParseDue; falls back to now on empty or error.
// rfc3339Field true means the Due field had an RFC3339 timestamp (preserve time).
func newCalendar(fieldValue string, now time.Time, rfc3339Field bool) *calendarModel {
	today := midnight(now)
	selected := today

	if fieldValue != "" {
		if ts, err := cli.ParseDue(fieldValue); err == nil {
			selected = midnight(ts.AsTime().UTC())
		}
	}

	cal := &calendarModel{
		selected: selected,
		today:    today,
	}

	if rfc3339Field && fieldValue != "" {
		if t, err := time.Parse(time.RFC3339, fieldValue); err == nil {
			dayStart := midnight(t.UTC())
			cal.keepTime = t.UTC().Sub(dayStart)
			cal.rfc3339Out = true
		}
	}

	return cal
}

// midnight normalizes t to midnight UTC.
func midnight(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// clampDay clamps day to the actual number of days in the given month/year.
func clampDay(year int, month time.Month, day int) int {
	last := daysInMonth(year, month)
	if day > last {
		return last
	}
	return day
}

// daysInMonth returns the number of days in the given month/year.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// moveDay moves the selection by delta days (positive or negative).
func (c *calendarModel) moveDay(delta int) {
	c.selected = c.selected.AddDate(0, 0, delta)
}

// moveMonth moves the selection by delta months with day clamping.
func (c *calendarModel) moveMonth(delta int) {
	y, m, d := c.selected.Date()
	m2 := time.Month(int(m) + delta)
	// Normalize month overflow via time.Date
	norm := time.Date(y, m2, 1, 0, 0, 0, 0, time.UTC)
	c.selected = time.Date(norm.Year(), norm.Month(), clampDay(norm.Year(), norm.Month(), d), 0, 0, 0, 0, time.UTC)
}

// moveYear moves the selection by delta years with day clamping.
func (c *calendarModel) moveYear(delta int) {
	y, m, d := c.selected.Date()
	y2 := y + delta
	c.selected = time.Date(y2, m, clampDay(y2, m, d), 0, 0, 0, 0, time.UTC)
}

// confirm returns the formatted date string to write into the field.
func (c *calendarModel) confirm() string {
	if c.rfc3339Out {
		t := c.selected.Add(c.keepTime)
		return t.Format(time.RFC3339)
	}
	return c.selected.Format("2006-01-02")
}

// handleKey processes a navigation key press for an open calendar.
// enter/esc/tab are handled by the form, not here.
func (c *calendarModel) handleKey(msg tea.KeyPressMsg) {
	switch {
	case msg.Code == tea.KeyLeft || msg.String() == "h":
		c.moveDay(-1)
	case msg.Code == tea.KeyRight || msg.String() == "l":
		c.moveDay(1)
	case msg.Code == tea.KeyUp || msg.String() == "k":
		c.moveDay(-7)
	case msg.Code == tea.KeyDown || msg.String() == "j":
		c.moveDay(7)
	case msg.String() == "[":
		c.moveMonth(-1)
	case msg.String() == "]":
		c.moveMonth(1)
	case msg.String() == "{":
		c.moveYear(-1)
	case msg.String() == "}":
		c.moveYear(1)
	case msg.String() == "t":
		c.selected = c.today
	}
}

// View renders the month grid as a ~22-column string.
func (c *calendarModel) View() string {
	y, m, _ := c.selected.Date()
	firstDay := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)

	var sb strings.Builder

	// Header: ◀ Month Year ▶
	header := fmt.Sprintf("◀ %s %d ▶", m.String(), y)
	sb.WriteString(header + "\n")

	// Weekday row
	sb.WriteString("Su Mo Tu We Th Fr Sa\n")

	// Starting weekday offset (Sunday = 0)
	startDow := int(firstDay.Weekday())
	totalDays := daysInMonth(y, m)
	ty, tm, td := c.today.Date()

	day := 1
	for row := 0; row < 6; row++ {
		if day > totalDays {
			break
		}
		var rowParts []string
		for col := 0; col < 7; col++ {
			cellNum := row*7 + col
			if cellNum < startDow || day > totalDays {
				rowParts = append(rowParts, dimStyle.Render("  "))
			} else {
				cell := fmt.Sprintf("%2d", day)
				isSelected := c.selected.Day() == day
				isToday := ty == y && tm == m && td == day

				if isSelected {
					cell = highlightStyle.Render(cell)
				} else if isToday {
					cell = accentStyle.Render(cell)
				}
				rowParts = append(rowParts, cell)
				day++
			}
		}
		sb.WriteString(strings.Join(rowParts, " ") + "\n")
	}

	return sb.String()
}
