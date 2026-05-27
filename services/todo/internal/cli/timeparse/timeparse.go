package timeparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	re24h        = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)
	re24hNoColon = regexp.MustCompile(`^(\d{2})(\d{2})$`)
	re12h        = regexp.MustCompile(`^(\d{1,2}):(\d{2})\s*(am|pm)$`)
	reDurH  = regexp.MustCompile(`^(\d+)h$`)
	reDurM  = regexp.MustCompile(`^(\d+)m$`)
	reDurHM = regexp.MustCompile(`^(\d+)h(\d+)m$`)
)

// ParseStart parses a time-of-day string and returns minutes since midnight.
// Accepted formats: "13:15" (24-hour), "1:15pm" / "1:15 pm" / "01:15 PM" (12-hour).
func ParseStart(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("start time is required")
	}

	// Try 24-hour first.
	if m := re24h.FindStringSubmatch(s); m != nil {
		h, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		if h > 23 {
			return 0, fmt.Errorf("hour %d out of range (0-23)", h)
		}
		if min > 59 {
			return 0, fmt.Errorf("minute %d out of range (0-59)", min)
		}
		return h*60 + min, nil
	}

	// Try 4-digit no-colon 24-hour (e.g. "0815", "1400").
	if m := re24hNoColon.FindStringSubmatch(s); m != nil {
		h, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		if h > 23 {
			return 0, fmt.Errorf("hour %d out of range (0-23)", h)
		}
		if min > 59 {
			return 0, fmt.Errorf("minute %d out of range (0-59)", min)
		}
		return h*60 + min, nil
	}

	// Try 12-hour (collapse space and lowercase meridiem).
	normalized := strings.ToLower(strings.ReplaceAll(s, " ", ""))
	if m := re12h.FindStringSubmatch(normalized); m != nil {
		h, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		mer := m[3]
		if h < 1 || h > 12 {
			return 0, fmt.Errorf("hour %d out of range for 12-hour format (1-12)", h)
		}
		if min > 59 {
			return 0, fmt.Errorf("minute %d out of range (0-59)", min)
		}
		if mer == "am" {
			if h == 12 {
				h = 0
			}
		} else {
			if h != 12 {
				h += 12
			}
		}
		return h*60 + min, nil
	}

	return 0, fmt.Errorf("unrecognized time format %q; use HH:MM, HHMM, or H:MMam/pm", s)
}

// ParseDuration parses a duration string and returns minutes.
// Accepted formats: "90m", "1h", "2h", "1h30m", "0h45m".
func ParseDuration(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("duration is required")
	}
	s = strings.ToLower(s)

	if m := reDurHM.FindStringSubmatch(s); m != nil {
		h, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		return h*60 + min, nil
	}
	if m := reDurH.FindStringSubmatch(s); m != nil {
		h, _ := strconv.Atoi(m[1])
		return h * 60, nil
	}
	if m := reDurM.FindStringSubmatch(s); m != nil {
		min, _ := strconv.Atoi(m[1])
		return min, nil
	}

	return 0, fmt.Errorf("unrecognized duration format %q; use Nh, Nm, or NhMm", s)
}

// ParseDurationOrEnd parses either a duration ("90m", "1h30m") or an end
// time-of-day ("15:00", "1500", "3:00pm"), returning the duration in minutes.
// startMinute is required when an end time is given, to compute the difference.
func ParseDurationOrEnd(s string, startMinute int) (int, error) {
	if dur, err := ParseDuration(s); err == nil {
		return dur, nil
	}
	end, err := ParseStart(s)
	if err != nil {
		return 0, fmt.Errorf("unrecognized duration or end time %q; use Nh, Nm, NhMm, or a time like 15:00 / 3:00pm", s)
	}
	dur := end - startMinute
	if dur <= 0 {
		return 0, fmt.Errorf("end time %q is not after start", s)
	}
	return dur, nil
}
