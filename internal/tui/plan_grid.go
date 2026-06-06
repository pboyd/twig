package tui

import (
	"charm.land/lipgloss/v2"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	"github.com/pboyd/twig/internal/cli"
)

// planGridOptions returns the cli.GridOptions for the current plan state:
// always HideID, SelectedID from the cursor, Styled from m.styled, and
// SelectionStyle set to the Tasks-tab highlight so selected cells match.
// When a timed form is open, also sets PreviewID, PreviewStyle, ConflictStyle,
// and PreviewConflictSlots for the live preview rendering.
func (m Model) planGridOptions() cli.GridOptions {
	opts := cli.GridOptions{HideID: true, Styled: m.styled}
	if len(m.plan.entries) > 0 && m.plan.cursor < len(m.plan.entries) {
		opts.SelectedID = m.plan.entries[m.plan.cursor].Id
		ld := lipgloss.LightDark(m.hasDarkBackground)
		bg := ld(lipgloss.Color("#DDEEFF"), lipgloss.Color("#1A2A3A"))
		hs := lipgloss.NewStyle().Bold(true).Background(bg)
		opts.SelectionStyle = func(s string) string { return hs.Render(s) }
	}

	// Wire preview fields when a timed form is active.
	switch m.plan.mode {
	case planTaskTime, planEventForm, planEdit:
		opts.PreviewID = previewID
		if m.styled {
			dimStyle := lipgloss.NewStyle().Foreground(dim).Faint(true).Italic(true)
			opts.PreviewStyle = func(s string) string { return dimStyle.Render(s) }
			errStyle := lipgloss.NewStyle().Foreground(errorColor).Faint(true).Italic(true)
			opts.ConflictStyle = func(s string) string { return errStyle.Render(s) }
		}
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
