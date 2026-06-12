package markdown

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

var testTheme = Theme{
	Accent: lipgloss.Color("#5F9FFF"),
	Dim:    lipgloss.Color("#7A7A7A"),
	CodeBg: lipgloss.Color("#1A2A3A"),
}

// newTestRenderer returns a Renderer using testTheme.
func newTestRenderer() *Renderer {
	return NewRenderer(testTheme)
}

// TestRender_HeadingStyledDistinct verifies that a heading in styled mode looks
// different from plain text (contains ANSI escapes) and does not pass through
// the raw `#` marker character.
func TestRender_HeadingStyledDistinct(t *testing.T) {
	r := newTestRenderer()
	input := "# Hello\n"
	styled := r.Render(input, Options{Width: 80, Styled: true})

	// Styled output must contain ANSI escapes (heading should be styled).
	if !strings.ContainsRune(styled, '\x1b') {
		t.Errorf("heading styled: expected ANSI escape sequences, got %q", styled)
	}

	// The raw `#` marker must NOT appear in the stripped output
	// (the renderer strips it and applies styling instead).
	strippedStyled := stripAnsi(styled)
	if strings.Contains(strippedStyled, "#") {
		t.Errorf("heading styled: raw '#' should not appear in stripped output %q", strippedStyled)
	}

	// The word "Hello" must appear in the stripped output.
	if !strings.Contains(strings.ToLower(strippedStyled), "hello") {
		t.Errorf("heading styled: expected 'Hello' in output, got %q", styled)
	}
}

// TestRender_UnorderedListMarkers verifies that unordered list items have a
// Unicode bullet marker (not just the raw `- ` syntax), and that the item text
// appears.
func TestRender_UnorderedListMarkers(t *testing.T) {
	r := newTestRenderer()
	input := "- item one\n- item two\n"

	for _, styled := range []bool{true, false} {
		out := r.Render(input, Options{Width: 80, Styled: styled})
		stripped := stripAnsi(out)

		// Item text must appear.
		if !strings.Contains(stripped, "item one") {
			t.Errorf("unordered list styled=%v: expected 'item one' in output, got %q", styled, stripped)
		}

		// The raw `- ` GFM prefix must NOT survive verbatim — it should be
		// converted to a proper bullet marker.
		// A properly rendered list never starts a line with "- item" (with the
		// dash followed by a space and then the exact item text untransformed).
		if strings.Contains(stripped, "- item one") {
			t.Errorf("unordered list styled=%v: raw '- ' prefix survived verbatim in %q", styled, stripped)
		}
	}

	// Styled mode: Unicode bullet characters (not plain `-`).
	styled := r.Render(input, Options{Width: 80, Styled: true})
	strippedStyled := stripAnsi(styled)
	unicodeBullets := []string{"•", "●", "▸", "‣", "›", "◦", "‣"}
	hasBullet := false
	for _, b := range unicodeBullets {
		if strings.Contains(strippedStyled, b) {
			hasBullet = true
			break
		}
	}
	if !hasBullet {
		t.Errorf("unordered list styled: expected a Unicode bullet marker in %q", strippedStyled)
	}
}

// TestRender_OrderedList verifies that ordered list items show numbering and
// the raw `N. ` GFM prefix is consumed (converted to proper formatting).
func TestRender_OrderedList(t *testing.T) {
	r := newTestRenderer()
	input := "1. first\n2. second\n3. third\n"

	for _, styled := range []bool{true, false} {
		out := r.Render(input, Options{Width: 80, Styled: styled})
		stripped := stripAnsi(out)

		// Item text must appear.
		if !strings.Contains(stripped, "first") {
			t.Errorf("ordered list styled=%v: expected 'first' in output, got %q", styled, stripped)
		}
		if !strings.Contains(stripped, "second") {
			t.Errorf("ordered list styled=%v: expected 'second' in output, got %q", styled, stripped)
		}

		// The raw `1. first` GFM notation must NOT survive verbatim —
		// a properly rendered list never has "1. first" as a raw line.
		if strings.Contains(stripped, "1. first") {
			t.Errorf("ordered list styled=%v: raw '1. ' prefix survived verbatim in %q", styled, stripped)
		}
	}
}

// TestRender_NestedListIndentation verifies that nested list items are indented
// more than top-level items and the raw GFM prefix does not survive verbatim.
func TestRender_NestedListIndentation(t *testing.T) {
	r := newTestRenderer()
	input := "- top\n  - nested\n"

	out := r.Render(input, Options{Width: 80, Styled: false})

	// The raw `- top` and `  - nested` prefixes must not survive verbatim.
	if strings.Contains(out, "- top") {
		t.Errorf("nested list: raw '- top' prefix survived verbatim in %q", out)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Find lines containing "top" and "nested".
	topIndent := -1
	nestedIndent := -1
	for _, line := range lines {
		if strings.Contains(line, "top") && topIndent == -1 {
			topIndent = countLeadingSpaces(line)
		}
		if strings.Contains(line, "nested") && nestedIndent == -1 {
			nestedIndent = countLeadingSpaces(line)
		}
	}

	if topIndent == -1 {
		t.Fatalf("nested list: could not find 'top' line in output %q", out)
	}
	if nestedIndent == -1 {
		t.Fatalf("nested list: could not find 'nested' line in output %q", out)
	}
	if nestedIndent <= topIndent {
		t.Errorf("nested list: expected nested indent (%d) > top indent (%d) in output %q",
			nestedIndent, topIndent, out)
	}
}

// countLeadingSpaces counts leading space characters in s.
func countLeadingSpaces(s string) int {
	n := 0
	for _, c := range s {
		if c == ' ' {
			n++
		} else {
			break
		}
	}
	return n
}

// TestRender_TaskList verifies that task list items contain checkbox
// representations and that the raw GFM `- [ ]` syntax is consumed.
func TestRender_TaskList(t *testing.T) {
	r := newTestRenderer()
	input := "- [ ] todo item\n- [x] done item\n"

	for _, styled := range []bool{true, false} {
		out := r.Render(input, Options{Width: 80, Styled: styled})
		stripped := stripAnsi(out)

		// The raw GFM `- [ ] ` prefix must NOT survive verbatim.
		if strings.Contains(stripped, "- [ ]") {
			t.Errorf("task list styled=%v: raw '- [ ]' prefix survived verbatim in %q", styled, stripped)
		}

		// Must contain some checkbox-like representation.
		checkboxForms := []string{"[ ]", "[x]", "[X]", "✓", "✗", "☐", "☑", "✔", "✘"}
		hasCheckbox := false
		for _, cb := range checkboxForms {
			if strings.Contains(stripped, cb) {
				hasCheckbox = true
				break
			}
		}
		if !hasCheckbox {
			t.Errorf("task list styled=%v: expected checkbox representation in %q", styled, stripped)
		}

		// Item text must appear.
		if !strings.Contains(stripped, "todo item") {
			t.Errorf("task list styled=%v: expected 'todo item' in %q", styled, stripped)
		}
		if !strings.Contains(stripped, "done item") {
			t.Errorf("task list styled=%v: expected 'done item' in %q", styled, stripped)
		}
	}
}

// TestRender_BlockquoteBar verifies that blockquotes have a left-bar or accent
// decoration in styled mode, and `>` prefix in plain mode.
func TestRender_BlockquoteBar(t *testing.T) {
	r := newTestRenderer()
	input := "> quote text\n"

	styled := r.Render(input, Options{Width: 80, Styled: true})
	// Styled mode: must contain some left-bar/accent. The text "quote text" must appear.
	if !strings.Contains(stripAnsi(styled), "quote text") {
		t.Errorf("blockquote styled: expected 'quote text' in output, got %q", styled)
	}
	// Must have ANSI escapes (accent/color decoration) or a border character.
	borderChars := []string{"│", "|", "▌", "▍", "▎", "▏"}
	hasBorder := strings.ContainsRune(styled, '\x1b')
	for _, bc := range borderChars {
		if strings.Contains(styled, bc) {
			hasBorder = true
			break
		}
	}
	if !hasBorder {
		t.Errorf("blockquote styled: expected ANSI escapes or border character in %q", styled)
	}

	plain := r.Render(input, Options{Width: 80, Styled: false})
	if !strings.Contains(plain, ">") {
		t.Errorf("blockquote plain: expected '>' prefix in output, got %q", plain)
	}
	if !strings.Contains(plain, "quote text") {
		t.Errorf("blockquote plain: expected 'quote text' in output, got %q", plain)
	}
}

// TestRender_CodeBlockVerbatim verifies that fenced code block content is
// preserved exactly (no reflowing of code lines).
func TestRender_CodeBlockVerbatim(t *testing.T) {
	r := newTestRenderer()
	input := "```\nfoo := bar\nbaz(qux)\n```\n"

	for _, styled := range []bool{true, false} {
		out := r.Render(input, Options{Width: 40, Styled: styled})
		stripped := stripAnsi(out)

		// The exact code lines must be present without reflowing.
		if !strings.Contains(stripped, "foo := bar") {
			t.Errorf("code block styled=%v: expected 'foo := bar' verbatim in %q", styled, stripped)
		}
		if !strings.Contains(stripped, "baz(qux)") {
			t.Errorf("code block styled=%v: expected 'baz(qux)' verbatim in %q", styled, stripped)
		}
		// The backtick fence markers must NOT appear in output.
		if strings.Contains(stripped, "```") {
			t.Errorf("code block styled=%v: fence markers should not appear in output, got %q", styled, stripped)
		}
	}
}

// TestRender_HorizontalRule verifies that `---` alone produces a horizontal
// line representation — specifically a line longer than 3 characters (not the
// raw `---` stub passthrough).
func TestRender_HorizontalRule(t *testing.T) {
	r := newTestRenderer()
	input := "---\n"

	for _, styled := range []bool{true, false} {
		out := r.Render(input, Options{Width: 80, Styled: styled})
		stripped := stripAnsi(out)

		// Must contain a horizontal line of at least 10 characters.
		// Box-drawing chars or dash runs of length >= 10 count.
		hrPatterns := []string{
			"──────────", "──────────", "━━━━━━━━━━",
			"----------", "──────────",
		}
		hasHR := false
		for _, hr := range hrPatterns {
			if strings.Contains(stripped, hr) {
				hasHR = true
				break
			}
		}
		// Also accept any line that is >= 10 chars of a single repeating rune.
		if !hasHR {
			lines := strings.Split(strings.TrimSpace(stripped), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if displayWidth(trimmed) >= 10 && isRepeatingChar(trimmed) {
					hasHR = true
					break
				}
			}
		}
		if !hasHR {
			t.Errorf("horizontal rule styled=%v: expected a horizontal line (>=10 chars) in %q", styled, stripped)
		}
	}
}

// isRepeatingChar returns true if s consists of a single repeated rune.
func isRepeatingChar(s string) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	first := runes[0]
	for _, r := range runes[1:] {
		if r != first {
			return false
		}
	}
	return true
}

// TestRender_ImageAltPlaceholder verifies that an image's alt text appears in
// output and the raw `![...]()` syntax is consumed.
func TestRender_ImageAltPlaceholder(t *testing.T) {
	r := newTestRenderer()
	input := "![alt text](img.png)\n"

	for _, styled := range []bool{true, false} {
		out := r.Render(input, Options{Width: 80, Styled: styled})
		stripped := stripAnsi(out)

		// Alt text must appear.
		if !strings.Contains(stripped, "alt text") {
			t.Errorf("image alt styled=%v: expected 'alt text' in output, got %q", styled, stripped)
		}

		// The raw `![` image syntax must NOT survive verbatim.
		if strings.Contains(stripped, "![") {
			t.Errorf("image alt styled=%v: raw '![' syntax survived verbatim in %q", styled, stripped)
		}
	}
}

// TestRender_RawHTMLPassthrough verifies that inline HTML produces non-empty output.
func TestRender_RawHTMLPassthrough(t *testing.T) {
	r := newTestRenderer()
	input := "<em>html</em>\n"

	for _, styled := range []bool{true, false} {
		out := r.Render(input, Options{Width: 80, Styled: styled})
		if strings.TrimSpace(out) == "" {
			t.Errorf("raw HTML styled=%v: expected non-empty output, got %q", styled, out)
		}
	}
}

// TestRender_WidthWrapping verifies that a long paragraph rendered at width=40
// has no line longer than 40 display columns.
func TestRender_WidthWrapping(t *testing.T) {
	r := newTestRenderer()
	// A paragraph longer than 40 characters.
	input := "This is a fairly long paragraph that should definitely be wrapped at forty columns when rendered.\n"

	for _, styled := range []bool{true, false} {
		out := r.Render(input, Options{Width: 40, Styled: styled})
		lines := strings.Split(out, "\n")
		for i, line := range lines {
			w := displayWidth(stripAnsi(line))
			if w > 40 {
				t.Errorf("width wrap styled=%v: line %d has display width %d > 40: %q",
					styled, i, w, line)
			}
		}
	}
}

// TestRender_MalformedInputNoPanic verifies that malformed markdown inputs do
// not cause panics.
func TestRender_MalformedInputNoPanic(t *testing.T) {
	r := newTestRenderer()
	inputs := []struct {
		name  string
		input string
	}{
		{"unclosed bold", "**unclosed"},
		{"bad table", "|bad|table"},
		{"unclosed code fence", "```\nno closing fence"},
		{"nested unclosed", "**bold *italic unclosed"},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			// If this panics, the test runner reports it as a failure.
			for _, styled := range []bool{true, false} {
				_ = r.Render(tc.input, Options{Width: 80, Styled: styled})
			}
		})
	}
}

// TestRender_PlainModeZeroEscape verifies that Styled:false produces output
// with no ANSI escape sequences, for various markdown inputs.
func TestRender_PlainModeZeroEscape(t *testing.T) {
	r := newTestRenderer()
	inputs := []struct {
		name  string
		input string
	}{
		{"heading", "# Hello\n"},
		{"bold", "**bold text**\n"},
		{"italic", "*italic text*\n"},
		{"unordered list", "- item\n"},
		{"ordered list", "1. item\n"},
		{"blockquote", "> quote\n"},
		{"code block", "```\ncode\n```\n"},
		{"horizontal rule", "---\n"},
		{"task list", "- [ ] todo\n- [x] done\n"},
		{"image", "![alt](img.png)\n"},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			out := r.Render(tc.input, Options{Width: 80, Styled: false})
			if strings.ContainsRune(out, '\x1b') {
				t.Errorf("plain mode: expected no ANSI escapes for %q, got %q", tc.input, out)
			}
		})
	}
}
