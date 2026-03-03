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

func (r *Repository) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash, userAgent, ip string, expiresAt time.Time) (*Session, error) {
	query := `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, ip_address, last_seen_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6, NOW())
		RETURNING id, user_id, refresh_token_hash, user_agent, ip_address, last_seen_at, expires_at, revoked_at, created_at`

	id := uuid.New()
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
