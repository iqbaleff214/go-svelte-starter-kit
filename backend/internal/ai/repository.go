package ai

import (
	"context"
	"encoding/json"
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

func (r *Repository) CreateConversation(ctx context.Context, userID uuid.UUID, title, model string, msgs []ChatMessage) (*Conversation, error) {
	id := uuid.New()
	msgsJSON, err := json.Marshal(msgs)
	if err != nil {
		return nil, fmt.Errorf("marshal messages: %w", err)
	}

	conv := &Conversation{ID: id, Title: title, Model: model, Messages: msgs}
	err = r.db.Pool.QueryRow(ctx,
		`INSERT INTO ai_conversations (id, user_id, title, messages, model, token_usage, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 0, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		id, userID, title, msgsJSON, model,
	).Scan(&conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return conv, nil
}

func (r *Repository) GetConversation(ctx context.Context, id, userID uuid.UUID) (*Conversation, error) {
	var msgsRaw []byte
	conv := &Conversation{}
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, title, model, messages, token_usage, created_at, updated_at
		 FROM ai_conversations WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&conv.ID, &conv.Title, &conv.Model, &msgsRaw, &conv.TokenUsage, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if err := json.Unmarshal(msgsRaw, &conv.Messages); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}
	if conv.Messages == nil {
		conv.Messages = []ChatMessage{}
	}
	return conv, nil
}

func (r *Repository) UpdateConversation(ctx context.Context, id uuid.UUID, msgs []ChatMessage, tokenUsage int) error {
	msgsJSON, err := json.Marshal(msgs)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}
	_, err = r.db.Pool.Exec(ctx,
		`UPDATE ai_conversations
		 SET messages = $2, token_usage = token_usage + $3, updated_at = NOW()
		 WHERE id = $1`,
		id, msgsJSON, tokenUsage,
	)
	return err
}

func (r *Repository) ListConversations(ctx context.Context, userID uuid.UUID) ([]*ConversationSummary, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, title, model, token_usage, updated_at, created_at
		 FROM ai_conversations
		 WHERE user_id = $1
		 ORDER BY updated_at DESC
		 LIMIT 50`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var convs []*ConversationSummary
	for rows.Next() {
		c := &ConversationSummary{}
		if err := rows.Scan(&c.ID, &c.Title, &c.Model, &c.TokenUsage, &c.UpdatedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	if convs == nil {
		convs = []*ConversationSummary{}
	}
	return convs, rows.Err()
}

func (r *Repository) DeleteConversation(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx,
		`DELETE FROM ai_conversations WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

func (r *Repository) PurgeOldConversations(ctx context.Context, ttl time.Duration) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM ai_conversations WHERE updated_at < NOW() - $1::interval`,
		ttl.String(),
	)
	return err
}
