package auth

import (
	"encoding/hex"
	"testing"
)

func TestHashToken_Deterministic(t *testing.T) {
	h1 := hashToken("my-secret-token")
	h2 := hashToken("my-secret-token")
	if h1 != h2 {
		t.Errorf("hashToken is not deterministic: %q != %q", h1, h2)
	}
}

func TestHashToken_Different(t *testing.T) {
	h1 := hashToken("token-a")
	h2 := hashToken("token-b")
	if h1 == h2 {
		t.Error("different inputs produced the same hash")
	}
}

func TestHashToken_NonEmpty(t *testing.T) {
	h := hashToken("any-input")
	if h == "" {
		t.Error("expected non-empty hash")
	}
	// SHA-256 produces 32 bytes = 64 hex chars
	if len(h) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(h))
	}
}

func TestGenerateToken_Format(t *testing.T) {
	plain, hash, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if len(plain) != 64 {
		t.Errorf("plain token length: got %d, want 64", len(plain))
	}
	if _, err := hex.DecodeString(plain); err != nil {
		t.Errorf("plain token is not valid hex: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if hash == plain {
		t.Error("hash should differ from plain token")
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	plain1, _, _ := generateToken()
	plain2, _, _ := generateToken()
	if plain1 == plain2 {
		t.Error("two generated tokens should not be equal")
	}
}
