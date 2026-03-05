package rbac

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/404nfid/go-svelte-starter-kit/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// ListUsers returns paginated users with their assigned roles, optionally filtered by role name.
func (r *Repository) ListUsers(ctx context.Context, limit, offset int, roleFilter string) ([]*AdminUser, int, error) {
	// Count query
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN roles ro ON ro.id = ur.role_id
		WHERE u.deleted_at IS NULL
		  AND ($1 = '' OR ro.name = $1)`
	if err := r.db.Pool.QueryRow(ctx, countQuery, roleFilter).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// Data query — aggregates roles per user via array_agg
	query := `
		SELECT u.id, u.email, u.display_name, u.avatar_url, u.email_verified_at,
		       u.two_fa_enabled, u.created_at,
		       COALESCE(array_agg(ro.name) FILTER (WHERE ro.name IS NOT NULL), '{}') AS roles
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN roles ro ON ro.id = ur.role_id
		WHERE u.deleted_at IS NULL
		  AND ($1 = '' OR ro.name = $1)
		GROUP BY u.id
		ORDER BY u.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Pool.Query(ctx, query, roleFilter, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*AdminUser
	for rows.Next() {
		u := &AdminUser{}
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL,
			&u.EmailVerifiedAt, &u.TwoFAEnabled, &u.CreatedAt, &u.Roles); err != nil {
			return nil, 0, err
		}
		if u.Roles == nil {
			u.Roles = []string{}
		}
		users = append(users, u)
	}
	if users == nil {
		users = []*AdminUser{}
	}
	return users, total, rows.Err()
}

// AssignRole adds a role to a user. Silently ignores if the assignment already exists.
func (r *Repository) AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, assigned_by, assigned_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_id, role_id) DO NOTHING`,
		userID, roleID, assignedBy,
	)
	return err
}

// RevokeRole removes a role from a user.
func (r *Repository) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`,
		userID, roleID,
	)
	return err
}

// DeleteUser soft-deletes a user by setting deleted_at.
func (r *Repository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// ListRoles returns all roles with their associated permissions.
func (r *Repository) ListRoles(ctx context.Context) ([]*Role, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		 FROM roles r ORDER BY r.created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		ro := &Role{Permissions: []Permission{}}
		if err := rows.Scan(&ro.ID, &ro.Name, &ro.Description, &ro.CreatedAt, &ro.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, ro)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if roles == nil {
		return []*Role{}, nil
	}

	// Fetch permissions for each role
	for _, ro := range roles {
		perms, err := r.getPermissionsForRole(ctx, ro.ID)
		if err != nil {
			return nil, err
		}
		ro.Permissions = perms
	}
	return roles, nil
}

// GetRole returns a single role with permissions.
func (r *Repository) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	ro := &Role{Permissions: []Permission{}}
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at FROM roles WHERE id = $1`, id,
	).Scan(&ro.ID, &ro.Name, &ro.Description, &ro.CreatedAt, &ro.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("get role: %w", err)
	}

	perms, err := r.getPermissionsForRole(ctx, ro.ID)
	if err != nil {
		return nil, err
	}
	ro.Permissions = perms
	return ro, nil
}

func (r *Repository) getPermissionsForRole(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT p.id, p.name, p.description, p.created_at
		 FROM permissions p
		 JOIN role_permissions rp ON rp.permission_id = p.id
		 WHERE rp.role_id = $1
		 ORDER BY p.name ASC`,
		roleID,
	)
	if err != nil {
		return nil, fmt.Errorf("get permissions for role: %w", err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		p := Permission{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	if perms == nil {
		perms = []Permission{}
	}
	return perms, rows.Err()
}

// CreateRole inserts a new role.
func (r *Repository) CreateRole(ctx context.Context, name string, description *string) (*Role, error) {
	ro := &Role{Permissions: []Permission{}}
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO roles (id, name, description, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id, name, description, created_at, updated_at`,
		uuid.New(), name, description,
	).Scan(&ro.ID, &ro.Name, &ro.Description, &ro.CreatedAt, &ro.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}
	return ro, nil
}

// UpdateRole updates a role's name and/or description.
func (r *Repository) UpdateRole(ctx context.Context, id uuid.UUID, name *string, description *string) (*Role, error) {
	ro := &Role{Permissions: []Permission{}}
	err := r.db.Pool.QueryRow(ctx,
		`UPDATE roles
		 SET name        = COALESCE($2, name),
		     description = COALESCE($3, description),
		     updated_at  = NOW()
		 WHERE id = $1
		 RETURNING id, name, description, created_at, updated_at`,
		id, name, description,
	).Scan(&ro.ID, &ro.Name, &ro.Description, &ro.CreatedAt, &ro.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("update role: %w", err)
	}

	perms, err := r.getPermissionsForRole(ctx, ro.ID)
	if err != nil {
		return nil, err
	}
	ro.Permissions = perms
	return ro, nil
}

// DeleteRole removes a role from the database.
func (r *Repository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("role not found")
	}
	return nil
}

// SetRolePermissions replaces the full permission set for a role (runs in a transaction).
func (r *Repository) SetRolePermissions(ctx context.Context, roleID uuid.UUID, permIDs []uuid.UUID) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return fmt.Errorf("clear permissions: %w", err)
	}

	for _, permID := range permIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			roleID, permID,
		); err != nil {
			return fmt.Errorf("insert permission: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ListPermissions returns all permissions.
func (r *Repository) ListPermissions(ctx context.Context) ([]*Permission, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, name, description, created_at FROM permissions ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()

	var perms []*Permission
	for rows.Next() {
		p := &Permission{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	if perms == nil {
		perms = []*Permission{}
	}
	return perms, rows.Err()
}

// GetPermissionsForRoles returns distinct permission names for a list of role names.
func (r *Repository) GetPermissionsForRoles(ctx context.Context, roles []string) ([]string, error) {
	if len(roles) == 0 {
		return []string{}, nil
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT DISTINCT p.name
		 FROM permissions p
		 JOIN role_permissions rp ON rp.permission_id = p.id
		 JOIN roles r ON r.id = rp.role_id
		 WHERE r.name = ANY($1)`,
		roles,
	)
	if err != nil {
		return nil, fmt.Errorf("get permissions for roles: %w", err)
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		perms = append(perms, name)
	}
	if perms == nil {
		perms = []string{}
	}
	return perms, rows.Err()
}

// GetRoleNameByID returns the role name for a given role ID.
func (r *Repository) GetRoleNameByID(ctx context.Context, roleID uuid.UUID) (string, error) {
	var name string
	err := r.db.Pool.QueryRow(ctx, `SELECT name FROM roles WHERE id = $1`, roleID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("get role name: %w", err)
	}
	return name, nil
}

// SearchUsers returns up to limit users whose display_name or email match the query (case-insensitive).
func (r *Repository) SearchUsers(ctx context.Context, q string, limit int) ([]*SearchResult, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT u.id, u.email, u.display_name, u.avatar_url
		FROM users u
		WHERE u.deleted_at IS NULL
		  AND (u.display_name ILIKE '%' || $1 || '%' OR u.email ILIKE '%' || $1 || '%')
		ORDER BY u.display_name ASC
		LIMIT $2`, q, limit)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var id uuid.UUID
		var email, displayName string
		var avatarURL *string
		if err := rows.Scan(&id, &email, &displayName, &avatarURL); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		results = append(results, &SearchResult{
			Type:      "user",
			ID:        id.String(),
			Title:     displayName,
			Subtitle:  email,
			AvatarURL: avatarURL,
			Href:      "/admin/users",
		})
	}
	return results, nil
}

// LogAction inserts a row into admin_audit_logs. Intended for fire-and-forget use.
func (r *Repository) LogAction(ctx context.Context, actorID uuid.UUID, action, targetType string, targetID *uuid.UUID, metadata map[string]any) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO admin_audit_logs (actor_id, action, target_type, target_id, metadata)
		 VALUES ($1, $2, $3, $4, $5)`,
		actorID, action, targetType, targetID, metadata,
	)
	return err
}

// ListAuditLogs returns paginated admin audit log entries with actor email joined.
func (r *Repository) ListAuditLogs(ctx context.Context, limit, offset int) ([]*AuditLog, int, error) {
	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_audit_logs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT l.id, l.actor_id, u.email, l.action, l.target_type, l.target_id, l.metadata, l.created_at
		FROM admin_audit_logs l
		LEFT JOIN users u ON u.id = l.actor_id
		ORDER BY l.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		l := &AuditLog{}
		var meta []byte
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorEmail, &l.Action, &l.TargetType, &l.TargetID, &meta, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(meta, &l.Metadata); err != nil {
			l.Metadata = map[string]any{}
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []*AuditLog{}
	}
	return logs, total, rows.Err()
}

// SearchRoles returns up to limit roles whose name matches the query (case-insensitive).
func (r *Repository) SearchRoles(ctx context.Context, q string, limit int) ([]*SearchResult, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, COALESCE(description, '')
		FROM roles
		WHERE name ILIKE '%' || $1 || '%'
		ORDER BY name ASC
		LIMIT $2`, q, limit)
	if err != nil {
		return nil, fmt.Errorf("search roles: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var id uuid.UUID
		var name, description string
		if err := rows.Scan(&id, &name, &description); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		results = append(results, &SearchResult{
			Type:     "role",
			ID:       id.String(),
			Title:    name,
			Subtitle: description,
			Href:     "/admin/roles",
		})
	}
	return results, nil
}
