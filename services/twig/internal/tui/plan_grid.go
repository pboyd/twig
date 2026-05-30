package tui

import (
	"github.com/charmbracelet/lipgloss"
	planv1 "github.com/pboyd/twig/services/twig/gen/plan/v1"
	"github.com/pboyd/twig/services/twig/internal/cli"
)

// planGridOptions returns the cli.GridOptions for the current plan state:
// always HideID, SelectedID from the cursor, Styled from m.styled, and
// SelectionStyle set to the Tasks-tab highlight so selected cells match.
func (m Model) planGridOptions() cli.GridOptions {
	opts := cli.GridOptions{HideID: true, Styled: m.styled}
	if len(m.plan.entries) > 0 && m.plan.cursor < len(m.plan.entries) {
		opts.SelectedID = m.plan.entries[m.plan.cursor].Id
		bgColor := cursorBg.Light
		if m.hasDarkBackground {
			bgColor = cursorBg.Dark
		}
		hs := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color(bgColor))
		opts.SelectionStyle = func(s string) string { return hs.Render(s) }
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
