package pomodoro_test

import (
	"testing"
	"time"

	"github.com/pboyd/twig/services/twig/internal/pomodoro"
)

func TestLength(t *testing.T) {
	if pomodoro.Length != 25*time.Minute {
		t.Fatalf("expected 25 minutes, got %v", pomodoro.Length)
	}
}

func TestRemaining(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		start time.Time
		now   time.Time
		want  time.Duration
	}{
		{
			name:  "start in the past with time remaining",
			start: now.Add(-10 * time.Minute),
			now:   now,
			want:  15 * time.Minute,
		},
		{
			name:  "start in the future",
			start: now.Add(5 * time.Minute),
			now:   now,
			want:  30 * time.Minute,
		},
		{
			name:  "exactly at expiry",
			start: now.Add(-25 * time.Minute),
			now:   now,
			want:  0,
		},
		{
			name:  "already expired",
			start: now.Add(-30 * time.Minute),
			now:   now,
			want:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pomodoro.Remaining(tc.start, tc.now)
			if got != tc.want {
				t.Errorf("Remaining(%v, %v) = %v, want %v", tc.start, tc.now, got, tc.want)
			}
		})
	}
}
