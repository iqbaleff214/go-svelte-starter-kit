package whatsapp

import (
	"time"

	"github.com/google/uuid"
)

// ---- Domain models ----

type SessionStatus string

const (
	StatusPending      SessionStatus = "pending"
	StatusConnected    SessionStatus = "connected"
	StatusDisconnected SessionStatus = "disconnected"
	StatusBanned       SessionStatus = "banned"
)

type Session struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Phone       string        `json:"phone"`
	JID         string        `json:"jid"`
	Status      SessionStatus `json:"status"`
	Paused      bool          `json:"paused"`
	LastUsedAt  *time.Time    `json:"last_used_at"`
	SentToday   int           `json:"sent_today"`
	CreatedBy   *uuid.UUID    `json:"created_by,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type MessageStatus string

const (
	MsgQueued  MessageStatus = "queued"
	MsgSent    MessageStatus = "sent"
	MsgFailed  MessageStatus = "failed"
)

type Message struct {
	ID        uuid.UUID     `json:"id"`
	SessionID *uuid.UUID    `json:"session_id"`
	Recipient string        `json:"recipient"`
	Body      string        `json:"body"`
	Status    MessageStatus `json:"status"`
	Error     string        `json:"error,omitempty"`
	QueuedAt  time.Time     `json:"queued_at"`
	SentAt    *time.Time    `json:"sent_at,omitempty"`
	CreatedBy *uuid.UUID    `json:"created_by,omitempty"`
}

// ---- HTTP request types ----

type CreateSessionRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type SendRequest struct {
	Recipient string `json:"recipient" validate:"required"`
	Body      string `json:"body"      validate:"required,min=1,max=4096"`
}

type BatchSendRequest struct {
	Messages []SendRequest `json:"messages" validate:"required,min=1,max=200,dive"`
}

type PairRequest struct {
	Phone string `json:"phone" validate:"required"`
}

// ---- Asynq task ----

const (
	QueueName   = "whatsapp"
	TaskSend    = "whatsapp:send"
	maxRetry    = 3
)

type SendPayload struct {
	MessageID string  `json:"message_id"`
	Recipient string  `json:"recipient"`
	Body      string  `json:"body"`
	CreatedBy *string `json:"created_by,omitempty"`
}
