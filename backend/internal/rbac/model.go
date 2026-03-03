package rbac

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description *string      `json:"description"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdminUser struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	DisplayName     string     `json:"display_name"`
	AvatarURL       *string    `json:"avatar_url"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	TwoFAEnabled    bool       `json:"two_fa_enabled"`
	Roles           []string   `json:"roles"`
	CreatedAt       time.Time  `json:"created_at"`
}

type AdminUsersResponse struct {
	Users []*AdminUser `json:"users"`
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}

type CreateRoleRequest struct {
	Name        string  `json:"name"        validate:"required,min=2,max=50"`
	Description *string `json:"description"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name"        validate:"omitempty,min=2,max=50"`
	Description *string `json:"description"`
}

type SetPermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}
