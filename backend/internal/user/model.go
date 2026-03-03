package user

import "time"

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name" validate:"omitempty,min=2,max=100"`
	Bio         string `json:"bio"          validate:"omitempty,max=500"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,min=8,max=72"`
}

type ProfileResponse struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	DisplayName     string     `json:"display_name"`
	AvatarURL       *string    `json:"avatar_url"`
	Bio             *string    `json:"bio"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	TwoFAEnabled    bool       `json:"two_fa_enabled"`
	CreatedAt       time.Time  `json:"created_at"`
}

type SessionResponse struct {
	ID         string    `json:"id"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
	LastSeenAt time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
	IsCurrent  bool      `json:"is_current"`
}
