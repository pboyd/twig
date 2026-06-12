package markdown

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// renderLink renders a markdown link for terminal display.
// styled=true: OSC 8 hyperlink wrapping underlined accent-colored text.
// styled=false: "text (url)" plain text.
func renderLink(text, url string, theme Theme, styled bool) string {
	if !styled {
		return text + " (" + url + ")"
	}

	linkStyle := lipgloss.NewStyle().Underline(true).Foreground(theme.Accent)
	styledText := linkStyle.Render(text)
	return ansi.SetHyperlink(url) + styledText + ansi.ResetHyperlink()
}
