package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
)

const refreshTokenCookie = "refresh_token"

type Handler struct {
	svc       *Service
	validator *validator.Validator
	secureCookie bool
	refreshTTL   time.Duration
}

func NewHandler(svc *Service, v *validator.Validator, secureCookie bool, refreshTTL time.Duration) *Handler {
	return &Handler{
		svc:          svc,
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
