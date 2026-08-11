package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.mau.fi/whatsmeow"
)

// Worker processes queued WhatsApp send tasks via asynq.
type Worker struct {
	server  *asynq.Server
	manager *Manager
	repo    *Repository
	log     *slog.Logger
}

func NewWorker(redisURL string, manager *Manager, repo *Repository, log *slog.Logger) *Worker {
	opt, _ := asynq.ParseRedisURI(redisURL)
	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: 3, // keep low to respect WA rate limits
		Queues:      map[string]int{QueueName: 5},
	})
	return &Worker{server: srv, manager: manager, repo: repo, log: log}
}

// Start registers task handlers and blocks until Stop is called.
func (w *Worker) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskSend, w.handleSend)
	mux.HandleFunc(TaskSyncPool, w.handleSyncPool)
	mux.HandleFunc("whatsapp:reset_daily", w.handleResetDaily)
	return w.server.Run(mux)
}

func (w *Worker) Stop() { w.server.Shutdown() }

func (w *Worker) handleSyncPool(ctx context.Context, _ *asynq.Task) error {
	w.manager.SyncPool(ctx)
	return nil
}

func (w *Worker) handleResetDaily(ctx context.Context, _ *asynq.Task) error {
	if err := w.repo.ResetDailyCounts(ctx); err != nil {
		w.log.Error("wa: reset daily counts", "error", err)
		return err
	}
	w.log.Info("wa: daily sent_today counters reset")
	return nil
}

func (w *Worker) handleSend(ctx context.Context, t *asynq.Task) error {
	var p SendPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	msgID, err := uuid.Parse(p.MessageID)
	if err != nil {
		return fmt.Errorf("invalid message_id: %w", err)
	}

	// Retry locally for transient "not connected" errors — the pool may be
	// repopulated within a few seconds once whatsmeow finishes reconnecting.
	var sessionID uuid.UUID
	var sendErr error
	for attempt := range 3 {
		sessionID, sendErr = w.manager.Send(ctx, p.Recipient, p.Body)
		if sendErr == nil {
			break
		}
		// Retry on transient connectivity errors: pool empty or socket dropped.
		isTransient := errors.Is(sendErr, whatsmeow.ErrNotConnected) ||
			strings.Contains(sendErr.Error(), "websocket not connected") ||
			strings.Contains(sendErr.Error(), "no connected whatsapp sessions")
		if !isTransient {
			break
		}
		if attempt < 2 {
			w.log.Warn("wa: socket not ready, retrying", "attempt", attempt+1, "message_id", p.MessageID)
			time.Sleep(3 * time.Second)
		}
	}

	if sendErr != nil {
		_ = w.repo.MarkFailed(ctx, msgID, sendErr.Error())
		w.log.Warn("wa: send failed", "message_id", p.MessageID, "recipient", p.Recipient, "error", sendErr)
		return sendErr // asynq will retry (up to maxRetry)
	}

	if err := w.repo.MarkSent(ctx, msgID, sessionID); err != nil {
		w.log.Error("wa: mark sent failed", "message_id", p.MessageID, "error", err)
	}

	w.log.Info("wa: message sent", "message_id", p.MessageID, "recipient", p.Recipient, "session", sessionID)
	return nil
}
