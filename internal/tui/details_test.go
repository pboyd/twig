package tui

import (
	"strings"
	"testing"
	"time"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestRenderDetails_CompletedTaskNoStrikethrough (T-G) asserts that renderDetails
// does not apply strikethrough or dim to the task name even when CompletedAt is set,
// and that the Completed: line is still present. (unstyled path)
func TestRenderDetails_CompletedTaskNoStrikethrough(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	task := &taskv1.Task{
		Id:          42,
		Name:        "finished work",
		CompletedAt: timestamppb.New(now),
	}

	out := renderDetails(task, 80, false)

	if strings.Contains(out, "\x1b[9m") {
		t.Errorf("renderDetails: name must not have strikethrough (\\x1b[9m); got:\n%q", out)
	}
	if strings.Contains(out, "\x1b[2m") {
		t.Errorf("renderDetails: name must not have dim (\\x1b[2m); got:\n%q", out)
	}
	if strings.Contains(out, "\x1b[2;9m") {
		t.Errorf("renderDetails: name must not have dim+strikethrough (\\x1b[2;9m); got:\n%q", out)
	}
	if !strings.Contains(out, "finished work") {
		t.Errorf("renderDetails: task name missing from output; got:\n%q", out)
	}
	if !strings.Contains(out, "Completed:") {
		t.Errorf("renderDetails: Completed: line missing from output; got:\n%q", out)
	}
}

// ── T002: wrapDescription (US1) ──────────────────────────────────────────────

func TestWrapDescription_TwoLines(t *testing.T) {
	out := wrapDescription("Line one\nLine two", 80)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), out)
	}
	if lines[0] != "Line one" {
		t.Errorf("line[0]: want %q, got %q", "Line one", lines[0])
	}
	if lines[1] != "Line two" {
		t.Errorf("line[1]: want %q, got %q", "Line two", lines[1])
	}
}

func TestWrapDescription_BlankLinePreserved(t *testing.T) {
	out := wrapDescription("para1\n\npara2", 80)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines (para1, blank, para2), got %d: %q", len(lines), out)
	}
	if lines[1] != "" {
		t.Errorf("blank line[1]: want empty string, got %q", lines[1])
	}
}

func TestWrapDescription_LeadingIndentPreserved(t *testing.T) {
	out := wrapDescription("  - milk", 80)
	if !strings.HasPrefix(out, "  ") {
		t.Errorf("leading indentation not preserved: got %q", out)
	}
	if !strings.Contains(out, "- milk") {
		t.Errorf("content missing from output: %q", out)
	}
}

func TestWrapDescription_LongLineWraps(t *testing.T) {
	out := wrapDescription("Line one\nthe quick brown fox jumps over", 20)
	lines := strings.Split(out, "\n")
	// "Line one" fits in 20 chars, then "the quick brown fox" (19 chars) and "jumps over" (10 chars)
	if len(lines) < 3 {
		t.Fatalf("long line should wrap to at least 3 lines; got %d: %q", len(lines), out)
	}
	if lines[0] != "Line one" {
		t.Errorf("line[0] (short line) must be unchanged: got %q", lines[0])
	}
}

func TestWrapDescription_MidLineSpacesCollapse(t *testing.T) {
	out := wrapDescription("a     b", 80)
	if out != "a b" {
		t.Errorf("mid-line spaces: want %q, got %q", "a b", out)
	}
}

func TestWrapDescription_ZeroWidthPassthrough(t *testing.T) {
	text := "some\nmultiline\ntext"
	if got := wrapDescription(text, 0); got != text {
		t.Errorf("width=0: want unchanged text, got %q", got)
	}
	if got := wrapDescription(text, -5); got != text {
		t.Errorf("width=-5: want unchanged text, got %q", got)
	}
}

// TestRenderDetails_StyledHeader (T017) asserts the styled path renders name as bold header
// and labels as dim column-aligned text (C5.1/C5.2).
func TestRenderDetails_StyledHeader(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	task := &taskv1.Task{
		Id:          7,
		Name:        "my task",
		Estimate:    3,
		CompletedAt: timestamppb.New(now),
	}

	out := renderDetails(task, 60, true)

	// Name must appear.
	if !strings.Contains(out, "my task") {
		t.Errorf("renderDetails styled: task name missing; got:\n%q", out)
	}
	// Labels must appear (dim column-aligned): no longer an "Est" label (replaced by glyph row).
	for _, label := range []string{"ID", "Completed"} {
		if !strings.Contains(out, label) {
			t.Errorf("renderDetails styled: label %q missing; got:\n%q", label, out)
		}
	}
	// Glyph row must appear for estimate=3, completed=0.
	if strings.Count(out, "🍅") != 3 {
		t.Errorf("renderDetails styled: want 3 glyphs for estimate=3; got:\n%q", out)
	}
	// Unstyled path: no ANSI codes on name (this checks styled=false, not styled=true).
	plain := renderDetails(task, 60, false)
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("renderDetails unstyled: must not emit ANSI codes; got:\n%q", plain)
	}
}

// TestRenderDetails_GlyphRow (T011) asserts the glyph row replaces the numeric Est line.
func TestRenderDetails_GlyphRow(t *testing.T) {
	tests := []struct {
		name      string
		estimate  int32
		completed int32
		wantN     int
		wantRow   bool
	}{
		{"estimate only", 5, 0, 5, true},
		{"partial", 5, 2, 5, true},
		{"over", 5, 6, 6, true},
		{"no estimate no completions", 0, 0, 0, false},
		{"only completions", 0, 2, 2, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := &taskv1.Task{
				Id:                     1,
				Name:                   "t",
				Estimate:               tc.estimate,
				CompletedPomodoroCount: tc.completed,
			}
			// Styled path.
			out := renderDetails(task, 60, true)
			n := strings.Count(out, "🍅")
			if n != tc.wantN {
				t.Errorf("styled: glyph count=%d, want %d; %q", n, tc.wantN, out)
			}
			// Must not have numeric Est: line.
			if strings.Contains(out, "Est:") {
				t.Errorf("styled: must not contain 'Est:' label; got %q", out)
			}
			// Plain path.
			plain := renderDetails(task, 60, false)
			np := strings.Count(plain, "🍅")
			if np != tc.wantN {
				t.Errorf("plain: glyph count=%d, want %d; %q", np, tc.wantN, plain)
			}
			if tc.wantRow && strings.Contains(plain, "\x1b[") {
				t.Errorf("plain mode must not emit ANSI; got %q", plain)
			}
			// Must not have numeric Est: line in plain mode either.
			if strings.Contains(plain, "Est:") {
				t.Errorf("plain: must not contain 'Est:' label; got %q", plain)
			}
		})
	}
}
