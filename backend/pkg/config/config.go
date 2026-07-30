package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Google   GoogleConfig
	Email    EmailConfig
	AI       AIConfig
	Rate     RateConfig
}

type AppConfig struct {
	Env         string
	Port        string
	URL         string
	FrontendURL string
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type EmailConfig struct {
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
	SendGridKey string
	Provider    string // "smtp" or "sendgrid"
}

type AIConfig struct {
	OpenRouterKey   string
	Model           string
	SystemPrompt    string
	ConversationTTL time.Duration
}

type RateConfig struct {
	AuthPerMin   int
	APIPerMin    int
	APIKeyPerMin int
}

func Load() (*Config, error) {
	// Load .env file if present (ignored in production)
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Env:         getEnv("APP_ENV", "development"),
			Port:        getEnv("APP_PORT", "8080"),
			URL:         getEnv("APP_URL", "http://localhost:8080"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),
		},
		Database: DatabaseConfig{
			URL:             requireEnv("DATABASE_URL"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			URL: requireEnv("REDIS_URL"),
		},
		JWT: JWTConfig{
			AccessSecret:  requireEnv("JWT_ACCESS_SECRET"),
			RefreshSecret: requireEnv("JWT_REFRESH_SECRET"),
			AccessTTL:     getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Google: GoogleConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),
		},
		Email: EmailConfig{
			SMTPHost:    getEnv("SMTP_HOST", ""),
			SMTPPort:    getEnvInt("SMTP_PORT", 587),
			SMTPUser:    getEnv("SMTP_USER", ""),
			SMTPPass:    getEnv("SMTP_PASS", ""),
			SMTPFrom:    getEnv("SMTP_FROM", "noreply@example.com"),
			SendGridKey: getEnv("SENDGRID_API_KEY", ""),
			Provider:    getEnv("EMAIL_PROVIDER", "smtp"),
		},
		AI: AIConfig{
			OpenRouterKey: getEnv("OPENROUTER_API_KEY", ""),
			Model:         getEnv("OPENROUTER_MODEL", "openrouter/free"),
			SystemPrompt: getEnv("AI_SYSTEM_PROMPT", `You are an AI assistant embedded in StarterKit — a full-stack web application built with Go (Chi router) and SvelteKit.

This application is developed and owned by M. Iqbal Effendi (iqbaleff214@gmail.com).

StarterKit provides:
- Authentication: email/password login, Google OAuth, JWT-based sessions (access + refresh tokens), two-factor authentication (TOTP)
- Role-based access control (RBAC): roles are "superadmin", "admin", and "user" with granular permissions
- Real-time notifications delivered via WebSocket, with a persistent notification history
- A public API with scoped API keys (sk_* prefix) for external integrations
- An admin panel for managing users, roles, and permissions
- Transactional email (SMTP or SendGrid) for verification and password reset flows

You are talking to an authenticated user of this application. You have access to tools that let you look up the current user's profile, list their notifications, and (for admins) search users. Use these tools when relevant to answer the user's question. Be concise and helpful.

Only answer questions related to this application and its features (auth, profile, notifications, RBAC, API keys, admin functions, etc.). If the user asks something unrelated to StarterKit or its functionality, politely decline and redirect them to ask about the app instead.`),
			ConversationTTL: getEnvDuration("AI_CONVERSATION_TTL", 30*24*time.Hour),
		},
		Rate: RateConfig{
			AuthPerMin:   getEnvInt("RATE_LIMIT_AUTH", 5),
			APIPerMin:    getEnvInt("RATE_LIMIT_API", 100),
			APIKeyPerMin: getEnvInt("RATE_LIMIT_API_KEY", 60),
		},
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
