package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// countPomodoroGlyphs counts the total number of pomodoro row glyphs
// (done ●, remaining ○, over ◆) in s, regardless of state.
func countPomodoroGlyphs(s string) int {
	return strings.Count(s, "●") + strings.Count(s, "○") + strings.Count(s, "◆")
}

func TestRenderPomodoroRow(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	tests := []struct {
		name       string
		estimate   int
		completed  int
		styled     bool
		wantEmpty  bool
		wantN      int // total glyph count (● ○ ◆ each count as 1)
		wantRed    int // bold-red (pomodoroDone) segments
		wantDim    int // dim segments
		wantYellow int // bold-yellow (pomodoroOver) segments
	}{
		{name: "empty state", estimate: 0, completed: 0, styled: true, wantEmpty: true},
		{name: "5/0 all dim", estimate: 5, completed: 0, styled: true, wantN: 5, wantRed: 0, wantDim: 5, wantYellow: 0},
		{name: "5/2 mixed", estimate: 5, completed: 2, styled: true, wantN: 5, wantRed: 2, wantDim: 3, wantYellow: 0},
		{name: "5/5 all done", estimate: 5, completed: 5, styled: true, wantN: 5, wantRed: 5, wantDim: 0, wantYellow: 0},
		{name: "5/6 over", estimate: 5, completed: 6, styled: true, wantN: 6, wantRed: 5, wantDim: 0, wantYellow: 1},
		{name: "3/1 mixed", estimate: 3, completed: 1, styled: true, wantN: 3, wantRed: 1, wantDim: 2, wantYellow: 0},
		{name: "0/2 all over", estimate: 0, completed: 2, styled: true, wantN: 2, wantRed: 0, wantDim: 0, wantYellow: 2},
		{name: "plain mode", estimate: 5, completed: 2, styled: false, wantN: 5},
	}

	doneStyle := lipgloss.NewStyle().Bold(true).Foreground(pomodoroDone)
	dimStyle := lipgloss.NewStyle().Foreground(dim)
	overStyle := lipgloss.NewStyle().Bold(true).Foreground(pomodoroOver)

	doneRendered := doneStyle.Render("●")
	dimRendered := dimStyle.Render("○")
	overRendered := overStyle.Render("◆")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderPomodoroRow(tc.estimate, tc.completed, tc.styled)

			if tc.wantEmpty {
				if got != "" {
					t.Errorf("want empty string, got %q", got)
				}
				return
			}

			if got == "" {
				t.Fatal("got empty string, want non-empty")
			}

			// Count glyphs.
			n := countPomodoroGlyphs(got)
			if n != tc.wantN {
				t.Errorf("glyph count: got %d, want %d; output: %q", n, tc.wantN, got)
			}

			if !tc.styled {
				// Plain mode: no ANSI codes.
				if strings.Contains(got, "\x1b[") {
					t.Errorf("plain mode must not contain ANSI codes; got %q", got)
				}
				return
			}

			// Count styled segments by counting occurrences of each styled glyph.
			gotRed := strings.Count(got, doneRendered)
			gotDim := strings.Count(got, dimRendered)
			gotYellow := strings.Count(got, overRendered)

			if gotRed != tc.wantRed {
				t.Errorf("red segments: got %d, want %d", gotRed, tc.wantRed)
			}
			if gotDim != tc.wantDim {
				t.Errorf("dim segments: got %d, want %d", gotDim, tc.wantDim)
			}
			if gotYellow != tc.wantYellow {
				t.Errorf("yellow segments: got %d, want %d", gotYellow, tc.wantYellow)
			}
		})
	}
}
