package markdown

import (
	"strings"

	"charm.land/lipgloss/v2"
	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// renderInlineNodes walks goldmark inline AST nodes and returns a styled or
// plain string. When styled is false, the output contains no ANSI escape
// sequences; only CodeSpan adds decoration (backtick wrapping).
func renderInlineNodes(node gast.Node, src []byte, theme Theme, styled bool) string {
	var sb strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		sb.WriteString(renderInlineNode(child, src, theme, styled))
	}
	return sb.String()
}

// renderInlineNode renders a single inline node and its descendants.
func renderInlineNode(node gast.Node, src []byte, theme Theme, styled bool) string {
	switch node.Kind() {
	case gast.KindText:
		t := node.(*gast.Text)
		s := string(t.Segment.Value(src))
		switch {
		case t.HardLineBreak():
			return s + "\n"
		case t.SoftLineBreak():
			return s + " "
		default:
			return s
		}

	case gast.KindString:
		s := node.(*gast.String)
		return string(s.Value)

	case gast.KindEmphasis:
		em := node.(*gast.Emphasis)
		inner := renderInlineNodes(node, src, theme, styled)
		if !styled {
			return inner
		}
		var s lipgloss.Style
		if em.Level >= 2 {
			// Level 2 = **bold**
			s = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
		} else {
			// Level 1 = *italic*
			s = lipgloss.NewStyle().Italic(true)
		}
		return s.Render(inner)

	case extast.KindStrikethrough:
		inner := renderInlineNodes(node, src, theme, styled)
		if !styled {
			return inner
		}
		return lipgloss.NewStyle().Strikethrough(true).Render(inner)

	case gast.KindCodeSpan:
		inner := renderInlineNodes(node, src, theme, styled)
		if !styled {
			return "`" + inner + "`"
		}
		return lipgloss.NewStyle().Background(theme.CodeBg).Render(inner)

	case gast.KindLink:
		link := node.(*gast.Link)
		text := renderInlineNodes(node, src, theme, styled)
		url := string(link.Destination)
		return renderLink(text, url, theme, styled)

	case gast.KindAutoLink:
		al := node.(*gast.AutoLink)
		label := string(al.Label(src))
		if !styled {
			// Plain mode: emit the URL as-is. renderLink would produce
			// "url (url)" because label==url for bare URLs, so we return
			// the label directly.
			return label
		}
		url := string(al.URL(src))
		return renderLink(label, url, theme, styled)

	default:
		// For any other node type, walk children and emit their content
		// without extra markup.
		return renderInlineNodes(node, src, theme, styled)
	}
}
