package cli

import (
	"fmt"
	"strings"

	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
)

// RenderGrid renders a list of plan entries as an ASCII day-planner grid.
// The grid spans from the earliest entry's start (rounded down to 15 min) to
// the latest entry's end (rounded up to 15 min), with 15-minute rows.
func RenderGrid(entries []*planv1.PlanEntry) string {
	if len(entries) == 0 {
		return "Plan is empty.\n"
	}

	// Compute floor and ceil in 15-minute boundaries.
	minStart := entries[0].StartMinute
	maxEnd := entries[0].StartMinute + entries[0].DurationMinute
	for _, e := range entries[1:] {
		if e.StartMinute < minStart {
			minStart = e.StartMinute
		}
		end := e.StartMinute + e.DurationMinute
		if end > maxEnd {
			maxEnd = end
		}
	}
	floor := (minStart / 15) * 15
	ceil := ((maxEnd + 14) / 15) * 15

	var sb strings.Builder
	for t := floor; t < ceil; t += 15 {
		timeLabel := fmt.Sprintf("%02d:%02d", t/60, t%60)
		cell := ""

		for _, e := range entries {
			start := e.StartMinute
			end := start + e.DurationMinute

			if start >= t+15 || end <= t {
				continue
			}

			startsHere := t <= start && start < t+15
			endsHere := t < end && end <= t+15

			if startsHere && start != t {
				timeLabel = fmt.Sprintf("~%02d:%02d", start/60, start%60)
			}

			switch {
			case startsHere && endsHere:
				cell = fmt.Sprintf("─ %d %s", e.Id, e.Name)
			case startsHere:
				cell = fmt.Sprintf("┌ %d %s", e.Id, e.Name)
			case endsHere:
				cell = "└"
			default:
				cell = "│"
			}
			break
		}

		sb.WriteString(fmt.Sprintf("%s │ %s\n", timeLabel, cell))
	}
	return sb.String()
}
