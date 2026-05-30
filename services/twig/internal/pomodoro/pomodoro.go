package pomodoro

import "time"

const Length = 25 * time.Minute

func Remaining(start, now time.Time) time.Duration {
	d := start.Add(Length).Sub(now)
	if d < 0 {
		return 0
	}
	return d
}
