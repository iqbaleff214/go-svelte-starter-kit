package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc       *Service
	validator *validator.Validator
}

func NewHandler(svc *Service, v *validator.Validator) *Handler {
	return &Handler{svc: svc, validator: v}
}

// GET /api/me
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	profile, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": profile})
}

// PATCH /api/me
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req UpdateProfileRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	userID := mustUserID(r)
	profile, err := h.svc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not update profile")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": profile})
}

// POST /api/me/avatar  (multipart/form-data, field: "avatar")
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Could not parse form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		respondError(w, http.StatusBadRequest, "missing_file", "avatar field is required")
		return
	}
	defer file.Close()

	userID := mustUserID(r)
	url, err := h.svc.UploadAvatar(r.Context(), userID, file, header)
	if err != nil {
		if errors.Is(err, ErrImageTooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "file_too_large", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidImageType) {
			respondError(w, http.StatusUnprocessableEntity, "invalid_image", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not upload avatar")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"avatar_url": url}})
}

// POST /api/me/change-password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	userID := mustUserID(r)
	if err := h.svc.ChangePassword(r.Context(), userID, req); err != nil {
		if errors.Is(err, ErrInvalidCurrentPassword) {
			respondError(w, http.StatusUnprocessableEntity, "invalid_password", "Current password is incorrect")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not change password")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Password changed successfully"}})
}

// DELETE /api/me
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	if err := h.svc.DeleteAccount(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not delete account")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Account deleted"}})
}

// GET /api/me/sessions
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	userID := mustUserID(r)

	sessions, err := h.svc.ListSessions(r.Context(), userID, claims.SessionID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not list sessions")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": sessions})
}

// DELETE /api/me/sessions/:id
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := mustUserID(r)

	if err := h.svc.RevokeSession(r.Context(), userID, sessionID); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "not_found", "Session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not revoke session")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Session revoked"}})
}

// DELETE /api/me/sessions
func (h *Handler) RevokeAllOtherSessions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	userID := mustUserID(r)

	if err := h.svc.RevokeAllOtherSessions(r.Context(), userID, claims.SessionID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not revoke sessions")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "All other sessions revoked"}})
}

// ---- helpers ----

type envelope map[string]any

func mustUserID(r *http.Request) uuid.UUID {
	claims := middleware.GetClaims(r)
	id, _ := uuid.Parse(claims.UserID)
	return id
}

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, envelope{
		"error": envelope{"code": code, "message": message},
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
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return false
	}
	return true
}
