package markdown

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

var linkTestTheme = Theme{
	Accent: lipgloss.Color("#5F9FFF"),
	Dim:    lipgloss.Color("#7A7A7A"),
	CodeBg: lipgloss.Color("#1A2A3A"),
}

func TestRenderLink_PlainFormat(t *testing.T) {
	result := renderLink("Go website", "https://go.dev", linkTestTheme, false)
	want := "Go website (https://go.dev)"
	if result != want {
		t.Errorf("plain renderLink: got %q, want %q", result, want)
	}
}

func TestRenderLink_PlainNoEscapes(t *testing.T) {
	result := renderLink("example", "https://example.com", linkTestTheme, false)
	if strings.ContainsRune(result, '\x1b') {
		t.Errorf("plain renderLink should contain no ANSI escapes, got %q", result)
	}
}

func TestRenderLink_StyledOSC8Present(t *testing.T) {
	result := renderLink("Click me", "https://example.com", linkTestTheme, true)
	// OSC 8 hyperlink sequences start with ESC ] 8 ;
	if !strings.Contains(result, "\x1b]8;;") {
		t.Errorf("styled renderLink: expected OSC 8 sequence (\\x1b]8;;), got %q", result)
	}
}

func TestRenderLink_StyledContainsURL(t *testing.T) {
	url := "https://example.com/path"
	result := renderLink("label", url, linkTestTheme, true)
	if !strings.Contains(result, url) {
		t.Errorf("styled renderLink: expected URL %q in result, got %q", url, result)
	}
}

func TestRenderLink_StyledContainsText(t *testing.T) {
	result := renderLink("label text", "https://example.com", linkTestTheme, true)
	// Strip ANSI sequences before checking for text content — the renderer
	// may apply per-character styling that interleaves escape codes with runes.
	stripped := stripAnsi(result)
	if !strings.Contains(stripped, "label text") {
		t.Errorf("styled renderLink: expected link text in stripped output, got raw %q", result)
	}
}
