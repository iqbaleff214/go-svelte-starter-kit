package token

import (
	"testing"
	"time"
)

func newTestManager() *Manager {
	return NewManager("access-secret-test", "refresh-secret-test", 15*time.Minute, 7*24*time.Hour)
}

func TestGeneratePair_AccessToken(t *testing.T) {
	m := newTestManager()
	pair, err := m.GeneratePair("user-1", "test@example.com", []string{"user"}, "session-1")
	if err != nil {
		t.Fatalf("GeneratePair: %v", err)
	}

	claims, err := m.VerifyAccess(pair.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccess: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID: got %q, want %q", claims.UserID, "user-1")
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email: got %q, want %q", claims.Email, "test@example.com")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "user" {
		t.Errorf("Roles: got %v, want [user]", claims.Roles)
	}
	if claims.Type != AccessToken {
		t.Errorf("Type: got %q, want %q", claims.Type, AccessToken)
	}
}

func TestGeneratePair_RefreshToken(t *testing.T) {
	m := newTestManager()
	pair, err := m.GeneratePair("user-2", "other@example.com", nil, "session-2")
	if err != nil {
		t.Fatalf("GeneratePair: %v", err)
	}

	claims, err := m.VerifyRefresh(pair.RefreshToken)
	if err != nil {
		t.Fatalf("VerifyRefresh: %v", err)
	}
	if claims.Type != RefreshToken {
		t.Errorf("Type: got %q, want %q", claims.Type, RefreshToken)
	}
	if claims.SessionID != "session-2" {
		t.Errorf("SessionID: got %q, want %q", claims.SessionID, "session-2")
	}
}

func TestVerifyAccess_Expired(t *testing.T) {
	m := NewManager("access-secret", "refresh-secret", time.Millisecond, 7*24*time.Hour)
	pair, err := m.GeneratePair("user-3", "exp@example.com", nil, "")
	if err != nil {
		t.Fatalf("GeneratePair: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	_, err = m.VerifyAccess(pair.AccessToken)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestVerifyAccess_WrongType(t *testing.T) {
	m := newTestManager()
	pair, err := m.GeneratePair("user-4", "x@example.com", nil, "")
	if err != nil {
		t.Fatalf("GeneratePair: %v", err)
	}
	// Passing a refresh token to VerifyAccess should fail
	_, err = m.VerifyAccess(pair.RefreshToken)
	if err != ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestVerifyRefresh_WrongType(t *testing.T) {
	m := newTestManager()
	pair, err := m.GeneratePair("user-5", "y@example.com", nil, "")
	if err != nil {
		t.Fatalf("GeneratePair: %v", err)
	}
	// Passing an access token to VerifyRefresh should fail
	_, err = m.VerifyRefresh(pair.AccessToken)
	if err != ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestGeneratePreAuth_Valid(t *testing.T) {
	m := newTestManager()
	token, err := m.GeneratePreAuth("user-6", "pre@example.com")
	if err != nil {
		t.Fatalf("GeneratePreAuth: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty pre-auth token")
	}

	claims, err := m.VerifyPreAuth(token)
	if err != nil {
		t.Fatalf("VerifyPreAuth: %v", err)
	}
	if claims.UserID != "user-6" {
		t.Errorf("UserID: got %q, want %q", claims.UserID, "user-6")
	}
	if claims.Type != PreAuthToken {
		t.Errorf("Type: got %q, want %q", claims.Type, PreAuthToken)
	}
}

func TestVerifyPreAuth_WrongType(t *testing.T) {
	m := newTestManager()
	pair, err := m.GeneratePair("user-7", "z@example.com", nil, "")
	if err != nil {
		t.Fatalf("GeneratePair: %v", err)
	}
	// Passing an access token to VerifyPreAuth should fail
	_, err = m.VerifyPreAuth(pair.AccessToken)
	if err != ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestVerifyAccess_InvalidString(t *testing.T) {
	m := newTestManager()
	_, err := m.VerifyAccess("not.a.valid.token")
	if err == nil {
		t.Error("expected error for invalid token string")
	}
}
