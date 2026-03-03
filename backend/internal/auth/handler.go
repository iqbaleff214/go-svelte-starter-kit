package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const refreshTokenCookie = "refresh_token"

type Handler struct {
	svc          *Service
	google       *GoogleProvider
	redis        *redis.Client
	frontendURL  string
	validator    *validator.Validator
	secureCookie bool
	refreshTTL   time.Duration
}

func NewHandler(svc *Service, google *GoogleProvider, rdb *redis.Client, frontendURL string, v *validator.Validator, secureCookie bool, refreshTTL time.Duration) *Handler {
	return &Handler{
		svc:          svc,
		google:       google,
		redis:        rdb,
		frontendURL:  frontendURL,
		validator:    v,
		secureCookie: secureCookie,
		refreshTTL:   refreshTTL,
	}
}

// POST /api/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	resp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			respondError(w, http.StatusConflict, "email_taken", "Email is already in use")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "Registration failed")
		return
	}

	respondJSON(w, http.StatusCreated, envelope{"data": resp})
}

// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	resp, refreshToken, err := h.svc.Login(r.Context(), req, r)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "Login failed")
		return
	}

	h.setRefreshCookie(w, refreshToken)
	respondJSON(w, http.StatusOK, envelope{"data": resp})
}

// POST /api/auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := h.getRefreshToken(r)
	if refreshToken == "" {
		respondError(w, http.StatusUnauthorized, "missing_token", "Refresh token is required")
		return
	}

	resp, newRefreshToken, err := h.svc.Refresh(r.Context(), refreshToken, r)
	if err != nil {
		h.clearRefreshCookie(w)
		respondError(w, http.StatusUnauthorized, "invalid_token", "Token refresh failed")
		return
	}

	h.setRefreshCookie(w, newRefreshToken)
	respondJSON(w, http.StatusOK, envelope{"data": resp})
}

// POST /api/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := h.getRefreshToken(r)
	_ = h.svc.Logout(r.Context(), refreshToken)
	h.clearRefreshCookie(w)
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Logged out successfully"}})
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    token,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(h.refreshTTL.Seconds()),
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   h.secureCookie,
		MaxAge:   -1,
	})
}

func (h *Handler) getRefreshToken(r *http.Request) string {
	// Check cookie first
	if cookie, err := r.Cookie(refreshTokenCookie); err == nil {
		return cookie.Value
	}
	// Fallback: JSON body
	var body RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		return body.RefreshToken
	}
	return ""
}

// ---- Google OAuth ----

const (
	oauthStatePrefix = "oauth:state:"
	oauthCodePrefix  = "oauth:code:"
	oauthStateTTL    = 10 * time.Minute
	oauthCodeTTL     = 30 * time.Second
)

// GET /api/auth/google
func (h *Handler) GoogleRedirect(w http.ResponseWriter, r *http.Request) {
	state, err := randomHex(16)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not initiate OAuth")
		return
	}

	if err := h.redis.Set(r.Context(), oauthStatePrefix+state, "1", oauthStateTTL).Err(); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not initiate OAuth")
		return
	}

	http.Redirect(w, r, h.google.AuthURL(state), http.StatusTemporaryRedirect)
}

// GET /api/auth/google/callback
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	ctx := r.Context()

	// Validate & consume the state token
	key := oauthStatePrefix + state
	if err := h.redis.GetDel(ctx, key).Err(); err != nil {
		http.Redirect(w, r, h.frontendURL+"/login?error=oauth_state", http.StatusTemporaryRedirect)
		return
	}

	info, err := h.google.Exchange(ctx, code)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/login?error=oauth_exchange", http.StatusTemporaryRedirect)
		return
	}

	loginResp, refreshToken, err := h.svc.LoginOrCreateWithOAuth(ctx, info, "google", r)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/login?error=oauth_failed", http.StatusTemporaryRedirect)
		return
	}

	// Store result under a one-time code
	exchangeCode, err := randomHex(16)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/login?error=internal", http.StatusTemporaryRedirect)
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"login_resp":    mustJSON(loginResp),
		"refresh_token": refreshToken,
	})
	h.redis.Set(ctx, oauthCodePrefix+exchangeCode, payload, oauthCodeTTL)

	http.Redirect(w, r, h.frontendURL+"/auth/callback?code="+exchangeCode, http.StatusTemporaryRedirect)
}

// POST /api/auth/google/exchange
func (h *Handler) ExchangeOAuthCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Code == "" {
		respondError(w, http.StatusBadRequest, "missing_code", "Code is required")
		return
	}

	ctx := r.Context()
	raw, err := h.redis.GetDel(ctx, oauthCodePrefix+body.Code).Bytes()
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid_code", "OAuth code is invalid or expired")
		return
	}

	var payload struct {
		LoginResp    string `json:"login_resp"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Failed to decode OAuth payload")
		return
	}

	var loginResp LoginResponse
	if err := json.Unmarshal([]byte(payload.LoginResp), &loginResp); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Failed to decode login response")
		return
	}

	h.setRefreshCookie(w, payload.RefreshToken)
	respondJSON(w, http.StatusOK, envelope{"data": loginResp})
}

// ---- 2FA ----

// POST /api/auth/2fa/setup  (protected)
func (h *Handler) SetupTwoFA(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	userID, err := parseUUID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_user", "Invalid user ID")
		return
	}

	resp, err := h.svc.SetupTwoFA(r.Context(), userID, claims.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not set up 2FA")
		return
	}

	respondJSON(w, http.StatusOK, envelope{"data": resp})
}

// POST /api/auth/2fa/confirm  (protected)
func (h *Handler) ConfirmTwoFA(w http.ResponseWriter, r *http.Request) {
	var req TwoFAConfirmRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	claims := middleware.GetClaims(r)
	userID, err := parseUUID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_user", "Invalid user ID")
		return
	}

	backupCodes, err := h.svc.ConfirmTwoFA(r.Context(), userID, req.Code)
	if err != nil {
		if errors.Is(err, ErrTwoFAInvalid) {
			respondError(w, http.StatusUnprocessableEntity, "invalid_code", "Invalid 2FA code")
			return
		}
		if errors.Is(err, ErrTwoFAAlreadyOn) {
			respondError(w, http.StatusConflict, "already_enabled", "2FA is already enabled")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not enable 2FA")
		return
	}

	respondJSON(w, http.StatusOK, envelope{"data": TwoFAConfirmResponse{BackupCodes: backupCodes}})
}

// POST /api/auth/2fa/verify  (no auth middleware — uses pre_auth_token)
func (h *Handler) VerifyTwoFA(w http.ResponseWriter, r *http.Request) {
	var req TwoFAVerifyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	resp, refreshToken, err := h.svc.VerifyTwoFA(r.Context(), req, r)
	if err != nil {
		if errors.Is(err, ErrTwoFAInvalid) {
			respondError(w, http.StatusUnprocessableEntity, "invalid_code", "Invalid 2FA code")
			return
		}
		respondError(w, http.StatusUnauthorized, "unauthorized", "Authentication failed")
		return
	}

	h.setRefreshCookie(w, refreshToken)
	respondJSON(w, http.StatusOK, envelope{"data": resp})
}

// DELETE /api/auth/2fa  (protected)
func (h *Handler) DisableTwoFA(w http.ResponseWriter, r *http.Request) {
	var req TwoFADisableRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	claims := middleware.GetClaims(r)
	userID, err := parseUUID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_user", "Invalid user ID")
		return
	}

	if err := h.svc.DisableTwoFA(r.Context(), userID, req.Code, req.BackupCode); err != nil {
		if errors.Is(err, ErrTwoFAInvalid) {
			respondError(w, http.StatusUnprocessableEntity, "invalid_code", "Invalid code or backup code")
			return
		}
		if errors.Is(err, ErrTwoFANotEnabled) {
			respondError(w, http.StatusConflict, "not_enabled", "2FA is not enabled")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not disable 2FA")
		return
	}

	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "2FA disabled"}})
}

// ---- Email verification / password reset ----

// POST /api/auth/forgot-password
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	// Always succeed to prevent email enumeration
	_ = h.svc.ForgotPassword(r.Context(), req.Email)
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "If that email exists, a reset link has been sent"}})
}

// POST /api/auth/reset-password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	if err := h.svc.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		respondError(w, http.StatusUnprocessableEntity, "invalid_token", "Reset link is invalid or expired")
		return
	}

	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Password updated successfully"}})
}

// POST /api/auth/verify-email
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	if err := h.svc.VerifyEmail(r.Context(), req.Token); err != nil {
		respondError(w, http.StatusUnprocessableEntity, "invalid_token", "Verification link is invalid or expired")
		return
	}

	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Email verified successfully"}})
}

// POST /api/auth/resend-verification  (protected)
func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	userID, err := parseUUID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_user", "Invalid user ID")
		return
	}

	if err := h.svc.ResendVerification(r.Context(), userID); err != nil {
		respondError(w, http.StatusConflict, "already_verified", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Verification email sent"}})
}

// ---- response helpers ----

type envelope map[string]any

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, envelope{
		"error": envelope{
			"code":    code,
			"message": message,
		},
	})
}

func respondValidation(w http.ResponseWriter, errs []validator.FieldError) {
	respondJSON(w, http.StatusUnprocessableEntity, envelope{
		"error": envelope{
			"code":    "validation_failed",
			"message": "Validation failed",
			"details": errs,
		},
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return false
	}
	return true
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
