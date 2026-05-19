package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pboyd/todo/services/todo/internal/db"
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
func LoginHandler(q HandlerQuerier, sessionLifetime time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "", http.StatusMethodNotAllowed)
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

		user, err := q.GetUserByUsername(r.Context(), creds.Username)
		if err != nil {
			if isNotFound(err) {
				log.Printf("auth: login failed for unknown user %q", creds.Username)
			} else {
				log.Printf("auth: GetUserByUsername error: %v", err)
			}
			writeJSON401(w)
			return
		}

		if err := VerifyPassword(user.PasswordHash, creds.Password); err != nil {
			log.Printf("auth: login failed (wrong password) for user %q", creds.Username)
			writeJSON401(w)
			return
		}

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
