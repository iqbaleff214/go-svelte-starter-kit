package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/internal/email"
	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTokenRefreshFailed = errors.New("could not refresh token")
	ErrTwoFAInvalid       = errors.New("invalid 2FA code")
	ErrTwoFANotEnabled    = errors.New("2FA is not enabled")
	ErrTwoFAAlreadyOn     = errors.New("2FA is already enabled")
)

type Service struct {
	repo         *Repository
	tokenManager *token.Manager
	emailQueue   *email.Queue
	refreshTTL   time.Duration
	accessTTL    time.Duration
}

func NewService(repo *Repository, tokenManager *token.Manager, emailQueue *email.Queue, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		repo:         repo,
		tokenManager: tokenManager,
		emailQueue:   emailQueue,
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

	// Register does not create a tracked session (no device info); use a throwaway session ID.
	sessionID := uuid.New()
	pair, err := s.tokenManager.GeneratePair(user.ID.String(), user.Email, roles, sessionID.String())
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Enqueue welcome + verification emails (fire-and-forget)
	go func() {
		bgCtx := context.Background()
		_ = s.emailQueue.EnqueueWelcome(bgCtx, user.ID, user.Email, user.DisplayName)
		if token, err := s.createVerificationToken(bgCtx, user.ID); err == nil {
			_ = s.emailQueue.EnqueueVerification(bgCtx, user.ID, user.Email, user.DisplayName, token)
		}
	}()

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

	// 2FA: return pre-auth token; no session created yet
	if user.TwoFAEnabled {
		preAuthToken, err := s.tokenManager.GeneratePreAuth(user.ID.String(), user.Email)
		if err != nil {
			return nil, "", fmt.Errorf("generate pre-auth token: %w", err)
		}
		twoFARequired := true
		return &LoginResponse{TwoFARequired: &twoFARequired, PreAuthToken: &preAuthToken}, "", nil
	}

	// Detect new device: send security alert if no existing session matches this user agent
	ua := r.UserAgent()
	isNewDevice := !s.repo.HasSessionByUserAgent(ctx, user.ID, ua)

	roles, _ := s.repo.GetUserRoles(ctx, user.ID)

	sessionID := uuid.New()
	pair, err := s.tokenManager.GeneratePair(user.ID.String(), user.Email, roles, sessionID.String())
	if err != nil {
		return nil, "", fmt.Errorf("generate tokens: %w", err)
	}

	tokenHash := hashToken(pair.RefreshToken)
	expiresAt := time.Now().Add(s.refreshTTL)

	_, err = s.repo.CreateSession(ctx, sessionID, user.ID, tokenHash, ua, r.RemoteAddr, expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	if isNewDevice {
		go func() {
			_ = s.emailQueue.EnqueueSecurityAlert(context.Background(), user.ID, user.Email, user.DisplayName, r.RemoteAddr, ua)
		}()
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

	sessionID := uuid.New()
	pair, err := s.tokenManager.GeneratePair(user.ID.String(), user.Email, roles, sessionID.String())
	if err != nil {
		return nil, "", fmt.Errorf("generate tokens: %w", err)
	}

	newHash := hashToken(pair.RefreshToken)
	expiresAt := time.Now().Add(s.refreshTTL)

	_, err = s.repo.CreateSession(ctx, sessionID, user.ID, newHash,
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

// LoginOrCreateWithOAuth handles Google OAuth: links to existing account or creates a new user.
func (s *Service) LoginOrCreateWithOAuth(ctx context.Context, info *OAuthUserInfo, provider string, r *http.Request) (*LoginResponse, string, error) {
	user, err := s.repo.FindUserByOAuth(ctx, provider, info.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, "", fmt.Errorf("find oauth user: %w", err)
	}

	if errors.Is(err, ErrNotFound) {
		user, err = s.repo.FindUserByEmail(ctx, info.Email)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, "", fmt.Errorf("find user by email: %w", err)
		}

		if errors.Is(err, ErrNotFound) {
			user, err = s.repo.CreateUserWithOAuth(ctx, info.Email, info.Name, info.Picture)
			if err != nil {
				return nil, "", fmt.Errorf("create oauth user: %w", err)
			}
			if err := s.repo.AssignDefaultRole(ctx, user.ID); err != nil {
				fmt.Printf("warn: assign default role to user %s: %v\n", user.ID, err)
			}
			// Welcome new OAuth users
			go func(u *User) {
				_ = s.emailQueue.EnqueueWelcome(context.Background(), u.ID, u.Email, u.DisplayName)
			}(user)
		}
	}

	if err := s.repo.UpsertOAuthProvider(ctx, user.ID, provider, info.ID, info.AccessToken, info.ExpiresAt); err != nil {
		fmt.Printf("warn: upsert oauth provider for user %s: %v\n", user.ID, err)
	}

	roles, _ := s.repo.GetUserRoles(ctx, user.ID)

	sessionID := uuid.New()
	pair, err := s.tokenManager.GeneratePair(user.ID.String(), user.Email, roles, sessionID.String())
	if err != nil {
		return nil, "", fmt.Errorf("generate tokens: %w", err)
	}

	tokenHash := hashToken(pair.RefreshToken)
	expiresAt := time.Now().Add(s.refreshTTL)
	_, err = s.repo.CreateSession(ctx, sessionID, user.ID, tokenHash, r.UserAgent(), r.RemoteAddr, expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return &LoginResponse{
		User:  toUserResponse(user),
		Token: s.toTokenResponse(pair),
	}, pair.RefreshToken, nil
}

// SetupTwoFA generates a TOTP secret and stores it (not yet enabled until ConfirmTwoFA).
func (s *Service) SetupTwoFA(ctx context.Context, userID uuid.UUID, emailAddr string) (*TwoFASetupResponse, error) {
	secret, otpauthURL, err := GenerateSecret(emailAddr, "StarterKit")
	if err != nil {
		return nil, fmt.Errorf("generate totp secret: %w", err)
	}

	if err := s.repo.SetTwoFASecret(ctx, userID, secret); err != nil {
		return nil, fmt.Errorf("store totp secret: %w", err)
	}

	pngBytes, err := GenerateQRCodePNG(otpauthURL)
	if err != nil {
		return nil, fmt.Errorf("generate qr code: %w", err)
	}

	return &TwoFASetupResponse{
		Secret:     secret,
		OTPAuthURL: otpauthURL,
		QRCodePNG:  base64.StdEncoding.EncodeToString(pngBytes),
	}, nil
}

// ConfirmTwoFA validates the TOTP code, enables 2FA, and returns 10 backup codes.
func (s *Service) ConfirmTwoFA(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.TwoFAEnabled {
		return nil, ErrTwoFAAlreadyOn
	}
	if user.TwoFASecret == nil {
		return nil, errors.New("2FA setup not initiated")
	}
	if !ValidateCode(*user.TwoFASecret, code) {
		return nil, ErrTwoFAInvalid
	}

	if err := s.repo.EnableTwoFA(ctx, userID); err != nil {
		return nil, fmt.Errorf("enable 2fa: %w", err)
	}

	plain, hashed := GenerateBackupCodes()
	_ = s.repo.DeleteBackupCodes(ctx, userID)
	if err := s.repo.CreateBackupCodes(ctx, userID, hashed); err != nil {
		return nil, fmt.Errorf("create backup codes: %w", err)
	}

	// Enqueue backup codes email
	if u, err := s.repo.FindUserByID(ctx, userID); err == nil {
		go func() {
			_ = s.emailQueue.EnqueueTwoFABackupCodes(context.Background(), u.ID, u.Email, u.DisplayName, plain)
		}()
	}

	return plain, nil
}

// DisableTwoFA turns off 2FA after verifying a TOTP code or backup code.
func (s *Service) DisableTwoFA(ctx context.Context, userID uuid.UUID, code, backupCode string) error {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.TwoFAEnabled {
		return ErrTwoFANotEnabled
	}

	if code != "" {
		if user.TwoFASecret == nil || !ValidateCode(*user.TwoFASecret, code) {
			return ErrTwoFAInvalid
		}
	} else if backupCode != "" {
		if err := s.repo.FindAndUseBackupCode(ctx, userID, hashToken(backupCode)); err != nil {
			return ErrTwoFAInvalid
		}
	} else {
		return ErrTwoFAInvalid
	}

	if err := s.repo.DisableTwoFA(ctx, userID); err != nil {
		return fmt.Errorf("disable 2fa: %w", err)
	}
	return s.repo.DeleteBackupCodes(ctx, userID)
}

// VerifyTwoFA validates a pre-auth token + TOTP/backup code and returns a full token pair.
func (s *Service) VerifyTwoFA(ctx context.Context, req TwoFAVerifyRequest, r *http.Request) (*LoginResponse, string, error) {
	claims, err := s.tokenManager.VerifyPreAuth(req.PreAuthToken)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if req.Code != "" {
		if user.TwoFASecret == nil || !ValidateCode(*user.TwoFASecret, req.Code) {
			return nil, "", ErrTwoFAInvalid
		}
	} else if req.BackupCode != "" {
		if err := s.repo.FindAndUseBackupCode(ctx, userID, hashToken(req.BackupCode)); err != nil {
			return nil, "", ErrTwoFAInvalid
		}
	} else {
		return nil, "", ErrTwoFAInvalid
	}

	roles, _ := s.repo.GetUserRoles(ctx, user.ID)

	sessionID := uuid.New()
	pair, err := s.tokenManager.GeneratePair(user.ID.String(), user.Email, roles, sessionID.String())
	if err != nil {
		return nil, "", fmt.Errorf("generate tokens: %w", err)
	}

	tokenHash := hashToken(pair.RefreshToken)
	expiresAt := time.Now().Add(s.refreshTTL)
	_, err = s.repo.CreateSession(ctx, sessionID, user.ID, tokenHash, r.UserAgent(), r.RemoteAddr, expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return &LoginResponse{
		User:  toUserResponse(user),
		Token: s.toTokenResponse(pair),
	}, pair.RefreshToken, nil
}

// ---- Password reset / email verification ----

// ForgotPassword creates a reset token and enqueues the reset email.
// Always returns nil to prevent email enumeration.
func (s *Service) ForgotPassword(ctx context.Context, emailAddr string) error {
	user, err := s.repo.FindUserByEmail(ctx, emailAddr)
	if err != nil {
		return nil // silent — don't reveal whether email exists
	}

	plain, tokenHash, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	expiresAt := time.Now().Add(time.Hour)
	if err := s.repo.CreatePasswordReset(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("create password reset: %w", err)
	}

	go func() {
		_ = s.emailQueue.EnqueuePasswordReset(context.Background(), user.ID, user.Email, user.DisplayName, plain)
	}()
	return nil
}

// ResetPassword validates the token and updates the user's password.
func (s *Service) ResetPassword(ctx context.Context, plainToken, newPassword string) error {
	tokenHash := hashToken(plainToken)
	user, err := s.repo.FindPasswordReset(ctx, tokenHash)
	if err != nil {
		return ErrInvalidCredentials
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.UsePasswordReset(ctx, tokenHash, string(newHash)); err != nil {
		return ErrInvalidCredentials
	}

	// Revoke all sessions so attackers can't stay logged in
	_ = s.repo.RevokeAllSessions(ctx, user.ID)
	return nil
}

// VerifyEmail marks the user's email as verified.
func (s *Service) VerifyEmail(ctx context.Context, plainToken string) error {
	tokenHash := hashToken(plainToken)
	user, err := s.repo.FindEmailVerification(ctx, tokenHash)
	if err != nil {
		return ErrInvalidCredentials
	}
	return s.repo.MarkEmailVerified(ctx, user.ID, tokenHash)
}

// ResendVerification creates a new verification token for the given user.
func (s *Service) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.EmailVerifiedAt != nil {
		return errors.New("email already verified")
	}

	plain, err := s.createVerificationToken(ctx, userID)
	if err != nil {
		return err
	}

	go func() {
		_ = s.emailQueue.EnqueueVerification(context.Background(), userID, user.Email, user.DisplayName, plain)
	}()
	return nil
}

// createVerificationToken generates a token, stores its hash, and returns the plain token.
func (s *Service) createVerificationToken(ctx context.Context, userID uuid.UUID) (string, error) {
	plain, tokenHash, err := generateToken()
	if err != nil {
		return "", err
	}
	if err := s.repo.CreateEmailVerification(ctx, userID, tokenHash, time.Now().Add(time.Hour)); err != nil {
		return "", err
	}
	return plain, nil
}

// generateToken creates 32 random bytes, returns hex plain + sha256 hash.
func generateToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	hash = hashToken(plain)
	return plain, hash, nil
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
