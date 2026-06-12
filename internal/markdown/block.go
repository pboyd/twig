package markdown

import (
	"strings"

	"charm.land/lipgloss/v2"
	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// renderBlocks renders a goldmark document AST as a terminal string.
func renderBlocks(doc gast.Node, src []byte, theme Theme, opts Options) string {
	blocks := collectBlocks(doc, src, theme, opts, 0)
	return strings.Join(blocks, "\n\n")
}

// collectBlocks recursively collects rendered block strings from node's children.
func collectBlocks(node gast.Node, src []byte, theme Theme, opts Options, depth int) []string {
	var blocks []string
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		rendered := renderBlock(child, src, theme, opts, depth)
		if rendered != "" {
			blocks = append(blocks, rendered)
		}
	}
	return blocks
}

// renderBlock renders a single block-level node.
func renderBlock(node gast.Node, src []byte, theme Theme, opts Options, depth int) string {
	switch node.Kind() {
	case gast.KindDocument:
		blocks := collectBlocks(node, src, theme, opts, depth)
		return strings.Join(blocks, "\n\n")

	case gast.KindHeading:
		h := node.(*gast.Heading)
		text := renderInlineNodes(node, src, theme, opts.Styled)
		if opts.Styled {
			return lipgloss.NewStyle().Bold(true).Foreground(theme.Accent).Render(text)
		}
		prefix := strings.Repeat("#", h.Level) + " "
		return prefix + text

	case gast.KindParagraph:
		text := renderInlineNodes(node, src, theme, opts.Styled)
		return wrapText(text, opts.Width)

	case gast.KindList:
		return renderList(node.(*gast.List), src, theme, opts, depth)

	case gast.KindListItem:
		// Should be handled by renderList, but handle standalone too.
		return renderListItemChildren(node, src, theme, opts, depth)

	case gast.KindBlockquote:
		return renderBlockquote(node, src, theme, opts, depth)

	case gast.KindFencedCodeBlock:
		return renderCodeBlock(node.(*gast.FencedCodeBlock), src, theme, opts)

	case gast.KindCodeBlock:
		return renderRawCodeBlock(node.(*gast.CodeBlock), src, theme, opts)

	case gast.KindThematicBreak:
		return renderThematicBreak(theme, opts)

	case gast.KindHTMLBlock:
		return renderHTMLBlock(node.(*gast.HTMLBlock), src)

	case gast.KindImage:
		return renderImage(node.(*gast.Image), src)

	default:
		// For table nodes and any unknown nodes, skip tables, recurse others.
		if node.Kind() == extast.KindTable {
			return "[table]"
		}
		// Walk children and collect their output.
		blocks := collectBlocks(node, src, theme, opts, depth)
		return strings.Join(blocks, "\n\n")
	}
}

// renderList renders an ordered or unordered list.
func renderList(list *gast.List, src []byte, theme Theme, opts Options, depth int) string {
	// Use 2 spaces per depth level, with minimum 2-space indent at depth 0
	// so that rendered output is visually distinct from raw GFM syntax.
	indent := strings.Repeat("  ", depth+1)
	var lines []string
	itemIdx := 0
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != gast.KindListItem {
			continue
		}
		itemIdx++

		// Determine bullet/number prefix.
		// Ordered lists use "N)" format (not "N." which matches raw GFM syntax).
		var marker string
		if list.IsOrdered() {
			marker = indent + itoa(itemIdx) + ") "
		} else {
			marker = indent + "• "
		}

		// Check for task checkbox as first child of list item.
		checkbox := ""
		firstInlineChild := child.FirstChild()
		if firstInlineChild != nil && firstInlineChild.Kind() == gast.KindTextBlock {
			// Task checkboxes appear as children of the text block.
			for n := firstInlineChild.FirstChild(); n != nil; n = n.NextSibling() {
				if n.Kind() == extast.KindTaskCheckBox {
					cb := n.(*extast.TaskCheckBox)
					if cb.IsChecked {
						checkbox = "[x] "
					} else {
						checkbox = "[ ] "
					}
					break
				}
			}
		}

		// Render the item content.
		itemContent := renderListItemContent(child, src, theme, opts, depth)

		// Prepend checkbox if present.
		if checkbox != "" {
			itemContent = checkbox + itemContent
		}

		// Combine marker and content; handle multi-line content by indenting continuation lines.
		contentLines := strings.Split(itemContent, "\n")
		var sb strings.Builder
		for i, cl := range contentLines {
			if i == 0 {
				sb.WriteString(marker + cl)
			} else {
				sb.WriteString("\n" + strings.Repeat(" ", len(marker)) + cl)
			}
		}
		lines = append(lines, sb.String())
	}
	return strings.Join(lines, "\n")
}

// renderListItemContent renders the content of a list item (paragraphs and nested lists).
func renderListItemContent(item gast.Node, src []byte, theme Theme, opts Options, depth int) string {
	var parts []string
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case gast.KindList:
			// Nested list — increase depth.
			nested := renderList(child.(*gast.List), src, theme, opts, depth+1)
			parts = append(parts, nested)
		case gast.KindTextBlock:
			// TextBlock contains inline content; render inline nodes but skip leading checkbox.
			text := renderInlineNodesSkipCheckbox(child, src, theme, opts.Styled)
			if text != "" {
				parts = append(parts, wrapText(text, opts.Width))
			}
		case gast.KindParagraph:
			text := renderInlineNodes(child, src, theme, opts.Styled)
			if text != "" {
				parts = append(parts, wrapText(text, opts.Width))
			}
		default:
			// Recurse for other block types.
			rendered := renderBlock(child, src, theme, opts, depth+1)
			if rendered != "" {
				parts = append(parts, rendered)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// renderListItemChildren renders all children of a list item as blocks (standalone fallback).
func renderListItemChildren(item gast.Node, src []byte, theme Theme, opts Options, depth int) string {
	return renderListItemContent(item, src, theme, opts, depth)
}

// renderInlineNodesSkipCheckbox renders inline nodes but skips any leading TaskCheckBox node.
func renderInlineNodesSkipCheckbox(node gast.Node, src []byte, theme Theme, styled bool) string {
	var sb strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == extast.KindTaskCheckBox {
			// Skip — handled separately.
			continue
		}
		sb.WriteString(renderInlineNode(child, src, theme, styled))
	}
	return strings.TrimPrefix(sb.String(), " ")
}

// renderBlockquote renders a blockquote node.
func renderBlockquote(node gast.Node, src []byte, theme Theme, opts Options, depth int) string {
	// Render inner content.
	innerOpts := Options{Width: opts.Width - 2, Styled: opts.Styled}
	if innerOpts.Width <= 0 {
		innerOpts.Width = opts.Width
	}
	blocks := collectBlocks(node, src, theme, innerOpts, depth)
	inner := strings.Join(blocks, "\n\n")

	lines := strings.Split(inner, "\n")
	var sb strings.Builder
	for i, line := range lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if opts.Styled {
			prefix := lipgloss.NewStyle().Foreground(theme.Dim).Render("│ ")
			sb.WriteString(prefix + line)
		} else {
			sb.WriteString("> " + line)
		}
	}
	return sb.String()
}

// renderCodeBlock renders a fenced code block.
func renderCodeBlock(node *gast.FencedCodeBlock, src []byte, theme Theme, opts Options) string {
	lines := extractNodeLines(node, src)
	// Remove trailing empty line if present.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if opts.Styled && theme.CodeBg != nil {
		// Apply background styling to each line.
		style := lipgloss.NewStyle().Background(theme.CodeBg)
		for i, l := range lines {
			lines[i] = style.Render(l)
		}
	}
	return strings.Join(lines, "\n")
}

// renderRawCodeBlock renders an indented code block.
func renderRawCodeBlock(node *gast.CodeBlock, src []byte, theme Theme, opts Options) string {
	lines := extractNodeLines(node, src)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if opts.Styled && theme.CodeBg != nil {
		style := lipgloss.NewStyle().Background(theme.CodeBg)
		for i, l := range lines {
			lines[i] = style.Render(l)
		}
	}
	return strings.Join(lines, "\n")
}

// extractNodeLines extracts raw text lines from a node's text segments.
func extractNodeLines(node gast.Node, src []byte) []string {
	var lines []string
	for i := 0; i < node.Lines().Len(); i++ {
		seg := node.Lines().At(i)
		line := string(seg.Value(src))
		// Remove trailing newline so we can join with \n ourselves.
		line = strings.TrimRight(line, "\n")
		lines = append(lines, line)
	}
	return lines
}

// renderThematicBreak renders a horizontal rule.
func renderThematicBreak(theme Theme, opts Options) string {
	width := opts.Width - 2
	if width < 10 {
		width = 10
	}
	if opts.Styled {
		line := strings.Repeat("─", width)
		return lipgloss.NewStyle().Foreground(theme.Dim).Render(line)
	}
	return strings.Repeat("-", width)
}

// renderHTMLBlock renders a raw HTML block by passing through its text.
func renderHTMLBlock(node *gast.HTMLBlock, src []byte) string {
	lines := extractNodeLines(node, src)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// renderImage renders an image node as a placeholder with alt text.
func renderImage(node *gast.Image, src []byte) string {
	alt := renderInlineNodes(node, src, Theme{}, false)
	alt = strings.TrimSpace(alt)
	if alt == "" {
		return "[image]"
	}
	return "[image: " + alt + "]"
}

// itoa converts a non-negative integer to its decimal string representation
// without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
