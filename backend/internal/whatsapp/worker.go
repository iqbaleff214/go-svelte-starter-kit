package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
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
	mux.HandleFunc("whatsapp:reset_daily", w.handleResetDaily)
	return w.server.Run(mux)
}

func (w *Worker) Stop() { w.server.Shutdown() }

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

	sessionID, err := w.manager.Send(ctx, p.Recipient, p.Body)
	if err != nil {
		_ = w.repo.MarkFailed(ctx, msgID, err.Error())
		w.log.Warn("wa: send failed", "message_id", p.MessageID, "recipient", p.Recipient, "error", err)
		return err // asynq will retry
	}

	if err := w.repo.MarkSent(ctx, msgID, sessionID); err != nil {
		w.log.Error("wa: mark sent failed", "message_id", p.MessageID, "error", err)
	}

	w.log.Info("wa: message sent", "message_id", p.MessageID, "recipient", p.Recipient, "session", sessionID)
	return nil
}
