package cli

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
)

// RenderGrid renders a day's plan entries as a calendar grid.
// Rows represent 15-minute slots; hour boundaries get a horizontal divider.
// The visible window defaults to 08:00–17:00 and is extended outward (rounded
// to the nearest hour) to contain every entry's snapped span.
func RenderGrid(entries []*planv1.PlanEntry, day string, now time.Time, width int, isTTY bool) string {
	winStart := 8 * 60
	winEnd := 17 * 60

	for _, e := range entries {
		sn := snapDown15(int(e.StartMinute))
		se := snapUp15(int(e.StartMinute) + int(e.DurationMinute))
		if h := (sn / 60) * 60; h < winStart {
			winStart = h
		}
		if h := ((se + 59) / 60) * 60; h > winEnd {
			winEnd = h
		}
	}

	fieldWidth := width - 8 // gutter(6) + leftEdge(1) + rightEdge(1)
	if fieldWidth < 1 {
		fieldWidth = 1
	}

	rowOf := func(t int) int { return (t - winStart) / 15 }

	type entryLayout struct {
		e         *planv1.PlanEntry
		topLine   int
		botLine   int // == topLine for single-row (15-min) entries
		snapStart int
		snapEnd   int
		labelRows []string // pre-wrapped label lines
	}

	layouts := make([]entryLayout, 0, len(entries))
	for _, e := range entries {
		sn := snapDown15(int(e.StartMinute))
		se := snapUp15(int(e.StartMinute) + int(e.DurationMinute))
		if se < sn+15 {
			se = sn + 15
		}
		tl := rowOf(sn)
		bl := rowOf(se)
		isSingle := bl == tl+1
		if isSingle {
			bl = tl
		}

		// Compute available rows for label.
		var labelRowCount int
		if isSingle {
			labelRowCount = 1
		} else {
			labelRowCount = bl - tl - 1 // interior rows
		}

		fullLabel := fmt.Sprintf("[%d] %02d:%02d-%02d:%02d %s", e.Id, sn/60, sn%60, se/60, se%60, e.Name)
		rows := wrapLabel(fullLabel, fieldWidth, labelRowCount)

		layouts = append(layouts, entryLayout{e, tl, bl, sn, se, rows})
	}

	// Build per-line lookup maps.
	totalLines := (winEnd-winStart)/15 + 1
	topAt := make(map[int]*entryLayout, len(layouts))
	botAt := make(map[int]*entryLayout, len(layouts))
	singleAt := make(map[int]*entryLayout, len(layouts))
	interiorAt := make(map[int]*entryLayout, len(layouts)*4)

	for i := range layouts {
		l := &layouts[i]
		if l.topLine == l.botLine {
			singleAt[l.topLine] = l
		} else {
			topAt[l.topLine] = l
			botAt[l.botLine] = l
			for inner := l.topLine + 1; inner < l.botLine; inner++ {
				interiorAt[inner] = l
			}
		}
	}

	// Compute now-marker row (-1 = no marker).
	nowLine := -1
	if now.Format("2006-01-02") == day {
		nowMin := now.Hour()*60 + now.Minute()
		snappedNow := snapDown15(nowMin)
		if snappedNow >= winStart && snappedNow < winEnd {
			nowLine = rowOf(snappedNow)
		}
	}

	hLight := strings.Repeat("─", fieldWidth)
	hHeavy := strings.Repeat("━", fieldWidth)

	var sb strings.Builder
	for L := 0; L < totalLines; L++ {
		t := winStart + L*15
		isHour := t%60 == 0

		// Build gutter (6 display columns).
		var gutter string
		if isHour {
			label := fmt.Sprintf("%02d:%02d", t/60, t%60)
			if L == nowLine {
				gutter = label + "▶"
			} else {
				gutter = label + " "
			}
		} else {
			if L == nowLine {
				gutter = "     ▶"
			} else {
				gutter = "      "
			}
		}

		top := topAt[L]
		bot := botAt[L]
		single := singleAt[L]
		interior := interiorAt[L]

		switch {
		case interior != nil:
			// Interior row of a multi-row entry.
			rowIdx := L - interior.topLine - 1
			label := ""
			if rowIdx < len(interior.labelRows) {
				label = interior.labelRows[rowIdx]
			}
			content := padRight(label, fieldWidth)
			content = applyCompletion(content, interior.e, isTTY)
			sb.WriteString(gutter + "┃" + content + "┃\n")

		case top != nil && bot != nil:
			// Shared border: multi-row entry A ends here, entry B starts here.
			sb.WriteString(gutter + "┣" + hHeavy + "┫\n")

		case bot != nil && single != nil:
			// Shared: multi-row entry ends here AND single-row entry starts here.
			label := singleLabelContent(single.labelRows, fieldWidth)
			label = applyCompletion(label, single.e, isTTY)
			sb.WriteString(gutter + "┣" + label + "┫\n")

		case top != nil:
			sb.WriteString(gutter + "┏" + hHeavy + "┓\n")

		case bot != nil:
			sb.WriteString(gutter + "┗" + hHeavy + "┛\n")

		case single != nil:
			label := singleLabelContent(single.labelRows, fieldWidth)
			label = applyCompletion(label, single.e, isTTY)
			sb.WriteString(gutter + "┣" + label + "┫\n")

		default:
			if isHour {
				sb.WriteString(gutter + "├" + hLight + "┤\n")
			} else {
				sb.WriteString(gutter + "│" + strings.Repeat(" ", fieldWidth) + "│\n")
			}
		}
	}
	return sb.String()
}

// singleLabelContent returns the field content for a single-row entry, padded with heavy chars.
func singleLabelContent(labelRows []string, fieldWidth int) string {
	label := ""
	if len(labelRows) > 0 {
		label = labelRows[0]
	}
	n := utf8.RuneCountInString(label)
	if n >= fieldWidth {
		return truncateRunes(label, fieldWidth)
	}
	return label + strings.Repeat("━", fieldWidth-n)
}

// applyCompletion wraps content with dim+strikethrough codes when entry is completed and isTTY.
func applyCompletion(content string, e *planv1.PlanEntry, isTTY bool) string {
	if e.Completed && isTTY {
		return DimStrike(content)
	}
	return content
}

// wrapLabel wraps text into at most maxRows lines of at most width runes each.
// The last row is truncated with "..." if needed.
func wrapLabel(s string, width int, maxRows int) []string {
	if maxRows <= 0 || width <= 0 {
		return nil
	}
	var rows []string
	for utf8.RuneCountInString(s) > 0 && len(rows) < maxRows {
		if utf8.RuneCountInString(s) <= width {
			rows = append(rows, s)
			break
		}
		if len(rows) == maxRows-1 {
			// Last available row: truncate.
			rows = append(rows, truncateRunes(s, width))
			break
		}
		// Find a word-break point within width.
		runes := []rune(s)
		cut := width
		for i := width - 1; i > 0; i-- {
			if runes[i] == ' ' {
				cut = i
				break
			}
		}
		rows = append(rows, string(runes[:cut]))
		s = strings.TrimLeft(string(runes[cut:]), " ")
	}
	return rows
}

// padRight pads s with spaces to exactly width runes, or truncates with "...".
func padRight(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n < width {
		return s + strings.Repeat(" ", width-n)
	}
	if n > width {
		return truncateRunes(s, width)
	}
	return s
}

// truncateRunes truncates s to width runes, appending "..." if truncation occurs.
func truncateRunes(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width >= 3 {
		return string(runes[:width-3]) + "..."
	}
	return string(runes[:width])
}

// snapDown15 rounds min down to the nearest 15-minute boundary.
func snapDown15(min int) int {
	return (min / 15) * 15
}

// snapUp15 rounds min up to the nearest 15-minute boundary.
func snapUp15(min int) int {
	return ((min + 14) / 15) * 15
}
