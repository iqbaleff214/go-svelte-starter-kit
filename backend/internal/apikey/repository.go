package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new API key row and returns it (without key_hash).
func (r *Repository) Create(
	ctx context.Context,
	userID uuid.UUID,
	name, keyHash, keyPrefix string,
	scopes []string,
	expiresAt *time.Time,
) (*APIKey, error) {
	key := &APIKey{}
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, scopes, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 RETURNING id, name, key_prefix, scopes, last_used_at, expires_at, revoked_at, created_at`,
		uuid.New(), userID, name, keyHash, keyPrefix, scopes, expiresAt,
	).Scan(&key.ID, &key.Name, &key.KeyPrefix, &key.Scopes,
		&key.LastUsedAt, &key.ExpiresAt, &key.RevokedAt, &key.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	if key.Scopes == nil {
		key.Scopes = []string{}
	}
	return key, nil
}

// GetByHash looks up a key by its SHA-256 hash, returning the ValidatedKey
// with user email and roles resolved in a single JOIN query.
func (r *Repository) GetByHash(ctx context.Context, hash string) (*ValidatedKey, error) {
	type row struct {
		ID        uuid.UUID
		UserID    uuid.UUID
		Scopes    []string
		RevokedAt *time.Time
		ExpiresAt *time.Time
		Email     string
		Deleted   bool
		Roles     []string
	}

	var res row
	err := r.db.Pool.QueryRow(ctx,
		`SELECT k.id, k.user_id, k.scopes, k.revoked_at, k.expires_at,
		        u.email, u.deleted_at IS NOT NULL,
		        COALESCE(array_agg(ro.name) FILTER (WHERE ro.name IS NOT NULL), '{}')
		 FROM api_keys k
		 JOIN users u ON u.id = k.user_id
		 LEFT JOIN user_roles ur ON ur.user_id = k.user_id
		 LEFT JOIN roles ro ON ro.id = ur.role_id
		 WHERE k.key_hash = $1
		 GROUP BY k.id, k.user_id, k.scopes, k.revoked_at, k.expires_at, u.email, u.deleted_at`,
		hash,
	).Scan(&res.ID, &res.UserID, &res.Scopes, &res.RevokedAt, &res.ExpiresAt,
		&res.Email, &res.Deleted, &res.Roles)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("get api key: %w", err)
	}

	if res.Deleted {
		return nil, fmt.Errorf("user account deleted")
	}
	if res.RevokedAt != nil {
		return nil, fmt.Errorf("api key revoked")
	}
	if res.ExpiresAt != nil && time.Now().After(*res.ExpiresAt) {
		return nil, fmt.Errorf("api key expired")
	}
	if res.Scopes == nil {
		res.Scopes = []string{}
	}
	if res.Roles == nil {
		res.Roles = []string{}
	}
	return &ValidatedKey{
		ID:     res.ID,
		UserID: res.UserID,
		Email:  res.Email,
		Roles:  res.Roles,
		Scopes: res.Scopes,
	}, nil
}

// ListByUser returns all API keys for a user (no key_hash).
func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, name, key_prefix, scopes, last_used_at, expires_at, revoked_at, created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		k := &APIKey{}
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.Scopes,
			&k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		if k.Scopes == nil {
			k.Scopes = []string{}
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []*APIKey{}
	}
	return keys, rows.Err()
}

// Revoke sets revoked_at for a key belonging to userID.
func (r *Repository) Revoke(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

// UpdateLastUsed sets last_used_at to NOW(). Safe to call concurrently.
func (r *Repository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

// CreateLog inserts an audit log entry.
func (r *Repository) CreateLog(ctx context.Context, log *APIKeyLog) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO api_key_logs (id, api_key_id, method, path, status_code, ip, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		uuid.New(), log.APIKeyID, log.Method, log.Path, log.StatusCode, log.IP,
	)
	return err
}

// ListLogs returns paginated logs for a specific key, verifying userID ownership.
func (r *Repository) ListLogs(ctx context.Context, keyID, userID uuid.UUID, limit, offset int) ([]*APIKeyLog, int, error) {
	// Verify ownership
	var ownerID uuid.UUID
	if err := r.db.Pool.QueryRow(ctx,
		`SELECT user_id FROM api_keys WHERE id = $1`, keyID,
	).Scan(&ownerID); err != nil || ownerID != userID {
		return nil, 0, fmt.Errorf("api key not found")
	}

	var total int
	if err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM api_key_logs WHERE api_key_id = $1`, keyID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, api_key_id, method, path, status_code, COALESCE(ip,''), created_at
		 FROM api_key_logs WHERE api_key_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		keyID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list logs: %w", err)
	}
	defer rows.Close()

	var logs []*APIKeyLog
	for rows.Next() {
		l := &APIKeyLog{}
		if err := rows.Scan(&l.ID, &l.APIKeyID, &l.Method, &l.Path,
			&l.StatusCode, &l.IP, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []*APIKeyLog{}
	}
	return logs, total, rows.Err()
}
