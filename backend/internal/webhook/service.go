package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo   *Repository
	client *http.Client
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo:   repo,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Dispatch finds all active webhooks for the user that subscribe to event and fires
// each delivery in a separate goroutine (fire-and-forget).
func (s *Service) Dispatch(ctx context.Context, userID uuid.UUID, event string, data any) {
	hooks, err := s.repo.FindActiveByUserAndEvent(ctx, userID, event)
	if err != nil {
		slog.Warn("webhook dispatch: query hooks", "err", err, "event", event)
		return
	}
	if len(hooks) == 0 {
		return
	}

	body, err := json.Marshal(Payload{
		ID:        uuid.New().String(),
		Event:     event,
		CreatedAt: time.Now().UTC(),
		Data:      data,
	})
	if err != nil {
		slog.Warn("webhook dispatch: marshal payload", "err", err)
		return
	}

	for _, wh := range hooks {
		go s.deliver(wh, event, body)
	}
}

func (s *Service) deliver(wh *Webhook, event string, body []byte) {
	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodPost, wh.URL, bytes.NewReader(body),
	)
	if err != nil {
		slog.Warn("webhook deliver: build request", "err", err, "url", wh.URL)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("X-Webhook-Delivery", uuid.New().String())

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Warn("webhook deliver: POST failed", "err", err, "url", wh.URL)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("webhook deliver: non-2xx response",
			"status", resp.StatusCode, "url", wh.URL, "event", event)
		return
	}
	slog.Info("webhook delivered", "event", event, "url", wh.URL, "status", resp.StatusCode)
}
