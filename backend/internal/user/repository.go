package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("record not found")

// User mirrors the users table. Kept local to this domain to avoid cross-package coupling.
type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    string
	DisplayName     string
	AvatarURL       *string
	Bio             *string
	EmailVerifiedAt *time.Time
	TwoFAEnabled    bool
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Session mirrors the user_sessions table columns needed for listing.
type Session struct {
	ID               uuid.UUID
	UserAgent        string
	IPAddress        string
	LastSeenAt       time.Time
	CreatedAt        time.Time
}

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash, display_name, avatar_url, bio,
		       email_verified_at, two_fa_enabled, deleted_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	u := &User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.AvatarURL, &u.Bio, &u.EmailVerifiedAt, &u.TwoFAEnabled,
		&u.DeletedAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *Repository) UpdateProfile(ctx context.Context, id uuid.UUID, displayName, bio string) (*User, error) {
	query := `
		UPDATE users SET display_name = $1, bio = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, email, password_hash, display_name, avatar_url, bio,
		          email_verified_at, two_fa_enabled, deleted_at, created_at, updated_at`

	u := &User{}
	err := r.db.Pool.QueryRow(ctx, query, displayName, bio, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.AvatarURL, &u.Bio, &u.EmailVerifiedAt, &u.TwoFAEnabled,
		&u.DeletedAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return u, nil
}

func (r *Repository) UpdateAvatar(ctx context.Context, id uuid.UUID, url string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET avatar_url = $1, updated_at = NOW() WHERE id = $2`,
		url, id)
	return err
}

func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		hash, id)
	return err
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id)
	return err
}

func (r *Repository) ListActiveSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	query := `
		SELECT id, user_agent, ip_address, last_seen_at, created_at
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
		if err := rows.Scan(&s.ID, &s.UserAgent, &s.IPAddress, &s.LastSeenAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = NOW()
		 WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
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
