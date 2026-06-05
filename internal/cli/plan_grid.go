package cli

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
)

// GridOptions controls optional rendering behaviour for RenderGrid.
type GridOptions struct {
	HideID         bool                // omit the "[id] " prefix from entry labels
	SelectedID     int32               // highlight this entry's rows (0 = none); applied only when isTTY
	Styled         bool                // when true (and isTTY): use accent-color highlight instead of plain bold
	SelectionStyle func(string) string // when non-nil, styles the selected entry's content cell only; nil = fallback plain behavior

	// WindowStartMin, when non-nil, fixes the grid's first minute (minutes since
	// midnight) and disables automatic backward expansion of the window start.
	WindowStartMin *int
	// WindowEndMin, when non-nil, fixes the grid's last minute and disables
	// automatic forward expansion of the window end.
	WindowEndMin *int

	// PreviewID, when non-zero, marks the entry (by Id) that is an unsaved,
	// in-progress preview. Its box is drawn with dashed runes instead of solid.
	PreviewID int32

	// PreviewStyle, when non-nil, styles the preview entry's non-conflicting
	// runes (e.g. a dim foreground). When nil, the dashed runes are emitted
	// unstyled (plain/non-TTY mode still distinguishes the preview structurally).
	PreviewStyle func(string) string

	// ConflictStyle, when non-nil, styles the preview rows that fall in
	// PreviewConflictSlots (e.g. a red foreground). When nil, the renderer
	// applies the plain-mode fallback marker (see contracts/grid-preview.md §4).
	ConflictStyle func(string) string

	// PreviewConflictSlots holds the slot-start minutes (multiples of 15) of the
	// preview entry that overlap another timed entry. Empty/nil ⇒ no conflict.
	PreviewConflictSlots map[int]bool
}

// RenderGrid renders a day's plan entries as a calendar grid.
// Rows represent 15-minute slots; hour boundaries get a horizontal divider.
// The visible window defaults to 08:00–17:00 and is extended outward (rounded
// to the nearest hour) to contain every entry's snapped span.
func RenderGrid(entries []*planv1.PlanEntry, day string, now time.Time, width int, isTTY bool, opts GridOptions) string {
	winStart := 8 * 60
	winEnd := 17 * 60

	if opts.WindowStartMin != nil {
		winStart = *opts.WindowStartMin
	}
	if opts.WindowEndMin != nil {
		winEnd = *opts.WindowEndMin
	}

	for _, e := range entries {
		sn := snapDown15(int(e.GetStartMinute()))
		se := snapUp15(int(e.GetStartMinute()) + int(e.DurationMinute))
		if opts.WindowStartMin == nil {
			if h := (sn / 60) * 60; h < winStart {
				winStart = h
			}
		}
		if opts.WindowEndMin == nil {
			if h := ((se + 59) / 60) * 60; h > winEnd {
				winEnd = h
			}
		}
	}

	// gutter(7) + leftRail(1) + leftPad(1) + rightPad(1) + rightRail(1) = 11
	boxWidth := width - 11
	if boxWidth < 1 {
		boxWidth = 1
	}
	contentWidth := boxWidth - 2
	if contentWidth < 0 {
		contentWidth = 0
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
		sn := snapDown15(int(e.GetStartMinute()))
		se := snapUp15(int(e.GetStartMinute()) + int(e.DurationMinute))
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

		var fullLabel string
		if opts.HideID {
			fullLabel = fmt.Sprintf("%02d:%02d-%02d:%02d %s", sn/60, sn%60, se/60, se%60, e.Name)
		} else {
			fullLabel = fmt.Sprintf("[%d] %02d:%02d-%02d:%02d %s", e.Id, sn/60, sn%60, se/60, se%60, e.Name)
		}
		rows := wrapLabel(fullLabel, contentWidth, labelRowCount)

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

	hLight := strings.Repeat("─", boxWidth+2)   // fills inner region for empty hour rows
	hHeavy := strings.Repeat("━", contentWidth) // fills between heavy box corners
	hDash := strings.Repeat("┅", contentWidth)  // dashed fill for preview entry borders

	// isPreviewEntry reports whether e is the transient preview entry.
	isPreviewEntry := func(e *planv1.PlanEntry) bool {
		return opts.PreviewID != 0 && e != nil && e.Id == opts.PreviewID
	}

	// previewBoxH returns the horizontal fill for a box border: dashed for preview, solid otherwise.
	previewBoxH := func(e *planv1.PlanEntry) string {
		if isPreviewEntry(e) {
			return hDash
		}
		return hHeavy
	}

	// previewBoxV returns the vertical border rune string: dashed for preview, solid otherwise.
	previewBoxV := func(e *planv1.PlanEntry) string {
		if isPreviewEntry(e) {
			return "┇"
		}
		return "┃"
	}

	// applyPreviewStyle wraps s with the appropriate style for a preview row.
	// slotMinute is the slot-start minute (winStart + L*15) for the current grid row.
	applyPreviewStyle := func(s string, e *planv1.PlanEntry, slotMinute int) string {
		if !isPreviewEntry(e) {
			return s
		}
		if opts.PreviewConflictSlots[slotMinute] && opts.ConflictStyle != nil {
			return opts.ConflictStyle(s)
		}
		if opts.PreviewStyle != nil {
			return opts.PreviewStyle(s)
		}
		return s
	}

	var sb strings.Builder
	for L := 0; L < totalLines; L++ {
		t := winStart + L*15
		isHour := t%60 == 0

		// Look up entry layouts for this row (needed before gutter for conflict detection).
		top := topAt[L]
		bot := botAt[L]
		single := singleAt[L]
		interior := interiorAt[L]

		// Determine whether this row needs a plain-mode conflict gutter marker.
		isPreviewConflictRow := false
		if opts.PreviewID != 0 && opts.PreviewConflictSlots[t] && opts.ConflictStyle == nil {
			switch {
			case interior != nil:
				// Also check top: preview may start at the same row as another entry's interior.
				isPreviewConflictRow = isPreviewEntry(interior.e) || (top != nil && isPreviewEntry(top.e))
			case top != nil && bot != nil:
				isPreviewConflictRow = isPreviewEntry(top.e) || isPreviewEntry(bot.e)
			case bot != nil && single != nil:
				isPreviewConflictRow = isPreviewEntry(bot.e) || isPreviewEntry(single.e)
			case top != nil:
				isPreviewConflictRow = isPreviewEntry(top.e)
			case bot != nil:
				isPreviewConflictRow = isPreviewEntry(bot.e)
			case single != nil:
				isPreviewConflictRow = isPreviewEntry(single.e)
			}
		}

		// Build gutter (7 display columns): label + space + marker-col.
		// Conflict ! takes precedence over now-marker ▶ per contract §4.
		var gutter string
		markerChar := " "
		if L == nowLine {
			markerChar = "▶"
		}
		if isPreviewConflictRow {
			markerChar = "!"
		}
		if isHour {
			label := fmt.Sprintf("%02d:%02d", t/60, t%60)
			gutter = label + " " + markerChar
		} else {
			gutter = "      " + markerChar
		}

		// Rail and padding chars: hour rows use ├/┤ and ─; non-hour use │ and space.
		var leftRail, rightRail, padChar string
		if isHour {
			leftRail = "├"
			rightRail = "┤"
			padChar = "─"
		} else {
			leftRail = "│"
			rightRail = "│"
			padChar = " "
		}

		var line string
		switch {
		case interior != nil:
			// When the preview starts at this same row (its top border coincides with
			// another entry's interior), render the preview's top border so conflict
			// styling is applied at the correct conflicting slot.
			if top != nil && isPreviewEntry(top.e) {
				h := previewBoxH(top.e)
				inner := applyPreviewStyle("┏"+h+"┓", top.e, t)
				line = gutter + leftRail + padChar + inner + padChar + rightRail
				break
			}
			// Interior row of a multi-row entry.
			rowIdx := L - interior.topLine - 1
			label := ""
			if rowIdx < len(interior.labelRows) {
				label = interior.labelRows[rowIdx]
			}
			content := padRight(label, contentWidth)
			content = applyCompletion(content, interior.e, isTTY)
			v := previewBoxV(interior.e)
			if isTTY && opts.SelectedID != 0 && interior.e.Id == opts.SelectedID {
				if opts.SelectionStyle != nil {
					// Include border walls in the styled region.
					line = gutter + leftRail + padChar + opts.SelectionStyle(v+content+v) + padChar + rightRail
				} else {
					line = gutter + leftRail + padChar + v + content + v + padChar + rightRail
					line = accentOpen(opts) + line + "\x1b[0m"
				}
			} else {
				inner := applyPreviewStyle(v+content+v, interior.e, t)
				line = gutter + leftRail + padChar + inner + padChar + rightRail
			}

		case top != nil && bot != nil:
			// Shared border: multi-row entry A ends here, entry B starts here.
			// Use dashed fill if either entry is the preview.
			hFill := hHeavy
			if isPreviewEntry(top.e) || isPreviewEntry(bot.e) {
				hFill = hDash
			}
			// Determine which entry drives the preview style (prefer the preview one).
			var previewE *planv1.PlanEntry
			if isPreviewEntry(top.e) {
				previewE = top.e
			} else if isPreviewEntry(bot.e) {
				previewE = bot.e
			}
			isSelected := isTTY && opts.SelectedID != 0 && (top.e.Id == opts.SelectedID || bot.e.Id == opts.SelectedID)
			if isSelected && opts.SelectionStyle != nil {
				line = gutter + leftRail + padChar + opts.SelectionStyle("┣"+hFill+"┫") + padChar + rightRail
			} else {
				inner := "┣" + hFill + "┫"
				if previewE != nil {
					inner = applyPreviewStyle(inner, previewE, t)
				}
				line = gutter + leftRail + padChar + inner + padChar + rightRail
				if isSelected {
					line = accentOpen(opts) + line + "\x1b[0m"
				}
			}

		case bot != nil && single != nil:
			// Shared: multi-row entry ends here AND single-row entry starts here.
			isPreviewSingle := isPreviewEntry(single.e)
			padRune := "━"
			if isPreviewSingle {
				padRune = "┅"
			}
			label := singleLabelContentPad(single.labelRows, contentWidth, padRune)
			label = applyCompletion(label, single.e, isTTY)
			isSelected := isTTY && opts.SelectedID != 0 && (bot.e.Id == opts.SelectedID || single.e.Id == opts.SelectedID)
			if isSelected && opts.SelectionStyle != nil {
				line = gutter + leftRail + padChar + "┣" + opts.SelectionStyle(label) + "┫" + padChar + rightRail
			} else {
				inner := "┣" + label + "┫"
				if isPreviewSingle {
					inner = applyPreviewStyle(inner, single.e, t)
				}
				line = gutter + leftRail + padChar + inner + padChar + rightRail
				if isSelected {
					line = accentOpen(opts) + line + "\x1b[0m"
				}
			}

		case top != nil:
			h := previewBoxH(top.e)
			isSelected := isTTY && opts.SelectedID != 0 && top.e.Id == opts.SelectedID
			if isSelected && opts.SelectionStyle != nil {
				line = gutter + leftRail + padChar + opts.SelectionStyle("┏"+h+"┓") + padChar + rightRail
			} else {
				inner := applyPreviewStyle("┏"+h+"┓", top.e, t)
				line = gutter + leftRail + padChar + inner + padChar + rightRail
				if isSelected {
					line = accentOpen(opts) + line + "\x1b[0m"
				}
			}

		case bot != nil:
			h := previewBoxH(bot.e)
			isSelected := isTTY && opts.SelectedID != 0 && bot.e.Id == opts.SelectedID
			if isSelected && opts.SelectionStyle != nil {
				line = gutter + leftRail + padChar + opts.SelectionStyle("┗"+h+"┛") + padChar + rightRail
			} else {
				inner := applyPreviewStyle("┗"+h+"┛", bot.e, t)
				line = gutter + leftRail + padChar + inner + padChar + rightRail
				if isSelected {
					line = accentOpen(opts) + line + "\x1b[0m"
				}
			}

		case single != nil:
			isPreviewSingle := isPreviewEntry(single.e)
			padRune := "━"
			if isPreviewSingle {
				padRune = "┅"
			}
			label := singleLabelContentPad(single.labelRows, contentWidth, padRune)
			label = applyCompletion(label, single.e, isTTY)
			if isTTY && opts.SelectedID != 0 && single.e.Id == opts.SelectedID && opts.SelectionStyle != nil {
				// Style only the content cell, not the border rails.
				line = gutter + leftRail + padChar + "┣" + opts.SelectionStyle(label) + "┫" + padChar + rightRail
			} else {
				inner := "┣" + label + "┫"
				inner = applyPreviewStyle(inner, single.e, t)
				line = gutter + leftRail + padChar + inner + padChar + rightRail
				line = applySelection(line, single.e.Id, opts, isTTY)
			}

		default:
			if isHour {
				line = gutter + "├" + hLight + "┤"
			} else {
				line = gutter + "│" + strings.Repeat(" ", boxWidth+2) + "│"
			}
		}
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

// renderBoxLine renders a single line of an entry's box (interior ┃content┃ row)
// with completion and selection styling applied. Used by both RenderGrid and RenderUntimed.
func renderBoxLine(content string, e *planv1.PlanEntry, isTTY bool, opts GridOptions) string {
	content = applyCompletion(content, e, isTTY)
	if isTTY && opts.SelectedID != 0 && e.Id == opts.SelectedID && opts.SelectionStyle != nil {
		return opts.SelectionStyle("┃" + content + "┃")
	}
	return "┃" + content + "┃"
}

// RenderUntimed renders untimed plan entries as stacked boxes above the day grid,
// using the same box geometry as RenderGrid (┏┓ top, ┃┃ interior, ┣┫ shared boundary,
// ┗┛ bottom). Adjacent multi-row entries share a ┣┫ boundary. Returns an empty string
// when entries is empty.
func RenderUntimed(entries []*planv1.PlanEntry, width int, isTTY bool, opts GridOptions) string {
	if len(entries) == 0 {
		return ""
	}

	// Same geometry constants as RenderGrid.
	boxWidth := width - 11 // gutter(7)+leftRail(1)+leftPad(1)+rightPad(1)+rightRail(1)
	if boxWidth < 1 {
		boxWidth = 1
	}
	contentWidth := boxWidth - 2
	if contentWidth < 0 {
		contentWidth = 0
	}
	hHeavy := strings.Repeat("━", contentWidth)
	const gutter = "       "

	var sb strings.Builder
	skipTop := false // true when the previous multi-row entry emitted a shared ┣┫ as our top

	for i, e := range entries {
		rows := int(e.DurationMinute) / 15
		if rows < 1 {
			rows = 1
		}
		isMulti := rows > 1
		isLast := i == len(entries)-1
		isSelected := isTTY && opts.SelectedID != 0 && e.Id == opts.SelectedID

		var labelText string
		if opts.HideID {
			labelText = e.Name
		} else {
			labelText = fmt.Sprintf("[%d] %s", e.Id, e.Name)
		}

		if !isMulti {
			// Single-row: ┣label━━━┫ (same as grid's single-row entry).
			skipTop = false
			label := singleLabelContent([]string{labelText}, contentWidth)
			label = applyCompletion(label, e, isTTY)
			var line string
			if isSelected && opts.SelectionStyle != nil {
				line = gutter + "│ ┣" + opts.SelectionStyle(label) + "┫ │"
			} else {
				line = gutter + "│ ┣" + label + "┫ │"
				line = applySelection(line, e.Id, opts, isTTY)
			}
			sb.WriteString(line + "\n")
			continue
		}

		// Multi-row entry: ┏┓ top + (rows-1) interior rows + ┗┛/┣┫ bottom.

		// Top border (skipped when the previous entry's shared bottom serves as our top).
		if !skipTop {
			if isSelected && opts.SelectionStyle != nil {
				line := gutter + "│ " + opts.SelectionStyle("┏"+hHeavy+"┓") + " │"
				sb.WriteString(line + "\n")
			} else {
				line := gutter + "│ ┏" + hHeavy + "┓ │"
				line = applySelection(line, e.Id, opts, isTTY)
				sb.WriteString(line + "\n")
			}
		}
		skipTop = false

		// Interior rows (rows-1 of them); label wrapped across as many as fit.
		labelRows := wrapLabel(labelText, contentWidth, rows-1)
		for r := 0; r < rows-1; r++ {
			lbl := ""
			if r < len(labelRows) {
				lbl = labelRows[r]
			}
			content := padRight(lbl, contentWidth)
			boxed := renderBoxLine(content, e, isTTY, opts)
			var line string
			if isSelected && opts.SelectionStyle != nil {
				line = gutter + "│ " + boxed + " │"
			} else {
				line = gutter + "│ " + boxed + " │"
				line = applySelection(line, e.Id, opts, isTTY)
			}
			sb.WriteString(line + "\n")
		}

		// Bottom border: ┗┛ standalone, or ┣┫ shared with the next multi-row entry.
		nextIsMulti := !isLast && int(entries[i+1].DurationMinute)/15 > 1
		if nextIsMulti {
			nextE := entries[i+1]
			nextSelected := isTTY && opts.SelectedID != 0 && nextE.Id == opts.SelectedID
			isEitherSelected := isSelected || nextSelected
			var line string
			if isEitherSelected && opts.SelectionStyle != nil {
				line = gutter + "│ " + opts.SelectionStyle("┣"+hHeavy+"┫") + " │"
			} else {
				line = gutter + "│ ┣" + hHeavy + "┫ │"
				if isEitherSelected {
					line = accentOpen(opts) + line + "\x1b[0m"
				}
			}
			sb.WriteString(line + "\n")
			skipTop = true // next entry's top is already provided by this ┣┫
		} else {
			if isSelected && opts.SelectionStyle != nil {
				line := gutter + "│ " + opts.SelectionStyle("┗"+hHeavy+"┛") + " │"
				sb.WriteString(line + "\n")
			} else {
				line := gutter + "│ ┗" + hHeavy + "┛ │"
				line = applySelection(line, e.Id, opts, isTTY)
				sb.WriteString(line + "\n")
			}
			skipTop = false
		}
	}
	return sb.String()
}

// RenderUntimedSeparator returns a single separator line between the untimed pane and the day
// grid, using the grid's hour-divider style. Returns an empty string when untimedIDs is empty
// (no untimed entries), so the no-untimed layout is byte-for-byte unchanged.
func RenderUntimedSeparator(untimedIDs []int, width int) string {
	if len(untimedIDs) == 0 {
		return ""
	}
	// Same geometry as the grid's hour-divider: gutter(7) + ├ + ─×(width-9) + ┤
	fill := width - 9
	if fill < 0 {
		fill = 0
	}
	return "       ├" + strings.Repeat("─", fill) + "┤\n"
}

// singleLabelContent returns the field content for a single-row entry, padded with heavy chars.
func singleLabelContent(labelRows []string, fieldWidth int) string {
	return singleLabelContentPad(labelRows, fieldWidth, "━")
}

// singleLabelContentPad returns the field content for a single-row entry, padded with padStr.
// padStr should be a single-rune string (e.g. "━" or "┅").
func singleLabelContentPad(labelRows []string, fieldWidth int, padStr string) string {
	label := ""
	if len(labelRows) > 0 {
		label = labelRows[0]
	}
	n := utf8.RuneCountInString(label)
	if n >= fieldWidth {
		return truncateRunes(label, fieldWidth)
	}
	return label + strings.Repeat(padStr, fieldWidth-n)
}

// applyCompletion wraps content with dim+strikethrough codes when entry is completed and isTTY.
func applyCompletion(content string, e *planv1.PlanEntry, isTTY bool) string {
	if e.Completed && isTTY {
		return DimStrike(content)
	}
	return content
}

// accentOpen returns the ANSI open code for the selection highlight: accent blue
// when Styled, plain bold otherwise.
func accentOpen(opts GridOptions) string {
	if opts.Styled {
		return "\x1b[94m" // bright blue — matches terminal accent palette
	}
	return "\x1b[1m"
}

// applySelection wraps a rendered line with highlight ANSI codes when the entry
// id matches opts.SelectedID and isTTY is true. When opts.Styled is true, an
// accent-color foreground is applied; otherwise plain bold is used.
func applySelection(line string, id int32, opts GridOptions, isTTY bool) string {
	if isTTY && opts.SelectedID != 0 && id == opts.SelectedID {
		return accentOpen(opts) + line + "\x1b[0m"
	}
	return line
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

// baseWindowFor returns the entry-extended default window [baseStart, baseEnd]
// (minutes since midnight, rounded to hour boundaries) that RenderGrid uses
// when WindowStartMin and WindowEndMin are both nil.
func baseWindowFor(entries []*planv1.PlanEntry) (baseStart, baseEnd int) {
	baseStart = 8 * 60
	baseEnd = 17 * 60
	for _, e := range entries {
		sn := snapDown15(int(e.GetStartMinute()))
		se := snapUp15(int(e.GetStartMinute()) + int(e.DurationMinute))
		if h := (sn / 60) * 60; h < baseStart {
			baseStart = h
		}
		if h := ((se + 59) / 60) * 60; h > baseEnd {
			baseEnd = h
		}
	}
	return
}

// GridWindow returns the [startMin, endMin] window (minutes since midnight,
// 15-minute aligned) the TUI should pass to RenderGrid so the timed grid fills
// availableRows of vertical space.
//
//   - entries:       the day's timed plan entries
//   - now:           current time (for the anchor block and today detection)
//   - day:           the in-view day, "YYYY-MM-DD"
//   - availableRows: rows the timed grid may occupy (>= 1)
//
// Modes:
//
//	FILL          availableRows >= baseRows           → start=baseStart, fill later
//	ANCHOR        constrained & today & now>=baseStart → start=now-block, fill later
//	TOP-TRUNCATE  constrained & (not today | early)   → start=baseStart, fill later
//
// end is always clamped to 24:00 (1440). The CLI does not call this helper.
func GridWindow(entries []*planv1.PlanEntry, now time.Time, day string, availableRows int) (startMin, endMin int) {
	baseStart, baseEnd := baseWindowFor(entries)
	baseRows := (baseEnd-baseStart)/15 + 1

	var start int
	if availableRows >= baseRows {
		// FILL: keep the default start, extend the end to use available rows.
		start = baseStart
	} else {
		// Constrained.
		if now.Format("2006-01-02") == day {
			nowBlock := snapDown15(now.Hour()*60 + now.Minute())
			if nowBlock >= baseStart {
				// ANCHOR: start the grid at the current-time block.
				start = nowBlock
			} else {
				// now is before the window; TOP-TRUNCATE from default start.
				start = baseStart
			}
		} else {
			// Not today: TOP-TRUNCATE from default start.
			start = baseStart
		}
	}

	end := start + (availableRows-1)*15
	if end > 1440 {
		end = 1440
	}
	return start, end
}
