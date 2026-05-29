package tui

import (
	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
)

// planGridOptions returns the cli.GridOptions for the current plan state:
// always HideID, and SelectedID set from the cursor position.
func (m Model) planGridOptions() cli.GridOptions {
	opts := cli.GridOptions{HideID: true}
	if len(m.plan.entries) > 0 && m.plan.cursor < len(m.plan.entries) {
		opts.SelectedID = m.plan.entries[m.plan.cursor].Id
	}
	return opts
}

// selectedPlanEntry returns the entry at the current cursor, or nil.
func selectedPlanEntry(entries []*planv1.PlanEntry, cursor int) *planv1.PlanEntry {
	if len(entries) == 0 || cursor < 0 || cursor >= len(entries) {
		return nil
	}
	return entries[cursor]
}
