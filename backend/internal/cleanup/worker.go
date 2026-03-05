// Package cleanup provides background task handlers for periodic database maintenance.
package cleanup

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/internal/ai"
	"github.com/404nfid/go-svelte-starter-kit/internal/auth"
	"github.com/hibiken/asynq"
)

// Task type names registered with the asynq scheduler and server.
const (
	TaskSessions           = "cleanup:sessions"
	TaskPasswordResets     = "cleanup:password_resets"
	TaskEmailVerifications = "cleanup:email_verifications"
	TaskAIConversations    = "cleanup:ai_conversations"

	// Queue is the asynq queue name for all cleanup tasks.
	Queue = "cleanup"
)

// Worker handles periodic maintenance tasks.
type Worker struct {
	authRepo *auth.Repository
	aiRepo   *ai.Repository
	aiTTL    time.Duration
}

// New creates a Worker. aiTTL is the maximum age of AI conversations before purging.
func New(authRepo *auth.Repository, aiRepo *ai.Repository, aiTTL time.Duration) *Worker {
	return &Worker{authRepo: authRepo, aiRepo: aiRepo, aiTTL: aiTTL}
}

// Register wires all cleanup task handlers into the provided ServeMux.
func (w *Worker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskSessions, w.handleSessions)
	mux.HandleFunc(TaskPasswordResets, w.handlePasswordResets)
	mux.HandleFunc(TaskEmailVerifications, w.handleEmailVerifications)
	mux.HandleFunc(TaskAIConversations, w.handleAIConversations)
}

func (w *Worker) handleSessions(ctx context.Context, _ *asynq.Task) error {
	n, err := w.authRepo.DeleteExpiredSessions(ctx)
	if err != nil {
		return fmt.Errorf("cleanup sessions: %w", err)
	}
	slog.Info("cleanup: deleted expired sessions", "count", n)
	return nil
}

func (w *Worker) handlePasswordResets(ctx context.Context, _ *asynq.Task) error {
	n, err := w.authRepo.DeleteExpiredPasswordResets(ctx)
	if err != nil {
		return fmt.Errorf("cleanup password_resets: %w", err)
	}
	slog.Info("cleanup: deleted expired password resets", "count", n)
	return nil
}

func (w *Worker) handleEmailVerifications(ctx context.Context, _ *asynq.Task) error {
	n, err := w.authRepo.DeleteExpiredEmailVerifications(ctx)
	if err != nil {
		return fmt.Errorf("cleanup email_verifications: %w", err)
	}
	slog.Info("cleanup: deleted expired email verifications", "count", n)
	return nil
}

func (w *Worker) handleAIConversations(ctx context.Context, _ *asynq.Task) error {
	if err := w.aiRepo.PurgeOldConversations(ctx, w.aiTTL); err != nil {
		return fmt.Errorf("cleanup ai_conversations: %w", err)
	}
	slog.Info("cleanup: purged old AI conversations")
	return nil
}
