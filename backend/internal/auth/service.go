package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTokenRefreshFailed = errors.New("could not refresh token")
)

type Service struct {
	repo         *Repository
	tokenManager *token.Manager
	refreshTTL   time.Duration
	accessTTL    time.Duration
}

func NewService(repo *Repository, tokenManager *token.Manager, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		repo:         repo,
		tokenManager: tokenManager,
		refreshTTL:   refreshTTL,
		accessTTL:    accessTTL,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*LoginResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, req.Email, string(hash), req.DisplayName)
	if err != nil {
		return nil, err
	}

	if err := s.repo.AssignDefaultRole(ctx, user.ID); err != nil {
		fmt.Printf("warn: assign default role to user %s: %v\n", user.ID, err)
	}

	roles, _ := s.repo.GetUserRoles(ctx, user.ID)

	pair, err := s.tokenManager.GeneratePair(user.ID.String(), user.Email, roles)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return &LoginResponse{
		User:  toUserResponse(user),
		Token: s.toTokenResponse(pair),
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, r *http.Request) (*LoginResponse, string, error) {
	user, err := s.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	roles, _ := s.repo.GetUserRoles(ctx, user.ID)

	pair, err := s.tokenManager.GeneratePair(user.ID.String(), user.Email, roles)
	if err != nil {
		return nil, "", fmt.Errorf("generate tokens: %w", err)
	}

	tokenHash := hashToken(pair.RefreshToken)
	expiresAt := time.Now().Add(s.refreshTTL)

	_, err = s.repo.CreateSession(ctx, user.ID, tokenHash,
		r.UserAgent(), r.RemoteAddr, expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return &LoginResponse{
		User:  toUserResponse(user),
		Token: s.toTokenResponse(pair),
	}, pair.RefreshToken, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string, r *http.Request) (*LoginResponse, string, error) {
	claims, err := s.tokenManager.VerifyRefresh(refreshToken)
	if err != nil {
		return nil, "", ErrTokenRefreshFailed
	}

	tokenHash := hashToken(refreshToken)
	session, err := s.repo.FindSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, "", ErrTokenRefreshFailed
	}

	// Rotate: revoke old session
	if err := s.repo.RevokeSession(ctx, session.ID); err != nil {
		return nil, "", fmt.Errorf("revoke old session: %w", err)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, "", ErrTokenRefreshFailed
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, "", ErrTokenRefreshFailed
	}

	roles, _ := s.repo.GetUserRoles(ctx, user.ID)

	pair, err := s.tokenManager.GeneratePair(user.ID.String(), user.Email, roles)
	if err != nil {
		return nil, "", fmt.Errorf("generate tokens: %w", err)
	}

	newHash := hashToken(pair.RefreshToken)
	expiresAt := time.Now().Add(s.refreshTTL)

	_, err = s.repo.CreateSession(ctx, user.ID, newHash,
		r.UserAgent(), r.RemoteAddr, expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("create new session: %w", err)
	}

	return &LoginResponse{
		User:  toUserResponse(user),
		Token: s.toTokenResponse(pair),
	}, pair.RefreshToken, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	tokenHash := hashToken(refreshToken)
	return s.repo.RevokeSessionByTokenHash(ctx, tokenHash)
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

func toUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:              u.ID.String(),
		Email:           u.Email,
		DisplayName:     u.DisplayName,
		AvatarURL:       u.AvatarURL,
		EmailVerifiedAt: u.EmailVerifiedAt,
		TwoFAEnabled:    u.TwoFAEnabled,
		CreatedAt:       u.CreatedAt,
	}
}

func (s *Service) toTokenResponse(pair *token.Pair) TokenResponse {
	return TokenResponse{
		AccessToken: pair.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.accessTTL.Seconds()),
	}
}
