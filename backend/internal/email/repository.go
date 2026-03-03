package email

import (
	"context"
	"fmt"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/google/uuid"
)

type EmailLog struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	Template  string
	Recipient string
	Status    string
	Error     string
	Attempts  int
	SentAt    *time.Time
	CreatedAt time.Time
}

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// CreateLog inserts an email log with status "queued" and returns the new log ID.
func (r *Repository) CreateLog(ctx context.Context, userID *uuid.UUID, tmpl, recipient string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO email_logs (id, user_id, template, recipient, status, created_at)
		 VALUES ($1, $2, $3, $4, 'queued', NOW())`,
		id, userID, tmpl, recipient,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create email log: %w", err)
	}
	return id, nil
}

// UpdateLog sets the status, error, sent_at, and attempts on a log row.
func (r *Repository) UpdateLog(ctx context.Context, id uuid.UUID, status, errMsg string, sentAt *time.Time, attempts int) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE email_logs SET status=$1, error=$2, sent_at=$3, attempts=$4 WHERE id=$5`,
		status, nullStr(errMsg), sentAt, attempts, id,
	)
	return err
}

// ListLogs returns paginated email logs for the admin panel.
func (r *Repository) ListLogs(ctx context.Context, limit, offset int) ([]*EmailLog, int, error) {
	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, template, recipient, status, COALESCE(error,''), attempts, sent_at, created_at
		 FROM email_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list email logs: %w", err)
	}
	defer rows.Close()

	var logs []*EmailLog
	for rows.Next() {
		l := &EmailLog{}
		if err := rows.Scan(&l.ID, &l.UserID, &l.Template, &l.Recipient,
			&l.Status, &l.Error, &l.Attempts, &l.SentAt, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
