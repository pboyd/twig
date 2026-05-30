package auth_test

import (
	"testing"

	"github.com/pboyd/twig/services/twig/internal/auth"
)

func TestHashPassword_VerifyPassword(t *testing.T) {
	hash, err := auth.HashPassword("s3cr3t")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "s3cr3t" {
		t.Fatal("hash should not equal plaintext")
	}

	if err := auth.VerifyPassword(hash, "s3cr3t"); err != nil {
		t.Errorf("VerifyPassword correct: %v", err)
	}
	if err := auth.VerifyPassword(hash, "wrong"); err == nil {
		t.Error("VerifyPassword wrong password: expected error, got nil")
	}
}

func TestGenerateToken(t *testing.T) {
	tok1, err := auth.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if len(tok1) != 64 {
		t.Errorf("token length = %d, want 64", len(tok1))
	}

	tok2, err := auth.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken second: %v", err)
	}
	if tok1 == tok2 {
		t.Error("two tokens should differ")
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "deadbeef"
	h1 := auth.HashAPIKey(key)
	h2 := auth.HashAPIKey(key)
	if h1 != h2 {
		t.Error("HashAPIKey should be deterministic")
	}
	if h1 == key {
		t.Error("hash should not equal input")
	}
	if len(h1) != 64 {
		t.Errorf("SHA-256 hex = %d chars, want 64", len(h1))
	}
}

func TestEqualConstantTime(t *testing.T) {
	if !auth.EqualConstantTime("abc", "abc") {
		t.Error("equal strings should return true")
	}
	if auth.EqualConstantTime("abc", "xyz") {
		t.Error("different strings should return false")
	}
	if auth.EqualConstantTime("abc", "abcd") {
		t.Error("different length strings should return false")
	}
}
