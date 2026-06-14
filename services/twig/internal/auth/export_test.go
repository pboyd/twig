package auth

import (
	"net/http"
	"time"
)

// SetLimiterClock replaces the clock function on a LoginLimiter. Used in tests
// to make window-based behaviour deterministic without real time.Sleep.
func SetLimiterClock(l *LoginLimiter, now func() time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.now = now
}

// ClientIPFromRequest exposes the unexported clientIP helper for white-box tests.
func ClientIPFromRequest(r *http.Request) string {
	return clientIP(r)
}
