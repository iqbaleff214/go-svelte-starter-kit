package auth

import "time"

type RegisterRequest struct {
	DisplayName string `json:"display_name" validate:"required,min=2,max=100"`
	Email       string `json:"email"        validate:"required,email"`
	Password    string `json:"password"     validate:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

type UserResponse struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	DisplayName     string     `json:"display_name"`
	AvatarURL       *string    `json:"avatar_url"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	TwoFAEnabled    bool       `json:"two_fa_enabled"`
	CreatedAt       time.Time  `json:"created_at"`
}

type LoginResponse struct {
	User          UserResponse  `json:"user,omitempty"`
	Token         TokenResponse `json:"token,omitempty"`
	TwoFARequired *bool         `json:"two_fa_required,omitempty"`
	PreAuthToken  *string       `json:"pre_auth_token,omitempty"`
}

// ---- 2FA ----

type TwoFASetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
	QRCodePNG  string `json:"qr_code_png"` // base64-encoded PNG
}

// ---- Email verification / password reset ----

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"    validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type TwoFAConfirmRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

type TwoFAConfirmResponse struct {
	BackupCodes []string `json:"backup_codes"`
}

type TwoFAVerifyRequest struct {
	PreAuthToken string `json:"pre_auth_token" validate:"required"`
	Code         string `json:"code"`
	BackupCode   string `json:"backup_code"`
}

type TwoFADisableRequest struct {
	Code       string `json:"code"`
	BackupCode string `json:"backup_code"`
}
