package email

// Task type constants — used as asynq task type names.
const (
	TypeEmailWelcome         = "email:welcome"
	TypeEmailVerification    = "email:verification"
	TypeEmailPasswordReset   = "email:password_reset"
	TypeEmail2FABackupCodes  = "email:2fa_backup_codes"
	TypeEmailSecurityAlert   = "email:security_alert"
	TypeEmailAccountDeletion = "email:account_deletion"
)

// ---- Payload structs ----

type WelcomePayload struct {
	LogID       string `json:"log_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type VerifyEmailPayload struct {
	LogID       string `json:"log_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Token       string `json:"token"`
}

type PasswordResetPayload struct {
	LogID       string `json:"log_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Token       string `json:"token"`
}

type TwoFABackupPayload struct {
	LogID       string   `json:"log_id"`
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Codes       []string `json:"codes"`
}

type SecurityAlertPayload struct {
	LogID       string `json:"log_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	IP          string `json:"ip"`
	Device      string `json:"device"`
}

type AccountDeletionPayload struct {
	LogID       string `json:"log_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}
