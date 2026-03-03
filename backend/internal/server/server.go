package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/internal/auth"
	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/config"
	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	rdb "github.com/404nfid/go-svelte-starter-kit/pkg/redis"
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

	// ---- Auth routes ----
	authRepo := auth.NewRepository(s.db)
	authSvc := auth.NewService(authRepo, tokenManager, s.cfg.JWT.AccessTTL, s.cfg.JWT.RefreshTTL)
	authHandler := auth.NewHandler(authSvc, v, s.cfg.IsProduction(), s.cfg.JWT.RefreshTTL)

	r.Route("/api", func(r chi.Router) {
		// Health checks
		r.Get("/health", s.handleHealth)
		r.Get("/ready", s.handleReady)

		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			// Strict limit on credential endpoints (brute-force protection)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RateLimit(s.cfg.Rate.AuthPerMin, time.Minute))
				r.Post("/register", authHandler.Register)
				r.Post("/login", authHandler.Login)
			})
			// Higher limit on session management (called automatically by the frontend)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RateLimit(s.cfg.Rate.APIPerMin, time.Minute))
				r.Post("/refresh", authHandler.Refresh)
				r.Post("/logout", authHandler.Logout)
			})
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(tokenManager))
			r.Use(middleware.RateLimit(s.cfg.Rate.APIPerMin, time.Minute))

			// Placeholder — domains added in later phases
			r.Get("/me", s.handleMe)
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

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"data":{"id":%q,"email":%q,"roles":%v}}`,
		claims.UserID, claims.Email, claims.Roles)
}
