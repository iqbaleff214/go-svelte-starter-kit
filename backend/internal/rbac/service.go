package rbac

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const permissionCacheTTL = 5 * time.Minute

func cacheKey(roleName string) string {
	return "rbac:role:perms:" + roleName
}

type Service struct {
	repo  *Repository
	redis *redis.Client
}

func NewService(repo *Repository, rdb *redis.Client) *Service {
	return &Service{repo: repo, redis: rdb}
}

// GetPermissionsForRoles returns the union of permissions for a set of role names.
// Results are cached per role in Redis for permissionCacheTTL to avoid repeated DB hits.
// Satisfies middleware.PermissionChecker.
func (s *Service) GetPermissionsForRoles(ctx context.Context, roles []string) ([]string, error) {
	if len(roles) == 0 {
		return []string{}, nil
	}

	seen := make(map[string]struct{})
	var result []string

	for _, role := range roles {
		key := cacheKey(role)
		cached, err := s.redis.SMembers(ctx, key).Result()
		if err == nil && len(cached) > 0 {
			for _, p := range cached {
				if _, ok := seen[p]; !ok {
					seen[p] = struct{}{}
					result = append(result, p)
				}
			}
			continue
		}

		// Cache miss — query DB for this single role
		perms, err := s.repo.GetPermissionsForRoles(ctx, []string{role})
		if err != nil {
			return nil, fmt.Errorf("fetch permissions for role %q: %w", role, err)
		}

		if len(perms) > 0 {
			// Populate cache
			args := make([]interface{}, len(perms))
			for i, p := range perms {
				args[i] = p
			}
			if err := s.redis.SAdd(ctx, key, args...).Err(); err != nil {
				slog.Warn("rbac: failed to cache permissions", "role", role, "err", err)
			} else {
				s.redis.Expire(ctx, key, permissionCacheTTL) //nolint:errcheck
			}
		}

		for _, p := range perms {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				result = append(result, p)
			}
		}
	}

	if result == nil {
		result = []string{}
	}
	return result, nil
}

// invalidatePermissionCache removes cached permissions for the given role names.
func (s *Service) invalidatePermissionCache(ctx context.Context, roleNames ...string) {
	for _, name := range roleNames {
		if err := s.redis.Del(ctx, cacheKey(name)).Err(); err != nil {
			slog.Warn("rbac: failed to invalidate permission cache", "role", name, "err", err)
		}
	}
}

// ---- User management ----

func (s *Service) ListUsers(ctx context.Context, page, limit int, roleFilter string) (*AdminUsersResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	users, total, err := s.repo.ListUsers(ctx, limit, offset, roleFilter)
	if err != nil {
		return nil, err
	}
	return &AdminUsersResponse{Users: users, Total: total, Page: page, Limit: limit}, nil
}

func (s *Service) AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	return s.repo.AssignRole(ctx, userID, roleID, assignedBy)
}

func (s *Service) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.repo.RevokeRole(ctx, userID, roleID)
}

func (s *Service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return s.repo.DeleteUser(ctx, userID)
}

// ---- Role management ----

func (s *Service) ListRoles(ctx context.Context) ([]*Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	return s.repo.GetRole(ctx, id)
}

func (s *Service) CreateRole(ctx context.Context, name string, description *string) (*Role, error) {
	return s.repo.CreateRole(ctx, name, description)
}

func (s *Service) UpdateRole(ctx context.Context, id uuid.UUID, name *string, description *string) (*Role, error) {
	return s.repo.UpdateRole(ctx, id, name, description)
}

func (s *Service) DeleteRole(ctx context.Context, id uuid.UUID) error {
	// Fetch the role name first so we can invalidate its cache
	ro, err := s.repo.GetRole(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteRole(ctx, id); err != nil {
		return err
	}
	s.invalidatePermissionCache(ctx, ro.Name)
	return nil
}

func (s *Service) SetRolePermissions(ctx context.Context, roleID uuid.UUID, permIDs []uuid.UUID) error {
	ro, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return err
	}
	if err := s.repo.SetRolePermissions(ctx, roleID, permIDs); err != nil {
		return err
	}
	s.invalidatePermissionCache(ctx, ro.Name)
	return nil
}

// ---- Permission management ----

func (s *Service) ListPermissions(ctx context.Context) ([]*Permission, error) {
	return s.repo.ListPermissions(ctx)
}
