package auth

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/pboyd/todo/services/todo/internal/db"
)

type contextKey int

const userIDKey contextKey = 1

// Querier is the subset of db.Queries used by the auth middleware.
type Querier interface {
	GetApiKeyByHash(ctx context.Context, keyHash string) (db.ApiKey, error)
	GetSession(ctx context.Context, id string) (db.Session, error)
}

// UserID returns the authenticated user's ID from the context, or 0 if absent.
func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

// WithUserID returns a context with the given user_id set. Used in tests.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// Middleware returns an http.Handler middleware that authenticates requests.
// Paths under /auth/ are exempt. All other paths must carry either an
// Authorization: Bearer header (API key) or a todo_session cookie (session).
func Middleware(q Querier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			userID, ok := resolve(r.Context(), q, r)
			if !ok {
				log.Printf("auth: rejected %s %s (no valid credential)", r.Method, r.URL.Path)
				http.Error(w, "", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resolve tries to resolve a user_id from the request. Returns (id, true) on
// success. Checks Authorization: Bearer first, then the session cookie.
func resolve(ctx context.Context, q Querier, r *http.Request) (int64, bool) {
	if bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); bearer != "" && bearer != r.Header.Get("Authorization") {
		hash := HashAPIKey(bearer)
		key, err := q.GetApiKeyByHash(ctx, hash)
		if err != nil {
			return 0, false
		}
		return key.UserID, true
	}

	cookie, err := r.Cookie("todo_session")
	if err != nil {
		return 0, false
	}
	session, err := q.GetSession(ctx, cookie.Value)
	if err != nil {
		if !isNotFound(err) {
			log.Printf("auth: GetSession error: %v", err)
		}
		return 0, false
	}
	if !session.ExpiresAt.Valid || time.Now().After(session.ExpiresAt.Time) {
		return 0, false
	}
	return session.UserID, true
}

func isNotFound(err error) bool {
	return err == pgx.ErrNoRows
}
