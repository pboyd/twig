package markdown

import (
	"strings"

	"github.com/rivo/uniseg"
)

// wrapText wraps text to width columns using display-width-aware measurement.
// Existing newlines are preserved. Returns text unchanged if width <= 0.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		out = append(out, wrapLine(line, width))
	}
	return strings.Join(out, "\n")
}

// wrapLine wraps a single line (no embedded newlines) to width columns.
// Words wider than width are emitted as-is on their own line.
func wrapLine(line string, width int) string {
	if line == "" {
		return ""
	}

	// Split line into words on whitespace boundaries.
	words := strings.Fields(line)
	if len(words) == 0 {
		return ""
	}

	var sb strings.Builder
	lineWidth := 0

	for i, word := range words {
		wordW := displayWidth(word)

		if i == 0 {
			// First word: always place it on the current line.
			sb.WriteString(word)
			lineWidth = wordW
			continue
		}

		// Space + word fits on current line?
		if lineWidth+1+wordW <= width {
			sb.WriteByte(' ')
			sb.WriteString(word)
			lineWidth += 1 + wordW
		} else {
			// Start a new line.
			sb.WriteByte('\n')
			sb.WriteString(word)
			lineWidth = wordW
		}
	}

	return sb.String()
}

// displayWidth returns the terminal display width of s, measured in columns,
// using Unicode grapheme cluster segmentation and East Asian Width rules.
func displayWidth(s string) int {
	return uniseg.StringWidth(s)
}
