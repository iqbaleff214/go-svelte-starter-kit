package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Worker processes queued email tasks using asynq.
type Worker struct {
	server *asynq.Server
	engine *Engine
	sender Sender
	repo   *Repository
	appURL string
}

const appName = "StarterKit"

// NewWorker creates an asynq server worker connected to Redis.
func NewWorker(redisURL string, engine *Engine, sender Sender, repo *Repository, appURL string) *Worker {
	opt, _ := asynq.ParseRedisURI(redisURL)
	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: 5,
		Queues:      map[string]int{queueName: 10},
	})
	return &Worker{server: srv, engine: engine, sender: sender, repo: repo, appURL: appURL}
}

// Start registers all task handlers and blocks until the server shuts down.
func (w *Worker) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeEmailWelcome, w.handleWelcome)
	mux.HandleFunc(TypeEmailVerification, w.handleVerification)
	mux.HandleFunc(TypeEmailPasswordReset, w.handlePasswordReset)
	mux.HandleFunc(TypeEmail2FABackupCodes, w.handleTwoFABackupCodes)
	mux.HandleFunc(TypeEmailSecurityAlert, w.handleSecurityAlert)
	mux.HandleFunc(TypeEmailAccountDeletion, w.handleAccountDeletion)
	return w.server.Run(mux)
}

// Stop gracefully shuts down the worker.
func (w *Worker) Stop() { w.server.Shutdown() }

// ---- task handlers ----

func (w *Worker) handleWelcome(ctx context.Context, t *asynq.Task) error {
	var p WelcomePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal welcome payload: %w", err)
	}
	logID, err := uuid.Parse(p.LogID)
	if err != nil {
		return fmt.Errorf("parse log id: %w", err)
	}

	html, text, err := w.engine.Render("welcome.html", WelcomeData{
		DisplayName: p.DisplayName,
		AppName:     appName,
		AppURL:      w.appURL,
	})
	if err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	msg := Message{To: p.Email, Subject: "Welcome to " + appName, HTML: html, Text: text}
	if err := w.sender.Send(ctx, msg); err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	return w.succeed(ctx, logID)
}

func (w *Worker) handleVerification(ctx context.Context, t *asynq.Task) error {
	var p VerifyEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal verification payload: %w", err)
	}
	logID, err := uuid.Parse(p.LogID)
	if err != nil {
		return fmt.Errorf("parse log id: %w", err)
	}

	html, text, err := w.engine.Render("verify_email.html", VerifyEmailData{
		DisplayName: p.DisplayName,
		VerifyURL:   w.appURL + "/auth/verify-email?token=" + p.Token,
		AppName:     appName,
		ExpiresIn:   "1 hour",
	})
	if err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	msg := Message{To: p.Email, Subject: "Verify your email address", HTML: html, Text: text}
	if err := w.sender.Send(ctx, msg); err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	return w.succeed(ctx, logID)
}

func (w *Worker) handlePasswordReset(ctx context.Context, t *asynq.Task) error {
	var p PasswordResetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal password reset payload: %w", err)
	}
	logID, err := uuid.Parse(p.LogID)
	if err != nil {
		return fmt.Errorf("parse log id: %w", err)
	}

	html, text, err := w.engine.Render("password_reset.html", PasswordResetData{
		DisplayName: p.DisplayName,
		ResetURL:    w.appURL + "/reset-password?token=" + p.Token,
		AppName:     appName,
		ExpiresIn:   "1 hour",
	})
	if err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	msg := Message{To: p.Email, Subject: "Reset your password", HTML: html, Text: text}
	if err := w.sender.Send(ctx, msg); err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	return w.succeed(ctx, logID)
}

func (w *Worker) handleTwoFABackupCodes(ctx context.Context, t *asynq.Task) error {
	var p TwoFABackupPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal 2fa backup payload: %w", err)
	}
	logID, err := uuid.Parse(p.LogID)
	if err != nil {
		return fmt.Errorf("parse log id: %w", err)
	}

	html, text, err := w.engine.Render("two_fa_backup_codes.html", TwoFABackupData{
		DisplayName: p.DisplayName,
		AppName:     appName,
		Codes:       p.Codes,
	})
	if err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	msg := Message{To: p.Email, Subject: "Your 2FA backup codes", HTML: html, Text: text}
	if err := w.sender.Send(ctx, msg); err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	return w.succeed(ctx, logID)
}

func (w *Worker) handleSecurityAlert(ctx context.Context, t *asynq.Task) error {
	var p SecurityAlertPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal security alert payload: %w", err)
	}
	logID, err := uuid.Parse(p.LogID)
	if err != nil {
		return fmt.Errorf("parse log id: %w", err)
	}

	html, text, err := w.engine.Render("security_alert.html", SecurityAlertData{
		DisplayName: p.DisplayName,
		AppName:     appName,
		IP:          p.IP,
		Device:      p.Device,
		Time:        time.Now().UTC().Format("Jan 2, 2006 15:04 UTC"),
	})
	if err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	msg := Message{To: p.Email, Subject: "New login detected on your account", HTML: html, Text: text}
	if err := w.sender.Send(ctx, msg); err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	return w.succeed(ctx, logID)
}

func (w *Worker) handleAccountDeletion(ctx context.Context, t *asynq.Task) error {
	var p AccountDeletionPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal account deletion payload: %w", err)
	}
	logID, err := uuid.Parse(p.LogID)
	if err != nil {
		return fmt.Errorf("parse log id: %w", err)
	}

	html, text, err := w.engine.Render("account_deletion.html", AccountDeletionData{
		DisplayName: p.DisplayName,
		AppName:     appName,
	})
	if err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	msg := Message{To: p.Email, Subject: "Your account has been deleted", HTML: html, Text: text}
	if err := w.sender.Send(ctx, msg); err != nil {
		return w.fail(ctx, logID, t.ResultWriter(), err)
	}
	return w.succeed(ctx, logID)
}

// ---- helpers ----

func (w *Worker) succeed(ctx context.Context, logID uuid.UUID) error {
	now := time.Now()
	if err := w.repo.UpdateLog(ctx, logID, "sent", "", &now, 1); err != nil {
		slog.Error("update email log", "log_id", logID, "err", err)
	}
	return nil
}

func (w *Worker) fail(ctx context.Context, logID uuid.UUID, _ *asynq.ResultWriter, cause error) error {
	if err := w.repo.UpdateLog(ctx, logID, "failed", cause.Error(), nil, 1); err != nil {
		slog.Error("update email log", "log_id", logID, "err", err)
	}
	return cause
}
