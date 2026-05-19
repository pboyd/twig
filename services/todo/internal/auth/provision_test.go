package auth_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pboyd/todo/services/todo/internal/auth"
	"github.com/pboyd/todo/services/todo/internal/db"
)

// stubProvisionQuerier implements auth.ProvisionQuerier for tests.
type stubProvisionQuerier struct {
	users   map[string]*db.User
	apiKeys map[int64][]db.ApiKey
}

func newStubProvisionQuerier() *stubProvisionQuerier {
	return &stubProvisionQuerier{
		users:   make(map[string]*db.User),
		apiKeys: make(map[int64][]db.ApiKey),
	}
}

func (s *stubProvisionQuerier) GetUserByUsername(_ context.Context, username string) (db.User, error) {
	u, ok := s.users[username]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return *u, nil
}

func (s *stubProvisionQuerier) CreateUser(_ context.Context, arg db.CreateUserParams) (db.User, error) {
	u := db.User{
		ID:           int64(len(s.users) + 1),
		Username:     arg.Username,
		PasswordHash: arg.PasswordHash,
	}
	s.users[arg.Username] = &u
	return u, nil
}

func (s *stubProvisionQuerier) UpdateUserPassword(_ context.Context, arg db.UpdateUserPasswordParams) error {
	u, ok := s.users[arg.Username]
	if !ok {
		return pgx.ErrNoRows
	}
	u.PasswordHash = arg.PasswordHash
	return nil
}

func (s *stubProvisionQuerier) DeleteApiKeysByUser(_ context.Context, userID int64) error {
	delete(s.apiKeys, userID)
	return nil
}

func (s *stubProvisionQuerier) CreateApiKey(_ context.Context, arg db.CreateApiKeyParams) (db.ApiKey, error) {
	key := db.ApiKey{
		ID:        int64(len(s.apiKeys) + 1),
		UserID:    arg.UserID,
		KeyHash:   arg.KeyHash,
		Label:     arg.Label,
		CreatedAt: pgtype.Timestamptz{},
	}
	s.apiKeys[arg.UserID] = append(s.apiKeys[arg.UserID], key)
	return key, nil
}

var _ auth.ProvisionQuerier = (*stubProvisionQuerier)(nil)
var _ auth.ProvisionQuerier = (*db.Queries)(nil)

func TestProvision_NewUser(t *testing.T) {
	q := newStubProvisionQuerier()
	rawKey, err := auth.ProvisionUser(context.Background(), q, "alice", "s3cr3t")
	if err != nil {
		t.Fatalf("ProvisionUser: %v", err)
	}
	if len(rawKey) != 64 {
		t.Errorf("raw key length = %d, want 64", len(rawKey))
	}

	u, ok := q.users["alice"]
	if !ok {
		t.Fatal("user alice not created")
	}
	if err := auth.VerifyPassword(u.PasswordHash, "s3cr3t"); err != nil {
		t.Error("password hash should match s3cr3t")
	}

	keys := q.apiKeys[u.ID]
	if len(keys) != 1 {
		t.Fatalf("expected 1 api key, got %d", len(keys))
	}
	if keys[0].KeyHash != auth.HashAPIKey(rawKey) {
		t.Error("stored key_hash should be SHA-256 of raw key")
	}
}

func TestProvision_ExistingUserUpdatesPasswordAndKey(t *testing.T) {
	q := newStubProvisionQuerier()

	// First provision.
	rawKey1, err := auth.ProvisionUser(context.Background(), q, "bob", "oldpass")
	if err != nil {
		t.Fatalf("first ProvisionUser: %v", err)
	}

	// Re-provision with new password.
	rawKey2, err := auth.ProvisionUser(context.Background(), q, "bob", "newpass")
	if err != nil {
		t.Fatalf("second ProvisionUser: %v", err)
	}

	if rawKey1 == rawKey2 {
		t.Error("re-provisioning should issue a new API key")
	}

	u := q.users["bob"]
	if err := auth.VerifyPassword(u.PasswordHash, "newpass"); err != nil {
		t.Error("password should have been updated to newpass")
	}
	if err := auth.VerifyPassword(u.PasswordHash, "oldpass"); err == nil {
		t.Error("old password should no longer work")
	}

	keys := q.apiKeys[u.ID]
	if len(keys) != 1 {
		t.Errorf("expected 1 api key after re-provision, got %d", len(keys))
	}
	if keys[0].KeyHash == auth.HashAPIKey(rawKey1) {
		t.Error("old key should have been deleted")
	}
	if keys[0].KeyHash != auth.HashAPIKey(rawKey2) {
		t.Error("new key_hash should match new raw key")
	}
}
