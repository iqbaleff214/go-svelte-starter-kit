package whatsapp

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

// ---- Sessions ----

func (r *Repository) CreateSession(ctx context.Context, name string, createdBy *uuid.UUID) (*Session, error) {
	var s Session
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO whatsapp_sessions (name, created_by)
		VALUES ($1, $2)
		RETURNING id, name, phone, jid, status, paused, last_used_at, sent_today, created_by, created_at, updated_at
	`, name, createdBy).Scan(
		&s.ID, &s.Name, &s.Phone, &s.JID, &s.Status,
		&s.Paused, &s.LastUsedAt, &s.SentToday, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create whatsapp session: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	var s Session
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, phone, jid, status, paused, last_used_at, sent_today, created_by, created_at, updated_at
		FROM whatsapp_sessions WHERE id = $1
	`, id).Scan(
		&s.ID, &s.Name, &s.Phone, &s.JID, &s.Status,
		&s.Paused, &s.LastUsedAt, &s.SentToday, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get whatsapp session: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetSessionByJID(ctx context.Context, jid string) (*Session, error) {
	var s Session
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, phone, jid, status, paused, last_used_at, sent_today, created_by, created_at, updated_at
		FROM whatsapp_sessions WHERE jid = $1
	`, jid).Scan(
		&s.ID, &s.Name, &s.Phone, &s.JID, &s.Status,
		&s.Paused, &s.LastUsedAt, &s.SentToday, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get whatsapp session by jid: %w", err)
	}
	return &s, nil
}

func (r *Repository) ListSessions(ctx context.Context) ([]*Session, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, phone, jid, status, paused, last_used_at, sent_today, created_by, created_at, updated_at
		FROM whatsapp_sessions ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list whatsapp sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		var s Session
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Phone, &s.JID, &s.Status,
			&s.Paused, &s.LastUsedAt, &s.SentToday, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}
	return sessions, nil
}

// ListConnectedSessions returns all active, non-paused sessions for pool restoration.
func (r *Repository) ListConnectedSessions(ctx context.Context) ([]*Session, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, phone, jid, status, paused, last_used_at, sent_today, created_by, created_at, updated_at
		FROM whatsapp_sessions WHERE status = 'connected' AND paused = FALSE ORDER BY created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		var s Session
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Phone, &s.JID, &s.Status,
			&s.Paused, &s.LastUsedAt, &s.SentToday, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}
	return sessions, nil
}

func (r *Repository) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status SessionStatus) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE whatsapp_sessions SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	return err
}

func (r *Repository) UpdateSessionPaired(ctx context.Context, id uuid.UUID, jid, phone string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE whatsapp_sessions SET jid = $1, phone = $2, status = 'connected', updated_at = NOW() WHERE id = $3`,
		jid, phone, id,
	)
	return err
}

func (r *Repository) SetPaused(ctx context.Context, id uuid.UUID, paused bool) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE whatsapp_sessions SET paused = $1, updated_at = NOW() WHERE id = $2`,
		paused, id,
	)
	return err
}

func (r *Repository) RecordUsage(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE whatsapp_sessions SET last_used_at = NOW(), sent_today = sent_today + 1, updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

// ResetDailyCounts zeroes sent_today — call via cron at midnight.
func (r *Repository) ResetDailyCounts(ctx context.Context) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE whatsapp_sessions SET sent_today = 0, updated_at = NOW()`)
	return err
}

func (r *Repository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM whatsapp_sessions WHERE id = $1`, id)
	return err
}

// ---- Messages ----

func (r *Repository) CreateMessage(ctx context.Context, recipient, body string, createdBy *uuid.UUID) (*Message, error) {
	var m Message
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO whatsapp_messages (recipient, body, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, session_id, recipient, body, status, error, queued_at, sent_at, created_by
	`, recipient, body, createdBy).Scan(
		&m.ID, &m.SessionID, &m.Recipient, &m.Body,
		&m.Status, &m.Error, &m.QueuedAt, &m.SentAt, &m.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("create whatsapp message: %w", err)
	}
	return &m, nil
}

func (r *Repository) MarkSent(ctx context.Context, id, sessionID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE whatsapp_messages SET status = 'sent', session_id = $1, sent_at = $2, error = '' WHERE id = $3`,
		sessionID, now, id,
	)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE whatsapp_messages SET status = 'failed', error = $1 WHERE id = $2`,
		errMsg, id,
	)
	return err
}

func (r *Repository) ListMessages(ctx context.Context, limit, offset int, status string) ([]*Message, int, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if status != "" {
		rows, err = r.db.Pool.Query(ctx, `
			SELECT id, session_id, recipient, body, status, error, queued_at, sent_at, created_by
			FROM whatsapp_messages WHERE status = $1 ORDER BY queued_at DESC LIMIT $2 OFFSET $3
		`, status, limit, offset)
	} else {
		rows, err = r.db.Pool.Query(ctx, `
			SELECT id, session_id, recipient, body, status, error, queued_at, sent_at, created_by
			FROM whatsapp_messages ORDER BY queued_at DESC LIMIT $1 OFFSET $2
		`, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	msgs := make([]*Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.Recipient, &m.Body,
			&m.Status, &m.Error, &m.QueuedAt, &m.SentAt, &m.CreatedBy,
		); err != nil {
			return nil, 0, err
		}
		msgs = append(msgs, &m)
	}

	var total int
	if status != "" {
		_ = r.db.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM whatsapp_messages WHERE status = $1`, status,
		).Scan(&total)
	} else {
		_ = r.db.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM whatsapp_messages`,
		).Scan(&total)
	}

	return msgs, total, nil
}
