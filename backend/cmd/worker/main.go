package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/internal/ai"
	"github.com/404nfid/go-svelte-starter-kit/internal/auth"
	"github.com/404nfid/go-svelte-starter-kit/internal/cleanup"
	"github.com/404nfid/go-svelte-starter-kit/internal/email"
	"github.com/404nfid/go-svelte-starter-kit/internal/whatsapp"
	"github.com/404nfid/go-svelte-starter-kit/pkg/config"
	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/404nfid/go-svelte-starter-kit/pkg/logger"
	rdb "github.com/404nfid/go-svelte-starter-kit/pkg/redis"
	"github.com/hibiken/asynq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.App.Env)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, database.Config{
		URL:             cfg.Database.URL,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		log.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redis, err := rdb.Connect(ctx, cfg.Redis.URL)
	if err != nil {
		log.Error("connect to redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	// ---- Email worker (asynq server for "email" queue) ----
	emailRepo := email.NewRepository(db)
	emailEngine, err := email.NewEngine("templates/email")
	if err != nil {
		log.Error("load email templates", "error", err)
		os.Exit(1)
	}
	emailSender := email.NewSender(cfg.Email)
	emailWorker := email.NewWorker(cfg.Redis.URL, emailEngine, emailSender, emailRepo, cfg.App.URL)

	// ---- WhatsApp worker ----
	waRepo := whatsapp.NewRepository(db)
	waManager, err := whatsapp.NewManager(cfg.Database.URL, waRepo, log)
	if err != nil {
		log.Error("whatsapp manager init failed", "error", err)
		os.Exit(1)
	}
	waManager.RestoreConnected(context.Background())
	waWorker := whatsapp.NewWorker(cfg.Redis.URL, waManager, waRepo, log)

	// ---- Cleanup worker (asynq server for "cleanup" queue) ----
	authRepo := auth.NewRepository(db)
	aiRepo := ai.NewRepository(db)
	cleanupWorker := cleanup.New(authRepo, aiRepo, cfg.AI.ConversationTTL)

	redisOpt, _ := asynq.ParseRedisURI(cfg.Redis.URL)
	cleanupSrv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 2,
		Queues:      map[string]int{cleanup.Queue: 5},
	})
	cleanupMux := asynq.NewServeMux()
	cleanupWorker.Register(cleanupMux)

	// ---- Scheduler (enqueues cleanup tasks on a cron schedule) ----
	scheduler := asynq.NewScheduler(redisOpt, nil)

	// Every 6 hours: purge expired auth tokens and sessions
	for _, task := range []string{
		cleanup.TaskSessions,
		cleanup.TaskPasswordResets,
		cleanup.TaskEmailVerifications,
	} {
		if _, err := scheduler.Register(
			"@every 6h",
			asynq.NewTask(task, nil, asynq.Queue(cleanup.Queue), asynq.MaxRetry(2)),
		); err != nil {
			log.Error("register cleanup task", "task", task, "error", err)
			os.Exit(1)
		}
	}

	// Every 24 hours: purge old AI conversations
	if _, err := scheduler.Register(
		"@every 24h",
		asynq.NewTask(cleanup.TaskAIConversations, nil, asynq.Queue(cleanup.Queue), asynq.MaxRetry(2)),
	); err != nil {
		log.Error("register cleanup task", "task", cleanup.TaskAIConversations, "error", err)
		os.Exit(1)
	}

	// Daily at midnight: reset WhatsApp sent_today counters
	if _, err := scheduler.Register(
		"0 0 * * *",
		asynq.NewTask("whatsapp:reset_daily", nil, asynq.Queue(whatsapp.QueueName), asynq.MaxRetry(2)),
	); err != nil {
		log.Error("register whatsapp reset task", "error", err)
		os.Exit(1)
	}

	if err := scheduler.Start(); err != nil {
		log.Error("start cleanup scheduler", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := cleanupSrv.Run(cleanupMux); err != nil {
			log.Error("cleanup worker error", "error", err)
		}
	}()

	log.Info("worker started", "env", cfg.App.Env)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Periodically sync the WA pool: connect newly paired sessions, drop removed ones.
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				waManager.SyncPool(context.Background())
			case <-quit:
				return
			}
		}
	}()

	go func() {
		<-quit
		log.Info("shutting down worker...")
		scheduler.Shutdown()
		cleanupSrv.Shutdown()
		waWorker.Stop()
		waManager.Shutdown()
		emailWorker.Stop()
	}()

	// Start WhatsApp worker in background
	go func() {
		if err := waWorker.Start(); err != nil {
			log.Error("whatsapp worker error", "error", err)
		}
	}()

	// emailWorker.Start() blocks until Stop() is called
	if err := emailWorker.Start(); err != nil {
		log.Error("email worker error", "error", err)
		os.Exit(1)
	}

	log.Info("worker stopped")
}
