package notification

import (
	"time"

	"github.com/google/uuid"
)

// Notification types matching the notification_type PG enum.
const (
	TypeInfo    = "info"
	TypeSuccess = "success"
	TypeWarning = "warning"
	TypeAlert   = "alert"
)

type Notification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      *string    `json:"body"`
	Link      *string    `json:"link"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type ListResponse struct {
	Notifications []*Notification `json:"notifications"`
	Total         int             `json:"total"`
	Page          int             `json:"page"`
	Limit         int             `json:"limit"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}
