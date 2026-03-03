package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
)

// PermissionChecker resolves the full set of permissions for a slice of role names.
// Implemented by rbac.Service with Redis caching.
type PermissionChecker interface {
	GetPermissionsForRoles(ctx context.Context, roles []string) ([]string, error)
}

// RequirePermission returns a middleware that checks whether the authenticated user
// has the given permission string (e.g. "users:write"). It resolves permissions via
// the PermissionChecker (typically rbac.Service with Redis caching).
func RequirePermission(checker PermissionChecker, required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				respondUnauthorized(w, "missing_claims", "Unauthorized")
				return
			}

			perms, err := checker.GetPermissionsForRoles(r.Context(), claims.Roles)
			if err != nil || !slices.Contains(perms, required) {
				respondForbidden(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type contextKey string

const ClaimsKey contextKey = "claims"

func Authenticate(tokenManager *token.Manager) func(http.Handler) http.Handler {
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

			claims, err := tokenManager.VerifyAccess(parts[1])
			if err != nil {
				respondUnauthorized(w, "invalid_token", "Token is invalid or expired")
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetClaims(r *http.Request) *token.Claims {
	claims, _ := r.Context().Value(ClaimsKey).(*token.Claims)
	return claims
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				respondUnauthorized(w, "missing_claims", "Unauthorized")
				return
			}

			for _, role := range claims.Roles {
				if _, ok := roleSet[role]; ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			respondForbidden(w)
		})
	}
}

func respondUnauthorized(w http.ResponseWriter, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}

func respondForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": "forbidden", "message": "You do not have permission to access this resource"},
	})
}
