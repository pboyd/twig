package markdown

import (
	"strings"
	"testing"
)

// ── T018: RenderInline tests ──────────────────────────────────────────────────

// TestRenderInline_SingleLine asserts that RenderInline always returns a
// single line (no embedded newlines).
func TestRenderInline_SingleLine(t *testing.T) {
	cases := []string{
		"hello world",
		"**bold** and *italic*",
		"# Heading",
		"- bullet one\n- bullet two",
		"paragraph one\n\nparagraph two",
		"`code` span",
		"[link](https://example.com)",
	}
	r := newTestRenderer()
	for _, src := range cases {
		label := src
		if len(label) > 20 {
			label = label[:20]
		}
		t.Run(label, func(t *testing.T) {
			out := r.RenderInline(src, Options{Width: 80, Styled: true})
			if strings.Contains(out, "\n") {
				t.Errorf("RenderInline produced newline in output %q for input %q", out, src)
			}
		})
	}
}

// TestRenderInline_BlockSyntaxFlattened asserts that block-level syntax
// (headings, list markers) is flattened: the output contains the text
// without raw block syntax like leading '# '.
func TestRenderInline_BlockSyntaxFlattened(t *testing.T) {
	r := newTestRenderer()

	// Heading: "# Heading" → should contain "Heading" without leading "# "
	out := r.RenderInline("# Heading", Options{Width: 80, Styled: false})
	if strings.Contains(out, "# ") {
		t.Errorf("RenderInline kept '# ' block syntax in %q", out)
	}
	if !strings.Contains(out, "Heading") {
		t.Errorf("RenderInline dropped heading text, got %q", out)
	}

	// List: "- item" → should contain "item"
	out = r.RenderInline("- item", Options{Width: 80, Styled: false})
	if !strings.Contains(out, "item") {
		t.Errorf("RenderInline dropped list item text, got %q", out)
	}
}

// TestRenderInline_MarkupCharsStripped asserts that in styled mode, markup
// delimiters (*, `, ~) are not present in stripped output.
func TestRenderInline_MarkupCharsStripped(t *testing.T) {
	cases := []struct {
		src     string
		badChar string
	}{
		{"**bold**", "*"},
		{"`code`", "`"},
		{"~~strike~~", "~"},
	}
	r := newTestRenderer()
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			out := r.RenderInline(tc.src, Options{Width: 80, Styled: true})
			stripped := stripAnsi(out)
			if strings.Contains(stripped, tc.badChar) {
				t.Errorf("RenderInline styled: leftover markup %q in stripped output %q (full: %q)",
					tc.badChar, stripped, out)
			}
		})
	}
}

// TestRenderInline_PlainModeZeroEscape asserts that plain mode (Styled: false)
// produces no ANSI escape sequences for any input.
func TestRenderInline_PlainModeZeroEscape(t *testing.T) {
	cases := []string{
		"**bold**",
		"*italic*",
		"`code`",
		"~~strike~~",
		"[link](https://example.com)",
		"# Heading\n- item",
		"plain text",
	}
	r := newTestRenderer()
	for _, src := range cases {
		label := src
		if len(label) > 20 {
			label = label[:20]
		}
		t.Run(label, func(t *testing.T) {
			out := r.RenderInline(src, Options{Width: 80, Styled: false})
			if strings.ContainsRune(out, '\x1b') {
				t.Errorf("RenderInline plain: ANSI escape in output %q for input %q", out, src)
			}
		})
	}
}

// TestRenderInline_MultipleBlocksJoined asserts that multiple block elements
// are joined with spaces into a single line.
func TestRenderInline_MultipleBlocksJoined(t *testing.T) {
	r := newTestRenderer()
	out := r.RenderInline("First paragraph\n\nSecond paragraph", Options{Width: 80, Styled: false})
	if strings.Contains(out, "\n") {
		t.Errorf("RenderInline: expected single line, got %q", out)
	}
	if !strings.Contains(out, "First paragraph") || !strings.Contains(out, "Second paragraph") {
		t.Errorf("RenderInline: missing content in %q", out)
	}
}

// ── T027: Edge-case tests ─────────────────────────────────────────────────────

// TestRender_EmptyInput asserts that empty or whitespace-only input renders
// without error and produces no ANSI codes.
func TestRender_EmptyInput(t *testing.T) {
	r := newTestRenderer()
	for _, src := range []string{"", " ", "\n", "\t", "  \n  "} {
		out := r.Render(src, Options{Width: 80, Styled: true})
		if strings.ContainsRune(out, '\x1b') {
			t.Errorf("Render(%q): unexpected ANSI in output %q", src, out)
		}
	}
}

// TestRenderInline_EmptyInput asserts that empty or whitespace-only input
// renders empty without panic.
func TestRenderInline_EmptyInput(t *testing.T) {
	r := newTestRenderer()
	for _, src := range []string{"", " ", "\n", "  \n  "} {
		out := r.RenderInline(src, Options{Width: 80, Styled: true})
		if strings.ContainsRune(out, '\x1b') {
			t.Errorf("RenderInline(%q): unexpected ANSI in output %q", src, out)
		}
	}
}

// TestRender_LiteralMarkupNotInterpreted asserts that punctuation that does not
// form valid markdown markup is preserved in plain-mode output.
func TestRender_LiteralMarkupNotInterpreted(t *testing.T) {
	r := newTestRenderer()
	cases := []struct {
		input   string
		contain string
	}{
		// Single * not forming emphasis should pass through.
		{"price is $5 * 2", "*"},
		// Standalone # in middle of line should not become a heading.
		{"foo # bar", "#"},
	}
	for _, tc := range cases {
		out := r.Render(tc.input, Options{Width: 80, Styled: false})
		if !strings.Contains(out, tc.contain) {
			t.Errorf("Render(%q): expected %q in plain output, got %q", tc.input, tc.contain, out)
		}
	}
}

// TestRender_EmojiDisplayWidth asserts that content with emoji does not panic
// and produces non-empty output.
func TestRender_EmojiDisplayWidth(t *testing.T) {
	r := newTestRenderer()
	inputs := []string{
		"Hello 🌍 world",
		"Task ✅ done",
		"🎉 Celebration",
	}
	for _, src := range inputs {
		out := r.Render(src, Options{Width: 40, Styled: false})
		if out == "" {
			t.Errorf("Render(%q): got empty output", src)
		}
	}
}

// TestRenderInline_EmojiSingleLine asserts that emoji-containing names are
// returned as a single line without panic.
func TestRenderInline_EmojiSingleLine(t *testing.T) {
	r := newTestRenderer()
	out := r.RenderInline("Buy 🥛 milk **today**", Options{Width: 40, Styled: true})
	if strings.Contains(out, "\n") {
		t.Errorf("RenderInline with emoji: unexpected newline in %q", out)
	}
	if !strings.Contains(stripAnsi(out), "Buy") {
		t.Errorf("RenderInline with emoji: missing text in %q", out)
	}
}

// TestRenderInline_MalformedInputNoPanic asserts that malformed markdown inputs
// do not cause panics in RenderInline.
func TestRenderInline_MalformedInputNoPanic(t *testing.T) {
	r := newTestRenderer()
	cases := []string{
		"**unclosed bold",
		"[broken link(no-close",
		"```\nunclosed code",
		"> > > deep blockquote",
		"| col | no | separator",
	}
	for _, src := range cases {
		_ = r.RenderInline(src, Options{Width: 80, Styled: true})
		_ = r.RenderInline(src, Options{Width: 80, Styled: false})
	}
}
