package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/google/uuid"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new notification and returns it.
func (r *Repository) Create(ctx context.Context, userID uuid.UUID, nType, title, body, link string) (*Notification, error) {
	var bodyPtr, linkPtr *string
	if body != "" {
		bodyPtr = &body
	}
	if link != "" {
		linkPtr = &link
	}

	n := &Notification{}
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO notifications (id, user_id, type, title, body, link, created_at)
		 VALUES ($1, $2, $3::notification_type, $4, $5, $6, NOW())
		 RETURNING id, user_id, type, title, body, link, read_at, created_at`,
		uuid.New(), userID, nType, title, bodyPtr, linkPtr,
	).Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}
	return n, nil
}

// List returns paginated notifications for a user, newest first.
func (r *Repository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, int, error) {
	var total int
	if err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, type, title, body, link, read_at, created_at
		 FROM notifications
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var ns []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		ns = append(ns, n)
	}
	return ns, total, rows.Err()
}

// MarkRead sets read_at for a specific notification belonging to userID.
func (r *Repository) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE notifications SET read_at = $1 WHERE id = $2 AND user_id = $3 AND read_at IS NULL`,
		now, id, userID,
	)
	return err
}

// MarkAllRead sets read_at on all unread notifications for userID.
func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`,
		userID,
	)
	return err
}

// UnreadCount returns the number of unread notifications for userID.
func (r *Repository) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`,
		userID,
	).Scan(&count)
	return count, err
}
