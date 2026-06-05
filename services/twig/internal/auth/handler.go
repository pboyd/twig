package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pboyd/twig/services/twig/internal/db"
)

// HandlerQuerier is the subset of db.Queries used by the auth HTTP handlers.
type HandlerQuerier interface {
	GetUserByUsername(ctx context.Context, username string) (db.User, error)
	CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error)
	DeleteSession(ctx context.Context, id string) error
}

const sessionCookieName = "todo_session"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginHandler returns an http.Handler for POST /auth/login.
// limiter enforces per-IP and per-username failure rate limits.
func LoginHandler(q HandlerQuerier, sessionLifetime time.Duration, limiter *LoginLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "", http.StatusMethodNotAllowed)
			return
		}

		// Check per-IP limit before doing any work.
		ipKey := "ip:" + clientIP(r)
		if blocked, retryAfter := limiter.Blocked(ipKey); blocked {
			log.Printf("auth: login rate-limited for IP %s", clientIP(r))
			writeJSON429(w, retryAfter)
			return
		}

		var creds loginRequest
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}
		if creds.Username == "" || creds.Password == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"username and password are required"}`))
			return
		}

		// Check per-username limit after we have the username.
		userKey := "user:" + creds.Username
		if blocked, retryAfter := limiter.Blocked(userKey); blocked {
			log.Printf("auth: login rate-limited for username %q", creds.Username)
			writeJSON429(w, retryAfter)
			return
		}

		user, err := q.GetUserByUsername(r.Context(), creds.Username)
		if err != nil {
			if isNotFound(err) {
				log.Printf("auth: login failed for unknown user %q", creds.Username)
			} else {
				log.Printf("auth: GetUserByUsername error: %v", err)
			}
			limiter.RecordFailure(ipKey)
			limiter.RecordFailure(userKey)
			writeJSON401(w)
			return
		}

		if err := VerifyPassword(user.PasswordHash, creds.Password); err != nil {
			log.Printf("auth: login failed (wrong password) for user %q", creds.Username)
			limiter.RecordFailure(ipKey)
			limiter.RecordFailure(userKey)
			writeJSON401(w)
			return
		}

		// Credentials verified — reset failure counters so earlier typos
		// don't carry over to future login windows.
		limiter.Reset(ipKey)
		limiter.Reset(userKey)

		token, err := GenerateToken()
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		now := time.Now()
		expires := now.Add(sessionLifetime)
		_, err = q.CreateSession(r.Context(), db.CreateSessionParams{
			ID:        token,
			UserID:    user.ID,
			CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
		})
		if err != nil {
			log.Printf("auth: CreateSession error: %v", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(sessionLifetime / time.Second),
		})
		log.Printf("auth: login success for user %q (user_id=%d)", user.Username, user.ID)
		w.WriteHeader(http.StatusOK)
	})
}

// LogoutHandler returns an http.Handler for POST /auth/logout.
func LogoutHandler(q HandlerQuerier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "", http.StatusMethodNotAllowed)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err == nil {
			if delErr := q.DeleteSession(r.Context(), cookie.Value); delErr != nil && delErr != pgx.ErrNoRows {
				log.Printf("auth: DeleteSession error: %v", delErr)
			}
			log.Printf("auth: logout (session deleted)")
		}

		// Always clear the cookie.
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   0,
		})
		w.WriteHeader(http.StatusOK)
	})
}

func writeJSON401(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"invalid username or password"}`))
}

func writeJSON429(w http.ResponseWriter, retryAfter time.Duration) {
	secs := int(math.Ceil(retryAfter.Seconds()))
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = fmt.Fprintf(w, `{"error":"too many attempts, please try again later"}`)
}
