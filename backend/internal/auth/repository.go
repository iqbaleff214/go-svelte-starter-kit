package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    string
	DisplayName     string
	AvatarURL       *string
	Bio             *string
	EmailVerifiedAt *time.Time
	TwoFAEnabled    bool
	TwoFASecret     *string
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	LastSeenAt       time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
}

var ErrNotFound = errors.New("record not found")
var ErrDuplicateEmail = errors.New("email already in use")

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, displayName string) (*User, error) {
	query := `
		INSERT INTO users (id, email, password_hash, display_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, email, password_hash, display_name, avatar_url, bio,
		          email_verified_at, two_fa_enabled, two_fa_secret,
		          deleted_at, created_at, updated_at`

	id := uuid.New()
	row := r.db.Pool.QueryRow(ctx, query, id, email, passwordHash, displayName)

	u, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, display_name, avatar_url, bio,
		       email_verified_at, two_fa_enabled, two_fa_secret,
		       deleted_at, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	row := r.db.Pool.QueryRow(ctx, query, email)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return u, nil
}

func (r *Repository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash, display_name, avatar_url, bio,
		       email_verified_at, two_fa_enabled, two_fa_secret,
		       deleted_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	row := r.db.Pool.QueryRow(ctx, query, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return u, nil
}

func (r *Repository) CreateSession(ctx context.Context, id, userID uuid.UUID, tokenHash, userAgent, ip string, expiresAt time.Time) (*Session, error) {
	query := `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, ip_address, last_seen_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6, NOW())
		RETURNING id, user_id, refresh_token_hash, user_agent, ip_address, last_seen_at, expires_at, revoked_at, created_at`

	row := r.db.Pool.QueryRow(ctx, query, id, userID, tokenHash, userAgent, ip, expiresAt)

	s, err := scanSession(row)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return s, nil
}

func (r *Repository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, last_seen_at, expires_at, revoked_at, created_at
		FROM user_sessions
		WHERE refresh_token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()`

	row := r.db.Pool.QueryRow(ctx, query, tokenHash)
	s, err := scanSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	return s, nil
}

func (r *Repository) RevokeSession(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

func (r *Repository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	query := `UPDATE user_sessions SET revoked_at = NOW() WHERE refresh_token_hash = $1`
	_, err := r.db.Pool.Exec(ctx, query, tokenHash)
	return err
}

func (r *Repository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT r.name
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}

	return roles, rows.Err()
}

func (r *Repository) AssignDefaultRole(ctx context.Context, userID uuid.UUID) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_at)
		SELECT $1, id, NOW() FROM roles WHERE name = 'user'`

	_, err := r.db.Pool.Exec(ctx, query, userID)
	return err
}

// ---- OAuth ----

func (r *Repository) FindUserByOAuth(ctx context.Context, provider, providerUserID string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.avatar_url, u.bio,
		       u.email_verified_at, u.two_fa_enabled, u.two_fa_secret,
		       u.deleted_at, u.created_at, u.updated_at
		FROM users u
		JOIN oauth_providers op ON op.user_id = u.id
		WHERE op.provider = $1 AND op.provider_user_id = $2 AND u.deleted_at IS NULL`

	row := r.db.Pool.QueryRow(ctx, query, provider, providerUserID)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by oauth: %w", err)
	}
	return u, nil
}

func (r *Repository) CreateUserWithOAuth(ctx context.Context, email, displayName, avatarURL string) (*User, error) {
	query := `
		INSERT INTO users (id, email, display_name, avatar_url, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW())
		RETURNING id, email, password_hash, display_name, avatar_url, bio,
		          email_verified_at, two_fa_enabled, two_fa_secret,
		          deleted_at, created_at, updated_at`

	id := uuid.New()
	var av *string
	if avatarURL != "" {
		av = &avatarURL
	}
	row := r.db.Pool.QueryRow(ctx, query, id, email, displayName, av)
	u, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("create oauth user: %w", err)
	}
	return u, nil
}

func (r *Repository) UpsertOAuthProvider(ctx context.Context, userID uuid.UUID, provider, providerUserID, accessToken string, expiresAt *time.Time) error {
	query := `
		INSERT INTO oauth_providers (id, user_id, provider, provider_user_id, access_token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (provider, provider_user_id)
		DO UPDATE SET access_token = EXCLUDED.access_token, expires_at = EXCLUDED.expires_at`

	_, err := r.db.Pool.Exec(ctx, query, uuid.New(), userID, provider, providerUserID, accessToken, expiresAt)
	return err
}

// ---- 2FA ----

func (r *Repository) SetTwoFASecret(ctx context.Context, userID uuid.UUID, secret string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET two_fa_secret = $1, updated_at = NOW() WHERE id = $2`,
		secret, userID)
	return err
}

func (r *Repository) EnableTwoFA(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET two_fa_enabled = TRUE, updated_at = NOW() WHERE id = $1`,
		userID)
	return err
}

func (r *Repository) DisableTwoFA(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET two_fa_enabled = FALSE, two_fa_secret = NULL, updated_at = NOW() WHERE id = $1`,
		userID)
	return err
}

func (r *Repository) CreateBackupCodes(ctx context.Context, userID uuid.UUID, hashes []string) error {
	for _, h := range hashes {
		_, err := r.db.Pool.Exec(ctx,
			`INSERT INTO two_fa_backup_codes (id, user_id, code_hash, created_at) VALUES ($1, $2, $3, NOW())`,
			uuid.New(), userID, h)
		if err != nil {
			return fmt.Errorf("create backup code: %w", err)
		}
	}
	return nil
}

func (r *Repository) FindAndUseBackupCode(ctx context.Context, userID uuid.UUID, codeHash string) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE two_fa_backup_codes SET used_at = NOW()
		 WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL`,
		userID, codeHash)
	if err != nil {
		return fmt.Errorf("use backup code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteBackupCodes(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM two_fa_backup_codes WHERE user_id = $1`, userID)
	return err
}

// ---- Sessions (for user domain) ----

func (r *Repository) ListActiveSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, last_seen_at, expires_at, revoked_at, created_at
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY last_seen_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		s := &Session{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent,
			&s.IPAddress, &s.LastSeenAt, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *Repository) RevokeSessionByID(ctx context.Context, sessionID, userID uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		sessionID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) RevokeAllOtherSessions(ctx context.Context, userID, currentSessionID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = NOW()
		 WHERE user_id = $1 AND id != $2 AND revoked_at IS NULL`,
		userID, currentSessionID)
	return err
}

func (r *Repository) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`,
		userID)
	return err
}

// HasSessionByUserAgent returns true if the user has an active session from the given user agent.
func (r *Repository) HasSessionByUserAgent(ctx context.Context, userID uuid.UUID, userAgent string) bool {
	var count int
	r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_sessions WHERE user_id=$1 AND user_agent=$2 AND revoked_at IS NULL AND expires_at > NOW()`,
		userID, userAgent).Scan(&count) //nolint:errcheck
	return count > 0
}

// ---- Email verification ----

func (r *Repository) CreateEmailVerification(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO email_verifications (id, user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (token_hash) DO NOTHING`,
		uuid.New(), userID, tokenHash, expiresAt,
	)
	return err
}

func (r *Repository) FindEmailVerification(ctx context.Context, tokenHash string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.avatar_url, u.bio,
		       u.email_verified_at, u.two_fa_enabled, u.two_fa_secret,
		       u.deleted_at, u.created_at, u.updated_at
		FROM email_verifications ev
		JOIN users u ON u.id = ev.user_id
		WHERE ev.token_hash = $1
		  AND ev.used_at IS NULL
		  AND ev.expires_at > NOW()
		  AND u.deleted_at IS NULL`

	row := r.db.Pool.QueryRow(ctx, query, tokenHash)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find email verification: %w", err)
	}
	return u, nil
}

func (r *Repository) MarkEmailVerified(ctx context.Context, userID uuid.UUID, tokenHash string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE email_verifications SET used_at = NOW() WHERE token_hash = $1`,
		tokenHash,
	)
	if err != nil {
		return err
	}
	_, err = r.db.Pool.Exec(ctx,
		`UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = $1`,
		userID,
	)
	return err
}

// ---- Password reset ----

func (r *Repository) CreatePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO password_resets (id, user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (token_hash) DO NOTHING`,
		uuid.New(), userID, tokenHash, expiresAt,
	)
	return err
}

func (r *Repository) FindPasswordReset(ctx context.Context, tokenHash string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.avatar_url, u.bio,
		       u.email_verified_at, u.two_fa_enabled, u.two_fa_secret,
		       u.deleted_at, u.created_at, u.updated_at
		FROM password_resets pr
		JOIN users u ON u.id = pr.user_id
		WHERE pr.token_hash = $1
		  AND pr.used_at IS NULL
		  AND pr.expires_at > NOW()
		  AND u.deleted_at IS NULL`

	row := r.db.Pool.QueryRow(ctx, query, tokenHash)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find password reset: %w", err)
	}
	return u, nil
}

func (r *Repository) UsePasswordReset(ctx context.Context, tokenHash, newPasswordHash string) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID uuid.UUID
	if err := tx.QueryRow(ctx,
		`UPDATE password_resets SET used_at = NOW()
		 WHERE token_hash = $1 AND used_at IS NULL
		 RETURNING user_id`, tokenHash).Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("mark reset used: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		newPasswordHash, userID); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return tx.Commit(ctx)
}

func scanUser(row pgx.Row) (*User, error) {
	u := &User{}
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.AvatarURL, &u.Bio, &u.EmailVerifiedAt,
		&u.TwoFAEnabled, &u.TwoFASecret,
		&u.DeletedAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func scanSession(row pgx.Row) (*Session, error) {
	s := &Session{}
	err := row.Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent,
		&s.IPAddress, &s.LastSeenAt, &s.ExpiresAt,
		&s.RevokedAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := fmt.Sprintf("%v", err)
	return strings.Contains(msg, "23505") || strings.Contains(msg, "unique constraint")
}
