package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
	boxes := planBoxes(entries)

	winStart := 8 * 60
	winEnd := 17 * 60

	if opts.WindowStartMin != nil {
		winStart = *opts.WindowStartMin
	}
	if opts.WindowEndMin != nil {
		winEnd = *opts.WindowEndMin
	}

	for _, b := range boxes {
		if opts.WindowStartMin == nil {
			if h := (b.snapStart / 60) * 60; h < winStart {
				winStart = h
			}
		}
		if opts.WindowEndMin == nil {
			if h := ((b.snapEnd + 59) / 60) * 60; h > winEnd {
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

	layouts := make([]entryLayout, 0, len(boxes))
	for _, b := range boxes {
		tl := rowOf(b.snapStart)
		bl := rowOf(b.snapEnd)
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

		// Labels print the entry's exact minutes, not the snapped box span: the
		// box may have been trimmed or pushed to clear a neighbour, but the time
		// the user typed must survive that.
		es, ee := b.exactStart, b.exactEnd

		var fullLabel string
		if opts.HideID {
			fullLabel = fmt.Sprintf("%02d:%02d-%02d:%02d %s", es/60, es%60, ee/60, ee%60, b.e.Name)
		} else {
			fullLabel = fmt.Sprintf("[%d] %02d:%02d-%02d:%02d %s", b.e.Id, es/60, es%60, ee/60, ee%60, b.e.Name)
		}
		rows := wrapLabel(fullLabel, contentWidth, labelRowCount)

		layouts = append(layouts, entryLayout{b.e, tl, bl, b.snapStart, b.snapEnd, rows})
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
	hDash := strings.Repeat("╍", contentWidth)  // dashed fill for preview entry borders

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
			return "╏"
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
				// When the preview box meets the shared divider, use light tees (├/┤)
				// so the boundary reads as tentative; heavy (┣/┫) when both are saved.
				leftTee, rightTee := "┣", "┫"
				if previewE != nil {
					leftTee, rightTee = "├", "┤"
				}
				inner := leftTee + hFill + rightTee
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
				padRune = "╍"
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
				padRune = "╍"
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

// RenderUntimed renders untimed plan entries as checkbox+name list rows above the
// day grid. Each entry produces exactly one line:
//
//	styled (isTTY=true):  "  ☐ name" / "  ☑ name"
//	plain (isTTY=false):  "  name" (no glyph)
//
// Returns "" when entries is empty.
func RenderUntimed(entries []*planv1.PlanEntry, width int, isTTY bool, opts GridOptions) string {
	if len(entries) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, e := range entries {
		prefixLen := 2 // "  "
		if isTTY {
			prefixLen = 4 // "  ☐ "
		}
		availWidth := width - prefixLen
		if availWidth < 1 {
			availWidth = 1
		}

		// Truncate by display width (not rune count) to match the Tasks tab's
		// renderList, so wide runes (CJK, emoji) can't push the row past width.
		name := e.Name
		if lipgloss.Width(name) > availWidth {
			name = ansi.Truncate(name, availWidth, "")
		}
		// Note: applyCompletion's DimStrike embeds a trailing \x1b[0m reset. That's
		// safe today because name is always the last element wrapped by
		// SelectionStyle below; if a trailing marker is ever appended after name,
		// the reset would need to move outside the selection wrap first.
		name = applyCompletion(name, e, isTTY)

		var checkbox string
		if isTTY {
			if e.Completed {
				checkbox = "☑"
			} else {
				checkbox = "☐"
			}
		}

		content := name
		if isTTY {
			content = checkbox + " " + name
		}

		isSelected := isTTY && opts.SelectedID != 0 && e.Id == opts.SelectedID
		var line string
		if isSelected && opts.SelectionStyle != nil {
			line = "  " + opts.SelectionStyle(content)
		} else {
			line = applySelection("  "+content, e.Id, opts, isTTY)
		}

		sb.WriteString(line + "\n")
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

// AutoScheduleSlot returns the earliest start minute (minute-of-day) at or
// after floorMin where a block of the needed length fits within a single day
// (ending at 1440) without overlapping any timed entry.
//
// Entries with Id == excludeID are ignored (the entry being moved).
// When durationMin <= 0 a 30-minute block is used for fitting.
// Returns ok == false when no qualifying slot exists before 1440.
func AutoScheduleSlot(timed []*planv1.PlanEntry, durationMin, floorMin int, excludeID int32) (startMin int, ok bool) {
	needed := durationMin
	if needed <= 0 {
		needed = 30
	}

	merged := mergedObstacles(timed, excludeID)

	// Scan free intervals from floorMin to 1440.
	freeStart := floorMin
	for _, iv := range merged {
		if iv.e <= freeStart {
			continue // obstacle ends before our search window
		}
		gapStart := ceil15(freeStart)
		gapEnd := iv.s
		if gapEnd > gapStart && gapEnd-gapStart >= needed {
			return gapStart, true
		}
		if iv.e > freeStart {
			freeStart = iv.e
		}
	}
	// Check tail after all obstacles.
	tailStart := ceil15(freeStart)
	if 1440-tailStart >= needed {
		return tailStart, true
	}
	return 0, false
}

// ceil15 rounds min up to the nearest 15-minute boundary.
func ceil15(min int) int {
	return ((min + 14) / 15) * 15
}

// obsInterval is a half-open [s, e) minute-of-day obstacle range.
type obsInterval struct{ s, e int }

// mergedObstacles builds the sorted, overlap-/adjacency-unioned list of timed
// entries as [start, end) intervals. Entries with Id == excludeID (the entry
// being moved) and entries without a start minute are ignored.
func mergedObstacles(timed []*planv1.PlanEntry, excludeID int32) []obsInterval {
	var obs []obsInterval
	for _, e := range timed {
		if excludeID != 0 && e.Id == excludeID {
			continue
		}
		if e.StartMinute == nil {
			continue
		}
		s := int(e.GetStartMinute())
		end := s + int(e.DurationMinute)
		obs = append(obs, obsInterval{s, end})
	}
	// Sort by start.
	for i := 1; i < len(obs); i++ {
		for j := i; j > 0 && obs[j].s < obs[j-1].s; j-- {
			obs[j], obs[j-1] = obs[j-1], obs[j]
		}
	}
	// Merge overlapping/adjacent intervals.
	var merged []obsInterval
	for _, iv := range obs {
		if len(merged) > 0 && iv.s < merged[len(merged)-1].e {
			if iv.e > merged[len(merged)-1].e {
				merged[len(merged)-1].e = iv.e
			}
		} else {
			merged = append(merged, iv)
		}
	}
	return merged
}

// NextGapFloor returns the end minute of the earliest timed entry that begins
// at or after fromMin — the lower bound of the next free gap. ok is false when
// no entry begins at or after fromMin.
//
// Entries with Id == excludeID are ignored (the entry being moved).
func NextGapFloor(timed []*planv1.PlanEntry, fromMin int, excludeID int32) (floorMin int, ok bool) {
	// Return the end of the first merged interval whose start >= fromMin.
	for _, iv := range mergedObstacles(timed, excludeID) {
		if iv.s >= fromMin {
			return iv.e, true
		}
	}
	return 0, false
}

// planBox is an entry's exact interval together with the snapped grid span its
// box occupies. Labels are drawn from the exact minutes; geometry from the
// snapped ones.
type planBox struct {
	e          *planv1.PlanEntry
	exactStart int
	exactEnd   int
	snapStart  int
	snapEnd    int
}

// planBoxes returns the grid boxes for entries, sorted by snapped span with
// snapped-box collisions resolved. Callers may rely on no two returned boxes
// claiming the same grid row unless their exact intervals genuinely overlap.
func planBoxes(entries []*planv1.PlanEntry) []planBox {
	boxes := make([]planBox, 0, len(entries))
	for _, e := range entries {
		es := int(e.GetStartMinute())
		ee := es + int(e.DurationMinute)
		sn := snapDown15(es)
		se := snapUp15(ee)
		if se < sn+15 {
			se = sn + 15 // every entry gets at least one row
		}
		boxes = append(boxes, planBox{e: e, exactStart: es, exactEnd: ee, snapStart: sn, snapEnd: se})
	}

	// Entries arrive sorted by start from the server, but the TUI appends its
	// preview entry after that list — sort rather than assume.
	sort.SliceStable(boxes, func(i, j int) bool {
		if boxes[i].snapStart != boxes[j].snapStart {
			return boxes[i].snapStart < boxes[j].snapStart
		}
		return boxes[i].snapEnd < boxes[j].snapEnd
	})

	resolveBoxCollisions(boxes)
	return boxes
}

// resolveBoxCollisions adjusts snapped spans so that no two boxes claim the same
// grid row. Snapping the start down and the end up means two entries that merely
// touch in exact time (13:00–13:50 and 13:50–14:10) can still overlap by a row
// or two. The per-row layout maps hold one entry per row, so without this the
// loser silently forfeits a border — or disappears from the grid entirely.
//
// Entries whose exact intervals genuinely overlap are left alone: two saved
// entries can never overlap (the server rejects that), so the only real overlap
// is an unsaved preview sitting on top of a saved entry, and moving its box
// would hide the very conflict the red styling exists to show.
//
// boxes must be sorted by (snapStart, snapEnd).
func resolveBoxCollisions(boxes []planBox) {
	for i := 1; i < len(boxes); i++ {
		prev, cur := &boxes[i-1], &boxes[i]
		if cur.snapStart >= prev.snapEnd {
			continue // boxes already clear of each other
		}
		if cur.exactStart < prev.exactEnd {
			continue // a real overlap; leave it visible
		}
		// Prefer pulling the earlier box's end back to where the later one
		// starts: both keep their true position and share one divider row.
		if cur.snapStart >= prev.snapStart+15 {
			prev.snapEnd = cur.snapStart
			continue
		}
		// The earlier box would collapse to nothing (both entries start inside
		// the same slot), so push the later box down instead.
		cur.snapStart = prev.snapEnd
		if cur.snapEnd < cur.snapStart+15 {
			cur.snapEnd = cur.snapStart + 15
		}
	}
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
	// Use the same resolved boxes RenderGrid will draw, so a box that got pushed
	// down to clear a neighbour can't land outside the window the TUI sized.
	for _, b := range planBoxes(entries) {
		if h := (b.snapStart / 60) * 60; h < baseStart {
			baseStart = h
		}
		if h := ((b.snapEnd + 59) / 60) * 60; h > baseEnd {
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
