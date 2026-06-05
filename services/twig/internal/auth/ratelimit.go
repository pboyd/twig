package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// window tracks failure count for one rate-limit key within a time window.
type window struct {
	failures int
	start    time.Time
}

// LoginLimiter is a mutex-guarded fixed-window failure counter that limits
// repeated login attempts per key (typically a client IP or username).
//
// State is in-memory and is lost on server restart, which is acceptable for
// basic brute-force protection on a single-instance deployment.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*window
	max      int
	dur      time.Duration
	now      func() time.Time
}

// NewLoginLimiter creates a LoginLimiter that blocks a key after max failures
// within the given window duration. A background goroutine periodically evicts
// expired entries to keep memory bounded.
func NewLoginLimiter(max int, dur time.Duration) *LoginLimiter {
	l := &LoginLimiter{
		attempts: make(map[string]*window),
		max:      max,
		dur:      dur,
		now:      time.Now,
	}
	go l.janitor()
	return l
}

// Blocked reports whether key is currently rate-limited. If so it also returns
// the approximate duration the caller should wait before retrying.
func (l *LoginLimiter) Blocked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	w, ok := l.attempts[key]
	if !ok {
		return false, 0
	}
	now := l.now()
	if now.Sub(w.start) >= l.dur {
		// Window expired — treat as unblocked (entry cleaned up by janitor).
		return false, 0
	}
	if w.failures < l.max {
		return false, 0
	}
	retryAfter := l.dur - now.Sub(w.start)
	return true, retryAfter
}

// RecordFailure increments the failure count for key. If the existing window
// has expired it starts a fresh one.
func (l *LoginLimiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	w, ok := l.attempts[key]
	if !ok || now.Sub(w.start) >= l.dur {
		l.attempts[key] = &window{failures: 1, start: now}
		return
	}
	w.failures++
}

// Reset removes the failure record for key. Call after a successful login so
// a legitimate user's earlier typos don't count against them.
func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

// janitor periodically evicts expired entries to keep the map from growing
// without bound.
func (l *LoginLimiter) janitor() {
	ticker := time.NewTicker(l.dur)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := l.now()
		for k, w := range l.attempts {
			if now.Sub(w.start) >= l.dur {
				delete(l.attempts, k)
			}
		}
		l.mu.Unlock()
	}
}

// clientIP extracts the best available client IP from the request.
//
// Precedence (highest first):
//  1. CF-Connecting-IP — set by Cloudflare for traffic through the tunnel;
//     not spoofable by clients when Cloudflare is the gateway.
//  2. Right-most X-Forwarded-For entry — the hop that Caddy appends when it
//     reverse-proxies to the server container; this is the real client IP for
//     direct LAN/HTTPS traffic.
//  3. r.RemoteAddr — used only when neither header is present (e.g. during
//     local dev without Caddy in front).
//
// Trust assumption: the backend is only reachable through Caddy (or
// cloudflared→Caddy). Do not expose port 8080 directly to the internet.
func clientIP(r *http.Request) string {
	// 1. Cloudflare Tunnel real-IP header.
	if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
		return cf
	}

	// 2. Right-most X-Forwarded-For entry (Caddy appends the connecting IP).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
			return last
		}
	}

	// 3. RemoteAddr fallback — strip port.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // already bare IP (no port)
	}
	return host
}
