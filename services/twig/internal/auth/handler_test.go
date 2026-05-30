package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
)

// stubHandlerQuerier implements auth.HandlerQuerier for tests.
type stubHandlerQuerier struct {
	users    map[string]db.User
	sessions map[string]db.Session
}

func newStubHandlerQuerier() *stubHandlerQuerier {
	return &stubHandlerQuerier{
		users:    make(map[string]db.User),
		sessions: make(map[string]db.Session),
	}
}

func (s *stubHandlerQuerier) addUser(username, password string) {
	hash, _ := auth.HashPassword(password)
	s.users[username] = db.User{ID: int64(len(s.users) + 1), Username: username, PasswordHash: hash}
}

func (s *stubHandlerQuerier) GetUserByUsername(_ context.Context, username string) (db.User, error) {
	u, ok := s.users[username]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (s *stubHandlerQuerier) CreateSession(_ context.Context, arg db.CreateSessionParams) (db.Session, error) {
	sess := db.Session{
		ID:        arg.ID,
		UserID:    arg.UserID,
		CreatedAt: arg.CreatedAt,
		ExpiresAt: arg.ExpiresAt,
	}
	s.sessions[arg.ID] = sess
	return sess, nil
}

func (s *stubHandlerQuerier) DeleteSession(_ context.Context, id string) error {
	delete(s.sessions, id)
	return nil
}

var _ auth.HandlerQuerier = (*stubHandlerQuerier)(nil)

// Verify db.Queries satisfies the interface.
var _ auth.HandlerQuerier = (*db.Queries)(nil)

func TestLoginHandler_Success(t *testing.T) {
	q := newStubHandlerQuerier()
	q.addUser("alice", "s3cr3t")

	h := auth.LoginHandler(q, 30*24*time.Hour)
	body := `{"username":"alice","password":"s3cr3t"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "todo_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("no todo_session cookie in response")
	}
	if len(sessionCookie.Value) != 64 {
		t.Errorf("cookie value length = %d, want 64", len(sessionCookie.Value))
	}
	if !sessionCookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if !sessionCookie.Secure {
		t.Error("cookie should be Secure")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", sessionCookie.SameSite)
	}
	if sessionCookie.MaxAge != int(30*24*time.Hour/time.Second) {
		t.Errorf("MaxAge = %d, want %d", sessionCookie.MaxAge, int(30*24*time.Hour/time.Second))
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	q := newStubHandlerQuerier()
	q.addUser("alice", "s3cr3t")

	h := auth.LoginHandler(q, 30*24*time.Hour)
	body := `{"username":"alice","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "todo_session" {
			t.Error("should not set session cookie on wrong password")
		}
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp["error"] == "" {
		t.Error("expected error field in response")
	}
}

func TestLoginHandler_UnknownUser(t *testing.T) {
	q := newStubHandlerQuerier()

	h := auth.LoginHandler(q, 30*24*time.Hour)
	body := `{"username":"nobody","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	// Message must be generic — must not distinguish unknown user from wrong password.
	if resp["error"] != "invalid username or password" {
		t.Errorf("error = %q, want generic message", resp["error"])
	}
}

func TestLoginHandler_BadJSON(t *testing.T) {
	q := newStubHandlerQuerier()
	h := auth.LoginHandler(q, 30*24*time.Hour)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestLoginHandler_MissingFields(t *testing.T) {
	q := newStubHandlerQuerier()
	h := auth.LoginHandler(q, 30*24*time.Hour)

	for _, body := range []string{`{"username":"alice"}`, `{"password":"pass"}`, `{}`} {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body=%s: want 400, got %d", body, rec.Code)
		}
	}
}

func TestLoginHandler_NonPost(t *testing.T) {
	q := newStubHandlerQuerier()
	h := auth.LoginHandler(q, 30*24*time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}

func TestLogoutHandler_DeletesSession(t *testing.T) {
	q := newStubHandlerQuerier()
	sessionID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	q.sessions[sessionID] = db.Session{
		ID:        sessionID,
		UserID:    1,
		CreatedAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	}

	h := auth.LogoutHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "todo_session", Value: sessionID})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if _, ok := q.sessions[sessionID]; ok {
		t.Error("session should have been deleted")
	}

	// Response should clear the cookie.
	cookies := rec.Result().Cookies()
	var clearCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "todo_session" {
			clearCookie = c
			break
		}
	}
	if clearCookie == nil {
		t.Fatal("expected clearing Set-Cookie header")
	}
	if clearCookie.MaxAge != 0 {
		t.Errorf("MaxAge = %d, want 0 to clear cookie", clearCookie.MaxAge)
	}
}

func TestLogoutHandler_IdempotentMissingCookie(t *testing.T) {
	q := newStubHandlerQuerier()
	h := auth.LogoutHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestLogoutHandler_IdempotentUnknownSession(t *testing.T) {
	q := newStubHandlerQuerier()
	h := auth.LogoutHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "todo_session", Value: "unknownsessionid"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestLogoutHandler_NonPost(t *testing.T) {
	q := newStubHandlerQuerier()
	h := auth.LogoutHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}
