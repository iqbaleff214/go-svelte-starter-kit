package apikey

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
	v   *validator.Validator
}

func NewHandler(svc *Service, v *validator.Validator) *Handler {
	return &Handler{svc: svc, v: v}
}

// POST /api/me/api-keys
func (h *Handler) CreateKey(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}
	if errs := h.v.Validate(req); errs != nil {
		respondJSON(w, http.StatusUnprocessableEntity, envelope{
			"error": envelope{"code": "validation_failed", "message": "Validation failed", "details": errs},
		})
		return
	}
	// Validate scopes
	if err := validateScopes(req.Scopes); err != nil {
		respondError(w, http.StatusUnprocessableEntity, "invalid_scopes", err.Error())
		return
	}

	resp, err := h.svc.Create(r.Context(), userID, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not create API key")
		return
	}
	respondJSON(w, http.StatusCreated, envelope{"data": resp})
}

// GET /api/me/api-keys
func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	keys, err := h.svc.ListKeys(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not list API keys")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": keys})
}

// DELETE /api/me/api-keys/{id}
func (h *Handler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid API key ID")
		return
	}
	if err := h.svc.RevokeKey(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusNotFound, "not_found", "API key not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/me/api-keys/{id}/logs
func (h *Handler) ListLogs(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid API key ID")
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	logs, total, err := h.svc.ListLogs(r.Context(), id, userID, limit, offset)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "API key not found")
		return
	}
	respondJSON(w, http.StatusOK, envelope{
		"data": envelope{"logs": logs, "total": total, "limit": limit, "offset": offset},
	})
}

// ---- helpers ----

type envelope map[string]any

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

func mustUserID(r *http.Request) uuid.UUID {
	claims := middleware.GetClaims(r)
	id, _ := uuid.Parse(claims.UserID)
	return id
}

func validateScopes(scopes []string) error {
	valid := make(map[string]struct{}, len(AllScopes))
	for _, s := range AllScopes {
		valid[s] = struct{}{}
	}
	for _, s := range scopes {
		if _, ok := valid[s]; !ok {
			return fmt.Errorf("invalid scope: %q", s)
		}
	}
	return nil
}
