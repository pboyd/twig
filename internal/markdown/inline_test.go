package markdown

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	"github.com/yuin/goldmark"
)

var inlineTestTheme = Theme{
	Accent: lipgloss.Color("#5F9FFF"),
	Dim:    lipgloss.Color("#7A7A7A"),
	CodeBg: lipgloss.Color("#1A2A3A"),
}

// parseInline parses markdown and returns the document root and source bytes.
// The first child of the document is typically a paragraph whose children
// are the inline nodes.
func parseInline(t *testing.T, src string) (gast.Node, []byte) {
	t.Helper()
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	srcBytes := []byte(src)
	reader := text.NewReader(srcBytes)
	doc := md.Parser().Parse(reader)
	return doc, srcBytes
}

// TestRenderInlineNodes_StyledBoldHasAnsi verifies that bold text produces
// ANSI escape sequences when styled=true.
func TestRenderInlineNodes_StyledBoldHasAnsi(t *testing.T) {
	doc, src := parseInline(t, "**foo**")
	para := doc.FirstChild()
	if para == nil {
		t.Fatal("expected a paragraph node, got nil")
	}
	result := renderInlineNodes(para, src, inlineTestTheme, true)
	if !strings.ContainsRune(result, '\x1b') {
		t.Errorf("styled bold: expected ANSI escape sequences in output, got %q", result)
	}
}

// TestRenderInlineNodes_PlainNoAnsi verifies that no ANSI escape sequences
// appear when styled=false, regardless of markup.
func TestRenderInlineNodes_PlainNoAnsi(t *testing.T) {
	cases := []string{
		"**bold**",
		"*italic*",
		"`code`",
		"~~strike~~",
		"[link](https://example.com)",
		"plain text",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			doc, srcBytes := parseInline(t, src)
			para := doc.FirstChild()
			if para == nil {
				t.Fatal("expected a paragraph node, got nil")
			}
			result := renderInlineNodes(para, srcBytes, inlineTestTheme, false)
			if strings.ContainsRune(result, '\x1b') {
				t.Errorf("plain mode: expected no ANSI escapes, got %q", result)
			}
		})
	}
}

// TestRenderInlineNodes_NoLeftoverMarkupChars verifies that markup characters
// (*, `, ~) are not passed through literally in styled mode.
func TestRenderInlineNodes_NoLeftoverMarkupChars(t *testing.T) {
	cases := []struct {
		src     string
		badChar string
	}{
		{"**bold**", "*"},
		{"`code`", "`"},
		{"~~strike~~", "~"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			doc, srcBytes := parseInline(t, tc.src)
			para := doc.FirstChild()
			if para == nil {
				t.Fatal("expected a paragraph node, got nil")
			}
			result := renderInlineNodes(para, srcBytes, inlineTestTheme, true)
			// Strip ANSI sequences to examine only text content.
			stripped := stripAnsi(result)
			if strings.Contains(stripped, tc.badChar) {
				t.Errorf("styled mode: leftover markup char %q in stripped output %q (full: %q)",
					tc.badChar, stripped, result)
			}
		})
	}
}

// TestRenderInlineNodes_InlineCodePlainBacktickWrapped verifies that inline
// code in plain mode is wrapped in backticks (e.g. `code`).
func TestRenderInlineNodes_InlineCodePlainBacktickWrapped(t *testing.T) {
	doc, src := parseInline(t, "`mycode`")
	para := doc.FirstChild()
	if para == nil {
		t.Fatal("expected a paragraph node, got nil")
	}
	result := renderInlineNodes(para, src, inlineTestTheme, false)
	if !strings.Contains(result, "`mycode`") {
		t.Errorf("plain inline code: expected `mycode` in result, got %q", result)
	}
}

// TestRenderInlineNodes_PlainTextPassthrough verifies that plain text is
// returned unchanged in both styled and plain modes.
func TestRenderInlineNodes_PlainTextPassthrough(t *testing.T) {
	doc, src := parseInline(t, "hello world")
	para := doc.FirstChild()
	if para == nil {
		t.Fatal("expected a paragraph node, got nil")
	}
	for _, styled := range []bool{true, false} {
		result := renderInlineNodes(para, src, inlineTestTheme, styled)
		if !strings.Contains(result, "hello world") {
			t.Errorf("styled=%v: expected plain text in result, got %q", styled, result)
		}
	}
}

