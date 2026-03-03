package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/internal/email"
	"github.com/404nfid/go-svelte-starter-kit/pkg/config"
	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/404nfid/go-svelte-starter-kit/pkg/logger"
	rdb "github.com/404nfid/go-svelte-starter-kit/pkg/redis"
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

	emailRepo := email.NewRepository(db)

	emailEngine, err := email.NewEngine("templates/email")
	if err != nil {
		log.Error("load email templates", "error", err)
		os.Exit(1)
	}

	emailSender := email.NewSender(cfg.Email)

	worker := email.NewWorker(cfg.Redis.URL, emailEngine, emailSender, emailRepo, cfg.App.URL)

	log.Info("worker started", "env", cfg.App.Env)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("shutting down worker...")
		worker.Stop()
	}()

	if err := worker.Start(); err != nil {
		log.Error("worker error", "error", err)
		os.Exit(1)
	}

	log.Info("worker stopped")
}
