package markdown

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestDisplayWidth_AnsiEscapesNotCounted guards M2: ANSI escape bytes must not
// inflate the measured column width. The old uniseg-only path counted escape
// bytes as printable columns, causing styled text to wrap prematurely.
func TestDisplayWidth_AnsiEscapesNotCounted(t *testing.T) {
	// lipgloss wraps "bold" in SGR bold+color escapes; the visible text is 4 columns.
	styled := lipgloss.NewStyle().Bold(true).Render("bold")
	got := displayWidth(styled)
	if got != 4 {
		t.Errorf("displayWidth of styled %q: got %d, want 4 (ANSI escapes should not count)", styled, got)
	}
}

// TestWrapText_StyledEmphasisDoesNotWrapPrematurely guards M2: a styled inline
// span followed by a tail word should fit on one line when their combined visible
// width fits within the budget. Without the ANSI-aware fix, escape bytes inflate
// the measured width of the first word past the budget, causing a wrong break.
//
// "hello" styled bold = 5 visible cols; "world" = 5 visible cols; total = 11.
// width=15 fits both. But uniseg misreads the styled word as ~10 cols wide, so
// 10+1+5=16 > 15 and would incorrectly wrap.
func TestWrapText_StyledEmphasisDoesNotWrapPrematurely(t *testing.T) {
	bold := lipgloss.NewStyle().Bold(true).Render("hello") // "hello" = 5 visible cols
	line := bold + " world"                                 // 5 + 1 + 5 = 11 visible cols

	wrapped := wrapText(line, 15)
	if strings.Contains(wrapped, "\n") {
		t.Errorf("wrapText introduced a newline in an 11-visible-column line within width=15:\n%q", wrapped)
	}
}
