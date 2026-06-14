package handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	accountv1 "github.com/pboyd/twig/api/gen/account/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
)

const defaultKeyLabel = "web key"

// AccountQuerier is the subset of db.Queries used by AccountService.
type AccountQuerier interface {
	GetUserByID(ctx context.Context, id int64) (db.User, error)
	UpdateUserPasswordByID(ctx context.Context, arg db.UpdateUserPasswordByIDParams) error
	ListApiKeysByUser(ctx context.Context, userID int64) ([]db.ListApiKeysByUserRow, error)
	CreateApiKey(ctx context.Context, arg db.CreateApiKeyParams) (db.ApiKey, error)
	DeleteApiKeyForUser(ctx context.Context, arg db.DeleteApiKeyForUserParams) (int64, error)
}

// Account implements accountv1connect.AccountServiceHandler.
type Account struct {
	Queries AccountQuerier
	Limiter *auth.LoginLimiter
}

func (a *Account) ChangePassword(
	ctx context.Context,
	req *connect.Request[accountv1.ChangePasswordRequest],
) (*connect.Response[accountv1.ChangePasswordResponse], error) {
	userID := auth.UserID(ctx)

	user, err := a.Queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	limiterKey := "user:" + user.Username
	if blocked, _ := a.Limiter.Blocked(limiterKey); blocked {
		return nil, connect.NewError(connect.CodeResourceExhausted,
			errors.New("too many attempts — take a breather and try again in a few minutes"))
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.Msg.CurrentPassword); err != nil {
		a.Limiter.RecordFailure(limiterKey)
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("current password is incorrect"))
	}

	if len(req.Msg.NewPassword) < 8 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("new password must be at least 8 characters"))
	}

	newHash, err := auth.HashPassword(req.Msg.NewPassword)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := a.Queries.UpdateUserPasswordByID(ctx, db.UpdateUserPasswordByIDParams{
		ID:           userID,
		PasswordHash: newHash,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	a.Limiter.Reset(limiterKey)
	return connect.NewResponse(&accountv1.ChangePasswordResponse{}), nil
}

func (a *Account) ListApiKeys(
	ctx context.Context,
	req *connect.Request[accountv1.ListApiKeysRequest],
) (*connect.Response[accountv1.ListApiKeysResponse], error) {
	userID := auth.UserID(ctx)

	rows, err := a.Queries.ListApiKeysByUser(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	keys := make([]*accountv1.ApiKeyMetadata, len(rows))
	for i, r := range rows {
		keys[i] = dbApiKeyRowToProto(r)
	}
	return connect.NewResponse(&accountv1.ListApiKeysResponse{Keys: keys}), nil
}

func (a *Account) CreateApiKey(
	ctx context.Context,
	req *connect.Request[accountv1.CreateApiKeyRequest],
) (*connect.Response[accountv1.CreateApiKeyResponse], error) {
	userID := auth.UserID(ctx)

	rawSecret, err := auth.GenerateToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	label := req.Msg.Label
	if label == "" {
		label = defaultKeyLabel
	}

	now := time.Now()
	key, err := a.Queries.CreateApiKey(ctx, db.CreateApiKeyParams{
		UserID:    userID,
		KeyHash:   auth.HashAPIKey(rawSecret),
		Label:     pgtype.Text{String: label, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	meta := &accountv1.ApiKeyMetadata{
		Id:        key.ID,
		Label:     key.Label.String,
		CreatedAt: timestamppb.New(key.CreatedAt.Time),
	}
	return connect.NewResponse(&accountv1.CreateApiKeyResponse{Key: meta, Secret: rawSecret}), nil
}

func (a *Account) RevokeApiKey(
	ctx context.Context,
	req *connect.Request[accountv1.RevokeApiKeyRequest],
) (*connect.Response[accountv1.RevokeApiKeyResponse], error) {
	userID := auth.UserID(ctx)

	_, err := a.Queries.DeleteApiKeyForUser(ctx, db.DeleteApiKeyForUserParams{
		ID:     req.Msg.Id,
		UserID: userID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&accountv1.RevokeApiKeyResponse{}), nil
}

func dbApiKeyRowToProto(r db.ListApiKeysByUserRow) *accountv1.ApiKeyMetadata {
	m := &accountv1.ApiKeyMetadata{Id: r.ID}
	if r.Label.Valid {
		m.Label = r.Label.String
	}
	if r.CreatedAt.Valid {
		m.CreatedAt = timestamppb.New(r.CreatedAt.Time)
	}
	return m
}
