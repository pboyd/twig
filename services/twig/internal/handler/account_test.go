package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	accountv1 "github.com/pboyd/twig/api/gen/account/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/handler"
)

// stubAccountQuerier implements handler.AccountQuerier for tests.
type stubAccountQuerier struct {
	user         db.User
	userErr      error
	updateCalled bool
	updateErr    error
	keys         []db.ListApiKeysByUserRow
	keysErr      error
	createdKey       db.ApiKey
	createErr        error
	lastCreateParams db.CreateApiKeyParams
	deleteRows       int64
	deleteErr    error
}

var _ handler.AccountQuerier = (*stubAccountQuerier)(nil)

func (s *stubAccountQuerier) GetUserByID(_ context.Context, _ int64) (db.User, error) {
	return s.user, s.userErr
}

func (s *stubAccountQuerier) UpdateUserPasswordByID(_ context.Context, _ db.UpdateUserPasswordByIDParams) error {
	s.updateCalled = true
	return s.updateErr
}

func (s *stubAccountQuerier) ListApiKeysByUser(_ context.Context, _ int64) ([]db.ListApiKeysByUserRow, error) {
	return s.keys, s.keysErr
}

func (s *stubAccountQuerier) CreateApiKey(_ context.Context, arg db.CreateApiKeyParams) (db.ApiKey, error) {
	s.lastCreateParams = arg
	if s.createErr != nil {
		return db.ApiKey{}, s.createErr
	}
	return db.ApiKey{
		ID:        s.createdKey.ID,
		UserID:    arg.UserID,
		KeyHash:   arg.KeyHash,
		Label:     arg.Label,
		CreatedAt: arg.CreatedAt,
	}, nil
}

func (s *stubAccountQuerier) DeleteApiKeyForUser(_ context.Context, _ db.DeleteApiKeyForUserParams) (int64, error) {
	return s.deleteRows, s.deleteErr
}

// makeAccountHandler builds an Account handler with the given stub and a fresh limiter.
func makeAccountHandler(q *stubAccountQuerier) *handler.Account {
	return &handler.Account{
		Queries: q,
		Limiter: auth.NewLoginLimiter(5, 15*time.Minute),
	}
}

// makeAccountHandlerWithLimiter builds an Account handler with a pre-configured limiter.
func makeAccountHandlerWithLimiter(q *stubAccountQuerier, limiter *auth.LoginLimiter) *handler.Account {
	return &handler.Account{
		Queries: q,
		Limiter: limiter,
	}
}

func makeUser(id int64, username, password string) db.User {
	hash, _ := auth.HashPassword(password)
	return db.User{ID: id, Username: username, PasswordHash: hash}
}

// ---- ChangePassword tests (US1, FR-003, FR-005, FR-006, FR-007, FR-019) ----

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	q := &stubAccountQuerier{
		user: makeUser(1, "alice", "correct-password"),
	}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.ChangePassword(ctx, connect.NewRequest(&accountv1.ChangePasswordRequest{
		CurrentPassword: "wrong-password",
		NewPassword:     "newpassword123",
	}))

	if err == nil {
		t.Fatal("expected error for wrong current password")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestChangePassword_WrongPassword_RecordsLimiterFailure(t *testing.T) {
	q := &stubAccountQuerier{
		user: makeUser(1, "alice", "correct-password"),
	}
	limiter := auth.NewLoginLimiter(1, time.Hour)
	h := makeAccountHandlerWithLimiter(q, limiter)
	ctx := ctxWithUser(1)

	req := connect.NewRequest(&accountv1.ChangePasswordRequest{
		CurrentPassword: "wrong-password",
		NewPassword:     "newpassword123",
	})

	_, _ = h.ChangePassword(ctx, req)

	// Limiter should now be blocking the key after 1 failure.
	blocked, _ := limiter.Blocked("user:alice")
	if !blocked {
		t.Error("expected limiter to block after failure")
	}
}

func TestChangePassword_NewPasswordTooShort(t *testing.T) {
	q := &stubAccountQuerier{
		user: makeUser(1, "alice", "correct-password"),
	}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.ChangePassword(ctx, connect.NewRequest(&accountv1.ChangePasswordRequest{
		CurrentPassword: "correct-password",
		NewPassword:     "short",
	}))

	if err == nil {
		t.Fatal("expected error for short new password")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestChangePassword_HappyPath_WritesNewHash(t *testing.T) {
	q := &stubAccountQuerier{
		user: makeUser(1, "alice", "correct-password"),
	}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.ChangePassword(ctx, connect.NewRequest(&accountv1.ChangePasswordRequest{
		CurrentPassword: "correct-password",
		NewPassword:     "brandnewpassword",
	}))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !q.updateCalled {
		t.Error("UpdateUserPasswordByID was not called")
	}
}

func TestChangePassword_HappyPath_ResetsLimiter(t *testing.T) {
	q := &stubAccountQuerier{
		user: makeUser(1, "alice", "correct-password"),
	}
	limiter := auth.NewLoginLimiter(5, time.Hour)
	// Record a prior failure to ensure reset clears it.
	limiter.RecordFailure("user:alice")
	h := makeAccountHandlerWithLimiter(q, limiter)
	ctx := ctxWithUser(1)

	_, err := h.ChangePassword(ctx, connect.NewRequest(&accountv1.ChangePasswordRequest{
		CurrentPassword: "correct-password",
		NewPassword:     "brandnewpassword",
	}))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocked, _ := limiter.Blocked("user:alice")
	if blocked {
		t.Error("expected limiter to be reset after successful password change")
	}
}

func TestChangePassword_RateLimited(t *testing.T) {
	q := &stubAccountQuerier{
		user: makeUser(1, "alice", "correct-password"),
	}
	limiter := auth.NewLoginLimiter(1, time.Hour)
	// Pre-fill the limiter to trigger blocking.
	limiter.RecordFailure("user:alice")
	limiter.RecordFailure("user:alice")
	h := makeAccountHandlerWithLimiter(q, limiter)
	ctx := ctxWithUser(1)

	_, err := h.ChangePassword(ctx, connect.NewRequest(&accountv1.ChangePasswordRequest{
		CurrentPassword: "correct-password",
		NewPassword:     "brandnewpassword",
	}))

	if err == nil {
		t.Fatal("expected rate-limit error")
	}
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Errorf("code = %v, want ResourceExhausted", connect.CodeOf(err))
	}
}

func TestChangePassword_DBError_GetUser(t *testing.T) {
	q := &stubAccountQuerier{
		userErr: errors.New("db down"),
	}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.ChangePassword(ctx, connect.NewRequest(&accountv1.ChangePasswordRequest{
		CurrentPassword: "any",
		NewPassword:     "newpassword123",
	}))

	if err == nil {
		t.Fatal("expected error on DB failure")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want Internal", connect.CodeOf(err))
	}
}

// ---- CreateApiKey tests (US2, FR-010, FR-012) ----

func TestCreateApiKey_ResponseCarriesSecretAndMetadata(t *testing.T) {
	now := time.Now()
	q := &stubAccountQuerier{
		createdKey: db.ApiKey{
			ID:        42,
			CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		},
	}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	resp, err := h.CreateApiKey(ctx, connect.NewRequest(&accountv1.CreateApiKeyRequest{
		Label: "my key",
	}))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Secret == "" {
		t.Error("expected non-empty secret in response")
	}
	if resp.Msg.Key == nil {
		t.Fatal("expected key metadata in response")
	}
	if resp.Msg.Key.Id != 42 {
		t.Errorf("key id = %d, want 42", resp.Msg.Key.Id)
	}
	if resp.Msg.Key.Label != "my key" {
		t.Errorf("label = %q, want %q", resp.Msg.Key.Label, "my key")
	}
}

func TestCreateApiKey_EmptyLabel_UsesDefault(t *testing.T) {
	q := &stubAccountQuerier{
		createdKey: db.ApiKey{ID: 1},
	}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	resp, err := h.CreateApiKey(ctx, connect.NewRequest(&accountv1.CreateApiKeyRequest{}))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Key.Label != "web key" {
		t.Errorf("label = %q, want %q", resp.Msg.Key.Label, "web key")
	}
}

func TestCreateApiKey_StoredHashMatches(t *testing.T) {
	q := &stubAccountQuerier{createdKey: db.ApiKey{ID: 1}}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	resp, err := h.CreateApiKey(ctx, connect.NewRequest(&accountv1.CreateApiKeyRequest{Label: "cli"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The stored hash must equal HashAPIKey(secret) so the existing GetApiKeyByHash
	// auth path can authenticate CLI requests made with this key (FR-012).
	want := auth.HashAPIKey(resp.Msg.Secret)
	if q.lastCreateParams.KeyHash != want {
		t.Errorf("stored KeyHash = %q, want HashAPIKey(secret) = %q", q.lastCreateParams.KeyHash, want)
	}
}

func TestCreateApiKey_DBError(t *testing.T) {
	q := &stubAccountQuerier{createErr: errors.New("db down")}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.CreateApiKey(ctx, connect.NewRequest(&accountv1.CreateApiKeyRequest{Label: "cli"}))

	if err == nil {
		t.Fatal("expected error on DB failure")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want Internal", connect.CodeOf(err))
	}
}

// ---- ListApiKeys + RevokeApiKey tests (US3, FR-009, FR-014) ----

func TestListApiKeys_ReturnsMetadataOnly(t *testing.T) {
	now := time.Now()
	q := &stubAccountQuerier{
		keys: []db.ListApiKeysByUserRow{
			{ID: 1, Label: pgtype.Text{String: "cli", Valid: true}, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
			{ID: 2, Label: pgtype.Text{String: "web key", Valid: true}, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
		},
	}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	resp, err := h.ListApiKeys(ctx, connect.NewRequest(&accountv1.ListApiKeysRequest{}))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2", len(resp.Msg.Keys))
	}
	if resp.Msg.Keys[0].Id != 1 {
		t.Errorf("keys[0].id = %d, want 1", resp.Msg.Keys[0].Id)
	}
	if resp.Msg.Keys[0].Label != "cli" {
		t.Errorf("keys[0].label = %q, want cli", resp.Msg.Keys[0].Label)
	}
}

func TestListApiKeys_Empty(t *testing.T) {
	q := &stubAccountQuerier{keys: nil}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	resp, err := h.ListApiKeys(ctx, connect.NewRequest(&accountv1.ListApiKeysRequest{}))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Keys) != 0 {
		t.Errorf("expected empty key list, got %d", len(resp.Msg.Keys))
	}
}

func TestListApiKeys_DBError(t *testing.T) {
	q := &stubAccountQuerier{keysErr: errors.New("db down")}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.ListApiKeys(ctx, connect.NewRequest(&accountv1.ListApiKeysRequest{}))

	if err == nil {
		t.Fatal("expected error on DB failure")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want Internal", connect.CodeOf(err))
	}
}

func TestRevokeApiKey_HappyPath(t *testing.T) {
	q := &stubAccountQuerier{deleteRows: 1}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.RevokeApiKey(ctx, connect.NewRequest(&accountv1.RevokeApiKeyRequest{Id: 99}))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRevokeApiKey_AlreadyGone_IsIdempotent(t *testing.T) {
	// Zero rows affected = key already gone; should succeed (idempotent).
	q := &stubAccountQuerier{deleteRows: 0}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.RevokeApiKey(ctx, connect.NewRequest(&accountv1.RevokeApiKeyRequest{Id: 999}))

	if err != nil {
		t.Errorf("expected success (idempotent), got: %v", err)
	}
}

func TestRevokeApiKey_NonOwnedID_IsIdempotentSuccess(t *testing.T) {
	// SQL WHERE user_id = $2 means a foreign id returns 0 rows — treated as success.
	q := &stubAccountQuerier{deleteRows: 0}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.RevokeApiKey(ctx, connect.NewRequest(&accountv1.RevokeApiKeyRequest{Id: 12345}))

	if err != nil {
		t.Errorf("expected success for non-owned id, got: %v", err)
	}
}

func TestRevokeApiKey_DBError(t *testing.T) {
	q := &stubAccountQuerier{deleteErr: errors.New("db failure")}
	h := makeAccountHandler(q)
	ctx := ctxWithUser(1)

	_, err := h.RevokeApiKey(ctx, connect.NewRequest(&accountv1.RevokeApiKeyRequest{Id: 1}))

	if err == nil {
		t.Fatal("expected error on DB failure")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want Internal", connect.CodeOf(err))
	}
}

// ---- Interface compliance ----

var _ handler.AccountQuerier = (*db.Queries)(nil)
