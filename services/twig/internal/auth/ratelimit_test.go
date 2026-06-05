package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pboyd/twig/services/twig/internal/auth"
)

// ---- LoginLimiter unit tests ------------------------------------------------

func TestLoginLimiter_NotBlockedInitially(t *testing.T) {
	l := auth.NewLoginLimiter(5, 15*time.Minute)
	blocked, _ := l.Blocked("ip:1.2.3.4")
	if blocked {
		t.Error("fresh limiter should not block")
	}
}

func TestLoginLimiter_BlocksAfterMaxFailures(t *testing.T) {
	l := auth.NewLoginLimiter(3, 15*time.Minute)
	key := "ip:1.2.3.4"
	for i := 0; i < 3; i++ {
		l.RecordFailure(key)
	}
	blocked, _ := l.Blocked(key)
	if !blocked {
		t.Error("should be blocked after max failures")
	}
}

func TestLoginLimiter_NotBlockedBelowMax(t *testing.T) {
	l := auth.NewLoginLimiter(3, 15*time.Minute)
	key := "ip:1.2.3.4"
	for i := 0; i < 2; i++ {
		l.RecordFailure(key)
	}
	blocked, _ := l.Blocked(key)
	if blocked {
		t.Error("should not be blocked before reaching max failures")
	}
}

func TestLoginLimiter_RetryAfterIsPositive(t *testing.T) {
	l := auth.NewLoginLimiter(2, 15*time.Minute)
	key := "ip:1.2.3.4"
	l.RecordFailure(key)
	l.RecordFailure(key)

	blocked, retryAfter := l.Blocked(key)
	if !blocked {
		t.Fatal("should be blocked")
	}
	if retryAfter <= 0 {
		t.Errorf("retryAfter should be positive, got %v", retryAfter)
	}
	if retryAfter > 15*time.Minute {
		t.Errorf("retryAfter should not exceed window (%v), got %v", 15*time.Minute, retryAfter)
	}
}

func TestLoginLimiter_WindowExpiryUnblocks(t *testing.T) {
	window := 10 * time.Minute
	l := auth.NewLoginLimiter(2, window)
	key := "ip:5.6.7.8"

	now := time.Now()
	auth.SetLimiterClock(l, func() time.Time { return now })

	l.RecordFailure(key)
	l.RecordFailure(key)

	// still blocked right before expiry
	auth.SetLimiterClock(l, func() time.Time { return now.Add(window - time.Second) })
	blocked, _ := l.Blocked(key)
	if !blocked {
		t.Error("should still be blocked just before window expires")
	}

	// unblocked once window has passed
	auth.SetLimiterClock(l, func() time.Time { return now.Add(window + time.Second) })
	blocked, _ = l.Blocked(key)
	if blocked {
		t.Error("should be unblocked after window expires")
	}
}

func TestLoginLimiter_ResetClearsKey(t *testing.T) {
	l := auth.NewLoginLimiter(2, 15*time.Minute)
	key := "user:alice"
	l.RecordFailure(key)
	l.RecordFailure(key)

	l.Reset(key)

	blocked, _ := l.Blocked(key)
	if blocked {
		t.Error("Reset should clear the block")
	}
}

func TestLoginLimiter_DistinctKeysAreIndependent(t *testing.T) {
	l := auth.NewLoginLimiter(2, 15*time.Minute)
	keyA := "ip:1.1.1.1"
	keyB := "ip:2.2.2.2"

	l.RecordFailure(keyA)
	l.RecordFailure(keyA)

	blockedB, _ := l.Blocked(keyB)
	if blockedB {
		t.Error("unrelated key should not be blocked")
	}

	blockedA, _ := l.Blocked(keyA)
	if !blockedA {
		t.Error("keyA should still be blocked")
	}
}

func TestLoginLimiter_RecordFailureAfterExpiredWindowStartsFresh(t *testing.T) {
	window := 5 * time.Minute
	l := auth.NewLoginLimiter(3, window)
	key := "user:bob"

	now := time.Now()
	auth.SetLimiterClock(l, func() time.Time { return now })

	l.RecordFailure(key)
	l.RecordFailure(key)

	// advance past window, then record one more failure
	auth.SetLimiterClock(l, func() time.Time { return now.Add(window + time.Second) })
	l.RecordFailure(key)

	// only 1 failure in the new window — should not be blocked (max=3)
	blocked, _ := l.Blocked(key)
	if blocked {
		t.Error("first failure in a new window should not block (max=3)")
	}
}

// ---- clientIP unit tests ----------------------------------------------------

func TestClientIP_CFConnectingIPTakesPrecedence(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.Header.Set("CF-Connecting-IP", "1.2.3.4")
	req.Header.Set("X-Forwarded-For", "9.8.7.6")
	req.RemoteAddr = "10.0.0.1:1234"

	got := auth.ClientIP(req)
	if got != "1.2.3.4" {
		t.Errorf("CF-Connecting-IP should win, got %q", got)
	}
}

func TestClientIP_XForwardedForRightMost(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.1")
	req.RemoteAddr = "10.0.0.1:9999"

	// The right-most (last) entry is the one appended by the trusted proxy.
	got := auth.ClientIP(req)
	if got != "198.51.100.1" {
		t.Errorf("want right-most XFF entry, got %q", got)
	}
}

func TestClientIP_XForwardedForSingleEntry(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.5")
	req.RemoteAddr = "10.0.0.1:9999"

	got := auth.ClientIP(req)
	if got != "203.0.113.5" {
		t.Errorf("want single XFF entry, got %q", got)
	}
}

func TestClientIP_RemoteAddrFallback(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "192.168.1.42:5678"

	got := auth.ClientIP(req)
	if got != "192.168.1.42" {
		t.Errorf("should fall back to RemoteAddr, got %q", got)
	}
}

func TestClientIP_RemoteAddrNoPort(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "192.168.1.1"

	got := auth.ClientIP(req)
	if got != "192.168.1.1" {
		t.Errorf("RemoteAddr without port should work, got %q", got)
	}
}

// ---- handler integration tests for rate-limiting ---------------------------

// TestLoginHandler_RateLimitsByIP checks that the Nth wrong attempt from the
// same IP returns 429 with a Retry-After header.
func TestLoginHandler_RateLimitsByIP(t *testing.T) {
	q := newStubHandlerQuerier()
	q.addUser("alice", "correct")

	const max = 3
	l := auth.NewLoginLimiter(max, 15*time.Minute)
	h := auth.LoginHandler(q, 30*24*time.Hour, l)

	makeAttempt := func(ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/auth/login",
			strings.NewReader(`{"username":"alice","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// First max attempts: 401
	for i := 0; i < max; i++ {
		rec := makeAttempt("1.2.3.4")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("attempt %d: want 401, got %d", i+1, rec.Code)
		}
	}

	// (max+1)th attempt: 429
	rec := makeAttempt("1.2.3.4")
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("blocked attempt: want 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("blocked response should include Retry-After header")
	}
}

// TestLoginHandler_RateLimitsByUsername checks that the per-username limit
// fires even when the same user is attacked from different IPs.
func TestLoginHandler_RateLimitsByUsername(t *testing.T) {
	q := newStubHandlerQuerier()
	q.addUser("bob", "correct")

	const max = 3
	l := auth.NewLoginLimiter(max, 15*time.Minute)
	h := auth.LoginHandler(q, 30*24*time.Hour, l)

	wrongBody := `{"username":"bob","password":"wrong"}`

	for i := 0; i < max; i++ {
		// Each attempt from a distinct IP — IP limiter never trips.
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(wrongBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = fmt.Sprintf("10.0.0.%d:9999", i+1)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("attempt %d: want 401, got %d", i+1, rec.Code)
		}
	}

	// Username limit exhausted — one more attempt from yet another new IP.
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(wrongBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.99:9999"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("per-username block: want 429, got %d", rec.Code)
	}
}

// TestLoginHandler_SuccessResetsCounters checks that a successful login clears
// the failure counters so the user can try again after the block is released.
func TestLoginHandler_SuccessResetsCounters(t *testing.T) {
	q := newStubHandlerQuerier()
	q.addUser("carol", "correct")

	const max = 5
	l := auth.NewLoginLimiter(max, 15*time.Minute)
	h := auth.LoginHandler(q, 30*24*time.Hour, l)

	ip := "5.5.5.5"

	// A few wrong attempts (below max so the IP is not blocked yet).
	const wrongAttempts = max - 1
	for i := 0; i < wrongAttempts; i++ {
		req := httptest.NewRequest("POST", "/auth/login",
			strings.NewReader(`{"username":"carol","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip + ":80"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}

	// Correct login — should succeed and reset.
	req := httptest.NewRequest("POST", "/auth/login",
		strings.NewReader(`{"username":"carol","password":"correct"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":80"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("correct login should succeed, got %d", rec.Code)
	}

	// After success, the counter was reset: next wrong attempt is 401 again.
	req2 := httptest.NewRequest("POST", "/auth/login",
		strings.NewReader(`{"username":"carol","password":"wrong"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = ip + ":80"
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("after reset, first failure should be 401, got %d", rec2.Code)
	}
}

// TestLoginHandler_BadRequestDoesNotCountAsFailure ensures malformed / empty
// requests don't consume failure budget.
func TestLoginHandler_BadRequestDoesNotCountAsFailure(t *testing.T) {
	q := newStubHandlerQuerier()
	q.addUser("dave", "correct")

	const max = 2
	l := auth.NewLoginLimiter(max, 15*time.Minute)
	h := auth.LoginHandler(q, 30*24*time.Hour, l)

	ip := "6.6.6.6"

	// Send max bad requests — these should NOT eat into the failure budget.
	for i := 0; i < max; i++ {
		req := httptest.NewRequest("POST", "/auth/login",
			strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip + ":80"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("empty request should be 400, got %d", rec.Code)
		}
	}

	// Wrong-password attempt should still get 401 (not 429).
	req := httptest.NewRequest("POST", "/auth/login",
		strings.NewReader(`{"username":"dave","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":80"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("after bad requests, wrong password should be 401, got %d", rec.Code)
	}
}

// TestLoginHandler_BlockedRequestDoesNotCreateSession verifies no session is
// recorded when the rate limit fires.
func TestLoginHandler_BlockedRequestDoesNotCreateSession(t *testing.T) {
	q := newStubHandlerQuerier()
	q.addUser("eve", "correct")

	const max = 1
	l := auth.NewLoginLimiter(max, 15*time.Minute)
	h := auth.LoginHandler(q, 30*24*time.Hour, l)

	ip := "7.7.7.7:80"

	// Exhaust the limit with a wrong attempt.
	req := httptest.NewRequest("POST", "/auth/login",
		strings.NewReader(`{"username":"eve","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("first wrong attempt: want 401, got %d", rec.Code)
	}

	// Now send correct credentials — should be blocked, no session created.
	req2 := httptest.NewRequest("POST", "/auth/login",
		strings.NewReader(`{"username":"eve","password":"correct"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = ip
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("want 429 when blocked, got %d", rec2.Code)
	}
	if len(q.sessions) != 0 {
		t.Error("no session should be created when rate-limited")
	}
}
