package tui

import (
	"strings"

	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	"github.com/pboyd/twig/internal/cli/timeparse"
)

// previewID is the sentinel Id used for the transient preview entry.
// It is negative so it can never collide with a real server-assigned Id.
const previewID int32 = -1

// buildPlanPreview returns a transient preview entry derived from the current
// form fields, or nil when no timed preview should be shown.
//
// Returns nil when:
//   - mode is not planTaskTime, planEventForm, or planEdit
//   - the Start field does not parse to a valid time
func (m Model) buildPlanPreview() *planv1.PlanEntry {
	switch m.plan.mode {
	case planTaskTime, planEventForm, planEdit:
	default:
		return nil
	}

	fields := m.plan.form.fields
	if len(fields) == 0 {
		return nil
	}

	var nameField, startField, durField string
	switch m.plan.mode {
	case planTaskTime:
		// fields: [start, duration]
		if len(fields) < 1 {
			return nil
		}
		startField = strings.TrimSpace(fields[0].Value())
		if len(fields) >= 2 {
			durField = strings.TrimSpace(fields[1].Value())
		}
		// Name from linked task, or a placeholder.
		if m.plan.form.taskID != 0 {
			task := findTask(m.tree, m.plan.form.taskID)
			if task != nil {
				nameField = task.Name
			}
		}
		if nameField == "" {
			nameField = "Task"
		}
	case planEventForm:
		// fields: [name, start, duration]
		if len(fields) < 2 {
			return nil
		}
		nameField = strings.TrimSpace(fields[0].Value())
		startField = strings.TrimSpace(fields[1].Value())
		if len(fields) >= 3 {
			durField = strings.TrimSpace(fields[2].Value())
		}
		if nameField == "" {
			nameField = "Event"
		}
	case planEdit:
		// fields: [name, start, duration]
		if len(fields) < 2 {
			return nil
		}
		nameField = strings.TrimSpace(fields[0].Value())
		startField = strings.TrimSpace(fields[1].Value())
		if len(fields) >= 3 {
			durField = strings.TrimSpace(fields[2].Value())
		}
		if nameField == "" {
			nameField = "Entry"
		}
	}

	startMin, err := timeparse.ParseStart(startField)
	if err != nil {
		return nil
	}

	durMin := 30 // match the server-side default for a blank duration field
	if durField != "" {
		if d, err := timeparse.ParseDurationOrEnd(durField, startMin); err == nil {
			durMin = d
		}
	}

	start32 := int32(startMin)
	return &planv1.PlanEntry{
		Id:             previewID,
		Name:           nameField,
		StartMinute:    &start32,
		DurationMinute: int32(durMin),
	}
}

// planPreviewConflicts returns the set of 15-min slot-start minutes (multiples
// of 15) where the given preview overlaps any entry in others.
//
// Compares exact intervals: a conflict exists only when two exact intervals
// overlap by at least one minute. Touching boundaries do not conflict.
// Marked slots are limited to those spanning the overlap region.
//
// others MUST already exclude the edit target (self-exclusion is the caller's
// responsibility). Returns nil when preview is nil or no overlap exists.
func planPreviewConflicts(preview *planv1.PlanEntry, others []*planv1.PlanEntry) map[int]bool {
	if preview == nil || preview.StartMinute == nil {
		return nil
	}

	pStart := int(preview.GetStartMinute())
	pEnd := pStart + int(preview.DurationMinute)

	var result map[int]bool
	for _, o := range others {
		if o.StartMinute == nil {
			continue
		}
		oStart := int(o.GetStartMinute())
		oEnd := oStart + int(o.DurationMinute)

		// Compute exact overlap region.
		ovStart := max(pStart, oStart)
		ovEnd := min(pEnd, oEnd)
		if ovStart >= ovEnd {
			continue // no overlap; touching boundaries land here
		}

		// Mark slots spanning the overlap region.
		for t := snapDown15(ovStart); t < ovEnd; t += 15 {
			if result == nil {
				result = make(map[int]bool)
			}
			result[t] = true
		}
	}
	return result
}

// snapDown15 rounds min down to the nearest 15-minute boundary.
// Duplicated here to avoid a circular import from internal/cli.
func snapDown15(min int) int {
	return (min / 15) * 15
}
