package ai

import (
	"time"

	"github.com/google/uuid"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Conversation struct {
	ID         uuid.UUID     `json:"id"`
	Title      string        `json:"title"`
	Model      string        `json:"model"`
	Messages   []ChatMessage `json:"messages"`
	TokenUsage int           `json:"token_usage"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type ConversationSummary struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Model      string    `json:"model"`
	TokenUsage int       `json:"token_usage"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type ChatRequest struct {
	ConversationID *string `json:"conversation_id"`
	Message        string  `json:"message" validate:"required,min=1,max=4000"`
}
