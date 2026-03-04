package apikey

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type APIKeyCreateResponse struct {
	APIKey
	Key string `json:"key"` // plaintext — shown once only
}

type APIKeyLog struct {
	ID         uuid.UUID `json:"id"`
	APIKeyID   uuid.UUID `json:"api_key_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateKeyRequest struct {
	Name      string     `json:"name" validate:"required,min=1,max=100"`
	Scopes    []string   `json:"scopes" validate:"required,min=1"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// ValidatedKey is what the auth middleware gets back after validating a raw key.
type ValidatedKey struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Email  string
	Roles  []string
	Scopes []string
}

// AllScopes is the canonical list of valid API key scopes.
var AllScopes = []string{
	"read:profile", "write:profile",
	"read:notifications", "write:notifications",
	"read:users", "write:users",
	"read:webhooks", "write:webhooks",
}
