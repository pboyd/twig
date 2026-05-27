package timeparse_test

import (
	"testing"

	"github.com/pboyd/todo/services/todo/internal/cli/timeparse"
)

func TestParseStart(t *testing.T) {
	cases := []struct {
		input   string
		want    int
		wantErr bool
	}{
		// 24-hour
		{"13:15", 795, false},
		{"00:00", 0, false},
		{"23:59", 1439, false},
		// 12-hour variants all map to 13:15 = 795
		{"1:15pm", 795, false},
		{"1:15PM", 795, false},
		{"1:15 pm", 795, false},
		{"01:15 PM", 795, false},
		// noon / midnight edge cases
		{"12:00pm", 720, false},
		{"12:00am", 0, false},
		{"12:30pm", 750, false},
		{"12:30am", 30, false},
		// 4-digit no-colon 24-hour
		{"0815", 495, false},
		{"1400", 840, false},
		{"0000", 0, false},
		{"2359", 1439, false},
		{"2400", 0, true},
		{"1360", 0, true},
		{"815", 0, true}, // 3-digit not accepted
		// error cases
		{"", 0, true},
		{"junk", 0, true},
		{"25:00", 0, true},
		{"13:60", 0, true},
		{"13:15am", 0, true}, // 13 is not valid in 12-hour
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got, err := timeparse.ParseStart(c.input)
			if c.wantErr {
				if err == nil {
					t.Errorf("ParseStart(%q) = %d, want error", c.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseStart(%q) unexpected error: %v", c.input, err)
				return
			}
			if got != c.want {
				t.Errorf("ParseStart(%q) = %d, want %d", c.input, got, c.want)
			}
		})
	}
}

func TestParseDurationOrEnd(t *testing.T) {
	cases := []struct {
		input       string
		startMinute int
		want        int
		wantErr     bool
	}{
		// duration passthrough
		{"90m", 0, 90, false},
		{"1h30m", 0, 90, false},
		{"2h", 720, 120, false},
		// HH:MM end time
		{"13:00", 720, 60, false},
		{"14:30", 720, 150, false},
		// HHMM no-colon end time
		{"1300", 720, 60, false},
		// 12-hour end time
		{"1:00pm", 720, 60, false},
		{"2:30pm", 720, 150, false},
		// end not after start
		{"11:00", 720, 0, true},
		{"12:00", 720, 0, true},
		// garbage
		{"junk", 0, 0, true},
		{"", 0, 0, true},
	}
	for _, c := range cases {
		name := c.input
		t.Run(name, func(t *testing.T) {
			got, err := timeparse.ParseDurationOrEnd(c.input, c.startMinute)
			if c.wantErr {
				if err == nil {
					t.Errorf("ParseDurationOrEnd(%q, %d) = %d, want error", c.input, c.startMinute, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDurationOrEnd(%q, %d) unexpected error: %v", c.input, c.startMinute, err)
				return
			}
			if got != c.want {
				t.Errorf("ParseDurationOrEnd(%q, %d) = %d, want %d", c.input, c.startMinute, got, c.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"90m", 90, false},
		{"1h30m", 90, false},
		{"0h45m", 45, false},
		{"2h", 120, false},
		{"1h", 60, false},
		{"45m", 45, false},
		// error cases
		{"", 0, true},
		{"junk", 0, true},
		{"1.5h", 0, true},
		{"90", 0, true},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got, err := timeparse.ParseDuration(c.input)
			if c.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) = %d, want error", c.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", c.input, err)
				return
			}
			if got != c.want {
				t.Errorf("ParseDuration(%q) = %d, want %d", c.input, got, c.want)
			}
		})
	}
}
