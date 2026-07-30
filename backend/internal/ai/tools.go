package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/404nfid/go-svelte-starter-kit/internal/notification"
	"github.com/404nfid/go-svelte-starter-kit/internal/rbac"
	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	"github.com/google/uuid"
)

type ToolInput map[string]any

type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type Tool struct {
	Def     ToolDef
	Execute func(ctx context.Context, claims *token.Claims, input ToolInput) (string, error)
}

func buildTools(notifRepo *notification.Repository, rbacRepo *rbac.Repository) []Tool {
	return []Tool{
		{
			Def: ToolDef{
				Name:        "get_current_user",
				Description: "Get the currently authenticated user's profile, including user ID, email, and roles.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
			Execute: func(ctx context.Context, claims *token.Claims, input ToolInput) (string, error) {
				result := map[string]any{
					"user_id": claims.UserID,
					"email":   claims.Email,
					"roles":   claims.Roles,
				}
				b, err := json.Marshal(result)
				return string(b), err
			},
		},
		{
			Def: ToolDef{
				Name:        "list_notifications",
				Description: "List recent notifications for the current user.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"limit": map[string]any{
							"type":        "integer",
							"description": "Maximum number of notifications to return (default 10, max 50)",
						},
					},
				},
			},
			Execute: func(ctx context.Context, claims *token.Claims, input ToolInput) (string, error) {
				limit := 10
				if v, ok := input["limit"]; ok {
					switch n := v.(type) {
					case float64:
						limit = int(n)
					case int:
						limit = n
					}
				}
				if limit > 50 {
					limit = 50
				}
				if limit < 1 {
					limit = 1
				}
				userID, err := uuid.Parse(claims.UserID)
				if err != nil {
					return "invalid user ID", nil
				}
				notifs, _, err := notifRepo.List(ctx, userID, limit, 0)
				if err != nil {
					return fmt.Sprintf("error: %s", err.Error()), nil
				}
				b, err := json.Marshal(notifs)
				return string(b), err
			},
		},
		{
			Def: ToolDef{
				Name:        "search_users",
				Description: "Search for users by email or display name. Only available to admins and superadmins.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Search query to filter users by email or display name",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "Maximum number of users to return (default 10, max 50)",
						},
					},
					"required": []string{"query"},
				},
			},
			Execute: func(ctx context.Context, claims *token.Claims, input ToolInput) (string, error) {
				if !slices.Contains(claims.Roles, "admin") && !slices.Contains(claims.Roles, "superadmin") {
					return `"Permission denied: only admins can search users"`, nil
				}
				query, _ := input["query"].(string)
				limit := 10
				if v, ok := input["limit"]; ok {
					if n, ok := v.(float64); ok {
						limit = int(n)
					}
				}
				if limit > 50 {
					limit = 50
				}
				users, _, err := rbacRepo.ListUsers(ctx, 50, 0, "")
				if err != nil {
					return fmt.Sprintf("error: %s", err.Error()), nil
				}
				var filtered []map[string]any
				for _, u := range users {
					if query == "" ||
						containsCI(u.Email, query) ||
						containsCI(u.DisplayName, query) {
						filtered = append(filtered, map[string]any{
							"id":           u.ID,
							"email":        u.Email,
							"display_name": u.DisplayName,
							"roles":        u.Roles,
						})
					}
					if len(filtered) >= limit {
						break
					}
				}
				if filtered == nil {
					filtered = []map[string]any{}
				}
				b, err := json.Marshal(filtered)
				return string(b), err
			},
		},
	}
}

func containsCI(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	sl, subl := len(s), len(substr)
	for i := 0; i <= sl-subl; i++ {
		if equalFoldN(s[i:i+subl], substr) {
			return true
		}
	}
	return false
}

func equalFoldN(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
