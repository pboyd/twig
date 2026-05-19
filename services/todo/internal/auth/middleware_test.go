package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pboyd/todo/services/todo/internal/auth"
	"github.com/pboyd/todo/services/todo/internal/db"
)

// stubQuerier is a test double for the auth.Querier interface.
type stubQuerier struct {
	apiKey *db.ApiKey
	session *db.Session
}

func (s *stubQuerier) GetApiKeyByHash(_ context.Context, hash string) (db.ApiKey, error) {
	if s.apiKey != nil && s.apiKey.KeyHash == hash {
		return *s.apiKey, nil
	}
	return db.ApiKey{}, pgx.ErrNoRows
}

func (s *stubQuerier) GetSession(_ context.Context, id string) (db.Session, error) {
	if s.session != nil && s.session.ID == id {
		return *s.session, nil
	}
	return db.Session{}, pgx.ErrNoRows
}

func newMiddlewareHandler(q auth.Querier, userIDOut *int64) http.Handler {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userIDOut != nil {
			*userIDOut = auth.UserID(r.Context())
		}
		w.WriteHeader(http.StatusOK)
	})
	return auth.Middleware(q)(inner)
}

func TestMiddleware_BypassAuthPaths(t *testing.T) {
	q := &stubQuerier{}
	h := newMiddlewareHandler(q, nil)

	for _, path := range []string{"/auth/login", "/auth/logout"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("path %s: want 200, got %d", path, rec.Code)
		}
	}
}

func TestMiddleware_MissingCredential(t *testing.T) {
	q := &stubQuerier{}
	h := newMiddlewareHandler(q, nil)

	req := httptest.NewRequest(http.MethodGet, "/task.v1.TaskService/ListTasks", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestMiddleware_ValidAPIKey(t *testing.T) {
	rawKey := "deadbeef"
	keyHash := auth.HashAPIKey(rawKey)
	q := &stubQuerier{
		apiKey: &db.ApiKey{ID: 1, UserID: 42, KeyHash: keyHash},
	}
	var gotUserID int64
	h := newMiddlewareHandler(q, &gotUserID)

	req := httptest.NewRequest(http.MethodPost, "/task.v1.TaskService/ListTasks", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if gotUserID != 42 {
		t.Errorf("user_id = %d, want 42", gotUserID)
	}
}

func TestMiddleware_UnknownAPIKey(t *testing.T) {
	q := &stubQuerier{}
	h := newMiddlewareHandler(q, nil)

	req := httptest.NewRequest(http.MethodPost, "/task.v1.TaskService/ListTasks", nil)
	req.Header.Set("Authorization", "Bearer unknownkey")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestMiddleware_ValidSession(t *testing.T) {
	sessionID := "abc123"
	q := &stubQuerier{
		session: &db.Session{
			ID:        sessionID,
			UserID:    7,
			CreatedAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		},
	}
	var gotUserID int64
	h := newMiddlewareHandler(q, &gotUserID)

	req := httptest.NewRequest(http.MethodPost, "/task.v1.TaskService/ListTasks", nil)
	req.AddCookie(&http.Cookie{Name: "todo_session", Value: sessionID})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if gotUserID != 7 {
		t.Errorf("user_id = %d, want 7", gotUserID)
	}
}

func TestMiddleware_ExpiredSession(t *testing.T) {
	sessionID := "expiredsession"
	q := &stubQuerier{
		session: &db.Session{
			ID:        sessionID,
			UserID:    7,
			CreatedAt: pgtype.Timestamptz{Time: time.Now().Add(-48 * time.Hour), Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		},
	}
	h := newMiddlewareHandler(q, nil)

	req := httptest.NewRequest(http.MethodPost, "/task.v1.TaskService/ListTasks", nil)
	req.AddCookie(&http.Cookie{Name: "todo_session", Value: sessionID})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestMiddleware_UnknownSession(t *testing.T) {
	q := &stubQuerier{}
	h := newMiddlewareHandler(q, nil)

	req := httptest.NewRequest(http.MethodPost, "/task.v1.TaskService/ListTasks", nil)
	req.AddCookie(&http.Cookie{Name: "todo_session", Value: "nosuchsession"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestMiddleware_APIKeyWinsOverSession(t *testing.T) {
	rawKey := "myapikey"
	keyHash := auth.HashAPIKey(rawKey)
	sessionID := "mysession"

	q := &stubQuerier{
		apiKey: &db.ApiKey{ID: 1, UserID: 100, KeyHash: keyHash},
		session: &db.Session{
			ID:        sessionID,
			UserID:    200,
			CreatedAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		},
	}
	var gotUserID int64
	h := newMiddlewareHandler(q, &gotUserID)

	req := httptest.NewRequest(http.MethodPost, "/task.v1.TaskService/ListTasks", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.AddCookie(&http.Cookie{Name: "todo_session", Value: sessionID})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if gotUserID != 100 {
		t.Errorf("user_id = %d, want 100 (API key user wins)", gotUserID)
	}
}

// Ensure db.Queries satisfies the Querier interface at compile time.
var _ auth.Querier = (*db.Queries)(nil)

// Ensure stubQuerier satisfies the Querier interface at compile time.
var _ auth.Querier = (*stubQuerier)(nil)

// errQuerier always returns errors to simulate DB failure.
type errQuerier struct{}

func (e *errQuerier) GetApiKeyByHash(_ context.Context, _ string) (db.ApiKey, error) {
	return db.ApiKey{}, errors.New("db error")
}
func (e *errQuerier) GetSession(_ context.Context, _ string) (db.Session, error) {
	return db.Session{}, errors.New("db error")
}

var _ auth.Querier = (*errQuerier)(nil)
