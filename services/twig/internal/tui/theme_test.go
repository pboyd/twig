package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTheme_PaletteColorsAreAdaptive(t *testing.T) {
	tests := []struct {
		name  string
		color lipgloss.AdaptiveColor
	}{
		{"accent", accent},
		{"border", border},
		{"borderActive", borderActive},
		{"dim", dim},
		{"completed", completed},
		{"errorColor", errorColor},
		{"cursorBar", cursorBar},
		{"cursorBg", cursorBg},
	}
	for _, tc := range tests {
		if tc.color.Light == "" {
			t.Errorf("palette color %s: Light must not be empty", tc.name)
		}
		if tc.color.Dark == "" {
			t.Errorf("palette color %s: Dark must not be empty", tc.name)
		}
	}
}

func TestTheme_StylesFromPalette(t *testing.T) {
	fg := errorStyle.GetForeground()
	if ac, ok := fg.(lipgloss.AdaptiveColor); !ok {
		t.Error("errorStyle foreground must be an AdaptiveColor")
	} else if ac.Light == "" || ac.Dark == "" {
		t.Error("errorStyle foreground AdaptiveColor must have non-empty Light and Dark")
	}

	bg := highlightStyle.GetBackground()
	if ac, ok := bg.(lipgloss.AdaptiveColor); !ok {
		t.Error("highlightStyle background must be an AdaptiveColor")
	} else if ac.Light == "" || ac.Dark == "" {
		t.Error("highlightStyle background AdaptiveColor must have non-empty Light and Dark")
	}
}
