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

// TestRender_SoftLineBreakJoinsWithSpace guards against editor-wrapped source
// text (a bare newline within a paragraph = CommonMark soft line break) being
// fused into a single word on render. Before the fix, goldmark's Text nodes
// were concatenated with no separator, so "words but\ndon't say" rendered as
// "words butdon't say".
func TestRender_SoftLineBreakJoinsWithSpace(t *testing.T) {
	r := NewRenderer(Theme{})
	got := r.Render("words but\ndon't say", Options{Width: 80})
	if strings.Contains(got, "butdon't") {
		t.Errorf("soft line break was fused without a space: %q", got)
	}
	if !strings.Contains(got, "but don't") {
		t.Errorf("expected \"but don't\" in output, got: %q", got)
	}
}

// TestRender_HardLineBreakPreservesNewline guards that an explicit hard line
// break (trailing two spaces before the newline) still renders as a newline,
// not a fused word or a plain space.
func TestRender_HardLineBreakPreservesNewline(t *testing.T) {
	r := NewRenderer(Theme{})
	got := r.Render("line one  \nline two", Options{Width: 80})
	if !strings.Contains(got, "line one\nline two") {
		t.Errorf("expected hard line break to render as a newline between \"line one\" and \"line two\", got: %q", got)
	}
}
