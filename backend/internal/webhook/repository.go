package webhook

import (
	"context"
	"fmt"

	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/google/uuid"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, url string, events []string) (*Webhook, error) {
	wh := &Webhook{}
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO webhooks (id, user_id, url, events, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW())
		 RETURNING id, url, events, active, created_at, updated_at`,
		uuid.New(), userID, url, events,
	).Scan(&wh.ID, &wh.URL, &wh.Events, &wh.Active, &wh.CreatedAt, &wh.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create webhook: %w", err)
	}
	if wh.Events == nil {
		wh.Events = []string{}
	}
	return wh, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Webhook, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, url, events, active, created_at, updated_at
		 FROM webhooks WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	var whs []*Webhook
	for rows.Next() {
		wh := &Webhook{}
		if err := rows.Scan(&wh.ID, &wh.URL, &wh.Events, &wh.Active, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			return nil, err
		}
		if wh.Events == nil {
			wh.Events = []string{}
		}
		whs = append(whs, wh)
	}
	if whs == nil {
		whs = []*Webhook{}
	}
	return whs, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx,
		`DELETE FROM webhooks WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}
