package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Type string

const (
	AccessToken   Type = "access"
	RefreshToken  Type = "refresh"
	PreAuthToken  Type = "pre_auth"
)

type Claims struct {
	UserID    string   `json:"sub"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Type      Type     `json:"type"`
	SessionID string   `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

type Pair struct {
	AccessToken  string
	RefreshToken string
}

type Manager struct {
	accessSecret  string
	refreshSecret string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (m *Manager) GeneratePair(userID, email string, roles []string, sessionID string) (*Pair, error) {
	access, err := m.generate(userID, email, roles, sessionID, AccessToken, m.accessSecret, m.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refresh, err := m.generate(userID, email, roles, sessionID, RefreshToken, m.refreshSecret, m.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &Pair{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

// GeneratePreAuth creates a short-lived token used when 2FA is required before a full login.
func (m *Manager) GeneratePreAuth(userID, email string) (string, error) {
	return m.generate(userID, email, nil, "", PreAuthToken, m.accessSecret, 5*time.Minute)
}

// VerifyPreAuth validates a pre-auth token and returns its claims.
func (m *Manager) VerifyPreAuth(tokenStr string) (*Claims, error) {
	return m.verify(tokenStr, m.accessSecret, PreAuthToken)
}

func (m *Manager) generate(userID, email string, roles []string, sessionID string, tokenType Type, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Roles:     roles,
		Type:      tokenType,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (m *Manager) VerifyAccess(tokenStr string) (*Claims, error) {
	return m.verify(tokenStr, m.accessSecret, AccessToken)
}

func (m *Manager) VerifyRefresh(tokenStr string) (*Claims, error) {
	return m.verify(tokenStr, m.refreshSecret, RefreshToken)
}

func (m *Manager) verify(tokenStr, secret string, expectedType Type) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	if claims.Type != expectedType {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

var (
	ErrTokenExpired = errors.New("token has expired")
	ErrTokenInvalid = errors.New("token is invalid")
)
