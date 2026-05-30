package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pboyd/twig/services/twig/internal/db"
)

// ProvisionQuerier is the subset of db.Queries used by ProvisionUser.
type ProvisionQuerier interface {
	GetUserByUsername(ctx context.Context, username string) (db.User, error)
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error
	DeleteApiKeysByUser(ctx context.Context, userID int64) error
	CreateApiKey(ctx context.Context, arg db.CreateApiKeyParams) (db.ApiKey, error)
}

// ProvisionUser creates or updates a user account and issues one API key.
// Returns the raw (unhashed) API key to be displayed once.
func ProvisionUser(ctx context.Context, q ProvisionQuerier, username, password string) (string, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}

	var userID int64
	existing, err := q.GetUserByUsername(ctx, username)
	if err != nil && err != pgx.ErrNoRows {
		return "", err
	}

	if err == pgx.ErrNoRows {
		user, err := q.CreateUser(ctx, db.CreateUserParams{
			Username:     username,
			PasswordHash: hash,
		})
		if err != nil {
			return "", err
		}
		userID = user.ID
	} else {
		userID = existing.ID
		if err := q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
			Username:     username,
			PasswordHash: hash,
		}); err != nil {
			return "", err
		}
		if err := q.DeleteApiKeysByUser(ctx, userID); err != nil {
			return "", err
		}
	}

	rawKey, err := GenerateToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	_, err = q.CreateApiKey(ctx, db.CreateApiKeyParams{
		UserID:    userID,
		KeyHash:   HashAPIKey(rawKey),
		Label:     pgtype.Text{String: "provisioned", Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return "", err
	}

	return rawKey, nil
}
