package tui

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestTheme_PaletteColorsAreNonNil(t *testing.T) {
	// initPalette is called via TestMain. Verify each exported palette var is set.
	colors := map[string]color.Color{
		"accent":          accent,
		"border":          border,
		"borderActive":    borderActive,
		"dim":             dim,
		"completed":       completed,
		"errorColor":      errorColor,
		"cursorBar":       cursorBar,
		"cursorBg":        cursorBg,
		"pomodoroDone":    pomodoroDone,
		"pomodoroOver":    pomodoroOver,
		"pomodoroRunning": pomodoroRunning,
	}
	for name, c := range colors {
		if c == nil {
			t.Errorf("palette color %s is nil after initPalette", name)
		}
	}
}

func TestTheme_PaletteAdaptsToBackground(t *testing.T) {
	// Verify light and dark backgrounds produce different cursorBg values.
	initPalette(false)
	lightCursorBg := cursorBg

	initPalette(true)
	darkCursorBg := cursorBg

	// Restore for other tests.
	initPalette(false)

	if lightCursorBg == darkCursorBg {
		t.Error("cursorBg should differ between light and dark backgrounds")
	}
}

func TestTheme_StylesFromPalette(t *testing.T) {
	fg := errorStyle.GetForeground()
	if fg == nil {
		t.Error("errorStyle foreground must not be nil")
	}

	bg := highlightStyle.GetBackground()
	if bg == nil {
		t.Error("highlightStyle background must not be nil")
	}
	_ = lipgloss.NewStyle() // confirm import is used
}
