package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/404nfid/go-svelte-starter-kit/internal/ws"
	"github.com/google/uuid"
)

// WebhookDispatcher is satisfied by *webhook.Service.
// Defined here to avoid an import cycle.
type WebhookDispatcher interface {
	Dispatch(ctx context.Context, userID uuid.UUID, event string, data any)
}

type Service struct {
	repo     *Repository
	hub      *ws.Hub
	webhooks WebhookDispatcher // optional; set after construction
}

func NewService(repo *Repository, hub *ws.Hub) *Service {
	return &Service{repo: repo, hub: hub}
}

func (s *Service) SetWebhookDispatcher(d WebhookDispatcher) {
	s.webhooks = d
}

type wsMessage struct {
	Type string        `json:"type"`
	Data *Notification `json:"data"`
}

// Push creates a notification in the DB and delivers it via WebSocket if the user is online.
func (s *Service) Push(ctx context.Context, userID uuid.UUID, nType, title, body, link string) error {
	n, err := s.repo.Create(ctx, userID, nType, title, body, link)
	if err != nil {
		return fmt.Errorf("push notification: %w", err)
	}

	payload, err := json.Marshal(wsMessage{Type: "notification", Data: n})
	if err == nil {
		s.hub.Send(userID, payload)
	} else {
		slog.Warn("ws: marshal notification payload", "err", err)
	}

	if s.webhooks != nil {
		s.webhooks.Dispatch(ctx, userID, "notification.created", n)
	}

	return nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, page, limit int) (*ListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	ns, total, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		ns = []*Notification{}
	}
	return &ListResponse{Notifications: ns, Total: total, Page: page, Limit: limit}, nil
}

func (s *Service) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}
