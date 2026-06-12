package markdown

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

var tableTestTheme = Theme{
	Accent: lipgloss.Color("#5F9FFF"),
	Dim:    lipgloss.Color("#7A7A7A"),
	CodeBg: lipgloss.Color("#1A2A3A"),
}

const basicTable = `| Name | Score | Notes |
|------|-------|-------|
| Alice | 100 | Great |
| Bob | 85 | Good |
`

const alignTable = `| Left | Center | Right |
|:-----|:------:|------:|
| a    |   b    |     c |
`

// TestTable_BasicRenderStripsRawPipes verifies that Render on a GFM table
// produces output that contains cell data and does NOT pass through raw pipe
// syntax unchanged (e.g. it should not look like "| Alice | 100 | Great |").
func TestTable_BasicRenderStripsRawPipes(t *testing.T) {
	r := NewRenderer(tableTestTheme)
	out := r.Render(basicTable, Options{Width: 80, Styled: true})
	stripped := stripAnsi(out)

	// Cell data must appear in output.
	if !strings.Contains(stripped, "Alice") {
		t.Errorf("basic table: expected cell value 'Alice' in output, got %q", stripped)
	}
	if !strings.Contains(stripped, "100") {
		t.Errorf("basic table: expected cell value '100' in output, got %q", stripped)
	}

	// Raw pipe-delimited rows should not pass through unchanged.
	// A stub returning input unchanged would contain literal "| Alice | 100 | Great |".
	if strings.Contains(stripped, "| Alice | 100 | Great |") {
		t.Errorf("basic table: raw pipe-delimited row passed through unchanged; output %q", stripped)
	}
}

// TestTable_ColumnAlignment verifies alignment handling. In plain mode an ASCII
// grid is expected; in styled mode the cell data must at least be present.
func TestTable_ColumnAlignment(t *testing.T) {
	r := NewRenderer(tableTestTheme)

	// Styled mode: cell data present.
	outStyled := r.Render(alignTable, Options{Width: 80, Styled: true})
	strippedStyled := stripAnsi(outStyled)
	for _, cell := range []string{"Left", "Center", "Right", "a", "b", "c"} {
		if !strings.Contains(strippedStyled, cell) {
			t.Errorf("alignment table styled: expected cell %q in output, got %q", cell, strippedStyled)
		}
	}

	// Plain mode: ASCII grid with pipe separators.
	outPlain := r.Render(alignTable, Options{Width: 80, Styled: false})
	if !strings.Contains(outPlain, "|") {
		t.Errorf("alignment table plain: expected '|' border characters in output, got %q", outPlain)
	}
}

// TestTable_WidthReflow verifies that a table rendered at a constrained width
// produces no line longer than that width (in visible characters, after stripping ANSI).
func TestTable_WidthReflow(t *testing.T) {
	const maxWidth = 40
	r := NewRenderer(tableTestTheme)
	out := r.Render(basicTable, Options{Width: maxWidth, Styled: true})
	stripped := stripAnsi(out)

	for _, line := range strings.Split(stripped, "\n") {
		if len(line) > maxWidth {
			t.Errorf("width reflow: line exceeds %d cols (%d): %q", maxWidth, len(line), line)
		}
	}
}

// TestTable_ShrinkToFit verifies that a very wide table rendered at a narrow
// width does not crash and produces non-empty output.
func TestTable_ShrinkToFit(t *testing.T) {
	wideTable := `| Column One | Column Two | Column Three | Column Four | Column Five | Column Six |
|------------|------------|--------------|-------------|-------------|------------|
| alpha      | beta       | gamma        | delta       | epsilon     | zeta       |
| eta        | theta      | iota         | kappa       | lambda      | mu         |
`
	r := NewRenderer(tableTestTheme)

	// Should not panic.
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("ShrinkToFit panicked: %v", p)
		}
	}()

	out := r.Render(wideTable, Options{Width: 20, Styled: false})
	if strings.TrimSpace(out) == "" {
		t.Error("ShrinkToFit: expected non-empty output for wide table at narrow width")
	}
}

// TestTable_PlainModeASCIIGrid verifies that plain-mode rendering of a table
// produces '|' ASCII border characters.
func TestTable_PlainModeASCIIGrid(t *testing.T) {
	r := NewRenderer(tableTestTheme)
	out := r.Render(basicTable, Options{Width: 80, Styled: false})

	if !strings.Contains(out, "|") {
		t.Errorf("plain mode ASCII grid: expected '|' characters in output, got %q", out)
	}
}

// TestTable_PlainModeNoEscapes verifies that plain-mode rendering of a table
// produces no ANSI escape sequences.
func TestTable_PlainModeNoEscapes(t *testing.T) {
	r := NewRenderer(tableTestTheme)
	out := r.Render(basicTable, Options{Width: 80, Styled: false})

	if strings.ContainsRune(out, '\x1b') {
		t.Errorf("plain mode zero-escape: expected no ANSI escapes, got %q", out)
	}
}
