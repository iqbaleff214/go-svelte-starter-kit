package webhook

import (
	"time"

	"github.com/google/uuid"
)

// Supported event names.
const (
	EventNotificationCreated = "notification.created"
)

// Payload is the envelope POSTed to webhook URLs.
type Payload struct {
	ID        string    `json:"id"`
	Event     string    `json:"event"`
	CreatedAt time.Time `json:"created_at"`
	Data      any       `json:"data"`
}

type Webhook struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateWebhookRequest struct {
	URL    string   `json:"url" validate:"required,url"`
	Events []string `json:"events" validate:"required,min=1"`
}
