package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type Service struct {
	repo    *Repository
	manager *Manager
	queue   *asynq.Client
	log     *slog.Logger
}

func NewService(repo *Repository, manager *Manager, redisURL string, log *slog.Logger) *Service {
	opt, _ := asynq.ParseRedisURI(redisURL)
	return &Service{
		repo:    repo,
		manager: manager,
		queue:   asynq.NewClient(opt),
		log:     log,
	}
}

func (s *Service) Close() { _ = s.queue.Close() }

// ---- Session management ----

func (s *Service) CreateSession(ctx context.Context, req CreateSessionRequest, userID *uuid.UUID) (*Session, error) {
	return s.repo.CreateSession(ctx, req.Name, userID)
}

func (s *Service) ListSessions(ctx context.Context) ([]*Session, error) {
	return s.repo.ListSessions(ctx)
}

func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	return s.repo.GetSession(ctx, id)
}

func (s *Service) DeleteSession(ctx context.Context, id uuid.UUID) error {
	s.manager.DisconnectSession(ctx, id)
	return s.repo.DeleteSession(ctx, id)
}

func (s *Service) PauseSession(ctx context.Context, id uuid.UUID) error {
	return s.repo.SetPaused(ctx, id, true)
}

func (s *Service) ResumeSession(ctx context.Context, id uuid.UUID) error {
	return s.repo.SetPaused(ctx, id, false)
}

// StartQR delegates to the manager and returns the QR event channel.
// When the "connected" event fires, an immediate sync_pool task is enqueued
// so the worker picks up the newly paired session without waiting for the
// periodic SyncPool tick.
func (s *Service) StartQR(ctx context.Context, sessionID uuid.UUID) (<-chan QREvent, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("whatsapp manager not initialised (check server logs)")
	}
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == StatusConnected {
		return nil, fmt.Errorf("session already connected")
	}
	inner, err := s.manager.StartQR(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	out := make(chan QREvent, 16)
	go func() {
		defer close(out)
		for evt := range inner {
			out <- evt
			if evt.Type == "connected" {
				_ = s.enqueueSyncPool()
			}
		}
	}()
	return out, nil
}

func (s *Service) enqueueSyncPool() error {
	task := asynq.NewTask(TaskSyncPool, nil, asynq.Queue(QueueName), asynq.MaxRetry(0))
	_, err := s.queue.Enqueue(task)
	return err
}

// GetPairingCode returns a phone-number-based 8-digit pairing code.
func (s *Service) GetPairingCode(ctx context.Context, sessionID uuid.UUID, phone string) (string, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if sess.Status == StatusConnected {
		return "", fmt.Errorf("session already connected")
	}
	return s.manager.GetPairingCode(ctx, sessionID, phone)
}

// ---- Messaging ----

// Enqueue adds a single message to the send queue.
func (s *Service) Enqueue(ctx context.Context, req SendRequest, userID *uuid.UUID) (*Message, error) {
	msg, err := s.repo.CreateMessage(ctx, req.Recipient, req.Body, userID)
	if err != nil {
		return nil, err
	}
	if err := s.enqueueTask(ctx, msg, userID); err != nil {
		_ = s.repo.MarkFailed(ctx, msg.ID, "enqueue failed: "+err.Error())
		return nil, err
	}
	return msg, nil
}

// EnqueueBatch enqueues up to 200 messages in a single request.
func (s *Service) EnqueueBatch(ctx context.Context, req BatchSendRequest, userID *uuid.UUID) ([]*Message, error) {
	var msgs []*Message
	for _, item := range req.Messages {
		msg, err := s.repo.CreateMessage(ctx, item.Recipient, item.Body, userID)
		if err != nil {
			return msgs, fmt.Errorf("create message for %s: %w", item.Recipient, err)
		}
		if err := s.enqueueTask(ctx, msg, userID); err != nil {
			_ = s.repo.MarkFailed(ctx, msg.ID, "enqueue failed: "+err.Error())
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func (s *Service) ListMessages(ctx context.Context, limit, offset int, status string) ([]*Message, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.ListMessages(ctx, limit, offset, status)
}

// ---- internal ----

func (s *Service) enqueueTask(ctx context.Context, msg *Message, userID *uuid.UUID) error {
	var createdByStr *string
	if userID != nil {
		str := userID.String()
		createdByStr = &str
	}
	payload := SendPayload{
		MessageID: msg.ID.String(),
		Recipient: msg.Recipient,
		Body:      msg.Body,
		CreatedBy: createdByStr,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TaskSend, data,
		asynq.Queue(QueueName),
		asynq.MaxRetry(maxRetry),
		asynq.TaskID(msg.ID.String()),
	)
	_, err = s.queue.EnqueueContext(ctx, task)
	return err
}
