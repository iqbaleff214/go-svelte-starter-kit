package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	queueName    = "email"
	maxRetry     = 3
	uniqueTTLSec = 300 // 5 min dedup window
)

// Queue is the client-side API for enqueuing email jobs.
type Queue struct {
	client *asynq.Client
	repo   *Repository
}

// NewQueue creates an asynq client connected to Redis.
func NewQueue(redisURL string, repo *Repository) *Queue {
	opt, _ := asynq.ParseRedisURI(redisURL)
	return &Queue{
		client: asynq.NewClient(opt),
		repo:   repo,
	}
}

// Close closes the underlying asynq client.
func (q *Queue) Close() { _ = q.client.Close() }

func (q *Queue) enqueue(ctx context.Context, taskType string, payload any, logID uuid.UUID) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	task := asynq.NewTask(taskType, data,
		asynq.MaxRetry(maxRetry),
		asynq.Queue(queueName),
		asynq.TaskID(logID.String()),
	)
	if _, err := q.client.EnqueueContext(ctx, task); err != nil {
		return fmt.Errorf("enqueue %s: %w", taskType, err)
	}
	return nil
}

func (q *Queue) EnqueueWelcome(ctx context.Context, userID uuid.UUID, email, displayName string) error {
	logID, err := q.repo.CreateLog(ctx, &userID, TypeEmailWelcome, email)
	if err != nil {
		return err
	}
	return q.enqueue(ctx, TypeEmailWelcome, WelcomePayload{
		LogID: logID.String(), UserID: userID.String(), Email: email, DisplayName: displayName,
	}, logID)
}

func (q *Queue) EnqueueVerification(ctx context.Context, userID uuid.UUID, email, displayName, token string) error {
	logID, err := q.repo.CreateLog(ctx, &userID, TypeEmailVerification, email)
	if err != nil {
		return err
	}
	return q.enqueue(ctx, TypeEmailVerification, VerifyEmailPayload{
		LogID: logID.String(), UserID: userID.String(), Email: email, DisplayName: displayName, Token: token,
	}, logID)
}

func (q *Queue) EnqueuePasswordReset(ctx context.Context, userID uuid.UUID, email, displayName, token string) error {
	logID, err := q.repo.CreateLog(ctx, &userID, TypeEmailPasswordReset, email)
	if err != nil {
		return err
	}
	return q.enqueue(ctx, TypeEmailPasswordReset, PasswordResetPayload{
		LogID: logID.String(), UserID: userID.String(), Email: email, DisplayName: displayName, Token: token,
	}, logID)
}

func (q *Queue) EnqueueTwoFABackupCodes(ctx context.Context, userID uuid.UUID, email, displayName string, codes []string) error {
	logID, err := q.repo.CreateLog(ctx, &userID, TypeEmail2FABackupCodes, email)
	if err != nil {
		return err
	}
	return q.enqueue(ctx, TypeEmail2FABackupCodes, TwoFABackupPayload{
		LogID: logID.String(), UserID: userID.String(), Email: email, DisplayName: displayName, Codes: codes,
	}, logID)
}

func (q *Queue) EnqueueSecurityAlert(ctx context.Context, userID uuid.UUID, email, displayName, ip, device string) error {
	logID, err := q.repo.CreateLog(ctx, &userID, TypeEmailSecurityAlert, email)
	if err != nil {
		return err
	}
	return q.enqueue(ctx, TypeEmailSecurityAlert, SecurityAlertPayload{
		LogID: logID.String(), UserID: userID.String(), Email: email, DisplayName: displayName, IP: ip, Device: device,
	}, logID)
}

func (q *Queue) EnqueueAccountDeletion(ctx context.Context, userID uuid.UUID, email, displayName string) error {
	logID, err := q.repo.CreateLog(ctx, &userID, TypeEmailAccountDeletion, email)
	if err != nil {
		return err
	}
	return q.enqueue(ctx, TypeEmailAccountDeletion, AccountDeletionPayload{
		LogID: logID.String(), UserID: userID.String(), Email: email, DisplayName: displayName,
	}, logID)
}
