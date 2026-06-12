package markdown

import (
	"charm.land/lipgloss/v2"
	ligtable "charm.land/lipgloss/v2/table"
	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// renderTable renders a goldmark GFM table node using lipgloss/v2 table.
func renderTable(node gast.Node, src []byte, theme Theme, opts Options) string {
	var headers []string
	var rows [][]string

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case extast.KindTableHeader:
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if cell.Kind() == extast.KindTableCell {
					headers = append(headers, renderInlineNodes(cell, src, theme, opts.Styled))
				}
			}
		case extast.KindTableRow:
			var row []string
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if cell.Kind() == extast.KindTableCell {
					row = append(row, renderInlineNodes(cell, src, theme, opts.Styled))
				}
			}
			rows = append(rows, row)
		}
	}

	tbl := ligtable.New()

	if !opts.Styled {
		tbl = tbl.Border(lipgloss.ASCIIBorder())
	} else {
		// MarkdownBorder uses only ASCII pipe characters, so byte-length equals
		// visual width. Disable top/bottom borders to keep the output clean.
		tbl = tbl.Border(lipgloss.MarkdownBorder()).BorderTop(false).BorderBottom(false)
	}

	if opts.Width > 0 {
		tbl = tbl.Width(opts.Width)
	}

	tbl = tbl.Wrap(true)

	if len(headers) > 0 {
		tbl = tbl.Headers(headers...)
	}

	if len(rows) > 0 {
		tbl = tbl.Rows(rows...)
	}

	return tbl.String()
}
