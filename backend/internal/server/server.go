package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/internal/auth"
	"github.com/404nfid/go-svelte-starter-kit/internal/email"
	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/internal/notification"
	"github.com/404nfid/go-svelte-starter-kit/internal/rbac"
	"github.com/404nfid/go-svelte-starter-kit/internal/user"
	"github.com/404nfid/go-svelte-starter-kit/internal/ws"
	"github.com/404nfid/go-svelte-starter-kit/pkg/config"
	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	rdb "github.com/404nfid/go-svelte-starter-kit/pkg/redis"
	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	cfg    *config.Config
	db     *database.DB
	redis  *rdb.Client
	logger *slog.Logger
	http   *http.Server
}

func New(cfg *config.Config, db *database.DB, redis *rdb.Client, logger *slog.Logger) *Server {
	s := &Server{cfg: cfg, db: db, redis: redis, logger: logger}
	s.http = &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      s.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	s.logger.Info("server starting", "addr", s.http.Addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) routes() http.Handler {
	// Ensure uploads directory exists
	_ = os.MkdirAll("uploads/avatars", 0755)

	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.Logger(s.logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.cfg.App.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	v := validator.New()
	tokenManager := token.NewManager(
		s.cfg.JWT.AccessSecret,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.AccessTTL,
		s.cfg.JWT.RefreshTTL,
	)

	// ---- Email ----
	emailRepo := email.NewRepository(s.db)
	emailQueue := email.NewQueue(s.cfg.Redis.URL, emailRepo)

	// ---- Auth ----
	authRepo := auth.NewRepository(s.db)
	authSvc := auth.NewService(authRepo, tokenManager, emailQueue, s.cfg.JWT.AccessTTL, s.cfg.JWT.RefreshTTL)
	googleProvider := auth.NewGoogleProvider(s.cfg.Google)
	authHandler := auth.NewHandler(
		authSvc, googleProvider, s.redis.Client, s.cfg.App.FrontendURL,
		v, s.cfg.IsProduction(), s.cfg.JWT.RefreshTTL,
	)

	// ---- User ----
	userRepo := user.NewRepository(s.db)
	userSvc := user.NewService(userRepo, s.cfg.App.URL)
	userHandler := user.NewHandler(userSvc, v)

	// ---- WebSocket Hub ----
	hub := ws.NewHub()
	go hub.Run()

	// ---- Notifications ----
	notifRepo := notification.NewRepository(s.db)
	notifSvc := notification.NewService(notifRepo, hub)
	notifHandler := notification.NewHandler(notifSvc, v)

	// ---- RBAC ----
	rbacRepo := rbac.NewRepository(s.db)
	rbacSvc := rbac.NewService(rbacRepo, s.redis.Client)
	rbacHandler := rbac.NewHandler(rbacSvc, emailRepo, v)

	// ---- Static file serving (avatars) ----
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	r.Route("/api", func(r chi.Router) {
		// Health checks
		r.Get("/health", s.handleHealth)
		r.Get("/ready", s.handleReady)

		// WebSocket — token auth via query param (browsers can't set headers on WS upgrade)
		r.Get("/ws", hub.ServeWS(tokenManager))

		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			// Credential endpoints — strict rate limit (brute-force protection)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RateLimit(s.cfg.Rate.AuthPerMin, time.Minute))
				r.Post("/register", authHandler.Register)
				r.Post("/login", authHandler.Login)
			})
			// Session management — higher rate limit (called automatically by the frontend)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RateLimit(s.cfg.Rate.APIPerMin, time.Minute))
				r.Post("/refresh", authHandler.Refresh)
				r.Post("/logout", authHandler.Logout)
			})
			// Google OAuth (no rate limit — redirects, not JSON endpoints)
			r.Get("/google", authHandler.GoogleRedirect)
			r.Get("/google/callback", authHandler.GoogleCallback)
			r.Post("/google/exchange", authHandler.ExchangeOAuthCode)
			// 2FA verify (no auth middleware — uses pre_auth_token)
			r.Post("/2fa/verify", authHandler.VerifyTwoFA)
			// Password reset & email verification (public, rate-limited)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RateLimit(s.cfg.Rate.AuthPerMin, time.Minute))
				r.Post("/forgot-password", authHandler.ForgotPassword)
				r.Post("/reset-password", authHandler.ResetPassword)
				r.Post("/verify-email", authHandler.VerifyEmail)
			})
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(tokenManager))
			r.Use(middleware.RateLimit(s.cfg.Rate.APIPerMin, time.Minute))

			// 2FA management
			r.Post("/auth/2fa/setup", authHandler.SetupTwoFA)
			r.Post("/auth/2fa/confirm", authHandler.ConfirmTwoFA)
			r.Delete("/auth/2fa", authHandler.DisableTwoFA)
			// Email verification resend
			r.Post("/auth/resend-verification", authHandler.ResendVerification)

			// Notifications
			r.Get("/notifications", notifHandler.List)
			r.Get("/notifications/unread-count", notifHandler.UnreadCount)
			r.Patch("/notifications/{id}/read", notifHandler.MarkRead)
			r.Patch("/notifications/read-all", notifHandler.MarkAllRead)
			r.Post("/notifications/test", notifHandler.Test)

			// Admin panel — role-gated (admin or superadmin)
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin", "superadmin"))
				// User management
				r.Get("/users", rbacHandler.ListUsers)
				r.Patch("/users/{userId}/roles/{roleId}", rbacHandler.AssignRole)
				r.Delete("/users/{userId}/roles/{roleId}", rbacHandler.RevokeRole)
				r.Delete("/users/{userId}", rbacHandler.DeleteUser)
				// Role management
				r.Get("/roles", rbacHandler.ListRoles)
				r.Post("/roles", rbacHandler.CreateRole)
				r.Get("/roles/{id}", rbacHandler.GetRole)
				r.Put("/roles/{id}", rbacHandler.UpdateRole)
				r.Delete("/roles/{id}", rbacHandler.DeleteRole)
				r.Put("/roles/{id}/permissions", rbacHandler.SetPermissions)
				// Permissions
				r.Get("/permissions", rbacHandler.ListPermissions)
				// Email logs
				r.Get("/emails", rbacHandler.ListEmailLogs)
			})

			// Profile & sessions
			r.Get("/me", userHandler.GetProfile)
			r.Patch("/me", userHandler.UpdateProfile)
			r.Post("/me/avatar", userHandler.UploadAvatar)
			r.Post("/me/change-password", userHandler.ChangePassword)
			r.Delete("/me", userHandler.DeleteAccount)
			r.Get("/me/sessions", userHandler.ListSessions)
			r.Delete("/me/sessions/{id}", userHandler.RevokeSession)
			r.Delete("/me/sessions", userHandler.RevokeAllOtherSessions)
		})
	})

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok"}`)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dbOK := s.db.Ping(ctx) == nil
	redisOK := s.redis.Ping(ctx).Err() == nil

	status := "ok"
	code := http.StatusOK
	if !dbOK || !redisOK {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"status":%q,"db":%t,"redis":%t}`, status, dbOK, redisOK)
}
