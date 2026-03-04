package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	"github.com/google/uuid"
)

// APIKeyValidator is implemented by apikey.Service.
type APIKeyValidator interface {
	ValidateKey(ctx context.Context, rawKey string) (APIKeyValidated, error)
	CheckRateLimit(ctx context.Context, keyID uuid.UUID) error
	LogUsage(keyID uuid.UUID, method, path, ip string, statusCode int)
}

// APIKeyValidated is the subset of apikey.ValidatedKey that the middleware needs.
type APIKeyValidated struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Email  string
	Roles  []string
	Scopes []string
}

// APIKeyClaims holds per-request API key metadata stored in context.
type APIKeyClaims struct {
	KeyID  uuid.UUID
	UserID uuid.UUID
	Scopes []string
}

const apiKeyClaimsKey contextKey = "api_key_claims"

// statusRecorder wraps http.ResponseWriter to capture the written status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// AuthenticateAPIKey validates "Authorization: Bearer sk_..." for /api/v1/* routes.
// On success it populates both ClaimsKey (so existing handlers work unchanged) and
// apiKeyClaimsKey (for RequireScope checks).
func AuthenticateAPIKey(svc APIKeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w, "missing_token", "Authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				respondUnauthorized(w, "invalid_token", "Authorization header format must be: Bearer <token>")
				return
			}

			vk, err := svc.ValidateKey(r.Context(), parts[1])
			if err != nil {
				respondUnauthorized(w, "invalid_api_key", "API key is invalid or revoked")
				return
			}

			if err := svc.CheckRateLimit(r.Context(), vk.ID); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"Rate limit exceeded"}}`))
				return
			}

			// Populate standard claims so existing handlers work unchanged.
			claims := &token.Claims{
				UserID: vk.UserID.String(),
				Email:  vk.Email,
				Roles:  vk.Roles,
				Type:   token.AccessToken,
			}
			ctx := context.WithValue(r.Context(), ClaimsKey, claims)

			akClaims := &APIKeyClaims{KeyID: vk.ID, UserID: vk.UserID, Scopes: vk.Scopes}
			ctx = context.WithValue(ctx, apiKeyClaimsKey, akClaims)

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			defer func() {
				ip := r.RemoteAddr
				if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
					ip = strings.SplitN(fwd, ",", 2)[0]
				}
				svc.LogUsage(vk.ID, r.Method, r.URL.Path, strings.TrimSpace(ip), rec.status)
			}()

			next.ServeHTTP(rec, r.WithContext(ctx))
		})
	}
}

// RequireScope checks that the API key has the given scope.
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			akc := GetAPIKeyClaims(r)
			if akc == nil {
				respondForbidden(w)
				return
			}
			for _, s := range akc.Scopes {
				if s == scope {
					next.ServeHTTP(w, r)
					return
				}
			}
			respondForbidden(w)
		})
	}
}

// GetAPIKeyClaims retrieves API key specific claims from the request context.
func GetAPIKeyClaims(r *http.Request) *APIKeyClaims {
	v, _ := r.Context().Value(apiKeyClaimsKey).(*APIKeyClaims)
	return v
}
